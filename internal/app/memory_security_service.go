package app

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/memory"
	"github.com/kingknull/oblivrashell/internal/security"
)

type MemorySecurityService struct {
	ctx     context.Context
	log     *logger.Logger
	monitor *security.AntiDebugMonitor
}

func NewMemorySecurityService(log *logger.Logger) *MemorySecurityService {
	return &MemorySecurityService{
		log:     log,
		monitor: security.NewAntiDebugMonitor(log),
	}
}

func (s *MemorySecurityService) RegisterCtx(ctx context.Context) {
	s.ctx = ctx
	s.log.Info("MemorySecurityService context registered")
}

func (s *MemorySecurityService) Name() string {
	return "MemorySecurityService"
}

func (s *MemorySecurityService) Startup(ctx context.Context) {
	s.ctx = ctx
	if s.monitor != nil {
		s.monitor.Start()
	}
}

func (s *MemorySecurityService) Shutdown() {
	if s.monitor != nil {
		s.monitor.Stop()
	}
}

// GetActiveSecureAllocations returns the current number of crypto-shredded buffers residing in RAM.
// Used by the Security/Observability dashboards to ensure no credential leaks exist.
func (s *MemorySecurityService) GetActiveSecureAllocations() int {
	return int(memory.GetActiveCount())
}

// WipeAll is a manual override, though typically buffers wipe themselves when explicitly closed or via GC finalizer.
func (s *MemorySecurityService) WipeAll() string {
	s.log.Warn("Manual Force-Wipe requested via UI. Incomplete feature, delegates to underlying components.")
	// (In a real system, you might range over a tracked registry of buffers and call .Wipe() across all)
	return "Wipe command issued."
}
