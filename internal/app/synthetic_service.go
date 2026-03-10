package app

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
	return "SyntheticService"
}

func (s *SyntheticService) Startup(ctx context.Context) {
	s.ctx = ctx
	s.manager.Start(ctx)
}

func (s *SyntheticService) Shutdown() {
	s.manager.Stop()
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
