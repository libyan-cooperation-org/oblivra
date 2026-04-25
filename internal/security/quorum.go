package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

type QuorumAction string

const (
	ActionKillSwitch    QuorumAction = "INGEST_KILL_SWITCH"
	ActionSOARPlaybook  QuorumAction = "SOAR_DESTRUCTIVE_EXEC"
	ActionVaultRotation QuorumAction = "VAULT_ROOT_ROTATION"
	ActionCryptoWipe    QuorumAction = "GDPR_CRYPTO_WIPE"
)

type ApprovalStatus string

const (
	StatusPending  ApprovalStatus = "pending"
	StatusApproved ApprovalStatus = "approved"
	StatusRejected ApprovalStatus = "rejected"
	StatusExpired  ApprovalStatus = "expired"
)

type QuorumRequest struct {
	ID          string         `json:"id"`
	Action      QuorumAction   `json:"action"`
	Description string         `json:"description"`
	Proposer    string         `json:"proposer"`
	CreatedAt   time.Time      `json:"created_at"`
	ExpiresAt   time.Time      `json:"expires_at"`
	Approvals   []string       `json:"approvals"` // User IDs who approved
	Required    int            `json:"required"`
	Status      ApprovalStatus `json:"status"`
	Payload     string         `json:"payload"` // Action-specific data
}

// QuorumManager handles M-of-N multi-party authorization ceremonies.
type QuorumManager struct {
	mu       sync.RWMutex
	requests map[string]*QuorumRequest
	log      *logger.Logger
	fido     *FIDO2Manager
}

func NewQuorumManager(fido *FIDO2Manager, log *logger.Logger) *QuorumManager {
	return &QuorumManager{
		requests: make(map[string]*QuorumRequest),
		log:      log.WithPrefix("security:quorum"),
		fido:     fido,
	}
}

// Propose starts a new M-of-N approval ceremony.
func (m *QuorumManager) Propose(action QuorumAction, description string, proposer string, required int, payload string) (*QuorumRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Generate unique Request ID
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s-%s-%d", action, proposer, time.Now().UnixNano())))
	id := hex.EncodeToString(h.Sum(nil))[:12]

	req := &QuorumRequest{
		ID:          id,
		Action:      action,
		Description: description,
		Proposer:    proposer,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(1 * time.Hour), // Quorum must be met within 1 hour
		Approvals:   []string{},
		Required:    required,
		Status:      StatusPending,
		Payload:     payload,
	}

	m.requests[id] = req
	m.log.Info("[QUORUM] Action Proposed: %s (ID: %s, Required: %d, Proposer: %s)", action, id, required, proposer)
	
	return req, nil
}

// Approve records a signature-backed approval for a request.
//
// challengeID is a FIDO2 challenge issued via FIDO2Manager.StartAuthentication
// before the user signed. credentialID/signature/authenticatorData/clientDataJSON
// are the WebAuthn assertion outputs from the user's hardware token.
//
// Phase 26.5 hardening: this function now drives FIDO2Manager.CompleteAuthentication
// to verify the ECDSA signature against the registered public key BEFORE
// counting the approval. Previously the approval was counted on trust ("we
// assume the caller has already verified") which made the M-of-N quorum
// gate trivially forgeable by any caller able to write to the requests map.
//
// If no FIDO2Manager is wired (development / dev-vault mode), the function
// emits a clearly-marked WARN log and falls back to count-only behaviour.
// Production deployments must NOT run without a FIDO2Manager.
func (m *QuorumManager) Approve(id string, userID string, challengeID string, credentialID []byte, signature []byte, authData []byte, clientData []byte, newSignCount uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, ok := m.requests[id]
	if !ok {
		return fmt.Errorf("quorum request %s not found", id)
	}

	if req.Status != StatusPending {
		return fmt.Errorf("request is no longer pending (status: %s)", req.Status)
	}

	if time.Now().After(req.ExpiresAt) {
		req.Status = StatusExpired
		return fmt.Errorf("request has expired")
	}

	// Check if already approved by this user — done BEFORE FIDO2 verify so
	// repeated-approval errors are cheap (no hardware round-trip).
	for _, a := range req.Approvals {
		if a == userID {
			return fmt.Errorf("user already approved this request")
		}
	}

	// Hardware signature verification.
	if m.fido != nil {
		if err := m.fido.CompleteAuthentication(challengeID, credentialID, signature, authData, clientData, newSignCount); err != nil {
			m.log.Warn("[QUORUM] FIDO2 verification REJECTED for user=%s req=%s: %v", userID, id, err)
			return fmt.Errorf("FIDO2 verification failed: %w", err)
		}
	} else {
		// Critical: development fallback. Production must wire a FIDO2Manager
		// in container.go; absence of one means *any* caller passing through
		// this code path can count an approval without a hardware token.
		m.log.Warn("[QUORUM] APPROVAL COUNTED WITHOUT HARDWARE VERIFICATION — FIDO2Manager is nil (development mode only). user=%s req=%s", userID, id)
	}

	req.Approvals = append(req.Approvals, userID)
	m.log.Info("[QUORUM] Action Approved by %s (ID: %s, Progress: %d/%d)", userID, id, len(req.Approvals), req.Required)

	// Check if quorum met
	if len(req.Approvals) >= req.Required {
		req.Status = StatusApproved
		m.log.Info("[QUORUM] Action AUTHORIZED (ID: %s)", id)
	}

	return nil
}

// GetRequest retrieves a request by ID.
func (m *QuorumManager) GetRequest(id string) (*QuorumRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, ok := m.requests[id]
	if !ok {
		return nil, fmt.Errorf("request not found")
	}
	return req, nil
}

// ListPending returns all active quorum requests.
func (m *QuorumManager) ListPending() []*QuorumRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pending := []*QuorumRequest{}
	for _, r := range m.requests {
		if r.Status == StatusPending {
			pending = append(pending, r)
		}
	}
	return pending
}
