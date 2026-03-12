package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/kingknull/oblivrashell/internal/logger"
)



// APIKeyMiddleware validates incoming HTTP requests against a list of authorized keys.
type APIKeyMiddleware struct {
	validKeys map[string]UserAccount
	log       *logger.Logger
}

// NewAPIKeyMiddleware initializes the auth guard with authorized keys.
// In a full feature, these keys would be loaded from Vault.
func NewAPIKeyMiddleware(keys []string, log *logger.Logger) *APIKeyMiddleware {
	valid := make(map[string]UserAccount)
	for _, k := range keys {
		// By default, system keys are given Admin. In a real DB they would map to specific user roles.
		valid[k] = UserAccount{
			ID:        "system",
			Username:  "ServiceAccount",
			Role:      RoleAdmin,
			Clearance: ClearanceTopSecret, // System gets highest clearance
		}
	}

	return &APIKeyMiddleware{
		validKeys: valid,
		log:       log,
	}
}

// Middleware wraps an http.Handler with Bearer token or X-API-Key validation.
func (m *APIKeyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check X-API-Key Header
		key := r.Header.Get("X-API-Key")

		// 2. Check Authorization: Bearer <token>
		if key == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				key = authHeader[7:]
			}
		}

		if key == "" {
			m.log.Warn("[AUTH] Missing API key for %s %s", r.Method, r.URL.Path)
			http.Error(w, "Unauthorized: Missing API Key", http.StatusUnauthorized)
			return
		}

		user, isValid := m.isValid(key)
		if !isValid {
			m.log.Warn("[AUTH] Invalid API key used for %s %s", r.Method, r.URL.Path)
			http.Error(w, "Unauthorized: Invalid API Key", http.StatusUnauthorized)
			return
		}

		// Build an IdentityUser from the UserAccount so that RBACEngine.Enforce()
		// and UserFromContext() work correctly throughout handler chain.
		identityUser := apiKeyToIdentityUser(user)

		// Inject the IdentityUser under the unified UserContextKey
		ctx := ContextWithUser(r.Context(), identityUser)

		// Authorized, proceed to handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *APIKeyMiddleware) isValid(provided string) (*UserAccount, bool) {
	// Scan ALL keys unconditionally to prevent timing side-channel that would
	// reveal the index of a matching key if we returned early on first match.
	var found *UserAccount
	for validKey, userAccount := range m.validKeys {
		if subtle.ConstantTimeCompare([]byte(validKey), []byte(provided)) == 1 {
			// Copy so the loop variable address is safe to take
			acc := userAccount
			found = &acc
			// Do NOT break — continue scanning all keys
		}
	}
	return found, found != nil
}

// apiKeyToIdentityUser converts a UserAccount (API-key auth) to the IdentityUser
// type used by RBACEngine and UserFromContext throughout the handler chain.
func apiKeyToIdentityUser(u *UserAccount) *IdentityUser {
	if u == nil {
		return nil
	}
	// Map role to permissions.
	perms := []string{}
	switch u.Role {
	case RoleAdmin:
		perms = []string{"*"} // wildcard — full access
	case RoleAnalyst:
		perms = []string{
			PermHostsRead, PermSessionsRead,
			PermSIEMRead, PermSIEMWrite,
			PermIncidentsRead, PermIncidentsWrite,
			PermEvidenceRead, PermEvidenceWrite,
			PermSnippetsRead,
			PermComplianceRead,
		}
	case RoleReadOnly:
		perms = []string{
			PermHostsRead, PermSessionsRead,
			PermSIEMRead, PermIncidentsRead, PermEvidenceRead,
			PermSnippetsRead, PermComplianceRead,
		}
	case RoleAgent:
		perms = []string{PermSIEMWrite} // agents may only ingest events
	}
	return &IdentityUser{
		ID:          u.ID,
		Email:       u.Username,
		Name:        u.Username,
		RoleID:      string(u.Role),
		RoleName:    string(u.Role),
		Permissions: perms,
	}
}

