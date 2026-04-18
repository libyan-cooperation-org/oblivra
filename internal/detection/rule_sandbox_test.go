package detection

import (
	"os"
	"testing"

	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestRuleSandboxValidateSyntax(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	sandbox := NewRuleSandbox(log)

	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name:    "empty rule",
			yaml:    "",
			wantErr: true,
		},
		{
			name: "valid rule",
			yaml: `
id: test-rule-001
name: Test Rule
type: threshold
severity: high
conditions:
  EventType: failed_login
  User: root
`,
			wantErr: false,
		},
		{
			name: "missing ID",
			yaml: `
name: No ID Rule
type: threshold
conditions:
  EventType: failed_login
`,
			wantErr: true,
		},
		{
			name: "missing conditions",
			yaml: `
id: test-rule-002
name: No Conditions
type: threshold
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sandbox.ValidateRuleSyntax(tt.yaml)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRuleSyntax() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRuleSandboxTestRule(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	sandbox := NewRuleSandbox(log)

	rule := Rule{
		ID:       "brute-force-001",
		Name:     "Brute Force Detection",
		Type:     ThresholdRule,
		Severity: "high",
		Conditions: map[string]interface{}{
			"EventType": "failed_login",
			"User":      "root",
		},
		MitreTechniques: []string{"T1110"},
	}

	sampleEvents := []events.SovereignEvent{
		{EventType: "failed_login", User: "root", Host: "server-1"},
		{EventType: "failed_login", User: "admin", Host: "server-1"},
		{EventType: "successful_login", User: "root", Host: "server-2"},
		{EventType: "failed_login", User: "root", Host: "server-3"},
	}

	result := sandbox.TestRule(rule, sampleEvents)

	if result.TotalEvents != 4 {
		t.Errorf("Expected 4 total events, got %d", result.TotalEvents)
	}
	if result.Matched != 2 {
		t.Errorf("Expected 2 matches (events 0 and 3), got %d", result.Matched)
	}
	if result.Missed != 2 {
		t.Errorf("Expected 2 missed, got %d", result.Missed)
	}
	if result.MatchRate != 50.0 {
		t.Errorf("Expected 50%% match rate, got %.1f%%", result.MatchRate)
	}
}

func TestRuleSandboxSimulation(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	sandbox := NewRuleSandbox(log)

	rules := []Rule{
		{
			ID:       "rule-1",
			Name:     "SSH Brute Force",
			Type:     ThresholdRule,
			Conditions: map[string]interface{}{
				"EventType": "failed_login",
			},
			MitreTechniques: []string{"T1110"},
		},
		{
			ID:       "rule-2",
			Name:     "Suspicious PowerShell",
			Type:     ThresholdRule,
			Conditions: map[string]interface{}{
				"EventType": "process_create",
			},
			MitreTechniques: []string{"T1059.001"},
		},
	}

	sampleEvents := []events.SovereignEvent{
		{EventType: "failed_login", User: "root", Host: "server-1"},
		{EventType: "process_create", User: "admin", Host: "workstation-1"},
		{EventType: "network_connection", Host: "server-2"},
		{EventType: "failed_login", User: "admin", Host: "server-3"},
	}

	report := sandbox.SimulateDetection(rules, sampleEvents)

	if report.TotalRules != 2 {
		t.Errorf("Expected 2 rules, got %d", report.TotalRules)
	}
	if report.TotalMatches != 3 {
		t.Errorf("Expected 3 total matches, got %d", report.TotalMatches)
	}
	if report.CoverageRate != 75.0 {
		t.Errorf("Expected 75%% coverage (3 of 4 events matched), got %.1f%%", report.CoverageRate)
	}
	if len(report.MITRECoverage) != 2 {
		t.Errorf("Expected 2 MITRE techniques covered, got %d", len(report.MITRECoverage))
	}
	if !report.MITRECoverage["T1110"] {
		t.Error("Expected T1110 to be covered")
	}
	if !report.MITRECoverage["T1059.001"] {
		t.Error("Expected T1059.001 to be covered")
	}
}
