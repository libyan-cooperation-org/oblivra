package io

// YAML config + hot-reload.
//
// Config shape:
//
//   tls:
//     mode: "on" | "off"
//     cert_file: "/etc/oblivra/cert.pem"
//     key_file: "/etc/oblivra/key.pem"
//
//   inputs:
//     - id: auth-log
//       type: file
//       paths: ["/var/log/auth.log"]
//       sourcetype: linux:auth
//
//   outputs:
//     - id: primary
//       type: oblivra
//       server: https://oblivra.internal:8443
//
//   pipeline:
//     - drop_if: 'event.source == "noisy-cron"'
//     - redact: [credit_card, ssn]
//
// Hot-reload: fsnotify on the config path; on WRITE, parse the new
// config and diff against current. Only restart plugins whose
// `id` is new OR whose config changed (compared by deep-equal of the
// raw `Raw` map). Plugins with unchanged config keep running.

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kingknull/oblivrashell/internal/logger"
	"gopkg.in/yaml.v3"
)

// Config is the full agent/server YAML config.
type Config struct {
	TLS      TLSConfig            `yaml:"tls"`
	Inputs   []PluginConfig       `yaml:"inputs"`
	Outputs  []PluginConfig       `yaml:"outputs"`
	Pipeline []map[string]any     `yaml:"pipeline"`
}

// TLSConfig controls the TLS-optional behaviour. See the security
// guardrails in api/rest.go and the SovereigntyService.
type TLSConfig struct {
	// Mode is "on" (default) or "off". Off triggers loud warnings
	// every 30s, the UI plaintext banner, and a 30-point sovereignty
	// score deduction. Refused entirely when OBLIVRA_PRODUCTION=1.
	Mode     string `yaml:"mode"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// IsTLSOff is the canonical accessor — config files might have
// "Off" / "OFF" / "no" / etc.; centralise the predicate here.
func (t TLSConfig) IsTLSOff() bool {
	switch t.Mode {
	case "off", "Off", "OFF", "no", "false", "disabled":
		return true
	}
	return false
}

// LoadConfig parses a YAML file at `path`. Returns a zero-valued
// Config if the file doesn't exist (caller decides whether that's
// fatal — the agent treats "no config" as "use defaults", the
// operator running `oblivra` standalone treats it as fatal).
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{TLS: TLSConfig{Mode: "on"}}, nil
		}
		return nil, fmt.Errorf("io: read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("io: parse config %s: %w", path, err)
	}
	if cfg.TLS.Mode == "" {
		cfg.TLS.Mode = "on"
	}
	return &cfg, nil
}

// Watcher hot-reloads a config file. Construct one with NewWatcher,
// call Watch in a goroutine, and read reload events from C().
type Watcher struct {
	path string
	fsw  *fsnotify.Watcher
	log  *logger.Logger

	out chan *Config
	mu  sync.Mutex
	// Debounce window: editors (vim, nano, VS Code) often write +
	// rename + chmod in quick succession; we coalesce events that
	// arrive within `debounce` of each other.
	debounce time.Duration
}

// NewWatcher constructs a fsnotify-backed watcher on `path`. The
// channel returned by C() emits a parsed *Config each time the file
// changes; nil is sent on parse failures so callers can decide
// whether to keep running with the previous config.
func NewWatcher(path string, log *logger.Logger) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fsw.Add(path); err != nil {
		fsw.Close()
		return nil, err
	}
	return &Watcher{
		path:     path,
		fsw:      fsw,
		log:      log.WithPrefix("io.config"),
		out:      make(chan *Config, 4),
		debounce: 500 * time.Millisecond,
	}, nil
}

// C returns the channel that emits a parsed *Config on each reload.
// nil values represent parse failures.
func (w *Watcher) C() <-chan *Config { return w.out }

// Watch is the blocking event loop. Run in a goroutine.
func (w *Watcher) Watch() {
	var debounceTimer *time.Timer
	for {
		select {
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			// Debounce: reset the timer; only when it fires do we reload.
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(w.debounce, w.reload)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			w.log.Warn("watcher error: %v", err)
		}
	}
}

// Stop closes the underlying watcher and the output channel.
func (w *Watcher) Stop() {
	w.fsw.Close()
	w.mu.Lock()
	defer w.mu.Unlock()
	close(w.out)
}

func (w *Watcher) reload() {
	cfg, err := LoadConfig(w.path)
	if err != nil {
		w.log.Error("reload failed: %v — keeping previous config", err)
		w.mu.Lock()
		select {
		case w.out <- nil:
		default:
		}
		w.mu.Unlock()
		return
	}
	w.log.Info("config reloaded: %d input(s), %d output(s)", len(cfg.Inputs), len(cfg.Outputs))
	w.mu.Lock()
	defer w.mu.Unlock()
	select {
	case w.out <- cfg:
	default:
		// If nothing's reading, drop the older event. The latest
		// config is what matters.
	}
}
