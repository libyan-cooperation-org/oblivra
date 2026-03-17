package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/security"
)

// IdentityProvider defines the subset of IdentityService required by the REST API.
type IdentityProvider interface {
	LoginLocal(email, password string) (*database.User, error)
	GetUser(id string) (*database.User, error)
	GetOIDCURL() (string, error)
	GetSAMLURL() (string, error)
	HandleOIDCCallback(code string) (*database.User, error)
	HandleSAMLCallback(data string) (*database.User, error)
}

// RESTServer exposes backend capabilities to external clients (headless mode)
type RESTServer struct {
	port     int
	server   *http.Server
	siem     database.SIEMStore
	pipeline *ingest.Pipeline
	auth     *auth.APIKeyMiddleware
	identity IdentityProvider
	bus      *eventbus.Bus
	log      *logger.Logger
	attest   *attestation.AttestationService
	certManager *security.CertificateManager
	agents   map[string]*AgentInfo // registered agent fleet
	limiter  *rate.Limiter
	upgrader websocket.Upgrader

	// Connection tracking
	activeWS int64
	maxWS    int64
}

// AgentInfo tracks a registered agent.
type AgentInfo struct {
	ID         string    `json:"id"`
	Hostname   string    `json:"hostname"`
	OS         string    `json:"os"`
	Arch       string    `json:"arch"`
	Version    string    `json:"version"`
	Collectors []string  `json:"collectors"`
	LastSeen   string    `json:"last_seen"`
	Status     string    `json:"status"`
}

// SearchRequest defines the JSON body for SIEM search endpoints
type SearchRequest struct {
	Query   string                 `json:"query"`
	Filters map[string]interface{} `json:"filters"`
}

// NewRESTServer configures the HTTP router and middleware
func NewRESTServer(port int, siem database.SIEMStore, pipeline *ingest.Pipeline, attest *attestation.AttestationService, authMw *auth.APIKeyMiddleware, identity IdentityProvider, bus *eventbus.Bus, certManager *security.CertificateManager, log *logger.Logger) *RESTServer {
	s := &RESTServer{
		port:     port,
		siem:     siem,
		pipeline: pipeline,
		auth:     authMw,
		identity: identity,
		bus:      bus,
		log:      log,
		attest:   attest,
		certManager: certManager,
		agents:   make(map[string]*AgentInfo),
		limiter:  rate.NewLimiter(rate.Limit(20), 50), // 20 req/sec, burst of 50
		maxWS:    100,                               // Max 100 concurrent websocket listeners
		upgrader: websocket.Upgrader{
			// Restrict WebSocket upgrades to same-origin and explicitly allowed origins.
			// Do NOT allow all origins — any web page could connect and receive live event data.
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					// Non-browser clients (CLI agents) omit Origin header; allow them.
					return true
				}
				// Allow same-host requests (Wails desktop shell and localhost agents)
				host := r.Host
				allowed := []string{
					"http://" + host,
					"https://" + host,
					"http://localhost",
					"https://localhost",
					"wails://wails",
				}
				for _, a := range allowed {
					if origin == a || len(origin) > len(a) && origin[:len(a)+1] == a+":" {
						return true
					}
				}
				log.Warn("[WS] Rejected connection from disallowed origin: %s", origin)
				return false
			},
		},
	}

	mux := http.NewServeMux()

	// SIEM endpoints
	mux.HandleFunc("/api/v1/siem/search", s.handleSIEMSearch)
	mux.HandleFunc("/api/v1/alerts", s.handleAlertsList)

	// Authentication endpoints
	mux.HandleFunc("/api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("/api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("/api/v1/auth/me", s.handleMe)
	mux.HandleFunc("/api/v1/auth/oidc/login", s.handleOIDCLogin)
	mux.HandleFunc("/api/v1/auth/oidc/callback", s.handleOIDCCallback)
	mux.HandleFunc("/api/v1/auth/saml/login", s.handleSAMLLogin)
	mux.HandleFunc("/api/v1/auth/saml/callback", s.handleSAMLCallback)
	mux.HandleFunc("/api/v1/auth/saml/metadata", s.handleSAMLMetadata)

	// Events endpoint
	mux.HandleFunc("/api/v1/events", s.handleEvents)

	// OpenAPI endpoints
	mux.HandleFunc("/api/v1/openapi.yaml", s.handleOpenAPI)
	mux.HandleFunc("/api/v1/docs", s.handleDocs)

	// System endpoints
	mux.HandleFunc("/api/v1/ingest/status", s.handleIngestStatus)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/debug/attestation", s.handleAttestation)

	// Agent endpoints
	mux.HandleFunc("/api/v1/agent/ingest", s.handleAgentIngest)
	mux.HandleFunc("/api/v1/agent/register", s.handleAgentRegister)
	mux.HandleFunc("/api/v1/agent/fleet", s.handleAgentFleet)

	var handler http.Handler = mux

	// Wrap entire mux with Authentication middleware if provided
	if s.auth != nil {
		handler = s.auth.Middleware(handler)
	}

	// Wrap entire router with security middleware (CORS, Headers, Rate Limiting)
	handler = s.secureMiddleware(handler)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}

	return s
}

// Start spawns the HTTP listener in the background
func (s *RESTServer) Start() {
	s.log.Info("[REST] Starting headless API server on port %d", s.port)
	
	if s.certManager != nil {
		// Initial load
		if err := s.certManager.Load(); err != nil {
			s.log.Warn("[REST] Initial TLS certificate load failed: %v", err)
		}
		
		s.server.TLSConfig = &tls.Config{
			GetCertificate: s.certManager.GetCertificate,
			MinVersion:     tls.VersionTLS13, // TLS 1.2 is deprecated; require 1.3 for all agent channels
		}
	}

	go func() {
		// If certManager is missing, fail hard.
		if s.certManager == nil {
			s.log.Error("[REST] TLS Certificate Manager NOT configured. Headless API is NOT running.")
			return
		}

		err := s.server.ListenAndServeTLS("", "") // cert/key provided by GetCertificate
		if err != nil && err != http.ErrServerClosed {
			s.log.Error("[REST] TLS server failed: %v. Headless API is NOT running.", err)
		}
	}()
}

// Stop gracefully shuts down the HTTP server
func (s *RESTServer) Stop(ctx context.Context) error {
	s.log.Info("[REST] Shutting down headless API server...")
	return s.server.Shutdown(ctx)
}

// IsTLS returns true if the server is configured with TLS certificates
func (s *RESTServer) IsTLS() bool {
	home, _ := os.UserHomeDir()
	certPath := filepath.Join(home, ".oblivrashell", "cert.pem")
	_, err := os.Stat(certPath)
	return err == nil
}

func (s *RESTServer) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// --- Middleware ---

func (s *RESTServer) secureMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Rate Limiting
		if !s.limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// 2. CORS Headers — restrict to local Wails frontend origins
		origin := r.Header.Get("Origin")
		allowedOrigins := map[string]bool{
			"http://localhost":        true,
			"http://localhost:5173":   true, // Vite dev server
			"https://wails.localhost": true,
			"wails://wails":           true,
		}
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key")
		w.Header().Set("Vary", "Origin")

		// 3. Security Headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- Handlers ---

func (s *RESTServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "alive", "time": time.Now().Format(time.RFC3339)})
}

func (s *RESTServer) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// In a real app, check DB connections, Vault unlock status, etc.
	// We assume readiness if the server is running.
	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *RESTServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	eps := int64(0)
	total := int64(0)
	if s.pipeline != nil {
		m := s.pipeline.GetMetrics()
		eps = m.EventsPerSecond
		total = m.TotalProcessed
	}

	activeAlerts := 0
	if s.siem != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		alerts, _ := s.siem.SearchHostEvents(ctx, "EventType:security_alert", 100)
		cancel()
		activeAlerts = len(alerts)
	}

	// Output minimal Prometheus formatted metrics
	fmt.Fprintf(w, "# HELP oblivra_ingest_eps Current events processed per second\n")
	fmt.Fprintf(w, "# TYPE oblivra_ingest_eps gauge\n")
	fmt.Fprintf(w, "oblivra_ingest_eps %d\n\n", eps)

	fmt.Fprintf(w, "# HELP oblivra_ingest_total Total events processed\n")
	fmt.Fprintf(w, "# TYPE oblivra_ingest_total counter\n")
	fmt.Fprintf(w, "oblivra_ingest_total %d\n\n", total)

	fmt.Fprintf(w, "# HELP oblivra_active_alerts Current active security anomalies\n")
	fmt.Fprintf(w, "# TYPE oblivra_active_alerts gauge\n")
	fmt.Fprintf(w, "oblivra_active_alerts %d\n", activeAlerts)
}

func (s *RESTServer) handleSIEMSearch(w http.ResponseWriter, r *http.Request) {
	// 1. RBAC Check: Require Analyst or Admin
	role := auth.GetRole(r.Context())
	if role != auth.RoleAnalyst && role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Analysts only", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	// limitStr unused.

	limit := 100 // default
	var req SearchRequest
	if r.Method == http.MethodPost {
		// Enforce maximum body size of 1MB to prevent JSON decoding OOM DoS
		r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.Query != "" {
				query = req.Query
			}
			const maxSearchLimit = 1000
			if l, ok := req.Filters["limit"].(float64); ok {
				limit = int(l)
			} else if l, ok := req.Filters["limit"].(int); ok {
				limit = l
			}
			// Cap to prevent OOM from malicious large-limit requests
			if limit > maxSearchLimit {
				limit = maxSearchLimit
			}
		} else {
			http.Error(w, "Invalid request body or body too large", http.StatusBadRequest)
			return
		}
	}

	if s.siem == nil {
		http.Error(w, "SIEM functionality disabled", http.StatusNotImplemented)
		return
	}

	// 2. Multi-Tenant Enforcement: Scoped Search
	identityUser := auth.UserFromContext(r.Context())
	if identityUser != nil && identityUser.TenantID != "" && identityUser.TenantID != "GLOBAL" {
		// Prepend TenantID filter if not already present or if we want to force scope.
		// For MVP, we'll force the scope.
		query = fmt.Sprintf("TenantID:%s AND (%s)", identityUser.TenantID, query)
	}

	events, err := s.siem.SearchHostEvents(r.Context(), query, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"query":  query,
		"count":  len(events),
		"events": events,
	})
}

func (s *RESTServer) handleAlertsList(w http.ResponseWriter, r *http.Request) {
	// 1. RBAC Check: Require Analyst or Admin
	role := auth.GetRole(r.Context())
	if role != auth.RoleAnalyst && role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Analysts only", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For now, this could query a local SQLite DB for active alerts.
	// Since we haven't implemented DB persistence for `detection.Match` yet,
	// returning a placeholder or querying SIEM events labeled as "security_alert".

	if s.siem == nil {
		http.Error(w, "SIEM functionality disabled", http.StatusNotImplemented)
		return
	}

	// 2. Multi-Tenant Enforcement: Scoped Alerts
	identityUser := auth.UserFromContext(r.Context())
	query := "EventType:security_alert"
	if identityUser != nil && identityUser.TenantID != "" && identityUser.TenantID != "GLOBAL" {
		query = fmt.Sprintf("TenantID:%s AND %s", identityUser.TenantID, query)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	alerts, err := s.siem.SearchHostEvents(ctx, query, 100)
	cancel()
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"active_alerts": len(alerts),
		"alerts":        alerts,
	})
}

func (s *RESTServer) handleIngestStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.pipeline == nil {
		http.Error(w, "Ingestion pipeline not active", http.StatusNotImplemented)
		return
	}

	metrics := s.pipeline.GetMetrics()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"eps":             metrics.EventsPerSecond,
		"total_processed": metrics.TotalProcessed,
		"buffer_usage":    metrics.BufferUsage,
		"buffer_capacity": metrics.BufferCapacity,
		"dropped_events":  metrics.DroppedEvents,
	})
}

func (s *RESTServer) handleAttestation(w http.ResponseWriter, r *http.Request) {
	// 1. RBAC Check: Require Admin
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := s.attest.GetStatus()
	buildInfo := attestation.GetBuildInfo()

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"attestation": status,
		"build":       buildInfo,
		"uptime_secs": time.Since(attestation.StartupTime).Seconds(),
	})
}

func (s *RESTServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	// 1. Connection Limit Check
	if atomic.LoadInt64(&s.activeWS) >= s.maxWS {
		s.log.Warn("[REST] WebSocket connection limit reached (%d)", s.maxWS)
		http.Error(w, "Too many concurrent connections", http.StatusServiceUnavailable)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("[REST] WebSocket upgrade failed: %v", err)
		return
	}
	
	atomic.AddInt64(&s.activeWS, 1)
	defer atomic.AddInt64(&s.activeWS, -1)
	defer conn.Close()

	if s.bus == nil {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Event bus not configured"}`))
		return
	}

	clientAddr := r.RemoteAddr
	s.log.Info("[REST] Client connected to event stream: %s", clientAddr)

	subCh := make(chan eventbus.Event, 100)
	ctxDone := make(chan struct{})

	// Unsubscribe when the client disconnects to prevent goroutine/memory leaks.
	subID := s.bus.SubscribeWithID(eventbus.AllEvents, func(e eventbus.Event) {
		select {
		case subCh <- e:
		case <-ctxDone:
			return
		default:
			// drop if client is too slow
		}
	})
	defer func() {
		close(ctxDone)
		s.bus.Unsubscribe(subID)
		s.log.Info("[REST] Client disconnected from event stream: %s (unsubscribed)", clientAddr)
	}()

	// Read loop to detect client disconnect
	go func() {
		defer close(ctxDone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Set ping handler
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	for {
		select {
		case event, ok := <-subCh:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			// Set write deadline for the event
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ctxDone:
			return
		}
	}
}

func (s *RESTServer) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	http.ServeFile(w, r, "docs/openapi.yaml")
}

func (s *RESTServer) handleDocs(w http.ResponseWriter, r *http.Request) {
	// SECURITY: Disabled to prevent leakage via external CDN links in production-like builds.
	// Documentation should be served from a separate, secure portal or localized assets.
	http.Error(w, "Documentation endpoint disabled for security", http.StatusForbidden)
}

func (s *RESTServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if s.identity == nil {
		http.Error(w, "Identity service disabled", http.StatusNotImplemented)
		return
	}

	user, err := s.identity.LoginLocal(req.Email, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// For the MVP, we'll return the user and a placeholder token.
	// In Phase 0.5, we'll implement full JWT/session persistence.
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": "oblivra-dev-key", // Temporary: using dev-key to pass existing middleware
	})
}

func (s *RESTServer) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "Identity service disabled", http.StatusNotImplemented)
		return
	}

	url, err := s.identity.GetOIDCURL()
	if err != nil {
		http.Error(w, "Failed to generate OIDC redirect", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *RESTServer) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing state or code", http.StatusBadRequest)
		return
	}

	user, err := s.identity.HandleOIDCCallback(code)
	if err != nil {
		http.Error(w, "Federated authentication failed", http.StatusUnauthorized)
		return
	}

	// Redirect to frontend with a temporary session fragment (Phase 0.5 will use secure cookies)
	http.Redirect(w, r, fmt.Sprintf("/?user=%s&token=oblivra-dev-key", user.ID), http.StatusTemporaryRedirect)
}

func (s *RESTServer) handleSAMLLogin(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "Identity service disabled", http.StatusNotImplemented)
		return
	}

	url, err := s.identity.GetSAMLURL()
	if err != nil {
		http.Error(w, "Failed to generate SAML redirect", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *RESTServer) handleSAMLCallback(w http.ResponseWriter, r *http.Request) {
	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		http.Error(w, "Missing SAML response", http.StatusBadRequest)
		return
	}

	user, err := s.identity.HandleSAMLCallback(samlResponse)
	if err != nil {
		http.Error(w, "SAML authentication failed", http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/?user=%s&token=oblivra-dev-key", user.ID), http.StatusTemporaryRedirect)
}

func (s *RESTServer) handleSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	// Placeholder: In a real implementation, this would be generated by a SAML library
	fmt.Fprintf(w, "<EntityDescriptor>OBLIVRA-SAML-V2-MVP</EntityDescriptor>")
}

func (s *RESTServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"status": "logged_out"})
}

func (s *RESTServer) handleMe(w http.ResponseWriter, r *http.Request) {
	// This relies on the middleware having injected the user into context.
	identityUser := auth.UserFromContext(r.Context())
	if identityUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	s.jsonResponse(w, http.StatusOK, identityUser)
}
