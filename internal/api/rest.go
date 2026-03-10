package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// RESTServer exposes backend capabilities to external clients (headless mode)
type RESTServer struct {
	port     int
	server   *http.Server
	siem     database.SIEMStore
	pipeline *ingest.Pipeline
	auth     *auth.APIKeyMiddleware
	bus      *eventbus.Bus
	log      *logger.Logger
	attest   *attestation.AttestationService
	agents   map[string]*AgentInfo // registered agent fleet
	limiter  *rate.Limiter
	upgrader websocket.Upgrader
}

// AgentInfo tracks a registered agent.
type AgentInfo struct {
	ID         string    `json:"id"`
	Hostname   string    `json:"hostname"`
	OS         string    `json:"os"`
	Arch       string    `json:"arch"`
	Version    string    `json:"version"`
	Collectors []string  `json:"collectors"`
	LastSeen   time.Time `json:"last_seen"`
	Status     string    `json:"status"`
}

// SearchRequest defines the JSON body for SIEM search endpoints
type SearchRequest struct {
	Query   string                 `json:"query"`
	Filters map[string]interface{} `json:"filters"`
}

// NewRESTServer configures the HTTP router and middleware
func NewRESTServer(port int, siem database.SIEMStore, pipeline *ingest.Pipeline, attest *attestation.AttestationService, authMw *auth.APIKeyMiddleware, bus *eventbus.Bus, log *logger.Logger) *RESTServer {
	s := &RESTServer{
		port:     port,
		siem:     siem,
		pipeline: pipeline,
		auth:     authMw,
		bus:      bus,
		log:      log,
		attest:   attest,
		agents:   make(map[string]*AgentInfo),
		limiter:  rate.NewLimiter(rate.Limit(20), 50), // 20 req/sec, burst of 50
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins for the API stream for now
		},
	}

	mux := http.NewServeMux()

	// SIEM endpoints
	mux.HandleFunc("/api/v1/siem/search", s.handleSIEMSearch)
	mux.HandleFunc("/api/v1/alerts", s.handleAlertsList)

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
	go func() {
		// Attempt TLS if certificates are provisioned in the user data dir
		certPath := "cert.pem" // Hardcoded stub for phase 2 setup
		keyPath := "key.pem"

		// If no TLS provisioned, fallback to HTTP (or fail based on strictness)
		err := s.server.ListenAndServeTLS(certPath, keyPath)
		if err != nil {
			s.log.Warn("[REST] TLS not configured, falling back to HTTP: %v", err)
			if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.log.Error("[REST] Server failed: %v", err)
			}
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
			if l, ok := req.Filters["limit"].(float64); ok {
				limit = int(l)
			} else if l, ok := req.Filters["limit"].(int); ok {
				limit = l
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	alerts, err := s.siem.SearchHostEvents(ctx, "EventType:security_alert", 100)
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
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("[REST] WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	if s.bus == nil {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Event bus not configured"}`))
		return
	}

	clientAddr := r.RemoteAddr
	s.log.Info("[REST] Client connected to event stream: %s", clientAddr)

	subCh := make(chan eventbus.Event, 100)

	s.bus.Subscribe(eventbus.AllEvents, func(e eventbus.Event) {
		select {
		case subCh <- e:
		default:
			// drop if client is too slow
		}
	})

	for {
		event := <-subCh
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			s.log.Info("[REST] Client disconnected from event stream: %s", clientAddr)
			break
		}
	}
}

func (s *RESTServer) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	http.ServeFile(w, r, "docs/openapi.yaml")
}

func (s *RESTServer) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	html := `<!DOCTYPE html>
<html>
<head>
  <title>Oblivra API Documentation</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
  <style>
    body { margin: 0; padding: 0; }
  </style>
</head>
<body>
  <redoc spec-url='/api/v1/openapi.yaml'></redoc>
  <script src="https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js"> </script>
</body>
</html>`
	w.Write([]byte(html))
}
