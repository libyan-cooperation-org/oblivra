package httpserver

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OIDCConfig holds the platform-side OIDC configuration. The handler is
// scaffolding — it implements the Authorization Code flow honestly (PKCE +
// state nonce + token exchange) but expects the operator to map the resulting
// `sub` / email to an OBLIVRA role through `OIDCConfig.RoleMapper`.
//
// We deliberately keep this minimal-deps: stdlib only, no external OIDC
// library. For more elaborate flows (refresh tokens, token revocation,
// dynamic client registration) bring in coreos/go-oidc behind a build tag.
type OIDCConfig struct {
	Issuer       string // e.g. https://login.example.com
	ClientID     string
	ClientSecret string
	RedirectURL  string // e.g. https://oblivra.internal/api/v1/auth/oidc/callback
	Scopes       []string

	// RoleMapper turns claims into an OBLIVRA role (admin / analyst / readonly / agent).
	// Returning an empty role rejects the login.
	RoleMapper func(claims map[string]any) string
}

type oidcEndpoints struct {
	AuthURL       string
	TokenURL      string
	UserInfoURL   string
	IssuerHost    string
}

// OIDCHandler is created lazily on first request — discovery is done once.
type OIDCHandler struct {
	cfg    OIDCConfig
	once   sync.Once
	disco  oidcEndpoints
	client *http.Client

	mu     sync.Mutex
	pkceByState map[string]string // state → code_verifier
}

func NewOIDC(cfg OIDCConfig) *OIDCHandler {
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "profile", "email"}
	}
	return &OIDCHandler{
		cfg:         cfg,
		client:      &http.Client{Timeout: 10 * time.Second},
		pkceByState: map[string]string{},
	}
}

// Configured reports whether OIDC is wired up with non-empty fields.
func (h *OIDCHandler) Configured() bool {
	return h.cfg.Issuer != "" && h.cfg.ClientID != "" && h.cfg.RedirectURL != ""
}

func (h *OIDCHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.Configured() {
		writeError(w, http.StatusServiceUnavailable, "OIDC not configured")
		return
	}
	if err := h.discover(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	state := randomB64(24)
	verifier := randomB64(32)
	challenge := s256Challenge(verifier)

	h.mu.Lock()
	h.pkceByState[state] = verifier
	h.mu.Unlock()

	u, _ := url.Parse(h.disco.AuthURL)
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", h.cfg.ClientID)
	q.Set("redirect_uri", h.cfg.RedirectURL)
	q.Set("scope", strings.Join(h.cfg.Scopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if err := h.discover(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeError(w, http.StatusBadRequest, "missing code or state")
		return
	}
	h.mu.Lock()
	verifier, ok := h.pkceByState[state]
	if ok {
		delete(h.pkceByState, state)
	}
	h.mu.Unlock()
	if !ok {
		writeError(w, http.StatusUnauthorized, "unknown state — possible CSRF")
		return
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", h.cfg.RedirectURL)
	form.Set("client_id", h.cfg.ClientID)
	if h.cfg.ClientSecret != "" {
		form.Set("client_secret", h.cfg.ClientSecret)
	}
	form.Set("code_verifier", verifier)
	req, _ := http.NewRequestWithContext(r.Context(), "POST", h.disco.TokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := h.client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	tokenBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		writeError(w, http.StatusBadGateway, "token endpoint: "+resp.Status+": "+string(tokenBody))
		return
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(tokenBody, &tok); err != nil {
		writeError(w, http.StatusBadGateway, "bad token response: "+err.Error())
		return
	}

	// Fetch user-info with the access token; the IdP returns claims that the
	// RoleMapper turns into an OBLIVRA role.
	uiReq, _ := http.NewRequestWithContext(r.Context(), "GET", h.disco.UserInfoURL, nil)
	uiReq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	uiResp, err := h.client.Do(uiReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer uiResp.Body.Close()
	uiBody, _ := io.ReadAll(uiResp.Body)
	if uiResp.StatusCode/100 != 2 {
		writeError(w, http.StatusBadGateway, "userinfo: "+uiResp.Status+": "+string(uiBody))
		return
	}
	var claims map[string]any
	if err := json.Unmarshal(uiBody, &claims); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	role := ""
	if h.cfg.RoleMapper != nil {
		role = h.cfg.RoleMapper(claims)
	}
	if role == "" {
		writeError(w, http.StatusForbidden, "no role mapping for this principal")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sub":      claims["sub"],
		"email":    claims["email"],
		"name":     claims["name"],
		"role":     role,
		"idToken":  tok.IDToken,
		"hint":     "exchange this id_token for an OBLIVRA bearer key via your operator workflow",
	})
}

func (h *OIDCHandler) discover() error {
	var derr error
	h.once.Do(func() {
		issuer := strings.TrimRight(h.cfg.Issuer, "/")
		req, _ := http.NewRequest("GET", issuer+"/.well-known/openid-configuration", nil)
		resp, err := h.client.Do(req)
		if err != nil {
			derr = err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			derr = errors.New("oidc discovery: " + resp.Status)
			return
		}
		var doc struct {
			AuthorizationEndpoint string `json:"authorization_endpoint"`
			TokenEndpoint         string `json:"token_endpoint"`
			UserInfoEndpoint      string `json:"userinfo_endpoint"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
			derr = err
			return
		}
		h.disco.AuthURL = doc.AuthorizationEndpoint
		h.disco.TokenURL = doc.TokenEndpoint
		h.disco.UserInfoURL = doc.UserInfoEndpoint
	})
	return derr
}

func randomB64(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func s256Challenge(verifier string) string {
	// Standard PKCE S256: base64url(sha256(verifier))
	return base64SHA256URL([]byte(verifier))
}

// base64SHA256URL is split out so the dependency stays at the top of the file.
func base64SHA256URL(b []byte) string {
	sum := sha256Sum(b)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func sha256Sum(b []byte) [32]byte {
	// inlined to avoid an extra import line at the top of the file
	return sha256SumInternal(b)
}
