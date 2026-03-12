package services

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/sync"
)

// SyncService exposes E2E sync functionality to the frontend
type SyncService struct {
	BaseService
	ctx        context.Context
	syncEngine *sync.SyncEngine
	bus        *eventbus.Bus
	log        *logger.Logger
}

func (s *SyncService) Name() string { return "sync-service" }

// Dependencies returns service dependencies
func (s *SyncService) Dependencies() []string {
	return []string{"eventbus"}
}

func NewSyncService(engine *sync.SyncEngine, bus *eventbus.Bus, log *logger.Logger) *SyncService {
	return &SyncService{
		syncEngine: engine,
		bus:        bus,
		log:        log.WithPrefix("sync"),
	}
}

func (s *SyncService) Start(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *SyncService) Stop(ctx context.Context) error {
	return nil
}

// Sync triggers a manual sync pass
func (s *SyncService) Sync() error {
	if s.syncEngine == nil {
		return fmt.Errorf("sync engine not initialized")
	}
	s.log.Info("Triggering manual cloud sync pass")
	return s.syncEngine.Sync()
}

// QueueUpdate queues an update manually
func (s *SyncService) QueueUpdate(itemType string, content interface{}, isDeleted bool) error {
	if s.syncEngine == nil {
		return fmt.Errorf("sync engine not initialized")
	}
	return s.syncEngine.QueueUpdate(itemType, content, isDeleted)
}

// ResolveConflict chooses which version of an item to keep
func (s *SyncService) ResolveConflict(conflictID string, resolution string) error {
	if s.syncEngine == nil {
		return fmt.Errorf("sync engine not initialized")
	}
	s.log.Info("Resolving conflict %s with %s", conflictID, resolution)
	return s.syncEngine.ResolveConflict(conflictID, resolution)
}
