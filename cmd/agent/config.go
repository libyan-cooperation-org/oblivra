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
//	/etc/oblivra/agent.yml          (Linux / macOS)
//	%PROGRAMDATA%\oblivra\agent.yml (Windows)
//
// Override via --config FILE or OBLIVRA_AGENT_CONFIG.
type Config struct {
	Server          Server        `yaml:"server"`
	DualEgress      []Server      `yaml:"dualEgress"`
	Hostname        string        `yaml:"hostname"`
	Tenant          string        `yaml:"tenant"`
	Tags            []string      `yaml:"tags"`
	Inputs          []Input       `yaml:"inputs"`
	Buffer          BufferOpts    `yaml:"buffer"`
	Heartbeat       HeartbeatOpts `yaml:"heartbeat"`
	Compression     string        `yaml:"compression"`
	BatchSize       int           `yaml:"batchSize"`
	FlushEvery      time.Duration `yaml:"flushEvery"`
	Multiline       MultilineOpts `yaml:"multiline"`
	StateDir        string        `yaml:"stateDir"`
	LogLevel        string        `yaml:"logLevel"`
	SignEvents      bool          `yaml:"signEvents"`
	Redact          bool          `yaml:"redact"`
	SpillSecret     string        `yaml:"spillSecret"`
	SpillSecretFile string        `yaml:"spillSecretFile"`
	AdaptiveBatch   bool          `yaml:"adaptiveBatch"`
	LocalRules      bool          `yaml:"localRules"`
	LocalStatusAddr string        `yaml:"localStatusAddr"`
}

type Server struct {
	URL            string        `yaml:"url"`
	Token          string        `yaml:"token"`
	TokenFile      string        `yaml:"tokenFile"`
	TLS            TLSOpts       `yaml:"tls"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
	HealthInterval time.Duration `yaml:"healthInterval"`
}

type TLSOpts struct {
	CACertFile         string `yaml:"caCertFile"`
	ClientCertFile     string `yaml:"clientCertFile"`
	ClientKeyFile      string `yaml:"clientKeyFile"`
	ServerNameOverride string `yaml:"serverNameOverride"`
	PinnedSHA256       string `yaml:"pinnedSha256"`
	Insecure           bool   `yaml:"insecure"`
}

// Input describes one log source. Valid types:
//
//	file       — tail a file or glob (Linux, macOS, Windows)
//	stdin      — read newline-delimited events from stdin (all platforms)
//	winlog     — Windows Event Log channel via wevtutil.exe (Windows only)
//	            alias accepted: "winevent", "winevt", "windows-event"
//	syslog-udp — receive RFC-3164/5424 datagrams on a UDP port (all platforms)
//	journald   — tail systemd journal via journalctl subprocess (Linux only)
type Input struct {
	Type        string            `yaml:"type"`
	Path        string            `yaml:"path"`
	Label       string            `yaml:"label"`
	SourceType  string            `yaml:"sourceType"`
	HostID      string            `yaml:"hostId"`
	Multiline   *MultilineOpts    `yaml:"multiline,omitempty"`
	Fields      map[string]string `yaml:"fields"`
	IncludeOnly string            `yaml:"includeOnly"`
	Exclude     string            `yaml:"exclude"`
	StartFrom   string            `yaml:"startFrom"` // "tail" (default) | "beginning"

	// winlog-specific
	Channel string `yaml:"channel"` // e.g. "Security", "System", "Microsoft-Windows-Sysmon/Operational"

	// journald-specific (Linux only)
	JournaldUnits     []string `yaml:"units"`
	JournaldMatches   []string `yaml:"matches"`
	JournaldPriority  string   `yaml:"priority"`
	JournaldSinceBoot bool     `yaml:"sinceBoot"`

	// Edge regex extraction — first matching rule contributes named-group
	// captures to the event fields map before the event is shipped.
	Extract []ExtractRule `yaml:"extract"`
}

type ExtractRule struct {
	Name  string `yaml:"name"`
	Regex string `yaml:"regex"`
}

type MultilineOpts struct {
	StartPattern string        `yaml:"startPattern"`
	MaxLines     int           `yaml:"maxLines"`
	Timeout      time.Duration `yaml:"timeout"`
}

type BufferOpts struct {
	Dir      string `yaml:"dir"`
	MaxBytes int64  `yaml:"maxBytes"`
}

type HeartbeatOpts struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
}

// normaliseInputType canonicalises user-supplied type strings so common
// aliases ("winevent", "winevt", "windows-event") map to "winlog".
// This is the fix for: config: inputs[0]: unknown type "winevent"
func normaliseInputType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "winevent", "winevt", "windows-event", "windows_event", "win-event":
		return "winlog"
	case "journal", "systemd-journal":
		return "journald"
	case "syslog_udp", "syslogudp", "syslog":
		return "syslog-udp"
	default:
		return t
	}
}

// LoadConfig reads, decrypts (if .enc), unmarshals, and validates the
// config file. Applies sane defaults for omitted fields.
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

	// Normalise input type aliases before validation.
	for i := range c.Inputs {
		c.Inputs[i].Type = normaliseInputType(c.Inputs[i].Type)
	}

	c.applyDefaults(path)
	if err := c.validate(); err != nil {
		return nil, err
	}
	if c.Server.Token == "" && c.Server.TokenFile != "" {
		tok, err := os.ReadFile(c.Server.TokenFile)
		if err != nil {
			return nil, fmt.Errorf("tokenFile: %w", err)
		}
		c.Server.Token = trimSpaceBoth(string(tok))
	}
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

// SampleConfigYAML returns a fully-commented starter config for `oblivra-agent
// init`. The generated inputs section is platform-appropriate:
//   - Windows: winlog channels (Security, System, Application, Sysmon if present)
//   - Linux:   journald + the most useful /var/log/* files
//   - macOS:   /var/log/system.log + /var/log/install.log
func SampleConfigYAML(hostname string) string {
	inputsBlock := platformSampleInputs()
	return fmt.Sprintf(`# OBLIVRA agent config — https://github.com/kingknull/oblivra
# Reload with SIGHUP (Linux/macOS) or service restart (Windows).

server:
  url: "https://oblivra.internal"
  # Use tokenFile (0600-mode file) instead of inline token in production:
  # tokenFile: "%s"
  token: "REPLACE-WITH-AGENT-API-KEY"
  requestTimeout: 10s
  healthInterval: 60s
  tls:
    caCertFile: ""         # path to your CA cert — leave blank to use OS truststore
    clientCertFile: ""     # mTLS client cert (optional)
    clientKeyFile: ""
    insecure: false        # NEVER true in production

hostname: "%s"
tenant: "default"
tags: ["env:prod", "tier:edge"]

batchSize: 100
flushEvery: 2s
compression: "gzip"

# Security hardening
signEvents: true           # ed25519-sign every event at the edge
redact: true               # mask credit-card numbers, AWS keys, JWT tokens, etc.
localRules: true           # pre-score events locally; high-severity ships first
adaptiveBatch: true        # auto-tune batch size to observed latency

buffer:
  dir: ""                  # default: <stateDir>/buffer
  maxBytes: 1073741824     # 1 GiB cap

heartbeat:
  enabled: true
  interval: 30s

stateDir: ""               # default: %s

multiline:
  startPattern: ""         # regex; blank = each line is its own event
  maxLines: 200
  timeout: 5s

inputs:
%s`,
		tokenFilePath(),
		hostname,
		defaultStateDir(),
		inputsBlock,
	)
}

func tokenFilePath() string {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("PROGRAMDATA")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "oblivra", "agent.token")
	default:
		return "/etc/oblivra/agent.token"
	}
}

// platformSampleInputs returns YAML for the inputs: block appropriate for
// the current OS.  Windows uses winlog channels only — never .evtx file
// paths, which require the Windows XML Event Log API and aren't tailable.
func platformSampleInputs() string {
	switch runtime.GOOS {

	case "windows":
		return `  # ── Windows Event Log channels ─────────────────────────────────────
  # Use type: winlog with a channel name. The agent reads from the live
  # event log via wevtutil.exe — no .evtx file paths needed.
  #
  # startFrom: "beginning" ships everything already in the channel on
  # first run (recommended for a new deployment). Change to "tail" once
  # you're caught up.

  - type: winlog
    channel: "Security"
    sourceType: "windows:security"
    startFrom: "beginning"
    label: "win-security"

  - type: winlog
    channel: "System"
    sourceType: "windows:system"
    startFrom: "beginning"
    label: "win-system"

  - type: winlog
    channel: "Application"
    sourceType: "windows:application"
    startFrom: "beginning"
    label: "win-application"

  # Sysmon — uncomment if Sysmon is installed (highly recommended).
  # Provides process creation, network connections, file events, etc.
  # - type: winlog
  #   channel: "Microsoft-Windows-Sysmon/Operational"
  #   sourceType: "windows:sysmon"
  #   startFrom: "beginning"
  #   label: "win-sysmon"

  # PowerShell script-block logging — uncomment if enabled via GPO.
  # - type: winlog
  #   channel: "Microsoft-Windows-PowerShell/Operational"
  #   sourceType: "windows:powershell"
  #   startFrom: "beginning"
  #   label: "win-powershell"

  # Windows Defender — uncomment to ship AV telemetry.
  # - type: winlog
  #   channel: "Microsoft-Windows-Windows Defender/Operational"
  #   sourceType: "windows:defender"
  #   startFrom: "beginning"
  #   label: "win-defender"

  # WMI activity — uncomment to detect WMI persistence / lateral movement.
  # - type: winlog
  #   channel: "Microsoft-Windows-WMI-Activity/Operational"
  #   sourceType: "windows:wmi"
  #   startFrom: "beginning"
  #   label: "win-wmi"
`

	case "linux":
		return `  # ── systemd journal (covers all systemd services) ───────────────────
  - type: journald
    sourceType: "linux:journal"
    startFrom: "beginning"    # ship the entire journal on first run
    label: "journal"
    # Optionally restrict to specific units:
    # units: ["sshd.service", "nginx.service", "sudo.service"]
    # priority: "info"        # emerg|alert|crit|err|warning|notice|info|debug

  # ── Auth / privilege escalation ──────────────────────────────────────
  - type: file
    path: "/var/log/auth.log"        # Debian / Ubuntu
    sourceType: "linux:auth"
    startFrom: "beginning"
    label: "auth"

  # - type: file
  #   path: "/var/log/secure"          # RHEL / CentOS / Fedora
  #   sourceType: "linux:auth"
  #   startFrom: "beginning"
  #   label: "auth"

  # ── Kernel & audit ───────────────────────────────────────────────────
  - type: file
    path: "/var/log/kern.log"
    sourceType: "linux:kernel"
    startFrom: "beginning"
    label: "kernel"

  - type: file
    path: "/var/log/audit/audit.log"
    sourceType: "linux:auditd"
    startFrom: "beginning"
    label: "auditd"

  # ── Syslog catch-all ─────────────────────────────────────────────────
  - type: file
    path: "/var/log/syslog"          # Debian / Ubuntu
    sourceType: "linux:syslog"
    startFrom: "beginning"
    label: "syslog"

  # - type: file
  #   path: "/var/log/messages"        # RHEL / CentOS / Fedora
  #   sourceType: "linux:syslog"
  #   startFrom: "beginning"
  #   label: "syslog"

  # ── Web servers ──────────────────────────────────────────────────────
  # - type: file
  #   path: "/var/log/nginx/access.log"
  #   sourceType: "nginx:access"
  #   startFrom: "beginning"
  #   label: "nginx-access"

  # - type: file
  #   path: "/var/log/nginx/error.log"
  #   sourceType: "nginx:error"
  #   startFrom: "beginning"
  #   label: "nginx-error"

  # ── Network device syslog (UDP receiver) ─────────────────────────────
  # - type: syslog-udp
  #   path: ":1514"
  #   sourceType: "syslog:network"
  #   label: "network-devices"
`

	default: // macOS / other
		return `  - type: file
    path: "/var/log/system.log"
    sourceType: "macos:system"
    startFrom: "beginning"
    label: "system"

  - type: file
    path: "/var/log/install.log"
    sourceType: "macos:install"
    startFrom: "beginning"
    label: "install"
`
	}
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
	if c.Heartbeat.Enabled && c.Heartbeat.Interval == 0 {
		c.Heartbeat.Interval = 30 * time.Second
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
		// Auto-assign label from type+index if not set.
		if c.Inputs[i].Label == "" {
			c.Inputs[i].Label = fmt.Sprintf("%s-%d", c.Inputs[i].Type, i)
		}
	}
	_ = configPath
}

func (c *Config) validate() error {
	if c.Server.URL == "" {
		return errors.New("server.url required")
	}
	if len(c.Inputs) == 0 {
		return errors.New("at least one input required")
	}
	for i, in := range c.Inputs {
		switch in.Type {
		case "file":
			if in.Path == "" {
				return fmt.Errorf("inputs[%d]: file input requires path", i)
			}
		case "stdin":
			// no further constraints
		case "winlog":
			if runtime.GOOS != "windows" {
				return fmt.Errorf("inputs[%d]: winlog requires Windows (current OS: %s)", i, runtime.GOOS)
			}
			if in.Channel == "" {
				return fmt.Errorf("inputs[%d]: winlog input requires channel (e.g. \"Security\")", i)
			}
		case "syslog-udp":
			if in.Path == "" {
				return fmt.Errorf("inputs[%d]: syslog-udp input requires path (listen address, e.g. \":1514\")", i)
			}
		case "journald":
			if runtime.GOOS != "linux" {
				return fmt.Errorf("inputs[%d]: journald requires Linux (current OS: %s)", i, runtime.GOOS)
			}
		default:
			return fmt.Errorf(
				"inputs[%d]: unknown type %q — valid types: file, stdin, winlog, syslog-udp, journald\n"+
					"  (aliases accepted: winevent/winevt/windows-event → winlog, journal → journald)",
				i, in.Type,
			)
		}
	}
	return nil
}

func (h HeartbeatOpts) isEmpty() bool { return !h.Enabled && h.Interval == 0 }

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
