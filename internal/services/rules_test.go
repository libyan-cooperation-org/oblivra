package services

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/kingknull/oblivra/internal/events"
)

func TestSshBruteforceRuleFires(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	rules := NewRulesService(logger, alerts)

	ev := events.Event{
		ID: "x", TenantID: "default", Source: events.SourceSyslog,
		Severity: events.SeverityWarn, HostID: "web-01",
		Message: "sshd Failed password for root from 10.0.0.1",
	}
	rules.Evaluate(context.Background(), ev)

	got := alerts.Recent(10)
	if len(got) == 0 {
		t.Fatalf("no alerts raised")
	}
	if got[0].RuleID != "builtin-ssh-bruteforce" {
		t.Errorf("rule = %q", got[0].RuleID)
	}
	if got[0].Severity != AlertSeverityHigh {
		t.Errorf("sev = %q", got[0].Severity)
	}
	if !contains(got[0].MITRE, "T1110.001") {
		t.Errorf("mitre missing T1110.001: %v", got[0].MITRE)
	}
}

func TestLsassRuleRequiresAllContain(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	rules := NewRulesService(logger, alerts)

	// Not a match — no "lsass" string anywhere.
	ev := events.Event{
		ID: "x", TenantID: "default", HostID: "h",
		Message: "comsvcs.dll loaded",
	}
	rules.Evaluate(context.Background(), ev)
	if got := alerts.Recent(10); len(got) > 0 {
		t.Errorf("unexpected alert from non-lsass event: %v", got[0])
	}

	// Match — has both lsass and an OR token.
	ev2 := events.Event{
		ID: "y", TenantID: "default", HostID: "dc-01",
		Message: "comsvcs.dll MiniDump on lsass.exe",
	}
	rules.Evaluate(context.Background(), ev2)
	if got := alerts.Recent(10); len(got) == 0 {
		t.Fatalf("expected alert for lsass + comsvcs.dll match")
	}
}

func TestRuleListAndReload(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	rules := NewRulesService(logger, alerts)

	if got := len(rules.List()); got < 5 {
		t.Errorf("expected at least 5 builtin rules, got %d", got)
	}
	if n, err := rules.Reload(); err != nil || n < 5 {
		t.Errorf("reload n=%d err=%v", n, err)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if strings.EqualFold(x, want) {
			return true
		}
	}
	return false
}

type testWriter struct{}

func (testWriter) Write(p []byte) (int, error) { return len(p), nil }

// Threshold rule fires only after N matching events in the window,
// not on the first one. Re-arm gate suppresses runs.
func TestThresholdRule_FiresOnNthHit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	rules := NewRulesService(logger, alerts)
	rules.rules = []Rule{{
		ID: "test-threshold", Name: "Threshold test", Severity: AlertSeverityHigh,
		Type: RuleTypeThreshold, Threshold: 3, Window: 60_000_000_000, // 60s
		GroupBy: "hostId",
		Fields: []string{"message"}, AnyContain: []string{"failed login"},
	}}

	ev := events.Event{HostID: "web-01", Message: "user x failed login"}

	// First two: no alert.
	rules.Evaluate(context.Background(), ev)
	rules.Evaluate(context.Background(), ev)
	if got := len(alerts.Recent(10)); got != 0 {
		t.Errorf("threshold should not fire before N hits, got %d alerts", got)
	}
	// Third: fire.
	rules.Evaluate(context.Background(), ev)
	if got := alerts.Recent(10); len(got) != 1 {
		t.Fatalf("expected 1 alert after threshold reached, got %d", len(got))
	}
	// Fourth (within re-arm gate): silent.
	rules.Evaluate(context.Background(), ev)
	if got := len(alerts.Recent(10)); got != 1 {
		t.Errorf("re-arm gate should suppress; got %d alerts", got)
	}
}

// Threshold rule per-host: hits on different hosts should NOT collapse
// into a single bucket — each gets its own count.
func TestThresholdRule_PerHostBuckets(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	rules := NewRulesService(logger, alerts)
	rules.rules = []Rule{{
		ID: "test-threshold-host", Name: "Per-host", Severity: AlertSeverityMedium,
		Type: RuleTypeThreshold, Threshold: 3, Window: 60_000_000_000,
		GroupBy: "hostId",
		Fields: []string{"message"}, AnyContain: []string{"x"},
	}}

	for i := 0; i < 2; i++ {
		rules.Evaluate(context.Background(), events.Event{HostID: "a", Message: "x"})
		rules.Evaluate(context.Background(), events.Event{HostID: "b", Message: "x"})
	}
	// Each host has 2 hits — neither at threshold.
	if got := len(alerts.Recent(10)); got != 0 {
		t.Fatalf("no host should have hit threshold yet, got %d", got)
	}
	// Push host a to 3.
	rules.Evaluate(context.Background(), events.Event{HostID: "a", Message: "x"})
	if got := len(alerts.Recent(10)); got != 1 {
		t.Errorf("expected one fire for host a, got %d", got)
	}
}

// Frequency rule: fires when N distinct values seen, not N total events.
func TestFrequencyRule_DistinctCardinality(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	rules := NewRulesService(logger, alerts)
	rules.rules = []Rule{{
		ID: "test-freq", Name: "Frequency", Severity: AlertSeverityHigh,
		Type: RuleTypeFrequency, Threshold: 3, Window: 60_000_000_000,
		GroupBy:    "hostId", // bucket: per host
		DistinctOf: "srcIP",  // cardinality: distinct source IPs
		Fields:     []string{"message"}, AnyContain: []string{"login"},
	}}

	mk := func(srcIP string) events.Event {
		return events.Event{HostID: "h", Message: "login attempt", Fields: map[string]string{"srcIP": srcIP}}
	}

	// 5 events from same IP → 1 distinct → no fire.
	for i := 0; i < 5; i++ {
		rules.Evaluate(context.Background(), mk("1.2.3.4"))
	}
	if got := len(alerts.Recent(10)); got != 0 {
		t.Errorf("same IP repeated should not fire frequency rule, got %d", got)
	}
	// 3rd distinct IP → fire.
	rules.Evaluate(context.Background(), mk("1.2.3.5"))
	rules.Evaluate(context.Background(), mk("1.2.3.6"))
	if got := alerts.Recent(10); len(got) != 1 {
		t.Errorf("expected 1 fire after 3 distinct IPs, got %d", len(got))
	}
}

// Sequence rule: fires only when steps observed in order within window.
func TestSequenceRule_InOrderOnly(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	rules := NewRulesService(logger, alerts)
	rules.rules = []Rule{{
		ID: "test-seq", Name: "Sequence", Severity: AlertSeverityCritical,
		Type: RuleTypeSequence, Window: 60_000_000_000,
		Sequence: []string{"failed password", "accepted publickey"},
		GroupBy:  "hostId",
		Fields:   []string{"message"}, AnyContain: []string{"sshd"},
	}}

	// Out of order — accepted first, no match for step 0.
	rules.Evaluate(context.Background(), events.Event{HostID: "h", Message: "sshd accepted publickey for x"})
	if got := len(alerts.Recent(10)); got != 0 {
		t.Errorf("step-0 mismatch should not fire, got %d", got)
	}
	// Step 0.
	rules.Evaluate(context.Background(), events.Event{HostID: "h", Message: "sshd failed password for x"})
	if got := len(alerts.Recent(10)); got != 0 {
		t.Errorf("partial sequence should not fire, got %d", got)
	}
	// Step 1 — fire.
	rules.Evaluate(context.Background(), events.Event{HostID: "h", Message: "sshd accepted publickey for x"})
	if got := alerts.Recent(10); len(got) != 1 {
		t.Errorf("expected 1 fire after sequence completes, got %d", len(got))
	}
}

// NotContain — negative gate that suppresses an otherwise-matching event.
func TestNotContainGate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	rules := NewRulesService(logger, alerts)
	rules.rules = []Rule{{
		ID: "test-not", Name: "Suppression", Severity: AlertSeverityMedium,
		Fields: []string{"message"},
		AnyContain: []string{"login"},
		NotContain: []string{"during deploy"},
	}}

	// "login" matches, but "during deploy" suppresses.
	rules.Evaluate(context.Background(), events.Event{HostID: "h", Message: "test login during deploy"})
	if got := len(alerts.Recent(10)); got != 0 {
		t.Errorf("NotContain should have suppressed; got %d alerts", got)
	}
	// "login" only — fires.
	rules.Evaluate(context.Background(), events.Event{HostID: "h", Message: "user x login ok"})
	if got := len(alerts.Recent(10)); got != 1 {
		t.Errorf("expected 1 fire when NotContain doesn't match; got %d", got)
	}
}
