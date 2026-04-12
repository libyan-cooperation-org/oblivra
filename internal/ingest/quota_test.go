package ingest

import (
	"testing"
)

func TestTenantQuotaEnforcement(t *testing.T) {
	const bufferSize = 100
	const limitPct = 0.70
	const maxPerTenant = 70
	
	qm := NewTenantQuotaManager(bufferSize, limitPct)
	
	tenantA := "tenant-alpha"
	tenantB := "tenant-beta"

	// 1. Fill tenant A to exactly the limit
	for i := 0; i < maxPerTenant; i++ {
		if err := qm.CheckQuota(tenantA); err != nil {
			t.Fatalf("CheckQuota failed at %d: %v", i, err)
		}
		qm.Inc(tenantA)
	}

	// 2. Next event for A should fail
	if err := qm.CheckQuota(tenantA); err == nil {
		t.Error("expected ErrTenantQuotaExceeded for tenant A, got nil")
	}

	// 3. Independent tenant B should still be allowed
	if err := qm.CheckQuota(tenantB); err != nil {
		t.Errorf("CheckQuota(tenantB) failed: %v", err)
	}

	// 4. Releasing capacity for A
	qm.Dec(tenantA)
	if err := qm.CheckQuota(tenantA); err != nil {
		t.Errorf("CheckQuota(tenantA) should pass after release, got %v", err)
	}
}
