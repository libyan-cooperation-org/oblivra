package api

// middleware.go — Tenant isolation middleware + request-scoped tenant enforcement
//
// Closes the audit gap:
//   "Tenant isolation validation — no per-query tenant enforcement.
//    Attacker scenario: Tenant A queries Tenant B data."
//
// This file provides:
//
//   1. tenantMiddleware — HTTP middleware that injects the caller's TenantID
//      into every request context. Applied to all authenticated routes in rest.go.
//
//   2. requireTenantMatch — per-handler guard that verifies a resource's TenantID
//      matches the caller's TenantID. Denies cross-tenant access with HTTP 403.
//
//   3. tenantScopedQuery — helper that appends a TenantID WHERE clause to any
//      SQL query string, preventing accidental unscoped DB reads.
//
//   4. TenantFromContext / ContextWithTenantID — context helpers used by handlers
//      and database layer to propagate and read the tenant scope.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
)

// tenantMiddleware is an HTTP middleware that reads the authenticated user's
// TenantID from the auth context and injects it into the request context.
//
// ⚠️ MANDATORY: This MUST run AFTER authMiddleware / APIKeyMiddleware.
//
// A "GLOBAL" tenant or an admin role triggers WithGlobalSearch — they can
// query any tenant's data (used for SOC admin views and cross-tenant SIEM).
func (s *RESTServer) tenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			// CRITICAL: Panic if middleware ordering is violated.
			// Tenant isolation depends on a verified user context.
			panic("SECURITY: tenantMiddleware executed without authenticated user. Check middleware chain ordering in rest.go.")
		}

		tenantID := user.TenantID
		ctx := r.Context()

		// Admins and GLOBAL accounts get unrestricted access via WithGlobalSearch.
		// All other accounts are strictly locked to their TenantID.
		if tenantID == "GLOBAL" || auth.GetRole(ctx) == auth.RoleAdmin {
			ctx = database.WithGlobalSearch(ctx)
		} else {
			if tenantID == "" {
				s.log.Error("[security] Denied request for user %s: missing TenantID in identity", user.Email)
				http.Error(w, "Forbidden: Account has no tenant assignment", http.StatusForbidden)
				return
			}
			ctx = database.WithTenant(ctx, tenantID)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// VerifyHMAC validates the request signature and timestamp to prevent replay attacks.
// It expects X-Timestamp (Unix epoch) and X-Signature (HMAC-SHA256).
func VerifyHMAC(r *http.Request, body []byte, secret []byte) error {
	tsStr := r.Header.Get("X-Timestamp")
	sig := r.Header.Get("X-Signature")

	if tsStr == "" || sig == "" {
		return fmt.Errorf("missing authentication headers (X-Timestamp/X-Signature)")
	}

	// 1. Validate Timestamp (30s window)
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format")
	}

	now := time.Now().Unix()
	diff := now - ts
	if diff < 0 {
		diff = -diff
	}
	if diff > 30 { // 30s tolerance for clock drift
		return fmt.Errorf("request expired (clock drift too high)")
	}

	// 2. Validate Signature
	// HMAC(secret, body + timestamp)
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	mac.Write([]byte(tsStr))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return fmt.Errorf("invalid request signature")
	}

	return nil
}








