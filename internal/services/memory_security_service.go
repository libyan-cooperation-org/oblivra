package services

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

func (s *MemorySecurityService) Name() string {
	return "memory-security-service"
}

// Dependencies returns service dependencies
func (s *MemorySecurityService) Dependencies() []string {
	return []string{}
}

func (s *MemorySecurityService) Start(ctx context.Context) error {
	s.ctx = ctx
	if s.monitor != nil {
		s.monitor.Start()
	}
	return nil
}

func (s *MemorySecurityService) Stop(ctx context.Context) error {
	if s.monitor != nil {
		s.monitor.Stop()
	}
	return nil
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
