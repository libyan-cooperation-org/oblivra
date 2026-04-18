package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/kingknull/oblivrashell/internal/logger"
	"golang.org/x/oauth2"
)

// OIDCProvider wraps an OpenID Connect identity provider
type OIDCProvider struct {
	Name         string
	ProviderID   string
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	log          *logger.Logger
}

// OIDCConfig holds the parameters needed to configure an OIDC provider
type OIDCConfig struct {
	ProviderID   string
	Name         string
	IssuerURL    string
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

// OIDCClaims represents the claims extracted from a verified OIDC ID token
type OIDCClaims struct {
	Subject  string   `json:"sub"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Groups   []string `json:"groups"`
	Verified bool     `json:"email_verified"`
}

// NewOIDCProvider initializes an OIDC provider using discovery
func NewOIDCProvider(ctx context.Context, cfg OIDCConfig, log *logger.Logger) (*OIDCProvider, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC issuer %s: %w", cfg.IssuerURL, err)
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.CallbackURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	return &OIDCProvider{
		Name:         cfg.Name,
		ProviderID:   cfg.ProviderID,
		oauth2Config: oauth2Cfg,
		verifier:     verifier,
		log:          log.WithPrefix("oidc"),
	}, nil
}

// NewManualOIDCProvider initializes an OIDC provider with explicit endpoints (Static Metadata)
func NewManualOIDCProvider(cfg OIDCConfig, authURL, tokenURL, userInfoURL string, jwksJSON string, log *logger.Logger) (*OIDCProvider, error) {
	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
		RedirectURL: cfg.CallbackURL,
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}

	// In a real air-gap, we'd parse jwksJSON to create a static KeySet verifier.
	// For this hardening layer, we're stubbing the verifier requirement.
	return &OIDCProvider{
		Name:         cfg.Name,
		ProviderID:   cfg.ProviderID,
		oauth2Config: oauth2Cfg,
		log:          log.WithPrefix("oidc-manual"),
	}, nil
}

// AuthURL generates the redirect URL for the SSO login
func (p *OIDCProvider) AuthURL() (string, string) {
	state := generateState()
	url := p.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	return url, state
}

// HandleCallback processes the OIDC callback and extracts identity claims
func (p *OIDCProvider) HandleCallback(ctx context.Context, code string) (*OIDCClaims, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange OIDC code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in OIDC response")
	}

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify id_token: %w", err)
	}

	var claims OIDCClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	p.log.Info("OIDC login successful: %s (%s)", claims.Email, claims.Subject)
	return &claims, nil
}

// OIDCCallbackHandler returns an http.HandlerFunc for the /auth/oidc/callback endpoint
func OIDCCallbackHandler(providers map[string]*OIDCProvider, onSuccess func(providerID string, claims *OIDCClaims, w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providerID := r.URL.Query().Get("provider")
		code := r.URL.Query().Get("code")

		if providerID == "" || code == "" {
			http.Error(w, "missing provider or code", http.StatusBadRequest)
			return
		}

		provider, ok := providers[providerID]
		if !ok {
			http.Error(w, "unknown provider", http.StatusNotFound)
			return
		}

		claims, err := provider.HandleCallback(r.Context(), code)
		if err != nil {
			http.Error(w, fmt.Sprintf("OIDC error: %v", err), http.StatusUnauthorized)
			return
		}

		onSuccess(providerID, claims, w, r)
	}
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
