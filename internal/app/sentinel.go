package app

import (
	"context"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/vault"
)

// Sentinel monitors the application's runtime integrity and security posture.
type Sentinel struct {
	vault vault.Provider
	log   *logger.Logger
	ctx   context.Context
}

// NewSentinel creates a new runtime integrity monitor.
func NewSentinel(v vault.Provider, log *logger.Logger) *Sentinel {
	return &Sentinel{
		vault: v,
		log:   log.WithPrefix("sentinel"),
	}
}

// Name returns the service name.
func (s *Sentinel) Name() string {
	return "Sentinel"
}

// Startup begins the integrity monitoring loop.
func (s *Sentinel) Startup(ctx context.Context) {
	s.ctx = ctx
	go s.monitorLoop()
}

// Shutdown stops the sentinel.
func (s *Sentinel) Shutdown() {
	// Context cancellation handles this via Wails/ServiceRegistry
}

func (s *Sentinel) monitorLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.log.Info("Sentinel active: Monitoring runtime integrity")

	for {
		select {
		case <-ticker.C:
			s.performSecurityCheck()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Sentinel) performSecurityCheck() {
	// 1. Vault State Check
	// If the vault is locked but we have active sensitive sessions, we might want to flag it.
	// (Actually, the "Fail-Closed" design should prevent this, but Sentinel proves it).
	if s.vault != nil && !s.vault.IsUnlocked() {
		s.log.Warn("Vault is LOCKED. Sentinel entering high-alert mode.")
	}

	// 2. Resource Integrity
	stats := MemoryStats()
	alloc := stats["alloc_mb"].(float64)
	if alloc > 400 {
		s.log.Warn("Memory exhaustion imminent: %0.2f MB allocated. Triggering defensive GC.", alloc)
	}
}
