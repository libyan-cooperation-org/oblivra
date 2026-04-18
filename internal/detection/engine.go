package detection

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

const (
	// MaxRuleCost is the upper bound for a rule's computational/memory pressure.
	// Rules exceeding this are throttled to ensure platform stability.
	MaxRuleCost = 10000
)

// Event represents a normalized log event for the detection engine to process
type Event struct {
	TenantID  string
	EventType string
	SourceIP  string
	User      string
	HostID    string
	RawLog    string
	Location  string
	Timestamp string
}

// Match represents a triggered detection rule.
type Match struct {
	TenantID        string
	RuleID          string
	RuleName        string
	Severity        string
	Description     string
	MitreTactics    []string
	MitreTechniques []string
	TriggeredAt     string
	Events          []Event
	Context         map[string]string
	// ConfidenceScore is 0–100. It reflects how strongly the rule evidence
	// supports the alert — higher threshold saturation and severity raises it.
	ConfidenceScore int `json:"confidence_score"`
}

// Evaluator wraps RuleEngine with state tracking for thresholds/sequences.
type Evaluator struct {
	*RuleEngine
	log *logger.Logger

	// routeIndex maps EventType → []Rule for O(1) rule lookup.
	routeIndex RouteIndex

	// State tracking: RuleID -> Bounded LRU Cache (GroupKey -> []Event)
	state   map[string]*expirable.LRU[string, []Event]
	stateMu sync.RWMutex

	// Deduplication tracker: RuleID -> LRU (GroupKey -> LastTriggerTime)
	alerts map[string]*expirable.LRU[string, time.Time]

	// Sharding scope
	IsLocal bool
	// bus is optional: when set, every triggered match publishes an
	// ExplainedMatch on "detection.explained_match" for the API ring buffer.
	bus *eventbus.Bus
}

// NewEvaluator creates a new stateful detection evaluator.
func NewEvaluator(rulesDir string, log *logger.Logger) (*Evaluator, error) {
	re, err := NewRuleEngine(rulesDir, log)
	if err != nil {
		return nil, err
	}
	ev := &Evaluator{
		RuleEngine: re,
		log:        log,
		state:      make(map[string]*expirable.LRU[string, []Event]),
		alerts:     make(map[string]*expirable.LRU[string, time.Time]),
		IsLocal:    true, // Default to local shard-level execution
	}
	ev.routeIndex = BuildRouteIndex(re.rules)
	log.Info("[DETECTION] Route index built: %d EventType buckets, %d wildcard rules",
		len(ev.routeIndex)-1, len(ev.routeIndex[wildcardKey]))
	return ev, nil
}

// SetBus wires the event bus for explainability log publishing.
// Call this after container initialisation if you want explained matches on the bus.
func (e *Evaluator) SetBus(bus *eventbus.Bus) {
	e.bus = bus
}

// RebuildRouteIndex rebuilds the EventType routing index after a rule reload.
func (e *Evaluator) RebuildRouteIndex() {
	e.stateMu.Lock()
	defer e.stateMu.Unlock()
	e.routeIndex = BuildRouteIndex(e.rules)
	e.log.Info("[DETECTION] Route index rebuilt: %d buckets", len(e.routeIndex))
}

// Clone creates a shallow clone of the evaluator.
// The clone shares the same RuleEngine and routeIndex but has its own
// isolated state and alert caches for thread-safe parallel processing in shards.
func (e *Evaluator) Clone() *Evaluator {
	e.stateMu.RLock()
	defer e.stateMu.RUnlock()

	return &Evaluator{
		RuleEngine: e.RuleEngine,
		log:        e.log,
		routeIndex: e.routeIndex,
		state:      make(map[string]*expirable.LRU[string, []Event]),
		alerts:     make(map[string]*expirable.LRU[string, time.Time]),
	}
}

// ProcessEvent analyzes a new incoming event against all loaded rules.
func (e *Evaluator) ProcessEvent(evt Event) []Match {
	var matches []Match

	var candidates []Rule
	if e.routeIndex != nil {
		candidates = e.routeIndex.CandidateRules(evt.EventType)
	} else {
		candidates = e.rules
	}

	for _, rule := range candidates {
		// Filter based on sharding scope
		if e.IsLocal && rule.IsGlobal {
			continue // Handled by global CorrelationHub
		}
		if !e.IsLocal && !rule.IsGlobal {
			continue // Handled by shard evaluators
		}

		// ── Circuit Breaker ──────────────────────────────────────────────────
		// Prevent resource exhaustion from expensive rules (e.g. complex regex,
		// extreme window sizes, or massive groupings).
		cost := rule.ExecutionCost()
		if cost > MaxRuleCost {
			e.log.Warn("[DETECTION:THROTTLED] Rule %s [%s] exceeds MaxRuleCost (%d > %d). Skipping.",
				rule.Name, rule.ID, cost, MaxRuleCost)
			continue
		}

		if rule.Type == SequenceRule {
			if match := e.evaluateRuleState(rule, evt); match != nil {
				matches = append(matches, *match)
			}
		} else {
			if e.matchesConditions(rule.Conditions, evt) {
				if match := e.evaluateRuleState(rule, evt); match != nil {
					matches = append(matches, *match)
				}
			}
		}
	}

	return matches
}

func safeRegexMatch(pattern, s string, timeout time.Duration) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	resultCh := make(chan bool, 1)
	go func() {
		resultCh <- re.MatchString(s)
	}()
	select {
	case res := <-resultCh:
		return res, nil
	case <-time.After(timeout):
		return false, fmt.Errorf("regex execution timed out (ReDoS protection)")
	}
}

func (e *Evaluator) matchesConditions(conditions map[string]interface{}, evt Event) bool {
	for k, v := range conditions {
		if !e.evalCondition(k, v, evt) {
			return false
		}
	}
	return true
}

func (e *Evaluator) evalCondition(k string, v interface{}, evt Event) bool {
	timeout := 100 * time.Millisecond

	if slice, ok := v.([]interface{}); ok {
		for _, item := range slice {
			if e.evalCondition(k, item, evt) {
				return true
			}
		}
		return false
	}
	if slice, ok := v.([]string); ok {
		for _, item := range slice {
			if e.evalCondition(k, item, evt) {
				return true
			}
		}
		return false
	}

	parts := strings.Split(k, "|")
	fieldName := parts[0]
	modifiers := parts[1:]

	var target string
	isKnownField := true
	switch strings.ToLower(fieldName) {
	case "eventtype":
		target = evt.EventType
	case "source_ip", "src_ip":
		target = evt.SourceIP
	case "user", "username":
		target = evt.User
	case "host", "hostid":
		target = evt.HostID
	case "output_contains", "rawlog":
		target = evt.RawLog
	default:
		target = evt.RawLog
		isKnownField = false
	}

	valStr := fmt.Sprintf("%v", v)
	valLower := strings.ToLower(valStr)

	if strings.HasPrefix(valLower, "cidr:") && strings.ToLower(fieldName) == "source_ip" {
		_, ipNet, err := net.ParseCIDR(strings.TrimSpace(valStr[5:]))
		if err == nil {
			ip := net.ParseIP(evt.SourceIP)
			if ip != nil && ipNet.Contains(ip) {
				return true
			}
		}
		return false
	}

	match := false
	isNegated := false
	matchType := "exact"
	if !isKnownField {
		matchType = "contains"
	}

	for _, m := range modifiers {
		switch strings.ToLower(m) {
		case "contains":
			matchType = "contains"
		case "startswith":
			matchType = "startswith"
		case "endswith":
			matchType = "endswith"
		case "not":
			isNegated = true
		}
	}

	if strings.HasPrefix(valLower, "regex:") {
		regexPattern := strings.TrimSpace(valStr[6:])
		matched, _ := safeRegexMatch(regexPattern, target, timeout)
		match = matched
	} else {
		tLower := strings.ToLower(target)
		switch matchType {
		case "contains":
			match = strings.Contains(tLower, valLower)
		case "startswith":
			match = strings.HasPrefix(tLower, valLower)
		case "endswith":
			match = strings.HasSuffix(tLower, valLower)
		default:
			match = tLower == valLower
		}
	}

	if isNegated {
		return !match
	}
	return match
}

func (e *Evaluator) evaluateRuleState(rule Rule, evt Event) *Match {
	e.stateMu.Lock()
	defer e.stateMu.Unlock()

	parseEvtTime := func(ts string) time.Time {
		t, _ := time.Parse(time.RFC3339, ts)
		return t
	}

	groupKey := "global"
	if evt.TenantID != "" {
		groupKey = evt.TenantID
	}

	if len(rule.GroupBy) > 0 {
		var parts []string
		if evt.TenantID != "" {
			parts = append(parts, "t:"+evt.TenantID)
		}
		for _, gb := range rule.GroupBy {
			switch strings.ToLower(gb) {
			case "source_ip":
				parts = append(parts, "ip:"+evt.SourceIP)
			case "user":
				parts = append(parts, "u:"+evt.User)
			case "host":
				parts = append(parts, "h:"+evt.HostID)
			}
		}
		if len(parts) > 0 {
			groupKey = strings.Join(parts, "|")
		}
	}

	if e.state[rule.ID] == nil {
		window := time.Duration(rule.WindowSec) * time.Second
		if window == 0 {
			window = 1 * time.Hour
		}
		e.state[rule.ID] = expirable.NewLRU[string, []Event](10000, nil, window)
	}

	windowCutoff := time.Now().Add(-time.Duration(rule.WindowSec) * time.Second)

	var activeEvents []Event
	if val, ok := e.state[rule.ID].Get(groupKey); ok {
		for _, tracked := range val {
			if parseEvtTime(tracked.Timestamp).After(windowCutoff) {
				activeEvents = append(activeEvents, tracked)
			}
		}
	}

	if rule.Type == SequenceRule {
		currentStepIdx := len(activeEvents)
		if currentStepIdx < len(rule.Sequence) {
			expectedStep := rule.Sequence[currentStepIdx]

			if e.matchesConditions(expectedStep.Conditions, evt) {
				activeEvents = append(activeEvents, evt)
				e.state[rule.ID].Add(groupKey, activeEvents)

				if len(activeEvents) == len(rule.Sequence) {
					return e.triggerAlert(rule, groupKey, activeEvents)
				}
			} else if len(activeEvents) > 0 && e.matchesConditions(rule.Sequence[0].Conditions, evt) {
				activeEvents = []Event{evt}
				e.state[rule.ID].Add(groupKey, activeEvents)
			} else {
				e.state[rule.ID].Add(groupKey, activeEvents)
			}
		}
		return nil
	}

	activeEvents = append(activeEvents, evt)
	e.state[rule.ID].Add(groupKey, activeEvents)

	threshold := rule.Threshold
	if threshold == 0 {
		threshold = 1
	}

	if len(activeEvents) >= threshold {
		return e.triggerAlert(rule, groupKey, activeEvents)
	}

	return nil
}

// triggerAlert verifies deduplication windows, constructs a Match,
// generates a structured explanation, and publishes it to the event bus.
func (e *Evaluator) triggerAlert(rule Rule, groupKey string, activeEvents []Event) *Match {
	if e.alerts[rule.ID] == nil {
		dedup := time.Duration(rule.DedupWindowSec) * time.Second
		if dedup == 0 {
			dedup = 5 * time.Minute
		}
		e.alerts[rule.ID] = expirable.NewLRU[string, time.Time](10000, nil, dedup)
	}

	lastAlert, hasAlerted := e.alerts[rule.ID].Get(groupKey)
	dedupCutoff := time.Now().Add(-time.Duration(rule.DedupWindowSec) * time.Second)

	if !hasAlerted || lastAlert.Before(dedupCutoff) {
		e.alerts[rule.ID].Add(groupKey, time.Now())
		e.state[rule.ID].Remove(groupKey)

		m := &Match{
			TenantID:        activeEvents[0].TenantID,
			RuleID:          rule.ID,
			RuleName:        rule.Name,
			Description:     rule.Description,
			Severity:        rule.Severity,
			MitreTactics:    rule.MitreTactics,
			MitreTechniques: rule.MitreTechniques,
			TriggeredAt:     time.Now().Format(time.RFC3339),
			Events:          activeEvents,
			ConfidenceScore: computeConfidence(rule, len(activeEvents)),
			Context: map[string]string{
				"group_key": groupKey,
				"count":     fmt.Sprintf("%d", len(activeEvents)),
			},
		}

		// ── Explainability ────────────────────────────────────────────────────
		// Build a structured explanation and log it at INFO level so that
		// the operator can see exactly why this rule fired, not just that it did.
		explained := ExplainMatch(*m, rule, groupKey, activeEvents)

		e.log.Info("[DETECTION:WHY] rule=%q severity=%s confidence=%.2f group=%q summary=%q",
			explained.RuleID,
			explained.Severity,
			explained.WhyFired.Confidence,
			explained.WhyFired.GroupKey,
			explained.WhyFired.Summary,
		)

		// Publish to event bus for the API ring buffer and DecisionInspector page.
		if e.bus != nil {
			e.bus.Publish("detection.explained_match", explained)
		}

		return m
	}
	return nil
}

// computeConfidence calculates a score (0-100) based on rule severity and evidence saturation.
func computeConfidence(rule Rule, matches int) int {
	base := 70
	switch strings.ToUpper(rule.Severity) {
	case "CRITICAL":
		base = 95
	case "HIGH":
		base = 85
	case "MEDIUM":
		base = 70
	case "LOW":
		base = 50
	}

	// Boost if matches significantly exceed threshold
	if rule.Threshold > 0 {
		saturation := float64(matches) / float64(rule.Threshold)
		if saturation > 2.0 {
			base += 5
		}
	}

	if base > 100 {
		base = 100
	}
	return base
}

