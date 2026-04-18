package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/policy"
)

// PolicyService exposes the Feature Governance Engine to the frontend
type PolicyService struct {
	engine   *policy.Engine
	mac      *policy.MACEngine
	approval *policy.ApprovalManager
	ctx      context.Context
}

// NewPolicyService creates a new PolicyService
func NewPolicyService(engine *policy.Engine, mac *policy.MACEngine, approval *policy.ApprovalManager) *PolicyService {
	return &PolicyService{
		engine:   engine,
		mac:      mac,
		approval: approval,
	}
}

// Start is called at application startup
func (s *PolicyService) Start(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

// Stop is called at application termination
func (s *PolicyService) Stop(ctx context.Context) error {
	return nil
}

// Name returns the service name
func (s *PolicyService) Name() string {
	return "policy-service"
}

// Dependencies returns service dependencies
func (s *PolicyService) Dependencies() []string {
	return []string{"vault"}
}

// GetActiveTier returns the current operation tier (e.g., free, pro, enterprise)
func (s *PolicyService) GetActiveTier() string {
	return s.engine.GetActiveTier()
}

// CheckFeature evaluates whether the given feature identifier is accessible
func (s *PolicyService) CheckFeature(feature string) bool {
	return s.engine.Evaluate(feature)
}

// GetCapabilitiesMatrix returns all features and their evaluation status
func (s *PolicyService) GetCapabilitiesMatrix() map[string]bool {
	return s.engine.GetCapabilitiesMatrix()
}

// SetTier allows the frontend to simulate tier upgrades for demonstration purposes
func (s *PolicyService) SetTier(tier string) error {
	return s.engine.SetTier(tier)
}

// EvaluateMAC exposed for frontend UI components to proactively hide buttons if clearance is insufficient
func (s *PolicyService) EvaluateMAC(subjectClearance int, requiredClearance int, resource string) bool {
	return s.mac.Evaluate(auth.Clearance(subjectClearance), auth.Clearance(requiredClearance), resource)
}

// RequestApproval initiates an Approval Chain for a destructive action.
func (s *PolicyService) RequestApproval(requesterID, actionType, targetResource string) (*policy.ApprovalRequest, error) {
	return s.approval.RequestApproval(requesterID, actionType, targetResource)
}

// GrantApproval authorizes a pending request (requires admin simulation for the demo frontend)
func (s *PolicyService) GrantApproval(reqID string, approverID string, approverRole string) error {
	approver := &auth.UserAccount{
		ID:       approverID,
		Username: approverID,
		Role:     auth.Role(approverRole),
	}
	return s.approval.GrantApproval(reqID, approver)
}

// DenyApproval rejects an approval request
func (s *PolicyService) DenyApproval(reqID string, denierID string, denierRole string) error {
	denier := &auth.UserAccount{
		ID:       denierID,
		Username: denierID,
		Role:     auth.Role(denierRole),
	}
	return s.approval.DenyApproval(reqID, denier)
}

// GetPendingApprovals gets all waiting requests
func (s *PolicyService) GetPendingApprovals() []*policy.ApprovalRequest {
	return s.approval.GetPendingRequests()
}
