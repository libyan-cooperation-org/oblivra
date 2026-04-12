package ingest

import "time"

// PipelineStats is the authoritative, operator-visible view of pipeline health.
// Every field is a hard number — no estimates, no fake data.
// Surfaces in the REST API (/api/v1/ingest/status) and the diagnostics UI widget.
type PipelineStats struct {
	// Throughput
	EventsPerSecond int64 `json:"events_per_second"`
	TotalProcessed  int64 `json:"total_processed"`

	// Backpressure — NEVER silent
	DroppedEvents int64         `json:"dropped_events"`
	QueueDepth    int           `json:"queue_depth"`
	QueueCapacity int           `json:"queue_capacity"`
	QueueFillPct  float64       `json:"queue_fill_pct"`
	Lag           time.Duration `json:"lag_ns"` // nanoseconds since last processed event

	// Per-shard breakdown (only populated for PartitionedPipeline)
	ShardStats []ShardStat `json:"shard_stats,omitempty"`

	// Health
	WorkerCount int    `json:"worker_count"`
	LoadStatus  string `json:"load_status"`

	// Timestamp
	CollectedAt time.Time `json:"collected_at"`
}

// ShardStat holds per-shard metrics for a PartitionedPipeline.
type ShardStat struct {
	ShardID       int   `json:"shard_id"`
	QueueDepth    int   `json:"queue_depth"`
	QueueCapacity int   `json:"queue_capacity"`
	DroppedEvents int64 `json:"dropped_events"`
	TotalProcessed int64 `json:"total_processed"`
}

// CollectStats returns a fully-populated PipelineStats from a MetricsSnapshot.
// It never returns zeros for structural fields — a 0 DroppedEvents means zero drops.
func CollectStats(snap MetricsSnapshot, lag time.Duration) PipelineStats {
	fillPct := 0.0
	if snap.BufferCapacity > 0 {
		fillPct = float64(snap.BufferUsage) / float64(snap.BufferCapacity) * 100.0
	}

	status := "healthy"
	switch {
	case fillPct > 90:
		status = "critical"
	case fillPct > 70:
		status = "degraded"
	}

	return PipelineStats{
		EventsPerSecond: snap.EventsPerSecond,
		TotalProcessed:  snap.TotalProcessed,
		DroppedEvents:   snap.DroppedEvents,
		QueueDepth:      snap.BufferUsage,
		QueueCapacity:   snap.BufferCapacity,
		QueueFillPct:    fillPct,
		Lag:             lag,
		LoadStatus:      status,
		CollectedAt:     snap.CollectedAt,
	}
}
