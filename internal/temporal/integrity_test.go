package temporal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestIntegrityService_Monotonicity(t *testing.T) {
	dataDir := "test_temporal_logs"
	os.MkdirAll(dataDir, 0755)
	defer os.RemoveAll(dataDir)

	log, _ := logger.New(logger.Config{
		Level:      logger.DebugLevel,
		OutputPath: filepath.Join(dataDir, "test.log"),
	})
	svc := NewIntegrityService(DefaultPolicy(), nil, log)

	host := "server-01"
	now := time.Now()

	// 1. Initial event establishes high-water mark
	v := svc.ValidateTimestamp(host, now)
	if v != nil {
		t.Fatalf("Initial valid timestamp failed: %v", v)
	}

	// 2. Event in the future moves high-water mark forward
	future := now.Add(10 * time.Second)
	v = svc.ValidateTimestamp(host, future)
	if v != nil {
		t.Fatalf("Future valid timestamp failed: %v", v)
	}

	// 3. Event in the past (relative to high-water mark) triggers inversion
	past := now.Add(5 * time.Second) // 5s after first, but 5s before future (high-water)
	v = svc.ValidateTimestamp(host, past)
	if v == nil || v.Type != "sequence_manipulation" {
		t.Fatalf("Failed to detect timestamp inversion. Got: %v", v)
	}
	t.Logf("Detected inversion: %v", v.Detail)

	// 4. Jitter within grace period (100ms) should be allowed
	jitter := future.Add(-50 * time.Millisecond)
	v = svc.ValidateTimestamp(host, jitter)
	if v != nil {
		t.Fatalf("Grace period jitter incorrectly flagged: %v", v)
	}
}

func TestIntegrityService_DriftAndFleet(t *testing.T) {
	dataDir := "test_temporal_fleet_logs"
	os.MkdirAll(dataDir, 0755)
	defer os.RemoveAll(dataDir)

	log, _ := logger.New(logger.Config{
		Level:      logger.DebugLevel,
		OutputPath: filepath.Join(dataDir, "test.log"),
	})
	svc := NewIntegrityService(DefaultPolicy(), nil, log)

	// Simulate 10 baseline agents and 1 outlier
	for i := 1; i <= 10; i++ {
		svc.RecordHeartbeat(fmt.Sprintf("agent-%d", i), time.Now().Add(-100*time.Millisecond))
	}
	svc.RecordHeartbeat("outlier", time.Now().Add(-60*time.Second)) 

	report := svc.DetectFleetDrift()
	if report.TotalAgents != 11 {
		t.Fatalf("Expected 11 agents, got %d", report.TotalAgents)
	}

	foundOutlier := false
	for _, o := range report.Outliers {
		if o.HostID == "outlier" {
			foundOutlier = true
			break
		}
	}

	if !foundOutlier {
		t.Fatalf("Fleet drift failed to detect 10s outlier")
	}
	t.Logf("Fleet Audit: Mean=%v, StdDev=%v, Outliers=%d", report.MeanDriftMs, report.StdDevMs, len(report.Outliers))
}
