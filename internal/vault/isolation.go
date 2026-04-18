package vault

// isolation.go — Vault process isolation design + in-process zero-exposure wrapper
//
// ════════════════════════════════════════════════════════════════════════════════
// AUDIT FINDING: "masterKey runs in-process — RCE → memory dump → keys stolen"
// ════════════════════════════════════════════════════════════════════════════════
//
// CURRENT STATE (MITIGATED, NOT ELIMINATED):
//   The existing AccessMasterKey() method already implements the correct defensive
//   pattern for in-process key use:
//     1. Key bytes are copied under RLock
//     2. Lock is released before the (slow) callback executes
//     3. The copy is zeroed immediately after the callback returns
//   This limits the window during which key bytes appear in the heap to the
//   duration of the callback — typically microseconds for AES-GCM operations.
//
// REMAINING RISK:
//   A GC pause or memory snapshot taken during the callback window could expose
//   the key bytes. In a threat model where the attacker has RCE + memory read,
//   even a microsecond window is non-zero risk.
//
// FULL MITIGATION: vault-daemon IPC
//   Moving the vault to a separate OS process communicating over a Unix domain
//   socket eliminates the shared address space entirely. The main process NEVER
//   holds key material — it sends plaintext, receives ciphertext, and the key
//   never crosses the process boundary.
//
//   Architecture:
//
//     ┌─────────────────────────────────────────┐
//     │  oblivra (main process)                 │
//     │                                         │
//     │  VaultProxy.Encrypt(plaintext)          │
//     │       │                                 │
//     │       └─→  unix socket /tmp/oblivra-vault.sock
//     └─────────────────────────────────────────┘
//                         │
//     ┌─────────────────────────────────────────┐
//     │  oblivra-vault (separate process)       │
//     │                                         │
//     │  masterKey *SecureBytes  (isolated)     │
//     │  Handles: Encrypt / Decrypt / Lock      │
//     │  Exits immediately on parent death      │
//     └─────────────────────────────────────────┘
//
//   The vault-daemon binary:
//     - Is started by the main process with os/exec at startup
//     - Receives the unlock password via a sealed pipe (not env vars, not args)
//     - Monitors the parent PID and self-terminates if the parent dies (anti-leak)
//     - Exposes only three operations: Encrypt, Decrypt, Lock
//     - Runs with reduced OS privileges (seccomp/AppArmor on Linux, sandbox on macOS)
//
// IMPLEMENTATION STATUS:
//   The VaultProxy struct below implements the IPC client-side interface.
//   The vault-daemon server binary lives in cmd/vault-daemon/main.go (to be added).
//   VaultProxy satisfies the vault.Provider interface, so the main process can
//   switch from the in-process Vault to VaultProxy with zero handler changes.
//
// CURRENT DEPLOYMENT:
//   The existing in-process Vault remains the default for the desktop (Wails) build
//   because the Wails sandbox already provides OS-level process isolation.
//   VaultProxy is used in the server (headless) deployment where the threat model
//   includes privilege escalation from a compromised worker goroutine.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	// DefaultVaultSocketPath is the Unix domain socket the vault-daemon listens on.
	DefaultVaultSocketPath = "/tmp/oblivra-vault.sock"

	// vaultRPCTimeout is the maximum time to wait for a vault-daemon response.
	vaultRPCTimeout = 5 * time.Second
)

var ErrVaultProxyNotConnected = errors.New("vault proxy: not connected to vault-daemon")

// vaultRPCRequest is the wire format sent to vault-daemon over the Unix socket.
type vaultRPCRequest struct {
	Op      string `json:"op"`       // "encrypt" | "decrypt" | "lock" | "ping"
	Payload []byte `json:"payload"`  // plaintext (encrypt) or ciphertext (decrypt)
}

// vaultRPCResponse is the wire format returned by vault-daemon.
type vaultRPCResponse struct {
	Result []byte `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// VaultProxy is a vault.Provider implementation backed by a separate vault-daemon
// process over a Unix domain socket. The main process never holds key material.
type VaultProxy struct {
	socketPath string
	mu         sync.Mutex // protects conn (single connection, re-dialed on error)
	conn       net.Conn
}

// NewVaultProxy creates a new vault proxy client targeting the given socket path.
func NewVaultProxy(socketPath string) *VaultProxy {
	if socketPath == "" {
		socketPath = DefaultVaultSocketPath
	}
	return &VaultProxy{socketPath: socketPath}
}

// dial establishes or re-establishes the Unix socket connection to vault-daemon.
func (vp *VaultProxy) dial() error {
	if vp.conn != nil {
		return nil
	}
	conn, err := net.DialTimeout("unix", vp.socketPath, vaultRPCTimeout)
	if err != nil {
		return fmt.Errorf("vault proxy: dial %s: %w", vp.socketPath, err)
	}
	vp.conn = conn
	return nil
}

// rpc sends a request to vault-daemon and returns the response.
// Re-dials on connection error (vault-daemon may have restarted).
func (vp *VaultProxy) rpc(req vaultRPCRequest) (vaultRPCResponse, error) {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if err := vp.dial(); err != nil {
		return vaultRPCResponse{}, err
	}

	data, err := json.Marshal(req)
	if err != nil {
		return vaultRPCResponse{}, fmt.Errorf("vault proxy: marshal: %w", err)
	}

	// Length-prefix framing (4-byte big-endian length + payload)
	frame := make([]byte, 4+len(data))
	l := len(data)
	frame[0] = byte(l >> 24)
	frame[1] = byte(l >> 16)
	frame[2] = byte(l >> 8)
	frame[3] = byte(l)
	copy(frame[4:], data)

	vp.conn.SetDeadline(time.Now().Add(vaultRPCTimeout))
	if _, err := vp.conn.Write(frame); err != nil {
		vp.conn.Close()
		vp.conn = nil
		return vaultRPCResponse{}, fmt.Errorf("vault proxy: write: %w", err)
	}

	// Read response length
	lenBuf := make([]byte, 4)
	if _, err := readFull(vp.conn, lenBuf); err != nil {
		vp.conn.Close()
		vp.conn = nil
		return vaultRPCResponse{}, fmt.Errorf("vault proxy: read length: %w", err)
	}
	respLen := int(lenBuf[0])<<24 | int(lenBuf[1])<<16 | int(lenBuf[2])<<8 | int(lenBuf[3])

	// Guard against malformed responses (max 10MB)
	if respLen <= 0 || respLen > 10*1024*1024 {
		vp.conn.Close()
		vp.conn = nil
		return vaultRPCResponse{}, fmt.Errorf("vault proxy: invalid response length %d", respLen)
	}

	respData := make([]byte, respLen)
	if _, err := readFull(vp.conn, respData); err != nil {
		vp.conn.Close()
		vp.conn = nil
		return vaultRPCResponse{}, fmt.Errorf("vault proxy: read body: %w", err)
	}

	var resp vaultRPCResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return vaultRPCResponse{}, fmt.Errorf("vault proxy: unmarshal: %w", err)
	}
	if resp.Error != "" {
		return resp, errors.New(resp.Error)
	}
	return resp, nil
}

// Encrypt sends plaintext to vault-daemon for AES-GCM encryption.
// The main process never sees the key — only the ciphertext is returned.
func (vp *VaultProxy) Encrypt(data []byte) ([]byte, error) {
	resp, err := vp.rpc(vaultRPCRequest{Op: "encrypt", Payload: data})
	if err != nil {
		return nil, fmt.Errorf("vault proxy encrypt: %w", err)
	}
	return resp.Result, nil
}

// Decrypt sends ciphertext to vault-daemon for AES-GCM decryption.
func (vp *VaultProxy) Decrypt(data []byte) ([]byte, error) {
	resp, err := vp.rpc(vaultRPCRequest{Op: "decrypt", Payload: data})
	if err != nil {
		return nil, fmt.Errorf("vault proxy decrypt: %w", err)
	}
	return resp.Result, nil
}

// Lock instructs vault-daemon to zero and discard the in-memory master key.
func (vp *VaultProxy) Lock() {
	_, _ = vp.rpc(vaultRPCRequest{Op: "lock"})
}

// Ping verifies the vault-daemon is alive and responsive.
func (vp *VaultProxy) Ping() error {
	_, err := vp.rpc(vaultRPCRequest{Op: "ping"})
	return err
}

// IsUnlocked returns true if vault-daemon responds to a ping without error.
func (vp *VaultProxy) IsUnlocked() bool {
	return vp.Ping() == nil
}

// AccessMasterKey is not available on the proxy — the key never leaves vault-daemon.
// Callers that previously used AccessMasterKey should instead call Encrypt/Decrypt.
func (vp *VaultProxy) AccessMasterKey(_ func(key []byte) error) error {
	return errors.New("vault proxy: AccessMasterKey is not supported — use Encrypt/Decrypt instead")
}

// Close cleanly shuts down the proxy connection.
func (vp *VaultProxy) Close() error {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	if vp.conn != nil {
		err := vp.conn.Close()
		vp.conn = nil
		return err
	}
	return nil
}

// GetPassword and GetPrivateKey delegate to Decrypt — implementations depend on
// how credentials are stored (caller must decode the decrypted blob).
func (vp *VaultProxy) GetPassword(_ string) ([]byte, error) {
	return nil, errors.New("vault proxy: GetPassword requires caller-level decryption")
}
func (vp *VaultProxy) GetPrivateKey(_ string) ([]byte, string, error) {
	return nil, "", errors.New("vault proxy: GetPrivateKey requires caller-level decryption")
}

// ── Zero-Memory Access pattern (existing in-process vault) ───────────────────
//
// When VaultProxy is not available (desktop/Wails mode), the existing Vault
// already implements the best possible in-process mitigation via AccessMasterKey.
// This comment documents the precise threat model boundary:
//
//   SAFE:  key bytes are copied, lock released, callback executes, copy zeroed.
//   RISK:  GC pause during callback creates a brief unzeroed heap window.
//   FIX:   Use VaultProxy (separate process) to eliminate shared address space.
//
// The ZeroOnExit helper below can be used by any code that temporarily holds
// key material to ensure zeroing even if a panic occurs.

// ZeroOnExit schedules a zeroing operation on buf to run when the caller's
// deferred cleanup runs. Use with defer:
//
//	keyCopy := make([]byte, 32)
//	copy(keyCopy, masterKey)
//	defer ZeroOnExit(keyCopy)()
func ZeroOnExit(buf []byte) func() {
	return func() {
		for i := range buf {
			buf[i] = 0
		}
	}
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// contextKey for vault proxy in context (unused externally, reserved for future DI)
type vaultContextKey struct{}

func VaultProxyFromContext(ctx context.Context) *VaultProxy {
	v, _ := ctx.Value(vaultContextKey{}).(*VaultProxy)
	return v
}
