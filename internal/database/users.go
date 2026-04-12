package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// User represents an identity within a tenant
type User struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	Email             string    `json:"email"`
	Name              string    `json:"name"`
	PasswordHash      string    `json:"-"`
	AuthProvider      string    `json:"auth_provider"`
	IsMFAEnabled      bool      `json:"is_mfa_enabled"`
	MFASecret         string    `json:"-"`
	RoleID            string    `json:"role_id"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
	LastLoginAt       string    `json:"last_login_at"`
	CriticalityScore  int       `json:"criticality_score"`
	CriticalityReason string    `json:"criticality_reason"`

	// SCIM Normalization fields
	ExternalID        string    `json:"external_id"`
	Active            bool      `json:"active"`
	DisplayName       string    `json:"display_name"`
	UserType          string    `json:"user_type"`
	Title             string    `json:"title"`
	Department        string    `json:"department"`
	Organization      string    `json:"organization"`
	PreferredLanguage string    `json:"preferred_language"`
	GroupsJSON        string    `json:"groups_json"`
	SCIMAttributes    string    `json:"scim_attributes_json"`
}

// Role defines a set of permissions
type Role struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	IsSystem    bool      `json:"is_system"` // System roles cannot be deleted
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// UserRepository handles user data persistence
type UserRepository struct {
	db DatabaseStore
}

func NewUserRepository(db DatabaseStore) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts a new user
func (r *UserRepository) CreateUser(ctx context.Context, u *User) error {
	r.db.Lock()
	defer r.db.Unlock()

	now := time.Now().Format(time.RFC3339)
	u.CreatedAt = now
	u.UpdatedAt = now
	u.TenantID = TenantFromContext(ctx)

	if u.ID == "" {
		u.ID = uuid.New().String()
	}

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO users (
			id, tenant_id, email, name, password_hash, auth_provider,
			is_mfa_enabled, mfa_secret, role_id, created_at, updated_at,
			external_id, active, display_name, user_type, title,
			department, organization, preferred_language, groups_json, scim_attributes_json,
			criticality_score, criticality_reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, u.ID, u.TenantID, u.Email, u.Name, u.PasswordHash, u.AuthProvider,
		u.IsMFAEnabled, u.MFASecret, u.RoleID, u.CreatedAt, u.UpdatedAt,
		u.ExternalID, u.Active, u.DisplayName, u.UserType, u.Title,
		u.Department, u.Organization, u.PreferredLanguage, u.GroupsJSON, u.SCIMAttributes,
		u.CriticalityScore, u.CriticalityReason)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

// GetUserByEmail finds a user by email
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	// Check for global search flag in context
	isGlobal := false
	if g, ok := ctx.Value(GlobalSearchKey).(bool); ok && g {
		isGlobal = true
	}

	var u User
	var lastLogin sql.NullString

	var query string
	var args []interface{}

	if isGlobal {
		query = `
			SELECT id, tenant_id, email, name, password_hash, auth_provider,
			       is_mfa_enabled, mfa_secret, role_id, created_at, updated_at, last_login_at,
			       external_id, active, display_name, user_type, title,
			       department, organization, preferred_language, groups_json, scim_attributes_json
			FROM users
			WHERE email = ?
		`
		args = []interface{}{email}
	} else {
		tenantID := TenantFromContext(ctx)
		query = `
			SELECT id, tenant_id, email, name, password_hash, auth_provider,
			       is_mfa_enabled, mfa_secret, role_id, created_at, updated_at, last_login_at,
			       external_id, active, display_name, user_type, title,
			       department, organization, preferred_language, groups_json, scim_attributes_json
			FROM users
			WHERE email = ? AND tenant_id = ?
		`
		args = []interface{}{email, tenantID}
	}

	err = conn.QueryRow(query, args...).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Name, &u.PasswordHash, &u.AuthProvider,
		&u.IsMFAEnabled, &u.MFASecret, &u.RoleID, &u.CreatedAt, &u.UpdatedAt, &lastLogin,
		&u.ExternalID, &u.Active, &u.DisplayName, &u.UserType, &u.Title,
		&u.Department, &u.Organization, &u.PreferredLanguage, &u.GroupsJSON, &u.SCIMAttributes,
		&u.CriticalityScore, &u.CriticalityReason,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query user by email: %w", err)
	}

	if lastLogin.Valid {
		u.LastLoginAt = lastLogin.String
	}

	return &u, nil
}

// GetUserByID finds a user by ID
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	var u User
	var lastLogin sql.NullString

	err = conn.QueryRow(`
		SELECT id, tenant_id, email, name, password_hash, auth_provider,
		       is_mfa_enabled, mfa_secret, role_id, created_at, updated_at, last_login_at,
		       external_id, active, display_name, user_type, title,
		       department, organization, preferred_language, groups_json, scim_attributes_json,
		       criticality_score, criticality_reason
		FROM users
		WHERE id = ? AND tenant_id = ?
	`, id, tenantID).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Name, &u.PasswordHash, &u.AuthProvider,
		&u.IsMFAEnabled, &u.MFASecret, &u.RoleID, &u.CreatedAt, &u.UpdatedAt, &lastLogin,
		&u.ExternalID, &u.Active, &u.DisplayName, &u.UserType, &u.Title,
		&u.Department, &u.Organization, &u.PreferredLanguage, &u.GroupsJSON, &u.SCIMAttributes,
		&u.CriticalityScore, &u.CriticalityReason,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query user by ID: %w", err)
	}

	if lastLogin.Valid {
		u.LastLoginAt = lastLogin.String
	}

	return &u, nil
}

// GetUserByExternalID finds a user by their external identifier (IdP ID)
func (r *UserRepository) GetUserByExternalID(ctx context.Context, externalID string) (*User, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	var u User
	var lastLogin sql.NullString

	err = conn.QueryRow(`
		SELECT id, tenant_id, email, name, password_hash, auth_provider,
		       is_mfa_enabled, mfa_secret, role_id, created_at, updated_at, last_login_at,
		       external_id, active, display_name, user_type, title,
		       department, organization, preferred_language, groups_json, scim_attributes_json,
		       criticality_score, criticality_reason
		FROM users
		WHERE external_id = ? AND tenant_id = ?
	`, externalID, tenantID).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Name, &u.PasswordHash, &u.AuthProvider,
		&u.IsMFAEnabled, &u.MFASecret, &u.RoleID, &u.CreatedAt, &u.UpdatedAt, &lastLogin,
		&u.ExternalID, &u.Active, &u.DisplayName, &u.UserType, &u.Title,
		&u.Department, &u.Organization, &u.PreferredLanguage, &u.GroupsJSON, &u.SCIMAttributes,
		&u.CriticalityScore, &u.CriticalityReason,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query user by external ID: %w", err)
	}

	if lastLogin.Valid {
		u.LastLoginAt = lastLogin.String
	}

	return &u, nil
}

// UpdateUser updates a user's details
func (r *UserRepository) UpdateUser(ctx context.Context, u *User) error {
	r.db.Lock()
	defer r.db.Unlock()

	u.UpdatedAt = time.Now().Format(time.RFC3339)
	u.TenantID = TenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE users SET
			email = ?, name = ?, password_hash = ?, auth_provider = ?,
			is_mfa_enabled = ?, mfa_secret = ?, role_id = ?, updated_at = ?,
			external_id = ?, active = ?, display_name = ?, user_type = ?,
			title = ?, department = ?, organization = ?, preferred_language = ?,
			groups_json = ?, scim_attributes_json = ?,
			criticality_score = ?, criticality_reason = ?
		WHERE id = ? AND tenant_id = ?
	`, u.Email, u.Name, u.PasswordHash, u.AuthProvider,
		u.IsMFAEnabled, u.MFASecret, u.RoleID, u.UpdatedAt,
		u.ExternalID, u.Active, u.DisplayName, u.UserType, u.Title,
		u.Department, u.Organization, u.PreferredLanguage, u.GroupsJSON, u.SCIMAttributes,
		u.CriticalityScore, u.CriticalityReason,
		u.ID, u.TenantID)

	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

// RecordLogin updates the last login time
func (r *UserRepository) RecordLogin(ctx context.Context, id string) error {
	r.db.Lock()
	defer r.db.Unlock()

	tenantID := TenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE users SET last_login_at = ? WHERE id = ? AND tenant_id = ?
	`, time.Now().Format(time.RFC3339), id, tenantID)

	return err
}

// DeleteUser removes a user
func (r *UserRepository) DeleteUser(ctx context.Context, id string) error {
	r.db.Lock()
	defer r.db.Unlock()

	tenantID := TenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		DELETE FROM users WHERE id = ? AND tenant_id = ?
	`, id, tenantID)

	return err
}

// ListUsers returns all users in a tenant
func (r *UserRepository) ListUsers(ctx context.Context) ([]User, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, email, name, password_hash, auth_provider,
		       is_mfa_enabled, mfa_secret, role_id, created_at, updated_at, last_login_at,
		       external_id, active, display_name, user_type, title,
		       department, organization, preferred_language, groups_json, scim_attributes_json,
		       criticality_score, criticality_reason
		FROM users
		WHERE tenant_id = ?
		ORDER BY name ASC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var lastLogin sql.NullString
		if err := rows.Scan(
			&u.ID, &u.TenantID, &u.Email, &u.Name, &u.PasswordHash, &u.AuthProvider,
			&u.IsMFAEnabled, &u.MFASecret, &u.RoleID, &u.CreatedAt, &u.UpdatedAt, &lastLogin,
			&u.ExternalID, &u.Active, &u.DisplayName, &u.UserType, &u.Title,
			&u.Department, &u.Organization, &u.PreferredLanguage, &u.GroupsJSON, &u.SCIMAttributes,
			&u.CriticalityScore, &u.CriticalityReason,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if lastLogin.Valid {
			u.LastLoginAt = lastLogin.String
		}
		users = append(users, u)
	}

	return users, nil
}

// RoleRepository handles role configuration
type RoleRepository struct {
	db DatabaseStore
}

func NewRoleRepository(db DatabaseStore) *RoleRepository {
	return &RoleRepository{db: db}
}

// CreateRole inserts a new role
func (r *RoleRepository) CreateRole(ctx context.Context, role *Role) error {
	r.db.Lock()
	defer r.db.Unlock()

	now := time.Now().Format(time.RFC3339)
	role.CreatedAt = now
	role.UpdatedAt = now
	role.TenantID = TenantFromContext(ctx)

	if role.ID == "" {
		role.ID = uuid.New().String()
	}

	permsBytes, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}

	_, err = r.db.ReplicatedExecContext(ctx, `
		INSERT INTO roles (
			id, tenant_id, name, description, permissions, is_system, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, role.ID, role.TenantID, role.Name, role.Description, string(permsBytes), role.IsSystem, role.CreatedAt, role.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create role: %w", err)
	}

	return nil
}

// GetRoleByID finds a role by ID
func (r *RoleRepository) GetRoleByID(ctx context.Context, id string) (*Role, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	var role Role
	var permsJSON string

	err = conn.QueryRow(`
		SELECT id, tenant_id, name, description, permissions, is_system, created_at, updated_at
		FROM roles
		WHERE id = ? AND tenant_id = ?
	`, id, tenantID).Scan(
		&role.ID, &role.TenantID, &role.Name, &role.Description, &permsJSON, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query role: %w", err)
	}

	if err := json.Unmarshal([]byte(permsJSON), &role.Permissions); err != nil {
		return nil, fmt.Errorf("unmarshal permissions: %w", err)
	}

	return &role, nil
}

// ListRoles returns all roles in a tenant
func (r *RoleRepository) ListRoles(ctx context.Context) ([]Role, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, name, description, permissions, is_system, created_at, updated_at
		FROM roles
		WHERE tenant_id = ?
		ORDER BY name ASC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		var permsJSON string
		if err := rows.Scan(
			&role.ID, &role.TenantID, &role.Name, &role.Description, &permsJSON, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		if err := json.Unmarshal([]byte(permsJSON), &role.Permissions); err != nil {
			return nil, fmt.Errorf("unmarshal permissions: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// UpdateRole updates a role's permissions
func (r *RoleRepository) UpdateRole(ctx context.Context, role *Role) error {
	r.db.Lock()
	defer r.db.Unlock()

	// Ensure system roles are not mutated lightly
	// (Enforced at service layer. At repo layer, just update.)

	role.UpdatedAt = time.Now().Format(time.RFC3339)
	role.TenantID = TenantFromContext(ctx)

	permsBytes, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("marshal permissions: %w", err)
	}

	_, err = r.db.ReplicatedExecContext(ctx, `
		UPDATE roles SET
			name = ?, description = ?, permissions = ?, is_system = ?, updated_at = ?
		WHERE id = ? AND tenant_id = ?
	`, role.Name, role.Description, string(permsBytes), role.IsSystem, role.UpdatedAt, role.ID, role.TenantID)

	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}

	return nil
}
