package detection

import (

	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// Event represents a normalized log event for the detection engine to process
type Event struct {
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
	RuleID          string
	RuleName        string
	Severity        string
	Description     string
	MitreTactics    []string
	MitreTechniques []string
	TriggeredAt     string
	Events          []Event           // The events that contributed to this match
	Context         map[string]string // Additional context (e.g., grouped by IP)
}

// Evaluator wraps RuleEngine with state tracking for thresholds/sequences.
type Evaluator struct {
	*RuleEngine
	log *logger.Logger

	// routeIndex maps EventType → []Rule for O(1) rule lookup instead of O(N).
	// Rebuilt atomically on every rule load / hot-reload.
	routeIndex RouteIndex

	// State tracking: RuleID -> Bounded LRU Cache (GroupKey -> []Event)
	// Keeps max 10,000 tracked entities per rule. Discards old entities via TTL.
	state   map[string]*expirable.LRU[string, []Event]
	stateMu sync.RWMutex

	// Deduplication tracker: RuleID -> Bounded LRU Cache (GroupKey -> LastTriggerTime)
	alerts map[string]*expirable.LRU[string, time.Time]
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
	}
	// Build the initial route index from loaded rules.
	ev.routeIndex = BuildRouteIndex(re.rules)
	log.Info("[DETECTION] Route index built: %d EventType buckets, %d wildcard rules",
		len(ev.routeIndex)-1, len(ev.routeIndex[wildcardKey]))
	return ev, nil
}

// RebuildRouteIndex rebuilds the EventType routing index after a rule reload.
// Called by AlertingService.reloadSigmaRules() after LoadSigmaDirectory completes.
func (e *Evaluator) RebuildRouteIndex() {
	e.stateMu.Lock()
	defer e.stateMu.Unlock()
	e.routeIndex = BuildRouteIndex(e.rules)
	e.log.Info("[DETECTION] Route index rebuilt: %d buckets", len(e.routeIndex))
}

// ProcessEvent analyzes a new incoming event against all loaded rules.
// Uses the RouteIndex to evaluate only rules relevant to the event's EventType
// instead of scanning all rules — O(matching) rather than O(all rules).
func (e *Evaluator) ProcessEvent(evt Event) []Match {
	var matches []Match

	// Fetch only candidate rules for this EventType.
	// Falls back to full rule set if routeIndex hasn't been built yet.
	var candidates []Rule
	if e.routeIndex != nil {
		candidates = e.routeIndex.CandidateRules(evt.EventType)
	} else {
		candidates = e.rules
	}

	for _, rule := range candidates {
		if rule.Type == SequenceRule {
			// Sequences track state without needing to match generic top-level conditions
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

// safeRegexMatch executes a regular expression against a string with a strict timeout to prevent ReDoS CPU exhaustion.
func safeRegexMatch(pattern, s string, timeout time.Duration) (bool, error) {
	// Precompile check to avoid panics on bad signatures
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}

	resultCh := make(chan bool, 1)

	go func() {
		// regexp execution isn't perfectly preemptible in Go, but if it hangs,
		// the goroutine is orphaned rather than blocking the main detection engine loop forever.
		res := re.MatchString(s)
		resultCh <- res
	}()

	select {
	case res := <-resultCh:
		return res, nil
	case <-time.After(timeout):
		return false, fmt.Errorf("regex execution timed out (ReDoS protection)")
	}
}

// matchesConditions checks if a single event matches the static criteria of a map of conditions.
// We pass the condition map directly so this function can be reused for sequence steps.
func (e *Evaluator) matchesConditions(conditions map[string]string, evt Event) bool {
	timeout := 100 * time.Millisecond // Strict limit for regex execution per condition

	for k, v := range conditions {
		valLower := strings.ToLower(v)
		isRegex := strings.HasPrefix(valLower, "regex:")
		regexPattern := ""
		if isRegex {
			regexPattern = strings.TrimSpace(v[6:])
		}

		switch strings.ToLower(k) {
		case "eventtype":
			if isRegex {
				if matched, _ := safeRegexMatch(regexPattern, evt.EventType, timeout); !matched {
					return false
				}
			} else if evt.EventType != v {
				return false
			}
		case "source_ip":
			if strings.HasPrefix(valLower, "cidr:") {
				_, ipNet, err := net.ParseCIDR(strings.TrimSpace(v[5:]))
				if err == nil {
					ip := net.ParseIP(evt.SourceIP)
					if ip == nil || !ipNet.Contains(ip) {
						return false
					}
				}
			} else if evt.SourceIP != v {
				return false
			}
		case "user":
			if isRegex {
				if matched, _ := safeRegexMatch(regexPattern, evt.User, timeout); !matched {
					return false
				}
			} else if evt.User != v {
				return false
			}
		case "host":
			if evt.HostID != v {
				return false
			}
		case "output_contains":
			if isRegex {
				if matched, _ := safeRegexMatch("(?i)"+regexPattern, evt.RawLog, timeout); !matched {
					return false
				}
			} else if !strings.Contains(strings.ToLower(evt.RawLog), valLower) {
				return false
			}
		case "location":
			if isRegex {
				if matched, _ := safeRegexMatch("(?i)"+regexPattern, evt.Location, timeout); !matched {
					return false
				}
			} else if !strings.Contains(strings.ToLower(evt.Location), valLower) {
				return false
			}
		}
	}
	return true
}

// evaluateRuleState handles the stateful aspect of rules (thresholds, time windows, sequences)
func (e *Evaluator) evaluateRuleState(rule Rule, evt Event) *Match {
	e.stateMu.Lock()
	defer e.stateMu.Unlock()

	// Helper to parse string timestamp for window comparison
	parseEvtTime := func(ts string) time.Time {
		t, _ := time.Parse(time.RFC3339, ts)
		return t
	}

	// 1. Determine Grouping Key
	groupKey := "global"
	if len(rule.GroupBy) > 0 {
		var parts []string
		for _, gb := range rule.GroupBy {
			switch strings.ToLower(gb) {
			case "source_ip":
				parts = append(parts, evt.SourceIP)
			case "user":
				parts = append(parts, evt.User)
			case "host":
				parts = append(parts, evt.HostID)
			}
		}
		if len(parts) > 0 {
			groupKey = strings.Join(parts, "-")
		}
	}

	if e.state[rule.ID] == nil {
		window := time.Duration(rule.WindowSec) * time.Second
		if window == 0 {
			window = 1 * time.Hour
		}
		// Bound to 10,000 active group keys to prevent runaway correlation memory leaks
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
				// Event matches the next step in the causal chain
				activeEvents = append(activeEvents, evt)
				e.state[rule.ID].Add(groupKey, activeEvents)

				if len(activeEvents) == len(rule.Sequence) {
					// Sequence completed correctly!
					return e.triggerAlert(rule, groupKey, activeEvents)
				}
			} else if len(activeEvents) > 0 && e.matchesConditions(rule.Sequence[0].Conditions, evt) {
				// Previous sequence broken, but matches new start
				activeEvents = []Event{evt}
				e.state[rule.ID].Add(groupKey, activeEvents)
			} else {
				// Neither matches next step nor restarts, just retain current state
				e.state[rule.ID].Add(groupKey, activeEvents)
			}
		}
		return nil
	}

	// Normal Threshold / Frequency rule
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

// triggerAlert verifies deduplication windows and returns a Match
func (e *Evaluator) triggerAlert(rule Rule, groupKey string, activeEvents []Event) *Match {
	if e.alerts[rule.ID] == nil {
		dedup := time.Duration(rule.DedupWindowSec) * time.Second
		if dedup == 0 {
			dedup = 5 * time.Minute // default dedup
		}
		e.alerts[rule.ID] = expirable.NewLRU[string, time.Time](10000, nil, dedup)
	}

	lastAlert, hasAlerted := e.alerts[rule.ID].Get(groupKey)
	dedupCutoff := time.Now().Add(-time.Duration(rule.DedupWindowSec) * time.Second)

	if !hasAlerted || lastAlert.Before(dedupCutoff) {
		e.alerts[rule.ID].Add(groupKey, time.Now())
		e.state[rule.ID].Remove(groupKey) // Reset state after alerting

		return &Match{
			RuleID:          rule.ID,
			RuleName:        rule.Name,
			Description:     rule.Description,
			Severity:        rule.Severity,
			MitreTactics:    rule.MitreTactics,
			MitreTechniques: rule.MitreTechniques,
			TriggeredAt:     time.Now().Format(time.RFC3339),
			Events:          activeEvents,
			Context: map[string]string{
				"group_key": groupKey,
				"count":     fmt.Sprintf("%d", len(activeEvents)),
			},
		}
	}
	return nil
}
