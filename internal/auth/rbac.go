package auth

import (
	"context"
	"fmt"
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
	TenantID  string
	Username  string
	Role      Role
	Clearance Clearance
	AgentID   string // Binding for agent-to-host verification
}

// contextKey is used to avoid collisions in context values.
type contextKey string

const (
	// UserContextKey is the single authoritative key for the authenticated user and their metadata.
	UserContextKey contextKey = "oblivra_auth_user"
)

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
	if user == nil {
		e.log.Warn("RBAC DENY: no user in context, denied=%s", required)
		return fmt.Errorf("access denied: no authenticated user context found")
	}
	if !e.HasPermission(user, required) {
		e.log.Warn("RBAC DENY: user=%s role=%s denied=%s", user.Email, user.RoleName, required)
		return fmt.Errorf("access denied: requires permission '%s'", required)
	}
	return nil
}

// ContextWithUser stores the identity user in a context
func ContextWithUser(ctx context.Context, user *IdentityUser) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}

// UserFromContext extracts the identity user from context
func UserFromContext(ctx context.Context) *IdentityUser {
	user, _ := ctx.Value(UserContextKey).(*IdentityUser)
	return user
}

// GetRole is a high-level helper that extracts the role from the identity user in context.
func GetRole(ctx context.Context) Role {
	user := UserFromContext(ctx)
	if user == nil {
		return RoleReadOnly
	}
	return Role(user.RoleName)
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

	PermMCPExecute  = "mcp:execute"
	PermMCPSimulate = "mcp:simulate"
	PermMCPApprove  = "mcp:approve"

	PermVaultRead  = "vault:read"
	PermVaultWrite = "vault:write"
	PermVaultAdmin = "vault:admin"

	PermIdentityRead  = "identity:read"
	PermIdentityWrite = "identity:write"
	PermIdentityAdmin = "identity:admin"
)
