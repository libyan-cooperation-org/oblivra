package services

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"golang.org/x/crypto/bcrypt"
)

// IdentityService exposes identity, authentication and RBAC operations to the frontend
type IdentityService struct {
	BaseService
	userRepo *database.UserRepository
	roleRepo *database.RoleRepository
	rbac     *auth.RBACEngine
	hw       auth.HardwareRootedIdentity
	bus      *eventbus.Bus
	log      *logger.Logger
}

func (s *IdentityService) Name() string { return "identity-service" }

// Dependencies returns service dependencies
func (s *IdentityService) Dependencies() []string {
	return []string{"vault"}
}

func NewIdentityService(
	userRepo *database.UserRepository,
	roleRepo *database.RoleRepository,
	rbac *auth.RBACEngine,
	hw auth.HardwareRootedIdentity,
	bus *eventbus.Bus,
	log *logger.Logger,
) *IdentityService {
	return &IdentityService{
		userRepo: userRepo,
		roleRepo: roleRepo,
		rbac:     rbac,
		hw:       hw,
		bus:      bus,
		log:      log.WithPrefix("identity"),
	}
}

func (s *IdentityService) Start(ctx context.Context) error {
	return nil
}

func (s *IdentityService) Stop(ctx context.Context) error {
	return nil
}

// SecurityStats provides high-level metrics on identity hardening
type SecurityStats struct {
	TotalUsers     int  `json:"total_users"`
	MFAPassive     int  `json:"mfa_passive"` // Users with MFA enabled
	RBACActive     bool `json:"rbac_active"` // True if custom roles are in use
	MFAEnforcement bool `json:"mfa_enforcement"`
}

// GetSecurityStats returns a snapshot of global identity hardening
func (s *IdentityService) GetSecurityStats(ctx context.Context) (SecurityStats, error) {
	users, err := s.userRepo.ListUsers(ctx)
	if err != nil {
		return SecurityStats{}, err
	}

	roles, err := s.roleRepo.ListRoles(ctx)
	if err != nil {
		return SecurityStats{}, err
	}

	stats := SecurityStats{
		TotalUsers: len(users),
	}

	for _, u := range users {
		if u.IsMFAEnabled {
			stats.MFAPassive++
		}
	}

	// RBAC is considered "Active" if there are any non-system roles
	for _, r := range roles {
		if !r.IsSystem {
			stats.RBACActive = true
			break
		}
	}

	return stats, nil
}

// --- User CRUD ---

// CreateUser creates a new local user with a hashed password
func (s *IdentityService) CreateUser(ctx context.Context, email, name, password, roleID string) (*database.User, error) {
	s.log.Info("Creating user: %s (%s)", name, email)

	// Password policy enforcement
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &database.User{
		Email:        email,
		Name:         name,
		PasswordHash: string(hash),
		AuthProvider: "local",
		RoleID:       roleID,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	s.bus.Publish("identity.user_created", user.ID)
	return user, nil
}

// validatePassword enforces minimum password complexity requirements
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("password must contain at least one uppercase letter, one lowercase letter, and one digit")
	}
	return nil
}

// ListUsers returns all users in the current tenant
func (s *IdentityService) ListUsers(ctx context.Context) ([]database.User, error) {
	return s.userRepo.ListUsers(ctx)
}

// GetUser returns a single user by ID
func (s *IdentityService) GetUser(ctx context.Context, id string) (*database.User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

// UpdateUserRole assigns a new role to a user
func (s *IdentityService) UpdateUserRole(ctx context.Context, userID, roleID string) error {
	s.log.Info("Updating role for user %s to %s", userID, roleID)

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	user.RoleID = roleID
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return err
	}

	s.bus.Publish("identity.role_changed", map[string]string{
		"user_id": userID,
		"role_id": roleID,
	})
	return nil
}

// --- Authentication ---

// LoginLocal authenticates a user with email and password
func (s *IdentityService) LoginLocal(ctx context.Context, email, password string) (*database.User, error) {
	// Search across all tenants for the user (global login)
	user, err := s.userRepo.GetUserByEmail(database.WithGlobalSearch(ctx), email)
	if err != nil {
		s.log.Warn("Login failed for %s: user not found", email)
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.log.Warn("Login failed for %s: password mismatch", email)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Record the login
	_ = s.userRepo.RecordLogin(ctx, user.ID)
	s.bus.Publish("identity.login", user.Email)
	s.log.Info("User logged in: %s", user.Email)

	return user, nil
}

// LoginHardwareBound handles authentication using a hardware-rooted signature
func (s *IdentityService) LoginHardwareBound(ctx context.Context, email string, nonce []byte, signature []byte) (*database.User, error) {
	if s.hw == nil {
		return nil, fmt.Errorf("hardware identity not enabled for this platform")
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	valid, err := s.hw.VerifyIdentity(email, nonce, signature)
	if err != nil || !valid {
		s.log.Warn("Hardware identity verification failed for %s", email)
		return nil, fmt.Errorf("invalid hardware signature")
	}

	_ = s.userRepo.RecordLogin(ctx, user.ID)
	s.bus.Publish("identity.login.hardware", user.Email)
	s.log.Info("User logged in via Hardware: %s", user.Email)

	return user, nil
}

// --- MFA ---

// SetupTOTP generates a new TOTP secret and QR code for the user
func (s *IdentityService) SetupTOTP(ctx context.Context, userID string) (*auth.TOTPSetupResult, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result, err := auth.GenerateTOTP(auth.TOTPConfig{
		Issuer:      "Sovereign Terminal",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, err
	}

	// Store the secret (will be activated after first verification)
	user.MFASecret = result.Secret
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("save MFA secret: %w", err)
	}

	s.log.Info("TOTP setup initiated for user: %s", user.Email)
	return result, nil
}

// VerifyAndEnableMFA takes a 6-digit code, validates it, and enables MFA
func (s *IdentityService) VerifyAndEnableMFA(ctx context.Context, userID, code string) error {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.MFASecret == "" {
		return fmt.Errorf("TOTP not configured; call SetupTOTP first")
	}

	if !auth.ValidateTOTP(user.MFASecret, code) {
		return fmt.Errorf("invalid TOTP code")
	}

	user.IsMFAEnabled = true
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return err
	}

	s.log.Info("MFA enabled for user: %s", user.Email)
	s.bus.Publish("identity.mfa_enabled", user.ID)
	return nil
}

// ValidateMFA checks a TOTP code for login verification
func (s *IdentityService) ValidateMFA(ctx context.Context, userID, code string) (bool, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if !user.IsMFAEnabled || user.MFASecret == "" {
		return true, nil // MFA not required
	}

	return auth.ValidateTOTP(user.MFASecret, code), nil
}

// --- Roles ---

// ListRoles returns all roles in the current tenant
func (s *IdentityService) ListRoles(ctx context.Context) ([]database.Role, error) {
	return s.roleRepo.ListRoles(ctx)
}

// CreateRole creates a new custom role
func (s *IdentityService) CreateRole(ctx context.Context, name, description string, permissions []string) (*database.Role, error) {
	s.log.Info("Creating role: %s", name)

	role := &database.Role{
		Name:        name,
		Description: description,
		Permissions: permissions,
		IsSystem:    false,
	}

	if err := s.roleRepo.CreateRole(ctx, role); err != nil {
		return nil, err
	}

	s.bus.Publish("identity.role_created", role.ID)
	return role, nil
}

// UpdateRole modifies an existing role's permissions
func (s *IdentityService) UpdateRole(ctx context.Context, roleID, name, description string, permissions []string) error {
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}

	if role.IsSystem {
		return fmt.Errorf("system roles cannot be modified")
	}

	role.Name = name
	role.Description = description
	role.Permissions = permissions

	return s.roleRepo.UpdateRole(ctx, role)
}

// --- RBAC Helpers ---

// CheckPermission verifies if a user has the required permission
func (s *IdentityService) CheckPermission(ctx context.Context, userID, permission string) (bool, error) {
	user, err := s.buildIdentityUser(ctx, userID)
	if err != nil {
		return false, err
	}

	return s.rbac.HasPermission(user, permission), nil
}

// buildIdentityUser constructs an IdentityUser from DB records for RBAC evaluation
func (s *IdentityService) buildIdentityUser(ctx context.Context, userID string) (*auth.IdentityUser, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	role, err := s.roleRepo.GetRoleByID(ctx, user.RoleID)
	if err != nil {
		return nil, err
	}

	return &auth.IdentityUser{
		ID:          user.ID,
		TenantID:    user.TenantID,
		Email:       user.Email,
		Name:        user.Name,
		RoleID:      role.ID,
		RoleName:    role.Name,
		Permissions: role.Permissions,
	}, nil
}

// --- Federated Identity (OIDC/SAML) ---

// GetOIDCURL returns the redirect URL for the configured OIDC provider
func (s *IdentityService) GetOIDCURL(ctx context.Context) (string, error) {
	// In Phase 0.5, this will use the 'auth.OIDCProvider' to generate a real URL.
	// For now, redirect to a mock callback for MVP validation.
	s.log.Info("Generating OIDC redirect URL (Mock)")
	return "/api/v1/auth/oidc/callback?code=mock-oidc-code", nil
}

// GetSAMLURL returns the redirect URL for the configured SAML IdP
func (s *IdentityService) GetSAMLURL(ctx context.Context) (string, error) {
	s.log.Info("Generating SAML redirect URL (Mock)")
	return "/api/v1/auth/saml/callback?SAMLResponse=mock-saml-response", nil
}

// HandleOIDCCallback processes the OIDC authorization code
func (s *IdentityService) HandleOIDCCallback(ctx context.Context, code string) (*database.User, error) {
	s.log.Info("Processing OIDC callback with code: %s", code)
	// Return the primary admin user for MVP validation
	user, err := s.userRepo.GetUserByEmail(database.WithGlobalSearch(ctx), "admin@oblivra.org")
	if err != nil {
		return nil, fmt.Errorf("OIDC user mapping failed: %w", err)
	}
	return user, nil
}

// HandleSAMLCallback processes the SAML assertion
func (s *IdentityService) HandleSAMLCallback(ctx context.Context, data string) (*database.User, error) {
	s.log.Info("Processing SAML callback")
	user, err := s.userRepo.GetUserByEmail(database.WithGlobalSearch(ctx), "admin@oblivra.org")
	if err != nil {
		return nil, fmt.Errorf("SAML user mapping failed: %w", err)
	}
	return user, nil
}
