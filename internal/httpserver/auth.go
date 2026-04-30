package httpserver

import (
	"net/http"
	"strings"

	"github.com/kingknull/oblivra/internal/rbac"
)

// AuthMiddleware enforces API-key auth on every request unless the path is on
// the public allowlist. Keys are passed via OBLIVRA_API_KEYS=
//   key1, key2:analyst, key3:readonly, agentkey:agent
// Each key may carry a ":role" suffix mapping it to one of the defined RBAC
// roles. Missing role defaults to "admin" for backwards compatibility.
type AuthMiddleware struct {
	subjects map[string]rbac.Subject
	exempt   map[string]struct{}
	required bool
}

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
	{"/api/v1/events", "GET", rbac.PermSiemRead},

	{"/api/v1/alerts", "GET", rbac.PermAlertsRead},
	{"/api/v1/detection/rules", "GET", rbac.PermRulesRead},
	{"/api/v1/detection/rules/reload", "POST", rbac.PermRulesWrite},
	{"/api/v1/mitre/heatmap", "GET", rbac.PermAlertsRead},

	{"/api/v1/threatintel/lookup", "GET", rbac.PermIntelRead},
	{"/api/v1/threatintel/indicators", "GET", rbac.PermIntelRead},
	{"/api/v1/threatintel/indicator", "POST", rbac.PermIntelWrite},

	{"/api/v1/audit/log", "GET", rbac.PermAuditRead},
	{"/api/v1/audit/verify", "GET", rbac.PermAuditRead},
	{"/api/v1/audit/packages/generate", "POST", rbac.PermAuditExport},

	{"/api/v1/agent/fleet", "GET", rbac.PermFleetRead},
	{"/api/v1/agent/register", "POST", rbac.PermFleetWrite},
	{"/api/v1/agent/ingest", "POST", rbac.PermSiemIngest},

	{"/api/v1/storage/promote", "POST", rbac.PermAdminAll},
}

func NewAuth(commaSeparatedKeys string) *AuthMiddleware {
	a := &AuthMiddleware{
		subjects: map[string]rbac.Subject{},
		exempt: map[string]struct{}{
			"/healthz":           {},
			"/readyz":            {},
			"/api/v1/auth/login": {},
		},
	}
	for _, raw := range strings.Split(commaSeparatedKeys, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		key, role := raw, rbac.RoleAdmin
		if i := strings.IndexByte(raw, ':'); i > 0 {
			key = raw[:i]
			role = rbac.Role(strings.TrimSpace(raw[i+1:]))
		}
		a.subjects[key] = rbac.Subject{ID: key[:min(len(key), 6)], Role: role, Tenant: "default"}
	}
	a.required = len(a.subjects) > 0
	return a
}

func (a *AuthMiddleware) Required() bool { return a.required }

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
			writeError(w, http.StatusUnauthorized, "auth required")
			return
		}
		sub, ok := a.subjects[key]
		if !ok {
			writeError(w, http.StatusForbidden, "invalid key")
			return
		}
		// Check route → required permission.
		if perm := requiredPerm(r.Method, r.URL.Path); perm != "" {
			if !sub.Role.HasPermission(perm) {
				writeError(w, http.StatusForbidden, "role "+string(sub.Role)+" lacks "+perm)
				return
			}
		}
		next.ServeHTTP(w, rbac.WithSubjectRequest(r, sub))
	})
}

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
