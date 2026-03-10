package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TunnelType defines the type of port forwarding
type TunnelType string

const (
	TunnelLocal   TunnelType = "local"   // -L
	TunnelRemote  TunnelType = "remote"  // -R
	TunnelDynamic TunnelType = "dynamic" // -D (SOCKS5)
)

// TunnelState represents tunnel lifecycle
type TunnelState string

const (
	TunnelStateActive TunnelState = "active"
	TunnelStateClosed TunnelState = "closed"
	TunnelStateError  TunnelState = "error"
)

// TunnelConfig defines a port forwarding tunnel
type TunnelConfig struct {
	Type       TunnelType `json:"type"`
	LocalHost  string     `json:"local_host"`
	LocalPort  int        `json:"local_port"`
	RemoteHost string     `json:"remote_host"`
	RemotePort int        `json:"remote_port"`
}

// Tunnel manages a single port forwarding tunnel
type Tunnel struct {
	ID        string       `json:"id"`
	Config    TunnelConfig `json:"config"`
	State     TunnelState  `json:"state"`
	StartedAt time.Time    `json:"started_at"`

	client   *Client
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	conns    int64
}

// NewTunnel creates a new tunnel
func NewTunnel(client *Client, cfg TunnelConfig) *Tunnel {
	ctx, cancel := context.WithCancel(context.Background())
	return &Tunnel{
		ID:     uuid.New().String(),
		Config: cfg,
		State:  TunnelStateClosed,
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start opens the tunnel
func (t *Tunnel) Start() error {
	switch t.Config.Type {
	case TunnelLocal:
		return t.startLocal()
	case TunnelRemote:
		return t.startRemote()
	case TunnelDynamic:
		return t.startDynamic()
	default:
		return fmt.Errorf("unsupported tunnel type: %s", t.Config.Type)
	}
}

// startLocal implements local port forwarding (-L)
func (t *Tunnel) startLocal() error {
	localAddr := net.JoinHostPort(t.Config.LocalHost, strconv.Itoa(t.Config.LocalPort))
	remoteAddr := net.JoinHostPort(t.Config.RemoteHost, strconv.Itoa(t.Config.RemotePort))

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		t.setState(TunnelStateError)
		return fmt.Errorf("listen on %s: %w", localAddr, err)
	}

	t.mu.Lock()
	t.listener = listener
	t.StartedAt = time.Now()
	t.setState(TunnelStateActive)
	t.mu.Unlock()

	go func() {
		for {
			select {
			case <-t.ctx.Done():
				return
			default:
			}
			localConn, err := listener.Accept()
			if err != nil {
				select {
				case <-t.ctx.Done():
					return
				default:
					continue
				}
			}
			go func() {
				defer localConn.Close()
				remoteConn, err := t.client.DialTCP(remoteAddr)
				if err != nil {
					return
				}
				defer remoteConn.Close()
				t.mu.Lock()
				t.conns++
				t.mu.Unlock()
				t.biCopy(localConn, remoteConn)
			}()
		}
	}()

	return nil
}

// startRemote implements remote port forwarding (-R)
func (t *Tunnel) startRemote() error {
	t.mu.RLock()
	conn := t.client.conn
	t.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("SSH client not connected")
	}

	remoteAddr := net.JoinHostPort(t.Config.RemoteHost, strconv.Itoa(t.Config.RemotePort))
	localAddr := net.JoinHostPort(t.Config.LocalHost, strconv.Itoa(t.Config.LocalPort))

	listener, err := conn.Listen("tcp", remoteAddr)
	if err != nil {
		t.setState(TunnelStateError)
		return fmt.Errorf("remote listen on %s: %w", remoteAddr, err)
	}

	t.mu.Lock()
	t.listener = listener
	t.StartedAt = time.Now()
	t.setState(TunnelStateActive)
	t.mu.Unlock()

	go func() {
		for {
			select {
			case <-t.ctx.Done():
				return
			default:
			}
			remoteConn, err := listener.Accept()
			if err != nil {
				select {
				case <-t.ctx.Done():
					return
				default:
					continue
				}
			}
			go func() {
				defer remoteConn.Close()
				localConn, err := net.Dial("tcp", localAddr)
				if err != nil {
					return
				}
				defer localConn.Close()
				t.mu.Lock()
				t.conns++
				t.mu.Unlock()
				t.biCopy(localConn, remoteConn)
			}()
		}
	}()

	return nil
}

// startDynamic implements SOCKS5 proxy (-D)
func (t *Tunnel) startDynamic() error {
	localAddr := net.JoinHostPort(t.Config.LocalHost, strconv.Itoa(t.Config.LocalPort))

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		t.setState(TunnelStateError)
		return fmt.Errorf("listen on %s: %w", localAddr, err)
	}

	t.mu.Lock()
	t.listener = listener
	t.StartedAt = time.Now()
	t.setState(TunnelStateActive)
	t.mu.Unlock()

	go func() {
		for {
			select {
			case <-t.ctx.Done():
				return
			default:
			}
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-t.ctx.Done():
					return
				default:
					continue
				}
			}
			go t.handleSOCKS5(conn)
		}
	}()

	return nil
}

// handleSOCKS5 implements basic SOCKS5 proxy protocol
func (t *Tunnel) handleSOCKS5(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 256)

	// Read version + auth methods
	n, err := conn.Read(buf)
	if err != nil || n < 2 || buf[0] != 0x05 {
		return
	}
	conn.Write([]byte{0x05, 0x00}) // no auth required

	// Read connect request
	n, err = conn.Read(buf)
	if err != nil || n < 7 {
		return
	}
	if buf[0] != 0x05 || buf[1] != 0x01 {
		conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	var targetAddr string
	switch buf[3] {
	case 0x01: // IPv4
		if n < 10 {
			return
		}
		targetAddr = fmt.Sprintf("%d.%d.%d.%d:%d",
			buf[4], buf[5], buf[6], buf[7],
			int(buf[8])<<8|int(buf[9]))
	case 0x03: // Domain
		domainLen := int(buf[4])
		if n < 5+domainLen+2 {
			return
		}
		domain := string(buf[5 : 5+domainLen])
		port := int(buf[5+domainLen])<<8 | int(buf[5+domainLen+1])
		targetAddr = fmt.Sprintf("%s:%d", domain, port)
	case 0x04: // IPv6
		if n < 22 {
			return
		}
		targetAddr = fmt.Sprintf("[%x:%x:%x:%x:%x:%x:%x:%x]:%d",
			int(buf[4])<<8|int(buf[5]), int(buf[6])<<8|int(buf[7]),
			int(buf[8])<<8|int(buf[9]), int(buf[10])<<8|int(buf[11]),
			int(buf[12])<<8|int(buf[13]), int(buf[14])<<8|int(buf[15]),
			int(buf[16])<<8|int(buf[17]), int(buf[18])<<8|int(buf[19]),
			int(buf[20])<<8|int(buf[21]))
	default:
		conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	remoteConn, err := t.client.DialTCP(targetAddr)
	if err != nil {
		conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remoteConn.Close()

	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	t.mu.Lock()
	t.conns++
	t.mu.Unlock()
	t.biCopy(conn, remoteConn)
}

// biCopy copies data bidirectionally between two connections
func (t *Tunnel) biCopy(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() { io.Copy(a, b); done <- struct{}{} }()
	go func() { io.Copy(b, a); done <- struct{}{} }()
	<-done
}

// Stop closes the tunnel
func (t *Tunnel) Stop() error {
	t.cancel()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.listener != nil {
		t.listener.Close()
		t.listener = nil
	}
	t.setState(TunnelStateClosed)
	return nil
}

// setState updates the tunnel state
func (t *Tunnel) setState(state TunnelState) { t.State = state }

// ConnectionCount returns total connections handled
func (t *Tunnel) ConnectionCount() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.conns
}
