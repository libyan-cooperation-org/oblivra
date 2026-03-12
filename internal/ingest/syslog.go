package ingest

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// SyslogServer listens for incoming external log events
type SyslogServer struct {
	udpConn  *net.UDPConn
	tcpList  net.Listener
	pipeline *Pipeline
	log      *logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	port     int
}

// NewSyslogServer initializes but does not start the listener
func NewSyslogServer(pipeline *Pipeline, port int, log *logger.Logger) *SyslogServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &SyslogServer{
		pipeline: pipeline,
		port:     port,
		log:      log,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start binds the TCP and UDP ports and begins accepting external logs
func (s *SyslogServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)

	// 1. Start UDP Listener
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("resolve udp: %w", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen udp: %w", err)
	}
	s.udpConn = udpConn

	// 2. Start TCP Listener
	tcpList, err := net.Listen("tcp", addr)
	if err != nil {
		s.udpConn.Close()
		return fmt.Errorf("listen tcp: %w", err)
	}
	s.tcpList = tcpList

	s.log.Info("[INGEST] Syslog Server listening on %s (UDP/TCP)", addr)

	s.wg.Add(2)
	go s.serveUDP()
	go s.serveTCP()

	return nil
}

// Stop safely drops the connections and shuts down the workers
func (s *SyslogServer) Stop() {
	s.log.Info("[INGEST] Stopping Syslog server...")
	s.cancel()
	if s.udpConn != nil {
		s.udpConn.Close()
	}
	if s.tcpList != nil {
		s.tcpList.Close()
	}
	s.wg.Wait()
}

func (s *SyslogServer) serveUDP() {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("[PANIC] Recovered in Syslog serveUDP: %v", r)
		}
	}()
	defer s.wg.Done()

	// Max typical UDP syslog packet is 1024 to 8192 bytes
	buf := make([]byte, 8192)

	for {
		s.udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := s.udpConn.ReadFromUDP(buf)

		if err != nil {
			if s.ctx.Err() != nil {
				return // Context canceled
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Normal timeout, loop again
			}
			s.log.Error("[INGEST] UDP read error: %v", err)
			continue
		}

		rawLine := string(buf[:n])
		evt := AutoParse(rawLine)

		// If parser couldn't determine host, use the IP from the packet
		if evt.Host == "" || evt.Host == "unknown" {
			evt.Host = addr.IP.String()
		}

		if err := s.pipeline.QueueEvent(evt); err != nil {
			s.log.Warn("[INGEST] Dropped syslog UDP event from %s: %v", evt.Host, err)
		}
	}
}

func (s *SyslogServer) serveTCP() {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("[PANIC] Recovered in Syslog serveTCP: %v", r)
		}
	}()
	defer s.wg.Done()

	for {
		conn, err := s.tcpList.Accept()
		if err != nil {
			if s.ctx.Err() != nil {
				return // Context canceled
			}
			s.log.Error("[INGEST] TCP accept error: %v", err)
			continue
		}

		s.wg.Add(1)
		go s.handleTCPConnection(conn)
	}
}

func (s *SyslogServer) handleTCPConnection(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("[PANIC] Recovered in Syslog handleTCPConnection: %v", r)
		}
	}()
	defer s.wg.Done()
	defer conn.Close()

	remoteHost, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		rawLine := scanner.Text()
		if strings.TrimSpace(rawLine) == "" {
			continue
		}

		evt := AutoParse(rawLine)
		if evt.Host == "" || evt.Host == "unknown" {
			evt.Host = remoteHost
		}

		if err := s.pipeline.QueueEvent(evt); err != nil {
			s.log.Warn("[INGEST] Dropped syslog TCP event from %s: %v", evt.Host, err)
		}
	}

	if err := scanner.Err(); err != nil {
		// Connection closed or reset
	}
}
