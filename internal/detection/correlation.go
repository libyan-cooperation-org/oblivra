package detection

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// CorrelationRule defines a cross-source, multi-event correlation pattern.
// Unlike sequence rules (ordered chain on one source), correlation rules
// require N distinct event types from potentially different sources within a window.
type CrossSourceRule struct {
	ID             string            `yaml:"id"`
	Name           string            `yaml:"name"`
	Description    string            `yaml:"description"`
	Severity       string            `yaml:"severity"`
	MitreTechnique string            `yaml:"mitre_technique"`
	WindowSec      int               `yaml:"window_sec"`
	DedupWindowSec int               `yaml:"dedup_window_sec"`
	// GroupBy determines correlation scope (e.g. ["user"] means per-user)
	GroupBy []string `yaml:"group_by"`
	// Required specifies the event types that must ALL appear within the window.
	Required []CrossSourceCondition `yaml:"required"`
}

// CrossSourceCondition is one required event type in a correlation rule.
type CrossSourceCondition struct {
	EventType string            `yaml:"event_type"`
	Source    string            `yaml:"source,omitempty"` // optional: "windows", "siem", "ndr" etc.
	Fields    map[string]string `yaml:"fields,omitempty"` // extra field matchers
}

// CorrelationMatch is the result of a triggered cross-source rule.
type CorrelationMatch struct {
	RuleID          string
	RuleName        string
	Description     string
	Severity        string
	MitreTechnique  string
	GroupKey        string
	ContributingEvt []Event
	TriggeredAt     string
}

// correlationState tracks which required conditions have been seen per group key.
type correlationState struct {
	seen map[string][]Event // condition EventType -> events matching it
	mu   sync.Mutex
}

// CorrelationEngine evaluates cross-source, stateful multi-event rules.
// It runs independently of the single-source Evaluator and publishes
// CorrelationMatch results to the event bus.
type CorrelationEngine struct {
	rules   []CrossSourceRule
	// state: rule ID -> group key -> per-condition tracking
	state   map[string]*expirable.LRU[string, *correlationState]
	dedup   map[string]*expirable.LRU[string, time.Time]
	stateMu sync.Mutex
	bus     *eventbus.Bus
	log     *logger.Logger
}

// NewCorrelationEngine initialises the engine with built-in sovereign rules.
func NewCorrelationEngine(bus *eventbus.Bus, log *logger.Logger) *CorrelationEngine {
	e := &CorrelationEngine{
		rules:   builtinCorrelationRules(),
		state:   make(map[string]*expirable.LRU[string, *correlationState]),
		dedup:   make(map[string]*expirable.LRU[string, time.Time]),
		bus:     bus,
		log:     log,
	}

	// Subscribe to all inbound events
	bus.Subscribe("detection.event", func(ev eventbus.Event) {
		if evt, ok := ev.Data.(Event); ok {
			e.ProcessEvent(evt)
		}
	})

	return e
}

// ProcessEvent ingests an event and evaluates it against all cross-source rules.
func (e *CorrelationEngine) ProcessEvent(evt Event) {
	for _, rule := range e.rules {
		e.evaluate(rule, evt)
	}
}

func (e *CorrelationEngine) evaluate(rule CrossSourceRule, evt Event) {
	// 1. Find which required condition(s) this event satisfies
	var matchedConditions []CrossSourceCondition
	for _, cond := range rule.Required {
		if conditionMatches(cond, evt) {
			matchedConditions = append(matchedConditions, cond)
		}
	}
	if len(matchedConditions) == 0 {
		return
	}

	// 2. Resolve group key
	groupKey := resolveGroupKey(rule.GroupBy, evt)

	// 3. Get or create per-rule state LRU
	e.stateMu.Lock()
	if e.state[rule.ID] == nil {
		window := time.Duration(rule.WindowSec) * time.Second
		if window == 0 {
			window = 5 * time.Minute
		}
		e.state[rule.ID] = expirable.NewLRU[string, *correlationState](10000, nil, window)
	}
	lru := e.state[rule.ID]
	e.stateMu.Unlock()

	// 4. Get or create correlation state for this group key
	cs, ok := lru.Get(groupKey)
	if !ok {
		cs = &correlationState{seen: make(map[string][]Event)}
		lru.Add(groupKey, cs)
	}

	cs.mu.Lock()
	for _, cond := range matchedConditions {
		cs.seen[cond.EventType] = append(cs.seen[cond.EventType], evt)
	}

	// 5. Check if ALL required conditions have been satisfied
	allSatisfied := true
	var contributing []Event
	for _, cond := range rule.Required {
		evts, found := cs.seen[cond.EventType]
		if !found || len(evts) == 0 {
			allSatisfied = false
			break
		}
		contributing = append(contributing, evts...)
	}
	cs.mu.Unlock()

	if !allSatisfied {
		return
	}

	// 6. Deduplication check
	e.stateMu.Lock()
	if e.dedup[rule.ID] == nil {
		dedup := time.Duration(rule.DedupWindowSec) * time.Second
		if dedup == 0 {
			dedup = 10 * time.Minute
		}
		e.dedup[rule.ID] = expirable.NewLRU[string, time.Time](10000, nil, dedup)
	}
	dedupLRU := e.dedup[rule.ID]
	e.stateMu.Unlock()

	if _, alreadyFired := dedupLRU.Get(groupKey); alreadyFired {
		return
	}
	dedupLRU.Add(groupKey, time.Now())

	// 7. Reset state for this group so the rule can re-trigger
	lru.Remove(groupKey)

	match := CorrelationMatch{
		RuleID:          rule.ID,
		RuleName:        rule.Name,
		Description:     fmt.Sprintf("%s [group=%s]", rule.Description, groupKey),
		Severity:        rule.Severity,
		MitreTechnique:  rule.MitreTechnique,
		GroupKey:        groupKey,
		ContributingEvt: contributing,
		TriggeredAt:     time.Now().Format(time.RFC3339),
	}

	e.log.Warn("[CorrelationEngine] Rule fired: %s | Group: %s | Technique: %s",
		rule.Name, groupKey, rule.MitreTechnique)

	e.bus.Publish("detection.correlation_match", match)
	e.bus.Publish("siem.alert_fired", map[string]interface{}{
		"type":        "CORRELATION_" + strings.ToUpper(rule.ID),
		"severity":    rule.Severity,
		"description": match.Description,
		"technique":   rule.MitreTechnique,
		"group_key":   groupKey,
		"rule_id":     rule.ID,
	})
}

// conditionMatches checks an event against a CrossSourceCondition.
func conditionMatches(cond CrossSourceCondition, evt Event) bool {
	if cond.EventType != "" && !strings.EqualFold(evt.EventType, cond.EventType) {
		return false
	}
	if cond.Source != "" && !strings.EqualFold(evt.HostID, cond.Source) {
		return false
	}
	for k, v := range cond.Fields {
		switch strings.ToLower(k) {
		case "user":
			if !strings.EqualFold(evt.User, v) {
				return false
			}
		case "source_ip":
			if evt.SourceIP != v {
				return false
			}
		case "output_contains":
			if !strings.Contains(strings.ToLower(evt.RawLog), strings.ToLower(v)) {
				return false
			}
		}
	}
	return true
}

// resolveGroupKey builds a correlation scope key from GroupBy fields.
func resolveGroupKey(groupBy []string, evt Event) string {
	if len(groupBy) == 0 {
		return "global"
	}
	var parts []string
	for _, gb := range groupBy {
		switch strings.ToLower(gb) {
		case "user":
			parts = append(parts, "u:"+evt.User)
		case "source_ip":
			parts = append(parts, "ip:"+evt.SourceIP)
		case "host":
			parts = append(parts, "h:"+evt.HostID)
		}
	}
	if len(parts) == 0 {
		return "global"
	}
	return strings.Join(parts, "|")
}

// builtinCorrelationRules returns sovereign-grade cross-source correlation rules.
// These cover common attacker kill-chain progressions across multiple event sources.
func builtinCorrelationRules() []CrossSourceRule {
	return []CrossSourceRule{
		{
			ID:             "CORR-001",
			Name:           "Credential Spray + Successful Login",
			Description:    "Multiple failed authentications followed by a successful login from same user — credential spraying succeeded.",
			Severity:       "CRITICAL",
			MitreTechnique: "T1110.003",
			WindowSec:      300,
			DedupWindowSec: 600,
			GroupBy:        []string{"user"},
			Required: []CrossSourceCondition{
				{EventType: "failed_login"},
				{EventType: "successful_login"},
			},
		},
		{
			ID:             "CORR-002",
			Name:           "Recon + Lateral Movement",
			Description:    "Internal port scan followed by SMB/RDP connection — likely attacker pivoting after initial recon.",
			Severity:       "CRITICAL",
			MitreTechnique: "T1021.002",
			WindowSec:      600,
			DedupWindowSec: 1200,
			GroupBy:        []string{"source_ip"},
			Required: []CrossSourceCondition{
				{EventType: "port_scan"},
				{EventType: "NDR_LATERAL_MOVEMENT"},
			},
		},
		{
			ID:             "CORR-003",
			Name:           "Privilege Escalation + Persistence Install",
			Description:    "Privilege escalation event followed by a new scheduled task or service — attacker establishing persistence.",
			Severity:       "HIGH",
			MitreTechnique: "T1053.005",
			WindowSec:      300,
			DedupWindowSec: 900,
			GroupBy:        []string{"host"},
			Required: []CrossSourceCondition{
				{EventType: "privilege_escalation"},
				{EventType: "scheduled_task_created"},
			},
		},
		{
			ID:             "CORR-004",
			Name:           "Data Staging + Exfiltration",
			Description:    "Large file archive creation followed by anomalous outbound transfer — possible data exfiltration.",
			Severity:       "HIGH",
			MitreTechnique: "T1048",
			WindowSec:      1800,
			DedupWindowSec: 3600,
			GroupBy:        []string{"host"},
			Required: []CrossSourceCondition{
				{EventType: "large_archive_created"},
				{EventType: "anomalous_outbound_transfer"},
			},
		},
		{
			ID:             "CORR-005",
			Name:           "Ransomware Behavioral Chain",
			Description:    "High-entropy file writes + shadow copy deletion in same window — ransomware execution confirmed.",
			Severity:       "CRITICAL",
			MitreTechnique: "T1486",
			WindowSec:      120,
			DedupWindowSec: 300,
			GroupBy:        []string{"host"},
			Required: []CrossSourceCondition{
				{EventType: "high_entropy_writes"},
				{EventType: "shadow_copy_deleted"},
			},
		},
		{
			ID:             "CORR-006",
			Name:           "C2 Beacon + Internal Scan",
			Description:    "C2 communication detected followed by internal network scanning — implant performing autonomous reconnaissance.",
			Severity:       "CRITICAL",
			MitreTechnique: "T1071",
			WindowSec:      600,
			DedupWindowSec: 1800,
			GroupBy:        []string{"source_ip"},
			Required: []CrossSourceCondition{
				{EventType: "c2_beacon_detected"},
				{EventType: "port_scan"},
			},
		},
		{
			ID:             "CORR-007",
			Name:           "New Admin Account + Immediate Login",
			Description:    "New privileged account created and immediately used — potential backdoor account creation.",
			Severity:       "HIGH",
			MitreTechnique: "T1136.001",
			WindowSec:      180,
			DedupWindowSec: 600,
			GroupBy:        []string{"user"},
			Required: []CrossSourceCondition{
				{EventType: "admin_account_created"},
				{EventType: "successful_login"},
			},
		},
	}
}
