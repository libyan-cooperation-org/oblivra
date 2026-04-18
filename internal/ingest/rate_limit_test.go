package ingest

import (
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestPartitionedPipeline_RateLimiting(t *testing.T) {
	log := logger.NewStdoutLogger()
	// Create pipeline with very low limit for testing
	pp := NewPartitionedPipeline(100, nil, nil, nil, nil, log, nil, nil)
	pp.maxEPS = 10 // 10 events per second

	tenantA := "tenant-alpha"
	tenantB := "tenant-beta"

	const floodSize = 50

	// 1. Flood Tenant A
	droppedA := 0
	for i := 0; i < floodSize; i++ {
		evt := &events.SovereignEvent{
			TenantID:  tenantA,
			Host:      "host-1",
			EventType: "test",
		}
		err := pp.QueueEvent(evt)
		if err != nil {
			t.Errorf("Unexpected error from QueueEvent: %v", err)
		}
	}

	// We expect significant drops for A
	if pp.Metrics.DroppedEvents.Load() == 0 {
		t.Errorf("Expected dropped events for flooded tenant A, got 0")
	}
	droppedA = int(pp.Metrics.DroppedEvents.Load())
	t.Logf("Tenant A dropped: %d/%d", droppedA, floodSize)

	// 2. Verify Tenant B is NOT affected
	// Reset metrics (mock simulation by checking delta)
	beforeB := pp.Metrics.DroppedEvents.Load()
	
	// Send a few events for B slowly
	for i := 0; i < 5; i++ {
		evt := &events.SovereignEvent{
			TenantID:  tenantB,
			Host:      "host-2",
			EventType: "test",
		}
		pp.QueueEvent(evt)
		time.Sleep(10 * time.Millisecond)
	}

	afterB := pp.Metrics.DroppedEvents.Load()
	if afterB > beforeB {
		t.Errorf("Tenant B events were dropped (%d), but B was under limit", afterB-beforeB)
	}
}
