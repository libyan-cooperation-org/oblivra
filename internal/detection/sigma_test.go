package detection

import (
	"strings"
	"testing"
)

const sigmaTestRuleMimikatz = `
title: Mimikatz Detection via LSASS Access
id: 5e5a66b2-2c5e-4b28-9f63-7e42c3b3f1a7
status: stable
description: Detects Mimikatz credential dumping via LSASS memory access
level: high
tags:
  - attack.credential-access
  - attack.t1003
  - attack.t1003.001
logsource:
  category: process_creation
  product: windows
detection:
  selection:
    CommandLine|contains:
      - sekurlsa
      - lsadump
      - privilege::debug
  condition: selection
timeframe: 5m
falsepositives:
  - Legitimate security tools
references:
  - https://github.com/gentilkiwi/mimikatz
`

const sigmaTestRuleSSH = `
title: SSH Brute Force
id: aaa11111-0000-0000-0000-000000000001
status: stable
description: Detects SSH brute force attacks
level: medium
tags:
  - attack.initial-access
  - attack.t1110
logsource:
  service: sshd
  product: linux
detection:
  keywords:
    - Failed password
    - Invalid user
  condition: keywords
timeframe: 10m
`

const sigmaTestRuleDeprecated = `
title: Old Rule
status: deprecated
description: This is deprecated
detection:
  selection:
    CommandLine|contains: test
  condition: selection
`

func TestTranspileSigma_Mimikatz(t *testing.T) {
	rule, err := TranspileSigma([]byte(sigmaTestRuleMimikatz))
	if err != nil {
		t.Fatalf("unexpected transpile error: %v", err)
	}

	if rule.ID != "sigma-5e5a66b2-2c5e-4b28-9f63-7e42c3b3f1a7" {
		t.Errorf("wrong ID: %s", rule.ID)
	}
	if rule.Severity != "high" {
		t.Errorf("expected severity high, got %s", rule.Severity)
	}
	if rule.WindowSec != 300 {
		t.Errorf("expected window_sec 300 (5m), got %d", rule.WindowSec)
	}

	// Should have a credential-access tactic
	foundTactic := false
	for _, ta := range rule.MitreTactics {
		if ta == "TA0006" {
			foundTactic = true
		}
	}
	if !foundTactic {
		t.Errorf("expected TA0006 (credential-access) in tactics, got %v", rule.MitreTactics)
	}

	// Should have T1003 technique
	foundTech := false
	for _, tech := range rule.MitreTechniques {
		if tech == "T1003" || tech == "T1003.001" {
			foundTech = true
		}
	}
	if !foundTech {
		t.Errorf("expected T1003 in techniques, got %v", rule.MitreTechniques)
	}

	// Conditions should include output_contains with a regex
	condInterface, ok := rule.Conditions["output_contains"]
	if !ok {
		t.Fatal("expected output_contains condition")
	}
	cond := condInterface.(string)
	if !strings.HasPrefix(cond, "regex:") {
		t.Errorf("expected regex: prefix in condition, got: %s", cond)
	}
	if !strings.Contains(cond, "sekurlsa") {
		t.Errorf("expected 'sekurlsa' in condition pattern, got: %s", cond)
	}

	// EventType should be process_creation from logsource
	if rule.Conditions["EventType"].(string) != "process_creation" {
		t.Errorf("expected EventType=process_creation, got: %v", rule.Conditions["EventType"])
	}
}

func TestTranspileSigma_Keywords(t *testing.T) {
	rule, err := TranspileSigma([]byte(sigmaTestRuleSSH))
	if err != nil {
		t.Fatalf("unexpected transpile error: %v", err)
	}

	if rule.Severity != "medium" {
		t.Errorf("expected medium severity, got %s", rule.Severity)
	}

	// Keywords should map to output_contains
	condInterface, ok := rule.Conditions["output_contains"]
	if !ok {
		t.Fatal("expected output_contains for keyword rule")
	}
	cond := condInterface.(string)
	if !strings.Contains(cond, "Failed") && !strings.Contains(cond, "failed") {
		t.Errorf("expected 'Failed' keyword in condition, got: %s", cond)
	}

	// logsource service=sshd should produce EventType=sshd
	if et, ok := rule.Conditions["EventType"]; ok {
		if et.(string) != "sshd" {
			t.Errorf("expected EventType=sshd, got %s", et)
		}
	}

	// GroupBy should include source_ip for SSH rules
	if len(rule.GroupBy) == 0 {
		t.Error("expected GroupBy to be set for SSH rule")
	}
}

func TestTranspileSigma_Deprecated(t *testing.T) {
	_, err := TranspileSigma([]byte(sigmaTestRuleDeprecated))
	if err == nil {
		t.Fatal("expected error for deprecated rule, got nil")
	}
	if !strings.Contains(err.Error(), "deprecated") {
		t.Errorf("expected 'deprecated' in error, got: %v", err)
	}
}

func TestTranspileSigma_MissingTitle(t *testing.T) {
	_, err := TranspileSigma([]byte(`
detection:
  selection:
    CommandLine|contains: test
  condition: selection
`))
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestTranspileSigma_MissingCondition(t *testing.T) {
	_, err := TranspileSigma([]byte(`
title: No Condition Rule
detection:
  selection:
    CommandLine|contains: test
`))
	if err == nil {
		t.Fatal("expected error for missing condition")
	}
}

func TestParseSigmaTimeframe(t *testing.T) {
	cases := []struct {
		input    string
		expected int
	}{
		{"5m", 300},
		{"1h", 3600},
		{"30s", 30},
		{"2d", 172800},
		{"", 0},
	}
	for _, c := range cases {
		got := parseSigmaTimeframe(c.input)
		if got != c.expected {
			t.Errorf("parseSigmaTimeframe(%q) = %d, want %d", c.input, got, c.expected)
		}
	}
}

func TestParseSigmaMitreTags(t *testing.T) {
	tags := []string{
		"attack.initial-access",
		"attack.t1110",
		"attack.t1003.001",
		"attack.defense-evasion",
		"cve.2021-44228", // should be ignored
	}
	tactics, techniques := parseSigmaMitreTags(tags)

	if len(tactics) != 2 {
		t.Errorf("expected 2 tactics, got %d: %v", len(tactics), tactics)
	}
	if len(techniques) != 2 {
		t.Errorf("expected 2 techniques, got %d: %v", len(techniques), techniques)
	}

	found := func(slice []string, val string) bool {
		for _, s := range slice {
			if s == val {
				return true
			}
		}
		return false
	}

	if !found(tactics, "TA0001") {
		t.Errorf("expected TA0001 in tactics: %v", tactics)
	}
	if !found(techniques, "T1110") {
		t.Errorf("expected T1110 in techniques: %v", techniques)
	}
	if !found(techniques, "T1003.001") {
		t.Errorf("expected T1003.001 in techniques: %v", techniques)
	}
}

func TestTranspileSigma_CountBy(t *testing.T) {
	yaml := []byte(`
title: Brute Force Detection
id: bf-001
level: high
logsource:
  category: authentication
detection:
  selection:
    EventType: 'failed_login'
  condition: selection | count() by SourceIp > 5
timeframe: 5m
`)
	rule, err := TranspileSigma(yaml)
	if err != nil {
		t.Fatalf("TranspileSigma: %v", err)
	}
	if rule.Type != FrequencyRule {
		t.Errorf("expected FrequencyRule, got %v", rule.Type)
	}
	if rule.Threshold != 5 {
		t.Errorf("expected threshold 5, got %d", rule.Threshold)
	}
	if len(rule.GroupBy) == 0 || rule.GroupBy[0] != "source_ip" {
		t.Errorf("expected GroupBy=[source_ip], got %v", rule.GroupBy)
	}
}

func TestTranspileSigma_CountByNoField(t *testing.T) {
	yaml := []byte(`
title: Mass File Rename
id: mfr-001
level: critical
logsource:
  category: file_event
detection:
  selection:
    EventType: 'mass_rename'
  condition: selection | count() > 100
timeframe: 1m
`)
	rule, err := TranspileSigma(yaml)
	if err != nil {
		t.Fatalf("TranspileSigma: %v", err)
	}
	if rule.Type != FrequencyRule {
		t.Errorf("expected FrequencyRule, got %v", rule.Type)
	}
	if rule.Threshold != 100 {
		t.Errorf("expected threshold 100, got %d", rule.Threshold)
	}
}
