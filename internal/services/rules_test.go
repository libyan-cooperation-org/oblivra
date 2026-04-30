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
