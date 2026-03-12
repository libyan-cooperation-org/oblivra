package policy

import (
	"errors"
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

// ApprovalRequest represents a destructive action held in staging until authorized
type ApprovalRequest struct {
	ID             string
	RequesterID    string
	ActionType     string
	TargetResource string
	Status         ApprovalStatus
	CreatedAt      string
	ApprovedBy     string
	ApprovedAt     string
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

// RequestApproval stages a destructive action for review
func (m *ApprovalManager) RequestApproval(requesterID, actionType, targetResource string) (*ApprovalRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	req := &ApprovalRequest{
		ID:             uuid.New().String(),
		RequesterID:    requesterID,
		ActionType:     actionType,
		TargetResource: targetResource,
		Status:         StatusPending,
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	m.requests[req.ID] = req
	m.log.Info("[POLICY] Approval requested: %s for %s by %s (ReqID: %s)", actionType, targetResource, requesterID, req.ID)

	m.bus.Publish(eventbus.EventPolicyApprovalRequested, req)
	return req, nil
}

// GrantApproval authorizes a pending request if the approver has the proper role
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

	// For Phase 7.5, we enforce that Admins or users with higher clearances than the requester can approve.
	// We'll enforce that the approver has Admin role to keep the workflow strict.
	if approver.Role != auth.RoleAdmin {
		m.log.Warn("[POLICY] Unauthorized approval attempt for %s by %s", reqID, approver.ID)
		return errors.New("insufficient privileges to grant approval")
	}

	if req.RequesterID == approver.ID {
		m.log.Warn("[POLICY] Self-approval blocked for %s by %s", reqID, approver.ID)
		return errors.New("cannot approve your own request")
	}

	req.Status = StatusApproved
	req.ApprovedBy = approver.ID
	req.ApprovedAt = time.Now().Format(time.RFC3339)

	m.log.Info("[POLICY] Approval GRANTED: %s by %s", reqID, approver.ID)
	m.bus.Publish(eventbus.EventPolicyApprovalGranted, req)
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
