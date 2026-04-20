package ingest

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/storage"
	"github.com/kingknull/oblivrashell/internal/temporal"
	"github.com/kingknull/oblivrashell/internal/messaging"
	"github.com/kingknull/oblivrashell/internal/integrity"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/engine/dag"
	"github.com/kingknull/oblivrashell/internal/graph"
	"golang.org/x/time/rate"
)

// PartitionCount is the number of independent pipeline shards.
// Each shard owns its own worker pool, state, and buffer — so correlation
// state (e.g. brute-force thresholds per source IP) remains CPU-local and
// never needs a mutex across shards.
//
// Power-of-two so the hash modulo is a cheap bitmask.
// PartitionCount is the number of independent pipeline shards.
// Default is 8, but can be overridden via OBLIVRA_PARTITION_COUNT.
var PartitionCount = 8

func init() {
	if val := getEnvInt("OBLIVRA_PARTITION_COUNT", 8); val > 0 {
		PartitionCount = val
	}
}

// PartitionedPipeline fans events across N independent Pipeline shards
// keyed by a stable partition field (HostID → consistent shard).
//
// Benefits over a single pipeline:
//   - CPU scales linearly with cores — each shard runs its own worker pool
//   - Correlation state (thresholds, sequences) stays consistent per entity
//   - No single channel becomes the bottleneck
//   - Emergency shedding is per-shard, never drops cross-entity events
type PartitionedPipeline struct {
	shards    []*Pipeline
	Metrics   PartitionedMetrics
	limiters  map[string]*rate.Limiter
	limiterMu sync.RWMutex
	maxEPS    int
	log              *logger.Logger
	metricsCollector *monitoring.MetricsCollector
	wal              *storage.WAL
	closeOnce        sync.Once
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
	mc *monitoring.MetricsCollector,
) *PartitionedPipeline {
	pp := &PartitionedPipeline{
		shards:           make([]*Pipeline, PartitionCount),
		log:              log.WithPrefix("partitioned-pipeline"),
		limiters:         make(map[string]*rate.Limiter),
		maxEPS:           getEnvInt("OBLIVRA_MAX_EPS_PER_TENANT", 1000),
		metricsCollector: mc,
		wal:              wal,
	}

	for i := 0; i < PartitionCount; i++ {
		shardID := fmt.Sprintf("%d", i)
		pp.shards[i] = NewPipeline(bufferPerShard, wal, ae, siem, bus,
			log.WithPrefix("shard-"+shardID), temporal, mc, map[string]string{"shard": shardID}, nil)
	}

	return pp
}

// Start launches all shards.
func (pp *PartitionedPipeline) Start() {
	pp.log.Info("[PART] Starting %d pipeline shards (%d workers each, %d max)",
		PartitionCount, runtime.NumCPU(), runtime.NumCPU()*4)
	
	// Replay WAL once before starting any shards to avoid race on single WAL file
	if err := pp.Replay(context.Background()); err != nil {
		pp.log.Error("[PART] Global WAL Replay failed: %v", err)
	}

	for _, s := range pp.shards {
		s.Start()
	}
}

// Stop flushes all shards and shuts down workers.
func (pp *PartitionedPipeline) Stop() {
	pp.log.Info("[PART] Shutting down %d pipeline shards...", len(pp.shards))
	for _, shard := range pp.shards {
		shard.Stop()
	}

	pp.closeOnce.Do(func() {
		if pp.wal != nil {
			if err := pp.wal.Close(); err != nil {
				pp.log.Error("[PART] Failed to close shared WAL: %v", err)
			}
		}
	})

	pp.log.Info("[PART] All pipeline shards stopped")
}

// SetEvaluator distributes clones of the detection engine to all shards.
func (pp *PartitionedPipeline) SetEvaluator(e *detection.Evaluator) {
	if e == nil {
		return
	}
	for _, shard := range pp.shards {
		// Shard-local evaluators get their own isolation (no shared mutex contention)
		shard.SetEvaluator(e.Clone())
	}
}

// SetIntegrityTree distributes the integrity MerkleTree to all shards.
func (pp *PartitionedPipeline) SetIntegrityTree(t *integrity.MerkleTree) {
	if t == nil {
		return
	}
	for _, shard := range pp.shards {
		shard.SetIntegrityTree(t)
	}
}

// SetGraphEngine distributes the graph engine to all shards for entity extraction.
func (pp *PartitionedPipeline) SetGraphEngine(g *graph.GraphEngine) {
	if g == nil {
		return
	}
	for _, shard := range pp.shards {
		shard.SetGraphEngine(g)
	}
}

// SetIdentityResolver distributes the identity service to all shards for event enrichment.
func (pp *PartitionedPipeline) SetIdentityResolver(r dag.UserResolver) {
	if r == nil {
		return
	}
	for _, shard := range pp.shards {
		shard.SetIdentityResolver(r)
	}
}

// SetNATSService distributes the messaging service to all shards.
func (pp *PartitionedPipeline) SetNATSService(s *messaging.NATSService) {
	if s == nil {
		return
	}
	for _, shard := range pp.shards {
		shard.SetNATSService(s)
	}
}

// Shutdown drains and stops all shards.
func (pp *PartitionedPipeline) Shutdown() {
	pp.Stop()
}

// GetLoadStatus returns the worst status across all shards.
func (pp *PartitionedPipeline) GetLoadStatus() LoadStatus {
	worst := LoadHealthy
	for _, s := range pp.shards {
		status := s.GetLoadStatus()
		if status > worst {
			worst = status
		}
	}
	return worst
}

// QueueEvent routes the event to the correct shard using a stable hash of HostID.
// Events from the same host always land in the same shard, preserving
// correlation state (brute-force counters, sequence detectors) per entity.
func (pp *PartitionedPipeline) QueueEvent(evt *events.SovereignEvent) error {
	// 1. Enforce Per-Tenant Rate Limiting
	if evt.TenantID != "" {
		limiter := pp.getLimiter(evt.TenantID)
		if !limiter.Allow() {
			pp.Metrics.DroppedEvents.Add(1)
			return nil // Drop silently with metric increment (Phase 22.3 requirement)
		}
	}

	shard := pp.shardFor(evt)
	err := shard.QueueEvent(evt)
	if err != nil {
		pp.Metrics.DroppedEvents.Add(1)
	}
	return err
}

// getLimiter returns an existing limiter or creates a new one for the tenant.
func (pp *PartitionedPipeline) getLimiter(tenantID string) *rate.Limiter {
	pp.limiterMu.RLock()
	l, ok := pp.limiters[tenantID]
	pp.limiterMu.RUnlock()

	if ok {
		return l
	}

	pp.limiterMu.Lock()
	defer pp.limiterMu.Unlock()

	// Double check under write lock
	if l, ok := pp.limiters[tenantID]; ok {
		return l
	}

	// Create new limiter: MaxEPS tokens/sec, burst size of MaxEPS/2
	l = rate.NewLimiter(rate.Limit(pp.maxEPS), pp.maxEPS/2)
	pp.limiters[tenantID] = l
	return l
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

	// Inline FNV-1a to avoid fnv.New32a allocation and []byte(key) conversion
	var h uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	
	shardIdx := h % uint32(PartitionCount)
	return pp.shards[shardIdx]
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

// GetMetricsCollector returns the centralized metrics collector.
func (pp *PartitionedPipeline) GetMetricsCollector() *monitoring.MetricsCollector {
	return pp.metricsCollector
}

func getEnvInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	var val int
	fmt.Sscanf(valStr, "%d", &val)
	if val == 0 {
		return defaultVal
	}
	return val
}
