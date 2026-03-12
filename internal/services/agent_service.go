package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// AgentService exposes agent control and reporting functions to the Wails frontend
type AgentService struct {
	BaseService
	server *ingest.AgentServer
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
	return nil
}

// NewAgentService injects the ingest AgentServer dependency to read telemetry stats
func NewAgentService(server *ingest.AgentServer, log *logger.Logger) *AgentService {
	return &AgentService{
		server: server,
		log:    log.WithPrefix("agent_service"),
	}
}

// AgentDTO represents the UI model for an agent
type AgentDTO struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	Version       string `json:"version"`
	LastSeen      string `json:"last_seen"` // ISO8601 string
	RemoteAddress string `json:"remote_address"`
	Status        string `json:"status"` // "online" | "offline"
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
			Version:       a.Version,
			LastSeen:      a.LastSeen,
			RemoteAddress: a.RemoteAddress,
			Status:        status,
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
