package sigma

import (
	"strings"
	"testing"

	"github.com/kingknull/oblivra/internal/services"
)

func TestParseSimpleSigma(t *testing.T) {
	raw := []byte(`
title: Suspicious PowerShell Download
id: sigma-ps-download
status: stable
level: high
logsource:
    product: windows
    category: process_creation
detection:
    selection:
        CommandLine|contains:
            - 'DownloadString'
            - 'Invoke-Expression'
    condition: selection
tags:
    - attack.execution
    - attack.t1059.001
`)
	rule, err := Parse(raw, "ps.yml")
	if err != nil {
		t.Fatal(err)
	}
	if rule.ID != "sigma-ps-download" {
		t.Errorf("id = %q", rule.ID)
	}
	if rule.Severity != services.AlertSeverityHigh {
		t.Errorf("severity = %q", rule.Severity)
	}
	if !contains(rule.AnyContain, "DownloadString") {
		t.Errorf("missing DownloadString in %v", rule.AnyContain)
	}
	if !contains(rule.MITRE, "T1059.001") {
		t.Errorf("missing T1059.001 in %v", rule.MITRE)
	}
	if rule.Source != "sigma" {
		t.Errorf("source = %q", rule.Source)
	}
}

func TestParseRejectsComplexCondition(t *testing.T) {
	raw := []byte(`
title: x
detection:
    selection:
        a: 1
    other:
        b: 2
    condition: selection or other
`)
	if _, err := Parse(raw, "x.yml"); err == nil {
		t.Fatal("expected error on multi-block condition")
	}
}

func TestParseRejectsEmptySelection(t *testing.T) {
	raw := []byte(`
title: x
detection:
    selection: {}
    condition: selection
`)
	if _, err := Parse(raw, "x.yml"); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty-selection error, got %v", err)
	}
}

// "1 of selection_*" — multi-block OR via glob pattern.
func TestParseOneOfPattern(t *testing.T) {
	raw := []byte(`
title: Defender disabled
id: cf-defender-stop
level: critical
detection:
    selection_ps:
        CommandLine|contains:
            - 'Set-MpPreference'
            - 'DisableRealtimeMonitoring'
    selection_reg:
        CommandLine|contains:
            - 'reg add'
            - 'Microsoft\Windows Defender\Real-Time Protection'
    condition: 1 of selection_*
tags:
    - attack.t1562.001
`)
	rule, err := Parse(raw, "cf-defender.yml")
	if err != nil {
		t.Fatal(err)
	}
	// Both blocks contribute to AnyContain.
	if !contains(rule.AnyContain, "Set-MpPreference") {
		t.Errorf("missing Set-MpPreference: %v", rule.AnyContain)
	}
	if !contains(rule.AnyContain, "reg add") {
		t.Errorf("missing 'reg add': %v", rule.AnyContain)
	}
	if rule.Severity != services.AlertSeverityCritical {
		t.Errorf("severity = %q", rule.Severity)
	}
}

// "1 of them" — every non-condition block unioned.
func TestParseOneOfThem(t *testing.T) {
	raw := []byte(`
title: Timestomp
detection:
    unix:
        CommandLine|contains:
            - 'touch -t '
            - 'touch -r '
    windows:
        CommandLine|contains:
            - '.CreationTime='
            - 'SetLastWriteTime'
    condition: 1 of them
`)
	rule, err := Parse(raw, "timestomp.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(rule.AnyContain, "touch -t ") {
		t.Errorf("missing unix needle")
	}
	if !contains(rule.AnyContain, "SetLastWriteTime") {
		t.Errorf("missing windows needle")
	}
}

// Numeric values (EventID arrays) should stringify, not silently drop.
func TestParseNumericValues(t *testing.T) {
	raw := []byte(`
title: Eventlog cleared
detection:
    selection:
        EventID:
            - 1102
            - 104
    condition: selection
`)
	rule, err := Parse(raw, "evt.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(rule.AnyContain, "1102") {
		t.Errorf("expected stringified 1102 in %v", rule.AnyContain)
	}
	if !contains(rule.AnyContain, "104") {
		t.Errorf("expected stringified 104 in %v", rule.AnyContain)
	}
}

// AND, OR, count-by, near, etc still error out (substring matcher
// can't honestly evaluate them).
func TestParseStillRejectsUnsupportedExprs(t *testing.T) {
	cases := []string{"all of them", "selection | count() > 5", "selection or other", "selection and other"}
	for _, c := range cases {
		raw := []byte("title: x\ndetection:\n  selection:\n    a: 1\n  other:\n    b: 2\n  condition: " + c + "\n")
		if _, err := Parse(raw, "x.yml"); err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}

// "selection and not exclude" — common Sigma shape. We translate it
// into AnyContain (positive union) + NotContain (negative union).
func TestParseAndNot(t *testing.T) {
	raw := []byte(`
title: Suspicious PowerShell minus dev fixtures
id: cf-ps-minus-dev
level: high
detection:
    selection:
        CommandLine|contains:
            - 'DownloadString'
            - 'Invoke-Expression'
    exclude:
        CommandLine|contains:
            - 'jenkins-fixture'
            - 'staging-canary'
    condition: selection and not exclude
`)
	rule, err := Parse(raw, "ps.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(rule.AnyContain, "DownloadString") {
		t.Errorf("missing positive token: %v", rule.AnyContain)
	}
	if !contains(rule.NotContain, "jenkins-fixture") {
		t.Errorf("missing negative token: %v", rule.NotContain)
	}
	if !contains(rule.NotContain, "staging-canary") {
		t.Errorf("missing second negative token: %v", rule.NotContain)
	}
}

// "1 of selection_* and not exclude" — combination of glob OR + negative.
func TestParseOneOfPatternAndNot(t *testing.T) {
	raw := []byte(`
title: Multi-method LSASS dump minus benign tools
id: cf-lsass-multi-minus
level: critical
detection:
    selection_proc:
        CommandLine|contains: 'lsass.exe'
    selection_dll:
        CommandLine|contains: 'comsvcs.dll'
    exclude:
        CommandLine|contains: 'procdumpsvc'
    condition: 1 of selection_* and not exclude
`)
	rule, err := Parse(raw, "lsass.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(rule.AnyContain, "lsass.exe") || !contains(rule.AnyContain, "comsvcs.dll") {
		t.Errorf("positive set incomplete: %v", rule.AnyContain)
	}
	if !contains(rule.NotContain, "procdumpsvc") {
		t.Errorf("negative not picked up: %v", rule.NotContain)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
