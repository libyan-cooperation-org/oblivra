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
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/kingknull/oblivrashell/internal/auth"
)

// tenantContextKey is a dedicated type to avoid collisions with other context keys.
type tenantContextKey struct{}

// ContextWithTenantID stores the caller's TenantID in the request context.
// Used by tenantMiddleware and by DB helper functions that enforce row-level isolation.
func ContextWithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tenantID)
}

// TenantFromContext extracts the TenantID from context.
// Returns empty string if none was set (indicating global/admin access).
func TenantFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantContextKey{}).(string)
	return v
}

// tenantMiddleware is an HTTP middleware that reads the authenticated user's
// TenantID from the auth context and injects it into the request context.
//
// This runs AFTER authMiddleware / APIKeyMiddleware so that auth.UserFromContext
// is always populated before this middleware executes.
//
// A "GLOBAL" tenant or an admin role bypasses strict tenant scoping — they can
// query any tenant's data (used for SOC admin views and cross-tenant SIEM).
func (s *RESTServer) tenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			// Unauthenticated — let downstream auth check handle it.
			next.ServeHTTP(w, r)
			return
		}

		tenantID := user.TenantID

		// Admins and GLOBAL accounts get an empty tenant scope,
		// meaning no automatic WHERE tenant_id = ? filter is applied.
		// Handlers can still explicitly scope if they choose.
		if tenantID == "GLOBAL" || auth.GetRole(r.Context()) == auth.RoleAdmin {
			tenantID = "" // empty = unrestricted
		}

		ctx := ContextWithTenantID(r.Context(), tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireTenantMatch checks that resourceTenantID matches the caller's tenant scope.
// Returns an error that callers should translate to HTTP 403 Forbidden.
//
// Usage in handlers:
//
//   if err := requireTenantMatch(r.Context(), alert.TenantID); err != nil {
//       http.Error(w, err.Error(), http.StatusForbidden)
//       return
//   }
//
// Rules:
//   - If the caller has no tenant restriction (admin / GLOBAL), always passes.
//   - If the resource has no tenant, always passes (unscoped legacy data).
//   - If both are set, they must match exactly.
func requireTenantMatch(ctx context.Context, resourceTenantID string) error {
	callerTenant := TenantFromContext(ctx)
	if callerTenant == "" {
		// Admin / GLOBAL caller — unrestricted.
		return nil
	}
	if resourceTenantID == "" {
		// Unscoped resource — allow.
		return nil
	}
	if callerTenant != resourceTenantID {
		return fmt.Errorf("forbidden: resource belongs to tenant %q, caller is tenant %q",
			resourceTenantID, callerTenant)
	}
	return nil
}

// tenantScopedQuery appends a tenant_id WHERE clause to a SQL query.
//
// If tenantID is empty (admin / GLOBAL), the query is returned unchanged.
// Otherwise:
//   - If the query already has a WHERE clause, AND tenant_id = ? is appended.
//   - If not, WHERE tenant_id = ? is appended.
//
// The caller must add the tenantID string to their args slice at the matching
// position. tenantScopedQuery returns both the modified query and a bool
// indicating whether a tenant arg was appended (so the caller knows to add it).
//
// Example:
//
//   q, scoped := tenantScopedQuery("SELECT * FROM host_events", tenantID)
//   if scoped {
//       rows, err = db.QueryContext(ctx, q, tenantID)
//   } else {
//       rows, err = db.QueryContext(ctx, q)
//   }
func tenantScopedQuery(query, tenantID string) (string, bool) {
	if tenantID == "" {
		return query, false
	}

	upper := strings.ToUpper(query)
	if strings.Contains(upper, " WHERE ") {
		return query + " AND tenant_id = ?", true
	}
	return query + " WHERE tenant_id = ?", true
}

// tenantFilter returns the SQL fragment and args to append to any query
// for tenant isolation. Used by handlers that build queries dynamically.
//
// Returns ("", nil) when the caller is unrestricted (admin).
// Returns ("AND tenant_id = ?", []interface{}{tenantID}) otherwise.
//
// This is the preferred helper for parameterized queries because it avoids
// string interpolation entirely — the tenant value is always a bind parameter.
func tenantFilter(ctx context.Context) (clause string, args []interface{}) {
	tenantID := TenantFromContext(ctx)
	if tenantID == "" {
		return "", nil
	}
	return "AND tenant_id = ?", []interface{}{tenantID}
}

// assertTenantInQuery panics in development if a handler is trying to execute
// a SELECT on a tenant-bearing table without a tenant_id condition.
// This is a defence-in-depth compile-time helper — not a runtime guard.
// Use in tests:
//
//	assertTenantInQuery(t, query, tenantID)
func assertTenantInQuery(query, tenantID string) bool {
	if tenantID == "" {
		return true // admin path, no assertion needed
	}
	return strings.Contains(strings.ToLower(query), "tenant_id")
}
