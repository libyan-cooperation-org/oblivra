package ingest

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/storage"
	"github.com/kingknull/oblivrashell/internal/temporal"
	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/engine/wasm"
	"github.com/kingknull/oblivrashell/internal/engine/dag"
	"github.com/kingknull/oblivrashell/internal/events"
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
}

// DiagnosticsUpdater is satisfied by services.DiagnosticsService.
// Using an interface avoids an import cycle between ingest → services.
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
}

// SetDiagnosticsUpdater wires the diagnostics service into the pipeline.
// Called by the container after both ingest and diagnostics are initialised.
func (p *Pipeline) SetDiagnosticsUpdater(d DiagnosticsUpdater) {
	p.diagnostics = d
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
		quota:     NewTenantQuotaManager(bufferSize, 0.70, mc), // Standard 70% max fair-share
		metricsCollector: mc,
		labels:           labels,
		evaluator:        evaluator,
	}
	p.lastProcessed.Store(time.Now().Unix())
	p.metrics.BufferCapacity = bufferSize

	// Initialize WASM engine
	wasmPM, err := wasm.NewPluginManager(ctx)
	if err != nil {
		log.Error("[INGEST] Failed to initialize WASM engine: %v", err)
	} else {
		p.wasm = wasmPM
	}

	// Initialize Production DAG
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
	// Root: WASM Filter
	wasmNode := &dag.Node{Processor: dag.NewWASMFilterNode(p.wasm, p.log)}

	// Identity Enrichment — Early enrichment for downstream branches
	var lastNode = wasmNode
	if p.identityResolver != nil {
		idNode := &dag.Node{Processor: dag.NewIdentityEnrichmentNode(p.identityResolver, p.log)}
		wasmNode.Children = append(wasmNode.Children, idNode)
		lastNode = idNode
	}

	// Branching Node: Fan out to SIEM, Analytics, and Detection
	fanoutNode := &dag.Node{Processor: dag.NewMultiDestinationNode("Ingest_Fanout")}
	lastNode.Children = append(lastNode.Children, fanoutNode)

	// Shard-Local Detection Branch
	if p.evaluator != nil {
		detNode := &dag.Node{Processor: dag.NewDetectionNode(p.evaluator, p.bus, p.log)}
		fanoutNode.Children = append(fanoutNode.Children, detNode)
	}

	// SIEM Branch: if isSecurityAnomaly
	siemCond := &dag.Node{Processor: dag.NewConditionNode("Is_Security_Anomaly", isSecurityAnomaly)}
	siemDest := &dag.Node{Processor: dag.NewSIEMNode(p.siem, p.bus, p.log)}
	siemCond.Children = append(siemCond.Children, siemDest)

	// Analytics Branch: if NOT isSecurityAnomaly
	analyticsCond := &dag.Node{Processor: dag.NewConditionNode("Is_Not_Security_Anomaly", func(evt *dag.Event) bool {
		return !isSecurityAnomaly(evt)
	})}
	analyticsDest := &dag.Node{Processor: dag.NewAnalyticsNode(p.analytics)}
	analyticsCond.Children = append(analyticsCond.Children, analyticsDest)

	fanoutNode.Children = append(fanoutNode.Children, siemCond, analyticsCond)

	return dag.NewEngine(wasmNode)
}

// Start begins processing the buffered queue in the background.
func (p *Pipeline) Start() {
	_, span := tracer.Start(p.ctx, "pipeline.Start")
	defer span.End()

	p.log.Info("[INGEST] Starting high-throughput pipeline (Buffer: %d, EPS target: %d)", p.metrics.BufferCapacity, EPSTarget)

	// synchronous replay before accepting new events
	if err := p.Replay(p.ctx); err != nil {
		p.log.Error("[INGEST] WAL Replay failed: %v", err)
	}

	// Base worker pool: one worker per logical CPU core
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}

	p.log.Info("[INGEST] Spawning %d base workers + adaptive controller (max %d workers)", numWorkers, numWorkers*4)

	p.wg.Add(1) // for the metric collector
	go p.metricCollector()

	for i := 0; i < numWorkers; i++ {
		// wg.Add MUST be called before go, not inside the goroutine body,
		// otherwise Shutdown()'s wg.Wait() can return before the worker starts.
		p.wg.Add(1)
		go p.worker()
	}

	// Launch the adaptive controller to scale workers and shed load automatically.
	ac := NewAdaptiveController(p)
	ac.Start()
	// Stop the controller when the pipeline shuts down.
	go func() {
		<-p.ctx.Done()
		ac.Stop()
	}()
}

// Shutdown stops the pipeline and waits for all workers to drain the buffer.
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

// Stop flushes remaining events and shuts down the pipeline.
// Deprecated: Use Shutdown() instead.
func (p *Pipeline) Stop() {
	p.Shutdown()
}

// QueueEvent pushes a SovereignEvent into the pipeline. Applies backpressure if full or quota exceeded.
func (p *Pipeline) QueueEvent(evt *events.SovereignEvent) error {
	// 1. Check if the tenant is already monopolising the buffer
	if err := p.quota.CheckQuota(evt.TenantID); err != nil {
		p.metrics.DroppedEvents.Add(1)
		return err
	}

	select {
	case p.buffer <- evt:
		// 2. Increment occupancy for the tenant
		p.quota.Inc(evt.TenantID)
		return nil
	default:
		// Buffer is full. Signal backpressure to the caller.
		p.metrics.DroppedEvents.Add(1)
		return ErrBufferFull
	}
}

// MetricsSnapshot is for frontend binding
type MetricsSnapshot struct {
	EventsPerSecond int64 `json:"events_per_second"`
	TotalProcessed  int64 `json:"total_processed"`
	BufferUsage     int   `json:"buffer_usage"`
	BufferCapacity  int   `json:"buffer_capacity"`
	DroppedEvents   int64 `json:"dropped_events"`
}

// GetMetrics returns a snapshot of current ingestion throughput
func (p *Pipeline) GetMetrics() MetricsSnapshot {
	return MetricsSnapshot{
		EventsPerSecond: p.metrics.EventsPerSecond.Load(),
		TotalProcessed:  p.metrics.TotalProcessed.Load(),
		BufferUsage:     len(p.buffer),
		BufferCapacity:  p.metrics.BufferCapacity,
		DroppedEvents:   p.metrics.DroppedEvents.Load(),
	}
}

// worker reads from the channel, writes to the WAL, and routes to storage tiers.
// NOTE: wg.Add(1) is called by the *caller* before launching this goroutine,
// so that wg.Wait() in Shutdown() cannot return before the worker registers.
func (p *Pipeline) worker() {
	defer p.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			p.log.Error("[INGEST] Worker panicked: %v. Restarting worker...", r)
			select {
			case <-p.ctx.Done():
				// Shutting down — do not restart, let wg drain cleanly.
			default:
				// Add to WaitGroup before spawning to avoid the shutdown race.
				p.wg.Add(1)
				go p.worker()
			}
		}
	}()

	for {
		select {
		case evt, ok := <-p.buffer:
			if !ok {
				// Channel closed and drained
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

			// Periodic WAL Checkpoint every 5,000 events to prevent indefinite log growth
			if total > 0 && total%5000 == 0 {
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
	// 1. Durability: Write to WAL (Assuming raw string matches what parser used)
	if p.wal != nil {
		if err := p.wal.Append([]byte(evt.RawLine)); err != nil {
			p.log.Error("[INGEST] WAL write failure: %v", err)
		}
	}

	if p.temporal != nil {
		p.temporal.ValidateTimestamp(evt.Host, parseTime(evt.Timestamp))
	}

	span.End()

	p.indexEvent(ctx, evt)
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

// Replay reads trapped events from the WAL and indexes them.
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
		evt := AutoParse(raw)
		p.indexEvent(ctx, evt)
		return nil
	})

	if err != nil {
		return err
	}

	// Truncate the WAL after successful replay to avoid double-indexing on next boot
	return p.wal.Checkpoint()
}

// metricCollector updates the EPS (Events Per Second) speedometer and pushes
// live metrics to the DiagnosticsService every tick.
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

			// Feed live stats to DiagnosticsService (if wired)
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

// Bus returns the event bus attached to this pipeline.
func (p *Pipeline) Bus() *eventbus.Bus {
	return p.bus
}

// isSecurityAnomaly contains very basic heuristics to route failed logins and sudo actions to the SIEM tier.
func isSecurityAnomaly(evt *events.SovereignEvent) bool {
	// Let all advanced parsed types flow into the SIEM directly
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
