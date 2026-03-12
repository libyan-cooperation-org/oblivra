package services

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestRuntimeTrustService_CalculateTrustIndex(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")
	log, _ := logger.New(logger.Config{Level: logger.InfoLevel, OutputPath: logPath})
	defer log.Close()
	bus := eventbus.NewBus(log)
	
	// Create service without attestation for baseline
	s := NewRuntimeTrustService(bus, log, nil, nil)
	
	// Manually set some statuses
	s.updateStatus("Vault Integrity", "TRUSTED", "OK")
	s.updateStatus("Attestation State", "TRUSTED", "OK")
	s.updateStatus("Detection Rules", "TRUSTED", "OK")
	s.updateStatus("Policy Engine", "TRUSTED", "OK")
	s.updateStatus("Audit Trail", "TRUSTED", "OK")
	
	index := s.CalculateTrustIndex()
	if index != 100.0 {
		t.Errorf("Expected trust index 100.0, got %f", index)
	}
	
	// Simulate a warning
	s.updateStatus("Detection Rules", "WARNING", "Rules missing")
	index = s.CalculateTrustIndex()
	if index != 92.5 {
		t.Errorf("Expected trust index 92.5, got %f", index)
	}
	
	// Simulate an untrusted state
	s.updateStatus("Attestation State", "UNTRUSTED", "Hash mismatch")
	index = s.CalculateTrustIndex()
	if index != 72.5 {
		t.Errorf("Expected trust index 72.5, got %f", index)
	}
}

func TestRuntimeTrustService_IntegrationWithAttestation(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test_int.log")
	log, _ := logger.New(logger.Config{Level: logger.InfoLevel, OutputPath: logPath})
	defer log.Close()
	bus := eventbus.NewBus(log)
	
	att := attestation.NewAttestationService()
	s := NewRuntimeTrustService(bus, log, att, nil)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	_ = s.Start(ctx)
	
	// Wait a bit for the async verification
	time.Sleep(100 * time.Millisecond)
	
	status := s.GetAggregatedStatus()
	found := false
	for _, st := range status {
		if st.Component == "Attestation State" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Attestation State not found in trust status")
	}
}
func TestRuntimeTrustService_DriftPrediction(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test_drift.log")
	log, _ := logger.New(logger.Config{Level: logger.InfoLevel, OutputPath: logPath})
	defer log.Close()
	bus := eventbus.NewBus(log)
	
	s := NewRuntimeTrustService(bus, log, nil, nil)
	
	// Pre-populate history to simulate drift
	// T=0: 100.0 score
	s.updateStatus("Vault Integrity", "TRUSTED", "OK")
	s.updateStatus("Attestation State", "TRUSTED", "OK")
	s.updateStatus("Detection Rules", "TRUSTED", "OK")
	s.updateStatus("Policy Engine", "TRUSTED", "OK")
	s.updateStatus("Audit Trail", "TRUSTED", "OK")
	
	s.history = append(s.history, TrustSnapshot{
		Score:     100.0,
		Pillars:   s.GetPillarScores(),
		Timestamp: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	})
	
	// T=1h: Attestation State drops to UNTRUSTED (-20.0 pts)
	s.updateStatus("Attestation State", "UNTRUSTED", "Compromised")
	s.history = append(s.history, TrustSnapshot{
		Score:     80.0, // 100 - 20
		Pillars:   s.GetPillarScores(),
		Timestamp: time.Now().Format(time.RFC3339),
	})
	
	metrics := s.GetTrustDriftMetrics()
	
	if metrics.VelocityPerHour != -20.0 {
		t.Errorf("Expected velocity -20.0/hr, got %f", metrics.VelocityPerHour)
	}
	
	if !metrics.IsBleeding {
		t.Error("Expected IsBleeding to be true")
	}
	
	// Distance to 50.0 is 30 pts (80 - 50). Velocity is -20/hr. 
	// ETTF = 30 / 20 = 1.5h = 1h 30m
	expectedETTF := "1h 30m"
	if metrics.EstimatedFailureTime != expectedETTF {
		t.Errorf("Expected ETTF %s, got %s", expectedETTF, metrics.EstimatedFailureTime)
	}
	
	// Verify pillar trends
	found := false
	for _, p := range metrics.PillarTrends {
		if p.Component == "Attestation State" {
			found = true
			if p.Trend != "Falling" {
				t.Errorf("Expected Attestation State trend Falling, got %s", p.Trend)
			}
			if p.Velocity != -20.0 {
				t.Errorf("Expected Attestation State velocity -20.0, got %f", p.Velocity)
			}
		}
	}
	if !found {
		t.Error("Attestation State trend not found in metrics")
	}
}
