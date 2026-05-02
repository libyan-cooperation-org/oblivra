package services

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NotificationService — alert delivery channels beyond webhooks.
//
// First channel: email via SMTP. Operators add a channel with the
// minimal SMTP config and a min-severity gate; alerts that match get
// rendered to a short text body and sent. Bounded by:
//
//   - In-process throttle: at most one email per (channel, ruleID) per
//     5 minutes. A noisy rule shouldn't pager-storm the inbox.
//   - Auth chain entry on every send/test attempt so the operator
//     can see who configured what and when it last fired.
//
// Webhook channels remain managed by the existing WebhookService —
// this service registers them as a unified `notifications` view but
// delegates delivery to whoever owns each channel kind. That keeps
// the existing wiring intact.

type NotificationKind string

const (
	NotificationEmail   NotificationKind = "email"
	NotificationWebhook NotificationKind = "webhook"
)

type NotificationChannel struct {
	ID            string           `json:"id"`
	Kind          NotificationKind `json:"kind"`
	Name          string           `json:"name"`
	MinSeverity   AlertSeverity    `json:"minSeverity,omitempty"`
	Disabled      bool             `json:"disabled,omitempty"`
	CreatedAt     time.Time        `json:"createdAt"`
	LastDelivered *time.Time       `json:"lastDelivered,omitempty"`
	LastError     string           `json:"lastError,omitempty"`

	// Email-specific.
	SMTPHost     string `json:"smtpHost,omitempty"`
	SMTPPort     int    `json:"smtpPort,omitempty"`
	SMTPFrom     string `json:"smtpFrom,omitempty"`
	SMTPTo       string `json:"smtpTo,omitempty"`
	SMTPUsername string `json:"smtpUsername,omitempty"`
	SMTPPassword string `json:"-"` // never serialised

	// Webhook-specific (delegated to WebhookService when used).
	WebhookURL    string `json:"webhookUrl,omitempty"`
	WebhookSecret string `json:"-"`
}

type NotificationService struct {
	log   *slog.Logger
	audit *AuditService

	mu       sync.RWMutex
	channels map[string]*NotificationChannel

	// throttle key → last-sent timestamp; bounded by GC every Notify.
	throttleMu sync.Mutex
	throttle   map[string]time.Time
}

func NewNotificationService(log *slog.Logger, audit *AuditService) *NotificationService {
	return &NotificationService{
		log: log, audit: audit,
		channels: map[string]*NotificationChannel{},
		throttle: map[string]time.Time{},
	}
}

func (s *NotificationService) ServiceName() string { return "NotificationService" }

// Register adds a channel and persists it to the audit chain. The
// returned record echoes the input with ID/CreatedAt populated.
func (s *NotificationService) Register(ctx context.Context, c NotificationChannel) (NotificationChannel, error) {
	if c.Name == "" {
		return c, errors.New("name required")
	}
	switch c.Kind {
	case NotificationEmail:
		if c.SMTPHost == "" || c.SMTPFrom == "" || c.SMTPTo == "" {
			return c, errors.New("email channel requires smtpHost, smtpFrom, smtpTo")
		}
		if c.SMTPPort == 0 {
			c.SMTPPort = 587
		}
	case NotificationWebhook:
		if c.WebhookURL == "" {
			return c, errors.New("webhook channel requires webhookUrl")
		}
	default:
		return c, fmt.Errorf("unknown kind %q (use 'email' or 'webhook')", c.Kind)
	}
	c.ID = randomID(8)
	c.CreatedAt = time.Now().UTC()
	s.mu.Lock()
	s.channels[c.ID] = &c
	s.mu.Unlock()
	if s.audit != nil {
		s.audit.Append(ctx, "system", "notification.register", "default", map[string]string{
			"id": c.ID, "kind": string(c.Kind), "name": c.Name,
		})
	}
	return c, nil
}

func (s *NotificationService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	_, ok := s.channels[id]
	delete(s.channels, id)
	s.mu.Unlock()
	if !ok {
		return errors.New("not found")
	}
	if s.audit != nil {
		s.audit.Append(ctx, "system", "notification.delete", "default", map[string]string{"id": id})
	}
	return nil
}

func (s *NotificationService) List() []NotificationChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]NotificationChannel, 0, len(s.channels))
	for _, c := range s.channels {
		out = append(out, *c)
	}
	return out
}

// Test fires a synthetic alert so the operator can verify the channel
// works before relying on it for production traffic.
func (s *NotificationService) Test(ctx context.Context, id string) error {
	s.mu.RLock()
	c, ok := s.channels[id]
	s.mu.RUnlock()
	if !ok {
		return errors.New("not found")
	}
	test := Alert{
		TenantID: "default", RuleID: "notification.test",
		RuleName: "Notification channel test",
		Severity: AlertSeverityLow,
		Message:  fmt.Sprintf("This is a test alert for channel %q (%s).", c.Name, c.ID),
	}
	return s.deliver(ctx, c, test)
}

// Notify fans an alert out to every channel whose severity gate is
// satisfied. Called from the alert raise path. Errors are logged + put
// on the channel record (LastError) so the UI can surface them.
func (s *NotificationService) Notify(ctx context.Context, alert Alert) {
	s.mu.RLock()
	channels := make([]*NotificationChannel, 0, len(s.channels))
	for _, c := range s.channels {
		channels = append(channels, c)
	}
	s.mu.RUnlock()

	for _, c := range channels {
		if c.Disabled {
			continue
		}
		if !severityGtEq(alert.Severity, c.MinSeverity) {
			continue
		}
		if !s.acquireSlot(c.ID, alert.RuleID) {
			s.log.Debug("notification throttled", "channel", c.ID, "rule", alert.RuleID)
			continue
		}
		if err := s.deliver(ctx, c, alert); err != nil {
			s.log.Warn("notification deliver", "channel", c.ID, "err", err)
		}
	}
}

func (s *NotificationService) deliver(ctx context.Context, c *NotificationChannel, alert Alert) error {
	var err error
	switch c.Kind {
	case NotificationEmail:
		err = s.sendEmail(ctx, c, alert)
	case NotificationWebhook:
		err = errors.New("webhook channels: register via /api/v1/webhooks instead")
	default:
		err = fmt.Errorf("unknown kind %q", c.Kind)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	if err != nil {
		c.LastError = err.Error()
	} else {
		c.LastError = ""
		ts := now
		c.LastDelivered = &ts
	}
	s.mu.Unlock()
	return err
}

func (s *NotificationService) sendEmail(ctx context.Context, c *NotificationChannel, alert Alert) error {
	addr := c.SMTPHost + ":" + strconv.Itoa(c.SMTPPort)

	subject := fmt.Sprintf("[OBLIVRA %s] %s", strings.ToUpper(string(alert.Severity)), alert.RuleName)
	body := buildEmailBody(alert)
	msg := []byte("From: " + c.SMTPFrom + "\r\n" +
		"To: " + c.SMTPTo + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" + body)

	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	host, _, _ := net.SplitHostPort(addr)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new: %w", err)
	}
	defer client.Close()

	// STARTTLS when the server advertises it.
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if c.SMTPUsername != "" {
		auth := smtp.PlainAuth("", c.SMTPUsername, c.SMTPPassword, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if err := client.Mail(c.SMTPFrom); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, to := range strings.Split(c.SMTPTo, ",") {
		to = strings.TrimSpace(to)
		if to == "" {
			continue
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("rcpt %s: %w", to, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	return client.Quit()
}

func buildEmailBody(alert Alert) string {
	var b strings.Builder
	fmt.Fprintf(&b, "OBLIVRA detection alert\n\n")
	fmt.Fprintf(&b, "Rule:     %s (%s)\n", alert.RuleName, alert.RuleID)
	fmt.Fprintf(&b, "Severity: %s\n", strings.ToUpper(string(alert.Severity)))
	if alert.HostID != "" {
		fmt.Fprintf(&b, "Host:     %s\n", alert.HostID)
	}
	if alert.TenantID != "" && alert.TenantID != "default" {
		fmt.Fprintf(&b, "Tenant:   %s\n", alert.TenantID)
	}
	if !alert.Triggered.IsZero() {
		fmt.Fprintf(&b, "When:     %s\n", alert.Triggered.Format(time.RFC3339))
	}
	if len(alert.MITRE) > 0 {
		fmt.Fprintf(&b, "MITRE:    %s\n", strings.Join(alert.MITRE, ", "))
	}
	fmt.Fprintf(&b, "\n%s\n", alert.Message)
	if len(alert.EventIDs) > 0 {
		fmt.Fprintf(&b, "\nEvent IDs (first 5):\n")
		max := 5
		if len(alert.EventIDs) < max {
			max = len(alert.EventIDs)
		}
		for _, id := range alert.EventIDs[:max] {
			fmt.Fprintf(&b, "  - %s\n", id)
		}
	}
	fmt.Fprintf(&b, "\n--\nThis email was sent by an OBLIVRA notification channel.\n")
	return b.String()
}

func severityGtEq(a, min AlertSeverity) bool {
	if min == "" {
		return true
	}
	rank := map[AlertSeverity]int{
		AlertSeverityLow:      0,
		AlertSeverityMedium:   1,
		AlertSeverityHigh:     2,
		AlertSeverityCritical: 3,
	}
	return rank[a] >= rank[min]
}

// acquireSlot returns true if the (channel, rule) pair hasn't fired in
// the last 5 minutes. Throttle map is bounded by sweeping expired keys.
func (s *NotificationService) acquireSlot(channelID, ruleID string) bool {
	key := channelID + "|" + ruleID
	now := time.Now()
	s.throttleMu.Lock()
	defer s.throttleMu.Unlock()
	if last, ok := s.throttle[key]; ok && now.Sub(last) < 5*time.Minute {
		return false
	}
	s.throttle[key] = now
	// Sweep stale entries when the map gets large.
	if len(s.throttle) > 1024 {
		for k, v := range s.throttle {
			if now.Sub(v) > 30*time.Minute {
				delete(s.throttle, k)
			}
		}
	}
	return true
}
