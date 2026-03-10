package app

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/temporal"
)

// TemporalService exposes temporal integrity capabilities to the frontend via Wails.
type TemporalService struct {
	engine *temporal.IntegrityService
	bus    *eventbus.Bus
	log    *logger.Logger
}

func NewTemporalService(engine *temporal.IntegrityService, bus *eventbus.Bus, log *logger.Logger) *TemporalService {
	return &TemporalService{
		engine: engine,
		bus:    bus,
		log:    log,
	}
}

func (s *TemporalService) Name() string { return "TemporalService" }

func (s *TemporalService) Startup(ctx context.Context) {
	s.log.Info("[TemporalService] Started")
}

func (s *TemporalService) Shutdown() {
	s.log.Info("[TemporalService] Stopped")
}

// GetViolations returns all recorded temporal violations.
func (s *TemporalService) GetViolations() []temporal.Violation {
	return s.engine.GetViolations()
}

// GetAgentDrift returns the latest clock drift readings per agent.
func (s *TemporalService) GetAgentDrift() map[string]int64 {
	return s.engine.GetAgentDrift()
}

// GetFleetDriftReport returns statistical analysis of fleet-wide clock drift.
func (s *TemporalService) GetFleetDriftReport() *temporal.FleetDriftReport {
	return s.engine.DetectFleetDrift()
}

// CheckSequenceManipulation scans events for timestamp inversions.
func (s *TemporalService) CheckSequenceManipulation(events []temporal.TimestampedEvent) []temporal.SequenceAnomaly {
	return s.engine.DetectSequenceManipulation(events)
}
