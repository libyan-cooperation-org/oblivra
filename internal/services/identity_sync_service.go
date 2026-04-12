package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/identity"
	"github.com/kingknull/oblivrashell/internal/identity/connectors"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// IdentitySyncService manages background synchronization of identities from external IdPs.
type IdentitySyncService struct {
	BaseService
	connectorRepo database.IdentityConnectorStore
	identity      *IdentityService
	tenantRepo    *database.TenantRepository
	log           *logger.Logger
	stop          chan struct{}
}

func NewIdentitySyncService(
	connectorRepo database.IdentityConnectorStore,
	identity *IdentityService,
	tenantRepo *database.TenantRepository,
	log *logger.Logger,
) *IdentitySyncService {
	return &IdentitySyncService{
		connectorRepo: connectorRepo,
		identity:      identity,
		tenantRepo:    tenantRepo,
		log:           log.WithPrefix("identity-sync"),
		stop:          make(chan struct{}),
	}
}

func (s *IdentitySyncService) Name() string { return "identity-sync-service" }

func (s *IdentitySyncService) Start(ctx context.Context) error {
	s.log.Info("Starting Identity Sync Service...")
	go s.run(ctx)
	return nil
}

func (s *IdentitySyncService) Stop(ctx context.Context) error {
	s.log.Info("Stopping Identity Sync Service...")
	close(s.stop)
	return nil
}

func (s *IdentitySyncService) run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performSyncCycle(ctx)
		case <-s.stop:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (s *IdentitySyncService) performSyncCycle(ctx context.Context) {
	tenants, err := s.tenantRepo.ListAllTenants(ctx)
	if err != nil {
		s.log.Error("Failed to list tenants for sync: %v", err)
		return
	}

	for _, t := range tenants {
		tenantCtx := database.WithTenant(ctx, t.ID)
		s.syncTenantConnectors(tenantCtx, t.ID)
	}
}

func (s *IdentitySyncService) syncTenantConnectors(ctx context.Context, tenantID string) {
	connectorsList, err := s.connectorRepo.List(ctx)
	if err != nil {
		return
	}

	for _, c := range connectorsList {
		if !c.Enabled {
			continue
		}

		// Check if sync is due
		lastSync, _ := time.Parse(time.RFC3339, c.LastSync)
		if time.Since(lastSync) < time.Duration(c.SyncIntervalMins)*time.Minute {
			continue
		}

		s.log.Info("Syncing identity connector %s (%s) for tenant %s", c.Name, c.Type, tenantID)
		
		// Load full connector including decrypted config
		fullConnector, err := s.connectorRepo.GetByID(ctx, c.ID)
		if err != nil {
			s.log.Error("Failed to load connector %s: %v", c.ID, err)
			continue
		}

		go s.runSync(ctx, fullConnector)
	}
}

func (s *IdentitySyncService) runSync(ctx context.Context, c *database.IdentityConnector) {
	s.connectorRepo.MarkSyncStart(ctx, c.ID)

	inst, err := connectors.Create(c.ID, c.Type, c.ConfigJSON)
	if err != nil {
		s.connectorRepo.UpdateStatus(ctx, c.ID, "error", fmt.Sprintf("failed to instantiate: %v", err))
		return
	}

	users, err := inst.FetchUsers(ctx)
	if err != nil {
		s.connectorRepo.UpdateStatus(ctx, c.ID, "error", fmt.Sprintf("fetch failed: %v", err))
		return
	}

	s.log.Info("Syncing %d users from %s", len(users), c.Name)

	for _, ur := range users {
		var u database.User
		identity.FromUserResource(ur, &u)
		u.TenantID = c.TenantID
		u.AuthProvider = c.Type

		if err := s.identity.ProvisionSCIMUser(ctx, &u); err != nil {
			s.log.Warn("Failed to provision user %s: %v", ur.UserName, err)
		}
	}

	s.connectorRepo.UpdateStatus(ctx, c.ID, "ok", "")
}
