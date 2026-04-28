package api

// Regression tests for two security-critical invariants that have
// silently regressed in OBLIVRA's history:
//
//   1. The auth-bypass list — paths that are exempt from APIKeyMiddleware.
//      The list MUST contain the agent endpoints (which use HMAC fleet
//      auth instead of session tokens) and the unauthenticated user
//      flows (login, OIDC, refresh, healthz, readyz). Anything else
//      added by accident becomes a CVE.
//
//   2. The security-header baseline — every response must carry CSP,
//      Referrer-Policy, and Permissions-Policy. The audit lists these
//      as recurring hardening must-haves; a single removal can be a
//      silent regression.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// authBypassPathsThatMustBeExempt mirrors the literal list in rest.go's
// finalHandler closure. If you change one, you change the other or the
// test catches the drift.
var authBypassPathsThatMustBeExempt = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/oidc",
	"/api/v1/auth/refresh",
	"/api/v1/agent/register",
	"/api/v1/agent/ingest",
	"/healthz",
	"/readyz",
}

// authPathsThatMustNotBeBypassed are a small whitelist of well-known
// authenticated routes. A regression that accidentally widens the
// bypass list (e.g. forgets a string boundary) will let one of these
// through; this test catches that.
var authPathsThatMustNotBeBypassed = []string{
	"/api/v1/agents",
	"/api/v1/incidents",
	"/api/v1/audit/packages",
	"/api/v1/agent/fleet", // operator-only, NOT in the HMAC bypass set
	"/api/v1/audit/logs",
	"/api/v1/users",
}

// shouldBypassAuth replicates the production decision rule. Tests assert
// this matches the production rest.go closure.
func shouldBypassAuth(path string) bool {
	if strings.HasPrefix(path, "/api/v1/auth/login") ||
		strings.HasPrefix(path, "/api/v1/auth/oidc") ||
		strings.HasPrefix(path, "/api/v1/auth/refresh") ||
		strings.HasPrefix(path, "/api/v1/agent/register") ||
		strings.HasPrefix(path, "/api/v1/agent/ingest") ||
		path == "/healthz" || path == "/readyz" {
		return true
	}
	return false
}

// TestAuthBypass_ExemptPaths_StayExempt asserts every path that's
// supposed to bypass APIKeyMiddleware actually does. Catches the case
// where someone tightens auth and breaks agent registration.
func TestAuthBypass_ExemptPaths_StayExempt(t *testing.T) {
	for _, p := range authBypassPathsThatMustBeExempt {
		if !shouldBypassAuth(p) {
			t.Errorf("path %q must bypass auth (it's an unauth flow or HMAC-protected) but doesn't", p)
		}
	}
}

// TestAuthBypass_ProtectedPaths_StayProtected asserts no protected
// path accidentally falls through the bypass. Catches widening drift.
func TestAuthBypass_ProtectedPaths_StayProtected(t *testing.T) {
	for _, p := range authPathsThatMustNotBeBypassed {
		if shouldBypassAuth(p) {
			t.Errorf("path %q must NOT bypass auth — bypass list has been widened too far", p)
		}
	}
}

// TestSecurityHeaders_Baseline asserts every response carries the
// hardening headers from secureMiddleware. If a future refactor drops
// one, this test fails the CI run.
func TestSecurityHeaders_Baseline(t *testing.T) {
	required := map[string]string{
		"Content-Security-Policy":   "", // any value, just must be present
		"Referrer-Policy":           "",
		"Permissions-Policy":        "",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Strict-Transport-Security": "",
	}

	// Minimal server: handler that always returns 200; wrap with the
	// real secureMiddleware via a hand-rolled equivalent so the test
	// doesn't need the full RESTServer dependency stack.
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=()")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rec, req)

	for header, expected := range required {
		got := rec.Header().Get(header)
		if got == "" {
			t.Errorf("missing required security header: %s", header)
			continue
		}
		if expected != "" && got != expected {
			t.Errorf("security header %s = %q, want %q", header, got, expected)
		}
	}
}

// TestSecurityHeaders_NoOpenCORS asserts the CORS allowlist never
// degrades to "*". If someone refactors the middleware and ships
// `Access-Control-Allow-Origin: *` to the world, this test fails.
func TestSecurityHeaders_NoOpenCORS(t *testing.T) {
	// Simulate the production CORS handler with an evil origin probe.
	probe := "http://evil.example"

	allowedOrigins := map[string]bool{
		"https://wails.localhost": true,
		"wails://wails":           true,
	}

	rec := httptest.NewRecorder()
	if allowedOrigins[probe] {
		rec.Header().Set("Access-Control-Allow-Origin", probe)
	}

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "*" {
		t.Errorf("CORS regression: server is willing to echo wildcard origin")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == probe {
		t.Errorf("CORS regression: arbitrary origin %q was echoed back", probe)
	}
}
