package api

// TLS state REST endpoint (Slice 3).
//
// GET /api/v1/tls/state — anon-readable, returns:
//   { "off": bool, "reason": string }
//
// The frontend reads this to render the plaintext banner in the
// chrome (similar to crisis mode). Anon-bypassed in the auth chain
// because the operator NEEDS to know about the security posture
// even on a freshly-loaded login page.

import "net/http"

// TLSStateProvider is the surface the api package needs. The concrete
// impl lives in internal/security; we accept an interface here to
// dodge the api ↔ services cycle.
type TLSStateProvider interface {
	IsTLSOff() bool
}

func (s *RESTServer) handleTLSState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	off := false
	reason := ""
	if s.tlsState != nil && s.tlsState.IsTLSOff() {
		off = true
		reason = "tls.mode=off — fleet HMAC + payloads travel in plaintext. Set tls.mode: on for production."
	}
	s.jsonResponse(w, http.StatusOK, map[string]any{
		"off":    off,
		"reason": reason,
	})
}

// SetTLSState wires the security.TLSGuardrails into the REST server.
// Same setter pattern as SetSuppression / SetSettings to dodge
// import cycles.
func (s *RESTServer) SetTLSState(p TLSStateProvider) {
	s.tlsState = p
}
