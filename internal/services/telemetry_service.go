package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/monitoring"
)

// TelemetryService exposes real-time host system metrics to the frontend
type TelemetryService struct {
	BaseService
	ctx     context.Context
	log     *logger.Logger
	manager *monitoring.TelemetryManager
}

// NewTelemetryService creates a new TelemetryService
func (s *TelemetryService) Name() string { return "telemetry-service" }

// Dependencies returns service dependencies
func (s *TelemetryService) Dependencies() []string {
	return []string{}
}

func NewTelemetryService(log *logger.Logger, manager *monitoring.TelemetryManager) *TelemetryService {
	return &TelemetryService{
		log:     log.WithPrefix("telemetry_service"),
		manager: manager,
	}
}

// Startup is called at application startup
func (s *TelemetryService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("Telemetry Service started")
	return nil
}

func (s *TelemetryService) Stop(ctx context.Context) error {
	return nil
}

// GetHostTelemetry returns the latest metrics for a specific host
func (s *TelemetryService) GetHostTelemetry(hostID string) (monitoring.HostTelemetry, bool) {
	if s.manager == nil {
		return monitoring.HostTelemetry{}, false
	}
	return s.manager.GetTelemetry(hostID)
}

// GetFleetTelemetry returns metrics for all hosts currently being monitored
func (s *TelemetryService) GetFleetTelemetry() []monitoring.HostTelemetry {
	if s.manager == nil {
		return nil
	}
	return s.manager.GetAll()
}
