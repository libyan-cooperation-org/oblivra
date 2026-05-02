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

// AlertState walks the operator workflow:
//   open       — fresh, no analyst has touched it
//   ack        — analyst saw it; not yet investigated
//   assigned   — analyst owns it (AssignedTo set)
//   resolved   — analyst closed it with a verdict
//
// Backwards-compat: "closed" is treated as a synonym for "resolved" so
// older alerts keep parsing.
type Alert struct {
	ID        string        `json:"id"`
	TenantID  string        `json:"tenantId"`
	RuleID    string        `json:"ruleId"`
	RuleName  string        `json:"ruleName"`
	Severity  AlertSeverity `json:"severity"`
	HostID    string        `json:"hostId,omitempty"`
	Message   string        `json:"message"`
	MITRE     []string      `json:"mitre,omitempty"`
	Triggered time.Time     `json:"triggered"`
	EventIDs  []string      `json:"eventIds"`
	State     string        `json:"state"` // open | ack | assigned | resolved

	// Lifecycle metadata (added Phase 51).
	AcknowledgedBy string     `json:"acknowledgedBy,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
	AssignedTo     string     `json:"assignedTo,omitempty"`
	AssignedAt     *time.Time `json:"assignedAt,omitempty"`
	ResolvedBy     string     `json:"resolvedBy,omitempty"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
	// Verdict ∈ {true-positive, false-positive, benign-true-positive, duplicate, ""}.
	Verdict string `json:"verdict,omitempty"`
	Notes   string `json:"notes,omitempty"`
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

// Subscribe lets a downstream consumer (e.g. WebhookService) receive every
// alert as it's raised. The returned channel is closed when ctx is cancelled.
func (s *AlertService) Subscribe(ctx context.Context, buffer int) <-chan Alert {
	if buffer <= 0 {
		buffer = 64
	}
	ch := make(chan Alert, buffer)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case a := <-s.bus:
				select {
				case ch <- a:
				default:
					// drop if subscriber is too slow — same posture as the
					// event Bus: detection delivery must never block ingest.
				}
			}
		}
	}()
	return ch
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

// Ack records that an analyst has seen the alert. Idempotent: re-acking
// an already-ack'd alert is a no-op (does not overwrite the original
// AcknowledgedBy).
func (s *AlertService) Ack(id, actor string) (Alert, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.all {
		if s.all[i].ID != id {
			continue
		}
		if s.all[i].State == "open" || s.all[i].State == "" {
			now := time.Now().UTC()
			s.all[i].State = "ack"
			s.all[i].AcknowledgedBy = actor
			s.all[i].AcknowledgedAt = &now
		}
		return s.all[i], true
	}
	return Alert{}, false
}

// Assign hands the alert to a specific analyst — implies ack if not yet
// acknowledged. Re-assign overwrites AssignedTo + AssignedAt so the
// "current owner" is unambiguous.
func (s *AlertService) Assign(id, actor, assignee string) (Alert, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.all {
		if s.all[i].ID != id {
			continue
		}
		now := time.Now().UTC()
		if s.all[i].AcknowledgedAt == nil {
			s.all[i].AcknowledgedBy = actor
			s.all[i].AcknowledgedAt = &now
		}
		s.all[i].State = "assigned"
		s.all[i].AssignedTo = assignee
		s.all[i].AssignedAt = &now
		return s.all[i], true
	}
	return Alert{}, false
}

// Resolve closes the alert with a verdict + optional analyst notes.
// Verdict is free-form but we encourage the four canonical values
// (true-positive / false-positive / benign-true-positive / duplicate)
// so the rule effectiveness stats stay comparable.
func (s *AlertService) Resolve(id, actor, verdict, notes string) (Alert, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.all {
		if s.all[i].ID != id {
			continue
		}
		now := time.Now().UTC()
		s.all[i].State = "resolved"
		s.all[i].ResolvedBy = actor
		s.all[i].ResolvedAt = &now
		s.all[i].Verdict = verdict
		if notes != "" {
			s.all[i].Notes = notes
		}
		return s.all[i], true
	}
	return Alert{}, false
}

// Reopen flips a resolved alert back to open. Useful when a verdict
// turns out to be wrong (e.g. "false positive" later traced to a real
// compromise that was missed).
func (s *AlertService) Reopen(id, actor string) (Alert, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.all {
		if s.all[i].ID != id {
			continue
		}
		s.all[i].State = "open"
		s.all[i].ResolvedAt = nil
		s.all[i].ResolvedBy = ""
		s.all[i].Verdict = ""
		_ = actor
		return s.all[i], true
	}
	return Alert{}, false
}

// Get returns one alert by ID. Drives the per-alert detail page.
func (s *AlertService) Get(id string) (Alert, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.all {
		if s.all[i].ID == id {
			return s.all[i], true
		}
	}
	return Alert{}, false
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
