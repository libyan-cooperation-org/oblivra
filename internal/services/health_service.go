package services

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/platform"
)

// HealthService exposes health checking functionality to the frontend
type HealthService struct {
	BaseService
	ctx           context.Context
	log           *logger.Logger
	healthChecker *monitoring.HealthChecker
	bus           *eventbus.Bus
	registry      *platform.Registry
}

// Name returns the name of the service
func (s *HealthService) Name() string { return "health-service" }

// Dependencies returns service dependencies.
func (s *HealthService) Dependencies() []string {
	return []string{}
}

// NewHealthService creates a new HealthService
func NewHealthService(log *logger.Logger, bus *eventbus.Bus, hc *monitoring.HealthChecker, registry *platform.Registry) *HealthService {
	s := &HealthService{
		log:           log.WithPrefix("healthservice"),
		bus:           bus,
		healthChecker: hc,
		registry:      registry,
	}

	// Route internal status changes to the frontend event bus
	if hc != nil {
		s.healthChecker.SetCallback(func(hostID string, health monitoring.HostHealth) {
			s.bus.Publish("health_status_changed", health)
		})
	}

	return s
}

// Startup is called at application startup
func (s *HealthService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("Health Service started")
	if s.healthChecker != nil {
		s.healthChecker.Start()
	}
	return nil
}

func (s *HealthService) Stop(ctx context.Context) error {
	if s.healthChecker != nil {
		s.healthChecker.Stop()
	}
	return nil
}

// RegisterHost adds a host to be monitored
func (s *HealthService) RegisterHost(hostID, address string) {
	if s.healthChecker == nil {
		return
	}
	s.healthChecker.RegisterHost(hostID, address)
}

// UnregisterHost removes a host from monitoring
func (s *HealthService) UnregisterHost(hostID string) {
	if s.healthChecker == nil {
		return
	}
	s.healthChecker.UnregisterHost(hostID)
}

// GetHealth returns the health of a specific host
func (s *HealthService) GetHealth(hostID string) map[string]interface{} {
	if s.healthChecker == nil {
		return map[string]interface{}{"status": "unknown"}
	}
	health, ok := s.healthChecker.GetHealth(hostID)
	if !ok {
		return map[string]interface{}{"status": "unknown"}
	}
	return map[string]interface{}{
		"host_id":       health.HostID,
		"status":        string(health.Status),
		"latency_ms":    health.Latency,
		"success_rate":  health.SuccessRate,
		"last_error":    health.LastError,
		"checks_total":  health.ChecksTotal,
		"checks_failed": health.ChecksFailed,
	}
}

// GetAllHealth returns the health of all monitored hosts and internally registered services
func (s *HealthService) GetAllHealth() map[string]interface{} {
	result := make(map[string]interface{})

	// 1. External Host Health
	if s.healthChecker != nil {
		hosts := s.healthChecker.GetAllHealth()
		result["hosts"] = hosts
	} else {
		result["hosts"] = map[string]monitoring.HostHealth{}
	}

	// 2. Internal Service Health
	servicesHealth := make(map[string]string)
	if s.registry != nil {
		for name, svc := range s.registry.GetServices() {
			if reporter, ok := svc.(platform.HealthReporter); ok {
				// Query the health
				err := reporter.Health(s.ctx)
				if err != nil {
					servicesHealth[name] = "degraded: " + err.Error()
				} else {
					servicesHealth[name] = "healthy"
				}
			} else {
				// Implicitly healthy if it doesn't report otherwise, but we omit it or mark as running
				servicesHealth[name] = "running (no health reporting)"
			}
		}
	}
	result["services"] = servicesHealth
	
	// 3. Overall System Status
	status := "Operational"
	if s.registry != nil {
		if vault, err := s.registry.Get("vault-service"); err == nil && vault != nil {
			// Type assertion for IsUnlocked (assuming VaultService implements it)
			if vs, ok := vault.(interface{ IsUnlocked() bool }); ok {
				if !vs.IsUnlocked() {
					status = "Locked"
				}
			}
		}
	}
	result["Status"] = status

	return result
}
