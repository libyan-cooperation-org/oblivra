package agent

import (
	"errors"
	"testing"
	"time"
)

// TestRouter_PrimaryWins: when the primary succeeds, no fallback is
// tried.
func TestRouter_PrimaryWins(t *testing.T) {
	r := NewOutputRouter([]AgentOutput{
		{URL: "https://primary", Priority: 1},
		{URL: "https://backup", Priority: 2},
	})
	tried := []string{}
	url, err := r.Send(func(o AgentOutput) error {
		tried = append(tried, o.URL)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if url != "https://primary" {
		t.Errorf("expected primary, got %q", url)
	}
	if len(tried) != 1 {
		t.Errorf("expected only primary tried, got %d attempts", len(tried))
	}
}

// TestRouter_FailoverOnError: primary fails, backup succeeds — both
// tried, backup wins.
func TestRouter_FailoverOnError(t *testing.T) {
	r := NewOutputRouter([]AgentOutput{
		{URL: "https://primary", Priority: 1},
		{URL: "https://backup", Priority: 2},
	})
	url, err := r.Send(func(o AgentOutput) error {
		if o.URL == "https://primary" {
			return errors.New("connection refused")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected backup to succeed: %v", err)
	}
	if url != "https://backup" {
		t.Errorf("expected backup, got %q", url)
	}
}

// TestRouter_DemoteAfterRepeatedFailures: after MaxConsecutiveFailures
// the primary is demoted to the back of the rotation, so the next
// Send hits the backup first (saves a connection-attempt round trip).
func TestRouter_DemoteAfterRepeatedFailures(t *testing.T) {
	r := NewOutputRouter([]AgentOutput{
		{URL: "https://primary", Priority: 1},
		{URL: "https://backup", Priority: 2},
	})
	r.MaxConsecutiveFailures = 2 // tighter so test is fast
	r.DemotionWindow = time.Hour

	primaryFailing := func(o AgentOutput) error {
		if o.URL == "https://primary" {
			return errors.New("down")
		}
		return nil
	}
	// Burn through 2 failures of the primary (each Send walks
	// primary then backup; backup always succeeds).
	_, _ = r.Send(primaryFailing)
	_, _ = r.Send(primaryFailing)

	// Third Send: primary should now be demoted, so the first
	// endpoint tried is backup.
	first := ""
	_, err := r.Send(func(o AgentOutput) error {
		if first == "" {
			first = o.URL
		}
		return nil // both succeed now
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if first != "https://backup" {
		t.Errorf("expected backup tried first after demotion, got %q", first)
	}
}

// TestRouter_RecoveryClearsFailureCount: a single success on the
// primary clears the failure counter so the next Send tries it first
// again.
func TestRouter_RecoveryClearsFailureCount(t *testing.T) {
	r := NewOutputRouter([]AgentOutput{
		{URL: "https://primary", Priority: 1},
		{URL: "https://backup", Priority: 2},
	})

	// Bump the counter once.
	_, _ = r.Send(func(o AgentOutput) error {
		if o.URL == "https://primary" {
			return errors.New("blip")
		}
		return nil
	})
	// Now succeed on primary.
	first := ""
	_, _ = r.Send(func(o AgentOutput) error {
		if first == "" {
			first = o.URL
		}
		return nil
	})
	if first != "https://primary" {
		t.Errorf("primary should still be tried first after recovery, got %q", first)
	}
	// And health should report 0 consecutive failures.
	hs := r.Health()
	for _, h := range hs {
		if h.URL == "https://primary" && h.ConsecutiveFailures != 0 {
			t.Errorf("primary failure counter not cleared: %d", h.ConsecutiveFailures)
		}
	}
}

// TestRouter_AllFail: every output fails, last error returned.
func TestRouter_AllFail(t *testing.T) {
	r := NewOutputRouter([]AgentOutput{
		{URL: "https://a"}, {URL: "https://b"},
	})
	_, err := r.Send(func(o AgentOutput) error {
		return errors.New("nope")
	})
	if err == nil {
		t.Errorf("expected error when every output fails")
	}
}

// TestRouter_NilSafe: a nil router is a no-op.
func TestRouter_NilSafe(t *testing.T) {
	var r *OutputRouter
	url, err := r.Send(func(AgentOutput) error { return errors.New("should never be called") })
	if err != nil || url != "" {
		t.Errorf("nil router should be no-op; got url=%q err=%v", url, err)
	}
	if h := r.Health(); h != nil {
		t.Errorf("nil router Health should be nil, got %v", h)
	}
}
