package monitoring

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// DiagnosticsSnapshot captures a point-in-time view of all critical platform
// subsystems. This is the payload served to the frontend Diagnostics Modal,
// giving analysts real-time confidence in data freshness during incidents.
type DiagnosticsSnapshot struct {
	// Time this snapshot was captured
	CapturedAt string `json:"captured_at"`

	// Event Bus health
	EventBus EventBusDiag `json:"event_bus"`

	// Ingest pipeline metrics (populated externally via SetIngestMetrics)
	Ingest IngestDiag `json:"ingest"`

	// Go runtime vitals
	Runtime RuntimeDiag `json:"runtime"`

	// Query subsystem latency (populated via RecordQueryLatency)
	Query QueryDiag `json:"query"`

	// Overall platform health grade: A / B / C / DEGRADED
	HealthGrade string `json:"health_grade"`
}

// EventBusDiag describes the health of the internal event bus.
type EventBusDiag struct {
	DroppedEvents   uint64  `json:"dropped_events"`
	RateLimitActive bool    `json:"rate_limit_active"`
	LagEstimateMs   float64 `json:"lag_estimate_ms"`
}

// IngestDiag mirrors the pipeline metrics relevant to analysts.
type IngestDiag struct {
	CurrentEPS      int64   `json:"current_eps"`
	TargetEPS       int64   `json:"target_eps"`
	PercentOfTarget float64 `json:"percent_of_target"`
	BufferFillPct   float64 `json:"buffer_fill_pct"`
	DroppedTotal    int64   `json:"dropped_total"`
	WorkerCount     int     `json:"worker_count"`
}

// RuntimeDiag captures Go runtime internals useful for memory pressure detection.
type RuntimeDiag struct {
	Goroutines   int     `json:"goroutines"`
	HeapAllocMB  float64 `json:"heap_alloc_mb"`
	HeapSysMB    float64 `json:"heap_sys_mb"`
	GCPauseNs    uint64  `json:"gc_pause_ns"`
	GCCount      uint32  `json:"gc_count"`
	NumCPU       int     `json:"num_cpu"`
	GoVersion    string  `json:"go_version"`
}

// QueryDiag reports DuckDB/SQLite query performance so analysts know if
// dashboards are showing stale data.
type QueryDiag struct {
	LastQueryMs    float64 `json:"last_query_ms"`
	AvgQueryMs     float64 `json:"avg_query_ms"`
	P99QueryMs     float64 `json:"p99_query_ms"`
	SlowQueryCount int64   `json:"slow_query_count"` // queries > 500ms
	TotalQueries   int64   `json:"total_queries"`
}

// DiagnosticsService collects and exposes real-time platform health data.
type DiagnosticsService struct {
	mu  sync.RWMutex
	log *logger.Logger
	bus *eventbus.Bus

	// Ingest metrics set by the pipeline via SetIngestMetrics
	ingest IngestDiag

	// Query latency ring buffer (last 100 samples)
	queryLatencies [100]float64
	queryHead      int
	queryCount     int64
	slowQueries    int64
	totalQueryMs   float64

	// Event bus dropped counter accessor
	busDropped func() uint64
}

// NewDiagnosticsService creates the platform diagnostics service.
// busDropped is a callback that returns the event bus dropped event count.
func NewDiagnosticsService(log *logger.Logger, bus *eventbus.Bus, busDropped func() uint64) *DiagnosticsService {
	return &DiagnosticsService{
		log:        log.WithPrefix("diagnostics"),
		bus:        bus,
		busDropped: busDropped,
	}
}

// SetIngestMetrics is called by the pipeline every second to update ingest stats.
func (d *DiagnosticsService) SetIngestMetrics(eps, target, dropped int64, bufFill float64, workers int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pct := 0.0
	if target > 0 {
		pct = float64(eps) / float64(target) * 100
	}
	d.ingest = IngestDiag{
		CurrentEPS:      eps,
		TargetEPS:       target,
		PercentOfTarget: pct,
		BufferFillPct:   bufFill,
		DroppedTotal:    dropped,
		WorkerCount:     workers,
	}
}

// RecordQueryLatency records the duration of a single database query in milliseconds.
// Call this from any DuckDB/SQLite query path to populate the Query diagnostics.
func (d *DiagnosticsService) RecordQueryLatency(ms float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.queryLatencies[d.queryHead%100] = ms
	d.queryHead++
	d.queryCount++
	d.totalQueryMs += ms

	if ms > 500 {
		d.slowQueries++
	}
}

// Snapshot builds a full DiagnosticsSnapshot from all subsystems.
func (d *DiagnosticsService) Snapshot() DiagnosticsSnapshot {
	d.mu.RLock()
	defer d.mu.RUnlock()

	snap := DiagnosticsSnapshot{
		CapturedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Ingest:     d.ingest,
		Runtime:    d.runtimeDiag(),
		Query:      d.queryDiag(),
	}

	// Event bus
	dropped := uint64(0)
	if d.busDropped != nil {
		dropped = d.busDropped()
	}
	snap.EventBus = EventBusDiag{
		DroppedEvents:   dropped,
		RateLimitActive: dropped > 0,
		// Lag estimate: if EPS is < 50% of target, estimate lag proportionally
		LagEstimateMs: d.estimateLagMs(),
	}

	snap.HealthGrade = d.gradeHealth(snap)
	return snap
}

// StartPeriodicBroadcast publishes a DiagnosticsSnapshot to the event bus every interval.
// The frontend subscribes to receive live updates without polling.
func (d *DiagnosticsService) StartPeriodicBroadcast(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if d.bus != nil {
					d.bus.Publish("diagnostics:snapshot", d.Snapshot())
				}
			}
		}
	}()
}

// runtimeDiag reads Go runtime stats without stopping the world.
func (d *DiagnosticsService) runtimeDiag() RuntimeDiag {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return RuntimeDiag{
		Goroutines:  runtime.NumGoroutine(),
		HeapAllocMB: float64(ms.HeapAlloc) / 1024 / 1024,
		HeapSysMB:   float64(ms.HeapSys) / 1024 / 1024,
		GCPauseNs:   ms.PauseNs[(ms.NumGC+255)%256],
		GCCount:     ms.NumGC,
		NumCPU:      runtime.NumCPU(),
		GoVersion:   runtime.Version(),
	}
}

// queryDiag computes avg and p99 from the ring buffer.
func (d *DiagnosticsService) queryDiag() QueryDiag {
	if d.queryCount == 0 {
		return QueryDiag{}
	}

	samples := d.queryCount
	if samples > 100 {
		samples = 100
	}

	// Collect populated samples
	vals := make([]float64, 0, samples)
	for i := int64(0); i < samples; i++ {
		idx := (d.queryHead - 1 - int(i) + 100) % 100
		vals = append(vals, d.queryLatencies[idx])
	}

	// Avg
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	avg := sum / float64(len(vals))

	// P99: sort a copy, take 99th percentile
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sortFloat64s(sorted)
	p99Idx := int(float64(len(sorted)) * 0.99)
	if p99Idx >= len(sorted) {
		p99Idx = len(sorted) - 1
	}

	return QueryDiag{
		LastQueryMs:    d.queryLatencies[(d.queryHead-1+100)%100],
		AvgQueryMs:     avg,
		P99QueryMs:     sorted[p99Idx],
		SlowQueryCount: d.slowQueries,
		TotalQueries:   d.queryCount,
	}
}

func (d *DiagnosticsService) estimateLagMs() float64 {
	if d.ingest.TargetEPS == 0 {
		return 0
	}
	if d.ingest.CurrentEPS >= d.ingest.TargetEPS {
		return 0
	}
	// Simple model: lag grows linearly as EPS falls below target
	deficit := float64(d.ingest.TargetEPS-d.ingest.CurrentEPS) / float64(d.ingest.TargetEPS)
	return deficit * 2000 // max ~2s estimated lag at 0 EPS
}

// gradeHealth returns a letter grade for the overall platform health.
func (d *DiagnosticsService) gradeHealth(snap DiagnosticsSnapshot) string {
	issues := 0

	if snap.Ingest.BufferFillPct > 80 {
		issues += 2
	} else if snap.Ingest.BufferFillPct > 50 {
		issues++
	}

	if snap.Ingest.PercentOfTarget < 50 {
		issues += 2
	} else if snap.Ingest.PercentOfTarget < 80 {
		issues++
	}

	if snap.EventBus.DroppedEvents > 10000 {
		issues += 2
	} else if snap.EventBus.DroppedEvents > 1000 {
		issues++
	}

	if snap.Runtime.Goroutines > 10000 {
		issues++
	}

	if snap.Query.P99QueryMs > 1000 {
		issues += 2
	} else if snap.Query.P99QueryMs > 500 {
		issues++
	}

	switch {
	case issues == 0:
		return "A"
	case issues == 1:
		return "B"
	case issues == 2:
		return "C"
	default:
		return "DEGRADED"
	}
}

// sortFloat64s is a simple insertion sort for small slices (avoids importing sort).
func sortFloat64s(a []float64) {
	for i := 1; i < len(a); i++ {
		key := a[i]
		j := i - 1
		for j >= 0 && a[j] > key {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = key
	}
}
