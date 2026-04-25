package architecture_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
)

// TestTenantIsolation validates structural per-tenant isolation at the
// scale claimed in task.md Phase 22.2: 50 tenants × 1000 events each =
// 50k events. Asserts (a) each tenant sees exactly its own events, (b)
// cross-tenant queries return zero matches that belong to other tenants.
//
// Phase 28 audit caught this test running 10 events/tenant; bumped to
// 1000 to match the task.md claim and exercise the index size sweet spot.
func TestTenantIsolation(t *testing.T) {
	application, cleanup := setupTestApp(t) // reuse setupTestApp from smoke_test.go
	defer cleanup()

	const numTenants = 50
	const eventsPerTenant = 1000

	if testing.Short() {
		t.Skip("tenant isolation: skipping 50k-event scale run in -short mode")
	}

	t.Logf("Creating %d tenants and ingesting %d events per tenant sequentially...", numTenants, eventsPerTenant)

	for i := 0; i < numTenants; i++ {
		tenantID := fmt.Sprintf("tenant_%d", i)
		ctx := database.WithTenant(context.Background(), tenantID)

		for j := 0; j < eventsPerTenant; j++ {
			ev := database.HostEvent{
				TenantID:  tenantID,
				HostID:    "host-1",
				Timestamp: time.Now().Format(time.RFC3339),
				EventType: "test_event",
				RawLog:    fmt.Sprintf("Secret event for %s #%d", tenantID, j),
			}
			if err := application.SIEMService.Store().InsertHostEvent(ctx, &ev); err != nil {
				t.Fatalf("Failed to insert event for tenant %s: %v", tenantID, err)
			}
		}
	}

	t.Log("Finished writing events. Waiting for indexing to catch up...")
	time.Sleep(3 * time.Second) // bleve indexing delay

	// Now verify isolation
	t.Logf("Verifying Search isolation across %d tenants", numTenants)
	for i := 0; i < numTenants; i++ {
		tenantID := fmt.Sprintf("tenant_%d", i)
		
		// Setup context as this tenant
		ctx := database.WithTenant(context.Background(), tenantID)

		results, err := application.SIEMService.Store().SearchHostEvents(ctx, "test_event", 5000)
		if err != nil {
			t.Fatalf("Search failed for tenant %s: %v", tenantID, err)
		}

		if len(results) != eventsPerTenant {
			t.Errorf("Tenant %s expected %d events, got %d", tenantID, eventsPerTenant, len(results))
		}

		// Ensure NO event belongs to another tenant based on raw log
		for _, r := range results {
			if !strings.Contains(r.RawLog, tenantID) {
				t.Errorf("Isolation breach! Tenant %s received event belonging to another: %s", tenantID, r.RawLog)
			}
		}

		// What if tenant 0 explicitly searches for tenant 1?
		if i == 0 {
			crossResults, err := application.SIEMService.Store().SearchHostEvents(ctx, "tenant_1", 100)
			if err == nil && len(crossResults) > 0 {
				for _, r := range crossResults {
					if !strings.Contains(r.RawLog, tenantID) {
						t.Errorf("Sandbox breach! Tenant 0 queried and found event belonging to another: %s", r.RawLog)
					}
				}
			}
		}
	}
}
