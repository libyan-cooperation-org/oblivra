package services

import (
	"context"
	"sync"

	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// FusionService bridges the AttackFusionEngine with the Wails frontend.
type FusionService struct {
	engine *detection.AttackFusionEngine
	bus    *eventbus.Bus
	log    *logger.Logger
	mu     sync.RWMutex
}

// NewFusionService creates a new fusion bridge service.
func NewFusionService(engine *detection.AttackFusionEngine, bus *eventbus.Bus, log *logger.Logger) *FusionService {
	return &FusionService{
		engine: engine,
		bus:    bus,
		log:    log,
	}
}

// Name returns the service identifier.
func (s *FusionService) Name() string {
	return "FusionService"
}

// GetActiveCampaigns returns all currently tracked attack campaigns.
func (s *FusionService) GetActiveCampaigns() []detection.Campaign {
	// We need to reach into the engine's LRU. 
	// Since Campaign is usually internal, we'll need an Export method in engine 
	// or just return the slice if engine provides it.
	return s.engine.GetActiveCampaigns()
}

// GetCampaign returns details for a specific entity.
func (s *FusionService) GetCampaign(entityID string) *detection.Campaign {
	return s.engine.GetCampaign(entityID)
}

// Dependencies returns naming dependencies.
func (s *FusionService) Dependencies() []string {
	return []string{}
}

// Start matches the platform.Service interface.
func (s *FusionService) Start(ctx context.Context) error {
	s.log.Info("[FUSION] Attack Fusion Engine started.")
	return nil
}

// Stop matches the platform.Service interface.
func (s *FusionService) Stop(ctx context.Context) error {
	return nil
}
