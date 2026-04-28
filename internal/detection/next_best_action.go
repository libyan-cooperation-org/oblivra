// Package detection — Next-Best-Action recommender (Phase 32).
//
// Given an alert, produces a structured RecommendedAction the UI can
// surface as the pre-highlighted button in the triage drawer. The
// recommender is intentionally rule-based (not ML) for three reasons:
//
//  1. Operators must be able to defend "why this action" in audit. ML
//     scores from a model trained six months ago are unauditable.
//  2. The rules are tiny and fast — no model load, no GPU, runs in
//     <1ms per alert at the REST handler tier.
//  3. The output is stable and predictable across deploys. Rule
//     changes ship in the binary; nothing drifts silently.
//
// The recommender ingests a small structured fact-set (the alert plus
// optional host / process / IOC context) and emits:
//
//   {
//     action:       "quarantine_host" | "evidence_capture" | …,
//     confidence:   0.0–1.0,
//     reason:       human-readable string the UI shows as a tooltip,
//     alternatives: ranked list of next-best fallbacks
//   }
//
// Confidence is a SUM-of-evidence score capped at 1.0 — not a calibrated
// probability. The UI surfaces 0.6+ as "Recommended" and below as
// "Suggested".
package detection

import (
	"sort"
	"strings"
)

// RecommendedAction describes the operator's recommended next step.
type RecommendedAction struct {
	Action       string   `json:"action"`
	Confidence   float64  `json:"confidence"`
	Reason       string   `json:"reason"`
	Alternatives []string `json:"alternatives"`
}

// NBAFacts is the fact-set the recommender consumes. Keep this struct
// small — every additional field becomes a contract the alert pipeline
// has to populate. Optional fields default to zero-values.
type NBAFacts struct {
	// Required.
	AlertID  string
	Severity string // critical | high | medium | low | info
	Category string // mitre tactic, e.g. "initial-access", "execution"

	// Optional context — empty fields contribute zero score.
	HasIOCMatch         bool   // alert correlated with known-bad IOC
	IOCSource           string // threatfox | misp | abuseipdb | …
	HostKnown           bool   // we have an agent on the host
	HostIsCritical      bool   // host tagged "production" / "tier-1"
	IsRepeatOffender    bool   // this rule fired ≥ N times this hour
	HasOutboundC2Beacon bool   // network sensor flagged beaconing
	IsFirstTimeBinary   bool   // hash never seen on this host
	UserIsService       bool   // alert principal is a service account
	IsFromCrownJewel    bool   // host hosts crown-jewel data
}

// Action constants — the UI maps these to button labels and to actual
// Wails / REST handlers (quarantine_host → networkisolatorservice).
const (
	ActionQuarantineHost = "quarantine_host"
	ActionEvidenceCapt   = "evidence_capture"
	ActionEscalateT3     = "escalate_tier_3"
	ActionSuppressFP     = "suppress_as_fp"
	ActionWatchOnly      = "watch_only"
	ActionInvestigate    = "investigate_host"
)

// allActions enumerates every action so the alternatives slice is
// deterministic (order matches the priority below). Keep in sync with
// the constants above.
var allActions = []string{
	ActionQuarantineHost,
	ActionEvidenceCapt,
	ActionEscalateT3,
	ActionInvestigate,
	ActionWatchOnly,
	ActionSuppressFP,
}

// scoreEntry is the per-action accumulator while we walk the rules.
type scoreEntry struct {
	score   float64
	reasons []string
}

// Recommend evaluates the rule set against a fact-set and returns the
// top-scoring action plus alternatives. The function is pure — no I/O,
// no allocations beyond the result struct, safe for parallel callers.
func Recommend(f NBAFacts) RecommendedAction {
	scores := map[string]*scoreEntry{}
	for _, a := range allActions {
		scores[a] = &scoreEntry{}
	}

	add := func(action string, weight float64, why string) {
		s := scores[action]
		if s == nil {
			return
		}
		s.score += weight
		if why != "" {
			s.reasons = append(s.reasons, why)
		}
	}

	sev := strings.ToLower(f.Severity)
	cat := strings.ToLower(f.Category)

	// ── Severity baseline ─────────────────────────────────────────
	// Critical alerts default toward containment. Medium baselines
	// toward investigation. Low/info baselines toward watch-only.
	switch sev {
	case "critical":
		add(ActionQuarantineHost, 0.40, "Critical severity")
		add(ActionEvidenceCapt, 0.25, "")
		add(ActionEscalateT3, 0.20, "")
	case "high":
		add(ActionInvestigate, 0.35, "High severity")
		add(ActionEvidenceCapt, 0.20, "")
		add(ActionQuarantineHost, 0.15, "")
	case "medium", "med":
		add(ActionInvestigate, 0.30, "Medium severity")
		add(ActionWatchOnly, 0.10, "")
	case "low":
		add(ActionWatchOnly, 0.40, "Low severity")
		add(ActionSuppressFP, 0.10, "")
	case "info":
		add(ActionWatchOnly, 0.50, "Info-level signal")
		add(ActionSuppressFP, 0.20, "")
	}

	// ── Category multipliers ─────────────────────────────────────
	if strings.Contains(cat, "initial-access") || strings.Contains(cat, "execution") {
		add(ActionQuarantineHost, 0.15, "Initial-access / execution tactic")
	}
	if strings.Contains(cat, "exfiltration") || strings.Contains(cat, "command-and-control") {
		add(ActionQuarantineHost, 0.20, "Exfil / C2 tactic")
		add(ActionEvidenceCapt, 0.15, "")
	}
	if strings.Contains(cat, "discovery") || strings.Contains(cat, "reconnaissance") {
		add(ActionInvestigate, 0.15, "Recon tactic")
	}
	if strings.Contains(cat, "impact") {
		add(ActionEvidenceCapt, 0.20, "Impact tactic")
	}

	// ── Context multipliers ──────────────────────────────────────
	if f.HasIOCMatch {
		add(ActionQuarantineHost, 0.20,
			"IOC match"+ifNotEmpty(" ("+f.IOCSource+")", f.IOCSource))
		add(ActionEvidenceCapt, 0.10, "")
	}
	if f.HasOutboundC2Beacon {
		add(ActionQuarantineHost, 0.25, "C2 beacon detected")
	}
	if f.IsFirstTimeBinary {
		add(ActionInvestigate, 0.15, "First-time-seen binary on host")
		add(ActionEvidenceCapt, 0.10, "")
	}
	if f.IsRepeatOffender {
		// A noisy rule is more likely a tuning issue than a real
		// breach — bias hard toward suppression review, away from
		// containment. We add 0.55 (not 0.30) so suppression wins
		// even against medium-severity baselines that already gave
		// investigate ~0.45.
		add(ActionSuppressFP, 0.55, "Repeat-firing rule (noise)")
		add(ActionQuarantineHost, -0.30, "")
		add(ActionInvestigate, -0.20, "")
	}
	if f.HostIsCritical || f.IsFromCrownJewel {
		// Containing a tier-1 host has high blast radius — push
		// toward evidence + escalation, not unilateral quarantine.
		// Penalty must exceed the execution-tactic boost (0.15) so
		// quarantine never wins on a crown-jewel host even with a
		// critical-severity baseline.
		add(ActionEvidenceCapt, 0.25, "Tier-1 / crown-jewel host")
		add(ActionEscalateT3, 0.20, "")
		add(ActionQuarantineHost, -0.30, "")
	}
	if f.UserIsService {
		// Service-account alerts often need human verification before
		// containment — quarantining a service breaks the app.
		add(ActionEscalateT3, 0.15, "Service-account principal")
		add(ActionQuarantineHost, -0.15, "")
	}
	if !f.HostKnown {
		// We can't act on a host we don't have an agent on.
		add(ActionQuarantineHost, -0.40, "")
		add(ActionInvestigate, 0.10, "")
	}

	// ── Rank ──────────────────────────────────────────────────────
	type ranked struct {
		action  string
		score   float64
		reasons []string
	}
	ranks := make([]ranked, 0, len(scores))
	for a, s := range scores {
		// Clamp to [0, 1].
		v := s.score
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		ranks = append(ranks, ranked{action: a, score: v, reasons: s.reasons})
	}
	sort.SliceStable(ranks, func(i, j int) bool {
		if ranks[i].score != ranks[j].score {
			return ranks[i].score > ranks[j].score
		}
		// Stable tie-breaker so the UI doesn't see the same alert
		// flicker between two equal-score actions across refreshes.
		return ranks[i].action < ranks[j].action
	})

	if len(ranks) == 0 || ranks[0].score == 0 {
		return RecommendedAction{
			Action:       ActionInvestigate,
			Confidence:   0.0,
			Reason:       "No signal strong enough to recommend a specific action",
			Alternatives: []string{ActionWatchOnly},
		}
	}

	top := ranks[0]
	alt := []string{}
	for i := 1; i < len(ranks) && len(alt) < 2; i++ {
		if ranks[i].score > 0 {
			alt = append(alt, ranks[i].action)
		}
	}

	reason := joinReasons(top.reasons)
	if reason == "" {
		reason = "Severity baseline"
	}

	return RecommendedAction{
		Action:       top.action,
		Confidence:   roundTo(top.score, 2),
		Reason:       reason,
		Alternatives: alt,
	}
}

func ifNotEmpty(s, guard string) string {
	if guard == "" {
		return ""
	}
	return s
}

func joinReasons(rs []string) string {
	out := []string{}
	for _, r := range rs {
		if r == "" {
			continue
		}
		// De-dup while preserving order.
		seen := false
		for _, o := range out {
			if o == r {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, r)
		}
	}
	return strings.Join(out, " · ")
}

func roundTo(v float64, places int) float64 {
	mul := 1.0
	for i := 0; i < places; i++ {
		mul *= 10
	}
	return float64(int(v*mul+0.5)) / mul
}
