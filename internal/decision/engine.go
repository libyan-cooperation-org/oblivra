package decision

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// DecisionTrace captures the full reasoning chain behind a security decision.
type DecisionTrace struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	RuleID          string                 `json:"rule_id"`
	RuleName        string                 `json:"rule_name"`
	InputEvents     []EventSummary         `json:"input_events"`
	EvidenceChain   []EvidenceLink         `json:"evidence_chain"`
	ConfidenceScore float64                `json:"confidence_score"`
	Alternatives    []Alternative          `json:"alternatives"`
	Explanation     string                 `json:"explanation"`
	ApprovedBy      string                 `json:"approved_by,omitempty"`
	CryptoProof     string                 `json:"crypto_proof"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// EventSummary is a lightweight representation of an input event.
type EventSummary struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	SourceIP  string    `json:"source_ip,omitempty"`
	User      string    `json:"user,omitempty"`
	HostID    string    `json:"host_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// EvidenceLink connects a decision to its supporting evidence.
type EvidenceLink struct {
	Type        string  `json:"type"` // "rule_match", "threshold_breach", "correlation"
	Description string  `json:"description"`
	SourceID    string  `json:"source_id,omitempty"`
	Confidence  float64 `json:"confidence"`
}

// Alternative represents a decision path that was NOT taken.
type Alternative struct {
	RuleID   string  `json:"rule_id"`
	RuleName string  `json:"rule_name"`
	Score    float64 `json:"score"`
	Reason   string  `json:"reason"` // why it was not chosen
}

// DecisionEngine captures and stores the reasoning behind security decisions.
type DecisionEngine struct {
	mu      sync.RWMutex
	traces  map[string]*DecisionTrace
	ordered []string // ordered IDs for recency queries
	maxSize int
	bus     *eventbus.Bus
	log     *logger.Logger
}

// NewDecisionEngine creates a new decision traceability engine.
func NewDecisionEngine(bus *eventbus.Bus, log *logger.Logger) *DecisionEngine {
	e := &DecisionEngine{
		traces:  make(map[string]*DecisionTrace),
		ordered: make([]string, 0),
		maxSize: 10000,
		bus:     bus,
		log:     log.WithPrefix("decision"),
	}

	// Auto-capture from detection matches
	if bus != nil {
		bus.Subscribe("detection.match", func(event eventbus.Event) {
			if m, ok := event.Data.(map[string]interface{}); ok {
				ruleID, _ := m["rule_id"].(string)
				ruleName, _ := m["rule_name"].(string)
				severity, _ := m["severity"].(string)

				trace := &DecisionTrace{
					RuleID:   ruleID,
					RuleName: ruleName,
					EvidenceChain: []EvidenceLink{
						{
							Type:        "rule_match",
							Description: fmt.Sprintf("Detection rule '%s' matched with severity '%s'", ruleName, severity),
							Confidence:  0.85,
						},
					},
					ConfidenceScore: 0.85,
					Metadata:        m,
				}
				e.CaptureDecision(trace)
			}
		})
	}

	return e
}

// CaptureDecision records a new decision trace.
func (e *DecisionEngine) CaptureDecision(trace *DecisionTrace) string {
	e.mu.Lock()
	defer e.mu.Unlock()

	trace.ID = e.generateID(trace.RuleID)
	trace.Timestamp = time.Now()
	trace.CryptoProof = e.generateProof(trace)
	trace.Explanation = e.generateExplanation(trace)

	e.traces[trace.ID] = trace
	e.ordered = append(e.ordered, trace.ID)

	// Enforce capacity
	if len(e.ordered) > e.maxSize {
		evict := e.ordered[:len(e.ordered)-e.maxSize]
		for _, id := range evict {
			delete(e.traces, id)
		}
		e.ordered = e.ordered[len(e.ordered)-e.maxSize:]
	}

	e.log.Info("[DECISION] Captured trace %s for rule %s", trace.ID, trace.RuleName)
	return trace.ID
}

// GetTrace returns a single decision trace by ID.
func (e *DecisionEngine) GetTrace(id string) *DecisionTrace {
	e.mu.RLock()
	defer e.mu.RUnlock()
	t, ok := e.traces[id]
	if !ok {
		return nil
	}
	copy := *t
	return &copy
}

// ListRecent returns the N most recent decision traces.
func (e *DecisionEngine) ListRecent(limit int) []DecisionTrace {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]DecisionTrace, 0, limit)
	start := len(e.ordered) - limit
	if start < 0 {
		start = 0
	}

	// Reverse order (newest first)
	for i := len(e.ordered) - 1; i >= start; i-- {
		if t, ok := e.traces[e.ordered[i]]; ok {
			result = append(result, *t)
		}
	}

	return result
}

// GetExplanation generates a human-readable explanation for a decision.
func (e *DecisionEngine) GetExplanation(id string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	t, ok := e.traces[id]
	if !ok {
		return "Decision trace not found."
	}

	return t.Explanation
}

// GetProof returns the cryptographic proof chain for a decision.
func (e *DecisionEngine) GetProof(id string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	t, ok := e.traces[id]
	if !ok {
		return ""
	}

	return t.CryptoProof
}

// Stats returns summary statistics.
func (e *DecisionEngine) Stats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	byRule := make(map[string]int)
	for _, t := range e.traces {
		byRule[t.RuleName]++
	}

	return map[string]interface{}{
		"total_decisions": len(e.traces),
		"by_rule":         byRule,
	}
}

func (e *DecisionEngine) generateID(ruleID string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", ruleID, time.Now().UnixNano())))
	return hex.EncodeToString(h[:8])
}

func (e *DecisionEngine) generateProof(trace *DecisionTrace) string {
	// Create a deterministic proof chain: rule_id | evidence_descriptions | confidence | timestamp
	var parts []string
	parts = append(parts, trace.RuleID)
	for _, ev := range trace.EvidenceChain {
		parts = append(parts, fmt.Sprintf("%s:%.2f", ev.Type, ev.Confidence))
	}
	parts = append(parts, fmt.Sprintf("%.4f", trace.ConfidenceScore))
	parts = append(parts, trace.Timestamp.Format(time.RFC3339Nano))

	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:])
}

func (e *DecisionEngine) generateExplanation(trace *DecisionTrace) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Detection rule '%s' (ID: %s) fired at %s.\n",
		trace.RuleName, trace.RuleID, trace.Timestamp.Format(time.RFC3339)))

	if len(trace.InputEvents) > 0 {
		sb.WriteString(fmt.Sprintf("Triggered by %d input event(s).\n", len(trace.InputEvents)))
	}

	sb.WriteString("Evidence chain:\n")
	for i, ev := range trace.EvidenceChain {
		sb.WriteString(fmt.Sprintf("  %d. [%s] %s (confidence: %.0f%%)\n",
			i+1, ev.Type, ev.Description, ev.Confidence*100))
	}

	sb.WriteString(fmt.Sprintf("Overall confidence: %.0f%%.\n", trace.ConfidenceScore*100))

	if len(trace.Alternatives) > 0 {
		sb.WriteString("Alternative decisions considered:\n")
		for _, alt := range trace.Alternatives {
			sb.WriteString(fmt.Sprintf("  - %s: NOT taken (%s)\n", alt.RuleName, alt.Reason))
		}
	}

	return sb.String()
}
