package ingest

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivrashell/internal/events"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/storage"
	"github.com/kingknull/oblivrashell/internal/temporal"
	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/engine/wasm"
	"github.com/kingknull/oblivrashell/internal/engine/dag"
	"github.com/kingknull/oblivrashell/internal/integrity"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer        = otel.Tracer("oblivrashell/ingest")
	ErrBufferFull = errors.New("pipeline buffer full")
)

// LoadStatus represents the operational health of the pipeline.
type LoadStatus int32

const (
	LoadHealthy  LoadStatus = iota // Normal operation
	LoadDegraded                   // High EPS or near-full buffers
	LoadCritical                   // Extreme EPS or critically full buffers
)

// IngestionPipeline defines the contract for any event ingestion implementation (single or sharded).
type IngestionPipeline interface {
	Start()
	Stop()
	Shutdown()
	QueueEvent(evt *events.SovereignEvent) error
	GetMetrics() MetricsSnapshot
	GetLoadStatus() LoadStatus
	SetDiagnosticsUpdater(d DiagnosticsUpdater)
	GetMetricsCollector() *monitoring.MetricsCollector
	SetEvaluator(e *detection.Evaluator)
	SetIdentityResolver(r dag.UserResolver)
	Bus() *eventbus.Bus
	Replay(ctx context.Context) error
	SetGraphEngine(g *graph.GraphEngine)
	SetIntegrityTree(t *integrity.MerkleTree)
}

// DiagnosticsUpdater is satisfied by services.DiagnosticsService.
type DiagnosticsUpdater interface {
	UpdateIngestMetrics(eps, target, dropped int64, bufFill float64, workers int)
	UpdateLoadStatus(status LoadStatus, message string)
}

// Pipeline manages the buffering, crash-protection, and routing of incoming log streams.
type Pipeline struct {
	buffer      chan *events.SovereignEvent
	wal         *storage.WAL
	analytics   *analytics.AnalyticsEngine
	siem        database.SIEMStore
	bus         *eventbus.Bus
	log         *logger.Logger
	temporal    *temporal.IntegrityService
	graphEngine *graph.GraphEngine
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	metrics     Metrics
	once        sync.Once
	wasm        *wasm.PluginManager
	dag         *dag.Engine
	diagnostics   DiagnosticsUpdater // optional; set after container init
	loadStatus    atomic.Int32
	quota            *TenantQuotaManager
	metricsCollector *monitoring.MetricsCollector
	lastProcessed    atomic.Int64 // Unix timestamp of last processed event
	labels           map[string]string
	evaluator        *detection.Evaluator
	identityResolver dag.UserResolver
	integrityTree    *integrity.MerkleTree
	eventChain       *EventChain // tamper-evident hash chain (#12)
}

func (p *Pipeline) SetDiagnosticsUpdater(d DiagnosticsUpdater) { p.diagnostics = d }

func (p *Pipeline) SetGraphEngine(g *graph.GraphEngine) {
	p.graphEngine = g
	p.dag = p.buildProductionDAG()
}

func (p *Pipeline) SetIntegrityTree(t *integrity.MerkleTree) {
	p.integrityTree = t
}

// SetLoadStatus updates the current operational state of the pipeline.
func (p *Pipeline) SetLoadStatus(status LoadStatus) {
	p.loadStatus.Store(int32(status))
}

// GetLoadStatus returns the current operational state of the pipeline.
func (p *Pipeline) GetLoadStatus() LoadStatus {
	return LoadStatus(p.loadStatus.Load())
}

// GetMetricsCollector returns the centralized metrics collector.
func (p *Pipeline) GetMetricsCollector() *monitoring.MetricsCollector {
	return p.metricsCollector
}

// Metrics tracks the real-time throughput of the pipeline
type Metrics struct {
	EventsPerSecond atomic.Int64
	TotalProcessed  atomic.Int64
	BufferUsage     int
	BufferCapacity  int
	DroppedEvents   atomic.Int64
}

// NewPipeline creates a new high-throughput event pipeline.
func NewPipeline(bufferSize int, wal *storage.WAL, ae *analytics.AnalyticsEngine, siem database.SIEMStore, bus *eventbus.Bus, log *logger.Logger, temporal *temporal.IntegrityService, mc *monitoring.MetricsCollector, labels map[string]string, evaluator *detection.Evaluator) *Pipeline {
ctx, cancel := context.WithCancel(context.Background())
p := &Pipeline{
buffer:    make(chan *events.SovereignEvent, bufferSize),
wal:       wal,
analytics: ae,
siem:      siem,
bus:       bus,
log:       log,
temporal:  temporal,
ctx:       ctx,
cancel:    cancel,
quota:       NewTenantQuotaManager(bufferSize, 0.70, mc), // Standard 70% max fair-share
eventChain:  NewEventChain(nil), // nil = in-memory only; wire HotStore via SetHotStore
metricsCollector: mc,
labels:           labels,
evaluator:        evaluator,
}
p.lastProcessed.Store(time.Now().Unix())
p.metrics.BufferCapacity = bufferSize

wasmPM, err := wasm.NewPluginManager(ctx)
if err != nil {
log.Error("[INGEST] Failed to initialize WASM engine: %v", err)
} else {
p.wasm = wasmPM
}
p.dag = p.buildProductionDAG()
return p
}

// SetEvaluator injects a rule engine for shard-local detection.
func (p *Pipeline) SetEvaluator(e *detection.Evaluator) {
p.evaluator = e
// Rebuild DAG to ensure the DetectionNode is active
p.dag = p.buildProductionDAG()
}

// SetIdentityResolver injects the identity service for event enrichment.
func (p *Pipeline) SetIdentityResolver(r dag.UserResolver) {
p.identityResolver = r
// Rebuild DAG to ensure the IdentityEnrichmentNode is active
p.dag = p.buildProductionDAG()
}

func (p *Pipeline) buildProductionDAG() *dag.Engine {
wasmNode := &dag.Node{Processor: dag.NewWASMFilterNode(p.wasm, p.log)}

// Identity Enrichment — Early enrichment for downstream branches
var lastNode = wasmNode
if p.identityResolver != nil {
idNode := &dag.Node{Processor: dag.NewIdentityEnrichmentNode(p.identityResolver, p.log)}
wasmNode.Children = append(wasmNode.Children, idNode)
lastNode = idNode
}

var graphNode *dag.Node
if p.graphEngine != nil {
graphNode = &dag.Node{Processor: dag.NewGraphNode(p.graphEngine, p.log)}
lastNode.Children = append(lastNode.Children, graphNode)
lastNode = graphNode
}

fanoutNode := &dag.Node{Processor: dag.NewMultiDestinationNode("Ingest_Fanout")}
lastNode.Children = append(lastNode.Children, fanoutNode)

// Shard-Local Detection Branch
if p.evaluator != nil {
detNode := &dag.Node{Processor: dag.NewDetectionNode(p.evaluator, p.bus, p.log)}
fanoutNode.Children = append(fanoutNode.Children, detNode)
}

	siemCond := &dag.Node{Processor: dag.NewConditionNode("Is_Security_Anomaly", isSecurityAnomaly)}
	if p.siem != nil {
		siemDest := &dag.Node{Processor: dag.NewSIEMNode(p.siem, p.bus, p.log)}
		siemCond.Children = append(siemCond.Children, siemDest)
	}

	analyticsCond := &dag.Node{Processor: dag.NewConditionNode("Is_Not_Security_Anomaly", func(evt *dag.Event) bool {
		return !isSecurityAnomaly(evt)
	})}
	if p.analytics != nil {
		analyticsDest := &dag.Node{Processor: dag.NewAnalyticsNode(p.analytics)}
		analyticsCond.Children = append(analyticsCond.Children, analyticsDest)
	}
	fanoutNode.Children = append(fanoutNode.Children, siemCond, analyticsCond)

	return dag.NewEngine(wasmNode)
}

func (p *Pipeline) Start() {
	_, span := tracer.Start(p.ctx, "pipeline.Start")
	defer span.End()

	p.log.Info("[INGEST] Starting high-throughput pipeline (Buffer: %d, EPS target: %d)", p.metrics.BufferCapacity, EPSTarget)

	if err := p.Replay(p.ctx); err != nil {
		p.log.Error("[INGEST] WAL Replay failed: %v", err)
	}

	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	p.log.Info("[INGEST] Spawning %d base workers + adaptive controller (max %d workers)", numWorkers, numWorkers*4)

	p.wg.Add(1)
	go p.metricCollector()

	for i := 0; i < numWorkers; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	ac := NewAdaptiveController(p)
	ac.Start()
	go func() {
		<-p.ctx.Done()
		ac.Stop()
	}()
}

func (p *Pipeline) Shutdown() {
	p.once.Do(func() {
		p.log.Info("[INGEST] Shutting down pipeline, draining %d buffered events...", len(p.buffer))
		p.cancel()
		close(p.buffer)
		p.wg.Wait()

		if p.wal != nil {
			if err := p.wal.Close(); err != nil {
				p.log.Error("[INGEST] Failed to close WAL: %v", err)
			}
		}
		if p.wasm != nil {
			if err := p.wasm.Close(context.Background()); err != nil {
				p.log.Error("[INGEST] Failed to close WASM engine: %v", err)
			}
		}
		p.log.Info("[INGEST] Pipeline shutdown complete")
	})
}

func (p *Pipeline) Stop() { p.Shutdown() }

// QueueEvent validates and enqueues a SovereignEvent.

// Validation stages (in order, cheapest first):
//   1. Size limit — rejects oversized payloads before any allocation
//   2. Null-byte / injection detection — rejects log-injection attempts
//   3. UTF-8 validity — rejects malformed encodings
//   4. Field sanitization — truncates unbounded fields before they grow
//
// Any rejection is counted in DroppedEvents and logged at WARN level so that
// the operator can see it in DiagnosticsService / metrics without log spam.
func (p *Pipeline) QueueEvent(evt *events.SovereignEvent) error {
// ── Input validation ──────────────────────────────────────────────────────
if evt.RawLine != "" {
if err := ValidateRawLine(evt.RawLine); err != nil {
p.metrics.DroppedEvents.Add(1)
p.log.Warn("[INGEST] Rejected event from host=%q tenant=%q: %v (raw_len=%d)",
evt.Host, evt.TenantID, err, len(evt.RawLine))
return err
}

// Soft-sanitize individual fields — truncate don't reject, as parsers
		// may have produced a legitimately long CommandLine from a real log.
		if len(evt.User) > MaxFieldValueBytes {
			evt.User, _ = SanitizeFieldValue(evt.User)
		}
		if len(evt.Host) > MaxFieldValueBytes {
			evt.Host, _ = SanitizeFieldValue(evt.Host)
		}
		if len(evt.Metadata) > MaxMetadataKeys {
			evt.Metadata = SanitizeMetadata(evt.Metadata)
		}

	}

	select {
	case p.buffer <- evt:
		// 2. Increment occupancy for the tenant
		p.quota.Inc(evt.TenantID)
		return nil
	default:
		p.metrics.DroppedEvents.Add(1)
		return ErrBufferFull
	}
}

type MetricsSnapshot struct {
	EventsPerSecond int64     `json:"events_per_second"`
	TotalProcessed  int64     `json:"total_processed"`
	BufferUsage     int       `json:"buffer_usage"`
	BufferCapacity  int       `json:"buffer_capacity"`
	DroppedEvents   int64     `json:"dropped_events"`
	CollectedAt     time.Time `json:"collected_at"`
}

func (p *Pipeline) GetMetrics() MetricsSnapshot {
	return MetricsSnapshot{
		EventsPerSecond: p.metrics.EventsPerSecond.Load(),
		TotalProcessed:  p.metrics.TotalProcessed.Load(),
		BufferUsage:     len(p.buffer),
		BufferCapacity:  p.metrics.BufferCapacity,
		DroppedEvents:   p.metrics.DroppedEvents.Load(),
		CollectedAt:     time.Now().UTC(),
	}
}

func (p *Pipeline) worker() {
	defer p.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			p.log.Error("[INGEST] Worker panicked: %v. Restarting worker...", r)
			select {
			case <-p.ctx.Done():
			default:
				p.wg.Add(1)
				go p.worker()
			}
		}
	}()

	for {
		select {
		case evt, ok := <-p.buffer:
			if !ok {
				return
			}

			// 3. Decrement occupancy once processed (or pulled from chan)
			p.quota.Dec(evt.TenantID)

			// 4. Update heartbeat for watchdog
			p.lastProcessed.Store(time.Now().Unix())

			ctx := evt.Ctx
			if ctx == nil {
				ctx = context.Background()
			}
			ctx, span := tracer.Start(ctx, "pipeline.processEvent", trace.WithAttributes(
				attribute.String("event.type", evt.EventType),
				attribute.String("event.host", evt.Host),
			))
			p.processEvent(ctx, evt)
			span.End()

			total := p.metrics.TotalProcessed.Add(1)
			if total > 0 && total%5000 == 0 && p.wal != nil {
				p.log.Debug("[INGEST] Routine WAL checkpoint at %d events", total)
				if err := p.wal.Checkpoint(); err != nil {
					p.log.Error("[INGEST] Routine WAL checkpoint failed: %v", err)
				}
			}
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Pipeline) processEvent(ctx context.Context, evt *events.SovereignEvent) {
	_, span := tracer.Start(ctx, "pipeline.WALWrite")
	rawBytes := []byte(evt.RawLine)

	// Stamp the tamper-evident hash chain BEFORE WAL — chain proves ordering (#12)
	if p.eventChain != nil {
		p.eventChain.Seal(evt)
	}

	if p.wal != nil {
		if err := p.wal.Append(rawBytes); err != nil {
			p.log.Error("[INGEST] WAL write failure: %v", err)
		}
	}
	if p.integrityTree != nil {
		hash, idx, err := p.integrityTree.AddLeaf(rawBytes)
		if err != nil {
			p.log.Error("[INGEST] Integrity hashing failed: %v", err)
		} else {
			// Only override if MerkleTree hash wins over chain hash
			if evt.IntegrityHash == "" {
				evt.IntegrityHash = hash
				evt.IntegrityIndex = int32(idx)
			}
		}
	}
	if p.temporal != nil {
		p.temporal.ValidateTimestamp(evt.Host, parseTime(evt.Timestamp))
	}
	span.End()
	p.indexEvent(ctx, evt)
}

// SetEventChain attaches a persistent-backed EventChain to the pipeline.
func (p *Pipeline) SetEventChain(ec *EventChain) {
	p.eventChain = ec
}

func (p *Pipeline) indexEvent(ctx context.Context, evt *events.SovereignEvent) {
	ctx, span := tracer.Start(ctx, "pipeline.dagExecution")
	defer span.End()
	if p.dag != nil {
		if err := p.dag.Execute(ctx, evt); err != nil {
			p.log.Error("[INGEST] DAG execution failure: %v", err)
		}
	}
}

func (p *Pipeline) Replay(ctx context.Context) error {
	if p.wal == nil {
		return nil
	}
	err := p.wal.Replay(func(payload []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		raw := string(payload)
		// Validate replayed events too — WAL may contain pre-validation entries from older versions.
		if err := ValidateRawLine(raw); err != nil {
			p.log.Warn("[INGEST] Skipping invalid WAL entry: %v", err)
			return nil
		}
		pCtx := events.EventProcessingContext{
			EventID:  fmt.Sprintf("evt-wal-%d", time.Now().UnixNano()),
			TenantID: "GLOBAL",
			Now:      time.Now().UTC(),
		}
		evt := AutoParse(raw, pCtx)
		p.indexEvent(ctx, evt)
		return nil
	})
	if err != nil {
		return err
	}
	return p.wal.Checkpoint()
}

func (p *Pipeline) metricCollector() {
	defer func() {
		if r := recover(); r != nil {
			p.log.Error("[INGEST] Panic in metricCollector: %v", r)
		}
	}()
	defer p.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastProcessed int64
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			current := p.metrics.TotalProcessed.Load()
			eps := current - lastProcessed
			p.metrics.EventsPerSecond.Store(eps)
			lastProcessed = current

			if p.diagnostics != nil {
				bufCap := p.metrics.BufferCapacity
				bufFill := 0.0
				if bufCap > 0 {
					bufFill = float64(len(p.buffer)) / float64(bufCap) * 100
				}
				p.diagnostics.UpdateIngestMetrics(
					eps,
					int64(EPSTarget),
					p.metrics.DroppedEvents.Load(),
					bufFill,
					runtime.NumCPU(),
				)
			}
		}
	}
}

func (p *Pipeline) Bus() *eventbus.Bus { return p.bus }

func isSecurityAnomaly(evt *events.SovereignEvent) bool {
	if strings.HasPrefix(evt.EventType, "windows_") ||
		strings.HasPrefix(evt.EventType, "linux_") ||
		strings.HasPrefix(evt.EventType, "aws_") ||
		strings.HasPrefix(evt.EventType, "azure_") ||
		strings.HasPrefix(evt.EventType, "network_") {
		return true
	}
	switch evt.EventType {
	case "failed_login", "sudo_exec", "cef", "security_alert", "successful_login":
		return true
	default:
		return false
	}
}
