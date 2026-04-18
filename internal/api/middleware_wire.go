package api

// middleware_wire.go — wires tenantMiddleware into the RESTServer handler chain.
//
// tenantMiddleware must run AFTER APIKeyMiddleware (so auth.UserFromContext is
// populated) and BEFORE any route handler (so TenantFromContext is available).
//
// The existing NewRESTServer in rest.go builds:
//
//   mux → finalHandler (auth wrapping) → secureMiddleware (CORS/headers/rate)
//
// We insert tenantMiddleware between finalHandler and secureMiddleware:
//
//   mux → finalHandler → tenantMiddleware → secureMiddleware
//
// Rather than modifying the large rest.go constructor directly, we provide
// WrapWithTenantMiddleware(), called in the container/app wiring.
// This keeps rest.go clean and allows the tenant layer to be tested in isolation.

import "net/http"

// WrapWithTenantMiddleware wraps an existing handler with the tenant isolation
// middleware. Call this on the RESTServer's internal http.Server.Handler after
// NewRESTServer() returns.
//
//   restSrv := api.NewRESTServer(...)
//   restSrv.WrapHandler(restSrv.WrapWithTenantMiddleware)
//
func (s *RESTServer) WrapWithTenantMiddleware(next http.Handler) http.Handler {
	return s.tenantMiddleware(next)
}

// WrapHandler replaces the server's root handler with the result of applying
// wrapFn to the existing handler. Used for composing middleware layers after
// server construction without touching the constructor.
func (s *RESTServer) WrapHandler(wrapFn func(http.Handler) http.Handler) {
	if s.server != nil && s.server.Handler != nil {
		s.server.Handler = wrapFn(s.server.Handler)
	}
}
