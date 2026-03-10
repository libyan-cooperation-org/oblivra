package policy

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// SignedBundle represents a cryptographically verified policy bundle for air-gapped nodes
type SignedBundle struct {
	Config    Config    `json:"config"`
	ExpiresAt time.Time `json:"expires_at"`
	Signature []byte    `json:"signature"`
}

// CreateBundle signs a configuration and produces the raw bytes of the `.obp` bundle
func CreateBundle(config Config, duration time.Duration, privKey ed25519.PrivateKey) ([]byte, error) {
	bundle := SignedBundle{
		Config:    config,
		ExpiresAt: time.Now().Add(duration),
	}

	// Payload is the JSON representation without the signature
	payloadBytes, err := json.Marshal(struct {
		Config    Config    `json:"config"`
		ExpiresAt time.Time `json:"expires_at"`
	}{
		Config:    bundle.Config,
		ExpiresAt: bundle.ExpiresAt,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload for signing: %w", err)
	}

	bundle.Signature = ed25519.Sign(privKey, payloadBytes)

	// Marshal the complete signed bundle
	return json.Marshal(bundle)
}

// LoadBundle reads a `.obp` file from disk, verifies its cryptographic signature,
// and ensures it hasn't expired before unlocking the configuration.
func LoadBundle(filePath string, pubKey ed25519.PublicKey) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy bundle: %w", err)
	}

	var bundle SignedBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse policy bundle: %w", err)
	}

	if time.Now().After(bundle.ExpiresAt) {
		return nil, errors.New("policy bundle has expired")
	}

	// Payload is the JSON representation without the signature
	payloadBytes, err := json.Marshal(struct {
		Config    Config    `json:"config"`
		ExpiresAt time.Time `json:"expires_at"`
	}{
		Config:    bundle.Config,
		ExpiresAt: bundle.ExpiresAt,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload for verification: %w", err)
	}

	if !ed25519.Verify(pubKey, payloadBytes, bundle.Signature) {
		return nil, errors.New("invalid cryptographic signature on policy bundle")
	}

	return &bundle.Config, nil
}
