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
	mu            sync.RWMutex
	counts        map[string]*atomic.Int64
	maxPerTenant  int64
	totalCapacity int64
	metrics       *monitoring.MetricsCollector
}

// NewTenantQuotaManager creates a quota manager with a percentage-based limit.
// limitPct is a value between 0.0 and 1.0 (e.g. 0.70 for 70%).
func NewTenantQuotaManager(totalCapacity int, limitPct float64, mc *monitoring.MetricsCollector) *TenantQuotaManager {
	return &TenantQuotaManager{
		counts:        make(map[string]*atomic.Int64),
		totalCapacity: int64(totalCapacity),
		maxPerTenant:  int64(float64(totalCapacity) * limitPct),
		metrics:       mc,
	}
}

func (qm *TenantQuotaManager) getCounter(tenantID string) *atomic.Int64 {
	qm.mu.RLock()
	counter, exists := qm.counts[tenantID]
	qm.mu.RUnlock()

	if exists {
		return counter
	}

	qm.mu.Lock()
	defer qm.mu.Unlock()
	
	// Double check
	if counter, exists = qm.counts[tenantID]; exists {
		return counter
	}
	
	counter = &atomic.Int64{}
	qm.counts[tenantID] = counter
	return counter
}

// CheckQuota returns ErrTenantQuotaExceeded if the tenant has already reached
// their fair-share of the pipeline buffer.
func (qm *TenantQuotaManager) CheckQuota(tenantID string) error {
	if tenantID == "" {
		return nil // No quota for untracked/system traffic
	}

	counter := qm.getCounter(tenantID)
	if counter.Load() >= qm.maxPerTenant {
		return ErrTenantQuotaExceeded
	}

	return nil
}

// Inc increments the occupancy count for a tenant.
func (qm *TenantQuotaManager) Inc(tenantID string) {
	if tenantID == "" {
		return
	}
	
	counter := qm.getCounter(tenantID)
	newCount := counter.Add(1)

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
	
	qm.mu.RLock()
	counter, ok := qm.counts[tenantID]
	qm.mu.RUnlock()
	
	if ok {
		newCount := counter.Add(-1)
		if qm.metrics != nil {
			occupancy := float64(newCount) / float64(qm.totalCapacity) * 100
			qm.metrics.SetGauge("ingest_tenant_buffer_occupancy", occupancy, map[string]string{"tenant_id": tenantID})
		}
	}
}
