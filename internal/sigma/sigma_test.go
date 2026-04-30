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

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
