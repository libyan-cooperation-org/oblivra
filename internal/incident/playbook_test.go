package incident

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type mockSSH struct {
	calls []string
}

func (m *mockSSH) Disconnect(sessionID string) error {
	m.calls = append(m.calls, "disconnect:"+sessionID)
	return nil
}

func (m *mockSSH) Exec(sessionID string, cmd string) (string, error) {
	m.calls = append(m.calls, "exec:"+cmd)
	return "ok", nil
}

type mockNotif struct{}

func (m *mockNotif) SendAlert(title, message string) {}

func TestPlaybookEngine_ApprovalAndRollback(t *testing.T) {
	dataDir := "test_soar_logs"
	os.MkdirAll(dataDir, 0755)
	defer os.RemoveAll(dataDir)

	ssh := &mockSSH{}
	log, _ := logger.New(logger.Config{
		Level:      logger.DebugLevel,
		OutputPath: filepath.Join(dataDir, "test.log"),
	})
	bus := eventbus.NewBus(log)

	engine := NewPlaybookEngine(ssh, &mockNotif{}, nil, nil, bus, log)

	ctx := context.Background()
	params := map[string]interface{}{
		"session_id": "session1",
		"ip":         "10.0.0.1",
	}

	// 1. Test Approval Gating
	output, err := engine.ExecuteAction(ctx, "block_ip", params)
	if err != nil {
		t.Fatalf("ExecuteAction failed: %v", err)
	}
	if output != "PENDING_APPROVAL" {
		t.Fatalf("Expected PENDING_APPROVAL, got %s", output)
	}
	if len(ssh.calls) != 0 {
		t.Fatalf("Action executed without approval!")
	}

	// 2. Test Execution (Directly via registry for simplicity in test, or override metadata)
	// We'll manually register a non-approval action to test rollback
	engine.registry.Register(ResponseAction{
		Name:         "test_action",
		Execute:      engine.registry.BlockIP,
		RollbackName: "unblock_ip",
	})

	_, err = engine.ExecuteAction(ctx, "test_action", params)
	if err != nil {
		t.Fatalf("ExecuteAction failed: %v", err)
	}

	if len(ssh.calls) != 1 || ssh.calls[0] != "exec:sudo iptables -A INPUT -s 10.0.0.1 -j DROP" {
		t.Fatalf("Wrong SSH call sequence: %v", ssh.calls)
	}

	// 3. Test Rollback
	err = engine.Rollback(ctx)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	if len(ssh.calls) != 2 || ssh.calls[1] != "exec:sudo iptables -D INPUT -s 10.0.0.1 -j DROP" {
		t.Fatalf("Rollback SSH call failed or missing: %v", ssh.calls)
	}

	if len(engine.actionHistory) != 0 {
		t.Fatalf("Action history not cleared after rollback")
	}
}
