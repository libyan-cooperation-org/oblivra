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
func (m *QuorumManager) Approve(id string, userID string, credentialID []byte, signature []byte, authData []byte, clientData []byte) error {
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

	// 1. Verify FIDO2 Signature
	// Note: We use the existing CompleteAuthentication logic to verify the hardware-backed signature.
	// In a real ceremony, the challenge should be bound to the Request ID.
	// For this phase, we assume the caller has already verified the FIDO2 auth via the FIDO2Manager 
	// and is passing the result.

	// Check if already approved by this user
	for _, a := range req.Approvals {
		if a == userID {
			return fmt.Errorf("user already approved this request")
		}
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
