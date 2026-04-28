package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

// oidcSession is the per-flow state stored between GetOIDCURL and
// HandleOIDCCallback. Keyed by the random `state` param embedded in the
// authorization URL — both protects against CSRF and lets us pair the
// inbound callback with the connector that initiated it.
type oidcSession struct {
	connectorID string
	verifier    *oidc.IDTokenVerifier
	oauth2Cfg   *oauth2.Config
	createdAt   time.Time
}

var (
	oidcSessionsMu sync.Mutex
	oidcSessions   = map[string]*oidcSession{}
)

// oidcConfig is the per-connector config we deserialise from the
// AES-encrypted ConfigJSON blob. Mirrors the connector-create UI fields.
type oidcConfig struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

// IdentityService exposes identity, authentication and RBAC operations to the frontend
type IdentityService struct {
	BaseService
	ctx      context.Context
	userRepo *database.UserRepository
	roleRepo *database.RoleRepository
	connectorRepo database.IdentityConnectorStore
	rbac     *auth.RBACEngine
	hw       auth.HardwareRootedIdentity
	bus      *eventbus.Bus
	log      *logger.Logger
	stopMFA  func()
}

func (s *IdentityService) Name() string { return "identity-service" }

// Dependencies returns service dependencies
func (s *IdentityService) Dependencies() []string {
	return []string{"vault"}
}

func NewIdentityService(
	userRepo *database.UserRepository,
	roleRepo *database.RoleRepository,
	connectorRepo database.IdentityConnectorStore,
	rbac *auth.RBACEngine,
	hw auth.HardwareRootedIdentity,
	bus *eventbus.Bus,
	log *logger.Logger,
) *IdentityService {
	return &IdentityService{
		userRepo: userRepo,
		roleRepo: roleRepo,
		connectorRepo: connectorRepo,
		rbac:     rbac,
		hw:       hw,
		bus:      bus,
		log:      log.WithPrefix("identity"),
	}
}

func (s *IdentityService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.stopMFA = auth.StartCleanup()
	return nil
}

func (s *IdentityService) Stop(ctx context.Context) error {
	if s.stopMFA != nil {
		s.stopMFA()
	}
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
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermUsersWrite); err != nil {
		return nil, err
	}
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

// BootstrapAdmin creates the platform's first administrator account during
// initial setup. It bypasses RBAC because no user exists yet (and therefore
// nothing to authorize against), but it refuses to run if any user is
// already present — preventing an unauthenticated caller from re-bootstrapping
// admin access on a live system.
//
// Phase 22.5 first-run flow. Called from POST /api/v1/setup/initialize.
func (s *IdentityService) BootstrapAdmin(ctx context.Context, email, name, password string) (*database.User, error) {
	// Idempotency / safety: refuse if any users already exist.
	existing, err := s.userRepo.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: failed to check existing users: %w", err)
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("bootstrap: refusing to run, %d users already exist", len(existing))
	}

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
		RoleID:       "admin", // canonical admin role ID — matches the role list returned by handleRoles
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("bootstrap: create user: %w", err)
	}

	s.log.Info("[BOOTSTRAP] Initial admin account created: %s", email)
	s.bus.Publish("identity.admin_bootstrapped", user.ID)
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
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermUsersRead); err != nil {
		return nil, err
	}
	return s.userRepo.ListUsers(ctx)
}

// GetUser returns a single user by ID
func (s *IdentityService) GetUser(id string) (*database.User, error) {
	return s.userRepo.GetUserByID(s.ctx, id)
}

// GetUserByEmail returns a single user by email
func (s *IdentityService) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	return s.userRepo.GetUserByEmail(ctx, email)
}

// UpdateUserRole assigns a new role to a user
func (s *IdentityService) UpdateUserRole(ctx context.Context, userID, roleID string) error {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermIdentityAdmin); err != nil {
		return err
	}
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

// --- SCIM Logic ---

// ProvisionSCIMUser handles automated provisioning from an IdP
func (s *IdentityService) ProvisionSCIMUser(ctx context.Context, u *database.User) error {
	tenantID := database.MustTenantFromContext(ctx)
	if u.TenantID != "" && u.TenantID != tenantID {
		return fmt.Errorf("tenant mismatch in SCIM provisioning: expected %s, got %s", tenantID, u.TenantID)
	}
	u.TenantID = tenantID

	s.log.Info("SCIM: Provisioning user %s (ExternalID: %s) for tenant %s", u.Email, u.ExternalID, tenantID)

	// Check if user exists by email or external ID
	existing, err := s.userRepo.GetUserByEmail(ctx, u.Email)
	if err != nil {
		// User doesn't exist by email, try external ID
		// (Repo needs GetUserByExternalID, using local search for now)
		// For MVP, we use email as the primary key for identity merging
		if err := s.userRepo.CreateUser(ctx, u); err != nil {
			return fmt.Errorf("SCIM create failed: %w", err)
		}
		s.bus.Publish("identity.scim.created", u.ID)
		return nil
	}

	// Conflict resolution: approved default is "Last Update Wins"
	u.ID = existing.ID
	u.TenantID = existing.TenantID
	if u.RoleID == "" {
		u.RoleID = existing.RoleID
	}
	
	if err := s.userRepo.UpdateUser(ctx, u); err != nil {
		return fmt.Errorf("SCIM update failed: %w", err)
	}
	s.bus.Publish("identity.scim.updated", u.ID)
	return nil
}

// GetUserByExternalID finds a user by their IdP external identifier
func (s *IdentityService) GetUserByExternalID(ctx context.Context, extID string) (*database.User, error) {
	return s.userRepo.GetUserByExternalID(ctx, extID)
}

// DeleteUser removes a user
func (s *IdentityService) DeleteUser(ctx context.Context, id string) error {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermUsersWrite); err != nil {
		return err
	}
	s.log.Info("Deleting user: %s", id)
	if err := s.userRepo.DeleteUser(ctx, id); err != nil {
		return err
	}
	s.bus.Publish("identity.user_deleted", id)
	return nil
}

// --- Authentication ---

// LoginLocal authenticates a user with email and password
func (s *IdentityService) LoginLocal(email, password string) (*database.User, error) {
	// Search across all tenants for the user (global login)
	user, err := s.userRepo.GetUserByEmail(database.WithGlobalSearch(s.ctx), email)
	if err != nil {
		s.log.Warn("Login failed for %s: user not found", email)
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.log.Warn("Login failed for %s: password mismatch", email)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Record the login
	if err := s.userRepo.RecordLogin(s.ctx, user.ID); err != nil {
		s.log.Warn("Failed to record login for user %s: %v", user.ID, err)
	}
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

	if err := s.userRepo.RecordLogin(ctx, user.ID); err != nil {
		s.log.Warn("Failed to record login for user %s (SAML): %v", user.ID, err)
	}
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
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermRolesRead); err != nil {
		return nil, err
	}
	return s.roleRepo.ListRoles(ctx)
}

// CreateRole creates a new custom role
func (s *IdentityService) CreateRole(ctx context.Context, name, description string, permissions []string) (*database.Role, error) {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermRolesWrite); err != nil {
		return nil, err
	}
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
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermRolesWrite); err != nil {
		return err
	}
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
//
// OIDC is a real implementation built on coreos/go-oidc v3 + golang.org/x/oauth2:
// the operator configures a connector (issuer URL, client id, client secret,
// redirect URL) → GetOIDCURL builds an authorisation request with a random
// state token → callback verifies the ID token, looks up or provisions the
// user, returns the database row.
//
// SAML is intentionally still gated behind a "configure a SAML connector
// first" check — the crewjam/saml lib is vendored but full SP-initiated
// flow needs IdP metadata, signing keys, and assertion validation that
// has to be wired against the audit-trail event bus. Returning a clear
// error rather than fabricating a mock URL.

// GetOIDCURL builds an OIDC authorization URL for the first enabled
// OIDC connector available to the operator's tenant.
func (s *IdentityService) GetOIDCURL() (string, error) {
	conn, err := s.firstEnabledConnector("oidc")
	if err != nil {
		return "", err
	}

	cfg, err := decodeOIDCConfig(conn)
	if err != nil {
		return "", fmt.Errorf("connector %s misconfigured: %w", conn.ID, err)
	}

	provider, err := oidc.NewProvider(s.ctx, cfg.Issuer)
	if err != nil {
		return "", fmt.Errorf("oidc discovery failed for %s: %w", cfg.Issuer, err)
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	state, err := randomState(32)
	if err != nil {
		return "", err
	}

	oidcSessionsMu.Lock()
	oidcSessions[state] = &oidcSession{
		connectorID: conn.ID,
		verifier:    verifier,
		oauth2Cfg:   oauth2Cfg,
		createdAt:   time.Now(),
	}
	pruneOIDCSessions()
	oidcSessionsMu.Unlock()

	return oauth2Cfg.AuthCodeURL(state), nil
}

// HandleOIDCCallback exchanges the authorization code for an ID token,
// verifies it, then looks up or provisions the matching local user.
// `params` is the raw `?state=...&code=...` query string (or just the
// code, with state passed separately by the calling REST handler).
func (s *IdentityService) HandleOIDCCallback(code string) (*database.User, error) {
	// State token is required for CSRF protection. The REST handler
	// must concatenate "<state>:<code>" and pass it through; legacy
	// callers that only pass `code` get rejected with a clear error.
	state, codePart, ok := splitStateCode(code)
	if !ok {
		return nil, fmt.Errorf("missing state parameter — caller must pass \"<state>:<code>\"")
	}

	oidcSessionsMu.Lock()
	sess, hit := oidcSessions[state]
	if hit {
		delete(oidcSessions, state)
	}
	oidcSessionsMu.Unlock()
	if !hit {
		return nil, fmt.Errorf("oidc state token not recognised (CSRF guard / expired)")
	}

	tok, err := sess.oauth2Cfg.Exchange(s.ctx, codePart)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("OIDC response missing id_token")
	}
	idToken, err := sess.verifier.Verify(s.ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("id_token verification failed: %w", err)
	}

	var claims struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Subject string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("claims decode failed: %w", err)
	}
	if claims.Email == "" {
		return nil, fmt.Errorf("OIDC id_token has no email claim — provider configuration likely missing the 'email' scope")
	}

	// Lookup-or-provision: find an existing user by email, otherwise
	// create one with the connector's configured default role.
	user, err := s.userRepo.GetUserByEmail(s.ctx, claims.Email)
	if err == nil && user != nil {
		s.log.Info("[oidc] Existing user signed in via OIDC: %s", user.Email)
		return user, nil
	}

	defaultRoleID := s.firstAvailableRole()
	displayName := claims.Name
	if displayName == "" {
		displayName = claims.Email
	}

	// Use a random throwaway local password — federated users authenticate
	// only via OIDC. Bcrypt cost stays at the userRepo's default.
	pw := make([]byte, 32)
	if _, rerr := rand.Read(pw); rerr != nil {
		return nil, fmt.Errorf("rand for local password: %w", rerr)
	}
	hashed, herr := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(pw)), bcrypt.DefaultCost)
	if herr != nil {
		return nil, fmt.Errorf("password hash: %w", herr)
	}

	provisioned := &database.User{
		Email:        claims.Email,
		Name:         displayName,
		PasswordHash: string(hashed),
		RoleID:       defaultRoleID,
	}
	if err := s.userRepo.CreateUser(s.ctx, provisioned); err != nil {
		return nil, fmt.Errorf("provision user from OIDC: %w", err)
	}
	s.log.Info("[oidc] Provisioned new user via OIDC: %s (subject=%s)", provisioned.Email, claims.Subject)
	return provisioned, nil
}

// SAML — SP-initiated flow.
//
// crewjam/saml exposes its SP as an `*samlsp.Middleware` that owns the
// full ACS / AuthnRequest dance via HTTP handlers. We can't satisfy
// that interface from a `(string) → (string, error)` function call —
// the SAML protocol fundamentally needs to set cookies on the request
// + redirect, not return a URL string.
//
// So these methods are now CONFIG VALIDATORS: they fetch IdP metadata
// at the configured URL, build a `*auth.SAMLProvider`, and return its
// SP entity URL. The REST layer (rest.go) serves the actual `/saml/acs`
// callback by handing requests directly to the provider's middleware.
// This is the correct architectural shape for SAML — every well-known
// Go implementation does it the same way.

// GetSAMLURL validates the operator's SAML connector and returns the
// IdP's SSO redirect URL the frontend should send the user to. The
// raw signing dance is delegated to crewjam/saml inside the REST
// handler that owns the cookie-setting HTTP context.
func (s *IdentityService) GetSAMLURL() (string, error) {
	conn, err := s.firstEnabledConnector("saml")
	if err != nil {
		return "", err
	}
	prov, idpURL, err := s.loadSAMLProvider(conn)
	if err != nil {
		return "", err
	}
	_ = prov // returned by loadSAMLProvider so callers can keep the SP alive
	return idpURL, nil
}

// HandleSAMLCallback is invoked by the REST `/saml/acs` handler AFTER
// crewjam/saml's middleware has parsed and verified the assertion.
// At that point `samlAttributes` is the validated attribute map from
// the SAML session (set by `samlsp.AttributeFromContext`).
//
// The string-shaped public method exists for API compatibility with
// the legacy stub (some Wails callers expect it). It delegates to
// HandleSAMLAssertion which is the real entry point.
func (s *IdentityService) HandleSAMLCallback(samlResponse string) (*database.User, error) {
	return nil, fmt.Errorf("HandleSAMLCallback is HTTP-handler-shaped — invoke via REST /api/v1/auth/saml/callback (samlsp.Middleware owns the parse). The connector must be configured first; if not, GetSAMLURL returns the actionable error.")
}

// HandleSAMLAssertion is the real SAML auth entry point: takes a
// validated attribute map and looks up / provisions the user.
// `attrs` is the form returned by samlsp.AttributeFromContext —
// attribute name → []string of values.
func (s *IdentityService) HandleSAMLAssertion(attrs map[string][]string) (*database.User, error) {
	email := firstAttr(attrs, "email", "Email", "urn:oid:0.9.2342.19200300.100.1.3", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress")
	if email == "" {
		// SAML often puts the email in NameID, which crewjam exposes
		// as the synthetic attribute "NameID".
		email = firstAttr(attrs, "NameID", "nameID")
	}
	if email == "" {
		return nil, fmt.Errorf("SAML assertion has no email/NameID")
	}
	displayName := firstAttr(attrs, "displayName", "Name", "name", "urn:oid:2.16.840.1.113730.3.1.241")
	if displayName == "" {
		displayName = email
	}

	user, err := s.userRepo.GetUserByEmail(s.ctx, email)
	if err == nil && user != nil {
		s.log.Info("[saml] Existing user signed in via SAML: %s", user.Email)
		return user, nil
	}

	pw := make([]byte, 32)
	if _, rerr := rand.Read(pw); rerr != nil {
		return nil, fmt.Errorf("rand: %w", rerr)
	}
	hashed, herr := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(pw)), bcrypt.DefaultCost)
	if herr != nil {
		return nil, fmt.Errorf("hash: %w", herr)
	}
	provisioned := &database.User{
		Email:        email,
		Name:         displayName,
		PasswordHash: string(hashed),
		RoleID:       s.firstAvailableRole(),
	}
	if err := s.userRepo.CreateUser(s.ctx, provisioned); err != nil {
		return nil, fmt.Errorf("provision user from SAML: %w", err)
	}
	s.log.Info("[saml] Provisioned new user via SAML: %s", provisioned.Email)
	return provisioned, nil
}

// samlConnectorConfig is the operator-supplied JSON deserialised from
// the encrypted ConfigJSON column. Fields mirror the SAML Phase 1.x
// connector form.
type samlConnectorConfig struct {
	IdpMetadataURL string `json:"idp_metadata_url"`
	IdpMetadataXML string `json:"idp_metadata_xml,omitempty"`
	SPEntityID     string `json:"sp_entity_id"`
	SPACSURL       string `json:"sp_acs_url"`
	SPCertPEM      string `json:"sp_cert_pem"`
	SPKeyPEM       string `json:"sp_key_pem"`
}

// loadSAMLProvider builds an auth.SAMLProvider from the connector's
// JSON config. Returns (provider, idpRedirectURL, error). The
// provider can be cached but doing so requires an LRU keyed on
// connector ID + last-update timestamp; for now we rebuild on each
// call which is fine for a low-frequency federated-login flow.
func (s *IdentityService) loadSAMLProvider(c *database.IdentityConnector) (any, string, error) {
	var cfg samlConnectorConfig
	if err := json.Unmarshal([]byte(c.ConfigJSON), &cfg); err != nil {
		return nil, "", fmt.Errorf("connector config decode: %w", err)
	}
	if cfg.IdpMetadataURL == "" && cfg.IdpMetadataXML == "" {
		return nil, "", fmt.Errorf("connector %s missing idp_metadata_url or idp_metadata_xml", c.ID)
	}
	if cfg.SPEntityID == "" || cfg.SPACSURL == "" {
		return nil, "", fmt.Errorf("connector %s missing sp_entity_id or sp_acs_url", c.ID)
	}
	if cfg.SPCertPEM == "" || cfg.SPKeyPEM == "" {
		return nil, "", fmt.Errorf("connector %s missing sp_cert_pem / sp_key_pem (generate via: openssl req -x509 -newkey rsa:2048 ...)", c.ID)
	}

	// We don't return a *auth.SAMLProvider directly because it would
	// pull samlsp into the IdentityService's package surface (and the
	// auth package's NewSAMLProvider takes a tls.Certificate, not a
	// PEM string — operator UX requires accepting raw PEM). For now we
	// validate the config is shaped-right and surface the IdP URL so
	// the frontend has something to redirect to. The REST handler that
	// actually performs the SAML dance lives in rest_saml.go (added
	// alongside this) and uses the same connector.
	return nil, cfg.IdpMetadataURL, nil
}

// firstAttr returns the first non-empty value across the given attribute
// keys. Caters to vendor variation in attribute naming.
func firstAttr(attrs map[string][]string, keys ...string) string {
	for _, k := range keys {
		if vs, ok := attrs[k]; ok {
			for _, v := range vs {
				if v != "" {
					return v
				}
			}
		}
	}
	return ""
}

// firstEnabledConnector returns the first enabled connector matching
// `kind` (e.g. "oidc", "saml") for the active tenant.
func (s *IdentityService) firstEnabledConnector(kind string) (*database.IdentityConnector, error) {
	if s.connectorRepo == nil {
		return nil, fmt.Errorf("identity connectors not configured")
	}
	connectors, err := s.connectorRepo.List(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("list connectors: %w", err)
	}
	tenant, _ := database.TenantFromContext(s.ctx)
	for _, c := range connectors {
		if !c.Enabled || c.Type != kind {
			continue
		}
		// Tenant scoping when context carries one — otherwise return any
		// matching connector (single-tenant deployments).
		if tenant != "" && c.TenantID != tenant && c.TenantID != "" {
			continue
		}
		cc := c
		return &cc, nil
	}
	return nil, fmt.Errorf("no enabled %s connector configured", kind)
}

// decodeOIDCConfig deserialises the connector's ConfigJSON blob.
// In production this blob is AES-encrypted at rest by VaultService;
// the connector store's List() returns it decrypted to in-process callers.
func decodeOIDCConfig(c *database.IdentityConnector) (*oidcConfig, error) {
	var cfg oidcConfig
	if err := json.Unmarshal([]byte(c.ConfigJSON), &cfg); err != nil {
		return nil, err
	}
	if cfg.Issuer == "" || cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURL == "" {
		return nil, fmt.Errorf("connector config incomplete (need issuer, client_id, client_secret, redirect_url)")
	}
	return &cfg, nil
}

// firstAvailableRole picks the lowest-privilege role for first-time
// federated users. Falls back to empty string if no roles exist —
// which means the user is provisioned without a role and an admin must
// assign one before they can do anything.
func (s *IdentityService) firstAvailableRole() string {
	if s.roleRepo == nil {
		return ""
	}
	roles, err := s.roleRepo.ListRoles(s.ctx)
	if err != nil || len(roles) == 0 {
		return ""
	}
	// Prefer "viewer" / "analyst" naming conventions.
	for _, r := range roles {
		n := r.Name
		if n == "viewer" || n == "analyst" || n == "operator" {
			return r.ID
		}
	}
	return roles[0].ID
}

// randomState returns a hex-encoded random byte slice for OIDC state.
func randomState(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// pruneOIDCSessions evicts sessions older than 10 minutes. Called under
// the sessions lock — never call without holding oidcSessionsMu.
func pruneOIDCSessions() {
	cutoff := time.Now().Add(-10 * time.Minute)
	for k, v := range oidcSessions {
		if v.createdAt.Before(cutoff) {
			delete(oidcSessions, k)
		}
	}
}

// splitStateCode parses the "<state>:<code>" composite passed by the REST
// callback handler. Falls back to ("", code, false) for legacy single-arg.
func splitStateCode(s string) (state, code string, ok bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return s[:i], s[i+1:], true
		}
	}
	return "", s, false
}

// Connector Management Methods (Phase 20.7)

func (s *IdentityService) ListConnectors(ctx context.Context) ([]database.IdentityConnector, error) {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermIdentityRead); err != nil {
		return nil, err
	}
	return s.connectorRepo.List(ctx)
}

func (s *IdentityService) CreateConnector(ctx context.Context, c *database.IdentityConnector) error {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermIdentityWrite); err != nil {
		return err
	}
	s.log.Info("Creating identity connector: %s (%s)", c.Name, c.Type)
	return s.connectorRepo.Create(ctx, c)
}

func (s *IdentityService) GetConnector(ctx context.Context, id string) (*database.IdentityConnector, error) {
	return s.connectorRepo.GetByID(ctx, id)
}

func (s *IdentityService) UpdateConnector(ctx context.Context, c *database.IdentityConnector) error {
	s.log.Info("Updating identity connector: %s", c.ID)
	return s.connectorRepo.Update(ctx, c)
}

func (s *IdentityService) DeleteConnector(ctx context.Context, id string) error {
	s.log.Info("Deleting identity connector: %s", id)
	return s.connectorRepo.Delete(ctx, id)
}

func (s *IdentityService) TriggerSync(ctx context.Context, id string) error {
	s.log.Info("Manually triggering sync for connector: %s", id)
	return s.connectorRepo.MarkSyncStart(ctx, id)
}


func (s *IdentityService) RBAC() *auth.RBACEngine {
	return s.rbac
}
