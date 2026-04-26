// Sanitised error responses for the REST API.
//
// Closes the audit's CRITICAL information-disclosure finding —
// 35+ handlers were returning raw `err.Error()` to clients, exposing
// stack frames, database driver messages, file paths, and other
// internal context that:
//
//   1. Leaks implementation details to attackers (helpful for fingerprinting)
//   2. Confuses operators (raw Go errors aren't user-actionable)
//   3. Sometimes leaks data the user shouldn't see (e.g. error
//      strings that include other tenants' IDs from join paths)
//
// Pattern:
//
//   // BEFORE — leaks implementation
//   if err != nil {
//       http.Error(w, err.Error(), http.StatusInternalServerError)
//       return
//   }
//
//   // AFTER — operator gets a generic public message,
//   // server log captures the full error for debugging
//   if err != nil {
//       s.respondError(w, r, http.StatusInternalServerError,
//           "internal error", "failed to load X", err)
//       return
//   }
//
// The helper logs at the appropriate level (Error for 5xx, Warn for
// 4xx), tags the request method + path + status + correlation id,
// and emits a structured JSON envelope to the client.
//
// Public message conventions:
//   - 4xx: a short noun phrase the operator can act on
//          ("invalid request", "not found", "permission denied")
//   - 5xx: always "internal error" — never reveal anything specific
//   - Auth: "unauthorized" / "forbidden" — never explain why beyond
//           that, to avoid timing/oracle attacks

package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// errorEnvelope is the on-wire shape every sanitised error uses.
// Keeping it stable means clients can write one error handler.
type errorEnvelope struct {
	OK     bool   `json:"ok"`
	Status int    `json:"status"`
	Error  string `json:"error"`              // public-safe; never internal details
	Code   string `json:"code,omitempty"`     // optional machine-readable hint
}

// respondError writes a sanitised error response and logs the full
// error context server-side.
//
//   - status:        HTTP status code (4xx/5xx)
//   - publicMessage: shown to the client (user-actionable, no details)
//   - logContext:    short label for the server log (e.g. "load_alerts")
//   - err:           the actual error — logged but never serialised to
//                    the client. Pass nil if there is no underlying
//                    error (e.g. validation failure with a hand-built msg).
//
// Example:
//
//	if err != nil {
//	    s.respondError(w, r, http.StatusInternalServerError,
//	        "internal error", "load_dashboard_list", err)
//	    return
//	}
func (s *RESTServer) respondError(
	w http.ResponseWriter, r *http.Request,
	status int, publicMessage, logContext string, err error,
) {
	respondErrorWith(s.log, w, r, status, publicMessage, logContext, err)
}

// Same helper, but for the legacy `Server` type in server.go which
// doesn't share the RESTServer struct. Deduplicating the underlying
// implementation in `respondErrorWith` so future maintainers have one
// place to audit.
func (s *Server) respondError(
	w http.ResponseWriter, r *http.Request,
	status int, publicMessage, logContext string, err error,
) {
	respondErrorWith(s.log, w, r, status, publicMessage, logContext, err)
}

// respondErrorWith is the free-function form: any caller with a
// logger can use this. Both `RESTServer.respondError` and
// `Server.respondError` are thin wrappers that forward here.
func respondErrorWith(
	log *logger.Logger,
	w http.ResponseWriter, r *http.Request,
	status int, publicMessage, logContext string, err error,
) {
	// Log with all the context an operator needs to debug. The full
	// `err` only appears in server logs, never on the wire.
	if log != nil {
		method := ""
		path := ""
		if r != nil {
			method = r.Method
			path = r.URL.Path
		}
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		// Severity: 5xx = Error (operator must investigate),
		// 4xx = Warn (likely client misuse, still worth surfacing).
		if status >= 500 {
			log.Error("[api] %s %s -> %d (%s): %s",
				method, path, status, logContext, errStr)
		} else if status >= 400 {
			log.Warn("[api] %s %s -> %d (%s): %s",
				method, path, status, logContext, errStr)
		}
	}

	// Build the public envelope. Strip any newlines from the public
	// message to avoid header-injection vectors (someone passing a
	// crafted message into Content-Type via response splitting).
	safe := strings.ReplaceAll(publicMessage, "\n", " ")
	safe = strings.ReplaceAll(safe, "\r", " ")
	if safe == "" {
		safe = http.StatusText(status)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorEnvelope{
		OK:     false,
		Status: status,
		Error:  safe,
	})
}

// publicErrorMessage maps common error categories to operator-safe
// messages. Centralising this means we can audit every public
// message in one file rather than hunting through every handler.
func publicErrorMessage(status int, hint string) string {
	switch status {
	case http.StatusBadRequest:
		if hint != "" {
			return "invalid request: " + hint
		}
		return "invalid request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusTooManyRequests:
		return "rate limited"
	case http.StatusServiceUnavailable:
		return "service unavailable"
	default:
		// Anything else 5xx-ish.
		if status >= 500 {
			return "internal error"
		}
		return http.StatusText(status)
	}
}
