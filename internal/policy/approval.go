package policy

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type ApprovalStatus string

const (
	StatusPending  ApprovalStatus = "pending"
	StatusApproved ApprovalStatus = "approved"
	StatusDenied   ApprovalStatus = "denied"
)

// TwoManActions is the set of action types that require M-of-N (Two-Man Rule)
// authorization before execution. These are the highest-risk "War Mode" operations.
var TwoManActions = map[string]bool{
	"NUCLEAR_DESTRUCTION":    true,
	"NETWORK_ISOLATION_BULK": true,
	"REMOTE_FILE_DELETE":     true,
	"KILLSWITCH_ACTIVATE":    true,
	"VAULT_WIPE":             true,
}

// DefaultQuorum is the minimum number of distinct admin approvals required
// for Two-Man Rule actions. Override per-deployment via config.
const DefaultQuorum = 2

// ApprovalRequest represents a destructive action held in staging until authorized
type ApprovalRequest struct {
	ID             string
	RequesterID    string
	ActionType     string
	TargetResource string
	Status         ApprovalStatus
	CreatedAt      string
	ApprovedBy     string   // first approver (legacy compat)
	ApprovedAt     string
	// Quorum fields for Two-Man Rule
	RequiredApprovals int
	GrantedBy         []string // all approver IDs in order
}

// ApprovalManager orchestrates the multi-party authorization workflows
type ApprovalManager struct {
	mu       sync.RWMutex
	requests map[string]*ApprovalRequest
	bus      *eventbus.Bus
	log      *logger.Logger
}

func NewApprovalManager(bus *eventbus.Bus, log *logger.Logger) *ApprovalManager {
	return &ApprovalManager{
		requests: make(map[string]*ApprovalRequest),
		bus:      bus,
		log:      log,
	}
}

// RequestApproval stages a destructive action for review.
// Two-Man Rule actions are automatically assigned a quorum of DefaultQuorum.
func (m *ApprovalManager) RequestApproval(requesterID, actionType, targetResource string) (*ApprovalRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	required := 1
	if TwoManActions[actionType] {
		required = DefaultQuorum
	}

	req := &ApprovalRequest{
		ID:                uuid.New().String(),
		RequesterID:       requesterID,
		ActionType:        actionType,
		TargetResource:    targetResource,
		Status:            StatusPending,
		CreatedAt:         time.Now().Format(time.RFC3339),
		RequiredApprovals: required,
		GrantedBy:         []string{},
	}

	m.requests[req.ID] = req
	twoManNote := ""
	if required > 1 {
		twoManNote = fmt.Sprintf(" [TWO-MAN RULE: %d approvals required]", required)
	}
	m.log.Info("[POLICY] Approval requested: %s for %s by %s (ReqID: %s)%s", actionType, targetResource, requesterID, req.ID, twoManNote)

	m.bus.Publish(eventbus.EventPolicyApprovalRequested, req)
	return req, nil
}

// GrantApproval authorizes a pending request.
//
// For Two-Man Rule (quorum > 1) actions:
//   - Each call from a unique admin adds one vote.
//   - The request is only marked Approved once RequiredApprovals distinct
//     admins have voted. Until then it remains Pending.
//   - The requester can never approve their own request.
func (m *ApprovalManager) GrantApproval(reqID string, approver *auth.UserAccount) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, exists := m.requests[reqID]
	if !exists {
		return errors.New("approval request not found")
	}
	if req.Status != StatusPending {
		return errors.New("request is not pending")
	}

	// Only admins may grant approvals
	if approver.Role != auth.RoleAdmin {
		m.log.Warn("[POLICY] Unauthorized approval attempt for %s by %s", reqID, approver.ID)
		return errors.New("insufficient privileges to grant approval")
	}

	// Requesters cannot vote on their own request
	if req.RequesterID == approver.ID {
		m.log.Warn("[POLICY] Self-approval blocked for %s by %s", reqID, approver.ID)
		return errors.New("cannot approve your own request")
	}

	// Prevent double-voting by the same approver
	for _, id := range req.GrantedBy {
		if id == approver.ID {
			return fmt.Errorf("approver %s has already voted on request %s", approver.ID, reqID)
		}
	}

	// Record this vote
	req.GrantedBy = append(req.GrantedBy, approver.ID)
	req.ApprovedBy = approver.ID // keep compat field pointing to most recent approver
	req.ApprovedAt = time.Now().Format(time.RFC3339)

	grantedCount := len(req.GrantedBy)
	required := req.RequiredApprovals
	if required == 0 {
		required = 1
	}

	m.log.Info("[POLICY] Vote GRANTED: %s by %s (%d/%d)", reqID, approver.ID, grantedCount, required)

	if grantedCount >= required {
		// Quorum reached — approve the action
		req.Status = StatusApproved
		m.log.Info("[POLICY] Approval COMPLETE (quorum met): %s — approvers: %v", reqID, req.GrantedBy)
		m.bus.Publish(eventbus.EventPolicyApprovalGranted, req)
	} else {
		// Still waiting for more votes
		m.log.Info("[POLICY] Approval PENDING quorum: %s — %d/%d votes received", reqID, grantedCount, required)
		m.bus.Publish(eventbus.EventPolicyApprovalRequested, req) // re-publish so UI updates
	}
	return nil
}

// DenyApproval rejects a pending request
func (m *ApprovalManager) DenyApproval(reqID string, denier *auth.UserAccount) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, exists := m.requests[reqID]
	if !exists {
		return errors.New("approval request not found")
	}

	if req.Status != StatusPending {
		return errors.New("request is not pending")
	}

	// Anyone Admin can deny
	if denier.Role != auth.RoleAdmin && denier.ID != req.RequesterID {
		m.log.Warn("[POLICY] Unauthorized denial attempt for %s by %s", reqID, denier.ID)
		return errors.New("insufficient privileges to deny this request")
	}

	req.Status = StatusDenied
	req.ApprovedBy = denier.ID // Use ApprovedBy to track who closed it
	req.ApprovedAt = time.Now().Format(time.RFC3339)

	m.log.Info("[POLICY] Approval DENIED: %s by %s", reqID, denier.ID)
	m.bus.Publish(eventbus.EventPolicyApprovalDenied, req)
	return nil
}

func (m *ApprovalManager) GetPendingRequests() []*ApprovalRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var pending []*ApprovalRequest
	for _, req := range m.requests {
		if req.Status == StatusPending {
			pending = append(pending, req)
		}
	}
	return pending
}

// GetRequest returns a single request by ID.
func (m *ApprovalManager) GetRequest(reqID string) (*ApprovalRequest, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	req, ok := m.requests[reqID]
	return req, ok
}

// IsApproved returns true if the request has reached its required quorum.
func (m *ApprovalManager) IsApproved(reqID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	req, ok := m.requests[reqID]
	if !ok {
		return false
	}
	return req.Status == StatusApproved
}

// RequiresQuorum returns true if an action type is subject to the Two-Man Rule.
func RequiresQuorum(actionType string) bool {
	return TwoManActions[actionType]
}
