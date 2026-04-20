package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// AgentService exposes agent control and reporting functions to the Wails frontend
type AgentService struct {
	BaseService
	server *ingest.AgentServer
	rbac   *auth.RBACEngine
	log    *logger.Logger
}

func (s *AgentService) Name() string { return "agent-service" }

// Dependencies returns service dependencies
func (s *AgentService) Dependencies() []string {
	return []string{}
}

func (s *AgentService) Start(ctx context.Context) error {
	return nil
}

func (s *AgentService) Stop(ctx context.Context) error {
	if s.server != nil {
		s.server.Stop()
	}
	return nil
}

// NewAgentService injects the ingest AgentServer dependency to read telemetry stats
func NewAgentService(server *ingest.AgentServer, rbac *auth.RBACEngine, log *logger.Logger) *AgentService {
	return &AgentService{
		server: server,
		rbac:   rbac,
		log:    log.WithPrefix("agent_service"),
	}
}

// AgentDTO represents the UI model for an agent
type AgentDTO struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	TenantID      string `json:"tenant_id"`
	Version       string `json:"version"`
	LastSeen      string   `json:"last_seen"` // ISO8601 string
	RemoteAddress string   `json:"remote_address"`
	Status        string   `json:"status"` // "online" | "offline"
	OS            string   `json:"os"`
	Arch          string   `json:"arch"`
	Collectors    []string `json:"collectors"`
	TrustLevel    string   `json:"trust_level"`
	WatchdogActive bool    `json:"watchdog_active"`
}

// ListAgents retrieves the list of currently active agents
func (s *AgentService) ListAgents() []AgentDTO {
	if s.server == nil {
		return []AgentDTO{}
	}

	active := s.server.GetActiveAgents()
	var dtos []AgentDTO

	for _, a := range active {
		status := "online"
		if time.Since(parseTime(a.LastSeen)) > 2*time.Minute {
			status = "offline"
		}

		dtos = append(dtos, AgentDTO{
			ID:            a.ID,
			Hostname:      a.Hostname,
			TenantID:      a.TenantID,
			Version:       a.Version,
			LastSeen:      a.LastSeen,
			RemoteAddress: a.RemoteAddress,
			Status:        status,
			OS:             a.OS,
			Arch:           a.Arch,
			Collectors:     a.Collectors,
			TrustLevel:     a.TrustLevel,
			WatchdogActive: a.WatchdogActive,
		})
	}
	return dtos
}

// PushFleetConfig updates the configuration for all agents in the fleet
func (s *AgentService) PushFleetConfig(intervalMs int, enableFIM, enableSyslog, enableMetrics, enableEventLog bool) error {
	if s.server == nil {
		return fmt.Errorf("agent server not initialized")
	}

	cfg := ingest.FleetConfig{
		Interval:       time.Duration(intervalMs) * time.Millisecond,
		EnableFIM:      enableFIM,
		EnableSyslog:   enableSyslog,
		EnableMetrics:  enableMetrics,
		EnableEventLog: enableEventLog,
	}

	s.server.SetFleetConfig(cfg)
	return nil
}

// KillProcess sends a termination signal for a specific process on a remote agent
func (s *AgentService) KillProcess(agentID string, pid int) error {
	s.log.Warn("Issuing KILL directive for agent=%s PID=%d", agentID, pid)
	s.server.AddAction(agentID, ingest.PendingAction{
		ID:   fmt.Sprintf("kill-%d", time.Now().Unix()),
		Type: ingest.ActionKillProcess,
		Payload: map[string]string{
			"pid": fmt.Sprintf("%d", pid),
		},
	})
	return nil
}

// ToggleQuarantine isolates or restores an agent's network access
func (s *AgentService) ToggleQuarantine(agentID string, enabled bool) error {
	actionType := ingest.ActionRestoreNetwork
	if enabled {
		actionType = ingest.ActionIsolateNetwork
	}

	s.log.Warn("Issuing QUARANTINE directive (enabled=%v) for agent=%s", enabled, agentID)
	s.server.AddAction(agentID, ingest.PendingAction{
		ID:   fmt.Sprintf("quar-%d", time.Now().Unix()),
		Type: actionType,
	})
	return nil
}

// RequestProcessInventory queues a request for a full process list from an agent
func (s *AgentService) RequestProcessInventory(agentID string) error {
	s.log.Info("Requesting process inventory from agent=%s", agentID)
	if s.server == nil {
		return fmt.Errorf("agent server not initialized")
	}

	s.server.AddAction(agentID, ingest.PendingAction{
		ID:   fmt.Sprintf("proc-inv-%d", time.Now().Unix()),
		Type: ingest.ActionProcessInventory,
	})
	return nil
}
