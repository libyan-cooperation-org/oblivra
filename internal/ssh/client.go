package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/pkg/sftp"
)

// ClientState represents the connection state
type ClientState int

const (
	StateDisconnected ClientState = iota
	StateConnecting
	StateConnected
	StateDegraded
	StateReconnecting
	StateError
)

func (s ClientState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateDegraded:
		return "degraded"
	case StateReconnecting:
		return "reconnecting"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// ClientEventType identifies event types
type ClientEventType int

const (
	EventStateChanged ClientEventType = iota
	EventDataReceived
	EventError
	EventClosed
)

// ClientEvent is emitted by the client
type ClientEvent struct {
	Type    ClientEventType
	Data    []byte
	State   ClientState
	Error   error
	Message string
}

// Client manages a single SSH connection
type Client struct {
	mu     sync.RWMutex
	config ConnectionConfig
	state  ClientState

	conn    *ssh.Client
	session *ssh.Session

	stdin  io.WriteCloser
	stdout io.Reader
	stderr io.Reader

	events chan ClientEvent

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	bytesIn     int64
	bytesOut    int64
	connectedAt time.Time
}

// NewClient creates a new SSH client
func NewClient(cfg ConnectionConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config: cfg,
		state:  StateDisconnected,
		events: make(chan ClientEvent, 4096),
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// Events returns the read-only event channel
func (c *Client) Events() <-chan ClientEvent { return c.events }

// State returns the current connection state
func (c *Client) State() ClientState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// RawClient returns the underlying golang.org/x/crypto/ssh Client
func (c *Client) RawClient() *ssh.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// Connect establishes the SSH connection
func (c *Client) Connect() error {
	c.mu.Lock()
	if c.state == StateConnected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	c.setState(StateConnecting)
	c.mu.Unlock()

	authMethods, err := buildAuthMethods(&c.config)
	if err != nil {
		c.setError(fmt.Errorf("build auth: %w", err))
		return err
	}

	hostKeyCallback, err := buildHostKeyCallback(c.config.StrictHostKey)
	if err != nil {
		c.setError(fmt.Errorf("host key callback: %w", err))
		return err
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.config.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         c.config.ConnectTimeout,
		Config: ssh.Config{
			Ciphers: []string{
				"chacha20-poly1305@openssh.com",
				"aes256-gcm@openssh.com",
				"aes128-gcm@openssh.com",
				"aes256-ctr",
				"aes192-ctr",
				"aes128-ctr",
			},
		},
	}

	var conn *ssh.Client
	if len(c.config.JumpHosts) > 0 {
		conn, err = c.connectViaJumpHosts(sshConfig)
	} else {
		conn, err = c.connectDirect(sshConfig)
	}

	if err != nil {
		c.setError(fmt.Errorf("connect: %w", err))
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.connectedAt = time.Now()
	c.setState(StateConnected)
	c.mu.Unlock()

	if c.config.KeepAliveInterval > 0 {
		go c.keepAlive()
	}

	return nil
}

// connectDirect connects directly to the target host
func (c *Client) connectDirect(sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	addr := c.config.Address()
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return conn, nil
}

// connectViaJumpHosts connects through one or more jump/bastion hosts
func (c *Client) connectViaJumpHosts(finalConfig *ssh.ClientConfig) (*ssh.Client, error) {
	var currentConn *ssh.Client

	for i, jump := range c.config.JumpHosts {
		jumpAuth, err := buildJumpHostAuth(&jump)
		if err != nil {
			return nil, fmt.Errorf("jump host %d auth: %w", i, err)
		}

		hostKeyCallback, err := buildHostKeyCallback(false)
		if err != nil {
			return nil, fmt.Errorf("jump host %d hostkey: %w", i, err)
		}

		jumpConfig := &ssh.ClientConfig{
			User:            jump.Username,
			Auth:            jumpAuth,
			HostKeyCallback: hostKeyCallback,
			Timeout:         c.config.ConnectTimeout,
		}

		jumpAddr := fmt.Sprintf("%s:%d", jump.Host, jump.Port)

		if currentConn == nil {
			currentConn, err = ssh.Dial("tcp", jumpAddr, jumpConfig)
			if err != nil {
				return nil, fmt.Errorf("dial jump %d (%s): %w", i, jumpAddr, err)
			}
		} else {
			netConn, err := currentConn.Dial("tcp", jumpAddr)
			if err != nil {
				return nil, fmt.Errorf("tunnel to jump %d (%s): %w", i, jumpAddr, err)
			}

			ncc, chans, reqs, err := ssh.NewClientConn(netConn, jumpAddr, jumpConfig)
			if err != nil {
				return nil, fmt.Errorf("handshake jump %d: %w", i, err)
			}

			currentConn = ssh.NewClient(ncc, chans, reqs)
		}
	}

	targetAddr := c.config.Address()
	netConn, err := currentConn.Dial("tcp", targetAddr)
	if err != nil {
		return nil, fmt.Errorf("tunnel to target (%s): %w", targetAddr, err)
	}

	ncc, chans, reqs, err := ssh.NewClientConn(netConn, targetAddr, finalConfig)
	if err != nil {
		return nil, fmt.Errorf("handshake target: %w", err)
	}

	return ssh.NewClient(ncc, chans, reqs), nil
}

// buildJumpHostAuth creates auth methods for a jump host
func buildJumpHostAuth(jump *JumpHostConfig) ([]ssh.AuthMethod, error) {
	cfg := &ConnectionConfig{
		AuthMethod: jump.AuthMethod,
		Password:   jump.Password,
		PrivateKey: jump.PrivateKey,
		Passphrase: jump.Passphrase,
	}
	return buildAuthMethods(cfg)
}

// StartShell starts an interactive shell session
func (c *Client) StartShell() error {
	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("not connected")
	}

	session, err := c.conn.NewSession()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("new session: %w", err)
	}
	c.session = session
	c.mu.Unlock()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}

	if err := session.RequestPty(c.config.TermType, c.config.Rows, c.config.Cols, modes); err != nil {
		return fmt.Errorf("request pty: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	c.mu.Lock()
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr
	c.mu.Unlock()

	if err := session.Shell(); err != nil {
		return fmt.Errorf("start shell: %w", err)
	}

	go c.readOutput(stdout, false)
	go c.readOutput(stderr, true)

	go func() {
		err := session.Wait()
		if err != nil {
			c.emit(ClientEvent{Type: EventError, Error: err, Message: "session ended with error"})
		}
		c.emit(ClientEvent{Type: EventClosed, Message: "session closed"})
		close(c.done)
	}()

	return nil
}

// Write sends data to the SSH session stdin
func (c *Client) Write(data []byte) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.stdin == nil {
		return 0, fmt.Errorf("no active session")
	}

	n, err := c.stdin.Write(data)
	c.bytesOut += int64(n)
	return n, err
}

// Resize changes the terminal size
func (c *Client) Resize(cols, rows int) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil {
		return fmt.Errorf("no active session")
	}

	return c.session.WindowChange(rows, cols)
}

// Close terminates the SSH connection gracefully
func (c *Client) Close() error {
	c.cancel()
	
	// Graceful Wait: Wait up to 5 seconds for active commands/shells to flush and close
	if c.done != nil {
		select {
		case <-c.done:
		case <-time.After(5 * time.Second):
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error

	if c.session != nil {
		if err := c.session.Close(); err != nil {
			firstErr = err
		}
		c.session = nil
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		c.conn = nil
	}

	c.setState(StateDisconnected)
	return firstErr
}

// readOutput reads from a reader and emits events
func (c *Client) readOutput(r io.Reader, _ bool) {
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		n, err := r.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])

			c.mu.Lock()
			c.bytesIn += int64(n)
			c.mu.Unlock()

			c.emit(ClientEvent{Type: EventDataReceived, Data: data})
		}

		if err != nil {
			if err != io.EOF {
				c.emit(ClientEvent{Type: EventError, Error: err, Message: "read error"})
			}
			return
		}
	}
}

// keepAlive sends periodic keepalive requests and tracks network latency
func (c *Client) keepAlive() {
	ticker := time.NewTicker(c.config.KeepAliveInterval)
	defer ticker.Stop()

	missed := 0
	slowCount := 0

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			currentState := c.state
			c.mu.RUnlock()

			if conn == nil {
				return
			}

			start := time.Now()
			// SendRequest blocks until the server replies with a wait response
			_, _, err := conn.SendRequest("keepalive@openssh.com", true, nil)
			latency := time.Since(start)

			if err != nil {
				missed++
				if missed >= c.config.KeepAliveMax {
					c.emit(ClientEvent{
						Type:    EventError,
						Error:   fmt.Errorf("keepalive timeout after %d missed", missed),
						Message: "connection lost",
					})
					c.Close()
					return
				}
			} else {
				missed = 0

				// Adaptive Degradation: If latency is unusually high (> 500ms),
				// we warn the client without outright killing the socket yet.
				if latency > 500*time.Millisecond {
					slowCount++
					if slowCount > 2 && currentState == StateConnected {
						c.mu.Lock()
						c.setState(StateDegraded)
						c.mu.Unlock()
					}
				} else {
					// Recover state if latency normalizes
					slowCount = 0
					if currentState == StateDegraded {
						c.mu.Lock()
						c.setState(StateConnected)
						c.mu.Unlock()
					}
				}
			}
		}
	}
}

// setState updates state and emits event (must be called with mu held or before lock released)
func (c *Client) setState(state ClientState) {
	c.state = state
	c.emit(ClientEvent{Type: EventStateChanged, State: state})
}

// setError sets error state
func (c *Client) setError(err error) {
	c.mu.Lock()
	c.setState(StateError)
	c.mu.Unlock()

	c.emit(ClientEvent{Type: EventError, State: StateError, Error: err, Message: err.Error()})
}

// emit sends an event to the internal channel.
// Important: data events are non-blocking to prevent reader goroutines from deadlocking
// during high-bandwidth output if the consumer (Session/UI) is slow.
func (c *Client) emit(event ClientEvent) {
	if event.Type == EventDataReceived {
		select {
		case c.events <- event:
		default:
			// Buffer full, drop data to prevent deadlock.
			// In a high-bandwidth scenario, it's better to lose some terminal frames
			// than to hang the entire application.
		}
		return
	}

	// For control events (State, Error, Closed), we allow a brief block to ensure delivery.
	select {
	case c.events <- event:
	case <-c.ctx.Done():
	case <-time.After(250 * time.Millisecond):
		// If we still can't send a control event, the system is likely in a fatal state.
	}
}

// Metrics returns connection metrics
func (c *Client) Metrics() (bytesIn int64, bytesOut int64, uptime time.Duration) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.state == StateConnected {
		uptime = time.Since(c.connectedAt)
	}
	return c.bytesIn, c.bytesOut, uptime
}

// ExecuteCommand runs a single command (non-interactive)
func (c *Client) ExecuteCommand(cmd string) ([]byte, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	session, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return output, fmt.Errorf("execute: %w", err)
	}

	return output, nil
}

// ExecuteCommandWithStdin runs a command and pipes the given string to stdin.
// This avoids embedding data in shell arguments, preventing process listing exposure.
func (c *Client) ExecuteCommandWithStdin(cmd string, stdinData string) ([]byte, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	session, err := conn.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	go func() {
		defer stdinPipe.Close()
		io.WriteString(stdinPipe, stdinData)
	}()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return output, fmt.Errorf("execute: %w", err)
	}

	return output, nil
}

// DialTCP opens a TCP connection through the SSH tunnel
func (c *Client) DialTCP(addr string) (net.Conn, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	return conn.Dial("tcp", addr)
}

// SftpClient returns an SFTP client using the existing SSH connection
func (c *Client) SftpClient() (*sftp.Client, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	return sftp.NewClient(conn)
}
