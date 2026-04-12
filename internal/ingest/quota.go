package ingest

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/kingknull/oblivrashell/internal/monitoring"
)

var (
	ErrTenantQuotaExceeded = errors.New("tenant ingestion quota exceeded")
)

// TenantQuotaManager tracks real-time buffer occupancy per tenant to prevent
// "Noisy Neighbor" starvation. It enforces that no single tenant can consume
// more than a predefined percentage of the total pipeline capacity.
type TenantQuotaManager struct {
	counts        sync.Map // map[string]*atomic.Int64
	maxPerTenant  int64
	totalCapacity int64
	metrics       *monitoring.MetricsCollector
}

// NewTenantQuotaManager creates a quota manager with a percentage-based limit.
// limitPct is a value between 0.0 and 1.0 (e.g. 0.70 for 70%).
func NewTenantQuotaManager(totalCapacity int, limitPct float64, mc *monitoring.MetricsCollector) *TenantQuotaManager {
	return &TenantQuotaManager{
		totalCapacity: int64(totalCapacity),
		maxPerTenant:  int64(float64(totalCapacity) * limitPct),
		metrics:       mc,
	}
}

// CheckQuota returns ErrTenantQuotaExceeded if the tenant has already reached
// their fair-share of the pipeline buffer.
func (qm *TenantQuotaManager) CheckQuota(tenantID string) error {
	if tenantID == "" {
		return nil // No quota for untracked/system traffic
	}

	val, _ := qm.counts.LoadOrStore(tenantID, &atomic.Int64{})
	count := val.(*atomic.Int64).Load()

	if count >= qm.maxPerTenant {
		return ErrTenantQuotaExceeded
	}

	return nil
}

// Inc increments the occupancy count for a tenant.
func (qm *TenantQuotaManager) Inc(tenantID string) {
	if tenantID == "" {
		return
	}
	val, _ := qm.counts.LoadOrStore(tenantID, &atomic.Int64{})
	newCount := val.(*atomic.Int64).Add(1)

	if qm.metrics != nil {
		occupancy := float64(newCount) / float64(qm.totalCapacity) * 100
		qm.metrics.SetGauge("ingest_tenant_buffer_occupancy", occupancy, map[string]string{"tenant_id": tenantID})
	}
}

// Dec decrements the occupancy count for a tenant.
func (qm *TenantQuotaManager) Dec(tenantID string) {
	if tenantID == "" {
		return
	}
	val, ok := qm.counts.Load(tenantID)
	if ok {
		newCount := val.(*atomic.Int64).Add(-1)
		if qm.metrics != nil {
			occupancy := float64(newCount) / float64(qm.totalCapacity) * 100
			qm.metrics.SetGauge("ingest_tenant_buffer_occupancy", occupancy, map[string]string{"tenant_id": tenantID})
		}
	}
}
