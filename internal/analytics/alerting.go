package analytics

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/notifications"
)

// Trigger defines a pattern-matching alert rule
type Trigger struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Pattern  *regexp.Regexp `json:"-"`
	RawExpr  string         `json:"pattern"`
	Severity string         `json:"severity"` // critical, high, medium, low
	Enabled  bool           `json:"enabled"`
}

// MetricTrigger defines a threshold-based alert for resource usage
type MetricTrigger struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`    // "cpu", "mem_percent", "load", "disk_percent"
	Threshold float64 `json:"threshold"` // e.g. 90.0
	Severity  string  `json:"severity"`
	Enabled   bool    `json:"enabled"`
}

// AlertEvent records a triggered alert for history
type AlertEvent struct {
	Timestamp string `json:"timestamp"`
	TriggerID string `json:"trigger_id"`
	Name      string `json:"name"`
	Severity  string `json:"severity"`
	Host      string `json:"host"`
	SessionID string `json:"session_id"`
	LogLine   string `json:"log_line"`
	Sent      bool   `json:"sent"`
}

// AlertEngine scans SSH output streams for trigger patterns
type AlertEngine struct {
	mu             sync.RWMutex
	triggers       []Trigger
	metricTriggers []MetricTrigger
	notifier       notifications.Notifier
	analytics      Engine               // For persisting alert events to SQLite
	lastSent       map[string]time.Time // "hostLabel-triggerID" -> last fire time
	history        []AlertEvent         // In-memory recent buffer
	ThrottlingEnabled bool              // If true, limit alert frequency
}

// NewAlertEngine creates an engine with built-in default triggers
func NewAlertEngine(notifier notifications.Notifier, ana Engine) *AlertEngine {
	return &AlertEngine{
		notifier:  notifier,
		analytics: ana,
		lastSent:  make(map[string]time.Time),
		history:   make([]AlertEvent, 0),
		ThrottlingEnabled: true,
		triggers: []Trigger{
			{ID: "builtin-1", Name: "OOM Killer", Pattern: regexp.MustCompile(`(?i)Out of memory|Kill process`), RawExpr: `(?i)Out of memory|Kill process`, Severity: "critical", Enabled: true},
			{ID: "builtin-2", Name: "Root Login", Pattern: regexp.MustCompile(`Accepted password for root`), RawExpr: `Accepted password for root`, Severity: "high", Enabled: true},
			{ID: "builtin-3", Name: "Kernel Panic", Pattern: regexp.MustCompile(`(?i)Kernel panic`), RawExpr: `(?i)Kernel panic`, Severity: "critical", Enabled: true},
			{ID: "builtin-4", Name: "Failed SSH", Pattern: regexp.MustCompile(`Failed password for`), RawExpr: `Failed password for`, Severity: "medium", Enabled: true},
			{ID: "builtin-5", Name: "Disk Full", Pattern: regexp.MustCompile(`(?i)No space left on device`), RawExpr: `(?i)No space left on device`, Severity: "critical", Enabled: true},
			{ID: "builtin-6", Name: "Segfault", Pattern: regexp.MustCompile(`segfault at`), RawExpr: `segfault at`, Severity: "high", Enabled: true},
		},
		metricTriggers: []MetricTrigger{
			{ID: "mt-1", Name: "High CPU Usage", Metric: "cpu", Threshold: 90.0, Severity: "critical", Enabled: true},
			{ID: "mt-2", Name: "High Memory Usage", Metric: "mem_percent", Threshold: 90.0, Severity: "high", Enabled: true},
			{ID: "mt-3", Name: "Critical Disk Space", Metric: "disk_percent", Threshold: 95.0, Severity: "critical", Enabled: true},
		},
	}
}

// AddTrigger adds a custom alert trigger
func (ae *AlertEngine) AddTrigger(id, name, pattern, severity string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex: %w", err)
	}
	ae.mu.Lock()
	defer ae.mu.Unlock()
	ae.triggers = append(ae.triggers, Trigger{
		ID: id, Name: name, Pattern: re, RawExpr: pattern, Severity: severity, Enabled: true,
	})
	return nil
}

// RemoveTrigger removes a trigger by ID
func (ae *AlertEngine) RemoveTrigger(id string) {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	for i, t := range ae.triggers {
		if t.ID == id {
			ae.triggers = append(ae.triggers[:i], ae.triggers[i+1:]...)
			return
		}
	}
}

// GetTriggers returns all triggers
func (ae *AlertEngine) GetTriggers() []Trigger {
	ae.mu.RLock()
	defer ae.mu.RUnlock()
	cp := make([]Trigger, len(ae.triggers))
	copy(cp, ae.triggers)
	return cp
}

// --- Metric Trigger Management ---

func (ae *AlertEngine) GetMetricTriggers() []MetricTrigger {
	ae.mu.RLock()
	defer ae.mu.RUnlock()
	cp := make([]MetricTrigger, len(ae.metricTriggers))
	copy(cp, ae.metricTriggers)
	return cp
}

func (ae *AlertEngine) UpdateMetricTrigger(mt MetricTrigger) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	for i, t := range ae.metricTriggers {
		if t.ID == mt.ID {
			ae.metricTriggers[i] = mt
			return
		}
	}
	ae.metricTriggers = append(ae.metricTriggers, mt)
}

func (ae *AlertEngine) RemoveMetricTrigger(id string) {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	for i, t := range ae.metricTriggers {
		if t.ID == id {
			ae.metricTriggers = append(ae.metricTriggers[:i], ae.metricTriggers[i+1:]...)
			return
		}
	}
}

func (ae *AlertEngine) SetMetricTriggers(trigs []MetricTrigger) {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	// Builtins are kept
	var combined []MetricTrigger
	for _, t := range ae.metricTriggers {
		if len(t.ID) >= 3 && t.ID[:3] == "mt-" {
			combined = append(combined, t)
		}
	}
	ae.metricTriggers = append(combined, trigs...)
}

// GetHistory returns recent in-memory alert events (fallback if DB unavailable)
func (ae *AlertEngine) GetHistory() []AlertEvent {
	ae.mu.RLock()
	defer ae.mu.RUnlock()
	cp := make([]AlertEvent, len(ae.history))
	copy(cp, ae.history)
	return cp
}

// ScanStream checks a line of SSH output against all triggers
func (ae *AlertEngine) ScanStream(sessionID, hostLabel, line string) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	now := time.Now()

	for _, t := range ae.triggers {
		if !t.Enabled {
			continue
		}

		if !t.Pattern.MatchString(line) {
			continue
		}

		// Throttle: max 1 alert per trigger per host every 5 minutes
		if ae.ThrottlingEnabled {
			key := fmt.Sprintf("%s-%s", hostLabel, t.ID)
			if last, ok := ae.lastSent[key]; ok && now.Sub(last) < 5*time.Minute {
				continue
			}
			ae.lastSent[key] = now
		}

		event := AlertEvent{
			Timestamp: now.Format(time.RFC3339),
			TriggerID: t.ID,
			Name:      t.Name,
			Severity:  t.Severity,
			Host:      hostLabel,
			SessionID: sessionID,
			LogLine:   truncateStr(line, 500),
			Sent:      false,
		}

		// Only notify on critical/high severity
		if t.Severity == "critical" || t.Severity == "high" {
			msg := fmt.Sprintf("Host: %s\nEvent: %s\nSeverity: %s\nLog: %s",
				hostLabel, t.Name, t.Severity, truncateStr(line, 300))
			go ae.notifier.SendAlert(fmt.Sprintf("Critical Event: %s", t.Name), msg)
			event.Sent = true
		}

		// Persist to SQLite
		if ae.analytics != nil {
			ae.analytics.SaveAlertEvent(t.ID, t.Name, t.Severity, hostLabel, sessionID, truncateStr(line, 500), event.Sent)
		}

		// Keep in-memory buffer too
		ae.history = append(ae.history, event)
		if len(ae.history) > 500 {
			ae.history = ae.history[len(ae.history)-500:]
		}
	}
}

// ScanTelemetry checks real-time host metrics against resource thresholds
func (ae *AlertEngine) ScanTelemetry(hostID string, t monitoring.HostTelemetry) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	now := time.Now()
	hostLabel := hostID

	for _, mt := range ae.metricTriggers {
		if !mt.Enabled {
			continue
		}

		var currentVal float64
		switch mt.Metric {
		case "cpu":
			currentVal = t.CPUUsage
		case "mem_percent":
			if t.MemTotalMB > 0 {
				currentVal = (t.MemUsedMB / t.MemTotalMB) * 100
			}
		case "load":
			currentVal = t.LoadAvg
		case "disk_percent":
			if t.DiskTotalGB > 0 {
				currentVal = (t.DiskUsedGB / t.DiskTotalGB) * 100
			}
		}

		if currentVal < mt.Threshold {
			continue
		}

		// Throttle: max 1 alert per trigger per host every 15 minutes for metrics
		if ae.ThrottlingEnabled {
			key := fmt.Sprintf("%s-%s", hostLabel, mt.ID)
			if last, ok := ae.lastSent[key]; ok && now.Sub(last) < 15*time.Minute {
				continue
			}
			ae.lastSent[key] = now
		}

		event := AlertEvent{
			Timestamp: now.Format(time.RFC3339),
			TriggerID: mt.ID,
			Name:      mt.Name,
			Severity:  mt.Severity,
			Host:      hostLabel,
			LogLine:   fmt.Sprintf("Metric %s exceeded threshold: %.2f > %.2f", mt.Metric, currentVal, mt.Threshold),
			Sent:      false,
		}

		if mt.Severity == "critical" || mt.Severity == "high" {
			msg := fmt.Sprintf("Host: %s\nMetric: %s\nValue: %.2f%%\nThreshold: %.2f%%\nSeverity: %s",
				hostLabel, mt.Metric, currentVal, mt.Threshold, mt.Severity)
			go ae.notifier.SendAlert(fmt.Sprintf("Resource Alert: %s", mt.Name), msg)
			event.Sent = true
		}

		// Persist
		if ae.analytics != nil {
			ae.analytics.SaveAlertEvent(mt.ID, mt.Name, mt.Severity, hostLabel, "", event.LogLine, event.Sent)
		}

		ae.history = append(ae.history, event)
		if len(ae.history) > 500 {
			ae.history = ae.history[len(ae.history)-500:]
		}
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
