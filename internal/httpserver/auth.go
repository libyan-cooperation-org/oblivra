package httpserver

import (
	"net/http"
	"strings"

	"github.com/kingknull/oblivra/internal/rbac"
	"github.com/kingknull/oblivra/internal/services"
)

// AuthMiddleware enforces API-key auth on every request unless the path is on
// the public allowlist. Keys are passed via OBLIVRA_API_KEYS=
//
//	key1, key2:analyst, key3:readonly, agentkey:agent, tenantAkey:analyst:tenant-a
//
// Each key may carry an optional ":role" suffix mapping it to one of
// the defined RBAC roles, and a further optional ":tenant" suffix that
// pins the key to a specific tenant id. Format is `key[:role[:tenant]]`.
// A key with no tenant suffix or a tenant of "*" (wildcard) can act on
// any tenant; otherwise the request's `tenant` query parameter (or
// JSON body field, depending on handler) MUST match the bound tenant
// or the request is rejected with 403. Missing role defaults to
// "admin" for backwards compatibility, missing tenant defaults to
// "default" (cross-tenant access requires explicit "*").
type AuthMiddleware struct {
	subjects map[string]rbac.Subject
	exempt   map[string]struct{}
	required bool

	// audit, when set, receives a signed entry for every cross-tenant
	// or unauthorized denial so an off-host monitoring stack can alert
	// on enumeration attempts. Nil-safe; deny logging is skipped when
	// unset.
	audit *services.AuditService
}

// tenantWildcard is the sentinel string that grants a key cross-tenant
// access. Storing it in rbac.Subject.Tenant keeps the type unchanged.
const tenantWildcard = "*"

// keyToPerm maps URL-path prefixes to required permissions. Anything not in
// the table is allowed for any authenticated principal — additional surfaces
// can register their needs via Require.
var pathPerm = []struct {
	prefix string
	method string // "" matches any
	perm   string
}{
	{"/api/v1/siem/ingest", "POST", rbac.PermSiemIngest},
	{"/api/v1/siem/search", "GET", rbac.PermSiemRead},
	{"/api/v1/siem/stats", "GET", rbac.PermSiemRead},
	{"/api/v1/siem/events", "GET", rbac.PermSiemRead}, // event detail page
	{"/api/v1/events", "GET", rbac.PermSiemRead},

	{"/api/v1/alerts", "", rbac.PermAlertsRead}, // GET list, POST {id}/ack/assign/resolve/reopen
	{"/api/v1/detection/rules", "GET", rbac.PermRulesRead},
	{"/api/v1/detection/rules/reload", "POST", rbac.PermRulesWrite},
	{"/api/v1/detection/rules/effectiveness", "GET", rbac.PermRulesRead},
	{"/api/v1/mitre/heatmap", "GET", rbac.PermAlertsRead},

	{"/api/v1/threatintel/lookup", "GET", rbac.PermIntelRead},
	{"/api/v1/threatintel/indicators", "GET", rbac.PermIntelRead},
	{"/api/v1/threatintel/indicator", "POST", rbac.PermIntelWrite},

	{"/api/v1/audit/log", "GET", rbac.PermAuditRead},
	{"/api/v1/audit/verify", "GET", rbac.PermAuditRead},
	{"/api/v1/audit/packages/generate", "POST", rbac.PermAuditExport},

	{"/api/v1/agent/fleet", "GET", rbac.PermFleetRead},
	{"/api/v1/agent/register", "POST", rbac.PermFleetWrite},
	{"/api/v1/agent/heartbeat", "POST", rbac.PermFleetWrite},
	{"/api/v1/agent/ingest", "POST", rbac.PermSiemIngest},

	{"/api/v1/storage/promote", "POST", rbac.PermAdminAll},

	// Phase 51-58 surfaces.
	{"/api/v1/categories", "GET", rbac.PermSiemRead},
	{"/api/v1/services/health", "GET", rbac.PermSiemRead},
	{"/api/v1/saved-searches", "", rbac.PermRulesRead},
	{"/api/v1/notifications", "", rbac.PermAdminAll},
	{"/api/v1/compliance/feed", "GET", rbac.PermAuditRead},
}

func NewAuth(commaSeparatedKeys string) *AuthMiddleware {
	a := &AuthMiddleware{
		subjects: map[string]rbac.Subject{},
		exempt: map[string]struct{}{
			"/healthz":                      {},
			"/readyz":                       {},
			"/metrics":                      {},
			// /metrics is intentionally allowlisted so a Prometheus
			// scraper can hit it without an auth token. It exposes only
			// the platform's own runtime + ingest counters, no event
			// data. If the operator wants /metrics behind auth too,
			// they front the server with a reverse proxy.
			"/api/v1/auth/login":            {},
			"/api/v1/auth/oidc/login":       {},
			"/api/v1/auth/oidc/callback":    {},
		},
	}
	for _, raw := range strings.Split(commaSeparatedKeys, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		// Format: key[:role[:tenant]]. We split at most twice so
		// keys containing colons (rare but legal — Splunk-style HEC
		// tokens are hex so it's not a concern in practice) don't
		// fragment unexpectedly.
		parts := strings.SplitN(raw, ":", 3)
		key := strings.TrimSpace(parts[0])
		role := rbac.RoleAdmin
		tenant := "default"
		if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
			role = rbac.Role(strings.TrimSpace(parts[1]))
		}
		if len(parts) >= 3 && strings.TrimSpace(parts[2]) != "" {
			tenant = strings.TrimSpace(parts[2])
		}
		a.subjects[key] = rbac.Subject{ID: key[:min(len(key), 6)], Role: role, Tenant: tenant}
	}
	a.required = len(a.subjects) > 0
	return a
}

func (a *AuthMiddleware) Required() bool { return a.required }

// AttachAudit wires the audit chain so denials emit an audit entry.
// Idempotent. Nil-safe.
func (a *AuthMiddleware) AttachAudit(audit *services.AuditService) {
	if a == nil {
		return
	}
	a.audit = audit
}

// recordDeny writes an "auth.deny" audit entry. Always-on best-effort;
// any error from the chain is swallowed so a denial path can never
// itself leak by raising a 500.
func (a *AuthMiddleware) recordDeny(r *http.Request, reason, subjectID, subjectTenant, attemptedTenant string) {
	if a == nil || a.audit == nil {
		return
	}
	detail := map[string]string{
		"method":          r.Method,
		"path":            r.URL.Path,
		"reason":          reason,
		"subject":         subjectID,
		"subjectTenant":   subjectTenant,
		"attemptedTenant": attemptedTenant,
		"remote":          strings.SplitN(r.RemoteAddr, ":", 2)[0],
		"status":          "403",
	}
	tenant := subjectTenant
	if tenant == "" {
		tenant = "default"
	}
	a.audit.Append(r.Context(), "auth", "auth.deny", tenant, detail)
}

func (a *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	if !a.required {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.exempt[r.URL.Path]; ok || !isProtected(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		key := extractKey(r)
		if key == "" {
			a.recordDeny(r, "missing-credential", "-", "-", "")
			writeError(w, http.StatusUnauthorized, "auth required")
			return
		}
		sub, ok := a.subjects[key]
		if !ok {
			a.recordDeny(r, "invalid-key", "-", "-", "")
			writeError(w, http.StatusForbidden, "invalid key")
			return
		}
		// Check route → required permission.
		if perm := requiredPerm(r.Method, r.URL.Path); perm != "" {
			if !sub.Role.HasPermission(perm) {
				a.recordDeny(r, "missing-perm:"+perm, sub.ID, sub.Tenant, "")
				writeError(w, http.StatusForbidden, "role "+string(sub.Role)+" lacks "+perm)
				return
			}
		}
		// Cross-tenant blast-radius defense: a key bound to a specific
		// tenant cannot read another tenant's data even if its role
		// permits the action. Wildcard ("*") subjects bypass this
		// check — that's the platform admin role.
		if sub.Tenant != tenantWildcard {
			if reqTenant := tenantFromRequest(r); reqTenant != "" && reqTenant != sub.Tenant {
				a.recordDeny(r, "cross-tenant", sub.ID, sub.Tenant, reqTenant)
				writeError(w, http.StatusForbidden,
					"key bound to tenant "+sub.Tenant+" cannot access tenant "+reqTenant)
				return
			}
			// If the request didn't carry a tenant id, rewrite it to
			// the bound tenant so handlers downstream can't accidentally
			// read across tenant boundaries due to a missing param.
			r = withForcedTenant(r, sub.Tenant)
		}
		next.ServeHTTP(w, rbac.WithSubjectRequest(r, sub))
	})
}

// isProtected returns true if the path requires auth. Everything under
// /api/ is gated. Static assets and the health/readiness/metrics
// endpoints are exempt by allowlist (see exempt map).
func isProtected(path string) bool {
	return strings.HasPrefix(path, "/api/")
}

func extractKey(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if h := r.Header.Get("X-API-Key"); h != "" {
		return strings.TrimSpace(h)
	}
	if q := r.URL.Query().Get("token"); q != "" {
		return q
	}
	return ""
}

func requiredPerm(method, path string) string {
	for _, p := range pathPerm {
		if p.method != "" && p.method != method {
			continue
		}
		if strings.HasPrefix(path, p.prefix) {
			return p.perm
		}
	}
	return ""
}

// tenantFromRequest pulls the tenant id the request is targeting from
// the standard places handlers look for it. Returns "" if no tenant
// scope was specified — the middleware then forces the bound tenant
// via withForcedTenant so handlers can't read across boundaries by
// accident.
func tenantFromRequest(r *http.Request) string {
	if v := strings.TrimSpace(r.URL.Query().Get("tenant")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("tenantId")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get("X-Tenant-ID")); v != "" {
		return v
	}
	return ""
}

// withForcedTenant rewrites the request so handlers downstream see the
// bound tenant in the standard `?tenant=` query param. We mutate a
// clone of the URL — never the original — so the request is safe to
// pass through subsequent middleware.
func withForcedTenant(r *http.Request, tenant string) *http.Request {
	u := *r.URL
	q := u.Query()
	q.Set("tenant", tenant)
	q.Set("tenantId", tenant)
	u.RawQuery = q.Encode()
	clone := r.Clone(r.Context())
	clone.URL = &u
	clone.Header = r.Header.Clone()
	clone.Header.Set("X-Tenant-ID", tenant)
	return clone
}
