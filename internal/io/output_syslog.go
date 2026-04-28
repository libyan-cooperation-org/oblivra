package io

// Syslog output — forward events as RFC 5424 messages over UDP or TCP.
//
// Use case: parallel-run alongside another SIEM during migration.
// Customer keeps their existing Splunk / QRadar / Elastic feed alive
// and points it at OBLIVRA-emitted syslog. Cuts over after they're
// satisfied with detection parity.
//
// Format: RFC 5424 always; receiving SIEMs that prefer 3164 will
// generally accept 5424 transparently. Frame-delimited on TCP using
// the octet-counting style (RFC 6587 §3.4.1) for robustness against
// embedded newlines.

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

type syslogOutputConfig struct {
	Target   string `yaml:"target"`   // "udp://host:514" | "tcp://host:514"
	Facility int    `yaml:"facility"` // RFC 5424 facility, default 1 (user)
	Severity int    `yaml:"severity"` // default 6 (informational)
	Hostname string `yaml:"hostname"` // default os.Hostname()
	AppName  string `yaml:"app_name"` // default "oblivra"
}

type SyslogOutput struct {
	id   string
	cfg  syslogOutputConfig
	log  *logger.Logger
	host string

	mu   sync.Mutex
	conn net.Conn
	addr string
	net_ string // "udp" | "tcp"
}

func NewSyslogOutputReal(id string, raw map[string]interface{}, log *logger.Logger) (*SyslogOutput, error) {
	cfg, err := decodeYAMLMap[syslogOutputConfig](raw)
	if err != nil {
		return nil, fmt.Errorf("output syslog %q: %w", id, err)
	}
	if cfg.Target == "" {
		return nil, fmt.Errorf("output syslog %q: target is required", id)
	}
	scheme, addr, ok := splitScheme(cfg.Target)
	if !ok || (scheme != "udp" && scheme != "tcp") {
		return nil, fmt.Errorf("output syslog %q: target must be udp://host:port or tcp://host:port", id)
	}
	if cfg.Facility == 0 {
		cfg.Facility = 1
	}
	if cfg.Severity == 0 {
		cfg.Severity = 6
	}
	if cfg.AppName == "" {
		cfg.AppName = "oblivra"
	}
	host := cfg.Hostname
	if host == "" {
		host, _ = os.Hostname()
	}
	return &SyslogOutput{
		id:   id,
		cfg:  cfg,
		log:  log.WithPrefix("output.syslog"),
		host: host,
		addr: addr,
		net_: scheme,
	}, nil
}

func (s *SyslogOutput) Name() string { return s.id }
func (s *SyslogOutput) Type() string { return "syslog" }

func (s *SyslogOutput) Write(_ context.Context, ev Event) error {
	msg := s.format(ev)
	return s.send(msg)
}

func (s *SyslogOutput) Flush(_ context.Context) error { return nil }

func (s *SyslogOutput) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
	return nil
}

// format renders the event as an RFC 5424 message:
//   <PRI>1 TIMESTAMP HOSTNAME APP - - - MSG
func (s *SyslogOutput) format(ev Event) string {
	pri := s.cfg.Facility*8 + s.cfg.Severity
	ts := ev.Timestamp.UTC().Format(time.RFC3339)
	host := ev.Host
	if host == "" {
		host = s.host
	}
	msg := ev.Raw
	if msg == "" && ev.Fields != nil {
		// No raw line — render fields as `k=v k=v` so downstream
		// SIEMs at least get something searchable.
		var b strings.Builder
		for k, v := range ev.Fields {
			fmt.Fprintf(&b, " %s=%v", k, v)
		}
		msg = strings.TrimSpace(b.String())
	}
	return fmt.Sprintf("<%d>1 %s %s %s - - - %s", pri, ts, host, s.cfg.AppName, msg)
}

func (s *SyslogOutput) send(msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		c, err := net.DialTimeout(s.net_, s.addr, 5*time.Second)
		if err != nil {
			return err
		}
		s.conn = c
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	var payload string
	if s.net_ == "tcp" {
		// Octet-counting frame: <length> <message>
		payload = fmt.Sprintf("%d %s", len(msg), msg)
	} else {
		payload = msg
	}
	if _, err := s.conn.Write([]byte(payload)); err != nil {
		// Drop the connection — next Write will reconnect.
		_ = s.conn.Close()
		s.conn = nil
		return err
	}
	return nil
}

// splitScheme parses "scheme://addr" → ("scheme", "addr", true).
// Returns false on malformed input.
func splitScheme(s string) (string, string, bool) {
	idx := strings.Index(s, "://")
	if idx <= 0 || idx+3 >= len(s) {
		return "", "", false
	}
	return s[:idx], s[idx+3:], true
}
