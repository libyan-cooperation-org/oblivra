package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

// RemoteProvider implements the Provider interface by communicating with an external vault daemon.
type RemoteProvider struct {
	mu         sync.Mutex
	socketPath string
	conn       net.Conn
}

// NewRemoteProvider creates a client for the isolated vault daemon.
func NewRemoteProvider(socketPath string) *RemoteProvider {
	return &RemoteProvider{
		socketPath: socketPath,
	}
}

func (p *RemoteProvider) Name() string { return "remote-vault" }

func (p *RemoteProvider) Dependencies() []string { return nil }

func (p *RemoteProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn == nil {
		conn, err := net.Dial("unix", p.socketPath)
		if err != nil {
			return fmt.Errorf("connect to vault daemon: %w", err)
		}
		p.conn = conn
	}
	return nil
}

func (p *RemoteProvider) Stop(ctx context.Context) error {
	p.Lock()
	return nil
}

func (p *RemoteProvider) Ping(ctx context.Context) error {
	_, err := p.call("ping", nil)
	return err
}

func (p *RemoteProvider) IsSetup() bool {
	// For remote provider, we assume it's setup if the daemon is reachable and says so,
	// but usually setup is a local filesystem check.
	return true 
}

func (p *RemoteProvider) IsUnlocked() bool {
	resp, err := p.call("ping", nil)
	if err != nil {
		return false
	}
	return string(resp) == "pong"
}

func (p *RemoteProvider) Unlock(password string, _ []byte, _ bool) error {
	_, err := p.call("unlock", []byte(password))
	return err
}

func (p *RemoteProvider) Lock() {
	_, _ = p.call("lock", nil)
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}
}

func (p *RemoteProvider) UnlockWithKeychain() error {
	return fmt.Errorf("auto-unlock not supported in isolated mode")
}

func (p *RemoteProvider) Setup(password string, serial string) error {
	return fmt.Errorf("setup not supported via remote provider")
}

func (p *RemoteProvider) SetupWithTPM(password string, serial string, pcr int) error {
	return fmt.Errorf("TPM setup not supported via remote provider")
}

func (p *RemoteProvider) GetYubiKeySerial() string { return "" }
func (p *RemoteProvider) IsTPMBound() bool         { return false }
func (p *RemoteProvider) HasKeychainEntry() bool   { return false }

func (p *RemoteProvider) GetPassword(id string) ([]byte, error) {
	return nil, fmt.Errorf("direct password retrieval not supported in isolated mode")
}

func (p *RemoteProvider) GetPrivateKey(id string) ([]byte, string, error) {
	return nil, "", fmt.Errorf("direct key retrieval not supported in isolated mode")
}

func (p *RemoteProvider) NuclearDestruction() error {
	return fmt.Errorf("nuclear destruction must be performed locally or on the daemon host")
}

func (p *RemoteProvider) Encrypt(data []byte) ([]byte, error) {
	return p.call("encrypt", data)
}

func (p *RemoteProvider) Decrypt(data []byte) ([]byte, error) {
	return p.call("decrypt", data)
}

func (p *RemoteProvider) GetSystemKey(purpose string) ([]byte, error) {
	return p.call("system_key", []byte(purpose))
}

func (p *RemoteProvider) GetTenantKey(tenantID string) ([]byte, error) {
	return p.call("tenant_key", []byte(tenantID))
}

func (p *RemoteProvider) AccessMasterKey(fn func(key []byte) error) error {
	key, err := p.call("master_key", nil)
	if err != nil {
		return err
	}
	defer ZeroSlice(key)
	return fn(key)
}

// rpcRequest/Response must match cmd/vault-daemon/main.go
type rpcRequest struct {
	Op      string `json:"op"`
	Payload []byte `json:"payload"`
}

type rpcResponse struct {
	Result []byte `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func (p *RemoteProvider) call(op string, payload []byte) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		conn, err := net.Dial("unix", p.socketPath)
		if err != nil {
			return nil, fmt.Errorf("connect to vault daemon: %w", err)
		}
		p.conn = conn
	}

	req := rpcRequest{Op: op, Payload: payload}
	data, _ := json.Marshal(req)

	// Write frame: [length:4][payload]
	frame := make([]byte, 4+len(data))
	l := len(data)
	frame[0], frame[1], frame[2], frame[3] = byte(l>>24), byte(l>>16), byte(l>>8), byte(l)
	copy(frame[4:], data)

	if _, err := p.conn.Write(frame); err != nil {
		p.conn.Close()
		p.conn = nil
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Read response frame
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(p.conn, lenBuf); err != nil {
		p.conn.Close()
		p.conn = nil
		return nil, fmt.Errorf("read response length: %w", err)
	}

	respLen := int(lenBuf[0])<<24 | int(lenBuf[1])<<16 | int(lenBuf[2])<<8 | int(lenBuf[3])
	respData := make([]byte, respLen)
	if _, err := io.ReadFull(p.conn, respData); err != nil {
		p.conn.Close()
		p.conn = nil
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var resp rpcResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("vault error: %s", resp.Error)
	}

	return resp.Result, nil
}
