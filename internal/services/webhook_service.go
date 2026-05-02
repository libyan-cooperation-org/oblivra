// WebhookService delivers alerts to external endpoints (Slack-compatible
// incoming webhooks, generic JSON receivers). The platform itself does not
// take *automated response* actions — that's an explicit Phase 36 non-goal.
// Webhook delivery is purely informational: "a thing fired, here's the
// payload, your SOAR can decide what to do".
package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Webhook struct {
	ID            string        `json:"id"`
	URL           string        `json:"url"`
	Secret        string        `json:"secret,omitempty"` // HMAC-SHA256 signs the body if present
	MinSeverity   AlertSeverity `json:"minSeverity,omitempty"`
	IncludeRules  []string      `json:"includeRules,omitempty"`  // empty = all rules
	ExcludeRules  []string      `json:"excludeRules,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
	// LastDelivered is a pointer so the JSON omits the field entirely
	// when the webhook has never fired — `time.Time,omitempty` doesn't
	// elide the zero value, which marshals as "0001-01-01T00:00:00Z"
	// and lets the UI render an obviously-wrong "1/1/0001" timestamp.
	LastDelivered *time.Time `json:"lastDelivered,omitempty"`
	Disabled      bool       `json:"disabled,omitempty"`
}

type WebhookDelivery struct {
	WebhookID  string    `json:"webhookId"`
	AlertID    string    `json:"alertId"`
	Status     int       `json:"status"`
	Error      string    `json:"error,omitempty"`
	DeliveredAt time.Time `json:"deliveredAt"`
}

type WebhookService struct {
	log    *slog.Logger
	audit  *AuditService
	mu     sync.RWMutex
	hooks  map[string]*Webhook
	recent []WebhookDelivery
	client *http.Client
}

func NewWebhookService(log *slog.Logger, audit *AuditService) *WebhookService {
	return &WebhookService{
		log: log, audit: audit,
		hooks:  map[string]*Webhook{},
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *WebhookService) ServiceName() string { return "WebhookService" }

func (s *WebhookService) Register(w Webhook) (Webhook, error) {
	if w.URL == "" {
		return Webhook{}, errors.New("webhook url required")
	}
	if w.ID == "" {
		w.ID = randomID(6)
	}
	if w.CreatedAt.IsZero() {
		w.CreatedAt = time.Now().UTC()
	}
	s.mu.Lock()
	s.hooks[w.ID] = &w
	s.mu.Unlock()
	if s.audit != nil {
		s.audit.Append(context.Background(), "system", "webhook.register", "default", map[string]string{
			"id":  w.ID,
			"url": w.URL,
		})
	}
	return w, nil
}

func (s *WebhookService) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.hooks, id)
}

func (s *WebhookService) List() []Webhook {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Webhook, 0, len(s.hooks))
	for _, h := range s.hooks {
		// strip the secret on the way out so it doesn't leak through `list`
		w := *h
		if w.Secret != "" {
			w.Secret = "[set]"
		}
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *WebhookService) Recent(limit int) []WebhookDelivery {
	if limit <= 0 {
		limit = 50
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.recent)
	if limit > n {
		limit = n
	}
	out := make([]WebhookDelivery, 0, limit)
	for i := n - 1; i >= n-limit; i-- {
		out = append(out, s.recent[i])
	}
	return out
}

// Deliver is called by the alert-fan-out for every raised alert. Filters by
// MinSeverity / Include / Exclude before sending.
func (s *WebhookService) Deliver(ctx context.Context, a Alert) {
	s.mu.RLock()
	hooks := make([]*Webhook, 0, len(s.hooks))
	for _, h := range s.hooks {
		hooks = append(hooks, h)
	}
	s.mu.RUnlock()

	for _, h := range hooks {
		if h.Disabled {
			continue
		}
		if !severityMeets(a.Severity, h.MinSeverity) {
			continue
		}
		if !ruleMatches(a.RuleID, h.IncludeRules, h.ExcludeRules) {
			continue
		}
		go s.deliverOne(ctx, h, a)
	}
}

func (s *WebhookService) deliverOne(ctx context.Context, h *Webhook, a Alert) {
	body, err := json.Marshal(map[string]any{
		"id":         a.ID,
		"ruleId":     a.RuleID,
		"ruleName":   a.RuleName,
		"severity":   a.Severity,
		"hostId":     a.HostID,
		"message":    a.Message,
		"mitre":      a.MITRE,
		"triggered":  a.Triggered,
		"eventIds":   a.EventIDs,
		"tenantId":   a.TenantID,
	})
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.URL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if h.Secret != "" {
		mac := hmac.New(sha256.New, []byte(h.Secret))
		mac.Write(body)
		req.Header.Set("X-OBLIVRA-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := s.client.Do(req)
	delivery := WebhookDelivery{
		WebhookID:   h.ID,
		AlertID:     a.ID,
		DeliveredAt: time.Now().UTC(),
	}
	if err != nil {
		delivery.Error = err.Error()
	} else {
		delivery.Status = resp.StatusCode
		_ = resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			delivery.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}
	s.mu.Lock()
	s.recent = append(s.recent, delivery)
	if len(s.recent) > 500 {
		s.recent = s.recent[len(s.recent)-500:]
	}
	if delivery.Error == "" {
		ts := delivery.DeliveredAt
		h.LastDelivered = &ts
	}
	s.mu.Unlock()
}

func severityMeets(actual, threshold AlertSeverity) bool {
	rank := func(s AlertSeverity) int {
		switch s {
		case AlertSeverityCritical:
			return 4
		case AlertSeverityHigh:
			return 3
		case AlertSeverityMedium:
			return 2
		case AlertSeverityLow:
			return 1
		default:
			return 0
		}
	}
	if threshold == "" {
		return true
	}
	return rank(actual) >= rank(threshold)
}

func ruleMatches(ruleID string, include, exclude []string) bool {
	for _, x := range exclude {
		if x == ruleID {
			return false
		}
	}
	if len(include) == 0 {
		return true
	}
	for _, i := range include {
		if i == ruleID {
			return true
		}
	}
	return false
}
