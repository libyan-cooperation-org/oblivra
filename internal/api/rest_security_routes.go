package api

import "net/http"

// rest_security_routes.go — registers security hardening + agent management endpoints.
//
// Routes:
//   GET  /api/v1/detection/explained    — explainability ring buffer
//   POST /api/v1/agent/fleet/config     — push fleet config to an agent
//   POST /api/v1/agent/action           — dispatch response action to an agent
//
// Call initSecurityRoutes(mux) from NewRESTServer after all existing routes.

func (s *RESTServer) initSecurityRoutes(mux *http.ServeMux) {
	// Detection explainability ring buffer
	mux.HandleFunc("/api/v1/detection/explained", s.handleExplainedMatches)

	// Agent fleet management — config push and action dispatch
	mux.HandleFunc("/api/v1/agent/fleet/config", s.handleAgentFleetConfig)
	mux.HandleFunc("/api/v1/agent/action", s.handleAgentAction)
}
