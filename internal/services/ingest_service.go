package services

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

// IngestService provides frontend controls for the syslog server and ingestion pipeline
type IngestService struct {
	BaseService
	pipeline ingest.IngestionPipeline
	server   *ingest.SyslogServer
	agentSrv *ingest.AgentServer
	bus      *eventbus.Bus
	log      *logger.Logger
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

func (s *IngestService) Name() string { return "ingest-service" }

// Dependencies returns service dependencies.
func (s *IngestService) Dependencies() []string {
	return []string{}
}

// NewIngestService injects the ingestion dependencies
func NewIngestService(p ingest.IngestionPipeline, srv *ingest.SyslogServer, agentSrv *ingest.AgentServer, bus *eventbus.Bus, log *logger.Logger) *IngestService {
	return &IngestService{
		pipeline: p,
		server:   srv,
		agentSrv: agentSrv,
		bus:      bus,
		log:      log.WithPrefix("ingest_service"),
		limiters: make(map[string]*rate.Limiter),
	}
}

// Pipeline returns the underlying ingest.IngestionPipeline for cross-service wiring.
func (s *IngestService) Pipeline() ingest.IngestionPipeline {
	return s.pipeline
}

// AgentServer returns the internal agent ingest server for command dispatch.
func (s *IngestService) AgentServer() *ingest.AgentServer {
	return s.agentSrv
}

// Start initializes background workers (the pipeline processes the queue)
func (s *IngestService) Start(ctx context.Context) error {
	s.log.Info("Starting ingestion pipeline workers...")
	if s.pipeline != nil {
		s.pipeline.Start()
	}

	// Auto-start ingestion servers on boot (Sovereign Mode)
	if err := s.StartSyslogServer(); err != nil {
		s.log.Error("Failed to auto-start ingestion servers: %v", err)
	}

	// EMERGENCY LISTENERS
	s.bus.Subscribe(eventbus.EventType("disaster:killswitch"), func(event eventbus.Event) {
		s.log.Warn("🚨 IngestService: Emergency Kill-Switch received. Stopping external ingestion.")
		s.StopSyslogServer()
	})

	s.bus.Subscribe(eventbus.EventType("disaster:nuclear"), func(event eventbus.Event) {
		s.log.Warn("☢️ IngestService: Nuclear Destruction received. Homing pipeline.")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.Stop(ctx)
	})
	return nil
}

// Stop gracefully stops the listeners and flushes the pipeline buffers
func (s *IngestService) Stop(ctx context.Context) error {
	s.log.Info("Shutting down ingestion services...")
	if s.server != nil {
		s.server.Stop()
	}
	if s.agentSrv != nil {
		s.agentSrv.Stop()
	}
	if s.pipeline != nil {
		s.pipeline.Stop()
	}
	return nil
}

// StartSyslogServer binds the UDP/TCP listener to begin receiving external logs
func (s *IngestService) StartSyslogServer() error {
	var errs []error

	if s.server != nil {
		if err := s.server.Start(); err != nil {
			s.log.Error("Failed to start syslog server: %v", err)
			errs = append(errs, err)
		} else {
			s.log.Info("Syslog server started successfully on port 1514")
		}
	}

	if s.agentSrv != nil {
		if err := s.agentSrv.Start(); err != nil {
			s.log.Error("Failed to start agent ingest server: %v", err)
			errs = append(errs, err)
		} else {
			s.log.Info("Agent ingest server started successfully on port 8443")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("one or more ingestion servers failed to start: %v", errs)
	}

	return nil
}

// StopSyslogServer halts external ingestion but leaves the pipeline running
func (s *IngestService) StopSyslogServer() error {
	if s.server == nil {
		return fmt.Errorf("syslog server not configured")
	}

	s.server.Stop()
	if s.agentSrv != nil {
		s.agentSrv.Stop()
	}
	return nil
}

// GetMetrics returns the current throughput and drops of the pipeline
func (s *IngestService) GetMetrics() map[string]interface{} {
	if s.pipeline == nil {
		return map[string]interface{}{
			"events_per_second": 0,
			"total_processed":   0,
			"buffer_usage":      0,
			"buffer_capacity":   0,
			"dropped_events":    0,
		}
	}

	m := s.pipeline.GetMetrics()
	return map[string]interface{}{
		"events_per_second": m.EventsPerSecond,
		"total_processed":   m.TotalProcessed,
		"buffer_usage":      m.BufferUsage,
		"buffer_capacity":   m.BufferCapacity,
		"dropped_events":    m.DroppedEvents,
	}
}

// QueueEvent submits an event directly into the ingestion pipeline
func (s *IngestService) QueueEvent(evt *events.SovereignEvent) error {
	if s.pipeline == nil {
		return fmt.Errorf("pipeline not initialized")
	}

	// ── Circuit Breaker: Rate Limiting ───────────────────────────────────────
	limiter := s.getLimiter(evt.TenantID)
	if !limiter.Allow() {
		s.log.Warn("[CIRCUIT_BREAKER] Rate limit exceeded for tenant %s. Dropping event.", evt.TenantID)
		return fmt.Errorf("tenant rate limit exceeded")
	}

	return s.pipeline.QueueEvent(evt)
}

func (s *IngestService) getLimiter(tenantID string) *rate.Limiter {
	s.mu.RLock()
	l, ok := s.limiters[tenantID]
	s.mu.RUnlock()

	if ok {
		return l
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double check
	if l, ok = s.limiters[tenantID]; ok {
		return l
	}

	// Sovereign Default: 5000 EPS per tenant burstable to 10000
	l = rate.NewLimiter(rate.Limit(5000), 10000)
	s.limiters[tenantID] = l
	return l
}

// Health reports the current operational state of the ingestion pipeline to the platform.
func (s *IngestService) Health(ctx context.Context) error {
	if s.pipeline == nil {
		return fmt.Errorf("pipeline inactive")
	}

	status := s.pipeline.GetLoadStatus()
	if status == ingest.LoadDegraded {
		metrics := s.pipeline.GetMetrics()
		return fmt.Errorf("ingestion pipeline is DEGRADED (EPS: %d, Drops: %d)", 
			metrics.EventsPerSecond, metrics.DroppedEvents)
	}
	
	if status == ingest.LoadCritical {
		return fmt.Errorf("ingestion pipeline is CRITICAL (buffer saturation)")
	}

	return nil
}
