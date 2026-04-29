package mcp

import (
	"context"
	"testing"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type mockEngine struct{}

func (m *mockEngine) Execute(ctx context.Context, tool string, params map[string]any) (any, error) {
	return map[string]any{"executed": tool}, nil
}

func (m *mockEngine) Simulate(ctx context.Context, tool string, params map[string]any) (any, error) {
	return map[string]any{"simulated": tool}, nil
}

type mockIntegrity struct{}

func (m *mockIntegrity) StateHash() string {
	return "mock-state-hash"
}

func TestMCPHandler(t *testing.T) {
	log := logger.NewStdoutLogger()
	reg := NewToolRegistry()
	engine := &mockEngine{}
	integrity := &mockIntegrity{}
	handler := NewHandler(reg, engine, integrity, log)

	t.Run("Unauthorized Request", func(t *testing.T) {
		req := MCPRequest{
			Version:   "1.0",
			RequestID: "test-1",
			Tool:      "get_alerts",
			TenantID:  "T1",
		}
		resp := handler.HandleRequest(context.Background(), req)
		if resp.Status != "denied" {
			t.Errorf("Expected status denied, got %s", resp.Status)
		}
	})

	t.Run("Authorized Execution", func(t *testing.T) {
		user := &auth.IdentityUser{
			ID:          "user-1",
			TenantID:    "T1",
			Permissions: []string{"*"},
		}
		ctx := auth.ContextWithUser(context.Background(), user)

		req := MCPRequest{
			Version:   "1.0",
			RequestID: "test-2",
			Tool:      "get_alerts",
			TenantID:  "T1",
			Params: map[string]any{
				"limit": 10,
			},
		}

		resp := handler.HandleRequest(ctx, req)
		if resp.Status != "success" {
			t.Errorf("Expected status success, got %s (Error: %+v)", resp.Status, resp.Error)
		}

		audit := resp.Audit
		if audit.InputHash == "" || audit.ExecutionHash == "" {
			t.Errorf("Missing audit hashes")
		}
	})

	t.Run("Simulation Mode", func(t *testing.T) {
		user := &auth.IdentityUser{
			ID:          "user-1",
			TenantID:    "T1",
			Permissions: []string{auth.PermMCPSimulate},
		}
		ctx := auth.ContextWithUser(context.Background(), user)

		req := MCPRequest{
			Version:   "1.0",
			RequestID: "test-3",
			Tool:      "enrich_indicator", // non-critical
			Mode:      "simulate",
			TenantID:  "T1",
			Params: map[string]any{
				"type":  "ip",
				"value": "1.1.1.1",
			},
		}

		resp := handler.HandleRequest(ctx, req)
		if resp.Status != "success" {
			t.Errorf("Expected success, got %s (Error: %+v)", resp.Status, resp.Error)
			return
		}
		if resp.Data == nil {
			t.Errorf("Expected data in response")
			return
		}
		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Errorf("Expected map[string]any, got %T", resp.Data)
			return
		}
		if _, ok := data["simulated"]; !ok {
			t.Errorf("Expected simulated result")
		}
	})

	t.Run("Approval Required", func(t *testing.T) {
		user := &auth.IdentityUser{
			ID:          "user-1",
			TenantID:    "T1",
			Permissions: []string{auth.PermMCPExecute},
			RoleName:    string(auth.RoleAdmin),
		}
		ctx := auth.ContextWithUser(context.Background(), user)

		// Bug fix: tool name mismatch with the registry.
		//   - `engine.go` treats `isolate_host` and `quarantine_host`
		//     as aliases for the same execution path.
		//   - `registry.go` only registers the canonical name
		//     `quarantine_host` (the one that actually carries
		//     `RequiresApproval: true, RiskLevel: critical`).
		// So a request for `isolate_host` short-circuits at
		// `GetTool` with TOOL_NOT_FOUND BEFORE reaching the approval
		// gate, and the test gets back `error` instead of
		// `pending_approval`. Use the canonical registry name; the
		// alias-vs-canonical inconsistency is a separate API hygiene
		// item tracked in the open backlog.
		req := MCPRequest{
			Version:   "1.0",
			RequestID: "test-4",
			Tool:      "quarantine_host", // requires approval (critical)
			TenantID:  "T1",
		}

		resp := handler.HandleRequest(ctx, req)
		if resp.Status != "pending_approval" {
			t.Errorf("Expected status pending_approval, got %s (Error: %+v)", resp.Status, resp.Error)
		}
	})
}
