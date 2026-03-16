package ingest

import (
	"context"
	"hash/fnv"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/storage"
	"github.com/kingknull/oblivrashell/internal/temporal"
)

// PartitionCount is the number of independent pipeline shards.
// Each shard owns its own worker pool, state, and buffer — so correlation
// state (e.g. brute-force thresholds per source IP) remains CPU-local and
// never needs a mutex across shards.
//
// Power-of-two so the hash modulo is a cheap bitmask.
const PartitionCount = 8

// PartitionedPipeline fans events across N independent Pipeline shards
// keyed by a stable partition field (HostID → consistent shard).
//
// Benefits over a single pipeline:
//   - CPU scales linearly with cores — each shard runs its own worker pool
//   - Correlation state (thresholds, sequences) stays consistent per entity
//   - No single channel becomes the bottleneck
//   - Emergency shedding is per-shard, never drops cross-entity events
type PartitionedPipeline struct {
	shards  [PartitionCount]*Pipeline
	metrics PartitionedMetrics
	log     *logger.Logger
}

// PartitionedMetrics aggregates stats across all shards.
type PartitionedMetrics struct {
	TotalProcessed  atomic.Int64
	DroppedEvents   atomic.Int64
	EventsPerSecond atomic.Int64
}

// NewPartitionedPipeline creates N pipeline shards backed by a shared WAL,
// analytics engine, SIEM store, and event bus.
func NewPartitionedPipeline(
	bufferPerShard int,
	wal *storage.WAL,
	ae *analytics.AnalyticsEngine,
	siem database.SIEMStore,
	bus *eventbus.Bus,
	log *logger.Logger,
	temporal *temporal.IntegrityService,
) *PartitionedPipeline {
	pp := &PartitionedPipeline{log: log.WithPrefix("partitioned-pipeline")}

	for i := 0; i < PartitionCount; i++ {
		pp.shards[i] = NewPipeline(bufferPerShard, wal, ae, siem, bus,
			log.WithPrefix("shard-"+string(rune('0'+i))), temporal)
	}

	return pp
}

// Start launches all shards.
func (pp *PartitionedPipeline) Start() {
	pp.log.Info("[PART] Starting %d pipeline shards (%d workers each, %d max)",
		PartitionCount, runtime.NumCPU(), runtime.NumCPU()*4)
	for _, s := range pp.shards {
		s.Start()
	}
}

// Stop drains and stops all shards.
func (pp *PartitionedPipeline) Stop() {
	var wg sync.WaitGroup
	for _, s := range pp.shards {
		wg.Add(1)
		go func(p *Pipeline) {
			defer wg.Done()
			p.Stop()
		}(s)
	}
	wg.Wait()
	pp.log.Info("[PART] All pipeline shards stopped")
}

// QueueEvent routes the event to the correct shard using a stable hash of HostID.
// Events from the same host always land in the same shard, preserving
// correlation state (brute-force counters, sequence detectors) per entity.
func (pp *PartitionedPipeline) QueueEvent(evt *events.SovereignEvent) error {
	shard := pp.shardFor(evt)
	err := shard.QueueEvent(evt)
	if err != nil {
		pp.metrics.DroppedEvents.Add(1)
	}
	return err
}

// shardFor computes the shard index for an event using FNV-1a on the partition key.
// Host is preferred; fall back to SourceIp so network events also distribute well.
func (pp *PartitionedPipeline) shardFor(evt *events.SovereignEvent) *Pipeline {
	key := evt.Host
	if key == "" {
		key = evt.SourceIp
	}
	if key == "" {
		key = evt.EventType // last resort — at least don't always hit shard 0
	}

	h := fnv.New32a()
	h.Write([]byte(key))
	idx := h.Sum32() % PartitionCount
	return pp.shards[idx]
}

// GetMetrics aggregates metrics across all shards.
func (pp *PartitionedPipeline) GetMetrics() MetricsSnapshot {
	var total, eps, dropped int64
	var bufUsage, bufCap int

	for _, s := range pp.shards {
		m := s.GetMetrics()
		total += m.TotalProcessed
		eps += m.EventsPerSecond
		dropped += m.DroppedEvents
		bufUsage += m.BufferUsage
		bufCap += m.BufferCapacity
	}

	return MetricsSnapshot{
		TotalProcessed:  total,
		EventsPerSecond: eps,
		DroppedEvents:   dropped,
		BufferUsage:     bufUsage,
		BufferCapacity:  bufCap,
	}
}

// SetDiagnosticsUpdater propagates the diagnostics updater to all shards.
func (pp *PartitionedPipeline) SetDiagnosticsUpdater(d DiagnosticsUpdater) {
	for _, s := range pp.shards {
		s.SetDiagnosticsUpdater(d)
	}
}

// Bus returns the event bus from shard 0 (all shards share the same bus).
func (pp *PartitionedPipeline) Bus() *eventbus.Bus {
	return pp.shards[0].Bus()
}

// Replay replays the WAL on shard 0 only — the WAL is shared and order is
// non-deterministic across shards anyway, so single-shard replay is correct.
func (pp *PartitionedPipeline) Replay(ctx context.Context) error {
	return pp.shards[0].Replay(ctx)
}
