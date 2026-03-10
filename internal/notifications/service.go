package notifications

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ChannelType defines the notification medium
type ChannelType string

const (
	ChannelEmail    ChannelType = "email"
	ChannelTelegram ChannelType = "telegram"
	ChannelTwilio   ChannelType = "twilio"
	ChannelWebhook  ChannelType = "webhook"
	ChannelSlack    ChannelType = "slack"
	ChannelTeams    ChannelType = "teams"
)

// NotificationConfig holds all provider credentials (stored in Vault)
type NotificationConfig struct {
	// Email (SMTP)
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	ToEmail      string `json:"to_email"`

	// Telegram
	TelegramToken  string `json:"telegram_token"`
	TelegramChatID string `json:"telegram_chat_id"`

	// Twilio (SMS + WhatsApp)
	TwilioAccountSID string `json:"twilio_sid"`
	TwilioAuthToken  string `json:"twilio_token"`
	TwilioFromNumber string `json:"twilio_from"`
	ToPhoneNumber    string `json:"to_phone"`

	// Per-channel toggles
	EnableEmail    bool `json:"enable_email"`
	EnableTelegram bool `json:"enable_telegram"`
	EnableSMS      bool `json:"enable_sms"`
	EnableWhatsApp bool `json:"enable_whatsapp"`
	EnableWebhook  bool `json:"enable_webhook"`

	// Webhook
	WebhookURL     string            `json:"webhook_url"`
	WebhookHeaders map[string]string `json:"webhook_headers,omitempty"`
	WebhookSecret  string            `json:"webhook_secret,omitempty"` // HMAC-SHA256 signing

	// Jira
	EnableJira    bool   `json:"enable_jira"`
	JiraURL       string `json:"jira_url"`
	JiraUser      string `json:"jira_user"`
	JiraToken     string `json:"jira_token"`
	JiraProject   string `json:"jira_project"`
	JiraIssueType string `json:"jira_issue_type"`

	// ServiceNow
	EnableServiceNow bool   `json:"enable_servicenow"`
	ServiceNowURL    string `json:"servicenow_url"`
	ServiceNowUser   string `json:"servicenow_user"`
	ServiceNowPass   string `json:"servicenow_pass"`
	ServiceNowTable  string `json:"servicenow_table"`

	// Slack (first-class)
	EnableSlack    bool   `json:"enable_slack"`
	SlackWebhookURL string `json:"slack_webhook_url"`
	SlackChannel   string `json:"slack_channel,omitempty"`
	SlackBotToken  string `json:"slack_bot_token,omitempty"`

	// Microsoft Teams (first-class)
	EnableTeams    bool   `json:"enable_teams"`
	TeamsWebhookURL string `json:"teams_webhook_url"`
}

// NotificationService handles multi-channel alert dispatch
type NotificationService struct {
	mu     sync.RWMutex
	config NotificationConfig
	log    *logger.Logger
	client *http.Client
}

// NewNotificationService creates a new service instance
func NewNotificationService(log *logger.Logger) *NotificationService {
	return &NotificationService{
		log:    log,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// UpdateConfig replaces the active configuration
func (s *NotificationService) UpdateConfig(cfg NotificationConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg
	s.log.Info("Notification config updated (email=%v, telegram=%v, sms=%v, whatsapp=%v, slack=%v, teams=%v)",
		cfg.EnableEmail, cfg.EnableTelegram, cfg.EnableSMS, cfg.EnableWhatsApp, cfg.EnableSlack, cfg.EnableTeams)
}

// GetConfig returns the current configuration
func (s *NotificationService) GetConfig() NotificationConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// SendAlert broadcasts a critical alert to all enabled channels concurrently
func (s *NotificationService) SendAlert(title, message string) {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	var wg sync.WaitGroup

	if cfg.EnableEmail && cfg.SMTPHost != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.sendEmail(cfg, title, message); err != nil {
				s.log.Error("Email notification failed: %v", err)
			} else {
				s.log.Info("Email alert sent: %s", title)
			}
		}()
	}

	if cfg.EnableTelegram && cfg.TelegramToken != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.sendTelegram(cfg, title, message); err != nil {
				s.log.Error("Telegram notification failed: %v", err)
			} else {
				s.log.Info("Telegram alert sent: %s", title)
			}
		}()
	}

	if cfg.EnableSMS && cfg.TwilioAccountSID != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.sendTwilioSMS(cfg, message); err != nil {
				s.log.Error("SMS notification failed: %v", err)
			} else {
				s.log.Info("SMS alert sent: %s", title)
			}
		}()
	}

	if cfg.EnableWhatsApp && cfg.TwilioAccountSID != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.sendTwilioWhatsApp(cfg, message); err != nil {
				s.log.Error("WhatsApp notification failed: %v", err)
			} else {
				s.log.Info("WhatsApp alert sent: %s", title)
			}
		}()
	}

	if cfg.EnableWebhook && cfg.WebhookURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.sendWebhook(cfg, title, message); err != nil {
				s.log.Error("Webhook notification failed: %v", err)
			} else {
				s.log.Info("Webhook alert sent: %s", title)
			}
		}()
	}

	if cfg.EnableJira && cfg.JiraURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.sendJira(cfg, title, message); err != nil {
				s.log.Error("Jira issue creation failed: %v", err)
			} else {
				s.log.Info("Jira issue created: %s", title)
			}
		}()
	}

	if cfg.EnableServiceNow && cfg.ServiceNowURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.sendServiceNow(cfg, title, message); err != nil {
				s.log.Error("ServiceNow incident creation failed: %v", err)
			} else {
				s.log.Info("ServiceNow incident created: %s", title)
			}
		}()
	}

	if cfg.EnableSlack && cfg.SlackWebhookURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.sendSlack(cfg, title, message); err != nil {
				s.log.Error("Slack notification failed: %v", err)
			} else {
				s.log.Info("Slack alert sent: %s", title)
			}
		}()
	}

	if cfg.EnableTeams && cfg.TeamsWebhookURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.sendTeams(cfg, title, message); err != nil {
				s.log.Error("Teams notification failed: %v", err)
			} else {
				s.log.Info("Teams alert sent: %s", title)
			}
		}()
	}

	wg.Wait()
}

// --- Provider Implementations ---

func (s *NotificationService) sendEmail(cfg NotificationConfig, subject, body string) error {
	auth := smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: [OblivraShell Alert] %s\r\n"+
		"\r\n"+
		"%s\r\n", cfg.ToEmail, subject, body))

	return smtp.SendMail(addr, auth, cfg.SMTPUsername, []string{cfg.ToEmail}, msg)
}

func (s *NotificationService) sendTelegram(cfg NotificationConfig, subject, body string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.TelegramToken)

	text := fmt.Sprintf("🚨 *%s*\n\n%s", subject, body)

	values := url.Values{}
	values.Set("chat_id", cfg.TelegramChatID)
	values.Set("text", text)
	values.Set("parse_mode", "Markdown")

	resp, err := s.client.PostForm(apiURL, values)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram api returned %d", resp.StatusCode)
	}
	return nil
}

func (s *NotificationService) sendTwilioSMS(cfg NotificationConfig, body string) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", cfg.TwilioAccountSID)

	values := url.Values{}
	values.Set("To", cfg.ToPhoneNumber)
	values.Set("From", cfg.TwilioFromNumber)
	values.Set("Body", "🚨 [OblivraShell Alert]: "+body)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.TwilioAccountSID, cfg.TwilioAuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio sms returned %d", resp.StatusCode)
	}
	return nil
}

func (s *NotificationService) sendTwilioWhatsApp(cfg NotificationConfig, body string) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", cfg.TwilioAccountSID)

	to := cfg.ToPhoneNumber
	if !strings.HasPrefix(to, "whatsapp:") {
		to = "whatsapp:" + to
	}
	from := cfg.TwilioFromNumber
	if !strings.HasPrefix(from, "whatsapp:") {
		from = "whatsapp:" + from
	}

	values := url.Values{}
	values.Set("To", to)
	values.Set("From", from)
	values.Set("Body", "🚨 *OblivraShell Alert*\n\n"+body)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.TwilioAccountSID, cfg.TwilioAuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("twilio whatsapp returned %d", resp.StatusCode)
	}
	return nil
}

func (s *NotificationService) sendWebhook(cfg NotificationConfig, title, message string) error {
	payload := map[string]interface{}{
		"text":    fmt.Sprintf("[ALERT] %s\n%s", title, message),
		"title":   title,
		"message": message,
		"source":  "OblivraShell",
		"ts":      time.Now().UTC().Format(time.RFC3339),
	}

	// Slack/Discord/Teams specific formatting
	if strings.Contains(cfg.WebhookURL, "slack.com") {
		payload = map[string]interface{}{
			"attachments": []map[string]interface{}{
				{
					"fallback": fmt.Sprintf("%s: %s", title, message),
					"color":    "#ef4444", // Red for alerts
					"title":    title,
					"text":     message,
					"footer":   "OblivraShell Fleet Intelligence",
					"ts":       time.Now().Unix(),
				},
			},
		}
	} else if strings.Contains(cfg.WebhookURL, "discord.com") {
		payload = map[string]interface{}{
			"username": "OblivraShell",
			"embeds": []map[string]interface{}{
				{
					"title":       title,
					"description": message,
					"color":       15671044, // Discord Red
					"footer":      map[string]string{"text": "Fleet Intelligence"},
					"timestamp":   time.Now().UTC().Format(time.RFC3339),
				},
			},
		}
	} else if strings.Contains(cfg.WebhookURL, "office.com") || strings.Contains(cfg.WebhookURL, "microsoft.com") {
		payload = map[string]interface{}{
			"@type":      "MessageCard",
			"@context":   "http://schema.org/extensions",
			"themeColor": "EF4444",
			"summary":    title,
			"sections": []map[string]interface{}{
				{
					"activityTitle":    title,
					"activitySubtitle": "OblivraShell Alert",
					"text":             message,
				},
			},
		}
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", cfg.WebhookURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for k, v := range cfg.WebhookHeaders {
		req.Header.Set(k, v)
	}

	// HMAC-SHA256 signature if secret is configured
	if cfg.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Signature-256", "sha256="+sig)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

func (s *NotificationService) sendJira(cfg NotificationConfig, title, message string) error {
	apiURL := fmt.Sprintf("%s/rest/api/2/issue", strings.TrimSuffix(cfg.JiraURL, "/"))

	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": cfg.JiraProject,
			},
			"summary":     "[Oblivra ALERT] " + title,
			"description": message,
			"issuetype": map[string]string{
				"name": cfg.JiraIssueType,
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.JiraUser, cfg.JiraToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("jira returned %d", resp.StatusCode)
	}
	return nil
}

func (s *NotificationService) sendServiceNow(cfg NotificationConfig, title, message string) error {
	table := cfg.ServiceNowTable
	if table == "" {
		table = "incident"
	}
	apiURL := fmt.Sprintf("%s/api/now/table/%s", strings.TrimSuffix(cfg.ServiceNowURL, "/"), table)

	payload := map[string]interface{}{
		"short_description": "[Oblivra ALERT] " + title,
		"comments":          message,
		"severity":          "1", // Critical
		"urgency":           "1", // High
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.ServiceNowUser, cfg.ServiceNowPass)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("servicenow returned %d", resp.StatusCode)
	}
	return nil
}

func (s *NotificationService) sendSlack(cfg NotificationConfig, title, message string) error {
	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"fallback": fmt.Sprintf("[ALERT] %s: %s", title, message),
				"color":    "#ef4444",
				"title":    fmt.Sprintf("🚨 %s", title),
				"text":     message,
				"footer":   "OblivraShell Fleet Intelligence",
				"ts":       time.Now().Unix(),
				"fields": []map[string]interface{}{
					{"title": "Source", "value": "OblivraShell SIEM", "short": true},
					{"title": "Severity", "value": "Critical", "short": true},
				},
			},
		},
	}

	if cfg.SlackChannel != "" {
		payload["channel"] = cfg.SlackChannel
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", cfg.SlackWebhookURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if cfg.SlackBotToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.SlackBotToken)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack returned %d", resp.StatusCode)
	}
	return nil
}

func (s *NotificationService) sendTeams(cfg NotificationConfig, title, message string) error {
	// Microsoft Teams Incoming Webhook uses Office 365 MessageCard format
	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"themeColor": "EF4444",
		"summary":    fmt.Sprintf("[ALERT] %s", title),
		"sections": []map[string]interface{}{
			{
				"activityTitle":    fmt.Sprintf("🚨 %s", title),
				"activitySubtitle": "OblivraShell Fleet Intelligence",
				"text":             message,
				"facts": []map[string]string{
					{"name": "Source", "value": "OblivraShell SIEM"},
					{"name": "Severity", "value": "Critical"},
					{"name": "Timestamp", "value": time.Now().UTC().Format(time.RFC3339)},
				},
				"markdown": true,
			},
		},
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", cfg.TeamsWebhookURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("teams returned %d", resp.StatusCode)
	}
	return nil
}
