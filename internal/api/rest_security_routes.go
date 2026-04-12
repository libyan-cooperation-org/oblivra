package api

// rest_security_routes.go — registers the security hardening endpoints added
// by the audit implementation cycle.
//
// Routes added here:
//   GET /api/v1/detection/explained  — explainability ring buffer
//
// These routes follow the same pattern as the rest of rest.go but are isolated
// here so the large constructor stays readable.
//
// IMPORTANT: Call RegisterSecurityRoutes(mux) from within NewRESTServer,
// immediately after the existing route blocks, passing the *http.ServeMux.
// Alternatively, wire via the init_routes hook below which is called at the
// end of NewRESTServer via s.initSecurityRoutes(mux).

import "net/http"

// initSecurityRoutes registers the audit-implementation routes on the provided mux.
// Called from NewRESTServer after all existing routes are registered.
func (s *RESTServer) initSecurityRoutes(mux *http.ServeMux) {
	// Detection explainability ring buffer — "why did this alert fire?"
	mux.HandleFunc("/api/v1/detection/explained", s.handleExplainedMatches)
}
