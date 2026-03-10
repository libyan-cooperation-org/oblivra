package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// Role represents an RBAC permission group
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleAnalyst  Role = "analyst"
	RoleReadOnly Role = "readonly"
	RoleAgent    Role = "agent"
)

// Clearance represents a Mandatory Access Control level
type Clearance int

const (
	ClearanceUnclassified Clearance = iota
	ClearanceConfidential
	ClearanceSecret
	ClearanceTopSecret
)

// UserAccount is a stub for future multi-user RBAC and MAC integration
type UserAccount struct {
	ID        string
	Username  string
	Role      Role
	Clearance Clearance
	AgentID   string // Binding for agent-to-host verification
}

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

		// Inject UserAccount into Context
		ctx := context.WithValue(r.Context(), "user", user)

		// Authorized, proceed to handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *APIKeyMiddleware) isValid(provided string) (*UserAccount, bool) {
	// Use constant-time comparison to prevent timing attacks
	for validKey, userAccount := range m.validKeys {
		if subtle.ConstantTimeCompare([]byte(validKey), []byte(provided)) == 1 {
			return &userAccount, true
		}
	}
	return nil, false
}

// GetRole is a helper that extracts the authorized role from the request context
func GetRole(ctx context.Context) Role {
	user, ok := ctx.Value("user").(*UserAccount)
	if !ok || user == nil {
		return RoleReadOnly
	}
	return user.Role
}
