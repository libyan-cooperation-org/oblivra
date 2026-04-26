package dlp

import (
	"strings"
	"testing"
)

// TestScrub_SSN verifies the last-4 visible mask form.
func TestScrub_SSN(t *testing.T) {
	r := NewRedactor()
	got := r.Scrub("user ssn 123-45-6789 was leaked")
	if !strings.Contains(got, "***-**-6789") {
		t.Errorf("ssn not masked correctly: %s", got)
	}
}

// TestScrub_CreditCard verifies that real (Luhn-valid) numbers get
// scrubbed and arbitrary 16-digit IDs do NOT.
func TestScrub_CreditCard(t *testing.T) {
	r := NewRedactor()

	// 4111-1111-1111-1111 is the canonical Visa test card — Luhn-valid.
	masked := r.Scrub("paid via 4111-1111-1111-1111 today")
	if strings.Contains(masked, "4111-1111-1111-1111") {
		t.Errorf("Luhn-valid CC should be scrubbed: %s", masked)
	}
	if !strings.Contains(masked, "1111") {
		t.Errorf("last 4 should be preserved: %s", masked)
	}

	// 1234-5678-9012-3456 fails Luhn — should be left alone.
	asis := r.Scrub("session id 1234-5678-9012-3456")
	if !strings.Contains(asis, "1234-5678-9012-3456") {
		t.Errorf("non-CC numeric should not be redacted: %s", asis)
	}
}

// TestScrub_JWT_AWS_Bearer covers the full-mask rules.
func TestScrub_JWT_AWS_Bearer(t *testing.T) {
	r := NewRedactor()
	in := `Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0In0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U
key: AKIAIOSFODNN7EXAMPLE
api_key: ghp_aBCdEfGhIjKlMnOpQrStUvWxYz0123456789`
	out := r.Scrub(in)
	if strings.Contains(out, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("AWS key not redacted: %s", out)
	}
	if strings.Contains(out, "eyJhbGciOiJIUzI1NiJ9") {
		t.Errorf("JWT not redacted: %s", out)
	}
	if !strings.Contains(out, "[REDACTED:TOKEN]") &&
		!strings.Contains(out, "[REDACTED:JWT]") {
		t.Errorf("expected redaction marker not present: %s", out)
	}
}

// TestScrub_Email preserves the domain for forensic value.
func TestScrub_Email(t *testing.T) {
	r := NewRedactor()
	got := r.Scrub("alice.smith+work@example.com logged in")
	if !strings.Contains(got, "***@example.com") {
		t.Errorf("email scrubbed wrong: %s", got)
	}
}

// TestSetEnabled confirms a rule can be turned off without affecting
// the others.
func TestSetEnabled(t *testing.T) {
	r := NewRedactor()
	if !r.SetEnabled(RuleSSN, false) {
		t.Fatal("expected SetEnabled to find the rule")
	}
	got := r.Scrub("123-45-6789 alice@example.com")
	if !strings.Contains(got, "123-45-6789") {
		t.Errorf("SSN should NOT be scrubbed when rule disabled: %s", got)
	}
	if !strings.Contains(got, "***@example.com") {
		t.Errorf("email rule should still apply: %s", got)
	}
}

// TestScrubMap walks nested structures.
func TestScrubMap(t *testing.T) {
	r := NewRedactor()
	m := map[string]interface{}{
		"user":  "alice@example.com",
		"plain": "hello",
		"meta": map[string]interface{}{
			"ssn": "123-45-6789",
		},
	}
	r.ScrubMap(m)
	if u := m["user"].(string); !strings.Contains(u, "***@") {
		t.Errorf("top-level email not scrubbed: %s", u)
	}
	if u := m["meta"].(map[string]interface{})["ssn"].(string); !strings.Contains(u, "***-**-") {
		t.Errorf("nested ssn not scrubbed: %s", u)
	}
	if m["plain"] != "hello" {
		t.Errorf("non-PII string mutated: %v", m["plain"])
	}
}

// TestReport tracks scan + redaction counts for the dashboard.
func TestReport(t *testing.T) {
	r := NewRedactor()
	r.Scrub("alice@example.com")          // 1 redact
	r.Scrub("nothing sensitive here")     // 0 redacts
	r.Scrub("123-45-6789 then 4111111111111111") // 2 redacts (in 1 string)
	rep := r.Report()
	if rep.Scanned() != 3 {
		t.Errorf("scanned: got %d, want 3", rep.Scanned())
	}
	if rep.Redacted() != 2 {
		t.Errorf("redacted: got %d, want 2", rep.Redacted())
	}
	hits := rep.Hits()
	if hits[RuleEmail] != 1 {
		t.Errorf("email hits: got %d, want 1", hits[RuleEmail])
	}
}

// TestLuhnValid sanity-checks the Luhn helper.
func TestLuhnValid(t *testing.T) {
	cases := []struct {
		num   string
		valid bool
	}{
		{"4111111111111111", true},
		{"5500000000000004", true},
		{"340000000000009", true},
		{"6011000000000004", true},
		{"1234567890123456", false},
		{"123", false},
	}
	for _, c := range cases {
		if got := luhnValid(c.num); got != c.valid {
			t.Errorf("luhnValid(%s): got %v, want %v", c.num, got, c.valid)
		}
	}
}
