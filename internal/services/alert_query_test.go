package services

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func newQueryFixture(t *testing.T) *AlertService {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	s := NewAlertService(logger)
	ctx := context.Background()
	now := time.Now().UTC()

	// 3× brute-force on web-01 (high), 1 critical tamper on dc-01,
	// 1 low anomaly on db-01. One resolved, one assigned.
	for i := 0; i < 3; i++ {
		s.Raise(ctx, Alert{RuleID: "r-brute", RuleName: "ssh brute force", Severity: AlertSeverityHigh,
			HostID: "web-01", Message: "Failed password burst", Triggered: now.Add(-time.Duration(i) * time.Hour)})
	}
	tamper := s.Raise(ctx, Alert{RuleID: "r-tamper", RuleName: "eventlog cleared", Severity: AlertSeverityCritical,
		HostID: "dc-01", Message: "wevtutil cl Security", Triggered: now.Add(-30 * time.Minute)})
	old := s.Raise(ctx, Alert{RuleID: "r-anom", RuleName: "new pattern", Severity: AlertSeverityLow,
		HostID: "db-01", Message: "unseen template", Triggered: now.Add(-40 * 24 * time.Hour)}) // outside 7d

	if _, ok := s.Assign(tamper.ID, "alice", "bob"); !ok {
		t.Fatal("assign failed")
	}
	if _, ok := s.Ack(old.ID, "alice"); !ok {
		t.Fatal("ack failed")
	}
	if _, ok := s.Resolve(old.ID, "alice", "false-positive", ""); !ok {
		t.Fatal("resolve failed")
	}
	return s
}

func TestAlertQueryFilters(t *testing.T) {
	s := newQueryFixture(t)

	if got := s.Query(AlertQuery{State: "open"}); len(got) != 3 {
		t.Errorf("state=open → %d, want 3", len(got))
	}
	if got := s.Query(AlertQuery{Severity: "high,critical"}); len(got) != 4 {
		t.Errorf("severity multi → %d, want 4", len(got))
	}
	if got := s.Query(AlertQuery{HostID: "dc-01"}); len(got) != 1 || got[0].RuleID != "r-tamper" {
		t.Errorf("hostId filter → %+v", got)
	}
	if got := s.Query(AlertQuery{RuleID: "r-brute"}); len(got) != 3 {
		t.Errorf("ruleId filter → %d, want 3", len(got))
	}
	if got := s.Query(AlertQuery{AssignedTo: "bob"}); len(got) != 1 {
		t.Errorf("assignedTo filter → %d, want 1", len(got))
	}
	if got := s.Query(AlertQuery{Q: "WEVTUTIL"}); len(got) != 1 { // case-insensitive, matches message
		t.Errorf("q filter → %d, want 1", len(got))
	}
	if got := s.Query(AlertQuery{Q: "brute"}); len(got) != 3 { // matches ruleName
		t.Errorf("q rule-name filter → %d, want 3", len(got))
	}
	from := time.Now().UTC().Add(-2 * 24 * time.Hour)
	if got := s.Query(AlertQuery{From: from}); len(got) != 4 { // excludes the 40d-old one
		t.Errorf("from filter → %d, want 4", len(got))
	}
	if got := s.Query(AlertQuery{Limit: 2}); len(got) != 2 {
		t.Errorf("limit → %d, want 2", len(got))
	}
	// newest first
	all := s.Query(AlertQuery{})
	for i := 1; i < len(all); i++ {
		if all[i].Triggered.After(all[i-1].Triggered) {
			t.Fatalf("not newest-first at %d", i)
		}
	}
	// resolved-state back-compat: "closed" filter finds resolved alerts
	if got := s.Query(AlertQuery{State: "closed"}); len(got) != 1 {
		t.Errorf("state=closed synonym → %d, want 1", len(got))
	}
}

func TestAlertMetrics(t *testing.T) {
	s := newQueryFixture(t)
	m := s.Metrics(7 * 24 * time.Hour)

	if m.Total != 4 { // 40d-old alert excluded from the 7d window
		t.Errorf("total = %d, want 4", m.Total)
	}
	if m.ByState["open"] != 3 || m.ByState["assigned"] != 1 {
		t.Errorf("byState = %+v", m.ByState)
	}
	if m.BySeverity["high"] != 3 || m.BySeverity["critical"] != 1 {
		t.Errorf("bySeverity = %+v", m.BySeverity)
	}
	if len(m.ByRule) != 2 || m.ByRule[0].RuleID != "r-brute" || m.ByRule[0].Count != 3 {
		t.Errorf("byRule = %+v", m.ByRule)
	}
	// tamper alert was assigned (implies ack) ~now, triggered 30min ago →
	// MTTA around 1800s. Allow generous slack for test scheduling.
	if m.MeanTimeToAckSecs < 1700 || m.MeanTimeToAckSecs > 1900 {
		t.Errorf("MTTA = %.0fs, want ~1800s", m.MeanTimeToAckSecs)
	}
	// nothing resolved inside the window
	if m.MeanTimeToResolveSecs != 0 {
		t.Errorf("MTTR = %.0fs, want 0 (no resolves in window)", m.MeanTimeToResolveSecs)
	}

	// widen the window: the resolved 40d-old alert enters MTTR
	m = s.Metrics(60 * 24 * time.Hour)
	if m.Total != 5 || m.MeanTimeToResolveSecs <= 0 {
		t.Errorf("60d window: total=%d mttr=%.0f", m.Total, m.MeanTimeToResolveSecs)
	}
}
