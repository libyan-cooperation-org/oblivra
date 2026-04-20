package services

import (
	"context"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// PlatformMetrics represents aggregate health and usage data across all tenants.
type PlatformMetrics struct {
	ActiveTenants   int     `json:"activeTenants"`
	TotalAgents     int     `json:"totalAgents"`
	PlatformEps     string  `json:"platformEps"`
	ActiveIncidents int     `json:"activeIncidents"`
	StorageUsage    float64 `json:"storageUsage"`
	CPUUsage        float64 `json:"cpuUsage"`
	MemoryUsage     float64 `json:"memoryUsage"`
	UptimeSeconds   int64   `json:"uptimeSeconds"`
}

// PlatformService manages cluster-wide visibility for sovereign administrators.
type PlatformService struct {
	BaseService
	tenantRepo *database.TenantRepository
	hostRepo   database.HostStore
	siemRepo   database.SIEMStore
	log        *logger.Logger
	startTime  time.Time
}

func NewPlatformService(
	tenantRepo *database.TenantRepository,
	hostRepo database.HostStore,
	siemRepo database.SIEMStore,
	log *logger.Logger,
) *PlatformService {
	return &PlatformService{
		tenantRepo: tenantRepo,
		hostRepo:   hostRepo,
		siemRepo:   siemRepo,
		log:        log.WithPrefix("platform"),
		startTime:  time.Now(),
	}
}

func (s *PlatformService) Name() string { return "platform-service" }

func (s *PlatformService) GetMetrics(ctx context.Context) (any, error) {
	tenants, err := s.tenantRepo.ListAllTenants(ctx)
	if err != nil {
		return nil, err
	}

	totalAgents := 0
	totalIncidents := 0
	totalEps := 0.0

	// Aggregate metrics across all tenants
	// In production, this would use cached values from a metrics store like Prometheus/ClickHouse
	for _, t := range tenants {
		tenantCtx := database.WithTenant(ctx, t.ID)
		hosts, _ := s.hostRepo.GetAll(tenantCtx)
		totalAgents += len(hosts)
		
		stats, _ := s.siemRepo.GetGlobalThreatStats(tenantCtx)
		if eps, ok := stats["eps"].(float64); ok {
			totalEps += eps
		}
		if inc, ok := stats["active_incidents"].(int); ok {
			totalIncidents += inc
		}
	}

	return &PlatformMetrics{
		ActiveTenants:   len(tenants),
		TotalAgents:     totalAgents,
		PlatformEps:     formatEPS(totalEps),
		ActiveIncidents: totalIncidents,
		UptimeSeconds:   int64(time.Since(s.startTime).Seconds()),
	}, nil
}

func formatEPS(eps float64) string {
	if eps >= 1000 {
		return fmt.Sprintf("%.1fK", eps/1000)
	}
	return fmt.Sprintf("%.0f", eps)
}
