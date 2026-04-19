package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type RotationService struct {
	BaseService
	db           database.DatabaseStore
	rotationRepo database.RotationStore
	vaultSvc     *VaultService
	sshSvc       *SSHService
	bus          *eventbus.Bus
	log          *logger.Logger
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewRotationService(db database.DatabaseStore, rotationRepo database.RotationStore, vaultSvc *VaultService, sshSvc *SSHService, bus *eventbus.Bus, log *logger.Logger) *RotationService {
	return &RotationService{
		db:           db,
		rotationRepo: rotationRepo,
		vaultSvc:     vaultSvc,
		sshSvc:       sshSvc,
		bus:          bus,
		log:          log.WithPrefix("rotation_service"),
	}
}

func (s *RotationService) Name() string { return "rotation-service" }

func (s *RotationService) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	go s.worker()
	return nil
}

func (s *RotationService) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}

func (s *RotationService) worker() {
	s.log.Info("Rotation worker started")
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndRotate()
		}
	}
}

func (s *RotationService) checkAndRotate() {
	if !s.vaultSvc.IsUnlocked() {
		s.log.Debug("Vault is locked, skipping rotation check")
		return
	}

	due, err := s.rotationRepo.GetDue(s.ctx)
	if err != nil {
		s.log.Error("Failed to fetch due rotation policies: %v", err)
		return
	}

	for _, policy := range due {
		if policy.NotifyOnly {
			s.notifyRotationDue(policy)
			continue
		}

		s.log.Info("Auto-rotating credential: %s", policy.CredentialID)
		err := s.RotateCredential(s.ctx, policy.CredentialID)
		if err != nil {
			s.log.Error("Auto-rotation failed for %s: %v", policy.CredentialID, err)
			s.bus.Publish("rotation.failed", map[string]string{
				"credential_id": policy.CredentialID,
				"error":         err.Error(),
			})
		}
	}
}

func (s *RotationService) notifyRotationDue(p database.RotationPolicy) {
	s.bus.Publish("rotation.due", map[string]string{
		"credential_id": p.CredentialID,
		"next_rotation": p.NextRotation,
	})
	EmitEvent("rotation:due", p)
}

func (s *RotationService) RotateCredential(ctx context.Context, credID string) error {
	policy, err := s.rotationRepo.GetByCredentialID(ctx, credID)
	if err != nil {
		return err
	}
	if policy == nil {
		return fmt.Errorf("no rotation policy found for credential %s", credID)
	}

	cred, err := s.vaultSvc.creds.GetByID(ctx, credID)
	if err != nil {
		return err
	}

	switch cred.Type {
	case "ssh_key", "key":
		return s.rotateSSHKey(ctx, cred, policy)
	case "password":
		return s.rotatePassword(ctx, cred, policy)
	default:
		return fmt.Errorf("rotation not supported for credential type: %s", cred.Type)
	}
}

func (s *RotationService) rotateSSHKey(ctx context.Context, cred *database.Credential, policy *database.RotationPolicy) error {
	s.log.Info("Rotating SSH Key: %s", cred.Label)

	// 1. Generate new Ed25519 key pair
	newLabel := fmt.Sprintf("%s (Rotated %s)", cred.Label, time.Now().Format("2006-01-02"))
	newPubKey, err := s.vaultSvc.GenerateEd25519Key(ctx, newLabel)
	if err != nil {
		return fmt.Errorf("generate new key: %w", err)
	}

	// 2. Find all hosts using this credential
	// This requires a new method in HostStore or just listing all and filtering
	// For now, let's assume we list all hosts.
	// TODO: Add GetByCredentialID to HostStore
	hosts, err := s.sshSvc.hosts.GetAll(ctx)
	if err != nil {
		return err
	}

	successCount := 0
	var lastErr error

	for _, host := range hosts {
		if host.CredentialID == cred.ID {
			s.log.Info("Pushing new key to host: %s (%s)", host.Label, host.Hostname)
			// We need a session to push the key.
			// If not connected, we try to connect using the OLD key.
			sessionID, err := s.sshSvc.Connect(ctx, host.ID)
			if err != nil {
				s.log.Warn("Failed to connect to %s for key rotation: %v", host.Hostname, err)
				lastErr = err
				continue
			}

			// Deploy the NEW public key
			// We use a simplified version of DeployKey logic
			cmd := fmt.Sprintf("echo '%s' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys", newPubKey)
			_, err = s.sshSvc.Exec(sessionID, cmd)
			if err != nil {
				s.log.Warn("Failed to deploy new key to %s: %v", host.Hostname, err)
				lastErr = err
			} else {
				successCount++
			}
			s.sshSvc.manager.Remove(sessionID)
		}
	}

	if successCount == 0 && len(hosts) > 0 {
		return fmt.Errorf("failed to deploy new key to any hosts: %v", lastErr)
	}

	// 3. Update the policy
	now := time.Now()
	policy.LastRotation = now.Format(time.RFC3339)
	policy.NextRotation = now.AddDate(0, 0, policy.FrequencyDays).Format(time.RFC3339)
	if err := s.rotationRepo.Upsert(ctx, policy); err != nil {
		return err
	}

	s.bus.Publish("rotation.success", cred.ID)
	EmitEvent("rotation:success", map[string]string{
		"id":    cred.ID,
		"label": cred.Label,
	})

	return nil
}

func (s *RotationService) rotatePassword(_ context.Context, cred *database.Credential, _ *database.RotationPolicy) error {
	// Implementation for password rotation (e.g. for database users or local linux users)
	// This is more complex as it requires 'passwd' or SQL 'ALTER USER'
	return fmt.Errorf("automated password rotation not yet fully implemented for credential: %s", cred.Label)
}

func (s *RotationService) RegisterPolicy(ctx context.Context, credID string, frequencyDays int, notifyOnly bool) (string, error) {
	existing, err := s.rotationRepo.GetByCredentialID(ctx, credID)
	if err != nil {
		return "", err
	}

	id := uuid.New().String()
	if existing != nil {
		id = existing.ID
	}

	now := time.Now()
	policy := &database.RotationPolicy{
		ID:            id,
		CredentialID:  credID,
		FrequencyDays: frequencyDays,
		LastRotation:  now.Format(time.RFC3339),
		NextRotation:  now.AddDate(0, 0, frequencyDays).Format(time.RFC3339),
		NotifyOnly:    notifyOnly,
		IsActive:      true,
	}

	if err := s.rotationRepo.Upsert(ctx, policy); err != nil {
		return "", err
	}

	return id, nil
}

func (s *RotationService) ListPolicies(ctx context.Context) ([]database.RotationPolicy, error) {
	return s.rotationRepo.List(ctx)
}
