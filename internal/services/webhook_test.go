package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWebhookDeliveryAndHMAC stands up a httptest receiver, registers it as
// a webhook with a secret, raises an alert, and confirms the receiver got a
// signed body whose HMAC verifies. Also confirms the body shape is what
// downstream tools (Slack-compatible receivers) expect.
func TestWebhookDeliveryAndHMAC(t *testing.T) {
	const secret = "webhook-secret-deadbeef"
	var got atomic.Int32
	var lastSig atomic.Pointer[string]
	var lastBody atomic.Pointer[[]byte]

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sig := r.Header.Get("X-OBLIVRA-Signature")
		lastSig.Store(&sig)
		lastBody.Store(&body)
		got.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	hooks := NewWebhookService(logger, nil)
	if _, err := hooks.Register(Webhook{URL: srv.URL, Secret: secret, MinSeverity: AlertSeverityMedium}); err != nil {
		t.Fatal(err)
	}

	hooks.Deliver(context.Background(), Alert{
		ID: "abc", RuleID: "test-rule", RuleName: "Test", Severity: AlertSeverityHigh,
		HostID: "h", Message: "boom", Triggered: time.Now().UTC(),
		EventIDs: []string{"e1"}, TenantID: "default",
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && got.Load() == 0 {
		time.Sleep(20 * time.Millisecond)
	}
	if got.Load() == 0 {
		t.Fatal("webhook not delivered")
	}

	body := *lastBody.Load()
	sig := *lastSig.Load()
	if sig == "" {
		t.Fatal("signature header missing")
	}
	const prefix = "sha256="
	if len(sig) <= len(prefix) || sig[:len(prefix)] != prefix {
		t.Fatalf("bad signature shape %q", sig)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := prefix + hex.EncodeToString(mac.Sum(nil))
	if sig != want {
		t.Fatalf("HMAC mismatch:\n  got  %s\n  want %s", sig, want)
	}

	// Body shape sanity check.
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"id", "ruleId", "ruleName", "severity", "hostId", "message", "triggered", "eventIds", "tenantId"} {
		if _, ok := doc[k]; !ok {
			t.Errorf("body missing field %q", k)
		}
	}
}

func TestWebhookSeverityFilter(t *testing.T) {
	var got atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		got.Add(1)
	}))
	defer srv.Close()

	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	hooks := NewWebhookService(logger, nil)
	hooks.Register(Webhook{URL: srv.URL, MinSeverity: AlertSeverityHigh})

	// Low + medium must NOT trigger; High + critical must.
	for _, sev := range []AlertSeverity{AlertSeverityLow, AlertSeverityMedium} {
		hooks.Deliver(context.Background(), Alert{ID: "x", Severity: sev})
	}
	time.Sleep(100 * time.Millisecond)
	if got.Load() != 0 {
		t.Errorf("expected zero deliveries below threshold, got %d", got.Load())
	}

	hooks.Deliver(context.Background(), Alert{ID: "y", Severity: AlertSeverityHigh})
	hooks.Deliver(context.Background(), Alert{ID: "z", Severity: AlertSeverityCritical})
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) && got.Load() < 2 {
		time.Sleep(20 * time.Millisecond)
	}
	if got.Load() != 2 {
		t.Errorf("expected 2 deliveries at threshold, got %d", got.Load())
	}
}

// TestWebhookConcurrentDelivery stresses the goroutine-per-alert fan-out
// and confirms recent-deliveries bookkeeping doesn't lose entries.
func TestWebhookConcurrentDelivery(t *testing.T) {
	var got atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		got.Add(1)
		time.Sleep(2 * time.Millisecond) // simulate slow receiver
	}))
	defer srv.Close()

	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	hooks := NewWebhookService(logger, nil)
	hooks.Register(Webhook{URL: srv.URL})

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			hooks.Deliver(context.Background(), Alert{
				ID: "id", Severity: AlertSeverityHigh,
			})
		}(i)
	}
	wg.Wait()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && got.Load() < n {
		time.Sleep(20 * time.Millisecond)
	}
	if got.Load() != n {
		t.Errorf("delivered %d/%d", got.Load(), n)
	}
	if r := hooks.Recent(n + 10); len(r) != n {
		t.Errorf("recent count = %d, want %d", len(r), n)
	}
}
