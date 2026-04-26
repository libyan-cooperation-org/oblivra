package ingest

import (
	"os"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/temporal"
)

func TestPipeline_TemporalIntegrity(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	bus := eventbus.NewBus(log)
	t.Cleanup(bus.Close) // Phase 25.4: drain bus workers on test exit
	ae := analytics.NewAnalyticsEngine(log)

	// Create Temporal Integrity Service with a strict policy
	policy := temporal.Policy{
		MaxFutureSkew: 1 * time.Minute,
		MaxPastAge:    24 * time.Hour,
	}
	temporalSvc := temporal.NewIntegrityService(policy, bus, log)

	// Initialize pipeline with temporal integrity
	p := NewPipeline(100, nil, ae, nil, bus, log, temporalSvc, nil, nil, nil)
	p.Start()
	defer p.Shutdown()

	// 1. Valid event
	p.QueueEvent(&events.SovereignEvent{
		Host:      "test-host",
		Timestamp: time.Now().Format(time.RFC3339),
		RawLine:   "Valid log entry",
	})

	// 2. Futuristic event (should trigger violation)
	futureTime := time.Now().Add(10 * time.Minute)
	p.QueueEvent(&events.SovereignEvent{
		Host:      "test-host-skewed",
		Timestamp: futureTime.Format(time.RFC3339),
		RawLine:   "Futuristic log entry",
	})

	// Process events (pipeline is asynchronous)
	time.Sleep(500 * time.Millisecond)

	// Wait for violation in the service
	vs := temporalSvc.GetViolations()
	found := false
	for _, v := range vs {
		if v.HostID == "test-host-skewed" && v.Type == "future_event" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected future_event violation for test-host-skewed not found in temporal service")
	}
}
