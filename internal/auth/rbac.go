package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// contextKey is used to avoid collisions in context values.
// Always use typed keys — never plain strings — to prevent accidental collisions.
type contextKey string

const (
	// contextKeyUserAccount is the single authoritative key for the full IdentityUser.
	contextKeyUserAccount contextKey = "identity_user"
)

// ContextKeyUser is exported so apikey.go can write the UserAccount under a
// consistent typed key that avoids the string-key collision.
const ContextKeyUser contextKey = "api_user_account"

// IdentityUser represents the authenticated user in context
type IdentityUser struct {
	ID          string   `json:"id"`
	TenantID    string   `json:"tenant_id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	RoleID      string   `json:"role_id"`
	RoleName    string   `json:"role_name"`
	Permissions []string `json:"permissions"`
}

// RBACEngine evaluates permission checks against a user's role
type RBACEngine struct {
	log *logger.Logger
}

// NewRBACEngine creates a new RBAC enforcement engine
func NewRBACEngine(log *logger.Logger) *RBACEngine {
	return &RBACEngine{
		log: log.WithPrefix("rbac"),
	}
}

// HasPermission checks if the user has the specified permission
// Supports wildcard "*" for full access
func (e *RBACEngine) HasPermission(user *IdentityUser, required string) bool {
	if user == nil {
		return false
	}

	for _, perm := range user.Permissions {
		if perm == "*" {
			return true
		}
		if perm == required {
			return true
		}
		// Support domain wildcards: "hosts:*" matches "hosts:read"
		if strings.HasSuffix(perm, ":*") {
			domain := strings.TrimSuffix(perm, ":*")
			if strings.HasPrefix(required, domain+":") {
				return true
			}
		}
	}

	return false
}

// Enforce checks permission and returns an error if denied
func (e *RBACEngine) Enforce(user *IdentityUser, required string) error {
	if !e.HasPermission(user, required) {
		e.log.Warn("RBAC DENY: user=%s role=%s denied=%s", user.Email, user.RoleName, required)
		return fmt.Errorf("access denied: requires permission '%s'", required)
	}
	return nil
}

// ContextWithUser stores the identity user in a context
func ContextWithUser(ctx context.Context, user *IdentityUser) context.Context {
	return context.WithValue(ctx, contextKeyUserAccount, user)
}

// UserFromContext extracts the identity user from context
func UserFromContext(ctx context.Context) *IdentityUser {
	user, _ := ctx.Value(contextKeyUserAccount).(*IdentityUser)
	return user
}

// --- Permission Constants ---

const (
	PermHostsRead   = "hosts:read"
	PermHostsWrite  = "hosts:write"
	PermHostsDelete = "hosts:delete"

	PermSessionsRead  = "sessions:read"
	PermSessionsWrite = "sessions:write"

	PermSIEMRead  = "siem:read"
	PermSIEMWrite = "siem:write"

	PermIncidentsRead  = "incidents:read"
	PermIncidentsWrite = "incidents:write"

	PermEvidenceRead  = "evidence:read"
	PermEvidenceWrite = "evidence:write"

	PermSnippetsRead  = "snippets:read"
	PermSnippetsWrite = "snippets:write"

	PermComplianceRead  = "compliance:read"
	PermComplianceWrite = "compliance:write"

	PermUsersRead  = "users:read"
	PermUsersWrite = "users:write"

	PermRolesRead  = "roles:read"
	PermRolesWrite = "roles:write"

	PermSettingsRead  = "settings:read"
	PermSettingsWrite = "settings:write"

	PermClusterRead  = "cluster:read"
	PermClusterWrite = "cluster:write"
)
