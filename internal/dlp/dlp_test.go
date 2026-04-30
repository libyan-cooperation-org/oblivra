package dlp

import (
	"strings"
	"testing"

	"github.com/kingknull/oblivra/internal/events"
)

func TestRedactsCommonSecrets(t *testing.T) {
	r := NewDefault()
	cases := []struct {
		in   string
		want string
	}{
		{"card 4111 1111 1111 1111 used", "[REDACTED:CC]"},
		{"AKIAIOSFODNN7EXAMPLE", "[REDACTED:AWS-KEY]"},
		{"ghp_abcdefghijklmnopqrstuvwxyz1234567890", "[REDACTED:GH-PAT]"},
		{"jwt eyJabcdefgh.eyJabcdefgh.SflKxwRJSMeKKF.token", "[REDACTED:JWT]"},
		{"password=hunter2", "[REDACTED]"},
		{"Authorization: Bearer abc123", "Authorization: Bearer [REDACTED]"},
		{"ssn 123-45-6789", "[REDACTED:SSN]"},
	}
	for _, c := range cases {
		got := r.Apply(events.Event{Message: c.in}).Message
		if !strings.Contains(got, c.want) {
			t.Errorf("redact(%q) = %q (want it to contain %q)", c.in, got, c.want)
		}
	}
}

func TestPreservesNonSensitive(t *testing.T) {
	r := NewDefault()
	in := events.Event{Message: "ordinary log line with no secrets"}
	out := r.Apply(in)
	if out.Message != in.Message {
		t.Errorf("unexpected mutation: %q → %q", in.Message, out.Message)
	}
}

func TestApplyToFields(t *testing.T) {
	r := NewDefault()
	in := events.Event{Fields: map[string]string{
		"user":     "alice",
		"password": "secret-value-123",
		"card":     "4111-1111-1111-1111",
	}}
	out := r.Apply(in)
	// "password" field value gets redacted via the password-kv pattern only if
	// the field text matches that pattern. The bare value "secret-value-123"
	// won't trigger the kv pattern, so we check that the card number was masked.
	if out.Fields["card"] == in.Fields["card"] {
		t.Errorf("card field not redacted: %q", out.Fields["card"])
	}
}

func TestAsAlertReason(t *testing.T) {
	r := NewDefault()
	got := r.AsAlertReason("password=foo card 4111 1111 1111 1111")
	if !strings.Contains(got, "credit-card") || !strings.Contains(got, "password-kv") {
		t.Errorf("expected both reasons, got %q", got)
	}
}

func TestHashStableAfterRedaction(t *testing.T) {
	r := NewDefault()
	ev := &events.Event{Message: "AKIAIOSFODNN7EXAMPLE", HostID: "h"}
	_ = ev.Validate()
	original := ev.Hash
	red := r.Apply(*ev)
	// Original event's stored Hash refers to the un-redacted message; the
	// redacted copy's Hash field is unchanged but VerifyHash() will fail
	// because the message differs. That's by design — the cryptographic
	// identity stays anchored to the on-disk content, while the UI surface
	// is masked.
	if ev.Hash != original {
		t.Errorf("on-disk hash mutated by redaction: %q != %q", ev.Hash, original)
	}
	if red.VerifyHash() {
		t.Errorf("redacted view should NOT verify against the original hash")
	}
}
