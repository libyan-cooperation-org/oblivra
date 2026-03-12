package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/lineage"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// LineageService exposes data provenance tracking to the frontend via Wails.
type LineageService struct {
	engine *lineage.LineageEngine
	bus    *eventbus.Bus
	log    *logger.Logger
}

func NewLineageService(engine *lineage.LineageEngine, bus *eventbus.Bus, log *logger.Logger) *LineageService {
	return &LineageService{
		engine: engine,
		bus:    bus,
		log:    log,
	}
}

func (s *LineageService) Name() string { return "lineage-service" }

// Dependencies returns service dependencies
func (s *LineageService) Dependencies() []string {
	return []string{}
}

func (s *LineageService) Start(ctx context.Context) error {
	s.log.Info("[LineageService] Started")
	return nil
}

func (s *LineageService) Stop(ctx context.Context) error {
	s.log.Info("[LineageService] Stopped")
	return nil
}

// GetChain returns the full provenance chain for an entity.
func (s *LineageService) GetChain(entityID string) *lineage.LineageChain {
	return s.engine.GetChain(entityID)
}

// GetProvenance returns a single lineage record by ID.
func (s *LineageService) GetProvenance(recordID string) *lineage.LineageRecord {
	return s.engine.GetProvenance(recordID)
}

// GetRecentLineage returns the N most recent lineage records.
func (s *LineageService) GetRecentLineage(limit int) []lineage.LineageRecord {
	return s.engine.GetRecentLineage(limit)
}

// GetStats returns summary statistics for the lineage store.
func (s *LineageService) GetStats() map[string]interface{} {
	return s.engine.Stats()
}
