package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/auth"
	// Phase 36.x: compliance package removed.
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
	"github.com/kingknull/oblivrashell/internal/licensing"
	"github.com/kingknull/oblivrashell/internal/platform"
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

	// BootstrapAdmin creates the very first admin account during initial setup.
	// Refuses to run if any users already exist. Phase 22.5.
	BootstrapAdmin(ctx context.Context, email, name, password string) (*database.User, error)

	// Connector management (Phase 20.7)
	ListConnectors(ctx context.Context) ([]database.IdentityConnector, error)
	CreateConnector(ctx context.Context, c *database.IdentityConnector) error
	GetConnector(ctx context.Context, id string) (*database.IdentityConnector, error)
	UpdateConnector(ctx context.Context, c *database.IdentityConnector) error
	DeleteConnector(ctx context.Context, id string) error
	TriggerSync(ctx context.Context, id string) error

	// Role management
	ListRoles(ctx context.Context) ([]database.Role, error)
}

type PlatformProvider interface {
	GetMetrics(ctx context.Context) (any, error)
}

type ForensicsProvider interface {
	ListEvidence(ctx context.Context, incidentID string) []*forensics.EvidenceItem
}

// ComplianceProvider — removed in Phase 36.x with the compliance packs.

type FusionProvider interface {
	GetActiveCampaigns() []detection.Campaign
	GetCampaignTimeline(ctx context.Context, entityID string) (*detection.CampaignTimeline, error)
}

// SuppressionProvider exposes the alert-suppression mutation surface
// without forcing the api package to import services (which would
// create an import cycle — services already depends on api). The
// container.go is the only seam where the concrete
// `*services.SuppressionService` is plugged in via SetSuppression.
type SuppressionProvider interface {
	CreateRule(ctx context.Context, rule *database.SuppressionRule) (string, error)
	ListRules(ctx context.Context) ([]database.SuppressionRule, error)
}

// SettingsProvider exposes key/value settings persistence — same
// interface-vs-cycle pattern as SuppressionProvider. Sensitive keys
// (anything matching password / token / secret / webhook /
// credential / private_key) are vault-encrypted at rest by the
// concrete service; this interface is plaintext-in / plaintext-out
// from the caller's perspective.
type SettingsProvider interface {
	Get(key string) (string, error)
	Set(key string, value string) error
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
	db       database.DatabaseStore // relational DB (settings, agent_registry, compliance_packages, login_lockouts)
	siem     database.SIEMStore
	pipeline ingest.IngestionPipeline
	auth     *auth.APIKeyMiddleware
	identity     IdentityProvider
	platformProvider PlatformProvider
	forensicsProvider ForensicsProvider
	fusion       FusionProvider
	// compliance ComplianceProvider — removed Phase 36.x
	bus          *eventbus.Bus
	log      *logger.Logger
	attest   *attestation.AttestationService
	certManager *security.CertificateManager
	lookups  *lookup.Manager
	escalation *notifications.EscalationManager
	matchEngine *threatintel.MatchEngine
	agents   map[string]*AgentInfo // registered agent fleet
	agentsMu sync.RWMutex           // protects agents map

	// Phase 31 — operator remote-control RPC queue. Holds per-agent
	// pending actions (Trigger Scan / Toggle Debug / Restart Agent
	// + the legacy response-action types) until the agent picks
	// them up on its next heartbeat. See agent_actions_queue.go.
	actionsQueue *agentActionsQueue
	agentProvider AgentProvider     // provider for agent fleet data
	globalLimiter  *rate.Limiter
	ipLimiters     sync.Map // IP -> *rate.Limiter
	failedLogins   sync.Map     // email -> {count, lockoutUntil}
	failedLoginsMu sync.Mutex   // serialises check-then-increment on failedLogins (TOCTOU fix)
	tenantLimiters sync.Map // map[string]*rate.Limiter
	upgrader websocket.Upgrader

	// Enrichment recent-query ring buffer
	enrichMu     sync.RWMutex
	enrichRecent map[string][]map[string]interface{}

	// TLS state
	isTLS bool
	isTLSMu sync.RWMutex

	// Phase 6.5 — Evidence Locker
	evidence *forensics.EvidenceLocker
	audit      *database.AuditRepository

	// Phase 32 — Suppression provider (set via SetSuppression after
	// construction to avoid an import cycle with internal/services).
	suppression SuppressionProvider
	// Phase 32 — Settings provider, same interface-vs-cycle dance.
	settings SettingsProvider
	// Slice 3 — TLS guardrail state for the chrome banner.
	tlsState TLSStateProvider
	// Slice 5 — I/O config provider (read/write the YAML config + stats).
	ioConfig IOConfigProvider

	// Phase 22.3 — Hot/Warm/Cold storage tier providers. Set via
	// SetTieringProvider after construction (interface-vs-cycle pattern).
	// nil means tiering is not configured — the relevant routes return 503.
	tierStats []TierStatProvider
	tierMig   TierMigrationProvider

	// Audit fix #1 — replay-attack cache for agent endpoints. HMAC +
	// 30s timestamp window doesn't prevent replay within the window;
	// this cache rejects (agent_id, ts, body) fingerprints we've seen.
	replayCache *ReplayCache

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
	license       licensing.Provider
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
	// LastAckedSeq is the highest agent-assigned sequence number this server
	// has durably accepted. Returned to the agent on every ingest so it can
	// truncate its WAL up to that point; events with Seq <= LastAckedSeq are
	// silently dropped as duplicate replays. Phase 22.1 reconnect guarantee.
	LastAckedSeq uint64 `json:"last_acked_seq"`
}

// SearchRequest defines the JSON body for SIEM search endpoints
type SearchRequest struct {
	Query   string                 `json:"query"`
	Filters map[string]interface{} `json:"filters"`
}

// NewRESTServer configures the HTTP router and middleware
// NewRESTServer — Phase 36.x: dropped `compliance ComplianceProvider`
// argument with the compliance pack deletion.
func NewRESTServer(port int, db database.DatabaseStore, siem database.SIEMStore, audit *database.AuditRepository, pipeline ingest.IngestionPipeline, graphEngine *graph.GraphEngine, ueba UEBAProvider, agentProvider AgentProvider, fleetSecret []byte, keyProvider SystemKeyProvider, license licensing.Provider, attest *attestation.AttestationService, authMw *auth.APIKeyMiddleware, identity IdentityProvider, platformProvider PlatformProvider, forensicsProvider ForensicsProvider, fusion FusionProvider, reports ReportingProvider, dashboards DashboardProvider, bus *eventbus.Bus, certManager *security.CertificateManager, log *logger.Logger, mcpRegistry *mcp.ToolRegistry, mcpHandler *mcp.Handler) *RESTServer {
	var tenantRepo *database.TenantRepository
	if db != nil {
		tenantRepo = database.NewTenantRepository(db)
	}

	s := &RESTServer{
		port:     port,
		db:       db,
		siem:     siem,
		audit:    audit,
		pipeline: pipeline,
		lookups:  lookup.NewManager(),
		auth:     authMw,
		identity: identity,
		platformProvider: platformProvider,
		forensicsProvider: forensicsProvider,
		fusion:   fusion,
		// Phase 36.x: compliance removed.
		bus:      bus,
		log:      log,
		attest:   attest,
		certManager: certManager,
		agents:       make(map[string]*AgentInfo),
		actionsQueue: newAgentActionsQueue(),
		enrichRecent: make(map[string][]map[string]interface{}),
		globalLimiter:  rate.NewLimiter(rate.Limit(200), 500), // global higher limit
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
		license:       license,
		upgrader: websocket.Upgrader{
			// Restrict WebSocket upgrades to same-origin and explicitly allowed origins.
			// Do NOT allow all origins — any web page could connect and receive live event data.
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					// SEC-16: Reject empty-origin to prevent CSRF / spoofing bypass
					return false
				}
				// Allow same-host requests (Wails desktop shell and localhost agents)
				host := r.Host
				allowed := []string{
					"http://" + host,
					"https://" + host,
					"wails://wails",
				}

				// Allow localhost:3000 only in development/debug mode
				if os.Getenv("OBLIVRA_DEBUG") == "true" {
					allowed = append(allowed, "http://localhost:3000", "https://localhost:3000")
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
	// Next-Best-Action recommender (Phase 32) — returns the
	// pre-highlighted action the triage drawer should show.
	mux.HandleFunc("/api/v1/alerts/recommend", s.handleAlertRecommend)
	// Per-alert suppression (Phase 32 follow-up): operator's `x`
	// keystroke promotes a transient localStorage suppression to a
	// durable SuppressionRule. Path: /api/v1/alerts/{id}/suppress.
	mux.HandleFunc("/api/v1/alerts/", s.handleAlertSubresource)

	// Authentication endpoints
	mux.HandleFunc("/api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("/api/v1/auth/logout", s.handleLogout)
	mux.HandleFunc("/api/v1/auth/refresh", s.handleRefresh)
	mux.HandleFunc("/api/v1/auth/me", s.handleMe)
	mux.HandleFunc("/api/v1/auth/oidc/login", s.handleOIDCLogin)
	mux.HandleFunc("/api/v1/auth/oidc/callback", s.handleOIDCCallback)
	mux.HandleFunc("/api/v1/auth/saml/login", s.handleSAMLLogin)
	mux.HandleFunc("/api/v1/auth/saml/callback", s.handleSAMLCallback)
	mux.HandleFunc("/api/v1/auth/saml/metadata", s.handleSAMLMetadata)

	// Identity connector CRUD (admin-only) — browser-mode mirror of
	// the Wails-bound IdentityService methods. Desktop frontend calls
	// IdentityService directly; web frontend uses these REST endpoints.
	mux.HandleFunc("/api/v1/identity/connectors", s.handleConnectorsCollection)
	mux.HandleFunc("/api/v1/identity/connectors/", s.handleConnectorByID)

	// Identity endpoints
	mux.HandleFunc("/api/v1/identities", s.handleIdentitiesList)
	mux.HandleFunc("/api/v1/identities/roles", s.handleRolesList)

	// Platform endpoints
	mux.HandleFunc("/api/v1/platform/metrics", s.handlePlatformMetrics)

	// Fusion endpoints
	mux.HandleFunc("/api/v1/fusion/campaigns", s.handleCampaignList)
	mux.HandleFunc("/api/v1/fusion/campaigns/", s.stubHandler(s.handleFusionCampaignDetail))
	mux.HandleFunc("/api/v1/fusion/timeline", s.handleCampaignTimeline)

	// Events endpoint
	mux.HandleFunc("/api/v1/events", s.handleEvents)

	// OpenAPI endpoints
	mux.HandleFunc("/api/v1/openapi.yaml", s.handleOpenAPI)
	mux.HandleFunc("/api/v1/docs", s.handleDocs)

	// System endpoints
	mux.HandleFunc("/api/v1/ingest/status", s.handleIngestStatus)
	mux.HandleFunc("/api/v1/ingest/replay", s.handleIngestReplay)
	// Lightweight pipeline-load probe for the frontend DEGRADED banner.
	// Returns just status + the two numbers needed to render the banner —
	// safe to poll every 10s without flooding the heavier /ingest/status path.
	mux.HandleFunc("/api/v1/health/load", s.handleHealthLoad)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/debug/attestation", s.handleAttestation)
	mux.HandleFunc("/api/v1/setup/initialize", s.handleSetupInitialize)

	// Agent endpoints
	mux.HandleFunc("/api/v1/agent/ingest", s.handleAgentIngest)
	mux.HandleFunc("/api/v1/agent/register", s.handleAgentRegister)
	mux.HandleFunc("/api/v1/agent/fleet", s.handleAgentFleet)
	// Tamper-evidence endpoints (Path 1, Layers 1 + 3 + 5).
	mux.HandleFunc("/api/v1/agent/oplog", s.handleAgentOplog)
	mux.HandleFunc("/api/v1/agent/heartbeat", s.handleAgentHeartbeat)
	mux.HandleFunc("/api/v1/integrity", s.handleIntegrity)

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
	// Phase 32: bulk seal — seals every unsealed item (optionally
	// scoped to an incident_id). Powers the Crisis Decision Panel's
	// "Seal evidence now" button.
	mux.HandleFunc("/api/v1/evidence/seal", s.handleEvidenceBulkSeal)
	// Crisis-lifecycle audit ping. Frontend's crisisStore POSTs here on
	// arm and stand-down so the destructive-action audit trail captures
	// every state transition. See rest_crisis_audit.go.
	mux.HandleFunc("/api/v1/crisis/state", s.handleCrisisState)

	// Settings (Phase 32) — key/value persistence. Plain key path
	// dispatches to per-key get/set; `_export` returns a redacted
	// dump. Sensitive keys are vault-encrypted at rest by the concrete
	// SettingsService; the REST surface deals in plaintext.
	mux.HandleFunc("/api/v1/settings/", s.handleSettingsRoot)

	// Audit / Regulator Portal (Phase 6.6)
	mux.HandleFunc("/api/v1/audit/log", s.handleAuditLog)
	mux.HandleFunc("/api/v1/audit/packages", s.handleAuditPackages)
	mux.HandleFunc("/api/v1/audit/packages/generate", s.handleAuditPackageGenerate)

	// GDPR / CCPA Data Subject Request workflow.
	mux.HandleFunc("/api/v1/dsr/requests", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.handleDSRList(w, r)
		case http.MethodPost:
			s.handleDSRCreate(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	// `/fulfill` and `/reject` use suffix dispatch on the path; no
	// separate endpoint constants because Go's net/http mux requires
	// one PathFunc per literal prefix.
	mux.HandleFunc("/api/v1/dsr/requests/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/fulfill"):
			s.handleDSRFulfill(w, r)
		case strings.HasSuffix(r.URL.Path, "/reject"):
			s.handleDSRReject(w, r)
		default:
			http.Error(w, "Use /fulfill or /reject", http.StatusBadRequest)
		}
	})

	// MCP Endpoints (Phase 22.1)
	mux.HandleFunc("/api/v1/mcp/tools", s.handleMCPDiscovery)
	mux.HandleFunc("/api/v1/mcp/execute", s.handleMCPExecute)
	mux.HandleFunc("/api/v1/mcp/approve", s.handleMCPApprove)

	// Graph endpoints (Phase 9)
	mux.HandleFunc("/api/v1/graph/subgraph", s.handleGraphSubgraph)
	mux.HandleFunc("/api/v1/graph/metrics", s.handleGraphMetrics)
	mux.HandleFunc("/api/v1/graph/full", s.handleGraphFull)

	// Sovereignty score (Phase 32) — single-number deployment posture
	// for the chrome badge + executive dash tile.
	mux.HandleFunc("/api/v1/sovereignty/score", s.handleSovereigntyScore)

	// TLS state (Slice 3) — anon-readable; drives the plaintext
	// banner in the chrome. The auth-bypass list below MUST include
	// this path or the banner can't render before login.
	mux.HandleFunc("/api/v1/tls/state", s.handleTLSState)

	// I/O config (Slice 5) — admin CRUD over the inputs/outputs YAML
	// + pipeline stats for the /connectors UI.
	mux.HandleFunc("/api/v1/io/config", s.handleIOConfig)
	mux.HandleFunc("/api/v1/io/stats",  s.handleIOStats)
	mux.HandleFunc("/api/v1/io/test",   s.handleIOTest)

	// Phase 22.3 — Hot/Warm/Cold storage tiering observability.
	// Stats is read-only (analyst+), promote is admin-only (manual cycle).
	mux.HandleFunc("/api/v1/storage/tiering/stats",   s.handleTieringStats)
	mux.HandleFunc("/api/v1/storage/tiering/promote", s.handleTieringPromote)

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
	
	// (Identity Connectors are registered earlier — see line ~341. The
	// Phase 20.7 stubs were superseded by the admin-gated, audit-logged
	// handlers in rest_connectors.go and removed to avoid pattern conflict.)

	// Report Factory — Phase 20.10
	mux.HandleFunc("/api/v1/reports/templates", s.handleReportTemplates)
	mux.HandleFunc("/api/v1/reports/generated", s.handleGeneratedReports)
	mux.HandleFunc("/api/v1/reports/generate", s.handleReportGenerate)
	mux.HandleFunc("/api/v1/reports/view/", s.handleReportView)

	// Phase 36.x: /api/v1/compliance/status route + handler removed.
	// Frontend caller (compliance.svelte.ts store) was deleted in Phase 36.9.

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
	// Phase 36: /api/v1/ransomware/isolate removed (response action).
	// Detection-side ransomware events/hosts/stats endpoints stay.

	// Phase 36: /api/v1/playbooks/* removed (SOAR scope cut).
	// Pair OBLIVRA detection events with external SOAR (Tines/XSOAR/Shuffle) instead.

	// Agent endpoints (fleet management)
	mux.HandleFunc("/api/v1/agents", s.stubHandler(s.handleAgentsList))

	// Peer Analytics endpoints (Phase 10.5)
	mux.HandleFunc("/api/v1/ueba/peer-groups", s.stubHandler(s.handlePeerGroups))
	mux.HandleFunc("/api/v1/ueba/peer-deviations", s.stubHandler(s.handlePeerDeviations))

	// Agentless collectors status (Phase 7.5)
	mux.HandleFunc("/api/v1/agentless/status", s.stubHandler(s.handleAgentlessStatus))
	mux.HandleFunc("/api/v1/agentless/collectors", s.stubHandler(s.handleAgentlessCollectors))

	// Security & Hardening routes
	s.initSecurityRoutes(mux)
	var handler http.Handler = mux

	// Wrap entire mux with Authentication middleware if provided, BUT exclude:
	//   - /auth/login, /auth/oidc, /auth/refresh - anonymous user flows.
	//   - /healthz, /readyz                       - kube-style probes.
	//   - /agent/register, /agent/ingest -
	//       agent endpoints authenticate via HMAC fleet-secret + timestamp
	//       inside the handler (VerifyHMAC). Routing them through
	//       APIKeyMiddleware would force agents to send a session token they
	//       cannot have, since they pre-date any user session.
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/v1/auth/login") ||
			strings.HasPrefix(path, "/api/v1/auth/oidc") ||
			strings.HasPrefix(path, "/api/v1/auth/refresh") ||
			strings.HasPrefix(path, "/api/v1/agent/register") ||
			strings.HasPrefix(path, "/api/v1/agent/ingest") ||
			path == "/api/v1/agent/oplog" ||
			path == "/api/v1/agent/heartbeat" ||
			path == "/api/v1/sovereignty/score" ||
			path == "/api/v1/tls/state" ||
			path == "/healthz" || path == "/readyz" {
			mux.ServeHTTP(w, r)
			return
		}

		if s.auth != nil {
			// Auth -> Tenant Isolation -> Rate Limit chain
			s.auth.Middleware(s.tenantMiddleware(s.tenantRateLimitMiddleware(mux))).ServeHTTP(w, r)
		} else {
			http.Error(w, "Auth not configured", http.StatusServiceUnavailable)
			return
		}
	})

	// Wrap entire router with security middleware (CORS, Headers, Global Rate Limiting)
	handler = s.secureMiddleware(finalHandler)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}

	// Audit fix #1 — replay-attack cache. 60s TTL > 30s HMAC window
	// so we never miss a replay due to clock-edge effects. Caps at
	// 100k fingerprints; bursts beyond that evict oldest.
	s.replayCache = NewReplayCache(60 * time.Second)

	// Audit fix #5 — bounded growth for the per-IP / per-tenant /
	// failed-login maps. 1h sweep, 24h TTL. See rate_limit_gc.go.
	s.startRateLimitGC()

	return s
}

// SetSuppression wires a SuppressionProvider into the REST server.
// Called from internal/core/container.go after the SuppressionService
// is constructed; we use a setter rather than a constructor argument
// to dodge the api ↔ services import cycle.
func (s *RESTServer) SetSuppression(p SuppressionProvider) {
	s.suppression = p
}

// SetSettings wires a SettingsProvider — same setter pattern as
// SetSuppression. See container.go for the hook-up.
func (s *RESTServer) SetSettings(p SettingsProvider) {
	s.settings = p
}

// Start spawns the HTTP listener in the background
func (s *RESTServer) Start() {
	s.log.Info("[REST] Starting headless API server on port %d", s.port)

	// SEC-AUDIT — production-mode CORS sanity check. Loud warning if
	// OBLIVRA_DEBUG=true is set on a host that looks like production
	// (TLS enabled, non-127.0.0.1 listener), because that flag widens
	// the CORS allowlist to include http://localhost:3000 + :5173 which
	// makes the deployment vulnerable to DNS-rebinding attacks. Operator
	// has to explicitly opt out via OBLIVRA_ALLOW_DEBUG_IN_PROD=true to
	// silence this warning (e.g. for a debug session in staging).
	if os.Getenv("OBLIVRA_DEBUG") == "true" &&
		os.Getenv("OBLIVRA_ALLOW_DEBUG_IN_PROD") != "true" &&
		s.certManager != nil {
		s.log.Warn("⚠ SECURITY: OBLIVRA_DEBUG=true is set on a TLS-serving host. " +
			"This widens CORS to include localhost:3000/5173 — vulnerable to DNS rebinding. " +
			"Unset OBLIVRA_DEBUG, or set OBLIVRA_ALLOW_DEBUG_IN_PROD=true to silence this warning.")
	}

	// Rehydrate the agent map from agent_registry so the fleet view is
	// populated immediately on restart (rather than waiting up to a
	// heartbeat interval for live re-registration).
	s.hydrateAgentRegistry()

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
			s.isTLSMu.Lock()
			s.isTLS = true
			s.isTLSMu.Unlock()
			err := s.server.ListenAndServeTLS("", "")
			if err != nil && err != http.ErrServerClosed {
				if os.Getenv("OBLIVRA_ENV") == "production" {
					s.log.Fatal("TLS required in production; cert load failed: %v", err)
				} else {
					s.log.Warn("[REST] TLS server failed: %v — falling back to plaintext HTTP", err)
				}
			} else {
				return
			}
		} else if os.Getenv("OBLIVRA_ENV") == "production" {
			s.log.Fatal("TLS required in production; no certificates loaded.")
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
	if s.enrichLimiter != nil {
		s.enrichLimiter.Stop()
	}
	return s.server.Shutdown(ctx)
}

func (s *RESTServer) IsTLS() bool {
	s.isTLSMu.RLock()
	defer s.isTLSMu.RUnlock()
	return s.isTLS
}

func (s *RESTServer) checkFeature(w http.ResponseWriter, f licensing.Feature) bool {
	if s.license == nil {
		return true // Missing provider (test/dev)
	}
	if err := s.license.RequireFeature(f); err != nil {
		s.log.Warn("[REST] Feature gate blocked: %s (Tier: %s)", f, s.license.CurrentTier())
		// checkFeature is a helper that doesn't carry the request,
		// so pass nil — respondError handles that gracefully.
		s.respondError(w, nil, http.StatusForbidden, "forbidden", "feature_gate", err)
		return false
	}
	return true
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
	if err == nil {
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err == nil {
			m["data_confidence"] = confidence
			json.NewEncoder(w).Encode(m)
			return
		}
	}
	
	// Fallback if data isn't an object
	wrapper := map[string]interface{}{
		"data_confidence": confidence,
		"data":            data,
	}
	json.NewEncoder(w).Encode(wrapper)
}

// stubHandler wraps endpoints whose backend implementation is
// in-progress. Critical-4 audit fix: instead of silently returning
// `http.Error("Not Implemented")` (which clients consumed as
// "succeeded with empty body"), we now respond with a structured
// envelope `{ok:false, code:"feature_pending"}` so frontend code
// can distinguish "feature not available" from "feature failed."
//
// Every stubHandler entry corresponds to a UI affordance that should
// either be hidden, disabled with a tooltip, or wired to a real
// backend. The `feature_pending` code lets frontend components opt
// into one of those behaviours instead of silently breaking.
//
// Production mode (OBLIVRA_ENV=production): returns 501 with the
// same envelope. Operators see the same shape — just the backend
// refuses to expose the un-finished feature surface at all.
//
// Non-production mode: requires an authenticated role so a stub
// can be exercised in dev without auth, but isn't open to the
// public internet.
func (s *RESTServer) stubHandler(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("OBLIVRA_ENV") == "production" {
			s.respondError(w, r, http.StatusNotImplemented,
				"feature not yet available in this build",
				"feature_pending", nil)
			// Hint the frontend can act on without parsing the message.
			w.Header().Set("X-Feature-Status", "pending")
			return
		}

		role := auth.GetRole(r.Context())
		if role == "" {
			s.respondError(w, r, http.StatusForbidden,
				"forbidden", "stub_unauth", nil)
			return
		}

		// Tag the response with the same hint so dev clients can
		// flag which paths are stubbed even when the body is empty.
		w.Header().Set("X-Feature-Status", "pending")
		h(w, r)
	}
}

// --- Middleware ---

func (s *RESTServer) tenantRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := database.TenantFromContext(r.Context())
		if tenantID == "" {
			tenantID = "GLOBAL"
		}
		
		// Audit fix #5 — entries are wrapped in *limiterEntry so the
		// background GC (sweepRateLimiters) can evict tenants we
		// haven't seen in 24h. The wrapper's touch() updates an
		// atomic last-used timestamp without taking a mutex.
		entryI, _ := s.tenantLimiters.LoadOrStore(tenantID, newLimiterEntry(rate.Limit(20), 50))
		entry := entryI.(*limiterEntry)
		entry.touch()

		if !entry.limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (s *RESTServer) secureMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Global Rate Limiting
		if !s.globalLimiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// 1.5 Per-IP Rate Limiting (Defense-in-Depth)
		// Audit fix #5 — wrap in *limiterEntry so the GC sweeper
		// can drop limiters belonging to IPs we haven't seen in
		// 24h. Without that wrapper the map grew without bound
		// against any drip portscan.
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		entryI, _ := s.ipLimiters.LoadOrStore(ip, newLimiterEntry(rate.Limit(5), 10)) // 5 req/sec per IP
		entry := entryI.(*limiterEntry)
		entry.touch()
		if !entry.limiter.Allow() {
			s.log.Warn("[SECURITY] Rate limit exceeded for IP: %s", ip)
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// 2. CORS Headers — restrict to local Wails frontend origins.
		//
		// Only echo Allow-Origin back when the caller's origin appears
		// in the allowlist. Cookies (HttpOnly auth) require
		// Allow-Credentials: true AND a specific (not wildcard) origin
		// — the browser will REFUSE to send cookies with
		// "Access-Control-Allow-Origin: *". Audit finding Critical #3.
		//
		// Allow-Methods now includes PUT/DELETE/PATCH because some
		// CRUD endpoints (e.g. tenant lifecycle, suppression rules)
		// genuinely use those verbs. Allow-Headers includes the new
		// X-Tenant-Id used by apiFetch (Phase 30.4d).
		origin := r.Header.Get("Origin")
		allowedOrigins := map[string]bool{
			// macOS / Linux WebKit2GTK
			"https://wails.localhost": true,
			"wails://wails":           true,
			"wails://wails.localhost": true,
			// Windows WebView2 — Wails v3 serves the asset bundle at
			// `http://wails.localhost` (not https). Without this entry
			// the desktop frontend's apiFetch calls land cross-origin
			// against the REST listener on 127.0.0.1:8080 and the
			// browser blocks the response. See apiFetch's getApiBase.
			"http://wails.localhost":  true,
		}
		// SEC-35: Allow localhost:3000 ONLY in debug mode to prevent DNS rebinding in production
		if os.Getenv("OBLIVRA_DEBUG") == "true" {
			allowedOrigins["http://localhost:3000"] = true
			allowedOrigins["http://localhost:5173"] = true // Vite dev server default
		}
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			// CRITICAL: required for HttpOnly auth cookies to ride
			// cross-origin requests. Without this, every browser-mode
			// API call after login fails silently with 401.
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key, X-Tenant-Id, X-Agent-ID, X-Agent-Hostname, X-Agent-Signature")
		w.Header().Set("Vary", "Origin")

		// 3. Security Headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws: wss:; frame-ancestors 'none';")
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
	if auth.GetRole(r.Context()) != auth.RoleAdmin {
		http.Error(w, "Forbidden: Admin only", http.StatusForbidden)
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

	// Multi-tenant isolation is enforced structurally one layer down:
	//   - auth middleware plumbs the authenticated tenant via database.WithTenant(ctx, ...)
	//   - SIEMStore.SearchHostEvents resolves it via MustTenantFromContext and dispatches
	//     to the tenant's dedicated Bleve index (internal/search/bleve.go:getIndex)
	// Concatenating "TenantID:X AND ..." into the query string here is redundant and
	// can interact badly with analyzer casing — the storage layer is the source of truth.
	identityUser := auth.UserFromContext(r.Context())

	events, err := s.siem.SearchHostEvents(r.Context(), query, limit)
	if err != nil {
		tenantLabel := ""
		if identityUser != nil {
			tenantLabel = identityUser.TenantID
		}
		s.log.Error("[REST] Search failed for tenant %s: %v", tenantLabel, err)
		http.Error(w, "Search unavailable. Internal processing error.", http.StatusInternalServerError)
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

	// Tenant scope is enforced by the storage layer (per-tenant Bleve index dispatch
	// keyed off the auth-middleware-supplied tenant context). See handleSearch above.
	identityUser := auth.UserFromContext(r.Context())
	query := "EventType:security_alert"

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	alerts, err := s.siem.SearchHostEvents(ctx, query, 100)
	cancel()
	if err != nil {
		tenantLabel := ""
		if identityUser != nil {
			tenantLabel = identityUser.TenantID
		}
		s.log.Error("[REST] Query failed for tenant %s: %v", tenantLabel, err)
		http.Error(w, "Query unavailable. Internal processing error.", http.StatusInternalServerError)
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

// handleHealthLoad exposes the minimal payload a frontend banner needs to decide
// whether to render a DEGRADED notice. It piggybacks on CollectStats so the
// load classification stays in lock-step with /api/v1/ingest/status.
//
// Response shape:
//
//	{
//	  "status": "healthy" | "degraded" | "critical",
//	  "queue_fill_pct": 0..100,
//	  "events_per_second": int64,
//	  "dropped_events": int64,
//	  "collected_at": RFC3339
//	}
//
// When the pipeline is not yet initialised the endpoint returns
// {"status":"unknown"} with HTTP 200 so the banner doesn't flash on cold boot.
func (s *RESTServer) handleHealthLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.pipeline == nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"status": "unknown",
		})
		return
	}

	snap := s.pipeline.GetMetrics()
	stats := ingest.CollectStats(snap, time.Since(snap.CollectedAt))
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":            stats.LoadStatus,
		"queue_fill_pct":    stats.QueueFillPct,
		"events_per_second": stats.EventsPerSecond,
		"dropped_events":    stats.DroppedEvents,
		"collected_at":      stats.CollectedAt.Format(time.RFC3339),
	})
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

	// 16 KB cap on replay request body — payload is just `{"wal_path":...}`.
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)

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

	// CS-05: Path Traversal and Directory Validation
	// Confirm the path is within the platform's WAL directory.
	walDir := platform.DataDir() + "/wal"
	cleanPath := filepath.Clean(req.WALPath)
	
	rel, err := filepath.Rel(walDir, cleanPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		s.log.Warn("[REST] Blocked suspicious WAL replay path: %s (Tenant: %s)", req.WALPath, auth.GetTenantID(r.Context()))
		http.Error(w, "Invalid WAL path: path must be within the WAL directory", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	result, err := s.replayer.ReplayWAL(ctx, cleanPath)
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
	// SEC-35: Enforce read limit to prevent DoS via massive WS payloads
	conn.SetReadLimit(32768)
	
	atomic.AddInt64(&s.activeWS, 1)
	defer atomic.AddInt64(&s.activeWS, -1)
	defer conn.Close()

	if s.bus == nil {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Event bus not configured"}`))
		return
	}

	// CS-22: Mandatory tenant isolation for event streams.
	// Only stream events belonging to the user's tenant — UNLESS the
	// caller is a platform-level global admin (wildcard '*' permission
	// or explicit 'platform:admin' / 'tenant:read:*' grant).
	// Phase 30 closed the historical TODO at this site by wiring
	// auth.IsGlobalAdminFromContext through the RBAC engine.
	userTenant := database.MustTenantFromContext(r.Context())
	isGlobalAdmin := auth.IsGlobalAdminFromContext(r.Context())
	if isGlobalAdmin {
		s.log.Info("[REST] Event stream: global-admin override (cross-tenant view enabled) tenant=%s", userTenant)
	}
	
	clientAddr := r.RemoteAddr
	s.log.Info("[REST] Client connected to event stream: %s (Tenant: %s)", clientAddr, userTenant)

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

	// Read loop to handle incoming collaboration messages and detect disconnect
	go func() {
		defer cleanup()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Parse incoming client message
			var clientMsg struct {
				Type string      `json:"type"`
				Data interface{} `json:"data"`
			}
			if err := json.Unmarshal(message, &clientMsg); err != nil {
				s.log.Warn("[REST] Failed to parse client WS message from %s: %v", clientAddr, err)
				continue
			}

			// CS-10 Hardening: Only allow specific collaboration types
			switch clientMsg.Type {
			case "collab.message", "presence.update":
				// CS-22: Inject tenant ID to ensure message is only broadcast to same-tenant analysts
				dataMap, ok := clientMsg.Data.(map[string]interface{})
				if !ok {
					s.log.Warn("[REST] Invalid data format for %s from %s", clientMsg.Type, clientAddr)
					continue
				}
				dataMap["tenant_id"] = userTenant
				s.bus.Publish(eventbus.EventType(clientMsg.Type), dataMap)
			default:
				s.log.Warn("[REST] Unauthorized client event type from %s: %s", clientAddr, clientMsg.Type)
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

			// CS-22: Filter by TenantID.
			// Allow "GLOBAL" events (system notifications) to all authenticated users.
			eventTenant := ""
			if t, ok := event.Data.(map[string]interface{})["tenant_id"].(string); ok {
				eventTenant = t
			}

			if !isGlobalAdmin && eventTenant != "" && eventTenant != "GLOBAL" && eventTenant != userTenant {
				continue
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

func (s *RESTServer) generateTokens(user *database.User) (string, string, error) {
	var jwtSecret []byte
	if s.keyProvider != nil {
		secret, err := s.keyProvider.GetSystemKey("jwt_signing_key")
		if err == nil {
			jwtSecret = secret
		}
	}
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("temp-bootstrap-secret-replace-me")
	}

	// 1. Access Token (15m)
	accessClaims := jwt.MapClaims{
		"sub":    user.ID,
		"email":  user.Email,
		"role":   user.RoleID,
		"tenant": user.TenantID,
		"type":   "access",
		"exp":    time.Now().Add(15 * time.Minute).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	// 2. Refresh Token (7d)
	refreshClaims := jwt.MapClaims{
		"sub":    user.ID,
		"type":   "refresh",
		"exp":    time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessStr, refreshStr, nil
}

func (s *RESTServer) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// CS-02: Read refresh token from secure cookie
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Missing refresh token", http.StatusUnauthorized)
		return
	}
	refreshTokenStr := cookie.Value

	var jwtSecret []byte
	if s.keyProvider != nil {
		secret, err := s.keyProvider.GetSystemKey("jwt_signing_key")
		if err == nil {
			jwtSecret = secret
		}
	}
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("temp-bootstrap-secret-replace-me")
	}

	token, err := jwt.Parse(refreshTokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "refresh" {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	userID := claims["sub"].(string)

	user, err := s.identity.GetUser(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	accessStr, refreshStr, err := s.generateTokens(user)
	if err != nil {
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	// CS-02: Secure Token Storage
	s.setAuthCookies(w, accessStr, refreshStr)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "refreshed",
	})
}

func (s *RESTServer) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	if auth.GetRole(r.Context()) == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	absPath, err := filepath.Abs("docs/openapi.yaml")
	if err != nil {
		http.Error(w, "File not found", http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, absPath)
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

	// 4 KB is plenty for login credentials; defends against
	// unbounded-body OOM probes against the auth endpoint.
	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
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

	// 2. Lockout Check
	//
	// Concurrency: two failed logins racing on the same email used to
	// both Load(count=4) → both compute count=5 → both Store, leaving
	// the counter at 5 instead of 6 (and effectively granting the
	// attacker one extra attempt). We serialize all failure-counter
	// mutations on a single mutex; login failures are rare so contention
	// is negligible. Successful logins still take the fast path through
	// failedLogins.Delete (sync.Map handles that).
	now := time.Now()
	s.failedLoginsMu.Lock()
	if lockout, ok := s.failedLogins.Load(req.Email); ok {
		info := lockout.(struct {
			count int
			until time.Time
		})
		if info.count >= 5 && now.Before(info.until) {
			s.failedLoginsMu.Unlock()
			s.log.Warn("[SECURITY] Login blocked for %s: too many failed attempts (locked until %v)", req.Email, info.until)
			http.Error(w, "Account temporarily locked. Please try again in 15 minutes.", http.StatusForbidden)
			return
		}
	}
	s.failedLoginsMu.Unlock()

	user, err := s.identity.LoginLocal(req.Email, req.Password)
	if err != nil {
		// Atomic check-then-increment under the same mutex used by the
		// lockout-read above. Without this, two parallel failed logins
		// could both read count=N, both compute N+1, both Store, and the
		// counter only advances by 1 — letting an attacker probe twice
		// per real failure.
		s.failedLoginsMu.Lock()
		count := 1
		if existing, ok := s.failedLogins.Load(req.Email); ok {
			count = existing.(struct {
				count int
				until time.Time
			}).count + 1
		}

		until := time.Time{}
		justLocked := false
		if count >= 5 {
			until = now.Add(15 * time.Minute)
			justLocked = true
		}

		s.failedLogins.Store(req.Email, struct {
			count int
			until time.Time
		}{count, until})
		s.failedLoginsMu.Unlock()

		// Persist the lockout state so a service restart can't be used to
		// reset the counter. Write-through on every increment (cheap; only
		// fires on failures, which are rare).
		if s.db != nil {
			lockedUntil := sql.NullTime{Time: until, Valid: !until.IsZero()}
			ip := r.RemoteAddr
			if _, err := s.db.DB().ExecContext(r.Context(),
				`INSERT INTO login_lockouts (ip, email, failed_count, last_failed_at, locked_until)
				 VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?)
				 ON CONFLICT(ip) DO UPDATE SET
				   email          = excluded.email,
				   failed_count   = excluded.failed_count,
				   last_failed_at = CURRENT_TIMESTAMP,
				   locked_until   = excluded.locked_until`,
				ip, req.Email, count, lockedUntil); err != nil {
				s.log.Warn("[security] Could not persist login lockout: %v", err)
			}
		}

		if justLocked {
			s.appendAuditEntry(req.Email, "lockout", "system", "locked", r)
		}

		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Success: clear failures
	s.failedLogins.Delete(req.Email)
	if s.db != nil {
		_, _ = s.db.DB().ExecContext(r.Context(),
			`DELETE FROM login_lockouts WHERE email = ?`, req.Email)
	}

	accessToken, refreshToken, err := s.generateTokens(user)
	if err != nil {
		http.Error(w, "Failed to generate session tokens", http.StatusInternalServerError)
		return
	}

	// CS-02: Secure Token Storage
	// Migrate from localStorage to HttpOnly cookies to prevent XSS-based exfiltration.
	s.setAuthCookies(w, accessToken, refreshToken)

	s.appendAuditEntry(user.Email, "login", "system", "success", r)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

func (s *RESTServer) setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	isSecure := s.IsTLS() || os.Getenv("OBLIVRA_ENV") == "production"

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   3600, // 1 hour
	})

	if refreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/api/v1/auth/refresh",
			HttpOnly: true,
			Secure:   isSecure,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400 * 7, // 7 days
		})
	}
}

func (s *RESTServer) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "Identity service disabled", http.StatusNotImplemented)
		return
	}

	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		MaxAge:   300,
		HttpOnly: true,
		Secure:   s.IsTLS(),
		SameSite: http.SameSiteLaxMode,
	})

	url, err := s.identity.GetOIDCURL()
	if err != nil {
		http.Error(w, "Failed to generate OIDC redirect", http.StatusInternalServerError)
		return
	}
	if strings.Contains(url, "?") {
		url += "&state=" + state
	} else {
		url += "?state=" + state
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *RESTServer) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state || state == "" {
		http.Error(w, "CSRF / State mismatch", http.StatusForbidden)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	user, err := s.identity.HandleOIDCCallback(code)
	if err != nil {
		http.Error(w, "Federated authentication failed", http.StatusUnauthorized)
		return
	}

	accessToken, refreshToken, err := s.generateTokens(user)
	if err != nil {
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/?user=%s&token=%s&refresh=%s", user.ID, accessToken, refreshToken), http.StatusTemporaryRedirect)
}

func (s *RESTServer) handleSAMLLogin(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.Error(w, "Identity service disabled", http.StatusNotImplemented)
		return
	}

	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	relayState := hex.EncodeToString(stateBytes)

	http.SetCookie(w, &http.Cookie{
		Name:     "saml_state",
		Value:    relayState,
		MaxAge:   300,
		HttpOnly: true,
		Secure:   s.IsTLS(),
		SameSite: http.SameSiteLaxMode,
	})

	url, err := s.identity.GetSAMLURL()
	if err != nil {
		http.Error(w, "Failed to generate SAML redirect", http.StatusInternalServerError)
		return
	}
	if strings.Contains(url, "?") {
		url += "&RelayState=" + relayState
	} else {
		url += "?RelayState=" + relayState
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *RESTServer) handleSAMLCallback(w http.ResponseWriter, r *http.Request) {
	relayState := r.FormValue("RelayState")
	cookie, err := r.Cookie("saml_state")
	if err != nil || cookie.Value != relayState || relayState == "" {
		http.Error(w, "CSRF / State mismatch", http.StatusForbidden)
		return
	}

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

	accessToken, refreshToken, err := s.generateTokens(user)
	if err != nil {
		http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/?user=%s&token=%s&refresh=%s", user.ID, accessToken, refreshToken), http.StatusTemporaryRedirect)
}

func (s *RESTServer) handleSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	baseURL := "https://" + r.Host
	if !s.IsTLS() {
		baseURL = "http://" + r.Host
	}
	// SEC-15: Output proper SP Metadata EntityDescriptor
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s/api/v1/auth/saml/metadata">
  <md:SPSSODescriptor AuthnRequestsSigned="false" WantAssertionsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:persistent</md:NameIDFormat>
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="%s/api/v1/auth/saml/callback" index="1"/>
  </md:SPSSODescriptor>
</md:EntityDescriptor>`, baseURL, baseURL)
	fmt.Fprint(w, xml)
}

func (s *RESTServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	isSecure := s.IsTLS() || os.Getenv("OBLIVRA_ENV") == "production"

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth/refresh",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
	})

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"status": "logged_out"})
}

// GET /api/v1/identities
func (s *RESTServer) handleIdentitiesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	users, err := s.identity.ListUsers(r.Context())
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"identities": users,
	})
}

// GET /api/v1/identities/roles
func (s *RESTServer) handleRolesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	roles, err := s.identity.ListRoles(r.Context())
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
		return
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"roles": roles,
	})
}

// GET /api/v1/platform/metrics
func (s *RESTServer) handlePlatformMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	metrics, err := s.platformProvider.GetMetrics(r.Context())
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
		return
	}
	s.jsonResponse(w, http.StatusOK, metrics)
}

// GET /api/v1/forensics/evidence?incident_id=...
func (s *RESTServer) handleEvidenceList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	incidentID := r.URL.Query().Get("incident_id")
	evidence := s.forensicsProvider.ListEvidence(r.Context(), incidentID)
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"evidence": evidence,
	})
}

// GET /api/v1/fusion/campaigns
func (s *RESTServer) handleCampaignList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	campaigns := s.fusion.GetActiveCampaigns()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"campaigns": campaigns,
	})
}

// GET /api/v1/fusion/timeline?id=...
func (s *RESTServer) handleCampaignTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	timeline, err := s.fusion.GetCampaignTimeline(r.Context(), id)
	if err != nil {
		s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
		return
	}
	s.jsonResponse(w, http.StatusOK, timeline)
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
	if s.auth == nil {
		s.respondError(w, r, http.StatusServiceUnavailable, "service unavailable", "auth_not_configured", nil)
		return
	}
	// This relies on the middleware having injected the user into context.
	identityUser := auth.UserFromContext(r.Context())
	if identityUser == nil {
		s.respondError(w, r, http.StatusUnauthorized, "unauthorized", "no_user_in_context", nil)
		return
	}

	// Audit fix M-16: cross-check the user's TenantID against the
	// request's tenant context. If a JWT or session was forged with
	// a tenant that doesn't match the request scope (X-Tenant-Id
	// header / URL param / session-derived), refuse — this is the
	// last line of defence against tenant-spoofing.
	requestTenant, _ := database.TenantFromContext(r.Context())
	if requestTenant != "" && identityUser.TenantID != "" &&
		requestTenant != identityUser.TenantID &&
		!auth.IsGlobalAdminFromContext(r.Context()) {
		s.log.Warn("[auth] handleMe: tenant mismatch user_tenant=%s request_tenant=%s user=%s",
			identityUser.TenantID, requestTenant, identityUser.Email)
		s.respondError(w, r, http.StatusForbidden, "forbidden", "tenant_mismatch", nil)
		return
	}

	userCopy := *identityUser
	if auth.Role(userCopy.RoleName) != auth.RoleAdmin {
		userCopy.Permissions = []string{}
	}
	s.jsonResponse(w, http.StatusOK, userCopy)
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
		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
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
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var req notifications.AckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	if err := s.getEscalation().Acknowledge(req); err != nil {
		s.respondError(w, r, http.StatusBadRequest, publicErrorMessage(http.StatusBadRequest, ""), "bad_request", err)
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

	// Add to recent ring buffer scoped by tenant
	tenant, _ := database.TenantFromContext(r.Context())
	if tenant == "" { tenant = "GLOBAL" }

	s.enrichMu.Lock()
	list := s.enrichRecent[tenant]
	list = append([]map[string]interface{}{result}, list...)
	if len(list) > 20 {
		list = list[:20]
	}
	s.enrichRecent[tenant] = list
	s.enrichMu.Unlock()

	s.jsonResponse(w, http.StatusOK, result)
}

// GET /api/v1/enrich/recent
func (s *RESTServer) handleEnrichRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tenant, _ := database.TenantFromContext(r.Context())
	if tenant == "" { tenant = "GLOBAL" }

	s.enrichMu.RLock()
	defer s.enrichMu.RUnlock()
	
	list := s.enrichRecent[tenant]
	if list == nil {
		list = make([]map[string]interface{}, 0)
	}
	
	s.jsonResponse(w, http.StatusOK, list)
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
		authUser := auth.UserFromContext(r.Context())
		if authUser == nil {
			s.jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{"error": "unauthorized"})
			return
		}
		actor := authUser.Email
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
		s.respondError(w, r, http.StatusInternalServerError, "internal error", "operation_failed", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"oblivra-evidence.json\"")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// ── Audit / Regulator Portal Handlers (Phase 6.6) ────────────────────────────

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		parts := strings.Split(ip, ",")
		return strings.TrimSpace(parts[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	return r.RemoteAddr
}

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
		"ip":         getClientIP(r),
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

// Compliance audit packages are persisted in the `compliance_packages`
// SQL table (migration v28). The previous in-memory map was lost on every
// restart, breaking audit-trail durability — flagged as a high-severity
// persistence bug in the database audit.

// GET /api/v1/audit/packages
func (s *RESTServer) handleAuditPackages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tenant, _ := database.TenantFromContext(r.Context())
	if tenant == "" {
		tenant = "GLOBAL"
	}
	rows, err := s.db.DB().QueryContext(r.Context(),
		`SELECT id, framework, generated_at, records, integrity_proof, download_url, from_ts, to_ts
		   FROM compliance_packages
		  WHERE tenant_id = ?
		  ORDER BY generated_at DESC`, tenant)
	if err != nil {
		s.log.Error("[compliance] list failed: %v", err)
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"packages": []any{}})
		return
	}
	defer rows.Close()
	pkgs := []map[string]interface{}{}
	for rows.Next() {
		var id, framework, gen, proof, url string
		var records int
		var from, to sql.NullString
		if err := rows.Scan(&id, &framework, &gen, &records, &proof, &url, &from, &to); err != nil {
			continue
		}
		pkg := map[string]interface{}{
			"id":              id,
			"framework":       framework,
			"generated_at":    gen,
			"records":         records,
			"integrity_proof": proof,
			"download_url":    url,
		}
		if from.Valid {
			pkg["from"] = from.String
		}
		if to.Valid {
			pkg["to"] = to.String
		}
		pkgs = append(pkgs, pkg)
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"packages": pkgs})
}

// POST /api/v1/audit/packages/generate
func (s *RESTServer) handleAuditPackageGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
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

	count, _ := s.audit.Count(r.Context())

	id := fmt.Sprintf("CP-%s-%d", req.Framework, time.Now().Unix())
	proof := fmt.Sprintf("%x", sha256Hash([]byte(id+req.Framework+req.From+req.To)))
	generatedAt := time.Now().Format(time.RFC3339)
	downloadURL := fmt.Sprintf("/api/v1/audit/packages/%s/download", id)

	tenant, _ := database.TenantFromContext(r.Context())
	if tenant == "" {
		tenant = "GLOBAL"
	}

	if _, err := s.db.DB().ExecContext(r.Context(),
		`INSERT INTO compliance_packages
		   (id, tenant_id, framework, generated_at, records, integrity_proof, download_url, from_ts, to_ts)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, tenant, req.Framework, generatedAt, count, proof, downloadURL, req.From, req.To); err != nil {
		s.log.Error("[compliance] persist failed: %v", err)
		http.Error(w, "persist failed", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"id":              id,
		"framework":       req.Framework,
		"generated_at":    generatedAt,
		"records":         count,
		"integrity_proof": proof,
		"download_url":    downloadURL,
	})
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

var mcpApprovalsMu sync.Mutex
var mcpApprovals = make(map[string]map[string]bool)

func (s *RESTServer) handleMCPApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
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
	
	mcpApprovalsMu.Lock()
	if mcpApprovals[req.ApprovalID] == nil {
		mcpApprovals[req.ApprovalID] = make(map[string]bool)
	}
	mcpApprovals[req.ApprovalID][req.ActorID] = true
	count := len(mcpApprovals[req.ApprovalID])
	mcpApprovalsMu.Unlock()

	requiredApprovals := 2 // SEC-12: M of N threshold

	// 2. Increment approval count for ApprovalID
	// 3. If threshold met, generate a one-time execution token
	if count < requiredApprovals {
		s.jsonResponse(w, http.StatusAccepted, map[string]interface{}{
			"status":   "pending_threshold",
			"approved": count,
			"required": requiredApprovals,
		})
		return
	}

	mcpApprovalsMu.Lock()
	delete(mcpApprovals, req.ApprovalID)
	mcpApprovalsMu.Unlock()

	// MVP: Generate a simple HMAC-based token
	token := s.mcpHandler.GenerateApprovalToken(req.ApprovalID, req.ActorID)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "approved",
		"token":  token,
	})
}

// ── Graph Handlers (Phase 9) ────────────────────────────────────────────────

func (s *RESTServer) handleGraphMetrics(w http.ResponseWriter, r *http.Request) {
	if s.graphEngine == nil {
		http.Error(w, "Graph engine not initialized", http.StatusNotImplemented)
		return
	}

	stats := s.graphEngine.Stats()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"node_count":       stats.NodeCount,
		"edge_count":       stats.EdgeCount,
		"rich_edge_count":  stats.RichEdgeCount,
	})
}

// handleGraphFull returns the entire security-graph snapshot. The
// payload powers the Topology page; for very large graphs the operator
// is expected to paginate / use the subgraph endpoint instead.
func (s *RESTServer) handleGraphFull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.graphEngine == nil {
		http.Error(w, "Graph engine not initialized", http.StatusNotImplemented)
		return
	}
	nodes, edges := s.graphEngine.GetAll()
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
}

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

// Phase 36.9: handleComplianceStatus removed. The 410-Gone shim was kept
// during Phase 36.x while the frontend store still pinged the endpoint;
// store deleted in Phase 36.9, so the shim is also gone now.

// SetupRequest is the payload SetupWizard.svelte POSTs on completion.
type SetupRequest struct {
	Admin struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	} `json:"admin"`
	AlertChannel *struct {
		Type   string `json:"type"`   // "email" | "webhook" | "slack"
		Target string `json:"target"` // address / URL
	} `json:"alert_channel,omitempty"`
	DetectionPack string `json:"detection_pack"` // "essential" | "extended" | "paranoid"
}

// POST /api/v1/setup/initialize — first-run platform configuration.
//
// Idempotency: once an admin user exists, this endpoint refuses with 409.
// That guards against an unauthenticated caller hijacking admin access on a
// running system — the underlying IdentityService.BootstrapAdmin enforces
// the same rule, but the API-layer pre-check produces a cleaner error.
func (s *RESTServer) handleSetupInitialize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.identity == nil {
		http.Error(w, "Identity service not available", http.StatusServiceUnavailable)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024) // 16 KB is plenty
	var req SetupRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "Invalid setup payload", http.StatusBadRequest)
		return
	}

	if req.Admin.Email == "" || req.Admin.Password == "" {
		http.Error(w, "admin.email and admin.password are required", http.StatusBadRequest)
		return
	}

	switch req.DetectionPack {
	case "", "essential", "extended", "paranoid":
		// allowed; "" defaults to "extended" below
	default:
		http.Error(w, "detection_pack must be essential, extended, or paranoid", http.StatusBadRequest)
		return
	}
	pack := req.DetectionPack
	if pack == "" {
		pack = "extended"
	}

	if req.AlertChannel != nil {
		switch req.AlertChannel.Type {
		case "email", "webhook", "slack":
			if req.AlertChannel.Target == "" {
				http.Error(w, "alert_channel.target is required when alert_channel is set", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "alert_channel.type must be email, webhook, or slack", http.StatusBadRequest)
			return
		}
	}

	// Bootstrap the admin user. The underlying service refuses to run if any
	// user already exists, so this safely no-ops on a re-run attempt.
	name := req.Admin.Name
	if name == "" {
		name = "Administrator"
	}
	user, err := s.identity.BootstrapAdmin(r.Context(), req.Admin.Email, name, req.Admin.Password)
	if err != nil {
		// Most likely cause: setup already run. Surface as 409 Conflict.
		s.log.Warn("[SETUP] BootstrapAdmin refused: %v", err)
		http.Error(w, "Setup already initialised or admin creation refused", http.StatusConflict)
		return
	}

	// Record the configuration in the Merkle-chained audit log so first-run
	// state is recoverable from the audit trail alone (useful for support
	// debugging "why is my detection pack X").
	if s.audit != nil {
		details := map[string]interface{}{
			"admin_user_id": user.ID,
			"admin_email":   user.Email,
			"detection_pack": pack,
			"actor_ip":      getClientIP(r),
		}
		if req.AlertChannel != nil {
			details["alert_channel_type"] = req.AlertChannel.Type
			// Never log the target — it might be a webhook secret or an
			// email distribution list that's privacy-sensitive.
			details["alert_channel_target_len"] = len(req.AlertChannel.Target)
		}
		if logErr := s.audit.Log(r.Context(), "setup.initialized", "", "", details); logErr != nil {
			s.log.Error("[SETUP] audit log failed: %v", logErr)
		}
	}

	// Notify the rest of the platform so listeners can react: the alerting
	// service can attach the channel; the rule loader can switch packs.
	if s.bus != nil {
		s.bus.Publish("setup:initialized", map[string]interface{}{
			"admin_user_id":  user.ID,
			"detection_pack": pack,
			"alert_channel":  req.AlertChannel, // ok to publish: subscribers run in-process
		})
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":         "initialised",
		"admin_user_id":  user.ID,
		"detection_pack": pack,
	})
}
