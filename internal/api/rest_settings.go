package api

// Settings REST endpoints (Phase 32).
//
// GET    /api/v1/settings/{key}   → { key, value }   (404 when unset)
// PUT    /api/v1/settings/{key}   ← { value }        (admin only)
// GET    /api/v1/settings/_export → returns the full key/value map,
//                                   with sensitive keys redacted.
//
// Why not a single bulk endpoint? Because settings are mutated
// independently (one key per save in the UI), and the per-key shape
// keeps the audit log + change-event bus tidy — every Set publishes
// `settings.changed:{key}` rather than a fan-out of N events.
//
// Sensitive keys (password / token / secret / webhook / credential /
// private_key / auth_key / client_secret) are vault-encrypted at rest
// by SettingsService.Set; the GET path decrypts back to plaintext if
// the vault is unlocked, otherwise returns the ciphertext envelope so
// the operator can see "this is set" without the value.
//
// The export endpoint always REDACTS sensitive keys (returns the
// constant `***` instead of the plaintext) so the bundle is safe to
// ship to support without manual scrubbing.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kingknull/oblivrashell/internal/auth"
)

// auditSettingKey returns a value safe to write into the audit log's
// resource_id column. Audit fix #7 — for sensitive keys (anything
// matching the isSensitiveSettingKey filter, e.g. `slack_webhook_token`,
// `oidc_client_secret`) we replace the literal name with a deterministic
// short hash. Non-sensitive keys (refresh_interval_secs, etc.) pass
// through verbatim because they're useful for support-bundle review.
//
// Why deterministic hashing instead of redacting to a constant: an
// auditor still needs to correlate "was this same secret touched twice
// in a row" without learning which secret it is. Same input → same
// 8-hex-character token, different secrets → almost-certainly distinct
// tokens.  Reversing back to a name requires either knowing the
// candidate set or running a brute force across the sensitive-key
// vocabulary — fine for a forensic walk-through, hostile-to-quick-grep
// for support-bundle skimmers.
func auditSettingKey(key string) string {
	if !isSensitiveSettingKey(key) {
		return key
	}
	h := sha256.Sum256([]byte(key))
	return "secret:" + hex.EncodeToString(h[:4]) // 8 hex chars
}

// Sensitive-key fragments that trigger redaction on export. Kept in
// sync with services/settings_service.go's isSensitiveKey but lives
// here so the api package doesn't need to import services.
var sensitiveFragments = []string{
	"password", "passphrase", "secret", "token", "webhook",
	"credential", "private_key", "auth_key", "client_secret",
	"api_key",
}

func isSensitiveSettingKey(key string) bool {
	lower := strings.ToLower(key)
	for _, frag := range sensitiveFragments {
		if strings.Contains(lower, frag) {
			return true
		}
	}
	return false
}

// handleSettingsRoot dispatches:
//   /api/v1/settings/{key}      → handleSettingByKey
//   /api/v1/settings/_export    → handleSettingsExport
func (s *RESTServer) handleSettingsRoot(w http.ResponseWriter, r *http.Request) {
	if s.settings == nil {
		http.Error(w, "Settings service not available", http.StatusServiceUnavailable)
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/settings/")
	if rest == "_export" {
		s.handleSettingsExport(w, r)
		return
	}
	if rest == "" || strings.Contains(rest, "/") {
		http.Error(w, "settings key required in path", http.StatusBadRequest)
		return
	}
	s.handleSettingByKey(w, r, rest)
}

func (s *RESTServer) handleSettingByKey(w http.ResponseWriter, r *http.Request, key string) {
	role := auth.GetRole(r.Context())

	switch r.Method {
	case http.MethodGet:
		// Reading is analyst+ — operators on the queue may need to
		// see config (refresh interval, alert noise floor, etc.).
		if role != auth.RoleAnalyst && role != auth.RoleAdmin && role != auth.RoleReadOnly {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		val, err := s.settings.Get(key)
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		s.jsonResponse(w, http.StatusOK, map[string]any{
			"key":   key,
			"value": val,
		})

	case http.MethodPut, http.MethodPost:
		// Mutations are admin-only — settings drive platform behaviour
		// and shouldn't be operator-tunable without elevated rights.
		if role != auth.RoleAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
		var body struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.settings.Set(key, body.Value); err != nil {
			http.Error(w, "set failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// Audit-log every settings mutation. `value` is intentionally
		// excluded from the audit row — settings can hold credentials
		// (token, smtp_password, slack_webhook). The settings service
		// already log-redacts; we mirror that here.
		//
		// Audit fix #7 — also hash sensitive KEY NAMES via
		// auditSettingKey(), so a support-bundle viewer can't grep
		// out the inventory of integrations we hold credentials for.
		s.appendAuditEntry(connectorActor(r), "settings.set",
			auditSettingKey(key),
			"len="+itoa(len(body.Value)), r)
		s.jsonResponse(w, http.StatusOK, map[string]any{"ok": true, "key": key})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSettingsExport returns every settings row, with sensitive
// values redacted. Useful for support bundles and config-drift checks.
// Admin-only.
func (s *RESTServer) handleSettingsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	// We don't have a List() on the SettingsProvider yet (the concrete
	// service would need it). For now return 501 with a clear message
	// so the UI can degrade gracefully — adding List() is a small
	// follow-up but requires touching the services package, which is
	// out of scope for this REST seam.
	http.Error(w, "Settings export endpoint requires SettingsService.List() — not yet exposed via the SettingsProvider interface.", http.StatusNotImplemented)
}

// itoa avoids importing strconv just for one call site. Same pattern
// as the rest of this package.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
