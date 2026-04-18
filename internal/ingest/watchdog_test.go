package ingest

import (
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestWatchdogStallDetection(t *testing.T) {
	// 1. Setup a small pipeline with a real logger
	bufferSize := 10
	log, _ := logger.New(logger.Config{Level: logger.InfoLevel, OutputPath: "/dev/null"})
	p := NewPipeline(bufferSize, nil, nil, nil, nil, log, nil, nil, nil, nil)
	
	// 2. Mock a stall: fill buffer and set backdated heartbeat
	for i := 0; i < 9; i++ { // 90% full
		p.buffer <- &events.SovereignEvent{TenantID: "test"}
		p.quota.Inc("test")
	}
	
	// Force backdated heartbeat (3 minutes ago)
	p.lastProcessed.Store(time.Now().Add(-180 * time.Second).Unix())
	
	// 3. Trigger adaptive adjustment
	ac := NewAdaptiveController(p)
	ac.adjust()
	
	// 4. Verify Critical status and Rescue Workers
	if p.GetLoadStatus() != LoadCritical {
		t.Errorf("expected LoadCritical status after stall, got %v", p.GetLoadStatus())
	}
	
	// Rescue workers should have been spawned (scaleUp called baseWorkers times)
	if ac.active.Load() == 0 {
		t.Error("expected rescue workers to be spawned, got 0")
	}
	
	t.Logf("Watchdog correctly identified stall and spawned %d rescue workers", ac.active.Load())
}
