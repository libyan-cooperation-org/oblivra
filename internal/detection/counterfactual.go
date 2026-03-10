package detection

import (
	"fmt"
	"time"
)

// CounterfactualResult captures the delta between original and simulated detection outcomes.
type CounterfactualResult struct {
	ID                    string    `json:"id"`
	Timestamp             time.Time `json:"timestamp"`
	EventCount            int       `json:"event_count"`
	DisabledRules         []string  `json:"disabled_rules"`
	OriginalMatches       []Match   `json:"original_matches"`
	SimulatedMatches      []Match   `json:"simulated_matches"`
	MissedDetections      []Match   `json:"missed_detections"`
	FalsePositivesRemoved []Match   `json:"false_positives_removed"`
	ImpactSummary         string    `json:"impact_summary"`
}

// CounterfactualEngine replays events against modified rule sets to analyze detection impact.
// This is a pure function engine — zero side effects, no state mutation.
type CounterfactualEngine struct {
	evaluator *Evaluator
}

// NewCounterfactualEngine creates a new counterfactual simulator.
func NewCounterfactualEngine(evaluator *Evaluator) *CounterfactualEngine {
	return &CounterfactualEngine{
		evaluator: evaluator,
	}
}

// RunSimulation replays a batch of events with the specified rules disabled.
// Returns a diff showing what would have been missed vs. what would have been prevented.
func (e *CounterfactualEngine) RunSimulation(events []Event, disabledRuleIDs []string) *CounterfactualResult {
	result := &CounterfactualResult{
		ID:            fmt.Sprintf("cf-%d", time.Now().UnixNano()),
		Timestamp:     time.Now(),
		EventCount:    len(events),
		DisabledRules: disabledRuleIDs,
	}

	if e.evaluator == nil || e.evaluator.RuleEngine == nil {
		result.ImpactSummary = "No detection evaluator available."
		return result
	}

	// Build disabled rule set for O(1) lookup
	disabled := make(map[string]bool, len(disabledRuleIDs))
	for _, id := range disabledRuleIDs {
		disabled[id] = true
	}

	// Phase 1: Run original detection (all rules)
	originalMatchesByEvent := make(map[int][]Match)
	for i, evt := range events {
		matches := e.evaluator.ProcessEvent(evt)
		if len(matches) > 0 {
			originalMatchesByEvent[i] = matches
			result.OriginalMatches = append(result.OriginalMatches, matches...)
		}
	}

	// Phase 2: Run simulated detection (disabled rules filtered out)
	for i, evt := range events {
		matches := e.evaluator.ProcessEvent(evt)

		var filtered []Match
		for _, m := range matches {
			if !disabled[m.RuleID] {
				filtered = append(filtered, m)
			}
		}

		if len(filtered) > 0 {
			result.SimulatedMatches = append(result.SimulatedMatches, filtered...)
		}

		// Calculate diff for this event
		orig := originalMatchesByEvent[i]
		for _, om := range orig {
			if disabled[om.RuleID] {
				result.MissedDetections = append(result.MissedDetections, om)
			}
		}
	}

	// Calculate false positives removed (matches that appeared in original but not simulated)
	simMatchSet := make(map[string]bool)
	for _, m := range result.SimulatedMatches {
		simMatchSet[m.RuleID+":"+m.TriggeredAt.String()] = true
	}
	for _, m := range result.OriginalMatches {
		key := m.RuleID + ":" + m.TriggeredAt.String()
		if !simMatchSet[key] {
			result.FalsePositivesRemoved = append(result.FalsePositivesRemoved, m)
		}
	}

	// Generate impact summary
	result.ImpactSummary = fmt.Sprintf(
		"Replayed %d events. Original: %d detections. Simulated (with %d rules disabled): %d detections. "+
			"Missed: %d detections. False positives removed: %d.",
		len(events),
		len(result.OriginalMatches),
		len(disabledRuleIDs),
		len(result.SimulatedMatches),
		len(result.MissedDetections),
		len(result.FalsePositivesRemoved),
	)

	return result
}

// AnalyzeRuleImpact runs a focused simulation for a single rule to determine its detection impact.
func (e *CounterfactualEngine) AnalyzeRuleImpact(events []Event, ruleID string) map[string]interface{} {
	result := e.RunSimulation(events, []string{ruleID})

	return map[string]interface{}{
		"rule_id":                ruleID,
		"events_analyzed":        len(events),
		"detections_by_rule":     len(result.MissedDetections),
		"total_original":         len(result.OriginalMatches),
		"total_without_rule":     len(result.SimulatedMatches),
		"detection_coverage_pct": coveragePct(len(result.OriginalMatches), len(result.MissedDetections)),
		"impact_summary":         result.ImpactSummary,
	}
}

func coveragePct(total, missed int) float64 {
	if total == 0 {
		return 100.0
	}
	return float64(total-missed) / float64(total) * 100.0
}
