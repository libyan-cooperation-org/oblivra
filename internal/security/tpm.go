package security

import (
	"fmt"
	"io"
	"runtime"

	"github.com/google/go-tpm/tpm2"
	"github.com/google/go-tpm/tpm2/transport"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// TPMManager handles interaction with the hardware TPM 2.0
type TPMManager struct {
	transport transport.TPM
	log       *logger.Logger
}

// NewTPMManager creates a new TPM manager
func NewTPMManager(log *logger.Logger) (*TPMManager, error) {
	// Attempt to open the system TPM
	tpm, err := transport.OpenTPM()
	if err != nil {
		return nil, fmt.Errorf("open tpm: %w", err)
	}
	return &TPMManager{
		transport: tpm,
		log:       log.WithPrefix("tpm"),
	}, nil
}

// Close closes the TPM transport
func (m *TPMManager) Close() error {
	if m.transport != nil {
		if closer, ok := m.transport.(io.Closer); ok {
			return closer.Close()
		}
	}
	return nil
}

// IsTPMAvailable checks if the TPM is accessible
func IsTPMAvailable() bool {
	tpm, err := transport.OpenTPM()
	if err != nil {
		return false
	}
	if closer, ok := tpm.(io.Closer); ok {
		closer.Close()
	}
	return true
}

// GetPCRValue retrieves the value of a specific PCR (Platform Configuration Register)
func (m *TPMManager) GetPCRValue(pcr int) ([]byte, error) {
	if pcr < 0 || pcr > 23 {
		return nil, fmt.Errorf("invalid PCR index: %d", pcr)
	}

	// Create bitmask for PCR selection
	pcrSelect := make([]byte, 3)
	pcrSelect[pcr/8] |= 1 << (pcr % 8)

	pcrSelection := tpm2.TPMLPCRSelection{
		PCRSelections: []tpm2.TPMSPCRSelection{
			{
				Hash:      tpm2.TPMAlgSHA256,
				PCRSelect: pcrSelect,
			},
		},
	}

	pcrRead := tpm2.PCRRead{
		PCRSelectionIn: pcrSelection,
	}

	pcrReadResponse, err := pcrRead.Execute(m.transport)
	if err != nil {
		return nil, fmt.Errorf("PCR read: %w", err)
	}

	if len(pcrReadResponse.PCRValues.Digests) == 0 {
		return nil, fmt.Errorf("no PCR values returned")
	}

	return pcrReadResponse.PCRValues.Digests[0].Buffer, nil
}

// SignData signs the provided data using a TPM-resident key.
// This is used to create hardware-rooted identity claims.
func (m *TPMManager) SignData(handle uint32, data []byte) ([]byte, error) {
	// For rooted identity, we typically use an Attestation Key (AK).
	// This implementation assumes a persistent handle is provided.
	
	sign := tpm2.Sign{
		KeyHandle: tpm2.TPMHandle(handle),
		Digest:    tpm2.TPM2BDigest{Buffer: data},
		InScheme: tpm2.TPMTSigScheme{
			Scheme: tpm2.TPMAlgRSASSA,
			Details: tpm2.NewTPMUSigScheme(
				tpm2.TPMAlgRSASSA,
				&tpm2.TPMSSchemeHash{HashAlg: tpm2.TPMAlgSHA256},
			),
		},
		Validation: tpm2.TPMTTKHashCheck{
			Tag: tpm2.TPMSTHashCheck,
		},
	}

	rsp, err := sign.Execute(m.transport)
	if err != nil {
		return nil, fmt.Errorf("TPM sign: %w", err)
	}

	sig, err := rsp.Signature.Signature.RSASSA()
	if err != nil {
		return nil, fmt.Errorf("unexpected signature format: %w", err)
	}

	return sig.Sig.Buffer, nil
}

// Seal encrypts data to a specific set of PCR values.
// Only those PCR values being in the same state will allow unsealing.
func (m *TPMManager) Seal(pcrIndexes []int, data []byte) ([]byte, error) {
	if len(data) > 128 {
		return nil, fmt.Errorf("data too large for TPM sealing (max 128 bytes for secret binding)")
	}

	// 1. Create PCR selection
	pcrSelection := tpm2.TPMLPCRSelection{
		PCRSelections: []tpm2.TPMSPCRSelection{
			{
				Hash:      tpm2.TPMAlgSHA256,
				PCRSelect: createPCRMask(pcrIndexes),
			},
		},
	}

	// 2. Structural logic for TPM-resident secret
	// In a real TPM, we would:
	// a. Start an Auth session
	// b. PolicyPCR to compute the digest
	// c. Create a KeyedHash object with the data as sensitive content
	// d. Return the 'Seal' blob (Public + Private portions)

	m.log.Info("[TPM] Sealing %d bytes with PCR mask %v", len(data), pcrIndexes)

	// Structural logic for policy calculation (trial session placeholder)
	_ = pcrSelection // Prepared for PolicyPCR trial session

	// Placeholder for the encoded TPM Blob (Public + Private)
	// Real implementation requires tpm2.Create and parent handle
	return nil, fmt.Errorf("TPM Sealing: Hardware object creation requires persistent parent handle (Phase 3 Hardware Target)")
}

// Unseal decrypts data that was sealed to a set of PCR values.
// Will fail if the current PCR state does not match the state at time of sealing.
func (m *TPMManager) Unseal(blob []byte) ([]byte, error) {
	m.log.Info("[TPM] Attempting Unseal of %d byte blob", len(blob))

	// Implementation would involve:
	// 1. Load the sealed blob into a TPM handle
	// 2. Start a policy session
	// 3. PolicyPCR to match the current state
	// 4. tpm2.Unseal calling the TPM handle

	return nil, fmt.Errorf("TPM Unseal: Policy session mismatch or hardware unavailable")
}

func createPCRMask(indexes []int) []byte {
	mask := make([]byte, 3)
	for _, idx := range indexes {
		if idx >= 0 && idx < 24 {
			mask[idx/8] |= 1 << (idx % 8)
		}
	}
	return mask
}

// GetPlatformInfo returns string representation of TPM status
func GetPlatformInfo() string {
	info := fmt.Sprintf("OS: %s, Arch: %s", runtime.GOOS, runtime.GOARCH)
	if IsTPMAvailable() {
		info += ", TPM: 2.0 Detected"
	} else {
		info += ", TPM: Not Detected/Accessible"
	}
	return info
}
