package plugin

// signing.go — Ed25519 plugin signature verification
//
// Every plugin .wasm binary and manifest must be signed by a trusted key.
// At load time, LoadAndVerify checks the signature before any code runs.
// This closes the audit gap: "any plugin = full logic access, no trust model".
//
// Signing workflow (for plugin authors):
//
//   1. Generate a keypair:
//        openssl genpkey -algorithm Ed25519 -out priv.pem
//        openssl pkey -in priv.pem -pubout -out pub.pem
//
//   2. Sign the WASM binary:
//        openssl pkeyutl -sign -inkey priv.pem -in plugin.wasm -out plugin.wasm.sig
//
//   3. Distribute plugin.wasm + plugin.wasm.sig + manifest.json
//
//   4. Register the trusted public key via AddTrustedKey().

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"sync"
)

// ErrNoSignature is returned when a plugin has no .sig file.
var ErrNoSignature = errors.New("plugin signature file not found")

// ErrSignatureMismatch is returned when the signature does not verify.
var ErrSignatureMismatch = errors.New("plugin signature verification FAILED — binary may be tampered")

// ErrNoTrustedKey is returned when there are no trusted keys registered.
var ErrNoTrustedKey = errors.New("no trusted keys registered — refusing to load unsigned plugin")

// trustStore holds the set of trusted Ed25519 public keys.
// Thread-safe for concurrent plugin loads.
type trustStore struct {
	mu   sync.RWMutex
	keys []ed25519.PublicKey
}

var globalTrustStore = &trustStore{}

// AddTrustedKey registers a raw Ed25519 public key (32 bytes) as trusted.
// Called during application startup, typically from a config-loaded key list.
func AddTrustedKey(pubKey ed25519.PublicKey) {
	globalTrustStore.mu.Lock()
	defer globalTrustStore.mu.Unlock()
	globalTrustStore.keys = append(globalTrustStore.keys, pubKey)
}

// AddTrustedKeyFromPEM parses a PEM-encoded Ed25519 public key and registers it.
func AddTrustedKeyFromPEM(pemBytes []byte) error {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return errors.New("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	edKey, ok := pub.(ed25519.PublicKey)
	if !ok {
		return errors.New("key is not Ed25519")
	}

	AddTrustedKey(edKey)
	return nil
}

// AddTrustedKeyFromHex parses a 64-char hex string (32-byte Ed25519 public key) and registers it.
// Convenient for config-file key lists.
func AddTrustedKeyFromHex(hexKey string) error {
	raw, err := hex.DecodeString(hexKey)
	if err != nil {
		return fmt.Errorf("decode hex key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid Ed25519 key length: got %d, want %d", len(raw), ed25519.PublicKeySize)
	}
	AddTrustedKey(ed25519.PublicKey(raw))
	return nil
}

// VerifyPlugin verifies the Ed25519 signature of a WASM plugin binary.
//
//   wasmPath: path to the .wasm file
//   sigPath:  path to the .wasm.sig file (detached signature)
//
// Returns nil if verification passes against at least one trusted key.
// Returns ErrNoTrustedKey, ErrNoSignature, or ErrSignatureMismatch otherwise.
func VerifyPlugin(wasmPath, sigPath string) error {
	globalTrustStore.mu.RLock()
	keys := make([]ed25519.PublicKey, len(globalTrustStore.keys))
	copy(keys, globalTrustStore.keys)
	globalTrustStore.mu.RUnlock()

	if len(keys) == 0 {
		return ErrNoTrustedKey
	}

	// Read the plugin binary
	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("read plugin binary: %w", err)
	}

	// Read the detached signature
	sigBytes, err := os.ReadFile(sigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNoSignature
		}
		return fmt.Errorf("read signature file: %w", err)
	}

	// Try each trusted key
	for _, pub := range keys {
		if ed25519.Verify(pub, wasmBytes, sigBytes) {
			return nil // verified
		}
	}

	return ErrSignatureMismatch
}

// VerifyManifestPlugin is a convenience wrapper that derives sigPath from
// manifest.Main by appending ".sig", then calls VerifyPlugin.
func VerifyManifestPlugin(m *Manifest) error {
	sigPath := m.Main + ".sig"
	return VerifyPlugin(m.Main, sigPath)
}

// IsTrustEnforced returns true if at least one trusted key has been registered,
// meaning signature enforcement is active.
func IsTrustEnforced() bool {
	globalTrustStore.mu.RLock()
	defer globalTrustStore.mu.RUnlock()
	return len(globalTrustStore.keys) > 0
}
