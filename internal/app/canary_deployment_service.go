package app

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/security"
)

// canaryPaths are the directories in which canary files are deployed per host.
// Chosen to be attractive to ransomware scanners: root-level, desktop, documents.
var canaryPaths = []string{
	"/",
	"/home",
	"/root",
	"/tmp",
	"/var/www",
	"/Users",      // macOS
	"C:\\Users",   // Windows
	"C:\\",        // Windows root
}

// CanaryDeploymentService listens for new agent registrations and automatically
// deploys canary files to each new host. When a canary is touched, it publishes
// a high-confidence ransomware signal to the detection bus.
type CanaryDeploymentService struct {
	BaseService
	canary  *security.CanaryService
	ssh     SSHManager
	bus     *eventbus.Bus
	log     *logger.Logger
	ctx     context.Context

	// Track deployment state to avoid re-deploying on every heartbeat
	deployed map[string]time.Time // hostID -> last deployment time
}

func NewCanaryDeploymentService(
	canary *security.CanaryService,
	ssh SSHManager,
	bus *eventbus.Bus,
	log *logger.Logger,
) *CanaryDeploymentService {
	return &CanaryDeploymentService{
		canary:   canary,
		ssh:      ssh,
		bus:      bus,
		log:      log.WithPrefix("canary_deploy"),
		deployed: make(map[string]time.Time),
	}
}

func (s *CanaryDeploymentService) Name() string { return "CanaryDeploymentService" }

func (s *CanaryDeploymentService) Startup(ctx context.Context) {
	s.ctx = ctx
	s.log.Info("Canary deployment service starting...")

	// Trigger automatic canary deployment when a new agent registers
	s.bus.Subscribe("agent.registered", func(e eventbus.Event) {
		data, ok := e.Data.(map[string]interface{})
		if !ok {
			return
		}
		hostID, _ := data["host_id"].(string)
		if hostID == "" {
			return
		}
		// Deploy asynchronously — don't block the ingestion path
		go s.deployToHost(hostID)
	})

	// Also watch FIM events for canary access — publish high-confidence signal
	s.bus.Subscribe("fim.file_accessed", func(e eventbus.Event) {
		data, ok := e.Data.(map[string]interface{})
		if !ok {
			return
		}
		hostID, _ := data["host_id"].(string)
		path, _ := data["path"].(string)
		action, _ := data["action"].(string)

		if hostID == "" || path == "" {
			return
		}
		if s.canary != nil && s.canary.IsCanaryFile(path) {
			s.log.Warn("[CANARY] Hit detected on %s — path: %s action: %s", hostID, path, action)
			s.bus.Publish("ransomware.canary_hit", map[string]interface{}{
				"host_id": hostID,
				"path":    path,
				"action":  action,
			})
		}
	})

	// Watch FIM modified events too — ransomware writes before it renames
	s.bus.Subscribe(string(eventbus.EventFIMModified), func(e eventbus.Event) {
		data, ok := e.Data.(map[string]interface{})
		if !ok {
			return
		}
		hostID, _ := data["host_id"].(string)
		path, _ := data["path"].(string)
		if hostID == "" || path == "" {
			return
		}
		if s.canary != nil && s.canary.IsCanaryFile(path) {
			s.log.Warn("[CANARY] Canary MODIFIED on %s — path: %s", hostID, path)
			s.bus.Publish("ransomware.canary_hit", map[string]interface{}{
				"host_id": hostID,
				"path":    path,
				"action":  "modified",
			})
		}
	})
}

func (s *CanaryDeploymentService) Shutdown() {
	s.log.Info("Canary deployment service shutting down...")
}

// deployToHost deploys canary files to all standard paths on the host.
// Skips paths that fail (e.g. permission denied) without blocking the rest.
func (s *CanaryDeploymentService) deployToHost(hostID string) {
	// Rate-limit: only redeploy if it's been more than 24h
	if last, ok := s.deployed[hostID]; ok {
		if time.Since(last) < 24*time.Hour {
			return
		}
	}

	s.log.Info("[CANARY] Deploying canary files to host %s...", hostID)

	deployed := 0
	for _, dir := range canaryPaths {
		canaryPath := filepath.Join(dir, ".oblivra_canary")
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)

		if err := s.deployFile(ctx, hostID, canaryPath); err != nil {
			s.log.Debug("[CANARY] Skipped %s on %s: %v", canaryPath, hostID, err)
		} else {
			deployed++
		}
		cancel()
	}

	s.deployed[hostID] = time.Now()
	s.log.Info("[CANARY] Deployed %d canary files to host %s", deployed, hostID)

	s.bus.Publish("canary.deployed", map[string]interface{}{
		"host_id":      hostID,
		"count":        deployed,
		"deployed_at":  time.Now().UTC().Format(time.RFC3339),
	})
}

// DeployToHost is the public frontend-callable version for manual deployment.
func (s *CanaryDeploymentService) DeployToHost(hostID string) error {
	go s.deployToHost(hostID)
	return nil
}

// deployFile writes a single canary file to the target host via SSH WriteFile.
func (s *CanaryDeploymentService) deployFile(ctx context.Context, hostID, path string) error {
	if s.canary == nil {
		return fmt.Errorf("canary service not available")
	}
	if s.ssh == nil {
		return fmt.Errorf("SSH provider not available")
	}

	// CanaryService.DeployCanary uses the SFTP provider (which is the SSH service)
	return s.canary.DeployCanary(ctx, hostID, path)
}
