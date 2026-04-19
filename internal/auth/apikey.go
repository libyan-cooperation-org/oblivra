package auth

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// APIKeyMiddleware validates incoming HTTP requests against a list of authorized keys or JWTs.
type APIKeyMiddleware struct {
	validKeys map[string]UserAccount
	log       *logger.Logger
	jwtKeyFn  func() ([]byte, error)
}

// NewAPIKeyMiddleware initializes the auth guard with authorized keys and a dynamic JWT secret function.
func NewAPIKeyMiddleware(keys []string, log *logger.Logger, jwtKeyFn func() ([]byte, error)) *APIKeyMiddleware {
	valid := make(map[string]UserAccount)
	for _, k := range keys {
		// By default, system keys are given Admin. In a real DB they would map to specific user roles.
		valid[k] = UserAccount{
			ID:        "system",
			TenantID:  "GLOBAL",
			Username:  "ServiceAccount",
			Role:      RoleAdmin,
			Clearance: ClearanceTopSecret, // System gets highest clearance
		}
	}

	return &APIKeyMiddleware{
		validKeys: valid,
		log:       log,
		jwtKeyFn:  jwtKeyFn,
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

		// 3. Fallback: Query parameter for WebSockets
		if key == "" {
			key = r.URL.Query().Get("token")
		}

		if key == "" {
			m.log.Warn("[AUTH] Missing API key for %s %s", r.Method, r.URL.Path)
			http.Error(w, "Unauthorized: Missing API Key", http.StatusUnauthorized)
			return
		}

		user, isValid := m.isValid(key)
		if !isValid {
			// Discard the simple string match and try to parse as JWT
			if m.jwtKeyFn != nil {
				if secret, err := m.jwtKeyFn(); err == nil && len(secret) > 0 {
					token, err := jwt.Parse(key, func(token *jwt.Token) (interface{}, error) {
						if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
							return nil, fmt.Errorf("unexpected signing method")
						}
						return secret, nil
					})

					if err == nil && token.Valid {
						if claims, ok := token.Claims.(jwt.MapClaims); ok {
							// Successfully authenticated via JWT
							id, _ := claims["sub"].(string)
							tenant, _ := claims["tenant"].(string)
							email, _ := claims["email"].(string)
							role, _ := claims["role"].(string)
							
							identityUser := &IdentityUser{
								ID:       id,
								TenantID: tenant,
								Email:    email,
								Name:     email,
								RoleID:   role,
								RoleName: role,
							}
							
							// Derive permissions
							switch role {
							case string(RoleAdmin):
								identityUser.Permissions = []string{"*"}
							case string(RoleAnalyst):
								identityUser.Permissions = []string{
									PermHostsRead, PermSessionsRead,
									PermSIEMRead, PermSIEMWrite,
									PermIncidentsRead, PermIncidentsWrite,
									PermEvidenceRead, PermEvidenceWrite,
									PermSnippetsRead,
									PermComplianceRead,
									PermMCPExecute, PermMCPSimulate,
								}
							case string(RoleReadOnly):
								identityUser.Permissions = []string{
									PermHostsRead, PermSessionsRead,
									PermSIEMRead, PermIncidentsRead, PermEvidenceRead,
									PermSnippetsRead, PermComplianceRead,
								}
							}

							ctx := ContextWithUser(r.Context(), identityUser)
							ctx = database.WithTenant(ctx, identityUser.TenantID)
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
				}
			}

			m.log.Warn("[AUTH] Invalid API key or Token used for %s %s", r.Method, r.URL.Path)
			http.Error(w, "Unauthorized: Invalid API Key or Token", http.StatusUnauthorized)
			return
		}

		// Build an IdentityUser from the UserAccount so that RBACEngine.Enforce()
		// and UserFromContext() work correctly throughout handler chain.
		identityUser := apiKeyToIdentityUser(user)

		// Inject the IdentityUser under the unified UserContextKey
		ctx := ContextWithUser(r.Context(), identityUser)

		// Also inject the tenant_id string key for database repositories
		ctx = database.WithTenant(ctx, identityUser.TenantID)

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
			PermMCPExecute, PermMCPSimulate,
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
		TenantID:    u.TenantID,
		Email:       u.Username,
		Name:        u.Username,
		RoleID:      string(u.Role),
		RoleName:    string(u.Role),
		Permissions: perms,
	}
}

