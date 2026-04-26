package cluster

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeApplier captures every ExecContext call for assertion.
type fakeApplier struct {
	calls []fakeCall
}

type fakeCall struct {
	query string
	args  []interface{}
}

type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) { return 1, nil }
func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

func (f *fakeApplier) ExecContext(_ context.Context, q string, args ...interface{}) (LocalExecResult, error) {
	f.calls = append(f.calls, fakeCall{q, args})
	return fakeResult{}, nil
}

// TestNoBackend verifies the safety rail when neither backend is set.
func TestNoBackend(t *testing.T) {
	r := NewStateReplicator(nil, nil)
	err := r.ApplyAlertState(context.Background(), AlertState{ID: "x", TenantID: "t"})
	if !errors.Is(err, ErrNoBackend) {
		t.Errorf("got %v, want ErrNoBackend", err)
	}
}

// TestApplyAlertState_Local verifies the single-node path writes the
// expected SQL and binds.
func TestApplyAlertState_Local(t *testing.T) {
	fa := &fakeApplier{}
	r := NewStateReplicator(nil, fa)

	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	err := r.ApplyAlertState(context.Background(), AlertState{
		ID:        "a-1",
		TenantID:  "t-prod",
		Name:      "ssh-bruteforce",
		Severity:  "high",
		Status:    "open",
		Host:      "srv-01",
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(fa.calls) != 1 {
		t.Fatalf("calls: got %d, want 1", len(fa.calls))
	}
	c := fa.calls[0]
	if !strings.Contains(c.query, "INSERT OR REPLACE INTO alerts") {
		t.Errorf("unexpected query: %s", c.query)
	}
	if c.args[0] != "a-1" || c.args[1] != "t-prod" {
		t.Errorf("bad bind values: %v", c.args[:2])
	}
}

// TestApplyPlaybook_Local — same shape, different table.
func TestApplyPlaybook_Local(t *testing.T) {
	fa := &fakeApplier{}
	r := NewStateReplicator(nil, fa)
	err := r.ApplyPlaybook(context.Background(), PlaybookDefinition{
		ID:       "pb-1", TenantID: "t-prod", Name: "isolate-host", Author: "alice",
		Version: 3, Body: "steps: []", Enabled: true,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(fa.calls[0].query, "INSERT OR REPLACE INTO playbooks") {
		t.Errorf("query mismatch: %s", fa.calls[0].query)
	}
	if fa.calls[0].args[6] != 1 {
		t.Errorf("expected enabled=1, got %v", fa.calls[0].args[6])
	}
}

// TestApplyThreatIntel_Local — same shape, threat_intel_indicators.
func TestApplyThreatIntel_Local(t *testing.T) {
	fa := &fakeApplier{}
	r := NewStateReplicator(nil, fa)
	err := r.ApplyThreatIntel(context.Background(), ThreatIntelIndicator{
		Value: "1.2.3.4", Type: "ip", Source: "AbuseIPDB",
		Severity: "high", CampaignID: "c-007",
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(fa.calls[0].query, "INSERT OR REPLACE INTO threat_intel_indicators") {
		t.Errorf("query mismatch: %s", fa.calls[0].query)
	}
}

// TestRequestIDIdempotent verifies the same logical write produces
// the same request id (so the FSM's _raft_applied dedupe table can
// catch retries).
func TestRequestIDIdempotent(t *testing.T) {
	fa := &fakeApplier{}
	r := NewStateReplicator(nil, fa)
	args1 := []interface{}{"a-1", "t-prod", "x", "high", "open", "h", "raw", "d", "2026-04-26T12:00:00Z"}
	args2 := []interface{}{"a-1", "t-prod", "x", "high", "open", "h", "raw", "d", "2026-04-26T12:00:00Z"}
	id1 := r.deriveRequestID("alert", "k", "INSERT", args1)
	id2 := r.deriveRequestID("alert", "k", "INSERT", args2)
	if id1 != id2 {
		t.Errorf("expected stable request id; got %s vs %s", id1, id2)
	}
	id3 := r.deriveRequestID("alert", "k", "INSERT", []interface{}{"a-2"})
	if id1 == id3 {
		t.Errorf("different bind values should hash differently")
	}
}
