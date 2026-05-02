package services

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// Volatile substrings collapse into a stable template so repeated
// "same shape, different IP/PID/path" lines fingerprint identically.
func TestTemplate_NormalisesVolatileTokens(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			// ssh2 is a literal protocol identifier — \b only anchors
			// at the START so the digit attached to letters stays put.
			"Failed password for root from 10.0.0.42 port 22 ssh2",
			"Failed password for root from <IP> port <N> ssh2",
		},
		{
			`Connection from 192.168.1.5 to "/api/users/9d8e2b3f-a7c4-4f8b-92d4-cba1234567ab"`,
			"Connection from <IP> to <STR>",
		},
		{
			"GC pause 12.5ms at 2026-05-02T08:14:23.001Z heap=4f9c2a1d8e7b",
			"GC pause <N>ms at <TS> heap=<HEX>",
		},
	}
	for _, c := range cases {
		got := template(c.in)
		if got != c.want {
			t.Errorf("template(%q)\n  got  %q\n  want %q", c.in, got, c.want)
		}
	}
}

// Two events differing only in volatile (number/IP) tokens must
// share a fingerprint. We deliberately don't normalise usernames or
// other identifier-shaped words — over-normalising would let real
// anomalies hide behind a too-broad template.
func TestTemplate_StableAcrossInstances(t *testing.T) {
	a := template("Failed password for root from 10.0.0.1 port 22")
	b := template("Failed password for root from 10.0.0.99 port 1234")
	if a != b {
		t.Errorf("expected same template, got\n  %q\n  %q", a, b)
	}
}

// New fingerprint after warmup → alert. Re-emitting the same event
// → no further alert.
func TestObserve_AlertsOnFirstNewPattern(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	a := NewAnomalyService(logger, alerts)
	// Skip warmup for the test.
	a.warmupDuration = 0
	a.minSourceVolume = 1

	ev := events.Event{
		ID: "x", TenantID: "default", HostID: "web-01",
		Severity: events.SeverityError, EventType: "linux:auth",
		Message: "Failed password for root from 10.0.0.1 port 22",
	}
	if !a.Observe(context.Background(), ev) {
		t.Error("expected anomaly on first sighting")
	}
	if got := len(alerts.Recent(10)); got != 1 {
		t.Errorf("expected 1 alert, got %d", got)
	}
	// Same shape, different IP — already-seen fingerprint, no alert.
	ev2 := ev
	ev2.ID = "y"
	ev2.Message = "Failed password for root from 10.0.0.99 port 1234"
	if a.Observe(context.Background(), ev2) {
		t.Error("expected no alert on second sighting (same template)")
	}
	if got := len(alerts.Recent(10)); got != 1 {
		t.Errorf("alert count should still be 1, got %d", got)
	}
}

// Warmup gate suppresses anomalies in the first 30 minutes.
func TestObserve_WarmupSilent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	a := NewAnomalyService(logger, alerts)
	// Default 30-min warmup; reset start to right now.
	a.startedAt = time.Now().UTC()
	a.minSourceVolume = 1

	ev := events.Event{
		HostID: "web-01", Severity: events.SeverityError,
		EventType: "test", Message: "Brand new error string foo",
	}
	if a.Observe(context.Background(), ev) {
		t.Error("expected silence during warmup")
	}
	if got := len(alerts.Recent(10)); got != 0 {
		t.Errorf("no alerts expected during warmup; got %d", got)
	}
}

// minSourceVolume gate: a brand-new sourceType doesn't fire on its
// first event; we wait until we've baselined.
func TestObserve_MinSourceVolumeGate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	a := NewAnomalyService(logger, alerts)
	a.warmupDuration = 0
	a.minSourceVolume = 5

	mk := func(msg string) events.Event {
		return events.Event{
			HostID: "h", Severity: events.SeverityError,
			EventType: "test", Message: msg,
		}
	}
	// First 4 events: under volume threshold → no alerts even though
	// each has a unique fingerprint.
	for i, m := range []string{"err A", "err B", "err C", "err D"} {
		if a.Observe(context.Background(), mk(m)) {
			t.Errorf("event %d (%q) should not have alerted under volume gate", i, m)
		}
	}
	if got := len(alerts.Recent(10)); got != 0 {
		t.Errorf("no alerts expected under volume gate; got %d", got)
	}
	// 5th event, new template → alert.
	a.Observe(context.Background(), mk("err E"))
	if got := len(alerts.Recent(10)); got != 1 {
		t.Errorf("expected 1 alert after volume threshold, got %d", got)
	}
}

// Below-min-severity events are skipped — debug noise stays silent.
func TestObserve_SeverityGate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	alerts := NewAlertService(logger)
	a := NewAnomalyService(logger, alerts)
	a.warmupDuration = 0
	a.minSourceVolume = 1

	ev := events.Event{
		HostID: "h", Severity: events.SeverityInfo,
		EventType: "t", Message: "info noise",
	}
	if a.Observe(context.Background(), ev) {
		t.Error("info-severity events must not trigger anomaly")
	}
}
