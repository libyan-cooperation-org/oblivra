package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the on-disk agent configuration. Designed to be hand-edited and
// version-controlled. Path conventions match what a Splunk Universal
// Forwarder operator would expect:
//
//	/etc/oblivra/agent.yml         (unix)
//	%PROGRAMDATA%\oblivra\agent.yml (windows)
//
// Override via --config FILE.
type Config struct {
	Server          Server         `yaml:"server"`
	DualEgress      []Server       `yaml:"dualEgress"`      // optional — ship to N additional servers
	Hostname        string         `yaml:"hostname"`
	Tenant          string         `yaml:"tenant"`
	Tags            []string       `yaml:"tags"`
	Inputs          []Input        `yaml:"inputs"`
	Buffer          BufferOpts     `yaml:"buffer"`
	Heartbeat       HeartbeatOpts  `yaml:"heartbeat"`
	Compression     string         `yaml:"compression"`     // "gzip" or "" (none)
	BatchSize       int            `yaml:"batchSize"`
	FlushEvery      time.Duration  `yaml:"flushEvery"`
	Multiline       MultilineOpts  `yaml:"multiline"`
	StateDir        string         `yaml:"stateDir"`        // position files + signing key
	LogLevel        string         `yaml:"logLevel"`
	SignEvents      bool           `yaml:"signEvents"`      // ed25519-sign every event at the edge
	SpillSecret     string         `yaml:"spillSecret"`     // AES-256-GCM key for disk spill
	SpillSecretFile string         `yaml:"spillSecretFile"` // alternative — read from file (mode 0600)
	AdaptiveBatch   bool           `yaml:"adaptiveBatch"`   // auto-tune batch size against observed p99
	LocalRules      bool           `yaml:"localRules"`      // run the local rule pack to prioritise high-sev events
}

type Server struct {
	URL              string        `yaml:"url"`
	Token            string        `yaml:"token"`
	TokenFile        string        `yaml:"tokenFile"`        // file holding the bearer token
	TLS              TLSOpts       `yaml:"tls"`
	RequestTimeout   time.Duration `yaml:"requestTimeout"`
	HealthInterval   time.Duration `yaml:"healthInterval"`
}

type TLSOpts struct {
	CACertFile     string `yaml:"caCertFile"`
	ClientCertFile string `yaml:"clientCertFile"`
	ClientKeyFile  string `yaml:"clientKeyFile"`
	ServerNameOverride string `yaml:"serverNameOverride"`
	PinnedSHA256   string `yaml:"pinnedSha256"` // base64 pin of server cert pubkey
	Insecure       bool   `yaml:"insecure"`     // dev only
}

type Input struct {
	Type        string            `yaml:"type"`        // "file" | "stdin" | "winlog" | "syslog-udp"
	Path        string            `yaml:"path"`        // file path or glob
	Label       string            `yaml:"label"`       // e.g. "auth.log"
	SourceType  string            `yaml:"sourceType"`  // freeform: "linux:auth", "iis:access"
	HostID      string            `yaml:"hostId"`      // override per-input
	Multiline   *MultilineOpts    `yaml:"multiline,omitempty"`
	Fields      map[string]string `yaml:"fields"`      // injected into every event
	IncludeOnly string            `yaml:"includeOnly"` // optional regex; lines must match to be sent
	Exclude     string            `yaml:"exclude"`     // optional regex; matching lines are dropped
	StartFrom   string            `yaml:"startFrom"`   // "tail" (default) | "beginning"
	// winlog-specific:
	Channel string `yaml:"channel"` // e.g. "Security", "System"

	// Extract is a list of named regex patterns. The first regex that
	// matches a line contributes its named-group captures to the
	// outgoing event's `fields`. This lets the agent promote captured
	// values (user, srcIP, status code, etc.) to top-level fields *at
	// the edge*, so the platform doesn't have to re-extract them and
	// the wire payload becomes cheaper. UF doesn't ship anything like
	// this — it forwards raw lines and lets indexers do the work.
	Extract []ExtractRule `yaml:"extract"`
}

// ExtractRule is a named regex run against each tailed line. Capture
// groups become event fields. The whole rule fires only on the first
// match — multiple rules cooperate by being listed in priority order
// (most specific first).
type ExtractRule struct {
	Name  string `yaml:"name"`  // operator-friendly label, used in field-extract diagnostics
	Regex string `yaml:"regex"` // Go regex with named (?P<name>...) groups; unnamed groups are ignored
}

type MultilineOpts struct {
	StartPattern string        `yaml:"startPattern"` // regex; lines matching start a NEW event
	MaxLines     int           `yaml:"maxLines"`
	Timeout      time.Duration `yaml:"timeout"`
}

type BufferOpts struct {
	Dir      string `yaml:"dir"`
	MaxBytes int64  `yaml:"maxBytes"` // hard cap on disk-buffer size
}

type HeartbeatOpts struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
}

// LoadConfig reads & validates a config file; applies sane defaults for any
// field the operator omitted. Files ending in `.enc` are AES-256-GCM
// encrypted with an Argon2id-derived key (see config_enc.go) — the
// passphrase comes from OBLIVRA_AGENT_PASSPHRASE or
// OBLIVRA_AGENT_PASSPHRASE_FILE.
func LoadConfig(path string) (*Config, error) {
	var body []byte
	if strings.HasSuffix(path, ".enc") {
		passphrase, err := readPassphrase()
		if err != nil {
			return nil, err
		}
		if passphrase == nil {
			return nil, fmt.Errorf("config %s is encrypted; set OBLIVRA_AGENT_PASSPHRASE or OBLIVRA_AGENT_PASSPHRASE_FILE", path)
		}
		body, err = loadEncryptedConfig(path, passphrase)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		body, err = os.ReadFile(path)
		if err != nil {
			return nil, err
		}
	}
	var c Config
	if err := yaml.Unmarshal(body, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	c.applyDefaults(path)
	if err := c.validate(); err != nil {
		return nil, err
	}
	// If a tokenFile is set, load the token from it (mode-0600 expected).
	if c.Server.Token == "" && c.Server.TokenFile != "" {
		tok, err := os.ReadFile(c.Server.TokenFile)
		if err != nil {
			return nil, fmt.Errorf("tokenFile: %w", err)
		}
		c.Server.Token = trimSpaceBoth(string(tok))
	}
	// Spill secret may also live in a 0600-mode file.
	if c.SpillSecret == "" && c.SpillSecretFile != "" {
		secret, err := os.ReadFile(c.SpillSecretFile)
		if err != nil {
			return nil, fmt.Errorf("spillSecretFile: %w", err)
		}
		c.SpillSecret = trimSpaceBoth(string(secret))
	}
	return &c, nil
}

func DefaultConfigPath() string {
	if v := os.Getenv("OBLIVRA_AGENT_CONFIG"); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("PROGRAMDATA")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "oblivra", "agent.yml")
	default:
		return "/etc/oblivra/agent.yml"
	}
}

// SampleConfigYAML returns a fully-commented example for `oblivra-agent
// init`. Operators run init, edit the generated file, and never have to
// guess at field names.
func SampleConfigYAML(hostname string) string {
	return fmt.Sprintf(`# OBLIVRA agent config — see https://github.com/libyan-cooperation-org/oblivra
# Reload with SIGHUP or "oblivra-agent reload".

server:
  url: "https://oblivra.internal"
  # Either token: ... here, OR put it in a 0600-mode file:
  # tokenFile: "/etc/oblivra/agent.token"
  token: "REPLACE-WITH-AGENT-API-KEY"
  requestTimeout: 10s
  healthInterval: 60s
  tls:
    # In production at minimum specify caCertFile so the agent verifies
    # the server cert against your CA, not the OS truststore.
    caCertFile: ""
    clientCertFile: ""        # mTLS client cert — optional
    clientKeyFile: ""
    serverNameOverride: ""
    pinnedSha256: ""          # base64 pin of server pubkey for air-gap
    insecure: false           # never set true in production

hostname: "%s"
tenant: "default"
tags: ["env:prod", "tier:edge"]

batchSize: 100
flushEvery: 2s
compression: "gzip"           # or "" to disable

buffer:
  dir: ""                     # default: state-dir/buffer
  maxBytes: 1073741824        # 1 GiB cap on disk-spill

heartbeat:
  enabled: true
  interval: 30s

stateDir: ""                  # default: %s

multiline:
  startPattern: ""            # default: each line is its own event
  maxLines: 200
  timeout: 5s

inputs:
  # File tail — the bread and butter
  - type: file
    path: "/var/log/auth.log"
    sourceType: "linux:auth"
    startFrom: "tail"          # or "beginning" for backfills
    fields:
      env: "prod"

  # Multi-line example — Java stack traces
  # - type: file
  #   path: "/var/log/myapp/*.log"
  #   sourceType: "java:app"
  #   multiline:
  #     startPattern: "^\\d{4}-\\d{2}-\\d{2}"   # ISO date starts each event
  #     maxLines: 100
  #     timeout: 2s

  # stdin — handy for "tail -F | oblivra-agent run --pipe"
  # - type: stdin
  #   sourceType: "manual:stdin"
`, hostname, defaultStateDir())
}

func defaultStateDir() string {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("PROGRAMDATA")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "oblivra", "agent")
	default:
		return "/var/lib/oblivra-agent"
	}
}

func (c *Config) applyDefaults(configPath string) {
	if c.Server.URL == "" {
		c.Server.URL = "http://localhost:8080"
	}
	if c.Server.RequestTimeout == 0 {
		c.Server.RequestTimeout = 10 * time.Second
	}
	if c.Server.HealthInterval == 0 {
		c.Server.HealthInterval = 60 * time.Second
	}
	if c.Hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			h = "unknown"
		}
		c.Hostname = h
	}
	if c.Tenant == "" {
		c.Tenant = "default"
	}
	if c.BatchSize == 0 {
		c.BatchSize = 100
	}
	if c.FlushEvery == 0 {
		c.FlushEvery = 2 * time.Second
	}
	if c.StateDir == "" {
		c.StateDir = defaultStateDir()
	}
	if c.Buffer.Dir == "" {
		c.Buffer.Dir = filepath.Join(c.StateDir, "buffer")
	}
	if c.Buffer.MaxBytes == 0 {
		c.Buffer.MaxBytes = 1 << 30 // 1 GiB
	}
	if !c.Heartbeat.isEmpty() {
		if c.Heartbeat.Interval == 0 {
			c.Heartbeat.Interval = 30 * time.Second
		}
	}
	if c.Multiline.MaxLines == 0 {
		c.Multiline.MaxLines = 200
	}
	if c.Multiline.Timeout == 0 {
		c.Multiline.Timeout = 5 * time.Second
	}
	for i := range c.Inputs {
		if c.Inputs[i].StartFrom == "" {
			c.Inputs[i].StartFrom = "tail"
		}
		if c.Inputs[i].HostID == "" {
			c.Inputs[i].HostID = c.Hostname
		}
	}
	_ = configPath // reserved for future include-relative resolution
}

func (c *Config) validate() error {
	if c.Server.URL == "" {
		return errors.New("server.url required")
	}
	if len(c.Inputs) == 0 {
		return errors.New("at least one input required (file / stdin / winlog)")
	}
	for i, in := range c.Inputs {
		switch in.Type {
		case "file":
			if in.Path == "" {
				return fmt.Errorf("inputs[%d]: file input needs path", i)
			}
		case "stdin":
			// no further fields required
		case "winlog":
			if runtime.GOOS != "windows" {
				return fmt.Errorf("inputs[%d]: winlog requires Windows", i)
			}
			if in.Channel == "" {
				return fmt.Errorf("inputs[%d]: winlog input needs channel", i)
			}
		case "syslog-udp":
			if in.Path == "" {
				return fmt.Errorf("inputs[%d]: syslog-udp input needs path (e.g. ':1515')", i)
			}
		default:
			return fmt.Errorf("inputs[%d]: unknown type %q (file|stdin|winlog|syslog-udp)", i, in.Type)
		}
	}
	return nil
}

func (h HeartbeatOpts) isEmpty() bool { return !h.Enabled && h.Interval == 0 }

// trimSpaceBoth strips leading/trailing whitespace including CR.
func trimSpaceBoth(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		c := s[len(s)-1]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}
