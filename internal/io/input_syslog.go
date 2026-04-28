package io

// Syslog input — UDP and/or TCP listener that parses RFC 3164 (BSD)
// and RFC 5424 (modern). Covers every network device on the planet.
//
// Behaviour:
//   • Bind to UDP and/or TCP — config can pick either or both
//   • Each datagram / line is one event
//   • Parse priority, facility, severity, hostname, app, message
//   • If parse fails, ship the raw line in `raw` and tag
//     `sourcetype: "syslog:malformed"` — visibility over silent loss
//
// Out of scope (defer):
//   • RELP transport (RFC 6587) — niche; few devices outside rsyslog
//   • Hot-reload binding change without restart — restart input
//     instance on bind-port change instead
//
// Config:
//
//   - id: net-syslog
//     type: syslog
//     listen_udp: "0.0.0.0:514"   # one or both required
//     listen_tcp: "0.0.0.0:514"
//     sourcetype: "syslog:rfc5424"  # default; override per-feed

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

type syslogInputConfig struct {
	ListenUDP  string `yaml:"listen_udp"`
	ListenTCP  string `yaml:"listen_tcp"`
	Sourcetype string `yaml:"sourcetype"`
}

type SyslogInput struct {
	id  string
	cfg syslogInputConfig
	log *logger.Logger

	udpConn *net.UDPConn
	tcpLn   *net.TCPListener

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewSyslogInputReal(id string, raw map[string]interface{}, log *logger.Logger) (*SyslogInput, error) {
	cfg, err := decodeYAMLMap[syslogInputConfig](raw)
	if err != nil {
		return nil, fmt.Errorf("input syslog %q: %w", id, err)
	}
	if cfg.ListenUDP == "" && cfg.ListenTCP == "" {
		return nil, fmt.Errorf("input syslog %q: at least one of listen_udp / listen_tcp is required", id)
	}
	if cfg.Sourcetype == "" {
		cfg.Sourcetype = "syslog:rfc5424"
	}
	return &SyslogInput{
		id:  id,
		cfg: cfg,
		log: log.WithPrefix("input.syslog"),
	}, nil
}

func (s *SyslogInput) Name() string { return s.id }
func (s *SyslogInput) Type() string { return "syslog" }

func (s *SyslogInput) Start(ctx context.Context, out chan<- Event) error {
	pluginCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	if s.cfg.ListenUDP != "" {
		addr, err := net.ResolveUDPAddr("udp", s.cfg.ListenUDP)
		if err != nil {
			return fmt.Errorf("syslog %q: resolve udp: %w", s.id, err)
		}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return fmt.Errorf("syslog %q: listen udp %s: %w", s.id, s.cfg.ListenUDP, err)
		}
		s.udpConn = conn
		s.log.Info("[%s] listening udp %s", s.id, s.cfg.ListenUDP)
		s.wg.Add(1)
		go s.udpLoop(pluginCtx, out)
	}

	if s.cfg.ListenTCP != "" {
		addr, err := net.ResolveTCPAddr("tcp", s.cfg.ListenTCP)
		if err != nil {
			return fmt.Errorf("syslog %q: resolve tcp: %w", s.id, err)
		}
		ln, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return fmt.Errorf("syslog %q: listen tcp %s: %w", s.id, s.cfg.ListenTCP, err)
		}
		s.tcpLn = ln
		s.log.Info("[%s] listening tcp %s", s.id, s.cfg.ListenTCP)
		s.wg.Add(1)
		go s.tcpAcceptLoop(pluginCtx, out)
	}
	return nil
}

func (s *SyslogInput) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.udpConn != nil {
		_ = s.udpConn.Close()
	}
	if s.tcpLn != nil {
		_ = s.tcpLn.Close()
	}
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.log.Warn("[%s] stop timed out", s.id)
	}
	return nil
}

func (s *SyslogInput) udpLoop(ctx context.Context, out chan<- Event) {
	defer s.wg.Done()
	buf := make([]byte, 65535) // max UDP datagram
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = s.udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, src, err := s.udpConn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			s.log.Warn("[%s] udp read: %v", s.id, err)
			continue
		}
		ev := s.parse(string(buf[:n]), src.IP.String())
		select {
		case out <- ev:
		case <-ctx.Done():
			return
		}
	}
}

func (s *SyslogInput) tcpAcceptLoop(ctx context.Context, out chan<- Event) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = s.tcpLn.SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := s.tcpLn.AcceptTCP()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			s.log.Warn("[%s] tcp accept: %v", s.id, err)
			continue
		}
		s.wg.Add(1)
		go s.tcpClientLoop(ctx, conn, out)
	}
}

func (s *SyslogInput) tcpClientLoop(ctx context.Context, conn *net.TCPConn, out chan<- Event) {
	defer s.wg.Done()
	defer conn.Close()
	src := conn.RemoteAddr().(*net.TCPAddr).IP.String()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		line := scanner.Text()
		if line == "" {
			continue
		}
		ev := s.parse(line, src)
		select {
		case out <- ev:
		case <-ctx.Done():
			return
		}
	}
}

// parse handles both RFC 3164 and RFC 5424 — auto-detects by looking
// for the version digit (`<...>1 `) that 5424 mandates after the PRI.
//
// 3164 example: "<34>Oct 11 22:14:15 mymachine su: 'su root' failed"
// 5424 example: "<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 - …"
//
// We don't fully validate either spec — just extract the obvious
// fields. Anything that doesn't parse goes through with raw line +
// sourcetype "syslog:malformed".
func (s *SyslogInput) parse(line, srcIP string) Event {
	ev := Event{
		Timestamp:  time.Now().UTC(),
		Source:     "syslog:" + srcIP,
		Sourcetype: s.cfg.Sourcetype,
		Host:       srcIP,
		Raw:        line,
		InputID:    s.id,
		Fields:     map[string]any{},
	}

	// Extract PRI: <NN>...
	if !strings.HasPrefix(line, "<") {
		ev.Sourcetype = "syslog:malformed"
		return ev
	}
	end := strings.Index(line, ">")
	if end < 1 {
		ev.Sourcetype = "syslog:malformed"
		return ev
	}
	priStr := line[1:end]
	pri, err := strconv.Atoi(priStr)
	if err != nil {
		ev.Sourcetype = "syslog:malformed"
		return ev
	}
	ev.Fields["facility"] = pri >> 3
	ev.Fields["severity"] = pri & 7
	rest := line[end+1:]

	// Detect RFC 5424 ("1 " right after PRI)
	if len(rest) >= 2 && rest[0] == '1' && rest[1] == ' ' {
		// RFC 5424: "1 TIMESTAMP HOSTNAME APP-NAME PROCID MSGID [SD-ELEMENT] MSG"
		parts := strings.SplitN(rest[2:], " ", 6)
		if len(parts) >= 5 {
			ev.Fields["timestamp_raw"] = parts[0]
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				ev.Timestamp = t
			}
			if parts[1] != "-" {
				ev.Host = parts[1]
				ev.Fields["hostname"] = parts[1]
			}
			ev.Fields["app"] = parts[2]
			ev.Fields["procid"] = parts[3]
			ev.Fields["msgid"] = parts[4]
			if len(parts) == 6 {
				ev.Fields["msg"] = parts[5]
			}
		}
		return ev
	}

	// RFC 3164: "MMM DD HH:MM:SS HOSTNAME TAG[PID]: MSG"
	parts := strings.SplitN(rest, " ", 5)
	if len(parts) >= 4 {
		ev.Fields["timestamp_raw"] = strings.Join(parts[:3], " ")
		ev.Host = parts[3]
		ev.Fields["hostname"] = parts[3]
		if len(parts) == 5 {
			ev.Fields["msg"] = parts[4]
		}
	} else {
		ev.Sourcetype = "syslog:malformed"
	}
	return ev
}
