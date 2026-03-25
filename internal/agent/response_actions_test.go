package agent

import (
	"os"
	"testing"

	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestResponseActionKillProcessRejectsInvalid(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	executor := NewResponseActionExecutor(log)

	// Invalid PID
	if err := executor.KillProcess(0); err == nil {
		t.Error("Expected error for PID 0")
	}
	if err := executor.KillProcess(-1); err == nil {
		t.Error("Expected error for negative PID")
	}

	// PID 1 (init) should be refused
	if err := executor.KillProcess(1); err == nil {
		t.Error("Expected error for PID 1")
	}

	// Own PID should be refused
	if err := executor.KillProcess(os.Getpid()); err == nil {
		t.Error("Expected error for own PID")
	}
}

func TestResponseActionCollectSnapshot(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	executor := NewResponseActionExecutor(log)

	// Snapshot of own process should succeed (we just read metadata)
	snapshot, err := executor.CollectProcessSnapshot(os.Getpid())
	if err != nil {
		t.Fatalf("CollectProcessSnapshot of self failed: %v", err)
	}

	if snapshot.PID != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), snapshot.PID)
	}

	if snapshot.CapturedAt == "" {
		t.Error("Expected non-empty CapturedAt timestamp")
	}

	t.Logf("Self snapshot: PID=%d, Name=%q, OpenFiles=%d", snapshot.PID, snapshot.Name, snapshot.OpenFiles)
}

func TestResponseActionRejectsInvalidSnapshot(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	executor := NewResponseActionExecutor(log)

	_, err := executor.CollectProcessSnapshot(0)
	if err == nil {
		t.Error("Expected error for PID 0 snapshot")
	}

	_, err = executor.CollectProcessSnapshot(-1)
	if err == nil {
		t.Error("Expected error for negative PID snapshot")
	}
}
