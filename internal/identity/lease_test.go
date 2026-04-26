package identity

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	// Use modernc.org/sqlite (CGO-free) so the test passes whether
	// CGO_ENABLED=0 or =1. The platform's primary database stack
	// already uses this driver — see internal/database/database_pure.go.
	_ "modernc.org/sqlite"
)

func newLedger(t *testing.T) *LeaseLedger {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	l := NewLeaseLedger(db)
	if err := l.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return l
}

// TestLookupAtTime_DhcpChurn verifies the "alert on Tuesday must map
// to Tuesday's lease holder, not today's" semantics from the audit.
func TestLookupAtTime_DhcpChurn(t *testing.T) {
	l := newLedger(t)
	ctx := context.Background()
	tenant := "t1"
	ip := "10.0.0.7"

	tue := time.Date(2026, 4, 21, 9, 0, 0, 0, time.UTC) // alert time
	wed := time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC)
	thu := time.Date(2026, 4, 23, 9, 0, 0, 0, time.UTC)

	// laptop-A held the lease across Tuesday.
	if err := l.Record(ctx, tenant, ip, "laptop-A", "aa:bb:cc:dd:ee:01", tue.Add(-2*time.Hour), "dhcp"); err != nil {
		t.Fatalf("record A: %v", err)
	}
	// laptop-B took the lease on Wednesday.
	if err := l.Record(ctx, tenant, ip, "laptop-B", "aa:bb:cc:dd:ee:02", wed, "dhcp"); err != nil {
		t.Fatalf("record B: %v", err)
	}
	// laptop-C took it on Thursday.
	if err := l.Record(ctx, tenant, ip, "laptop-C", "aa:bb:cc:dd:ee:03", thu, "dhcp"); err != nil {
		t.Fatalf("record C: %v", err)
	}

	// At Tuesday alert time the lease must resolve to laptop-A.
	got, err := l.LookupAtTime(ctx, tenant, ip, tue)
	if err != nil {
		t.Fatalf("lookup tue: %v", err)
	}
	if got.Hostname != "laptop-A" {
		t.Errorf("Tuesday lookup: got %q, want laptop-A", got.Hostname)
	}

	// At Wednesday it must have moved to laptop-B.
	got, err = l.LookupAtTime(ctx, tenant, ip, wed.Add(time.Hour))
	if err != nil {
		t.Fatalf("lookup wed: %v", err)
	}
	if got.Hostname != "laptop-B" {
		t.Errorf("Wed lookup: got %q, want laptop-B", got.Hostname)
	}

	// At "now" (after thursday) it must be laptop-C.
	got, err = l.LookupAtTime(ctx, tenant, ip, thu.Add(time.Hour))
	if err != nil {
		t.Fatalf("lookup thu: %v", err)
	}
	if got.Hostname != "laptop-C" {
		t.Errorf("Thu lookup: got %q, want laptop-C", got.Hostname)
	}

	// Querying BEFORE any lease should return ErrLeaseNotFound.
	_, err = l.LookupAtTime(ctx, tenant, ip, tue.Add(-24*time.Hour))
	if !errors.Is(err, ErrLeaseNotFound) {
		t.Errorf("pre-history: got err=%v, want ErrLeaseNotFound", err)
	}
}

// TestRecord_RefreshNoop verifies that re-recording the same lease is
// a no-op (no extra row, no churn).
func TestRecord_RefreshNoop(t *testing.T) {
	l := newLedger(t)
	ctx := context.Background()
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		if err := l.Record(ctx, "t1", "10.0.0.1", "alpha", "mac1", now.Add(time.Duration(i)*time.Minute), "dhcp"); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	hist, err := l.History(ctx, "t1", "10.0.0.1", 10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(hist) != 1 {
		t.Errorf("expected 1 row (refresh no-op), got %d", len(hist))
	}
}

// TestTenantIsolation verifies that two tenants holding the same IP
// resolve independently.
func TestTenantIsolation(t *testing.T) {
	l := newLedger(t)
	ctx := context.Background()
	t0 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if err := l.Record(ctx, "tenantA", "192.168.1.5", "hostA", "macA", t0, "dhcp"); err != nil {
		t.Fatalf("record A: %v", err)
	}
	if err := l.Record(ctx, "tenantB", "192.168.1.5", "hostB", "macB", t0, "dhcp"); err != nil {
		t.Fatalf("record B: %v", err)
	}
	a, _ := l.LookupAtTime(ctx, "tenantA", "192.168.1.5", t0.Add(time.Hour))
	b, _ := l.LookupAtTime(ctx, "tenantB", "192.168.1.5", t0.Add(time.Hour))
	if a.Hostname != "hostA" || b.Hostname != "hostB" {
		t.Errorf("tenant isolation broken: a=%q b=%q", a.Hostname, b.Hostname)
	}
}
