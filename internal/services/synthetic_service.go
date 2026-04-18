package services

import (
	"context"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/monitoring"
)

// SyntheticService bridges the monitoring engine to the UI
type SyntheticService struct {
	mu      *sync.RWMutex
	manager *monitoring.SyntheticManager
	log     *logger.Logger
	ctx     context.Context
}

func NewSyntheticService(manager *monitoring.SyntheticManager, log *logger.Logger) *SyntheticService {
	return &SyntheticService{
		mu:      &sync.RWMutex{},
		manager: manager,
		log:     log.WithPrefix("synthetic-svc"),
	}
}

func (s *SyntheticService) Name() string {
	return "synthetic-service"
}

// Dependencies returns service dependencies
func (s *SyntheticService) Dependencies() []string {
	return []string{}
}

func (s *SyntheticService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.manager.Start(ctx)
	return nil
}

func (s *SyntheticService) Stop(ctx context.Context) error {
	s.manager.Stop()
	return nil
}

// AddProbe adds a new synthetic probe via the UI
func (s *SyntheticService) AddProbe(name, ptype, target string, intervalSeconds int) {
	p := &monitoring.SyntheticProbe{
		ID:       name, // Simplified for now
		Name:     name,
		Type:     monitoring.ProbeType(ptype),
		Target:   target,
		Interval: time.Duration(intervalSeconds) * time.Second,
		Timeout:  5 * time.Second,
	}
	s.manager.AddProbe(p)
}

// GetResults returns the latest probe results to the UI
func (s *SyntheticService) GetResults() []monitoring.ProbeResult {
	return s.manager.GetResults()
}
