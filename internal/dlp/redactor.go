// Package dlp — server-side Data Loss Prevention redactor.
//
// Phase 27.2.3 — closes the "cloud logs bypass agent redaction" gap.
// The agent has its own PII redactor (`internal/agent/redactor.go`)
// that scrubs at the edge, but events pulled from cloud sources
// (CloudTrail, Google Workspace, M365, etc.) NEVER touch an agent —
// they hit the ingest API directly. This package runs at the ingest
// layer, between event-bus subscribe and SIEM index, so EVERY event
// is scanned regardless of source.
//
// Patterns shipped by default:
//   - US Social Security Numbers     (NNN-NN-NNNN, with weak validation)
//   - Credit card numbers             (Luhn-validated, 13–19 digits)
//   - JWT tokens                      (header.payload.signature shape)
//   - AWS access key IDs              (AKIA + 16 base32 chars)
//   - Generic Bearer / API keys       ("Bearer <token>" headers)
//   - Email addresses                 (RFC-5322 simplified)
//
// Redaction strategy:
//   - SSN / CC numbers: mask all but last 4 digits → `***-**-1234`
//   - Tokens / keys:    full mask → `[REDACTED:JWT]`
//   - Emails:           preserve domain → `***@example.com`
//
// IPs are NOT scrubbed here for the same reason the agent doesn't:
// they're load-bearing security signals (geo, TI, correlation).
//
// Configuration:
//   - Each rule has an `Enabled` toggle so a tenant can turn off
//     individual patterns from the Settings UI.
//   - The redactor returns a `report` capturing per-rule hit counts
//     so the operator dashboard can show "this many SSNs / CCs were
//     scrubbed in the last 24h."

package dlp

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// RuleID is a stable identifier for a DLP rule. Used by the Settings
// UI to enable/disable specific patterns without depending on
// pattern ordering.
type RuleID string

const (
	RuleSSN          RuleID = "us_ssn"
	RuleCreditCard   RuleID = "credit_card"
	RuleJWT          RuleID = "jwt"
	RuleAWSAccessKey RuleID = "aws_access_key"
	RuleBearer       RuleID = "bearer_token"
	RuleEmail        RuleID = "email"
)

// Rule describes a single redaction pattern.
type Rule struct {
	ID      RuleID
	Pattern *regexp.Regexp
	// Replace is invoked with the matched substring and must return
	// the redacted form. Allows per-rule logic (Luhn validation,
	// email-domain preservation, last-4 unmask, etc.).
	Replace func(match string) string
	// Enabled controls whether the rule is applied at scan time.
	// Atomic int32 so it can be toggled from any goroutine without
	// holding a lock.
	Enabled atomic.Bool
}

// Report carries scan-time statistics for the dashboard.
type Report struct {
	mu       sync.Mutex
	hits     map[RuleID]uint64
	scanned  uint64
	redacted uint64 // count of fields that had at least one redaction
}

// Hits returns a snapshot of per-rule match counts.
func (r *Report) Hits() map[RuleID]uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[RuleID]uint64, len(r.hits))
	for k, v := range r.hits {
		out[k] = v
	}
	return out
}

// Scanned returns the total number of strings scanned.
func (r *Report) Scanned() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.scanned
}

// Redacted returns the total number of strings that had at least one match.
func (r *Report) Redacted() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.redacted
}

func (r *Report) note(rule RuleID, hits int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.hits == nil {
		r.hits = make(map[RuleID]uint64)
	}
	r.hits[rule] += uint64(hits)
}

// Redactor scrubs DLP-sensitive strings according to its rule set.
type Redactor struct {
	mu     sync.RWMutex
	rules  []*Rule
	report Report
}

// NewRedactor constructs a redactor preloaded with the default rules.
// All rules start enabled. Call SetEnabled to flip individual rules.
func NewRedactor() *Redactor {
	r := &Redactor{}
	for _, rule := range defaultRules() {
		rule.Enabled.Store(true)
		r.rules = append(r.rules, rule)
	}
	return r
}

// SetEnabled toggles a rule on or off. Returns false if the rule is
// not registered.
func (r *Redactor) SetEnabled(id RuleID, enabled bool) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rl := range r.rules {
		if rl.ID == id {
			rl.Enabled.Store(enabled)
			return true
		}
	}
	return false
}

// Report returns a pointer to the live scan report. Reports are
// cumulative for the redactor's lifetime — caller is responsible for
// snapshotting + resetting if running long-lived.
func (r *Redactor) Report() *Report { return &r.report }

// Scrub returns the input with every enabled rule applied. If no
// rules match, the original string is returned unchanged.
func (r *Redactor) Scrub(s string) string {
	if s == "" {
		return s
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.report.mu.Lock()
	r.report.scanned++
	r.report.mu.Unlock()

	out := s
	matched := false
	for _, rule := range r.rules {
		if !rule.Enabled.Load() {
			continue
		}
		var hits int
		out = rule.Pattern.ReplaceAllStringFunc(out, func(m string) string {
			hits++
			return rule.Replace(m)
		})
		if hits > 0 {
			matched = true
			r.report.note(rule.ID, hits)
		}
	}
	if matched {
		r.report.mu.Lock()
		r.report.redacted++
		r.report.mu.Unlock()
	}
	return out
}

// ScrubMap walks every string-typed value in `m` and replaces it
// with the scrubbed form. Non-string values pass through. Used by
// the ingest DAG node to walk an event's `Data` map without forcing
// callers to know the shape.
func (r *Redactor) ScrubMap(m map[string]interface{}) {
	for k, v := range m {
		switch t := v.(type) {
		case string:
			m[k] = r.Scrub(t)
		case map[string]interface{}:
			r.ScrubMap(t)
		case []interface{}:
			for i, child := range t {
				if s, ok := child.(string); ok {
					t[i] = r.Scrub(s)
				}
			}
		}
	}
}

// ── Default rules ────────────────────────────────────────────────────

func defaultRules() []*Rule {
	return []*Rule{
		{
			ID:      RuleSSN,
			Pattern: regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
			Replace: func(m string) string {
				// last 4 digits visible
				return "***-**-" + m[len(m)-4:]
			},
		},
		{
			ID: RuleCreditCard,
			// 13-19 digits with optional spaces or hyphens; we
			// validate via Luhn before redacting to avoid clobbering
			// arbitrary numeric IDs.
			Pattern: regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`),
			Replace: func(m string) string {
				digits := stripNonDigits(m)
				if !luhnValid(digits) {
					return m
				}
				if len(digits) < 4 {
					return strings.Repeat("*", len(m))
				}
				return strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
			},
		},
		{
			ID:      RuleJWT,
			Pattern: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`),
			Replace: func(string) string { return "[REDACTED:JWT]" },
		},
		{
			ID:      RuleAWSAccessKey,
			Pattern: regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
			Replace: func(string) string { return "[REDACTED:AWS_ACCESS_KEY]" },
		},
		{
			ID:      RuleBearer,
			Pattern: regexp.MustCompile(`(?i)\b(bearer|api[_-]?key|x-api-key)[\s:=]+["']?[A-Za-z0-9._-]{16,}["']?`),
			Replace: func(string) string { return "[REDACTED:TOKEN]" },
		},
		{
			ID:      RuleEmail,
			Pattern: regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@([A-Za-z0-9.-]+\.[A-Za-z]{2,})\b`),
			Replace: func(m string) string {
				at := strings.LastIndex(m, "@")
				if at <= 0 {
					return "[REDACTED:EMAIL]"
				}
				return "***@" + m[at+1:]
			},
		},
	}
}

// stripNonDigits returns only the decimal digits in s.
func stripNonDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// luhnValid implements the Luhn checksum used by credit-card numbers.
// Returns false on length mismatch or any non-digit input — caller
// passes the digit-stripped form.
func luhnValid(digits string) bool {
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		n, err := strconv.Atoi(string(digits[i]))
		if err != nil {
			return false
		}
		if double {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}
	return sum%10 == 0
}
