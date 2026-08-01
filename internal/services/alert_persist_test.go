package services

import (
	"context"
	"log/slog"
	"testing"
)

// TestAlertPersistAcrossRestart proves alerts (and their lifecycle state)
// survive a service restart via the alerts.log journal.
func TestAlertPersistAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	ctx := context.Background()

	// First lifetime: raise two alerts, walk one through the lifecycle.
	s1 := NewAlertService(logger)
	if err := s1.AttachJournal(dir); err != nil {
		t.Fatalf("attach journal: %v", err)
	}
	a := s1.Raise(ctx, Alert{RuleID: "r-1", RuleName: "ssh brute", Severity: AlertSeverityHigh, HostID: "web-01", Message: "brute force"})
	b := s1.Raise(ctx, Alert{RuleID: "r-2", RuleName: "log clear", Severity: AlertSeverityCritical, HostID: "dc-01", Message: "wevtutil cl"})
	if _, ok := s1.Ack(a.ID, "alice"); !ok {
		t.Fatal("ack failed")
	}
	if _, ok := s1.Assign(a.ID, "alice", "bob"); !ok {
		t.Fatal("assign failed")
	}
	if _, ok := s1.Resolve(a.ID, "bob", "true-positive", "confirmed via auth chain"); !ok {
		t.Fatal("resolve failed")
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Second lifetime: replay must restore both alerts with final state.
	s2 := NewAlertService(logger)
	if err := s2.AttachJournal(dir); err != nil {
		t.Fatalf("re-attach journal: %v", err)
	}
	defer s2.Close()

	if got := s2.Count(); got != 2 {
		t.Fatalf("count after restart = %d, want 2", got)
	}
	ra, ok := s2.Get(a.ID)
	if !ok {
		t.Fatalf("alert %s missing after restart", a.ID)
	}
	if ra.State != "resolved" || ra.Verdict != "true-positive" || ra.AssignedTo != "bob" || ra.ResolvedBy != "bob" {
		t.Fatalf("lifecycle state lost after restart: %+v", ra)
	}
	if ra.AcknowledgedAt == nil || ra.AssignedAt == nil || ra.ResolvedAt == nil {
		t.Fatalf("lifecycle timestamps lost after restart: %+v", ra)
	}
	if ra.Notes != "confirmed via auth chain" {
		t.Fatalf("notes lost: %q", ra.Notes)
	}
	rb, ok := s2.Get(b.ID)
	if !ok {
		t.Fatalf("alert %s missing after restart", b.ID)
	}
	if rb.State != "open" || rb.Severity != AlertSeverityCritical {
		t.Fatalf("untouched alert mutated by replay: %+v", rb)
	}

	// Recent order preserved: newest first.
	recent := s2.Recent(2)
	if len(recent) != 2 || recent[0].ID != b.ID || recent[1].ID != a.ID {
		t.Fatalf("recent order wrong after replay: %+v", recent)
	}

	// Third lifetime: mutations in lifetime 2 must persist too (reopen).
	if _, ok := s2.Reopen(a.ID, "carol"); !ok {
		t.Fatal("reopen failed")
	}
	_ = s2.Close()

	s3 := NewAlertService(logger)
	if err := s3.AttachJournal(dir); err != nil {
		t.Fatalf("third attach: %v", err)
	}
	defer s3.Close()
	ra3, ok := s3.Get(a.ID)
	if !ok || ra3.State != "open" || ra3.Verdict != "" {
		t.Fatalf("reopen not persisted: %+v", ra3)
	}
}

// TestAlertJournalNotAttached confirms the service still works purely
// in-memory when no journal is attached (unit-test / ephemeral usage).
func TestAlertJournalNotAttached(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	s := NewAlertService(logger)
	a := s.Raise(context.Background(), Alert{RuleID: "r-1", Message: "m"})
	if _, ok := s.Ack(a.ID, "alice"); !ok {
		t.Fatal("ack failed without journal")
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close without journal: %v", err)
	}
}
