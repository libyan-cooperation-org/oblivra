package agent

import (
	"regexp"
	"strings"
	"sync"
)

// PIIRedactor scrubs sensitive data from event payloads before transmission.
// Runs at the edge (on-agent) to ensure PII never leaves the endpoint.
//
// IMPORTANT: IP addresses are intentionally NOT redacted here.
// Source IPs are essential security signals — redacting them would break
// SIEM detection rules that match on SourceIP, geo-enrichment, and TI lookups.
// IP redaction should be applied selectively at the query layer for regulated exports,
// not at the ingest layer where it would destroy detection signal.
type PIIRedactor struct {
	mu       sync.RWMutex
	patterns []*redactionRule
}

type redactionRule struct {
	name    string
	pattern *regexp.Regexp
	replace string
}

// NewPIIRedactor creates a redactor with standard PII patterns.
// Note: IPv4 addresses are excluded — see struct comment above.
func NewPIIRedactor() *PIIRedactor {
	return &PIIRedactor{
		patterns: []*redactionRule{
			{
				name:    "email",
				pattern: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
				replace: "[REDACTED_EMAIL]",
			},
			// IPv4 addresses intentionally omitted — they are security signals.
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
// The Host, Source, AgentID, and Type fields are never redacted — they are
// routing metadata required for correct SIEM indexing and fleet management.
func (r *PIIRedactor) RedactEvent(event *Event) {
	if event.Data == nil {
		return
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for key, val := range event.Data {
		if str, ok := val.(string); ok {
			event.Data[key] = r.redactString(str)
		}
	}
}

// RedactString applies all PII patterns to a string and returns the result.
// Thread-safe.
func (r *PIIRedactor) RedactString(s string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.redactString(s)
}

// redactString is the non-locking internal implementation.
func (r *PIIRedactor) redactString(s string) string {
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

// AddPattern adds a custom redaction rule. Thread-safe.
func (r *PIIRedactor) AddPattern(name, pattern, replacement string) error {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.patterns = append(r.patterns, &redactionRule{
		name:    name,
		pattern: compiled,
		replace: replacement,
	})
	return nil
}

// SensitiveFieldNames is the set of data field names that are fully replaced
// with [FULLY_REDACTED] regardless of their value.
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

// RedactSensitiveFields fully redacts any data field whose name matches
// SensitiveFieldNames. Called after RedactEvent for defence-in-depth.
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
