package httpserver

import (
	"net/http"
	"strings"
)

// AuthMiddleware enforces API-key auth on every request unless the path is on
// the public allowlist. Keys are passed via OBLIVRA_API_KEYS=k1,k2,k3 (matched
// against `Authorization: Bearer <k>` or `X-API-Key: <k>`).
type AuthMiddleware struct {
	keys     map[string]struct{}
	exempt   map[string]struct{}
	required bool
}

func NewAuth(commaSeparatedKeys string) *AuthMiddleware {
	a := &AuthMiddleware{
		keys: map[string]struct{}{},
		exempt: map[string]struct{}{
			"/healthz":           {},
			"/readyz":            {},
			"/api/v1/auth/login": {},
		},
	}
	for _, k := range strings.Split(commaSeparatedKeys, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			a.keys[k] = struct{}{}
		}
	}
	a.required = len(a.keys) > 0
	return a
}

// Required reports whether the auth gate is active.
func (a *AuthMiddleware) Required() bool { return a.required }

func (a *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	if !a.required {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always allow the SPA shell + assets and the public allowlist.
		if _, ok := a.exempt[r.URL.Path]; ok || !isProtected(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		key := extractKey(r)
		if key == "" {
			writeError(w, http.StatusUnauthorized, "auth required")
			return
		}
		if _, ok := a.keys[key]; !ok {
			writeError(w, http.StatusForbidden, "invalid key")
			return
		}
		next.ServeHTTP(w, r)
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
