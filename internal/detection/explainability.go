package detection

// explainability.go — Structured detection explanation for every Match
//
// Closes the audit gap:
//   "Detection explainability — alerts fire but no 'why' log.
//    No structured reason why a rule triggered."
//
// Every Match now carries a WhyFired field with:
//   - RuleType (threshold / sequence / correlation / fusion / graph)
//   - Trigger reason in human-readable prose
//   - Contributing events summarised (not full dumps — avoids log bloat)
//   - Graph context if a graph edge triggered the match
//   - MITRE ATT&CK mapping
//   - Confidence signal (0.0–1.0)
//
// Explainability logs are emitted at INFO level so they appear in the
// detection explainability ring buffer surfaced by the API and the frontend
// DecisionInspector page.

import (
	"fmt"
	"strings"
	"time"
)

// ExplainedMatch extends Match with a structured explanation of why it fired.
// This is the type emitted on the "detection.explained_match" bus topic.
type ExplainedMatch struct {
	Match

	// WhyFired contains the structured explanation of the detection decision.
	WhyFired WhyFiredReason `json:"why_fired"`
}

// WhyFiredReason is the structured explanation attached to every detection match.
type WhyFiredReason struct {
	// RuleType categorises the detection mechanism that produced this match.
	RuleType RuleKind `json:"rule_type"`

	// Summary is a single human-readable sentence explaining the trigger.
	// Example: "User 'bob' failed login 6 times in 60s from 192.168.1.10 (threshold=5)"
	Summary string `json:"summary"`

	// Evidence is a compact list of contributing event fingerprints.
	// Each entry is "<EventType>@<HostID>(<Timestamp>)" — avoids raw log dumps.
	Evidence []string `json:"evidence"`

	// GroupKey is the correlation/grouping key that scoped this match
	// (e.g. "t:acme|ip:192.168.1.10").
	GroupKey string `json:"group_key"`

	// Threshold is the count threshold that was reached (for frequency rules).
	Threshold int `json:"threshold,omitempty"`

	// WindowSec is the time window the threshold was measured over.
	WindowSec int `json:"window_sec,omitempty"`

	// SequenceProgress describes which steps in a sequence were matched,
	// formatted as "Step N/M: <EventType>".
	SequenceProgress []string `json:"sequence_progress,omitempty"`

	// GraphEdges describes graph relationships that contributed to this match
	// (populated for graph-aware correlation rules).
	GraphEdges []string `json:"graph_edges,omitempty"`

	// MITREMapping summarises the ATT&CK tactics and techniques.
	MITREMapping []string `json:"mitre_mapping"`

	// Confidence is a 0.0–1.0 signal of how certain the engine is.
	// For threshold rules: min(count/threshold, 1.0).
	// For sequence rules: 1.0 (all steps completed).
	// For fusion engine: the Bayesian probability score.
	Confidence float64 `json:"confidence"`

	// ExplainedAt is the RFC3339 timestamp when the explanation was generated.
	ExplainedAt string `json:"explained_at"`
}

// RuleKind categorises the type of detection mechanism.
type RuleKind string

const (
	RuleKindThreshold   RuleKind = "threshold"
	RuleKindSequence    RuleKind = "sequence"
	RuleKindCorrelation RuleKind = "correlation"
	RuleKindFusion      RuleKind = "fusion_campaign"
	RuleKindGraph       RuleKind = "graph_edge"
	RuleKindSigma       RuleKind = "sigma"
)

// ExplainMatch builds an ExplainedMatch from a raw Match and its originating rule.
// Called from Evaluator.triggerAlert() so every fired rule produces an explanation.
func ExplainMatch(m Match, rule Rule, groupKey string, contributingEvents []Event) ExplainedMatch {
	kind := RuleKindThreshold
	if rule.Type == SequenceRule {
		kind = RuleKindSequence
	}

	evidence := buildEvidence(contributingEvents)

	var sequenceProgress []string
	if rule.Type == SequenceRule {
		for i, step := range rule.Sequence {
			label := fmt.Sprintf("Step %d/%d", i+1, len(rule.Sequence))
			for k := range step.Conditions {
				label += ": " + k
				break
			}
			sequenceProgress = append(sequenceProgress, label)
		}
	}

	// Human-readable summary
	summary := buildThresholdSummary(m, rule, groupKey, contributingEvents)
	if rule.Type == SequenceRule {
		summary = buildSequenceSummary(m, rule, groupKey, contributingEvents)
	}

	// Confidence: for threshold rules, how far over the threshold we are
	confidence := 1.0
	if rule.Type != SequenceRule && rule.Threshold > 1 {
		confidence = float64(len(contributingEvents)) / float64(rule.Threshold)
		if confidence > 1.0 {
			confidence = 1.0
		}
	}

	mitreMapping := buildMITREMapping(m)

	return ExplainedMatch{
		Match: m,
		WhyFired: WhyFiredReason{
			RuleType:         kind,
			Summary:          summary,
			Evidence:         evidence,
			GroupKey:         groupKey,
			Threshold:        rule.Threshold,
			WindowSec:        rule.WindowSec,
			SequenceProgress: sequenceProgress,
			MITREMapping:     mitreMapping,
			Confidence:       confidence,
			ExplainedAt:      time.Now().Format(time.RFC3339),
		},
	}
}

// ExplainCorrelationMatch builds an ExplainedMatch from a CorrelationMatch.
// Called from CorrelationEngine when a cross-source rule fires.
func ExplainCorrelationMatch(m CorrelationMatch, rule CrossSourceRule) ExplainedMatch {
	evidence := buildEvidence(m.ContributingEvt)

	summary := fmt.Sprintf(
		"Cross-source correlation fired: %q | Group: %s | Technique: %s | %d contributing events across %d required event types",
		m.RuleName, m.GroupKey, m.MitreTechnique,
		len(m.ContributingEvt), len(rule.Required),
	)

	var reqTypes []string
	for _, r := range rule.Required {
		reqTypes = append(reqTypes, r.EventType)
	}

	return ExplainedMatch{
		Match: Match{
			TenantID:        m.TenantID,
			RuleID:          m.RuleID,
			RuleName:        m.RuleName,
			Description:     m.Description,
			Severity:        m.Severity,
			MitreTechniques: []string{m.MitreTechnique},
			TriggeredAt:     m.TriggeredAt,
			Events:          m.ContributingEvt,
			Context:         map[string]string{"group_key": m.GroupKey},
		},
		WhyFired: WhyFiredReason{
			RuleType:    RuleKindCorrelation,
			Summary:     summary,
			Evidence:    evidence,
			GroupKey:    m.GroupKey,
			WindowSec:   rule.WindowSec,
			MITREMapping: []string{m.MitreTechnique},
			Confidence:  1.0, // All required conditions satisfied = 100% confidence
			ExplainedAt: time.Now().Format(time.RFC3339),
		},
	}
}

// ExplainFusionCampaign builds an ExplainedMatch from a fusion engine campaign trigger.
// Called from AttackFusionEngine.triggerFusionAlert().
func ExplainFusionCampaign(camp *Campaign) ExplainedMatch {
	tactics := make([]string, 0, len(camp.Tactics))
	for t := range camp.Tactics {
		tactics = append(tactics, t)
	}

	var alertSummaries []string
	for _, a := range camp.Alerts {
		alertSummaries = append(alertSummaries,
			fmt.Sprintf("%s[%s]@%s", a.Name, a.Tactic, a.Timestamp.Format("15:04:05")))
	}

	summary := fmt.Sprintf(
		"Multi-stage campaign detected for entity %q | %.0f%% confidence | %d distinct ATT&CK tactics: [%s] | %d alerts correlated",
		camp.EntityID,
		camp.Probability*100,
		len(camp.Tactics),
		strings.Join(tactics, ", "),
		len(camp.Alerts),
	)

	return ExplainedMatch{
		Match: Match{
			RuleID:          "FUSION_CAMPAIGN",
			RuleName:        "Multi-Stage Campaign",
			Description:     summary,
			Severity:        "CRITICAL",
			MitreTactics:    tactics,
			TriggeredAt:     camp.LastSeen.Format(time.RFC3339),
			Context: map[string]string{
				"entity_id":  camp.EntityID,
				"confidence": fmt.Sprintf("%.2f", camp.Probability),
				"stages":     fmt.Sprintf("%d", len(camp.Tactics)),
			},
		},
		WhyFired: WhyFiredReason{
			RuleType:    RuleKindFusion,
			Summary:     summary,
			Evidence:    alertSummaries,
			GroupKey:    camp.EntityID,
			MITREMapping: tactics,
			Confidence:  camp.Probability,
			ExplainedAt: time.Now().Format(time.RFC3339),
		},
	}
}

// ExplainGraphEdgeMatch builds an ExplainedMatch for a graph-triggered detection.
func ExplainGraphEdgeMatch(entityFrom, entityTo, edgeType, tactic, tenantID string, confidence float64) ExplainedMatch {
	summary := fmt.Sprintf(
		"Graph edge detected: %s -[%s]→ %s | ATT&CK tactic: %s | confidence: %.0f%%",
		entityFrom, edgeType, entityTo, tactic, confidence*100,
	)

	return ExplainedMatch{
		Match: Match{
			TenantID:        tenantID,
			RuleID:          "GRAPH_EDGE_" + strings.ToUpper(edgeType),
			RuleName:        "Graph Edge Detection",
			Description:     summary,
			Severity:        "HIGH",
			MitreTactics:    []string{tactic},
			TriggeredAt:     time.Now().Format(time.RFC3339),
			Context: map[string]string{
				"from":      entityFrom,
				"to":        entityTo,
				"edge_type": edgeType,
			},
		},
		WhyFired: WhyFiredReason{
			RuleType: RuleKindGraph,
			Summary:  summary,
			GraphEdges: []string{
				fmt.Sprintf("%s -[%s]→ %s", entityFrom, edgeType, entityTo),
			},
			MITREMapping: []string{tactic},
			Confidence:   confidence,
			ExplainedAt:  time.Now().Format(time.RFC3339),
		},
	}
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func buildEvidence(events []Event) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(events))
	for _, e := range events {
		key := fmt.Sprintf("%s@%s(%s)", e.EventType, e.HostID, e.Timestamp)
		if !seen[key] {
			out = append(out, key)
			seen[key] = true
		}
	}
	return out
}

func buildThresholdSummary(m Match, rule Rule, groupKey string, events []Event) string {
	// Extract the most-common host and user from contributing events
	hosts := mostCommon(events, func(e Event) string { return e.HostID })
	users := mostCommon(events, func(e Event) string { return e.User })
	ips := mostCommon(events, func(e Event) string { return e.SourceIP })

	parts := []string{
		fmt.Sprintf("Rule %q fired", m.RuleName),
		fmt.Sprintf("severity=%s", m.Severity),
		fmt.Sprintf("count=%d threshold=%d", len(events), rule.Threshold),
	}
	if rule.WindowSec > 0 {
		parts = append(parts, fmt.Sprintf("window=%ds", rule.WindowSec))
	}
	if hosts != "" {
		parts = append(parts, "host="+hosts)
	}
	if users != "" {
		parts = append(parts, "user="+users)
	}
	if ips != "" {
		parts = append(parts, "src_ip="+ips)
	}
	parts = append(parts, "group="+groupKey)
	return strings.Join(parts, " | ")
}

func buildSequenceSummary(m Match, rule Rule, groupKey string, events []Event) string {
	steps := make([]string, 0, len(rule.Sequence))
	for i, step := range rule.Sequence {
		for k := range step.Conditions {
			steps = append(steps, fmt.Sprintf("Step%d:%s", i+1, k))
			break
		}
	}
	return fmt.Sprintf(
		"Sequence rule %q completed: [%s] | %d events | group=%s",
		m.RuleName, strings.Join(steps, "→"), len(events), groupKey,
	)
}

func buildMITREMapping(m Match) []string {
	var out []string
	for _, t := range m.MitreTactics {
		if t != "" {
			out = append(out, "Tactic:"+t)
		}
	}
	for _, t := range m.MitreTechniques {
		if t != "" {
			out = append(out, "Technique:"+t)
		}
	}
	return out
}

// mostCommon returns the most-frequent non-empty value of field fn across events.
func mostCommon(events []Event, fn func(Event) string) string {
	counts := make(map[string]int)
	for _, e := range events {
		v := fn(e)
		if v != "" {
			counts[v]++
		}
	}
	best, bestCount := "", 0
	for v, c := range counts {
		if c > bestCount {
			best, bestCount = v, c
		}
	}
	return best
}
