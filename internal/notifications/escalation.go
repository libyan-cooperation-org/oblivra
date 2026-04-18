// Package notifications — Escalation Engine (Phase 2.1.5)
//
// Provides multi-level escalation chains, on-call rotation schedules,
// SLA-based alert timeout tracking, and acknowledgment management.

package notifications

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ─────────────────────────────────────────────────────────────────────────────
// Domain types
// ─────────────────────────────────────────────────────────────────────────────

// EscalationLevel defines one tier in an escalation chain.
type EscalationLevel struct {
	Level    int      `json:"level"`    // 1 = Analyst, 2 = Lead, 3 = Manager, 4 = CISO…
	Name     string   `json:"name"`     // Human-readable label
	Users    []string `json:"users"`    // Target email/user IDs
	Channel  string   `json:"channel"`  // "slack", "email", "webhook", etc.
	WaitMins int      `json:"wait_mins"` // Minutes to wait before escalating to next level
}

// EscalationPolicy is a named chain of levels associated with one or more alert types.
type EscalationPolicy struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	AlertTypes []string          `json:"alert_types"` // e.g., ["security_alert", "failed_login"]
	Levels     []EscalationLevel `json:"levels"`
	SLAMins    int               `json:"sla_mins"` // SLA breach threshold in minutes
	Active     bool              `json:"active"`
}

// OnCallSchedule represents a weekly on-call rotation.
type OnCallSchedule struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Entries []OnCallEntry `json:"entries"`
}

type OnCallEntry struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	WeekdayStart int  `json:"weekday_start"` // 0=Sun, 1=Mon…6=Sat
	WeekdayEnd   int  `json:"weekday_end"`
	HourStart  int    `json:"hour_start"` // 0–23 UTC
	HourEnd    int    `json:"hour_end"`
}

// ActiveEscalation tracks the live state of an escalating alert.
type ActiveEscalation struct {
	AlertID      string    `json:"alert_id"`
	PolicyID     string    `json:"policy_id"`
	CurrentLevel int       `json:"current_level"`
	CreatedAt    time.Time `json:"created_at"`
	LastEscalAt  time.Time `json:"last_escalated_at"`
	AckedBy      string    `json:"acked_by,omitempty"`
	AckedAt      *time.Time `json:"acked_at,omitempty"`
	SLABreached  bool      `json:"sla_breached"`
	Closed       bool      `json:"closed"`
}

// AckRequest is sent by a user to acknowledge an alert.
type AckRequest struct {
	AlertID string `json:"alert_id"`
	UserID  string `json:"user_id"`
	Comment string `json:"comment,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// EscalationManager
// ─────────────────────────────────────────────────────────────────────────────

type EscalationManager struct {
	mu         sync.RWMutex
	policies   map[string]*EscalationPolicy
	schedules  map[string]*OnCallSchedule
	active     map[string]*ActiveEscalation // keyed by alert_id
	history    []*ActiveEscalation          // closed escalations log
	notifier   *NotificationService
	log        *logger.Logger
	done       chan struct{}
}

func NewEscalationManager(notifier *NotificationService, log *logger.Logger) *EscalationManager {
	m := &EscalationManager{
		policies:  make(map[string]*EscalationPolicy),
		schedules: make(map[string]*OnCallSchedule),
		active:    make(map[string]*ActiveEscalation),
		notifier:  notifier,
		log:       log.WithPrefix("escalation"),
		done:      make(chan struct{}),
	}
	m.seedDefaultPolicy()
	go m.ticker()
	return m
}

func (m *EscalationManager) Stop() { close(m.done) }

// ─────────────────────────────────────────────────────────────────────────────
// Policy CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (m *EscalationManager) ListPolicies() []*EscalationPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*EscalationPolicy, 0, len(m.policies))
	for _, p := range m.policies {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (m *EscalationManager) GetPolicy(id string) (*EscalationPolicy, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.policies[id]
	return p, ok
}

func (m *EscalationManager) UpsertPolicy(p *EscalationPolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[p.ID] = p
	m.log.Info("Policy upserted: %s (%d levels, SLA=%dmin)", p.Name, len(p.Levels), p.SLAMins)
}

func (m *EscalationManager) DeletePolicy(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.policies[id]; !ok {
		return false
	}
	delete(m.policies, id)
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// On-call schedule CRUD
// ─────────────────────────────────────────────────────────────────────────────

func (m *EscalationManager) ListSchedules() []*OnCallSchedule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*OnCallSchedule, 0, len(m.schedules))
	for _, s := range m.schedules {
		out = append(out, s)
	}
	return out
}

func (m *EscalationManager) UpsertSchedule(s *OnCallSchedule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schedules[s.ID] = s
}

func (m *EscalationManager) CurrentOnCall(scheduleID string) *OnCallEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sched, ok := m.schedules[scheduleID]
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	wd := int(now.Weekday()) // 0=Sun
	h := now.Hour()
	for i := range sched.Entries {
		e := &sched.Entries[i]
		if wd >= e.WeekdayStart && wd <= e.WeekdayEnd &&
			h >= e.HourStart && h < e.HourEnd {
			return e
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Alert escalation lifecycle
// ─────────────────────────────────────────────────────────────────────────────

// TriggerEscalation creates a new escalation for a given alert and policy.
func (m *EscalationManager) TriggerEscalation(alertID, policyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.active[alertID]; exists {
		return nil // already escalating
	}
	policy, ok := m.policies[policyID]
	if !ok {
		return fmt.Errorf("policy %q not found", policyID)
	}
	if !policy.Active || len(policy.Levels) == 0 {
		return nil
	}

	now := time.Now()
	esc := &ActiveEscalation{
		AlertID:     alertID,
		PolicyID:    policyID,
		CurrentLevel: 1,
		CreatedAt:   now,
		LastEscalAt: now,
	}
	m.active[alertID] = esc

	// Immediately notify level 1
	go m.notifyLevel(policy, 1, alertID)
	m.log.Info("Escalation triggered: alert=%s policy=%s", alertID, policy.Name)
	return nil
}

// Acknowledge marks an alert escalation as acknowledged, halting further escalation.
func (m *EscalationManager) Acknowledge(req AckRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	esc, ok := m.active[req.AlertID]
	if !ok {
		return fmt.Errorf("no active escalation for alert %q", req.AlertID)
	}
	if esc.Closed {
		return fmt.Errorf("escalation already closed")
	}
	now := time.Now()
	esc.AckedBy = req.UserID
	esc.AckedAt = &now
	esc.Closed = true
	m.history = append(m.history, esc)
	delete(m.active, req.AlertID)
	m.log.Info("Alert %s acknowledged by %s", req.AlertID, req.UserID)
	return nil
}

// ListActive returns all currently active escalations.
func (m *EscalationManager) ListActive() []*ActiveEscalation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*ActiveEscalation, 0, len(m.active))
	for _, e := range m.active {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

// ListHistory returns the last N closed escalations.
func (m *EscalationManager) ListHistory(limit int) []*ActiveEscalation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > len(m.history) {
		limit = len(m.history)
	}
	// return the most recent
	return m.history[len(m.history)-limit:]
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal: escalation ticker loop
// ─────────────────────────────────────────────────────────────────────────────

func (m *EscalationManager) ticker() {
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-m.done:
			return
		case <-t.C:
			m.tick()
		}
	}
}

func (m *EscalationManager) tick() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for alertID, esc := range m.active {
		if esc.Closed {
			continue
		}
		policy, ok := m.policies[esc.PolicyID]
		if !ok {
			continue
		}

		// SLA breach check
		slaDeadline := esc.CreatedAt.Add(time.Duration(policy.SLAMins) * time.Minute)
		if !esc.SLABreached && now.After(slaDeadline) {
			esc.SLABreached = true
			m.log.Warn("SLA BREACH: alert=%s policy=%s sla=%dmin", alertID, policy.Name, policy.SLAMins)
			go m.notifier.SendAlert(
				fmt.Sprintf("⚠ SLA BREACH — %s", alertID),
				fmt.Sprintf("Alert %q has not been acknowledged within the %d-minute SLA defined by policy %q.", alertID, policy.SLAMins, policy.Name),
			)
		}

		// Time-based escalation: advance to next level if wait exceeded
		if esc.CurrentLevel < len(policy.Levels) {
			currentLevelDef := policy.Levels[esc.CurrentLevel-1]
			waitDur := time.Duration(currentLevelDef.WaitMins) * time.Minute
			if now.After(esc.LastEscalAt.Add(waitDur)) {
				esc.CurrentLevel++
				esc.LastEscalAt = now
				m.log.Info("Escalating alert=%s to level %d", alertID, esc.CurrentLevel)
				go m.notifyLevel(policy, esc.CurrentLevel, alertID)
			}
		}
	}
}

func (m *EscalationManager) notifyLevel(policy *EscalationPolicy, level int, alertID string) {
	if level < 1 || level > len(policy.Levels) {
		return
	}
	lvl := policy.Levels[level-1]
	title := fmt.Sprintf("[L%d ESCALATION] %s — Alert %s", level, policy.Name, alertID)
	body  := fmt.Sprintf(
		"Alert %q has been escalated to Level %d (%s).\nAssigned to: %v\nPolicy: %s\nSLA: %d min",
		alertID, level, lvl.Name, lvl.Users, policy.Name, policy.SLAMins,
	)
	m.notifier.SendAlert(title, body)
}

// ─────────────────────────────────────────────────────────────────────────────
// Built-in seed policy
// ─────────────────────────────────────────────────────────────────────────────

func (m *EscalationManager) seedDefaultPolicy() {
	m.policies["default"] = &EscalationPolicy{
		ID:         "default",
		Name:       "Default Security Escalation",
		AlertTypes: []string{"security_alert", "failed_login"},
		SLAMins:    30,
		Active:     true,
		Levels: []EscalationLevel{
			{Level: 1, Name: "Analyst",  Users: []string{"analyst@oblivra.io"},     Channel: "slack", WaitMins: 10},
			{Level: 2, Name: "Team Lead", Users: []string{"lead@oblivra.io"},       Channel: "email", WaitMins: 15},
			{Level: 3, Name: "Manager",  Users: []string{"manager@oblivra.io"},     Channel: "email", WaitMins: 20},
			{Level: 4, Name: "CISO",     Users: []string{"ciso@oblivra.io"},        Channel: "sms",   WaitMins: 999},
		},
	}
	m.schedules["primary"] = &OnCallSchedule{
		ID:   "primary",
		Name: "Primary On-Call Rotation",
		Entries: []OnCallEntry{
			{UserID: "analyst1", Name: "Alex (Analyst)", WeekdayStart: 1, WeekdayEnd: 5, HourStart: 8,  HourEnd: 18},
			{UserID: "analyst2", Name: "Jordan (Analyst)", WeekdayStart: 1, WeekdayEnd: 5, HourStart: 18, HourEnd: 8},
			{UserID: "oncall1",  Name: "Sam (Weekend)",  WeekdayStart: 0, WeekdayEnd: 0, HourStart: 0,  HourEnd: 24},
			{UserID: "oncall2",  Name: "Riley (Saturday)", WeekdayStart: 6, WeekdayEnd: 6, HourStart: 0, HourEnd: 24},
		},
	}
}
