package mcp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// ExecutionEngine defines the interface for executing tools
type ExecutionEngine interface {
	Execute(ctx context.Context, tool string, params map[string]any) (any, error)
	Simulate(ctx context.Context, tool string, params map[string]any) (any, error)
}

// Handler implements the MCP execution pipeline
type Handler struct {
	registry  *ToolRegistry
	engine    ExecutionEngine
	integrity StateProvider
	log       *logger.Logger
	hmacKey   []byte // Secret used to sign approval tokens
}

// StateProvider defines the interface for state hashing
type StateProvider interface {
	StateHash() string
}

// NewHandler creates a new MCP handler
func NewHandler(registry *ToolRegistry, engine ExecutionEngine, integrity StateProvider, log *logger.Logger) *Handler {
	key := make([]byte, 32)
	rand.Read(key) // Generate a secure ephemeral key for token signing 

	return &Handler{
		registry:  registry,
		engine:    engine,
		integrity: integrity,
		log:       log.WithPrefix("mcp"),
		hmacKey:   key,
	}
}

// HandleRequest processes an incoming MCP request
func (h *Handler) HandleRequest(ctx context.Context, req MCPRequest) MCPResponse {
	startedAt := time.Now()
	
	resp := MCPResponse{
		RequestID: req.RequestID,
		Status:    "error", // default
		Execution: ExecutionInfo{
			StartedAt: startedAt,
		},
	}

	// 1. Validation
	if req.Version != "1.0" {
		resp.Error = &MCPError{Code: "INVALID_VERSION", Message: "MCP version 1.0 required"}
		return h.finalize(resp, req)
	}

	// 2. Auth & Tenant Isolation
	user := auth.UserFromContext(ctx)
	if user == nil {
		resp.Status = "denied"
		resp.Error = &MCPError{Code: "UNAUTHORIZED", Message: "Authentication required"}
		return h.finalize(resp, req)
	}

	if user.TenantID != "GLOBAL" && user.TenantID != req.TenantID {
		resp.Status = "denied"
		resp.Error = &MCPError{Code: "TENANT_ISOLATION_VIOLATION", Message: "Cross-tenant access denied"}
		return h.finalize(resp, req)
	}

	// 3. Tool Discovery
	toolDef, ok := h.registry.GetTool(req.Tool)
	if !ok {
		resp.Error = &MCPError{Code: "TOOL_NOT_FOUND", Message: fmt.Sprintf("Tool '%s' not registered", req.Tool)}
		return h.finalize(resp, req)
	}

	// 4. RBAC Check
	requiredPerm := auth.PermMCPExecute
	if req.Mode == "simulate" {
		requiredPerm = auth.PermMCPSimulate
	}

	// We should ideally use the RBACEngine here, but h.HandleRequest doesn't have it.
	// We'll rely on the middleware having set the user, and we'll check permissions manually
	// or assume the middleware already did some basic checks.
	// For production, we should pass RBACEngine to NewHandler.
	
	hasPerm := false
	for _, p := range user.Permissions {
		if p == "*" || p == requiredPerm {
			hasPerm = true
			break
		}
	}
	if !hasPerm {
		resp.Status = "denied"
		resp.Error = &MCPError{Code: "INSUFFICIENT_PRIVILEGES", Message: fmt.Sprintf("Missing permission: %s", requiredPerm)}
		return h.finalize(resp, req)
	}

	// For critical tools, require specific permissions if defined or higher role
	if toolDef.RiskLevel == "critical" || toolDef.RiskLevel == "high" {
		if user.RoleName != string(auth.RoleAdmin) {
			resp.Status = "denied"
			resp.Error = &MCPError{Code: "INSUFFICIENT_PRIVILEGES", Message: fmt.Sprintf("Tool '%s' requires admin privileges", req.Tool)}
			return h.finalize(resp, req)
		}
	}

	// 5. Cost Validation
	if req.Constraints.CostLimit > 0 && toolDef.Constraints.MaxCost > req.Constraints.CostLimit {
		resp.Status = "denied"
		resp.Error = &MCPError{Code: "COST_LIMIT_EXCEEDED", Message: fmt.Sprintf("Tool cost exceeds requested limit (%d > %d)", toolDef.Constraints.MaxCost, req.Constraints.CostLimit)}
		return h.finalize(resp, req)
	}

	// 6. Approval Gate
	if toolDef.RequiresApproval && (req.Approval.Token == "" || !h.validateApproval(req.Approval.Token, user.ID)) {
		resp.Status = "pending_approval"
		resp.Data = ApprovalChallenge{
			ApprovalID: uuid.New().String(),
			Required:   1, // M-of-N: 1 for now
			ExpiresAt:  time.Now().Add(1 * time.Hour),
		}
		return h.finalize(resp, req)
	}

	// 7. Execution
	var err error
	var result any
	if req.Mode == "simulate" {
		result, err = h.engine.Simulate(ctx, req.Tool, req.Params)
	} else {
		result, err = h.engine.Execute(ctx, req.Tool, req.Params)
	}

	if err != nil {
		resp.Status = "error"
		resp.Error = &MCPError{Code: "EXECUTION_FAILED", Message: err.Error()}
		return h.finalize(resp, req)
	}

	resp.Status = "success"
	resp.Data = result
	resp.Meta = ResponseMeta{
		Cost: toolDef.Constraints.MaxCost, // simplify for MVP
	}

	return h.finalize(resp, req)
}

// GenerateApprovalToken creates a cryptographically verifiable token.
func (h *Handler) GenerateApprovalToken(approvalID, actorID string) string {
	mac := hmac.New(sha256.New, h.hmacKey)
	mac.Write([]byte(approvalID + ":" + actorID))
	signature := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("approved:%s:%s:%s", approvalID, actorID, signature)
}

func (h *Handler) validateApproval(token string, userID string) bool {
	parts := strings.Split(token, ":")
	if len(parts) != 4 || parts[0] != "approved" || parts[2] != userID {
		return false
	}
	approvalID := parts[1]
	expected := h.GenerateApprovalToken(approvalID, userID)
	return hmac.Equal([]byte(token), []byte(expected))
}

func (h *Handler) finalize(resp MCPResponse, req MCPRequest) MCPResponse {
	resp.Execution.CompletedAt = time.Now()
	resp.Execution.DurationMS = time.Since(resp.Execution.StartedAt).Milliseconds()
	
	// Generate Audit Hashes
	resp.Audit = h.generateAuditInfo(resp, req)
	
	return resp
}

func (h *Handler) generateAuditInfo(resp MCPResponse, req MCPRequest) AuditInfo {
	paramData, _ := json.Marshal(req.Params)
	inputHash := sha256.Sum256(paramData)
	
	outputData, _ := json.Marshal(resp.Data)
	outputHash := sha256.Sum256(outputData)
	
	// Deterministic State Chaining: Input + Global State + Output
	stateHash := h.integrity.StateHash()
	combined := fmt.Sprintf("%x:%s:%x", inputHash, stateHash, outputHash)
	execHash := sha256.Sum256([]byte(combined))

	return AuditInfo{
		EventHash:        hex.EncodeToString(outputHash[:]),
		RuleVersion:      "v1.0",
		PipelineVersion:  "mcp-v1",
		InputHash:        hex.EncodeToString(inputHash[:]),
		ExecutionHash:    hex.EncodeToString(execHash[:]),
		OutputHash:       hex.EncodeToString(outputHash[:]),
	}
}
