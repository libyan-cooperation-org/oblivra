package services

import (
	"context"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/monitoring"
)

// DiagnosticsService is the Wails-bound wrapper around monitoring.DiagnosticsService.
// It exposes GetSnapshot() for direct frontend calls and manages the
// periodic broadcast lifecycle.
type DiagnosticsService struct {
	BaseService
	ctx   context.Context
	inner *monitoring.DiagnosticsService
	bus   *eventbus.Bus
	log   *logger.Logger
}

func (s *DiagnosticsService) Name() string         { return "diagnostics-service" }
func (s *DiagnosticsService) Dependencies() []string { return []string{} }

// NewDiagnosticsService creates the service.
// busDropped is a callback returning the event-bus cumulative dropped-event count.
func NewDiagnosticsService(bus *eventbus.Bus, log *logger.Logger, busDropped func() uint64) *DiagnosticsService {
	inner := monitoring.NewDiagnosticsService(log, bus, busDropped)
	return &DiagnosticsService{
		inner: inner,
		bus:   bus,
		log:   log.WithPrefix("diagnostics"),
	}
}

// Start begins the 2-second broadcast loop.
func (s *DiagnosticsService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.inner.StartPeriodicBroadcast(ctx, 2*time.Second)
	s.log.Info("DiagnosticsService started — broadcasting diagnostics:snapshot every 2s")
	return nil
}

// Stop is a no-op — the broadcast goroutine stops when ctx is cancelled by the Kernel.
func (s *DiagnosticsService) Stop(_ context.Context) error { return nil }

// GetSnapshot returns the current full diagnostics state for the frontend.
// Called directly by the Diagnostics Modal via its Wails binding.
func (s *DiagnosticsService) GetSnapshot() monitoring.DiagnosticsSnapshot {
	return s.inner.Snapshot()
}

// UpdateIngestMetrics is called by the ingest metricCollector goroutine every second
// to keep the Diagnostics service current with live pipeline throughput.
func (s *DiagnosticsService) UpdateIngestMetrics(eps, target, dropped int64, bufFill float64, workers int) {
	s.inner.SetIngestMetrics(eps, target, dropped, bufFill, workers)
}

// UpdateLoadStatus is called by the adaptive controller during workload transitions.
// It maps internal LoadStatus to user-facing health events.
func (s *DiagnosticsService) UpdateLoadStatus(status ingest.LoadStatus, message string) {
	statusStr := "healthy"
	switch status {
	case ingest.LoadDegraded:
		statusStr = "degraded"
	case ingest.LoadCritical:
		statusStr = "critical"
	}

	if s.bus != nil {
		s.bus.Publish("health_status_changed", map[string]string{
			"status":  statusStr,
			"message": message,
		})
	}
}

// RecordQueryLatency records a single database query duration (milliseconds) for
// P99 and average tracking. Wire this into every DuckDB/SQLite query path.
func (s *DiagnosticsService) RecordQueryLatency(ms float64) {
	s.inner.RecordQueryLatency(ms)
}
