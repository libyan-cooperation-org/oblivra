package mcp

import (
	"time"
)

// MCPRequest defines the strict protocol envelope for all MCP traffic
type MCPRequest struct {
	Version     string         `json:"version"`
	RequestID   string         `json:"request_id"`
	Timestamp   time.Time      `json:"timestamp"`
	TenantID    string         `json:"tenant_id"`
	Actor       Actor          `json:"actor"`
	Context     RequestContext `json:"context"`
	Tool        string         `json:"tool"`
	Mode        string         `json:"mode"` // execute|simulate|dry_run
	Params      map[string]any `json:"params"`
	Constraints Constraints    `json:"constraints"`
	Approval    ApprovalRequest `json:"approval"`
}

// Actor represents the entity requesting the action
type Actor struct {
	Type  string   `json:"type"` // user|api_key|agent|ai
	ID    string   `json:"id"`
	Roles []string `json:"roles"`
}

// RequestContext provides forensic metadata about the request source
type RequestContext struct {
	SessionID string `json:"session_id"`
	Source    string `json:"source"` // ui|cli|api|ai
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
}

// Constraints define execution limits and cost controls
type Constraints struct {
	TimeoutMS    int    `json:"timeout_ms"`
	MaxRows      int    `json:"max_rows"`
	CostLimit    int    `json:"cost_limit"`
	MaxTimeRange string `json:"max_time_range,omitempty"`
	Priority     string `json:"priority,omitempty"` // normal|high
}

// ApprovalRequest contains optional approval tokens
type ApprovalRequest struct {
	Required bool   `json:"required"`
	Token    string `json:"token,omitempty"`
}

// MCPResponse defines the standard response schema
type MCPResponse struct {
	RequestID string         `json:"request_id"`
	Status    string         `json:"status"` // success|error|denied|pending_approval
	Execution ExecutionInfo  `json:"execution"`
	Data      any            `json:"data,omitempty"`
	Meta      ResponseMeta   `json:"meta,omitempty"`
	Audit     AuditInfo      `json:"audit"`
	Error     *MCPError      `json:"error,omitempty"`
}

// ExecutionInfo tracks the timing of the execution
type ExecutionInfo struct {
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	DurationMS  int64     `json:"duration_ms"`
}

// ResponseMeta provides technical metadata about the result
type ResponseMeta struct {
	Rows     int  `json:"rows"`
	Cost     int  `json:"cost"`
	Degraded bool `json:"degraded"`
}

// AuditInfo provides hash-based integrity verification
type AuditInfo struct {
	EventHash        string `json:"event_hash"`
	RuleVersion      string `json:"rule_version"`
	PipelineVersion  string `json:"pipeline_version"`
	InputHash        string `json:"input_hash"`
	ExecutionHash    string `json:"execution_hash"`
	OutputHash       string `json:"output_hash"`
}

// MCPError defines the structured error format
type MCPError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ToolDefinition defines a tool in the registry
type ToolDefinition struct {
	Name             string         `json:"name"`
	Version          string         `json:"version"`
	Description      string         `json:"description"`
	Category         string         `json:"category"`
	RiskLevel        string         `json:"risk_level"` // low|medium|high|critical
	RequiresApproval bool           `json:"requires_approval"`
	Idempotent       bool           `json:"idempotent"`
	InputSchema      map[string]any `json:"input_schema"`
	OutputSchema     map[string]any `json:"output_schema"`
	Constraints      ToolConstraints `json:"constraints"`
}

// ToolConstraints define tool-specific limits
type ToolConstraints struct {
	MaxCost   int    `json:"max_cost"`
	RateLimit string `json:"rate_limit"`
}

// ApprovalChallenge is returned when an action requires approval
type ApprovalChallenge struct {
	ApprovalID string    `json:"approval_id"`
	Required   int       `json:"required"`
	Received   int       `json:"received"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// ApprovalSubmission is used to submit an approval token
type ApprovalSubmission struct {
	ApprovalToken string `json:"approval_token"`
	ApproverID    string `json:"approver_id"`
}
