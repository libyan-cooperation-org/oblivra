package security

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// SftpProvider defines the interface for remote file operations.
type SftpProvider interface {
	WriteFile(ctxID string, path string, contentBase64 string) error
}

// canaryRecord tracks deployment state per host so we don't re-deploy on every heartbeat.
type canaryRecord struct {
	deployedAt time.Time
	paths      []string
}

// CanaryService manages honeypot files on remote hosts.
// On agent registration it automatically deploys canaries to standard locations.
// FIM events on canary paths are published as "ransomware.canary_hit" for RansomwareEngine.
type CanaryService struct {
	log  *logger.Logger
	sftp SftpProvider
	bus  *eventbus.Bus

	mu      sync.Mutex
	deployed map[string]*canaryRecord // hostID -> deployment record
}

// NewCanaryService creates the service. bus may be nil for tests.
func NewCanaryService(log *logger.Logger, sftp SftpProvider) *CanaryService {
	return &CanaryService{
		log:      log.WithPrefix("canary"),
		sftp:     sftp,
		deployed: make(map[string]*canaryRecord),
	}
}

// SetBus injects the event bus (called from container after construction to avoid circular deps).
func (s *CanaryService) SetBus(bus *eventbus.Bus) {
	s.bus = bus
}

func (s *CanaryService) Name() string { return "CanaryService" }

func (s *CanaryService) Startup(ctx context.Context) {
	if s.bus == nil {
		s.log.Warn("[CANARY] No event bus — auto-deployment and hit detection disabled")
		return
	}

	// Auto-deploy canaries when a new agent registers or sends its first heartbeat
	s.bus.Subscribe("agent.registered", s.handleAgentRegistered)
	s.bus.Subscribe("agent.heartbeat", s.handleAgentHeartbeat)

	// Monitor FIM events for canary file access
	s.bus.Subscribe(eventbus.EventFIMModified, s.handleFIMEvent)
	s.bus.Subscribe(eventbus.EventFIMDeleted, s.handleFIMEvent)
	s.bus.Subscribe(eventbus.EventFIMCreated, s.handleFIMEvent)
	s.bus.Subscribe(eventbus.EventFIMRenamed, s.handleFIMEvent)

	s.log.Info("[CANARY] Service started — auto-deployment and FIM monitoring active")
}

func (s *CanaryService) Shutdown() {
	s.log.Info("[CANARY] Service shutting down")
}

// ── Auto-deployment ───────────────────────────────────────────────────────────

// handleAgentRegistered deploys canaries immediately when an agent registers.
func (s *CanaryService) handleAgentRegistered(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}
	hostID, _ := data["host_id"].(string)
	if hostID == "" {
		return
	}
	s.autoDeployIfNeeded(context.Background(), hostID)
}

// handleAgentHeartbeat deploys canaries on first heartbeat if not already deployed.
func (s *CanaryService) handleAgentHeartbeat(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}
	hostID, _ := data["host_id"].(string)
	if hostID == "" {
		return
	}
	s.autoDeployIfNeeded(context.Background(), hostID)
}

// autoDeployIfNeeded deploys the canary set to a host if not already deployed in the last 24h.
func (s *CanaryService) autoDeployIfNeeded(ctx context.Context, hostID string) {
	s.mu.Lock()
	rec, exists := s.deployed[hostID]
	if exists && time.Since(rec.deployedAt) < 24*time.Hour {
		s.mu.Unlock()
		return // already deployed recently
	}
	s.mu.Unlock()

	s.log.Info("[CANARY] Auto-deploying canaries to host %s", hostID)

	// Standard deployment paths — chosen to be attractive to ransomware scanners.
	// We use randomized suffixes for world-writable directories to prevent symlink privilege escalation attacks.
	suffix := time.Now().UnixNano()
	deployPaths := []string{
		fmt.Sprintf("/tmp/.oblivra_canary_%d", suffix),
		"/home/.oblivra_canary",
		fmt.Sprintf("/var/tmp/.oblivra_canary_%d", suffix),
	}

	var deployed []string
	for _, path := range deployPaths {
		if err := s.DeployCanary(ctx, hostID, path); err != nil {
			s.log.Warn("[CANARY] Failed to deploy canary to %s:%s — %v", hostID, path, err)
		} else {
			deployed = append(deployed, path)
		}
	}

	if len(deployed) > 0 {
		s.mu.Lock()
		s.deployed[hostID] = &canaryRecord{
			deployedAt: time.Now(),
			paths:      deployed,
		}
		s.mu.Unlock()

		s.log.Info("[CANARY] Deployed %d canary files to host %s: %s",
			len(deployed), hostID, strings.Join(deployed, ", "))

		if s.bus != nil {
			s.bus.Publish("canary.deployed", map[string]interface{}{
				"host_id":     hostID,
				"paths":       deployed,
				"deployed_at": time.Now().Format(time.RFC3339),
			})
		}
	}
}

// ── FIM monitoring ────────────────────────────────────────────────────────────

// handleFIMEvent checks if a FIM event touches a known canary path.
func (s *CanaryService) handleFIMEvent(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}

	hostID, _ := data["host_id"].(string)
	filePath, _ := data["path"].(string)
	if hostID == "" || filePath == "" {
		return
	}

	if !s.IsCanaryFile(filePath) && !s.isDeployedCanary(hostID, filePath) {
		return
	}

	action := "modified"
	switch event.Type {
	case eventbus.EventFIMDeleted:
		action = "deleted"
	case eventbus.EventFIMCreated:
		action = "created"
	case eventbus.EventFIMRenamed:
		action = "renamed"
	}

	s.log.Warn("[CANARY] 🚨 CANARY FILE %s on host %s: %s", strings.ToUpper(action), hostID, filePath)

	if s.bus != nil {
		// Publish the canary hit — RansomwareEngine subscribes to this
		s.bus.Publish("ransomware.canary_hit", map[string]interface{}{
			"host_id":    hostID,
			"path":       filePath,
			"action":     action,
			"detected_at": time.Now().Format(time.RFC3339),
		})
	}
}

// ── Manual deployment API ─────────────────────────────────────────────────────

// DeployCanary writes a single canary file to a remote host.
func (s *CanaryService) DeployCanary(ctx context.Context, hostID string, path string) error {
	s.log.Info("[CANARY] Deploying canary to %s at %s", hostID, path)

	canaryContent := fmt.Sprintf(
		"OBLIVRA_SECURITY_CANARY\nDO_NOT_MODIFY_OR_DELETE\nTOKEN_%s_%d\n",
		hostID, time.Now().UnixNano(),
	)
	contentBase64 := base64.StdEncoding.EncodeToString([]byte(canaryContent))

	if s.sftp == nil {
		s.log.Warn("[CANARY] SFTP provider not available — skipping deployment to %s", hostID)
		return nil
	}

	return s.sftp.WriteFile(hostID, path, contentBase64)
}

// DeployCanaryFolder deploys multiple canary files under a root path on a remote host.
func (s *CanaryService) DeployCanaryFolder(ctx context.Context, hostID string, rootPath string) error {
	s.log.Info("[CANARY] Deploying canary folder to %s at %s", hostID, rootPath)

	files := []string{
		".oblivra_canary",
		"secrets.txt",
		"passwords.db",
		"backups.zip",
		"credentials.xlsx",
	}
	for _, f := range files {
		path := filepath.Join(rootPath, f)
		if err := s.DeployCanary(ctx, hostID, path); err != nil {
			return fmt.Errorf("deploy canary %s: %w", path, err)
		}
	}
	return nil
}

// GetDeployedHosts returns a snapshot of all hosts with active canary deployments.
func (s *CanaryService) GetDeployedHosts() map[string][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string][]string, len(s.deployed))
	for hostID, rec := range s.deployed {
		result[hostID] = rec.paths
	}
	return result
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// IsCanaryFile returns true if the path matches a well-known canary filename pattern.
func (s *CanaryService) IsCanaryFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	canaryNames := map[string]bool{
		".oblivra_canary": true,
		"secrets.txt":     true,
		"passwords.db":    true,
		"backups.zip":     true,
		"credentials.xlsx": true,
	}
	return canaryNames[base] || strings.Contains(strings.ToLower(path), "oblivra_canary")
}

// isDeployedCanary checks if the path matches one of the paths we specifically deployed.
func (s *CanaryService) isDeployedCanary(hostID, path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, exists := s.deployed[hostID]
	if !exists {
		return false
	}
	for _, p := range rec.paths {
		if p == path {
			return true
		}
	}
	return false
}
