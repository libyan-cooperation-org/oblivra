// Package listeners contains network/file ingest sources. The syslog listener
// listens on UDP (RFC 5424/3164) and forwards each datagram into the ingest
// pipeline after parsing.
package listeners

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"

	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/parsers"
)

type SyslogUDP struct {
	log      *slog.Logger
	pipeline *ingest.Pipeline
	addr     string
	tenant   string
	conn     *net.UDPConn
	count    atomic.Int64
}

type SyslogOptions struct {
	Addr     string // e.g. ":1514"
	TenantID string
}

// NewSyslogUDP returns a listener that hasn't started yet.
func NewSyslogUDP(log *slog.Logger, p *ingest.Pipeline, opts SyslogOptions) *SyslogUDP {
	if opts.Addr == "" {
		opts.Addr = ":1514"
	}
	if opts.TenantID == "" {
		opts.TenantID = "default"
	}
	return &SyslogUDP{log: log, pipeline: p, addr: opts.Addr, tenant: opts.TenantID}
}

// Start binds the UDP socket and processes datagrams until ctx is cancelled.
func (s *SyslogUDP) Start(ctx context.Context) error {
	udpAddr, err := net.ResolveUDPAddr("udp", s.addr)
	if err != nil {
		return fmt.Errorf("syslog resolve: %w", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("syslog listen: %w", err)
	}
	s.conn = conn
	s.log.Info("syslog UDP listening", "addr", conn.LocalAddr().String(), "tenant", s.tenant)

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	buf := make([]byte, 64*1024)
	for {
		n, peer, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			s.log.Warn("syslog read", "err", err)
			continue
		}
		raw := string(buf[:n])
		ev, perr := parsers.Parse(raw, parsers.FormatAuto)
		if perr != nil || ev == nil {
			s.log.Warn("syslog parse", "err", perr)
			continue
		}
		ev.TenantID = s.tenant
		ev.Provenance.IngestPath = "syslog-udp"
		if peer != nil {
			ev.Provenance.Peer = peer.IP.String()
		}
		ev.Provenance.Format = "syslog"
		ev.Provenance.Parser = string(parsers.FormatAuto)
		if err := s.pipeline.Submit(ctx, ev); err != nil {
			s.log.Error("syslog submit", "err", err)
			continue
		}
		s.count.Add(1)
	}
}

func (s *SyslogUDP) Count() int64 { return s.count.Load() }
func (s *SyslogUDP) Addr() string {
	if s.conn == nil {
		return s.addr
	}
	return s.conn.LocalAddr().String()
}
