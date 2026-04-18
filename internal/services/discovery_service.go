package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/discovery"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// DiscoveryService exposes host discovery functionality to the frontend
type DiscoveryService struct {
	BaseService
	ctx              context.Context
	log              *logger.Logger
	discoveryManager *discovery.DiscoveryManager
}

// NewDiscoveryService creates a new DiscoveryService
func (s *DiscoveryService) Name() string { return "discovery-service" }

// Dependencies returns service dependencies
func (s *DiscoveryService) Dependencies() []string {
	return []string{}
}

func NewDiscoveryService(log *logger.Logger, dm *discovery.DiscoveryManager) *DiscoveryService {
	return &DiscoveryService{
		log:              log.WithPrefix("discoveryservice"),
		discoveryManager: dm,
	}
}

// Startup is called at application startup
func (s *DiscoveryService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("Discovery Service started")
	return nil
}

func (s *DiscoveryService) Stop(ctx context.Context) error {
	return nil
}

// DiscoverAll runs all sources to find local networks hosts and configuration hosts
func (s *DiscoveryService) DiscoverAll() ([]discovery.DiscoveredHost, error) {
	s.log.Info("Starting local host discovery")
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	hosts, err := s.discoveryManager.DiscoverAll(ctx)
	if err != nil {
		s.log.Error("Discovery failed: %v", err)
		return nil, err
	}
	s.log.Info("Discovered %d hosts", len(hosts))
	return hosts, nil
}
