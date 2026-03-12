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
	buffer    chan *SovereignEvent
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
		buffer:    make(chan *SovereignEvent, bufferSize),
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

	return p
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
func (p *Pipeline) QueueEvent(evt *SovereignEvent) error {
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
func (p *Pipeline) worker() {
	p.wg.Add(1)
	defer p.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			p.log.Error("[INGEST] Worker panicked: %v. Restarting worker...", r)
			select {
			case <-p.ctx.Done():
				// Don't restart if shutting down
			default:
				go p.worker() // Restart the worker
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

func (p *Pipeline) processEvent(ctx context.Context, evt *SovereignEvent) {
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

	// 2. Sandboxing: Execute WASM plugins if available
	if p.wasm != nil {
		dropped := false
		wasmCtx := wasm.WithEvent(ctx, evt.RawLine, &dropped)
		
		if err := p.wasm.ExecuteAll(wasmCtx); err != nil {
			p.log.Error("[INGEST] WASM plugin execution error: %v", err)
		}
		
		if dropped {
			p.metrics.DroppedEvents.Add(1)
			span.AddEvent("event_dropped_by_wasm")
			span.SetAttributes(attribute.Bool("dropped", true))
			span.End()
			return
		}
	}
	span.End()

	p.indexEvent(ctx, evt)
}

func (p *Pipeline) indexEvent(ctx context.Context, evt *SovereignEvent) {
	ctx, span := tracer.Start(ctx, "pipeline.indexEvent")
	defer span.End()

	// 2. Routing: Is this a security anomaly or just a standard log?
	if isSecurityAnomaly(evt) {
		// Route to fast Badger HotStore
		if p.siem != nil {
			hostEvent := &database.HostEvent{
				HostID:    evt.Host,
				Timestamp: evt.Timestamp,
				EventType: evt.EventType,
				SourceIP:  evt.SourceIp,
				User:      evt.User,
				RawLog:    evt.RawLine,
			}
			// Inject context with timeout for SIEM insertion
			insertCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel() // Ensure the context is cancelled to release resources

			if err := p.siem.InsertHostEvent(insertCtx, hostEvent); err != nil {
				p.log.Error("[INGEST] Failed to index SIEM event: %v", err)
			} else if p.bus != nil {
				// Broadcast the event to the alerting engine for real-time YAML rule matching
				p.bus.Publish("siem.event_indexed", *hostEvent)
			}
		}
	} else {
		// Standard terminal / system log -> AnalyticsEngine (SQLite + Bleve Dual-Write)
		if p.analytics != nil {
			// Ingest handles the batching automatically
			sessionID := evt.SessionId
			if sessionID == "" {
				sessionID = "syslog"
			}
			p.analytics.Ingest(sessionID, evt.Host, evt.RawLine)
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
func isSecurityAnomaly(evt *SovereignEvent) bool {
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
