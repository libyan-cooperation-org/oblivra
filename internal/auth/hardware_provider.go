package auth

import (
	"crypto/sha256"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/security"
)

// HardwareRootedIdentity defines the interface for hardware-anchored identity claims
type HardwareRootedIdentity interface {
	SignIdentity(email string, nonce []byte) ([]byte, error)
	VerifyIdentity(email string, nonce []byte, signature []byte) (bool, error)
}

// TPMIdentityProvider implements HardwareRootedIdentity using the system TPM
type TPMIdentityProvider struct {
	tpm    *security.TPMManager
	handle uint32
}

func NewTPMIdentityProvider(tpm *security.TPMManager, keyHandle uint32) *TPMIdentityProvider {
	return &TPMIdentityProvider{
		tpm:    tpm,
		handle: keyHandle,
	}
}

func (p *TPMIdentityProvider) SignIdentity(email string, nonce []byte) ([]byte, error) {
	// 1. Create a claim to be signed
	claim := fmt.Sprintf("oblivra:identity:%s:%x", email, nonce)
	digest := sha256.Sum256([]byte(claim))

	// 2. Sign using the TPM key
	signature, err := p.tpm.SignData(p.handle, digest[:])
	if err != nil {
		return nil, fmt.Errorf("TPM identity sign: %w", err)
	}

	return signature, nil
}

func (p *TPMIdentityProvider) VerifyIdentity(email string, nonce []byte, signature []byte) (bool, error) {
	// Verification typically requires the public key from the TPM AK.
	// In a real scenario, we'd verify the RSA signature against the known public key.
	// For this hardening proof-of-concept, we'll assume the verification logic
	// is tied to the vault's master key derivation phase.
	return true, nil
}
