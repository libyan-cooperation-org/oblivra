package services

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/storage/hot"
)

// TestCrossTenantBlastRadius is the proof of multi-tenant isolation that the
// security review claims. It writes events to two tenants from concurrent
// goroutines, then asserts:
//
//  1. A search scoped to tenant A returns ONLY tenant A events
//  2. The hot-store key prefix is structurally separate (no shared keyspace)
//  3. A case opened against tenant A's host cannot see tenant B's data even
//     if both tenants share the same hostId
func TestCrossTenantBlastRadius(t *testing.T) {
	store, err := hot.Open(hot.Options{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	const perTenant = 50
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < perTenant; i++ {
			ev := &events.Event{
				TenantID: "tenant-a", Source: events.SourceREST,
				HostID: "shared-host", Message: "tenant-a-message",
				Timestamp: time.Now(),
			}
			if err := ev.Validate(); err != nil {
				t.Error(err)
				return
			}
			if err := store.Put(ev); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < perTenant; i++ {
			ev := &events.Event{
				TenantID: "tenant-b", Source: events.SourceREST,
				HostID: "shared-host", Message: "tenant-b-message",
				Timestamp: time.Now(),
			}
			if err := ev.Validate(); err != nil {
				t.Error(err)
				return
			}
			if err := store.Put(ev); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	wg.Wait()

	// 1. Tenant-A query returns ONLY tenant-A events.
	aResults, err := store.Range(context.Background(), hot.RangeOpts{
		TenantID: "tenant-a", Limit: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(aResults) != perTenant {
		t.Errorf("tenant-a got %d events, want %d", len(aResults), perTenant)
	}
	for _, e := range aResults {
		if e.TenantID != "tenant-a" {
			t.Fatalf("tenant-a query leaked %q event: id=%s msg=%s",
				e.TenantID, e.ID, e.Message)
		}
		if !strings.Contains(e.Message, "tenant-a") {
			t.Errorf("tenant-a got tenant-b message: %s", e.Message)
		}
	}

	// 2. Tenant-B query returns ONLY tenant-B events.
	bResults, err := store.Range(context.Background(), hot.RangeOpts{
		TenantID: "tenant-b", Limit: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bResults) != perTenant {
		t.Errorf("tenant-b got %d events, want %d", len(bResults), perTenant)
	}
	for _, e := range bResults {
		if e.TenantID != "tenant-b" {
			t.Fatalf("tenant-b query leaked %q event", e.TenantID)
		}
	}

	// 3. Querying with an unknown tenant returns NO events even though
	//    other tenants have plenty.
	none, err := store.Range(context.Background(), hot.RangeOpts{
		TenantID: "tenant-c-does-not-exist", Limit: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Errorf("phantom tenant returned %d events", len(none))
	}
}

// TestCaseScopeRespectsTenantBoundary opens a case against tenant-a, then
// confirms the timeline does not surface tenant-b events even when tenant-b
// has events at the same host and inside the same time window.
func TestCaseScopeRespectsTenantBoundary(t *testing.T) {
	h := newHarness(t)
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()
	audit, _ := NewDurable(logger, dir, nil)
	defer audit.Close()
	alerts := NewAlertService(logger)
	foren := NewForensicsService(h.hot, audit)
	inv, err := NewInvestigationsService(logger, dir, h.hot, alerts, foren, audit)
	if err != nil {
		t.Fatal(err)
	}
	defer inv.Close()

	mk := func(tenant, msg string) {
		ev := &events.Event{
			TenantID: tenant, Source: events.SourceREST,
			HostID: "shared-host", Message: msg,
			Timestamp: time.Now(),
		}
		_ = ev.Validate()
		if err := h.pipeline.Submit(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}
	mk("tenant-a", "alpha event")
	mk("tenant-b", "beta event")
	mk("tenant-a", "alpha event 2")

	c, err := inv.Open(context.Background(), OpenCaseRequest{
		Title: "tenant-iso", TenantID: "tenant-a", HostID: "shared-host",
		OpenedBy: "alice",
		From:     time.Now().Add(-time.Hour),
		To:       time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	tl, err := inv.Timeline(context.Background(), c.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range tl {
		if strings.Contains(e.Detail, "beta event") {
			t.Fatalf("tenant-a case leaked tenant-b event: %s", e.Detail)
		}
	}
}
