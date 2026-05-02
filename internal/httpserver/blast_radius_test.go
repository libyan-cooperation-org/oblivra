package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/services"
	"github.com/kingknull/oblivra/internal/storage/hot"
	"github.com/kingknull/oblivra/internal/storage/search"
	"github.com/kingknull/oblivra/internal/wal"
)

// TestBlastRadius_TenantBoundedKeyCannotReadOtherTenants is the live-
// HTTP integration test for the auth middleware's tenant-scoping
// enforcement. Wires up a minimal Server with two API keys —
// `keyA` bound to tenant-a and `keyB` bound to tenant-b — seeds the
// hot store with events for both tenants, then asserts:
//
//  1. keyA + ?tenant=tenant-a returns ONLY tenant-a events.
//  2. keyA + ?tenant=tenant-b returns 403 (cross-tenant access denied).
//  3. keyA with NO tenant param still only sees tenant-a events
//     (middleware injects the bound tenant — no implicit cross-tenant
//     read via "missing parameter").
//  4. keyB sees ONLY tenant-b events with the same checks reversed.
//
// This is the "blast-radius" property: compromise of a single tenant's
// API key MUST NOT enable enumeration of other tenants.
func TestBlastRadius_TenantBoundedKeyCannotReadOtherTenants(t *testing.T) {
	srv, cleanup := newBlastRadiusServer(t, "keyA:admin:tenant-a,keyB:admin:tenant-b,keyAdmin:admin:*")
	defer cleanup()

	// Hit tenant-a from keyA — must return only tenant-a events.
	{
		body, status := doSearch(t, srv.URL, "keyA", "tenant-a")
		if status != http.StatusOK {
			t.Fatalf("keyA tenant-a: status=%d body=%s", status, body)
		}
		mustOnlyTenant(t, body, "tenant-a", "keyA→tenant-a")
	}

	// keyA explicitly asking for tenant-b → 403, no leakage.
	{
		body, status := doSearch(t, srv.URL, "keyA", "tenant-b")
		if status != http.StatusForbidden {
			t.Fatalf("expected 403 for cross-tenant, got %d body=%s", status, body)
		}
	}

	// keyA with NO tenant param → middleware forces tenant-a.
	{
		body, status := doSearchNoTenant(t, srv.URL, "keyA")
		if status != http.StatusOK {
			t.Fatalf("keyA no-tenant: status=%d body=%s", status, body)
		}
		mustOnlyTenant(t, body, "tenant-a", "keyA→implicit")
	}

	// keyB symmetric: only tenant-b, blocked from tenant-a.
	{
		body, status := doSearch(t, srv.URL, "keyB", "tenant-b")
		if status != http.StatusOK {
			t.Fatalf("keyB tenant-b: status=%d body=%s", status, body)
		}
		mustOnlyTenant(t, body, "tenant-b", "keyB→tenant-b")
	}
	{
		_, status := doSearch(t, srv.URL, "keyB", "tenant-a")
		if status != http.StatusForbidden {
			t.Fatalf("keyB→tenant-a expected 403, got %d", status)
		}
	}

	// Wildcard admin key sees both tenants — that's intentional, the
	// platform admin role has cross-tenant scope.
	{
		body, _ := doSearch(t, srv.URL, "keyAdmin", "tenant-a")
		mustOnlyTenant(t, body, "tenant-a", "admin→a")
		body, _ = doSearch(t, srv.URL, "keyAdmin", "tenant-b")
		mustOnlyTenant(t, body, "tenant-b", "admin→b")
	}
}

// TestBlastRadius_AuditTrailRecordsRejection asserts that every
// cross-tenant denial is observable — the audit middleware logs the
// 403 with the offending key id + path + tried-tenant so a SIEM can
// alert on enumeration attempts after the fact.
func TestBlastRadius_AuditTrailRecordsRejection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dir := t.TempDir()
	audit, err := services.NewDurable(logger, dir, nil)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	t.Cleanup(func() { _ = audit.Close() })
	srv, cleanup := newBlastRadiusServerWithAudit(t, "keyA:admin:tenant-a", audit)
	defer cleanup()

	// keyA tries tenant-b — must 403.
	_, status := doSearch(t, srv.URL, "keyA", "tenant-b")
	if status != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", status)
	}

	// At least one cross-tenant auth.deny entry should land in audit.
	found := false
	for _, e := range audit.Recent(50) {
		if e.Action == "auth.deny" && e.Detail["reason"] == "cross-tenant" &&
			e.Detail["attemptedTenant"] == "tenant-b" && strings.HasPrefix(e.Detail["subject"], "keyA") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no auth.deny audit entry recorded the cross-tenant attempt — silent rejections are unacceptable")
	}
}

// ---- helpers ----

func newBlastRadiusServer(t *testing.T, apiKeys string) (*httptest.Server, func()) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dir := t.TempDir()
	audit, err := services.NewDurable(logger, dir, nil)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	srv, cleanup := newBlastRadiusServerWithAudit(t, apiKeys, audit)
	prev := cleanup
	return srv, func() {
		prev()
		_ = audit.Close()
	}
}

func newBlastRadiusServerWithAudit(t *testing.T, apiKeys string, audit *services.AuditService) (*httptest.Server, func()) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	walDir := t.TempDir()
	w, err := wal.Open(wal.Options{Dir: walDir})
	if err != nil {
		t.Fatalf("wal.Open: %v", err)
	}
	store, err := hot.Open(hot.Options{InMemory: true})
	if err != nil {
		t.Fatalf("hot.Open: %v", err)
	}
	idx, err := search.Open(search.Options{InMemory: true})
	if err != nil {
		t.Fatalf("search.Open: %v", err)
	}
	pipe := ingest.New(logger, w, store, idx, nil)
	siem := services.NewSiemService(logger, pipe)

	// Seed events for both tenants. Same hostId so the only thing
	// distinguishing them is TenantID — exactly the surface we want
	// the middleware to enforce.
	for _, tn := range []string{"tenant-a", "tenant-b"} {
		for i := 0; i < 5; i++ {
			ev := &events.Event{
				TenantID:  tn,
				Source:    events.SourceREST,
				HostID:    "shared-host",
				EventType: "test.seed",
				Severity:  events.SeverityInfo,
				Message:   tn + " seed event",
				Timestamp: time.Now(),
			}
			if err := ev.Validate(); err != nil {
				t.Fatalf("validate: %v", err)
			}
			if err := pipe.Submit(context.Background(), ev); err != nil {
				t.Fatalf("submit: %v", err)
			}
		}
	}

	auth := NewAuth(apiKeys)
	auth.AttachAudit(audit)
	srv := New(logger, Deps{
		System: services.NewSystemService(logger),
		Siem:   siem,
		Audit:  audit,
		Auth:   auth,
	})
	ts := httptest.NewServer(srv.Handler())
	cleanup := func() {
		ts.Close()
		_ = idx.Close()
		_ = store.Close()
		_ = w.Close()
	}
	return ts, cleanup
}

func doSearch(t *testing.T, base, key, tenant string) (string, int) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, base+"/api/v1/siem/search?tenant="+tenant+"&q=*", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), resp.StatusCode
}

func doSearchNoTenant(t *testing.T, base, key string) (string, int) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, base+"/api/v1/siem/search?q=*", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), resp.StatusCode
}

// mustOnlyTenant decodes a SIEM search response and fails the test if
// any returned event belongs to a tenant other than `wantTenant`.
func mustOnlyTenant(t *testing.T, body, wantTenant, label string) {
	t.Helper()
	var resp struct {
		Events []events.Event `json:"events"`
		Total  int            `json:"total"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("%s: bad json: %v body=%s", label, err, body)
	}
	if len(resp.Events) == 0 {
		t.Fatalf("%s: zero events returned (seed expected) body=%s", label, body)
	}
	for _, e := range resp.Events {
		if e.TenantID != wantTenant {
			t.Fatalf("%s: leaked event from tenant %q (want %q): id=%s msg=%s",
				label, e.TenantID, wantTenant, e.ID, e.Message)
		}
	}
}
