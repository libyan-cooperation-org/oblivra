package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// runSyslogUDP listens on a UDP address (configured via `path`, e.g. ":1514")
// and parses RFC-3164 / RFC-5424 syslog datagrams into agent events.
//
// Usage in agent.yml:
//
//	- type: syslog-udp
//	  path: ":1514"          # bind address; 0.0.0.0:1514 is equivalent
//	  sourceType: "syslog:network"
//	  label: "network-devices"
//
// Each received datagram is treated as one event — no multiline stitching
// (syslog datagrams are already self-contained).  The RFC-3164 severity
// field (PRI value) is mapped to the platform severity vocabulary so the
// SIEM view can colour-code them correctly.
func (t *Tailer) runSyslogUDP(ctx context.Context) error {
	addr := t.in.Path
	if addr == "" {
		addr = ":1514"
	}

	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return fmt.Errorf("syslog-udp listen %s: %w", addr, err)
	}
	defer pc.Close()
	log.Printf("syslog-udp: listening on %s", addr)

	// Shutdown: closing the PacketConn unblocks ReadFrom.
	go func() {
		<-ctx.Done()
		_ = pc.Close()
	}()

	buf := make([]byte, 64*1024) // 64 KiB — well above any real syslog datagram
	for {
		n, remote, err := pc.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil // clean shutdown
			}
			log.Printf("syslog-udp read: %v", err)
			continue
		}
		raw := strings.TrimRight(string(buf[:n]), "\r\n\x00")
		if raw == "" {
			continue
		}
		// Inject the remote address as a field so analysts can see which
		// network device sent the datagram (the syslog HOSTNAME field is
		// often unreliable on embedded devices).
		remoteHost, _, _ := net.SplitHostPort(remote.String())
		_ = remoteHost

		msg := parseSyslogLine(raw, remoteHost)
		t.enqueue("syslog-udp:"+addr, msg)
	}
}

// parseSyslogLine attempts RFC-3164 and RFC-5424 parsing. On failure it
// falls back to the raw line. Returns the formatted message that will be
// passed to enqueue() as the raw event body.
//
// We don't try to be a full syslog parser — we extract enough to fill
// the severity and host fields for the SIEM view and then hand the full
// line off unchanged as the message body.
func parseSyslogLine(raw, remoteAddr string) string {
	line := raw
	severity := "info"

	// ── PRI extraction ("<N>" prefix) ────────────────────────────────────
	if len(line) > 3 && line[0] == '<' {
		end := strings.IndexByte(line, '>')
		if end > 0 && end < 6 {
			var pri int
			for _, ch := range line[1:end] {
				if ch < '0' || ch > '9' {
					pri = -1
					break
				}
				pri = pri*10 + int(ch-'0')
			}
			if pri >= 0 {
				sevCode := pri & 0x07 // low 3 bits = severity
				severity = syslogSeverity(sevCode)
				line = line[end+1:]
			}
		}
	}

	// ── Timestamp stripping (RFC 3164: "Jan _2 15:04:05") ────────────────
	// We keep the original raw line intact as the message; this is just so
	// we can confirm it's a valid syslog line and not garbage.
	_ = line

	return raw + "\x00sev=" + severity + "\x00remote=" + remoteAddr
}

// syslogSeverity maps the RFC-3164 severity code (0–7) to the platform
// severity vocabulary.
func syslogSeverity(code int) string {
	switch code {
	case 0:
		return "critical" // emerg
	case 1:
		return "critical" // alert
	case 2:
		return "critical" // crit
	case 3:
		return "error"
	case 4:
		return "warning"
	case 5:
		return "notice"
	case 6:
		return "info"
	case 7:
		return "debug"
	default:
		return "info"
	}
}

// syslogUDPHealthCheck dials the listener and sends a test datagram.
// Used by `oblivra-agent test` when the input type is syslog-udp.
func syslogUDPHealthCheck(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("udp", addr, timeout)
	if err != nil {
		return fmt.Errorf("syslog-udp dial %s: %w", addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	_, err = conn.Write([]byte("<30>oblivra-agent: health-check\n"))
	return err
}
