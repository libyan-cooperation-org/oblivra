package services

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// Rule is the OBLIVRA-native rule type. A subset of Sigma's expressiveness:
// every contains-token in any of the AnyOf groups must appear in at least one
// of the configured fields. This is enough for ~80% of common detections and
// keeps the engine boringly fast.
type Rule struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Severity   AlertSeverity `json:"severity"`
	Fields     []string      `json:"fields"`               // event fields to match against
	AnyContain []string      `json:"anyContain,omitempty"` // OR — match if any token appears
	AllContain []string      `json:"allContain,omitempty"` // AND — every token must appear
	EventType  string        `json:"eventType,omitempty"`  // optional eventType filter
	MITRE      []string      `json:"mitre,omitempty"`
	Source     string        `json:"source,omitempty"`     // "builtin" | "user" | "sigma"
	Disabled   bool          `json:"disabled,omitempty"`
}

type RulesService struct {
	log    *slog.Logger
	alerts *AlertService

	mu       sync.RWMutex
	rules    []Rule
	matched  map[string]int  // ruleID → match count
	heatmap  map[string]int  // MITRE technique → count
	lastLoad time.Time
}

func NewRulesService(log *slog.Logger, alerts *AlertService) *RulesService {
	r := &RulesService{
		log:     log,
		alerts:  alerts,
		matched: map[string]int{},
		heatmap: map[string]int{},
	}
	r.rules = builtinRules()
	r.lastLoad = time.Now().UTC()
	return r
}

func (r *RulesService) ServiceName() string { return "RulesService" }

// Evaluate checks an event against every active rule and raises alerts on
// matches. Called from the ingest fan-out (post WAL/hot/Bleve).
func (r *RulesService) Evaluate(ctx context.Context, ev events.Event) {
	r.mu.RLock()
	rules := r.rules
	r.mu.RUnlock()

	for _, rule := range rules {
		if rule.Disabled {
			continue
		}
		if rule.EventType != "" && rule.EventType != ev.EventType {
			continue
		}
		if !matchRule(rule, ev) {
			continue
		}
		r.mu.Lock()
		r.matched[rule.ID]++
		for _, m := range rule.MITRE {
			r.heatmap[m]++
		}
		r.mu.Unlock()
		alert := AlertFromEvent(ev, rule.ID, rule.Name, rule.Severity, rule.MITRE)
		r.alerts.Raise(ctx, alert)
	}
}

func (r *RulesService) List() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Rule, len(r.rules))
	copy(out, r.rules)
	return out
}

// Reload re-imports built-ins. Hot-reload from disk lands in Phase 4 polish.
func (r *RulesService) Reload() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = builtinRules()
	r.lastLoad = time.Now().UTC()
	return len(r.rules), nil
}

type HeatmapEntry struct {
	Technique string `json:"technique"`
	Count     int    `json:"count"`
}

func (r *RulesService) Heatmap() []HeatmapEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]HeatmapEntry, 0, len(r.heatmap))
	for k, v := range r.heatmap {
		out = append(out, HeatmapEntry{Technique: k, Count: v})
	}
	return out
}

func matchRule(rule Rule, ev events.Event) bool {
	target := buildTarget(rule.Fields, ev)
	target = strings.ToLower(target)

	if len(rule.AllContain) > 0 {
		for _, t := range rule.AllContain {
			if !strings.Contains(target, strings.ToLower(t)) {
				return false
			}
		}
	}
	if len(rule.AnyContain) > 0 {
		hit := false
		for _, t := range rule.AnyContain {
			if strings.Contains(target, strings.ToLower(t)) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

func buildTarget(fields []string, ev events.Event) string {
	if len(fields) == 0 {
		fields = []string{"message", "raw"}
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		switch f {
		case "message":
			parts = append(parts, ev.Message)
		case "raw":
			parts = append(parts, ev.Raw)
		case "hostId":
			parts = append(parts, ev.HostID)
		case "eventType":
			parts = append(parts, ev.EventType)
		case "severity":
			parts = append(parts, string(ev.Severity))
		default:
			if v, ok := ev.Fields[f]; ok {
				parts = append(parts, v)
			}
		}
	}
	return strings.Join(parts, " ")
}

func builtinRules() []Rule {
	return []Rule{
		{
			ID:       "builtin-ssh-bruteforce",
			Name:     "Possible SSH brute force",
			Severity: AlertSeverityHigh,
			AnyContain: []string{"failed password", "authentication failure"},
			Fields:   []string{"message", "raw"},
			MITRE:    []string{"T1110.001"},
			Source:   "builtin",
		},
		{
			ID:       "builtin-sudo-failed",
			Name:     "Failed sudo attempt",
			Severity: AlertSeverityMedium,
			AnyContain: []string{"sudo: pam_unix", "sudo: 3 incorrect"},
			Fields:   []string{"message", "raw"},
			MITRE:    []string{"T1548.003"},
			Source:   "builtin",
		},
		{
			ID:       "builtin-firewall-drop",
			Name:     "Firewall dropped traffic",
			Severity: AlertSeverityLow,
			AnyContain: []string{"firewalld dropped", "iptables: drop", "kernel: ufw block"},
			Fields:   []string{"message", "raw"},
			MITRE:    []string{"T1190"},
			Source:   "builtin",
		},
		{
			ID:       "builtin-windows-lsass",
			Name:     "LSASS access (possible credential dumping)",
			Severity: AlertSeverityCritical,
			AnyContain: []string{"lsass.exe", "MiniDump", "comsvcs.dll"},
			AllContain: []string{"lsass"},
			Fields:   []string{"message", "raw"},
			MITRE:    []string{"T1003.001"},
			Source:   "builtin",
		},
		{
			ID:       "builtin-powershell-encoded",
			Name:     "PowerShell encoded command",
			Severity: AlertSeverityHigh,
			AnyContain: []string{"powershell -enc", "powershell.exe -encodedcommand", " -e JAB"},
			Fields:   []string{"message", "raw"},
			MITRE:    []string{"T1059.001"},
			Source:   "builtin",
		},
		{
			ID:       "builtin-ransomware-shadow-delete",
			Name:     "Volume shadow copy deletion",
			Severity: AlertSeverityCritical,
			AnyContain: []string{"vssadmin delete shadows", "wmic shadowcopy delete", "bcdedit /set bootstatuspolicy"},
			Fields:   []string{"message", "raw"},
			MITRE:    []string{"T1490"},
			Source:   "builtin",
		},
		{
			ID:       "builtin-ioc-match",
			Name:     "Threat-intel indicator matched in event",
			Severity: AlertSeverityHigh,
			Fields:   []string{"message", "raw", "hostId"},
			AnyContain: []string{"198.51.100.7", "malicious.example.com"}, // mirrors threatintel seed
			MITRE:      []string{"T1071"},
			Source:     "builtin",
		},
	}
}
