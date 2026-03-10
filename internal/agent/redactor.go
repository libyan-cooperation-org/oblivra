package agent

import (
	"regexp"
	"strings"
)

// PIIRedactor scrubs sensitive data from event payloads before transmission.
// Runs at the edge (on-agent) to ensure PII never leaves the endpoint.
type PIIRedactor struct {
	patterns []*redactionRule
}

type redactionRule struct {
	name    string
	pattern *regexp.Regexp
	replace string
}

// NewPIIRedactor creates a redactor with standard PII patterns.
func NewPIIRedactor() *PIIRedactor {
	return &PIIRedactor{
		patterns: []*redactionRule{
			{
				name:    "email",
				pattern: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
				replace: "[REDACTED_EMAIL]",
			},
			{
				name:    "ipv4",
				pattern: regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
				replace: "[REDACTED_IP]",
			},
			{
				name:    "ssn",
				pattern: regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
				replace: "[REDACTED_SSN]",
			},
			{
				name:    "credit_card",
				pattern: regexp.MustCompile(`\b(?:\d{4}[\s\-]?){3}\d{4}\b`),
				replace: "[REDACTED_CC]",
			},
			{
				name:    "api_key",
				pattern: regexp.MustCompile(`(?i)(api[_\-]?key|token|secret|password|bearer)\s*[:=]\s*\S+`),
				replace: "[REDACTED_SECRET]",
			},
			{
				name:    "jwt",
				pattern: regexp.MustCompile(`eyJ[a-zA-Z0-9_\-]+\.eyJ[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+`),
				replace: "[REDACTED_JWT]",
			},
			{
				name:    "private_key_header",
				pattern: regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE KEY-----`),
				replace: "[REDACTED_PRIVATE_KEY]",
			},
			{
				name:    "aws_key",
				pattern: regexp.MustCompile(`(?i)AKIA[0-9A-Z]{16}`),
				replace: "[REDACTED_AWS_KEY]",
			},
		},
	}
}

// RedactEvent scrubs PII from all string fields in an event's Data map.
func (r *PIIRedactor) RedactEvent(event *Event) {
	if event.Data == nil {
		return
	}
	for key, val := range event.Data {
		if str, ok := val.(string); ok {
			event.Data[key] = r.RedactString(str)
		}
	}
	// Redact source field if it contains PII
	event.Source = r.RedactString(event.Source)
}

// RedactString applies all PII patterns to a string.
func (r *PIIRedactor) RedactString(s string) string {
	for _, rule := range r.patterns {
		s = rule.pattern.ReplaceAllString(s, rule.replace)
	}
	return s
}

// RedactEvents processes a batch of events in-place.
func (r *PIIRedactor) RedactEvents(events []Event) {
	for i := range events {
		r.RedactEvent(&events[i])
	}
}

// AddPattern adds a custom redaction rule.
func (r *PIIRedactor) AddPattern(name, pattern, replacement string) error {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	r.patterns = append(r.patterns, &redactionRule{
		name:    name,
		pattern: compiled,
		replace: replacement,
	})
	return nil
}

// SensitiveFieldNames returns field names that should be fully redacted.
var SensitiveFieldNames = map[string]bool{
	"password":      true,
	"passwd":        true,
	"secret":        true,
	"token":         true,
	"api_key":       true,
	"apikey":        true,
	"private_key":   true,
	"access_token":  true,
	"refresh_token": true,
	"session_id":    true,
	"cookie":        true,
}

// RedactSensitiveFields fully redacts any field whose name matches sensitive patterns.
func (r *PIIRedactor) RedactSensitiveFields(event *Event) {
	if event.Data == nil {
		return
	}
	for key := range event.Data {
		if SensitiveFieldNames[strings.ToLower(key)] {
			event.Data[key] = "[FULLY_REDACTED]"
		}
	}
}
