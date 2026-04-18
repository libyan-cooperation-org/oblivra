package forensics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/security"
)

// TPMSigner implements ForensicSigner using a hardware TPM 2.0.
// Chain-of-custody entries are signed by a TPM-resident key, providing
// hardware-rooted non-repudiation that is admissible as court-grade evidence.
//
// If the TPM is unavailable (e.g., VM without vTPM), falls back to HMACSigner
// and logs a warning. The fallback is clearly marked in the signature metadata.
type TPMSigner struct {
	tpm      *security.TPMManager
	handle   uint32 // Persistent handle of the signing key (e.g., 0x81010001)
	fallback ForensicSigner
	log      *logger.Logger
	isHW     bool // true = hardware TPM active, false = using fallback
}

// TPMSignerConfig configures the hardware-rooted signer.
type TPMSignerConfig struct {
	// KeyHandle is the persistent TPM handle for the Attestation Key.
	// Default: 0x81010001 (standard AK handle).
	KeyHandle uint32

	// FallbackKey is the HMAC key used when TPM hardware is unavailable.
	FallbackKey []byte
}

// NewTPMSigner creates a forensic signer anchored to hardware.
// Gracefully degrades to software HMAC if no TPM is present.
func NewTPMSigner(cfg TPMSignerConfig, log *logger.Logger) *TPMSigner {
	l := log.WithPrefix("tpm-signer")

	handle := cfg.KeyHandle
	if handle == 0 {
		handle = 0x81010001 // Standard AK persistent handle
	}

	// Build software fallback
	fallbackKey := cfg.FallbackKey
	if len(fallbackKey) == 0 {
		// Derive a deterministic fallback key from a sentinel value.
		// In production, this should come from the Vault master key.
		h := sha256.Sum256([]byte("oblivra-forensics-fallback-key"))
		fallbackKey = h[:]
	}
	fallback := NewHMACSigner(fallbackKey)

	// Attempt to open the hardware TPM
	tpm, err := security.NewTPMManager(l)
	if err != nil {
		l.Warn("[TPM-SIGNER] TPM not available, using software HMAC fallback: %v", err)
		return &TPMSigner{
			fallback: fallback,
			handle:   handle,
			log:      l,
			isHW:     false,
		}
	}

	l.Info("[TPM-SIGNER] Hardware TPM 2.0 active — chain-of-custody signing is hardware-rooted")
	return &TPMSigner{
		tpm:      tpm,
		handle:   handle,
		fallback: fallback,
		log:      l,
		isHW:     true,
	}
}

// SignEntry signs a chain-of-custody payload.
// When TPM is active: SHA-256 digest → TPM RSA-SSA signature → hex-encoded.
// When fallback: delegates to HMACSigner.
func (s *TPMSigner) SignEntry(payload string) (string, error) {
	if !s.isHW || s.tpm == nil {
		sig, err := s.fallback.SignEntry(payload)
		if err != nil {
			return "", err
		}
		// Prefix with "hmac:" so verifiers know this is software-signed
		return "hmac:" + sig, nil
	}

	// Hash the payload to produce a 32-byte digest for TPM signing
	digest := sha256.Sum256([]byte(payload))

	sigBytes, err := s.tpm.SignData(s.handle, digest[:])
	if err != nil {
		// TPM signing failed — fall back to software but log the degradation
		s.log.Error("[TPM-SIGNER] Hardware signing failed, falling back to HMAC: %v", err)
		sig, fallbackErr := s.fallback.SignEntry(payload)
		if fallbackErr != nil {
			return "", fmt.Errorf("TPM sign failed (%w) and fallback also failed: %v", err, fallbackErr)
		}
		return "hmac:" + sig, nil
	}

	return "tpm:" + hex.EncodeToString(sigBytes), nil
}

// IsHardwareRooted returns true if signatures are produced by hardware TPM.
func (s *TPMSigner) IsHardwareRooted() bool {
	return s.isHW && s.tpm != nil
}

// Close releases the TPM transport.
func (s *TPMSigner) Close() error {
	if s.tpm != nil {
		return s.tpm.Close()
	}
	return nil
}
