// Package dlp does search-time field redaction. Sensitive substrings
// (credit cards, AWS keys, JWT tokens, "password=...") are masked in the
// event surface that the UI / REST search returns. The on-disk events are
// untouched so the audit chain still verifies — redaction is a UI affordance,
// not a mutation.
package dlp

import (
	"regexp"
	"strings"

	"github.com/kingknull/oblivra/internal/events"
)

type Pattern struct {
	Name string
	RE   *regexp.Regexp
	Mask string // function-style: "$1***" — captures preserved
}

var defaultPatterns = []Pattern{
	{
		Name: "credit-card",
		RE:   regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`),
		Mask: "[REDACTED:CC]",
	},
	{
		Name: "aws-access-key",
		RE:   regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
		Mask: "[REDACTED:AWS-KEY]",
	},
	{
		Name: "github-pat",
		RE:   regexp.MustCompile(`\bghp_[A-Za-z0-9]{30,}\b`),
		Mask: "[REDACTED:GH-PAT]",
	},
	{
		Name: "jwt",
		RE:   regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`),
		Mask: "[REDACTED:JWT]",
	},
	{
		Name: "password-kv",
		RE:   regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`),
		Mask: "$1=[REDACTED]",
	},
	{
		Name: "bearer-token",
		RE:   regexp.MustCompile(`(?i)Authorization\s*:\s*Bearer\s+\S+`),
		Mask: "Authorization: Bearer [REDACTED]",
	},
	{
		Name: "ssn",
		RE:   regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		Mask: "[REDACTED:SSN]",
	},
}

// Redactor applies the configured patterns to event payloads.
type Redactor struct {
	patterns []Pattern
}

func NewDefault() *Redactor {
	return &Redactor{patterns: defaultPatterns}
}

// Apply returns a redacted copy of the event. Hash + ID are preserved so the
// caller can still match against the audit chain or sealed evidence.
func (r *Redactor) Apply(ev events.Event) events.Event {
	out := ev
	out.Message = r.redact(ev.Message)
	out.Raw = r.redact(ev.Raw)
	if len(ev.Fields) > 0 {
		nf := make(map[string]string, len(ev.Fields))
		for k, v := range ev.Fields {
			nf[k] = r.redact(v)
		}
		out.Fields = nf
	}
	return out
}

// ApplyAll redacts every event in-place (returns a new slice).
func (r *Redactor) ApplyAll(in []events.Event) []events.Event {
	out := make([]events.Event, len(in))
	for i := range in {
		out[i] = r.Apply(in[i])
	}
	return out
}

func (r *Redactor) redact(s string) string {
	if s == "" {
		return s
	}
	for _, p := range r.patterns {
		s = p.RE.ReplaceAllString(s, p.Mask)
	}
	return s
}

// HasMatches returns true if any pattern fires on the input — useful for
// flagging events that contain candidate sensitive data without masking
// them (e.g. for an "events containing secrets" alert).
func (r *Redactor) HasMatches(s string) bool {
	if s == "" {
		return false
	}
	for _, p := range r.patterns {
		if p.RE.MatchString(s) {
			return true
		}
	}
	return false
}

// Names returns the list of configured pattern names.
func (r *Redactor) Names() []string {
	out := make([]string, 0, len(r.patterns))
	for _, p := range r.patterns {
		out = append(out, p.Name)
	}
	return out
}

// AddPattern appends a custom pattern. RE is compiled by the caller to keep
// AddPattern fail-safe — bad regex panics at call time.
func (r *Redactor) AddPattern(name, expr, mask string) error {
	re, err := regexp.Compile(expr)
	if err != nil {
		return err
	}
	r.patterns = append(r.patterns, Pattern{Name: name, RE: re, Mask: mask})
	return nil
}

// AsAlertReason returns a short note like "matches: credit-card, jwt" that
// can be attached to an alert message.
func (r *Redactor) AsAlertReason(s string) string {
	if s == "" {
		return ""
	}
	hits := []string{}
	for _, p := range r.patterns {
		if p.RE.MatchString(s) {
			hits = append(hits, p.Name)
		}
	}
	if len(hits) == 0 {
		return ""
	}
	return "matches: " + strings.Join(hits, ", ")
}
