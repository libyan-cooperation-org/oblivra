package app

import (
	"os"
	"testing"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestGovernanceFalsePositive(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	bus := eventbus.NewBus(log)

	svc := NewGovernanceService(bus, log)

	// Test marking FP
	evidence := []map[string]interface{}{
		{"metric": "z-score", "value": 4.5},
	}
	err := svc.MarkFalsePositive("anomaly-123", "Legitimate admin activity", evidence)
	if err != nil {
		t.Fatalf("Failed to mark false positive: %v", err)
	}

	logs := svc.GetBiasLogs()
	if len(logs) != 1 {
		t.Fatalf("Expected 1 bias log, got %d", len(logs))
	}

	if logs[0].AnomalyID != "anomaly-123" {
		t.Errorf("Expected anomaly-123, got %s", logs[0].AnomalyID)
	}

	if logs[0].Reason != "Legitimate admin activity" {
		t.Errorf("Expected reason, got %s", logs[0].Reason)
	}
}
