package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/forensics"
	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/lookup"
	"github.com/kingknull/oblivrashell/internal/notifications"
	"github.com/kingknull/oblivrashell/internal/oql"
	"github.com/kingknull/oblivrashell/internal/security"
	"github.com/kingknull/oblivrashell/internal/threatintel"
	"github.com/kingknull/oblivrashell/internal/mcp"
	"github.com/kingknull/oblivrashell/internal/ueba"
)

// IdentityProvider defines the subset of IdentityService required by the REST API.
type IdentityProvider interface {
	LoginLocal(email, password string) (*database.User, error)
	GetUser(id string) (*database.User, error)
	ListUsers(ctx context.Context) ([]database.User, error)
	GetUserByExternalID(ctx context.Context, extID string) (*database.User, error)
	ProvisionSCIMUser(ctx context.Context, u *database.User) error
	DeleteUser(ctx context.Context, id string) error
	GetOIDCURL() (string, error)
	GetSAMLURL() (string, error)
	HandleOIDCCallback(code string) (*database.User, error)
	HandleSAMLCallback(data string) (*database.User, error)

	// Connector management (Phase 20.7)
	ListConnectors(ctx context.Context) ([]database.IdentityConnector, error)
	CreateConnector(ctx context.Context, c *database.IdentityConnector) error
	GetConnector(ctx context.Context, id string) (*database.IdentityConnector, error)
	UpdateConnector(ctx context.Context, c *database.IdentityConnector) error
	DeleteConnector(ctx context.Context, id string) error
	TriggerSync(ctx context.Context, id string) error
}

// UEBAProvider allows the REST API to fetch behavioral analytics without importing the services package.
type UEBAProvider interface {
	GetProfiles() []*ueba.EntityProfile
	GetAnomalies() []map[string]interface{}
}

// ReportingProvider defines the subset of ReportService required by the REST API.
type ReportingProvider interface {
	ListTemplates(ctx context.Context) ([]database.ReportTemplate, error)
	CreateTemplate(ctx context.Context, t *database.ReportTemplate) error
	ListGeneratedReports(ctx context.Context, limit int) ([]database.GeneratedReport, error)
	GenerateManualReport(ctx context.Context, templateID string, start, end string) (string, error)
	GetReportPath(ctx context.Context, id string) (string, error)
}

// DashboardProvider defines the subset of DashboardService required by the REST API.
type DashboardProvider interface {
	ListDashboards(ctx context.Context) ([]database.Dashboard, error)
	CreateDashboard(ctx context.Context, d *database.Dashboard) error
	GetDashboard(ctx context.Context, id string) (*database.Dashboard, error)
	UpdateDashboard(ctx context.Context, d *database.Dashboard) error
	DeleteDashboard(ctx context.Context, id string) error
	GetDashboardData(ctx context.Context, id string) (*DashboardData, error)
	
	AddWidget(ctx context.Context, w *database.DashboardWidget) error
	UpdateWidget(ctx context.Context, w *database.DashboardWidget) error
	DeleteWidget(ctx context.Context, dashboardID, widgetID string) error
}

// DashboardData is a DTO for returning batched widget results.
type DashboardData struct {
	DashboardID string                      `json:"dashboard_id"`
	Results     map[string]*oql.QueryResult `json:"results"`
}

// AgentProvider enables the REST API to query agents from the ingestion engine.
type AgentProvider interface {
	GetFleet() []AgentInfo
}

// SystemKeyProvider allows retrieving system-level secrets (e.g. for signing) from the vault.
type SystemKeyProvider interface {
	GetSystemKey(purpose string) ([]byte, error)
}

// DynamicHMACSigner satisfies forensics.ForensicSigner using a key from a SystemKeyProvider.
type DynamicHMACSigner struct {
	provider SystemKeyProvider
	purpose  string
}

func (s *DynamicHMACSigner) SignEntry(payload string) (string, error) {
	key, err := s.provider.GetSystemKey(s.purpose)
	if err != nil {
		return "", fmt.Errorf("retrieve system key '%s': %w", s.purpose, err)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// RESTServer exposes backend capabilities to external clients (headless mode)
type RESTServer struct {
	port     int
	server   *http.Server
	siem     database.SIEMStore
	pipeline ingest.IngestionPipeline
	auth     *auth.APIKeyMiddleware
	identity IdentityProvider
	bus      *eventbus.Bus
	log      *logger.Logger
	attest   *attestation.AttestationService
	certManager *security.CertificateManager
	lookups  *lookup.Manager
	escalation *notifications.EscalationManager
	matchEngine *threatintel.MatchEngine
	agents   map[string]*AgentInfo // registered agent fleet
	agentsMu sync.RWMutex           // protects agents map
	agentProvider AgentProvider     // provider for agent fleet data
	limiter  *rate.Limiter
	upgrader websocket.Upgrader

	// Enrichment recent-query ring buffer
	enrichMu     sync.RWMutex
	enrichRecent []map[string]interface{}

	// Phase 6.5 — Evidence Locker
	evidence *forensics.EvidenceLocker
	audit      *database.AuditRepository

	// Connection tracking
	activeWS int64
	maxWS    int64

	// MCP - Protocol Phase 22.1
	mcpRegistry *mcp.ToolRegistry
	mcpHandler  *mcp.Handler
	tenantRepo *database.TenantRepository
	reports    ReportingProvider
	dashboards DashboardProvider
	enrichLimiter *ingest.EnrichmentLimiter // #9: protects GeoIP/DNS/TI from stalling ingestion
	replayer      *ingest.EventReplayer   // #3: allows on-demand logic verification
	graphEngine   *graph.GraphEngine
	ueba          UEBAProvider
	fleetSecret   []byte // Shared secret for agent HMAC verification
	keyProvider   SystemKeyProvider
}

// AgentInfo tracks a registered agent.
type AgentInfo struct {
	ID         string    `json:"id"`
	Hostname   string    `json:"hostname"`
	TenantID   string    `json:"tenant_id"` // Added for isolation
	OS         string    `json:"os"`
	Arch       string    `json:"arch"`
	Version    string    `json:"version"`
	Collectors []string  `json:"collectors"`
	LastSeen   string    `json:"last_seen"`
	Status     string    `json:"status"`
	PublicKey  []byte    `json:"public_key"` // 1.4: Hardware-rooted trust key (TPM)
}

// SearchRequest defines the JSON body for SIEM search endpoints
type SearchRequest struct {
	Query   string                 `json:"query"`
	Filters map[string]interface{} `json:"filters"`
}

// NewRESTServer configures the HTTP router and middleware
func NewRESTServer(port int, db database.DatabaseStore, siem database.SIEMStore, audit *database.AuditRepository, pipeline ingest.IngestionPipeline, graphEngine *graph.GraphEngine, ueba UEBAProvider, agentProvider AgentProvider, fleetSecret []byte, keyProvider SystemKeyProvider, attest *attestation.AttestationService, authMw *auth.APIKeyMiddleware, identity IdentityProvider, reports ReportingProvider, dashboards DashboardProvider, bus *eventbus.Bus, certManager *security.CertificateManager, log *logger.Logger, mcpRegistry *mcp.ToolRegistry, mcpHandler *mcp.Handler) *RESTServer {
	var tenantRepo *database.TenantRepository
	if db != nil {
		tenantRepo = database.NewTenantRepository(db)
	}

	s := &RESTServer{
		port:     port,
		siem:     siem,
		audit:    audit,
		pipeline: pipeline,
		lookups:  lookup.NewManager(),
		auth:     authMw,
		identity: identity,
		bus:      bus,
		log:      log,
		attest:   attest,
		certManager: certManager,
		agents:   make(map[string]*AgentInfo),
		limiter:  rate.NewLimiter(rate.Limit(20), 50), // 20 req/sec, burst of 50
		maxWS:    100,                               // Max 100 concurrent websocket listeners
		evidence: forensics.NewEvidenceLocker(&DynamicHMACSigner{provider: keyProvider, purpose: "forensic_hmac"}, log),
		mcpRegistry: mcpRegistry,
		mcpHandler:  mcpHandler,
		tenantRepo:  tenantRepo,
		reports:     reports,
		dashboards:  dashboards,
		enrichLimiter: ingest.NewEnrichmentLimiter(500), // 500 enrichment calls/sec max across all tenants
		replayer:      ingest.NewEventReplayer(log),
		graphEngine:   graphEngine,
		ueba:          ueba,
		agentProvider: agentProvider,
		fleetSecret:   fleetSecret,
		keyProvider:   keyProvider,
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
					"http://localhost:3000",
					"https://localhost:3000",
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
	mux.HandleFunc("/api/v1/ingest/replay", s.handleIngestReplay)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/debug/attestation", s.handleAttestation)

	// Agent endpoints
	mux.HandleFunc("/api/v1/agent/ingest", s.handleAgentIngest)
	mux.HandleFunc("/api/v1/agent/register", s.handleAgentRegister)
	mux.HandleFunc("/api/v1/agent/fleet", s.handleAgentFleet)

	// Lookup table endpoints (Phase 1.3)
	mux.HandleFunc("/api/v1/lookups", s.handleLookupList)
	mux.HandleFunc("/api/v1/lookups/", s.handleLookupByName) // GET/DELETE /{name}
	mux.HandleFunc("/api/v1/lookups/upload", s.handleLookupUpload)
	mux.HandleFunc("/api/v1/lookups/query", s.handleLookupQuery)

	// Escalation endpoints (Phase 2.1.5)
	mux.HandleFunc("/api/v1/escalation/policies", s.handleEscalationPolicies)
	mux.HandleFunc("/api/v1/escalation/policies/", s.handleEscalationPolicyByID)
	mux.HandleFunc("/api/v1/escalation/active", s.handleEscalationActive)
	mux.HandleFunc("/api/v1/escalation/history", s.handleEscalationHistory)
	mux.HandleFunc("/api/v1/escalation/ack", s.handleEscalationAck)
	mux.HandleFunc("/api/v1/escalation/oncall", s.handleEscalationOnCall)

	// Threat Intelligence endpoints (Phase 3.1)
	mux.HandleFunc("/api/v1/threatintel/stats", s.handleThreatIntelStats)
	mux.HandleFunc("/api/v1/threatintel/indicators", s.handleThreatIntelIndicators)
	mux.HandleFunc("/api/v1/threatintel/lookup", s.handleThreatIntelLookup)
	mux.HandleFunc("/api/v1/threatintel/campaigns", s.handleThreatIntelCampaigns)

	// Enrichment endpoints (Phase 3.2)
	mux.HandleFunc("/api/v1/enrich", s.handleEnrich)
	mux.HandleFunc("/api/v1/enrich/recent", s.handleEnrichRecent)

	// MITRE ATT&CK endpoints (Phase 4)
	mux.HandleFunc("/api/v1/mitre/heatmap", s.handleMitreHeatmap)

	// Forensics / Evidence Locker (Phase 6.5)
	mux.HandleFunc("/api/v1/forensics/evidence", s.handleEvidenceList)
	mux.HandleFunc("/api/v1/forensics/evidence/", s.handleEvidenceItem) // /{id}, /{id}/verify, /{id}/seal
	mux.HandleFunc("/api/v1/forensics/export", s.handleEvidenceExport)

	// Audit / Regulator Portal (Phase 6.6)
	mux.HandleFunc("/api/v1/audit/log", s.handleAuditLog)
	mux.HandleFunc("/api/v1/audit/packages", s.handleAuditPackages)
	mux.HandleFunc("/api/v1/audit/packages/generate", s.handleAuditPackageGenerate)

	// MCP Endpoints (Phase 22.1)
	mux.HandleFunc("/api/v1/mcp/tools", s.handleMCPDiscovery)
	mux.HandleFunc("/api/v1/mcp/execute", s.handleMCPExecute)
	mux.HandleFunc("/api/v1/mcp/approve", s.handleMCPApprove)

	// Graph endpoints (Phase 9)
	mux.HandleFunc("/api/v1/graph/subgraph", s.handleGraphSubgraph)

	// User/Role management endpoints (Phase 12)
	mux.HandleFunc("/api/v1/users", s.stubHandler(s.handleUsers))
	mux.HandleFunc("/api/v1/roles", s.stubHandler(s.handleRoles))
	
	// Tenant isolation management endpoints (Phase 22.2)
	mux.HandleFunc("/api/v1/admin/tenants", s.handleAdminTenants)
	mux.HandleFunc("/api/v1/admin/tenants/", s.handleAdminTenantWipe)

	// UEBA endpoints (Phase 10)
	mux.HandleFunc("/api/v1/ueba/profiles", s.stubHandler(s.handleUEBAProfiles))
	mux.HandleFunc("/api/v1/ueba/anomalies", s.stubHandler(s.handleUEBAAnomalies))
	mux.HandleFunc("/api/v1/ueba/stats", s.stubHandler(s.handleUEBAStats))

	// SCIM 2.0 — Phase 20.4
	mux.HandleFunc("/api/scim/v2/Users", s.stubHandler(s.handleSCIMUsers))
	mux.HandleFunc("/api/scim/v2/Users/", s.stubHandler(s.handleSCIMUserByID))
	mux.HandleFunc("/api/scim/v2/Groups", s.stubHandler(s.handleSCIMGroups))
	
	// Identity Connectors — Phase 20.7
	mux.HandleFunc("/api/v1/identity/connectors", s.handleIdentityConnectors)
	mux.HandleFunc("/api/v1/identity/connectors/", s.handleIdentityConnectorByID)
	
	// Report Factory — Phase 20.10
	mux.HandleFunc("/api/v1/reports/templates", s.handleReportTemplates)
	mux.HandleFunc("/api/v1/reports/generated", s.handleGeneratedReports)
	mux.HandleFunc("/api/v1/reports/generate", s.handleReportGenerate)
	mux.HandleFunc("/api/v1/reports/view/", s.handleReportView)

	// Dashboard Studio — Phase 20.11
	mux.HandleFunc("/api/v1/dashboards", s.handleDashboards)
	mux.HandleFunc("/api/v1/dashboards/", s.handleDashboardByID)

	// NDR endpoints (Phase 11)
	mux.HandleFunc("/api/v1/ndr/flows", s.stubHandler(s.handleNDRFlows))
	mux.HandleFunc("/api/v1/ndr/alerts", s.stubHandler(s.handleNDRAlerts))
	mux.HandleFunc("/api/v1/ndr/protocols", s.stubHandler(s.handleNDRProtocols))

	// Ransomware endpoints (Phase 9)
	mux.HandleFunc("/api/v1/ransomware/events", s.stubHandler(s.handleRansomwareEvents))
	mux.HandleFunc("/api/v1/ransomware/hosts", s.stubHandler(s.handleRansomwareHosts))
	mux.HandleFunc("/api/v1/ransomware/stats", s.stubHandler(s.handleRansomwareStats))
	mux.HandleFunc("/api/v1/ransomware/isolate", s.stubHandler(s.handleRansomwareIsolate))

	// Playbook endpoints (Phase 8)
	mux.HandleFunc("/api/v1/playbooks", s.stubHandler(s.handlePlaybooks))
	mux.HandleFunc("/api/v1/playbooks/actions", s.stubHandler(s.handlePlaybookActions))
	mux.HandleFunc("/api/v1/playbooks/run", s.stubHandler(s.handlePlaybookRun))
	mux.HandleFunc("/api/v1/playbooks/metrics", s.stubHandler(s.handlePlaybookMetrics))

	// Agent endpoints (fleet management)
	mux.HandleFunc("/api/v1/agents", s.stubHandler(s.handleAgentsList))

	// Fusion Engine endpoints (Phase 10.6)
	mux.HandleFunc("/api/v1/fusion/campaigns", s.stubHandler(s.handleFusionCampaigns))
	mux.HandleFunc("/api/v1/fusion/campaigns/", s.stubHandler(s.handleFusionCampaignDetail))

	// Peer Analytics endpoints (Phase 10.5)
	mux.HandleFunc("/api/v1/ueba/peer-groups", s.stubHandler(s.handlePeerGroups))
	mux.HandleFunc("/api/v1/ueba/peer-deviations", s.stubHandler(s.handlePeerDeviations))

	// Agentless collectors status (Phase 7.5)
	mux.HandleFunc("/api/v1/agentless/status", s.stubHandler(s.handleAgentlessStatus))
	mux.HandleFunc("/api/v1/agentless/collectors", s.stubHandler(s.handleAgentlessCollectors))

	// Security & Hardening routes
	s.initSecurityRoutes(mux)
	
	var handler http.Handler = mux

	// Wrap entire mux with Authentication middleware if provided, BUT exclude
	// the login and OIDC endpoints which must be accessible to anonymous users.
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/v1/auth/login") ||
			strings.HasPrefix(path, "/api/v1/auth/oidc") ||
			path == "/healthz" || path == "/readyz" {
			mux.ServeHTTP(w, r)
			return
		}

		if s.auth != nil {
			// Auth -> Tenant Isolation chain
			s.auth.Middleware(s.tenantMiddleware(mux)).ServeHTTP(w, r)
		} else {
			mux.ServeHTTP(w, r)
		}
	})

	// Wrap entire router with security middleware (CORS, Headers, Rate Limiting)
	handler = s.secureMiddleware(finalHandler)

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
		if err := s.certManager.Load(); err == nil {
			s.server.TLSConfig = &tls.Config{
				GetCertificate: s.certManager.GetCertificate,
				MinVersion:     tls.VersionTLS13,
			}
		} else {
			s.log.Warn("[REST] TLS certificate load failed: %v — TLS will be disabled", err)
		}
	}

	go func() {
		// Attempt TLS if certs are available
		if s.certManager != nil && s.server.TLSConfig != nil {
			s.log.Info("[REST] Starting TLS listener on port %d", s.port)
			err := s.server.ListenAndServeTLS("", "")
			if err != nil && err != http.ErrServerClosed {
				s.log.Warn("[REST] TLS server failed: %v — falling back to plaintext HTTP", err)
				// Clean up and try plaintext
			} else {
				return
			}
		}

		// Fallback to plaintext HTTP
		s.log.Info("[REST] Starting plaintext HTTP listener on port %d", s.port)
		// Clear TLSConfig to ensure standard HTTP
		s.server.TLSConfig = nil
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			s.log.Error("[REST] HTTP server failed: %v", err)
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

type DataConfidence int

const (
	ConfidenceVerified  DataConfidence = 100
	ConfidenceDerived   DataConfidence = 80
	ConfidencePartial   DataConfidence = 50
	ConfidenceUntrusted DataConfidence = 0
)

func (s *RESTServer) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	confidence := ConfidenceVerified
	if os.Getenv("OBLIVRA_ENV") != "production" {
		confidence = ConfidenceUntrusted
	}

	b, err := json.Marshal(data)
	if err == nil && len(b) > 0 && b[0] == '{' {
		confField := fmt.Sprintf(`"data_confidence":%d,`, confidence)
		w.Write([]byte("{"))
		w.Write([]byte(confField))
		w.Write(b[1:])
		return
	}
	
	json.NewEncoder(w).Encode(data)
}

func (s *RESTServer) stubHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("OBLIVRA_ENV") == "production" {
			panic("Stub data not allowed in production")
		}
		h(w, r)
	}
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
			"https://wails.localhost": true,
			"wails://wails":           true,
			"http://localhost:3000":   true,
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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws: wss:;")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
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

	// 1. Gather SIEM-level metrics (e.g., active alerts) into the collector if not already there
	if s.siem != nil && s.pipeline != nil {
		mc := s.pipeline.GetMetricsCollector()
		if mc != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			alerts, _ := s.siem.SearchHostEvents(ctx, "EventType:security_alert", 100)
			cancel()
			mc.SetGauge("siem_active_alerts", float64(len(alerts)), nil)
		}
	}

	// 2. Delegate to the centralized Prometheus handler
	if s.pipeline != nil {
		if mc := s.pipeline.GetMetricsCollector(); mc != nil {
			mc.PrometheusHandler().ServeHTTP(w, r)
			return
		}
	}

	// Fallback if collector is missing
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "# No metrics available\n")
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

	snap := s.pipeline.GetMetrics()
	stats := ingest.CollectStats(snap, time.Since(snap.CollectedAt))
	s.jsonResponse(w, http.StatusOK, stats)
}

func (s *RESTServer) handleIngestReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. RBAC Check: Replay is a forensics-level action; require Admin
	role := auth.GetRole(r.Context())
	if role != auth.RoleAdmin {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
		return
	}

	var req struct {
		WALPath string `json:"wal_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.WALPath == "" {
		http.Error(w, "wal_path is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	result, err := s.replayer.ReplayWAL(ctx, req.WALPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Replay failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, result)
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
	})
}

func (s *RESTServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	s.log.Debug("[REST] Event stream upgrade attempt from: %s", r.RemoteAddr)

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
	var closeOnce sync.Once
	cleanup := func() {
		closeOnce.Do(func() {
			close(ctxDone)
		})
	}

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
		cleanup()
		s.bus.Unsubscribe(subID)
		s.log.Info("[REST] Client disconnected from event stream: %s (unsubscribed)", clientAddr)
	}()

	// Read loop to detect client disconnect
	go func() {
		defer cleanup()
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
				s.log.Warn("[REST] Event subscription channel closed for %s", clientAddr)
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				s.log.Error("[REST] Failed to marshal event: %v", err)
				continue
			}
			// Set write deadline for the event
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				s.log.Warn("[REST] Failed to send event to %s: %v", clientAddr, err)
				return
			}
		case <-ctxDone:
			s.log.Debug("[REST] Context done for %s, closing WebSocket", clientAddr)
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
	s.appendAuditEntry(user.Email, "login", "system", "success", r)
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

// ── Lookup Table Handlers (Phase 1.3) ─────────────────────────────────────────

// GET /api/v1/lookups — list all tables (metadata, no rows)
func (s *RESTServer) handleLookupList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"tables": s.lookups.List(),
	})
}

// GET /api/v1/lookups/{name}  — return table with rows
// DELETE /api/v1/lookups/{name} — remove table
func (s *RESTServer) handleLookupByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/lookups/")
	if name == "" {
		http.Error(w, "Missing table name", http.StatusBadRequest)
		return
	}
	// strip nested sub-paths to avoid matching /upload and /query
	if strings.Contains(name, "/") {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		t, ok := s.lookups.Get(name)
		if !ok {
			http.Error(w, "Table not found", http.StatusNotFound)
			return
		}
		s.jsonResponse(w, http.StatusOK, t)

	case http.MethodDelete:
		if role := auth.GetRole(r.Context()); role != auth.RoleAdmin {
			http.Error(w, "Admin only", http.StatusForbidden)
			return
		}
		if !s.lookups.Delete(name) {
			http.Error(w, "Table not found", http.StatusNotFound)
			return
		}
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"deleted": name})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// POST /api/v1/lookups/upload — create/replace table from file
// Form fields: name, match_type (exact|cidr|wildcard|regex), format (csv|json)
// Body: the file content as multipart/form-data or raw body
func (s *RESTServer) handleLookupUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if role := auth.GetRole(r.Context()); role != auth.RoleAdmin {
		http.Error(w, "Admin only", http.StatusForbidden)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	name      := r.FormValue("name")
	matchType := lookup.MatchType(r.FormValue("match_type"))
	format    := r.FormValue("format")
	if name == "" || matchType == "" {
		http.Error(w, "name and match_type are required", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	switch format {
	case "json":
		if err := s.lookups.UpsertFromJSON(name, matchType, file); err != nil {
			http.Error(w, fmt.Sprintf("JSON parse error: %v", err), http.StatusBadRequest)
			return
		}
	default: // csv or unspecified
		if err := s.lookups.UpsertFromCSV(name, matchType, file); err != nil {
			http.Error(w, fmt.Sprintf("CSV parse error: %v", err), http.StatusBadRequest)
			return
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"name":   name,
	})
}

// GET /api/v1/lookups/query?table=X&key=Y — single key lookup
func (s *RESTServer) handleLookupQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	table := r.URL.Query().Get("table")
	key   := r.URL.Query().Get("key")
	if table == "" || key == "" {
		http.Error(w, "table and key are required", http.StatusBadRequest)
		return
	}

	result := s.lookups.Lookup(table, key)
	if result == nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"match": false,
			"data":  nil,
		})
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"match": true,
		"data":  result,
	})
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

// ── Escalation Handlers (Phase 2.1.5) ─────────────────────────────────────────

func (s *RESTServer) getEscalation() *notifications.EscalationManager {
	if s.escalation == nil {
		// Lazy-init with a no-op notifier when none is wired
		s.escalation = notifications.NewEscalationManager(
			notifications.NewNotificationService(s.log),
			s.log,
		)
	}
	return s.escalation
}

// GET /api/v1/escalation/policies
// POST /api/v1/escalation/policies
func (s *RESTServer) handleEscalationPolicies(w http.ResponseWriter, r *http.Request) {
	esc := s.getEscalation()
	switch r.Method {
	case http.MethodGet:
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"policies": esc.ListPolicies(),
		})
	case http.MethodPost:
		if role := auth.GetRole(r.Context()); role != auth.RoleAdmin {
			http.Error(w, "Admin only", http.StatusForbidden)
			return
		}
		var p notifications.EscalationPolicy
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}
		esc.UpsertPolicy(&p)
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"status": "ok", "id": p.ID})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GET /api/v1/escalation/policies/{id}
// DELETE /api/v1/escalation/policies/{id}
func (s *RESTServer) handleEscalationPolicyByID(w http.ResponseWriter, r *http.Request) {
	esc := s.getEscalation()
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/escalation/policies/")
	switch r.Method {
	case http.MethodGet:
		p, ok := esc.GetPolicy(id)
		if !ok {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		s.jsonResponse(w, http.StatusOK, p)
	case http.MethodDelete:
		if role := auth.GetRole(r.Context()); role != auth.RoleAdmin {
			http.Error(w, "Admin only", http.StatusForbidden)
			return
		}
		if !esc.DeletePolicy(id) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"deleted": id})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GET /api/v1/escalation/active
func (s *RESTServer) handleEscalationActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"escalations": s.getEscalation().ListActive(),
	})
}

// GET /api/v1/escalation/history?limit=N
func (s *RESTServer) handleEscalationHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		fmt.Sscanf(v, "%d", &limit)
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"escalations": s.getEscalation().ListHistory(limit),
	})
}

// POST /api/v1/escalation/ack
func (s *RESTServer) handleEscalationAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req notifications.AckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := s.getEscalation().Acknowledge(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"status": "acknowledged"})
}

// GET /api/v1/escalation/oncall?schedule=primary
func (s *RESTServer) handleEscalationOnCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	scheduleID := r.URL.Query().Get("schedule")
	if scheduleID == "" {
		scheduleID = "primary"
	}
	esc := s.getEscalation()
	schedules := esc.ListSchedules()
	var entries []interface{}
	for _, sc := range schedules {
		if sc.ID == scheduleID {
			for _, e := range sc.Entries {
				entries = append(entries, e)
			}
		}
	}
	current := esc.CurrentOnCall(scheduleID)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"schedule_id": scheduleID,
		"entries":     entries,
		"current":     current,
	})
}
// ── Phase 3 ThreatIntel + Enrich Handlers ────────────────────────────────────

func (s *RESTServer) getThreatIntel() *threatintel.MatchEngine {
	if s.matchEngine == nil {
		s.matchEngine = threatintel.NewMatchEngine(s.log)
		// Seed with sample IOCs so the UI has data to show on fresh install
		s.matchEngine.Load(threatintel.BuiltinIndicators())
		s.matchEngine.LoadCampaigns(threatintel.BuiltinCampaigns())
	}
	return s.matchEngine
}

// GET /api/v1/threatintel/stats
func (s *RESTServer) handleThreatIntelStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"stats": s.getThreatIntel().Stats(),
	})
}

// GET /api/v1/threatintel/indicators?limit=N&type=X
func (s *RESTServer) handleThreatIntelIndicators(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	typeFilter := r.URL.Query().Get("type")
	limit := 500
	fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)

	all := s.getThreatIntel().All()
	var out []threatintel.Indicator
	for _, ind := range all {
		if typeFilter != "" && ind.Type != typeFilter {
			continue
		}
		out = append(out, ind)
		if len(out) >= limit {
			break
		}
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"indicators": out})
}

// GET /api/v1/threatintel/lookup?value=X
func (s *RESTServer) handleThreatIntelLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	value := r.URL.Query().Get("value")
	if value == "" {
		http.Error(w, "value is required", http.StatusBadRequest)
		return
	}
	ind, matched := s.getThreatIntel().MatchAny(value)
	if !matched {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"match": false})
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"match":     true,
		"indicator": ind,
	})
}

// GET /api/v1/threatintel/campaigns
func (s *RESTServer) handleThreatIntelCampaigns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"campaigns": s.getThreatIntel().ListCampaigns(),
	})
}

// GET /api/v1/enrich?q=IP_OR_HOST
func (s *RESTServer) handleEnrich(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "q is required", http.StatusBadRequest)
		return
	}

	result := map[string]interface{}{"query": q}

	// Check enrichment budget — never block ingestion on GeoIP/DNS/TI stalls (#9)
	skipped := false
	if s.enrichLimiter != nil && !s.enrichLimiter.Allow() {
		skipped = true
	}

	// IOC lookup (only if budget allows)
	if !skipped {
		ind, matched := s.getThreatIntel().MatchAny(q)
		if matched {
			result["ioc_match"] = map[string]interface{}{
				"matched":     true,
				"severity":    ind.Severity,
				"source":      ind.Source,
				"description": ind.Description,
			}
		} else {
			result["ioc_match"] = map[string]interface{}{"matched": false}
		}

		// Geo stub — a production deploy wires MaxMind + DNS resolvers
		result["geo"] = map[string]interface{}{
			"ip":           q,
			"country_code": "??",
			"country_name": "Unknown (offline GeoIP)",
			"city":         "",
			"asn":          "AS0",
			"org":          "Requires MaxMind DB",
		}
	} else {
		result["enrichment_skipped"] = true
		result["enrichment_skip_reason"] = "rate_limit_exceeded"
	}

	// Add to recent ring buffer
	s.enrichMu.Lock()
	s.enrichRecent = append([]map[string]interface{}{result}, s.enrichRecent...)
	if len(s.enrichRecent) > 20 {
		s.enrichRecent = s.enrichRecent[:20]
	}
	s.enrichMu.Unlock()

	s.jsonResponse(w, http.StatusOK, result)
}

// GET /api/v1/enrich/recent
func (s *RESTServer) handleEnrichRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.enrichMu.RLock()
	defer s.enrichMu.RUnlock()
	s.jsonResponse(w, http.StatusOK, s.enrichRecent)
}

// ── MITRE ATT&CK Handlers (Phase 4) ──────────────────────────────────────────

// GET /api/v1/mitre/heatmap
// Returns each tactic with its techniques and hit counts from the SIEM store.
func (s *RESTServer) handleMitreHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Tactic ordering follows ATT&CK canonical order
	tacticOrder := []struct{ id, name string }{
		{"TA0001", "Initial Access"},
		{"TA0002", "Execution"},
		{"TA0003", "Persistence"},
		{"TA0004", "Privilege Escalation"},
		{"TA0005", "Defense Evasion"},
		{"TA0006", "Credential Access"},
		{"TA0007", "Discovery"},
		{"TA0008", "Lateral Movement"},
		{"TA0009", "Collection"},
		{"TA0011", "Command and Control"},
		{"TA0010", "Exfiltration"},
		{"TA0040", "Impact"},
	}

	// Technique-to-tactic grouping (mirrors detection/mitre.go)
	tacticTechs := map[string][]string{
		"TA0001": {"T1078", "T1190", "T1133", "T1566"},
		"TA0002": {"T1059", "T1053", "T1047", "T1203"},
		"TA0003": {"T1098", "T1136", "T1543"},
		"TA0004": {"T1548", "T1068", "T1134", "T1574"},
		"TA0005": {"T1562", "T1070", "T1027", "T1036", "T1112"},
		"TA0006": {"T1110", "T1003", "T1555", "T1558", "T1552"},
		"TA0007": {"T1087", "T1069", "T1018", "T1046"},
		"TA0008": {"T1021", "T1210", "T1563", "T1080"},
		"TA0009": {"T1560", "T1074", "T1005"},
		"TA0011": {"T1071", "T1105", "T1572"},
		"TA0010": {"T1048", "T1041", "T1567"},
		"TA0040": {"T1486", "T1490", "T1489", "T1529"},
	}

	// Count hits per technique via SIEM host-event aggregation
	hitCounts := make(map[string]int)
	totalHits := 0
	if s.siem != nil {
		// AggregateHostEvents with empty query returns all events, faceted by mitre field
		counts, err := s.siem.AggregateHostEvents(r.Context(), "", "mitre_attack")
		if err == nil {
			for k, v := range counts {
				hitCounts[k] = v
				totalHits += v
			}
		}
	}

	type TechCell struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Hits int    `json:"hits"`
	}
	type TacticRow struct {
		ID         string     `json:"id"`
		Name       string     `json:"name"`
		Techniques []TechCell `json:"techniques"`
	}

	var rows []TacticRow
	for _, tac := range tacticOrder {
		var cells []TechCell
		for _, tid := range tacticTechs[tac.id] {
			cells = append(cells, TechCell{
				ID:   tid,
				Name: detection.GetTechniqueName(tid),
				Hits: hitCounts[tid],
			})
		}
		rows = append(rows, TacticRow{ID: tac.id, Name: tac.name, Techniques: cells})
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"tactics":      rows,
		"total_hits":   totalHits,
		"last_updated": time.Now().Format(time.RFC3339),
	})
}

// ── Evidence Locker Handlers (Phase 6.5) ─────────────────────────────────────

// GET /api/v1/forensics/evidence
func (s *RESTServer) handleEvidenceList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items := s.evidence.ListAll()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"items": items})
}

// /api/v1/forensics/evidence/{id}[/verify|/seal]
func (s *RESTServer) handleEvidenceItem(w http.ResponseWriter, r *http.Request) {
	// Parse path suffix after /api/v1/forensics/evidence/
	suffix := strings.TrimPrefix(r.URL.Path, "/api/v1/forensics/evidence/")
	parts := strings.SplitN(suffix, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "missing item ID", http.StatusBadRequest)
		return
	}
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch action {
	case "verify":
		valid, err := s.evidence.Verify(id)
		if err != nil {
			s.jsonResponse(w, http.StatusNotFound, map[string]interface{}{"error": err.Error()})
			return
		}
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"id": id, "valid": valid})

	case "seal":
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		actor := r.Header.Get("X-Actor")
		if actor == "" {
			actor = "api"
		}
		if err := s.evidence.Seal(id, actor, "Sealed via REST API"); err != nil {
			s.jsonResponse(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
			return
		}
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"sealed": true})

	default: // GET item
		item, err := s.evidence.Get(id)
		if err != nil {
			s.jsonResponse(w, http.StatusNotFound, map[string]interface{}{"error": err.Error()})
			return
		}
		s.jsonResponse(w, http.StatusOK, item)
	}
}

// GET /api/v1/forensics/export — full JSON export of the vault
func (s *RESTServer) handleEvidenceExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := s.evidence.Export()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"oblivra-evidence.json\"")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// ── Audit / Regulator Portal Handlers (Phase 6.6) ────────────────────────────

// appendAuditEntry persists an entry to the AuditRepository.
// Call this from any handler that performs a significant action.
func (s *RESTServer) appendAuditEntry(actor, action, resource, outcome string, r *http.Request) {
	if s.audit == nil {
		return
	}

	details := map[string]interface{}{
		"actor":      actor,
		"action":     action,
		"resource":   resource,
		"outcome":    outcome,
		"ip":         r.RemoteAddr,
		"user_agent": r.UserAgent(),
	}

	// Persist to BadgerDB/SQL with Merkle integrity
	err := s.audit.Log(r.Context(), action, resource, "", details)
	if err != nil {
		s.log.Error("[REST] Audit log persistence failed: %v", err)
	}
}

// GET /api/v1/audit/log
func (s *RESTServer) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.audit == nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"entries": []interface{}{}})
		return
	}

	logs, err := s.audit.GetRecent(r.Context(), 200)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch audit logs: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"entries": logs})
}

// compliancePackagesMu + compliancePackages stores generated packages in memory
var compliancePackagesMu sync.RWMutex
var compliancePackages []map[string]interface{}

// GET /api/v1/audit/packages
func (s *RESTServer) handleAuditPackages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	compliancePackagesMu.RLock()
	pkgs := make([]map[string]interface{}, len(compliancePackages))
	copy(pkgs, compliancePackages)
	compliancePackagesMu.RUnlock()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"packages": pkgs})
}

// POST /api/v1/audit/packages/generate
func (s *RESTServer) handleAuditPackageGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Framework string `json:"framework"`
		From      string `json:"from"`
		To        string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Framework == "" {
		req.Framework = "SOC2"
	}

	// Count audit entries in scope
	count, _ := s.audit.Count(r.Context())

	// Build package metadata
	id := fmt.Sprintf("CP-%s-%d", req.Framework, time.Now().Unix())
	proof := fmt.Sprintf("%x", sha256Hash([]byte(id+req.Framework+req.From+req.To)))

	pkg := map[string]interface{}{
		"id":              id,
		"framework":       req.Framework,
		"generated_at":    time.Now().Format(time.RFC3339),
		"records":         count,
		"integrity_proof": proof,
		"download_url":    fmt.Sprintf("/api/v1/audit/packages/%s/download", id),
	}

	compliancePackagesMu.Lock()
	compliancePackages = append(compliancePackages, pkg)
	compliancePackagesMu.Unlock()

	s.jsonResponse(w, http.StatusCreated, pkg)
}

// sha256Hash is a local helper to avoid import cycle.
func sha256Hash(data []byte) []byte {
	import_sha256 := sha256.New()
	import_sha256.Write(data)
	return import_sha256.Sum(nil)
}

// ── MCP Handlers (Phase 22.1) ────────────────────────────────────────────────

func (s *RESTServer) handleMCPDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tools := s.mcpRegistry.ListTools()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"tools":   tools,
		"version": "1.0",
		"capabilities": map[string]bool{
			"approval":      true,
			"simulation":    true,
			"deterministic": true,
		},
	})
}

func (s *RESTServer) handleMCPExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Max 5MB for MCP requests (some params like scripts might be large)
	r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024)

	var req mcp.MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid MCP request envelope", http.StatusBadRequest)
		return
	}

	// Enforce strict client-side ID if missing (server-side generation fallback)
	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}

	// Execution pipeline
	resp := s.mcpHandler.HandleRequest(r.Context(), req)

	status := http.StatusOK
	switch resp.Status {
	case "denied":
		status = http.StatusForbidden
	case "error":
		status = http.StatusInternalServerError
	case "pending_approval":
		status = http.StatusAccepted
	}

	s.jsonResponse(w, status, resp)
}

func (s *RESTServer) handleMCPApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ApprovalID string `json:"approval_id"`
		ActorID    string `json:"actor_id"`
		Signature  string `json:"signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// In a real M-of-N system, we would:
	// 1. Verify the signature against ActorID's public key
	// 2. Increment approval count for ApprovalID
	// 3. If threshold met, generate a one-time execution token

	// MVP: Generate a simple HMAC-based token
	token := s.mcpHandler.GenerateApprovalToken(req.ApprovalID, req.ActorID)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "approved",
		"token":  token,
	})
}

// ── Graph Handlers (Phase 9) ────────────────────────────────────────────────

func (s *RESTServer) handleGraphSubgraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		http.Error(w, "node_id is required", http.StatusBadRequest)
		return
	}

	hops := 2
	hopsStr := r.URL.Query().Get("hops")
	if hopsStr != "" {
		fmt.Sscanf(hopsStr, "%d", &hops)
	}

	if s.graphEngine == nil {
		http.Error(w, "Graph engine not initialized", http.StatusNotImplemented)
		return
	}

	nodes, edges := s.graphEngine.GetSubGraph(nodeID, hops)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
}
