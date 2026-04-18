package security

import (
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestHoneypotService(t *testing.T) {
	l, _ := logger.New(logger.Config{
		Level:      logger.InfoLevel,
		OutputPath: "test.log",
	})
	svc := NewHoneypotService(l)

	// Test Injection
	id := svc.InjectHoneypotCredential("test-1", "decoy_admin")
	if id != "test-1" {
		t.Errorf("expected ID test-1, got %s", id)
	}

	decoys := svc.GetDecoyStatus()
	if len(decoys) != 1 {
		t.Fatalf("expected 1 decoy, got %d", len(decoys))
	}
	if decoys[0].Value != "decoy_admin" {
		t.Errorf("expected value decoy_admin, got %s", decoys[0].Value)
	}

	// Test Trigger
	svc.RegisterTrigger("test-1")
	decoys = svc.GetDecoyStatus()
	if decoys[0].LastTrigger == nil {
		t.Error("expected LastTrigger to be set after trigger")
	}

	if time.Since(parseTime(*decoys[0].LastTrigger)) > time.Second {
		t.Error("LastTrigger time is too far in the past")
	}

	// Test non-existent trigger
	svc.RegisterTrigger("invalid") // Should not panic
}
