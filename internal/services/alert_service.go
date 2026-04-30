package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

type Alert struct {
	ID          string        `json:"id"`
	TenantID    string        `json:"tenantId"`
	RuleID      string        `json:"ruleId"`
	RuleName    string        `json:"ruleName"`
	Severity    AlertSeverity `json:"severity"`
	HostID      string        `json:"hostId,omitempty"`
	Message     string        `json:"message"`
	MITRE       []string      `json:"mitre,omitempty"`
	Triggered   time.Time     `json:"triggered"`
	EventIDs    []string      `json:"eventIds"`
	State       string        `json:"state"` // open|ack|closed
}

// AlertService is an in-memory alerts buffer. Phase 5+ will back it with SQLite.
type AlertService struct {
	log *slog.Logger
	mu  sync.RWMutex
	all []Alert
	cap int
	bus chan Alert
}

func NewAlertService(log *slog.Logger) *AlertService {
	return &AlertService{log: log, cap: 5000, bus: make(chan Alert, 256)}
}

func (s *AlertService) ServiceName() string { return "AlertService" }

// Raise records a new alert and returns it.
func (s *AlertService) Raise(_ context.Context, a Alert) Alert {
	if a.ID == "" {
		var b [10]byte
		_, _ = rand.Read(b[:])
		a.ID = hex.EncodeToString(b[:])
	}
	if a.TenantID == "" {
		a.TenantID = "default"
	}
	if a.Triggered.IsZero() {
		a.Triggered = time.Now().UTC()
	}
	if a.State == "" {
		a.State = "open"
	}
	s.mu.Lock()
	s.all = append(s.all, a)
	if len(s.all) > s.cap {
		s.all = s.all[len(s.all)-s.cap:]
	}
	s.mu.Unlock()
	select {
	case s.bus <- a:
	default:
	}
	s.log.Info("alert raised", "rule", a.RuleID, "severity", a.Severity, "host", a.HostID)
	return a
}

// Recent returns the most recent N alerts, newest first.
func (s *AlertService) Recent(limit int) []Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.all)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]Alert, 0, limit)
	for i := n - 1; i >= n-limit; i-- {
		out = append(out, s.all[i])
	}
	return out
}

// Count returns the current alert ring size without copying.
func (s *AlertService) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.all)
}

// Ack flips an alert to acknowledged.
func (s *AlertService) Ack(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.all {
		if s.all[i].ID == id {
			s.all[i].State = "ack"
			return true
		}
	}
	return false
}

// AlertFromEvent is a convenience constructor used by the rules engine.
func AlertFromEvent(ev events.Event, ruleID, ruleName string, sev AlertSeverity, mitre []string) Alert {
	return Alert{
		TenantID: ev.TenantID,
		RuleID:   ruleID,
		RuleName: ruleName,
		Severity: sev,
		HostID:   ev.HostID,
		Message:  ev.Message,
		MITRE:    mitre,
		EventIDs: []string{ev.ID},
	}
}
