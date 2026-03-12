package services

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

func (s *TemporalService) Name() string { return "temporal-service" }

// Dependencies returns service dependencies
func (s *TemporalService) Dependencies() []string {
	return []string{}
}

func (s *TemporalService) Start(ctx context.Context) error {
	s.log.Info("[TemporalService] Started")
	return nil
}

func (s *TemporalService) Stop(ctx context.Context) error {
	s.log.Info("[TemporalService] Stopped")
	return nil
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
