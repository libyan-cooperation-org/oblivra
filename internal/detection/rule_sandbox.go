package detection

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/logger"
	"gopkg.in/yaml.v3"
)

// RuleSandbox provides a safe, offline environment for testing detection rules
// against sample events WITHOUT affecting the production detection pipeline.
// This is a DESKTOP-ONLY capability — operators use it to validate rules
// before promoting them to the Web (production) layer.
type RuleSandbox struct {
	log *logger.Logger
}

// SandboxResult captures the outcome of testing a single rule against sample events.
type SandboxResult struct {
	RuleName       string              `json:"rule_name"`
	RuleID         string              `json:"rule_id"`
	TotalEvents    int                 `json:"total_events"`
	Matched        int                 `json:"matched"`
	Missed         int                 `json:"missed"`
	FalsePositives int                 `json:"false_positives"`
	MatchRate      float64             `json:"match_rate"`
	Matches        []SandboxMatchEntry `json:"matches,omitempty"`
	Duration       time.Duration       `json:"duration_ns"`
	Errors         []string            `json:"errors,omitempty"`
}

// SandboxMatchEntry is a single match from a sandbox test run.
type SandboxMatchEntry struct {
	EventIndex int    `json:"event_index"`
	EventType  string `json:"event_type"`
	Host       string `json:"host,omitempty"`
}

// SimulationReport is the result of running multiple rules against a set of events.
type SimulationReport struct {
	TotalRules    int             `json:"total_rules"`
	TotalEvents   int             `json:"total_events"`
	TotalMatches  int             `json:"total_matches"`
	CoverageRate  float64         `json:"coverage_rate"`
	Results       []SandboxResult `json:"results"`
	Duration      time.Duration   `json:"duration_ns"`
	MITRECoverage map[string]bool `json:"mitre_coverage,omitempty"`
}

// NewRuleSandbox creates a new offline rule testing sandbox.
func NewRuleSandbox(log *logger.Logger) *RuleSandbox {
	return &RuleSandbox{
		log: log.WithPrefix("rule-sandbox"),
	}
}

// ValidateRuleSyntax validates a YAML rule definition for structural correctness
// without executing it against any events.
func (s *RuleSandbox) ValidateRuleSyntax(ruleYAML string) error {
	if strings.TrimSpace(ruleYAML) == "" {
		return fmt.Errorf("empty rule definition")
	}

	// Attempt to parse as a detection rule via the existing YAML loader
	var rule Rule
	if err := parseRuleYAMLBytes([]byte(ruleYAML), &rule); err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	if rule.Type == "" {
		return fmt.Errorf("rule type is required")
	}
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}

	return nil
}

// TestRule runs a single rule against a set of sample events and returns
// detailed match results. This is the core sandbox operation.
func (s *RuleSandbox) TestRule(rule Rule, sampleEvents []events.SovereignEvent) SandboxResult {
	start := time.Now()

	result := SandboxResult{
		RuleName:    rule.Name,
		RuleID:      rule.ID,
		TotalEvents: len(sampleEvents),
	}

	for i, event := range sampleEvents {
		matched := sandboxMatchEvent(rule, event)

		if matched {
			result.Matched++

			matchEntry := SandboxMatchEntry{
				EventIndex: i,
				EventType:  event.EventType,
				Host:       event.Host,
			}

			// If the event has a "benign" metadata tag, count as false positive
			if event.Metadata != nil {
				if tag, ok := event.Metadata["_tag"]; ok && strings.Contains(strings.ToLower(tag), "benign") {
					result.FalsePositives++
				}
			}

			result.Matches = append(result.Matches, matchEntry)
		}
	}

	result.Missed = result.TotalEvents - result.Matched
	if result.TotalEvents > 0 {
		result.MatchRate = float64(result.Matched) / float64(result.TotalEvents) * 100
	}
	result.Duration = time.Since(start)

	s.log.Info("[SANDBOX] Rule %q tested: %d/%d matched (%.1f%%), %d FP, took %v",
		rule.Name, result.Matched, result.TotalEvents, result.MatchRate,
		result.FalsePositives, result.Duration)

	return result
}

// SimulateDetection runs multiple rules against a set of events,
// producing a comprehensive simulation report with MITRE coverage.
func (s *RuleSandbox) SimulateDetection(rules []Rule, sampleEvents []events.SovereignEvent) SimulationReport {
	start := time.Now()

	report := SimulationReport{
		TotalRules:    len(rules),
		TotalEvents:   len(sampleEvents),
		MITRECoverage: make(map[string]bool),
	}

	eventMatched := make([]bool, len(sampleEvents))

	for _, rule := range rules {
		result := s.TestRule(rule, sampleEvents)
		report.Results = append(report.Results, result)
		report.TotalMatches += result.Matched

		for _, m := range result.Matches {
			if m.EventIndex < len(eventMatched) {
				eventMatched[m.EventIndex] = true
			}
		}

		for _, tech := range rule.MitreTechniques {
			report.MITRECoverage[tech] = true
		}
	}

	coveredCount := 0
	for _, matched := range eventMatched {
		if matched {
			coveredCount++
		}
	}
	if len(sampleEvents) > 0 {
		report.CoverageRate = float64(coveredCount) / float64(len(sampleEvents)) * 100
	}

	report.Duration = time.Since(start)

	s.log.Info("[SANDBOX] Simulation complete: %d rules × %d events = %d matches (%.1f%% coverage), %d MITRE techniques",
		len(rules), len(sampleEvents), report.TotalMatches, report.CoverageRate, len(report.MITRECoverage))

	return report
}

// ──────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────

// sandboxMatchEvent checks if a rule matches a SovereignEvent.
// The rule's Conditions map is matched against the event's typed fields.
func sandboxMatchEvent(rule Rule, event events.SovereignEvent) bool {
	// Build a flat map from the event's typed fields for condition matching
	eventMap := map[string]string{
		"EventType": event.EventType,
		"Host":      event.Host,
		"User":      event.User,
		"SourceIp":  event.SourceIp,
		"SessionId": event.SessionId,
		"RawLine":   event.RawLine,
		"TenantID":  event.TenantID,
	}
	// Include metadata as matchable fields
	if event.Metadata != nil {
		for k, v := range event.Metadata {
			eventMap[k] = v
		}
	}

	// Use the same matching logic as the production engine
	for field, expected := range rule.Conditions {
		expStr, ok := expected.(string)
		if !ok {
			continue
		}

		evtVal, exists := eventMap[field]
		if !exists {
			return false
		}

		// Support regex: prefix (same as production matchRuleConditions)
		if strings.HasPrefix(expStr, "regex:") {
			pattern := strings.TrimPrefix(expStr, "regex:")
			if !regexpMatch(pattern, evtVal) {
				return false
			}
		} else if !strings.EqualFold(evtVal, expStr) {
			return false
		}
	}

	return true
}

// regexpMatch is a safe regexp matcher that returns false on invalid patterns.
func regexpMatch(pattern, value string) bool {
	matched, err := regexp.MatchString(pattern, value)
	return err == nil && matched
}

// parseRuleYAMLBytes parses raw YAML bytes into a Rule struct.
func parseRuleYAMLBytes(data []byte, rule *Rule) error {
	return yaml.Unmarshal(data, rule)
}
