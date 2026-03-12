package ingest

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/storage"
	"github.com/kingknull/oblivrashell/internal/temporal"
)

// Pipeline manages the buffering, crash-protection, and routing of incoming log streams.
type Pipeline struct {
	buffer    chan ParsedEvent
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
	mu        sync.RWMutex
}

// Metrics tracks the real-time throughput of the pipeline
type Metrics struct {
	EventsPerSecond int64
	TotalProcessed  int64
	BufferUsage     int
	BufferCapacity  int
	DroppedEvents   int64
}

// NewPipeline creates a new high-throughput event pipeline.
func NewPipeline(bufferSize int, wal *storage.WAL, ae *analytics.AnalyticsEngine, siem database.SIEMStore, bus *eventbus.Bus, log *logger.Logger, temporal *temporal.IntegrityService) *Pipeline {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pipeline{
		buffer:    make(chan ParsedEvent, bufferSize),
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
	return p
}

// Start begins processing the buffered queue in the background.
func (p *Pipeline) Start() {
	p.log.Info("[INGEST] Starting high-throughput pipeline (Buffer: %d)", p.metrics.BufferCapacity)

	// synchronous replay before accepting new events
	if err := p.Replay(); err != nil {
		p.log.Error("[INGEST] WAL Replay failed: %v", err)
	}

	// Domain 7 Performance: Fan-Out Worker Pool based on available logical CPU cores
	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}

	p.log.Info("[INGEST] Spawning %d parallel parsing workers...", numWorkers)

	p.wg.Add(numWorkers + 1) // +1 for the metric collector
	for i := 0; i < numWorkers; i++ {
		go p.worker()
	}

	go p.metricCollector()
}

// Shutdown stops the pipeline and waits for all workers to drain the buffer.
func (p *Pipeline) Shutdown() {
	p.log.Info("[INGEST] Shutting down pipeline, draining %d buffered events...", len(p.buffer))
	p.cancel()
	close(p.buffer)
	p.wg.Wait()

	if p.wal != nil {
		if err := p.wal.Close(); err != nil {
			p.log.Error("[INGEST] Failed to close WAL: %v", err)
		}
	}
	p.log.Info("[INGEST] Pipeline shutdown complete")
}

// Stop flushes remaining events and shuts down the pipeline.
func (p *Pipeline) Stop() {
	p.log.Info("[INGEST] Stopping pipeline...")
	p.cancel()
	p.wg.Wait()
	if p.wal != nil {
		p.wal.Close()
	}
}

// QueueEvent pushes a ParsedEvent into the pipeline. Applies backpressure if full.
func (p *Pipeline) QueueEvent(evt ParsedEvent) {
	select {
	case p.buffer <- evt:
		// Success
	default:
		// Buffer is full. Drop event to prevent blocking the listener (syslog usually expects drops on overload).
		p.mu.Lock()
		p.metrics.DroppedEvents++
		p.mu.Unlock()
	}
}

// GetMetrics returns a snapshot of current ingestion throughput
func (p *Pipeline) GetMetrics() Metrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	m := p.metrics
	m.BufferUsage = len(p.buffer)
	return m
}

// worker reads from the channel, writes to the WAL, and routes to storage tiers.
func (p *Pipeline) worker() {
	defer p.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			p.log.Error("[INGEST] Worker panicked: %v. Restarting worker...", r)
			go p.worker() // Restart the worker
		}
	}()

	for {
		select {
		case evt, ok := <-p.buffer:
			if !ok {
				// Channel closed and drained
				return
			}
			p.processEvent(evt)

			p.mu.Lock()
			p.metrics.TotalProcessed++
			total := p.metrics.TotalProcessed
			p.mu.Unlock()

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

func (p *Pipeline) processEvent(evt ParsedEvent) {
	// 1. Durability: Write to WAL (Assuming raw string matches what parser used)
	if p.wal != nil {
		if err := p.wal.Append([]byte(evt.RawLine)); err != nil {
			p.log.Error("[INGEST] WAL write failure: %v", err)
		}
	}

	if p.temporal != nil {
		p.temporal.ValidateTimestamp(evt.Host, evt.Timestamp)
	}

	p.indexEvent(evt)
}

func (p *Pipeline) indexEvent(evt ParsedEvent) {
	// 2. Routing: Is this a security anomaly or just a standard log?
	if isSecurityAnomaly(evt) {
		// Route to fast Badger HotStore
		if p.siem != nil {
			hostEvent := &database.HostEvent{
				HostID:    evt.Host,
				Timestamp: evt.Timestamp.Format(time.RFC3339),
				EventType: evt.EventType,
				SourceIP:  evt.SourceIP,
				User:      evt.User,
				RawLog:    evt.RawLine,
			}
			// Inject context with timeout for SIEM insertion
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel() // Ensure the context is cancelled to release resources

			if err := p.siem.InsertHostEvent(ctx, hostEvent); err != nil {
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
			sessionID := evt.SessionID
			if sessionID == "" {
				sessionID = "syslog"
			}
			p.analytics.Ingest(sessionID, evt.Host, evt.RawLine)
		}
	}
}

// Replay reads trapped events from the WAL and indexes them.
func (p *Pipeline) Replay() error {
	if p.wal == nil {
		return nil
	}

	err := p.wal.Replay(func(payload []byte) error {
		raw := string(payload)
		evt := AutoParse(raw)
		p.indexEvent(evt)
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
	defer p.wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastProcessed int64

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.mu.Lock()
			current := p.metrics.TotalProcessed
			p.metrics.EventsPerSecond = current - lastProcessed
			lastProcessed = current
			p.mu.Unlock()
		}
	}
}

// Bus returns the event bus attached to this pipeline.
func (p *Pipeline) Bus() *eventbus.Bus {
	return p.bus
}

// isSecurityAnomaly contains very basic heuristics to route failed logins and sudo actions to the SIEM tier.
func isSecurityAnomaly(evt ParsedEvent) bool {
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
