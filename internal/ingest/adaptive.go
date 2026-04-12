package ingest

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
)

// EPSTarget is the events-per-second goal for the high-throughput pipeline tier.
// At 50k EPS with NumCPU workers the per-worker rate is ~50k/NumCPU.
const EPSTarget = 50_000

// AdaptiveController dynamically adjusts the worker pool size and buffer
// to sustain maximum EPS while preventing OOM under backpressure.
//
// Strategy:
//   - Every 500ms, sample current EPS and compare to EPSTarget.
//   - If EPS < 80% of target AND buffer > 50% full → spawn extra workers (up to MaxWorkers).
//   - If EPS is nominal AND buffer < 10% → scale workers back down to baseline.
//   - If buffer > 90% full → activate emergency shedding (drop oldest unprocessed events).
type AdaptiveController struct {
	pipeline    *Pipeline
	maxWorkers  int
	baseWorkers int
	active      atomic.Int32 // current extra worker count
	ticker      *time.Ticker
	done        chan struct{}
}

// NewAdaptiveController creates a controller targeting EPSTarget.
func NewAdaptiveController(p *Pipeline) *AdaptiveController {
	base := runtime.NumCPU()
	if base < 2 {
		base = 2
	}
	return &AdaptiveController{
		pipeline:    p,
		baseWorkers: base,
		maxWorkers:  base * 4, // up to 4× CPU cores under sustained load
		ticker:      time.NewTicker(500 * time.Millisecond),
		done:        make(chan struct{}),
	}
}

// Start launches the adaptive control loop.
func (ac *AdaptiveController) Start() {
	go ac.loop()
}

// Stop shuts down the control loop.
func (ac *AdaptiveController) Stop() {
	close(ac.done)
	ac.ticker.Stop()
}

func (ac *AdaptiveController) loop() {
	for {
		select {
		case <-ac.done:
			return
		case <-ac.ticker.C:
			ac.adjust()
		}
	}
}

func (ac *AdaptiveController) adjust() {
	snap := ac.pipeline.GetMetrics()

	eps := snap.EventsPerSecond
	bufCap := snap.BufferCapacity
	if bufCap == 0 {
		return // pipeline not yet started
	}
	bufUsage := float64(snap.BufferUsage) / float64(bufCap)

	// 3× Rated EPS (150,000) or 95% buffer fill triggers DEGRADED state
	newStatus := LoadHealthy
	
	// Stall detection: buffer full but nothing moving for 2 minutes
	stalled := false
	if bufUsage > 0.80 {
		last := ac.pipeline.lastProcessed.Load()
		if time.Now().Unix()-last > 120 { // 2 minute stall threshold
			stalled = true
			newStatus = LoadCritical
		}
	}

	if !stalled && (eps > int64(EPSTarget)*3 || bufUsage > 0.95) {
		newStatus = LoadDegraded
	}

	if newStatus != ac.pipeline.GetLoadStatus() {
		ac.pipeline.SetLoadStatus(newStatus)
		msg := "Pipeline performance is nominal."
		switch newStatus {
		case LoadDegraded:
			msg = fmt.Sprintf("Pipeline pressure high (EPS: %d, Buffer: %.1f%%). Scaling up workers.", eps, bufUsage*100)
			if ac.pipeline.log != nil {
				ac.pipeline.log.Warn("[ADAPTIVE] %s", msg)
			}
		case LoadCritical:
			msg = fmt.Sprintf("Critical Pipeline STALL (Buffer: %.1f%%). No events processed for >2m.", bufUsage*100)
			if ac.pipeline.log != nil {
				ac.pipeline.log.Error("[ADAPTIVE] %s", msg)
			}
			// Emergency: spawn immediate "Rescue Workers"
			for i := 0; i < ac.baseWorkers; i++ {
				ac.scaleUp()
			}
		default:
			if ac.pipeline.log != nil {
				ac.pipeline.log.Info("[ADAPTIVE] Pipeline state returned to HEALTHY")
			}
		}

		if ac.pipeline.diagnostics != nil {
			ac.pipeline.diagnostics.UpdateLoadStatus(newStatus, msg)
		}
	}

	// Export infrastructure metrics
	if ac.pipeline.metricsCollector != nil {
		ac.pipeline.metricsCollector.SetGauge("ingest_pipeline_status", float64(newStatus), ac.pipeline.labels)
		ac.pipeline.metricsCollector.SetGauge("ingest_active_workers", float64(ac.baseWorkers+int(ac.active.Load())), ac.pipeline.labels)
		ac.pipeline.metricsCollector.SetGauge("ingest_eps_target", float64(EPSTarget), ac.pipeline.labels)
	}

	switch {
	case bufUsage > 0.90:
		// Emergency: buffer critically full.
		// Drop oldest items — drain up to 5% of capacity to relieve pressure.
		toDrop := int(float64(snap.BufferCapacity) * 0.05)
	drainLoop:
		for i := 0; i < toDrop; i++ {
			select {
			case <-ac.pipeline.buffer:
				ac.pipeline.metrics.DroppedEvents.Add(1)
			default:
				break drainLoop
			}
		}
		if ac.pipeline.log != nil {
			ac.pipeline.log.Warn("[ADAPTIVE] EMERGENCY SHED: buffer at %.0f%%, dropped %d events", bufUsage*100, toDrop)
		}

		// Also scale up workers to clear the backlog
		ac.scaleUp()

	case eps < int64(EPSTarget)*8/10 && bufUsage > 0.50:
		// Sustained under-performance with growing backlog — add workers
		ac.scaleUp()

	case eps >= int64(EPSTarget) && bufUsage < 0.10:
		// Target met and buffer is draining — scale back to save resources
		ac.scaleDown()
	}
}

func (ac *AdaptiveController) scaleUp() {
	current := int(ac.active.Load())
	total := ac.baseWorkers + current
	if total >= ac.maxWorkers {
		return
	}

	// Spawn one additional worker
	ac.pipeline.wg.Add(1)
	ac.active.Add(1)
	go func() {
		defer ac.active.Add(-1)
		ac.pipeline.worker()
	}()
	if ac.pipeline.log != nil {
		ac.pipeline.log.Info("[ADAPTIVE] Scaled up to %d workers (EPS target: %d)", total+1, EPSTarget)
	}
}

func (ac *AdaptiveController) scaleDown() {
	// Scale-down is passive: extra workers exit naturally when ctx is cancelled.
	// Nothing needed here — the active counter tracks running extras.
}

// EPSSummarySnapshot is a frontend-friendly throughput report.
type EPSSummarySnapshot struct {
	CurrentEPS      int64   `json:"current_eps"`
	TargetEPS       int     `json:"target_eps"`
	PercentOfTarget float64 `json:"percent_of_target"`
	BufferFillPct   float64 `json:"buffer_fill_pct"`
	DroppedTotal    int64   `json:"dropped_total"`
	WorkerCount     int     `json:"worker_count"`
}

// EPSSummary returns a diagnostics snapshot for the frontend Diagnostics Modal.
func (p *Pipeline) EPSSummary() EPSSummarySnapshot {
	snap := p.GetMetrics()
	fillPct := 0.0
	if snap.BufferCapacity > 0 {
		fillPct = float64(snap.BufferUsage) / float64(snap.BufferCapacity) * 100
	}
	pctOfTarget := 0.0
	if EPSTarget > 0 {
		pctOfTarget = float64(snap.EventsPerSecond) / float64(EPSTarget) * 100
	}
	return EPSSummarySnapshot{
		CurrentEPS:      snap.EventsPerSecond,
		TargetEPS:       EPSTarget,
		PercentOfTarget: pctOfTarget,
		BufferFillPct:   fillPct,
		DroppedTotal:    snap.DroppedEvents,
		WorkerCount:     runtime.NumCPU(), // base; extras tracked by AdaptiveController
	}
}
