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

// Pipeline manages the buffering, crash-protection, and routing of incoming log streams.
type Pipeline struct {
	buffer    chan *events.SovereignEvent
	wal       *storage.WAL
	analytics *analytics.AnalyticsEngine
	siem      database.SIEMStore
	bus       *eventbus.Bus
	log       *logger.Logger
	temporal  *temporal.IntegrityService
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	metrics   Metrics
	once      sync.Once
	wasm      *wasm.PluginManager
	dag       *dag.Engine
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
func NewPipeline(bufferSize int, wal *storage.WAL, ae *analytics.AnalyticsEngine, siem database.SIEMStore, bus *eventbus.Bus, log *logger.Logger, temporal *temporal.IntegrityService) *Pipeline {
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
	}

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

func (p *Pipeline) buildProductionDAG() *dag.Engine {
	// Root: WASM Filter
	wasmNode := &dag.Node{Processor: dag.NewWASMFilterNode(p.wasm, p.log)}

	// Branching Node: Fan out to SIEM and Analytics
	fanoutNode := &dag.Node{Processor: dag.NewMultiDestinationNode("Ingest_Fanout")}
	wasmNode.Children = append(wasmNode.Children, fanoutNode)

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

	p.log.Info("[INGEST] Starting high-throughput pipeline (Buffer: %d)", p.metrics.BufferCapacity)

	// synchronous replay before accepting new events
	if err := p.Replay(p.ctx); err != nil {
		p.log.Error("[INGEST] WAL Replay failed: %v", err)
	}

	// Domain 7 Performance: Fan-Out Worker Pool based on available logical CPU cores
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}

	p.log.Info("[INGEST] Spawning %d parallel parsing workers...", numWorkers)

	p.wg.Add(1) // for the metric collector
	go p.metricCollector()

	for i := 0; i < numWorkers; i++ {
		// wg.Add MUST be called before go, not inside the goroutine body,
		// otherwise Shutdown()'s wg.Wait() can return before the worker starts.
		p.wg.Add(1)
		go p.worker()
	}
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

// QueueEvent pushes a SovereignEvent into the pipeline. Applies backpressure if full.
func (p *Pipeline) QueueEvent(evt *events.SovereignEvent) error {
	select {
	case p.buffer <- evt:
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

// metricCollector updates the EPS (Events Per Second) speedometer
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
			p.metrics.EventsPerSecond.Store(current - lastProcessed)
			lastProcessed = current
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
