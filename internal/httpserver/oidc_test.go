package httpserver

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
)

// TestOIDCFlow stands up a fake IdP that:
//   - serves /.well-known/openid-configuration
//   - returns a code on the auth endpoint
//   - exchanges the code for a token, validating the PKCE code_verifier
//   - serves user-info
// Then drives the OBLIVRA OIDCHandler through Login + Callback and asserts:
//   - state nonce is honored (CSRF protection)
//   - PKCE S256 challenge → verifier round-trips
//   - Role mapping turns claims into a role
//   - Re-using the state (replay) fails
func TestOIDCFlow(t *testing.T) {
	type tokenReq struct {
		grantType    string
		code         string
		codeVerifier string
		redirect     string
	}
	var capturedAuthState string
	var capturedAuthChallenge string
	var capturedTokenReq tokenReq

	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			base := "http://" + r.Host
			doc := map[string]string{
				"authorization_endpoint": base + "/auth",
				"token_endpoint":         base + "/token",
				"userinfo_endpoint":      base + "/userinfo",
			}
			_ = json.NewEncoder(w).Encode(doc)
		case "/auth":
			// Real IdPs would redirect the browser back to redirect_uri.
			// Here we capture the params and return them so the test can
			// drive the callback directly.
			q := r.URL.Query()
			capturedAuthState = q.Get("state")
			capturedAuthChallenge = q.Get("code_challenge")
			if q.Get("code_challenge_method") != "S256" {
				w.WriteHeader(400)
				return
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte("auth-displayed"))
		case "/token":
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			capturedTokenReq = tokenReq{
				grantType:    form.Get("grant_type"),
				code:         form.Get("code"),
				codeVerifier: form.Get("code_verifier"),
				redirect:     form.Get("redirect_uri"),
			}
			// Validate the PKCE verifier against the stored challenge.
			h := sha256.Sum256([]byte(capturedTokenReq.codeVerifier))
			derived := base64.RawURLEncoding.EncodeToString(h[:])
			if derived != capturedAuthChallenge {
				dump, _ := httputil.DumpRequest(r, true)
				t.Errorf("PKCE failure: derived=%s challenge=%s\n%s",
					derived, capturedAuthChallenge, dump)
				w.WriteHeader(400)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "fake-token",
				"id_token":     "fake-id",
				"token_type":   "Bearer",
			})
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer fake-token" {
				w.WriteHeader(401)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{
				"sub":   "user-123",
				"email": "alice@example.com",
				"name":  "Alice",
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer idp.Close()

	cfg := OIDCConfig{
		Issuer:       idp.URL,
		ClientID:     "oblivra-client",
		ClientSecret: "shh",
		RedirectURL:  "http://oblivra.local/api/v1/auth/oidc/callback",
		RoleMapper: func(claims map[string]any) string {
			if claims["email"] == "alice@example.com" {
				return "analyst"
			}
			return ""
		},
	}
	h := NewOIDC(cfg)
	if !h.Configured() {
		t.Fatal("OIDC reported not-configured")
	}

	// Drive Login. This 302s to the IdP's /auth endpoint with state +
	// challenge as query params.
	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest("GET", "/api/v1/auth/oidc/login", nil)
	h.Login(loginRec, loginReq)
	if loginRec.Code != http.StatusFound {
		t.Fatalf("login: got %d", loginRec.Code)
	}
	loc := loginRec.Header().Get("Location")
	u, _ := url.Parse(loc)
	state := u.Query().Get("state")
	if state == "" {
		t.Fatal("login did not include a state nonce")
	}
	// Simulate the browser following the redirect to the IdP. The mock
	// captures `state` and the PKCE challenge there; we need it for the
	// next step.
	resp, err := http.Get(loc)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if state != capturedAuthState {
		t.Fatalf("state mismatch: redirect=%q captured=%q", state, capturedAuthState)
	}
	if capturedAuthChallenge == "" {
		t.Fatal("no PKCE challenge captured")
	}

	// Drive Callback with the captured code/state pair.
	cbReq := httptest.NewRequest("GET",
		"/api/v1/auth/oidc/callback?code=fake-code&state="+url.QueryEscape(state), nil)
	cbRec := httptest.NewRecorder()
	h.Callback(cbRec, cbReq)
	if cbRec.Code != http.StatusOK {
		t.Fatalf("callback: %d body=%s", cbRec.Code, cbRec.Body.String())
	}
	if !strings.Contains(cbRec.Body.String(), `"role":"analyst"`) {
		t.Errorf("expected role:analyst in response, got %s", cbRec.Body.String())
	}
	if capturedTokenReq.codeVerifier == "" {
		t.Error("token endpoint never received a code_verifier")
	}

	// Replay the same state — must fail.
	replayReq := httptest.NewRequest("GET",
		"/api/v1/auth/oidc/callback?code=fake-code-2&state="+url.QueryEscape(state), nil)
	replayRec := httptest.NewRecorder()
	h.Callback(replayRec, replayReq)
	if replayRec.Code != http.StatusUnauthorized {
		t.Errorf("replay should be rejected, got %d", replayRec.Code)
	}
}

func TestOIDCNotConfigured(t *testing.T) {
	h := NewOIDC(OIDCConfig{})
	rec := httptest.NewRecorder()
	h.Login(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}
