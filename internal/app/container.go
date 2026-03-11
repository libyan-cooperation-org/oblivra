package app

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/isolation"
	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/compliance"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/decision"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/discovery"
	"github.com/kingknull/oblivrashell/internal/enrich"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/gdpr"
	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/incident"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/lineage"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/logsources"
	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/ndr"
	"github.com/kingknull/oblivrashell/internal/notes"
	"github.com/kingknull/oblivrashell/internal/notifications"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/kingknull/oblivrashell/internal/policy"
	"github.com/kingknull/oblivrashell/internal/risk"
	"github.com/kingknull/oblivrashell/internal/search"
	"github.com/kingknull/oblivrashell/internal/security"
	"github.com/kingknull/oblivrashell/internal/sharing"
	"github.com/kingknull/oblivrashell/internal/simulation"
	"github.com/kingknull/oblivrashell/internal/storage"
	syncpkg "github.com/kingknull/oblivrashell/internal/sync"
	"github.com/kingknull/oblivrashell/internal/temporal"
	"github.com/kingknull/oblivrashell/internal/threatintel"
	uebapkg "github.com/kingknull/oblivrashell/internal/ueba"
	"github.com/kingknull/oblivrashell/internal/updater"
	"github.com/kingknull/oblivrashell/internal/vault"
	"github.com/kingknull/oblivrashell/internal/workspace"
)

// busShim bridges the eventbus with sharing package
type busShim struct {
	bus *eventbus.Bus
}

func (s *busShim) Publish(topic string, data interface{}) {
	s.bus.Publish(eventbus.EventType(topic), data)
}

// ServiceRegistry manages the lifecycle of all application services.
type ServiceRegistry struct {
	services []Service
	mu       sync.RWMutex
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make([]Service, 0),
	}
}

func (r *ServiceRegistry) Register(s Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services = append(r.services, s)
}

func (r *ServiceRegistry) StartAll(ctx context.Context) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.services {
		s.Startup(ctx)
	}
}

func (r *ServiceRegistry) StopAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := len(r.services) - 1; i >= 0; i-- {
		s := r.services[i]
		s.Shutdown()
	}
}

// Container holds all application dependencies and core services.
type Container struct {
	// Infrastructure
	DB               *database.Database
	Vault            *vault.Vault
	Bus              *eventbus.Bus
	Log              *logger.Logger
	Registry         *ServiceRegistry
	FIDO2Manager     *security.FIDO2Manager
	SearchEngine     *search.SearchEngine // Holds the initialized bleve index
	RecordingManager *sharing.RecordingManager
	ShareManager     *sharing.ShareManager
	BroadcastService *BroadcastService
	HotStore         *storage.HotStore
	SIEMForwarder    *security.SIEMForwarder
	TelemetryManager *monitoring.TelemetryManager
	HealthChecker    *monitoring.HealthChecker
	MetricsCollector *monitoring.MetricsCollector
	AnalyticsEngine  *analytics.AnalyticsEngine
	AlertEngine      *analytics.AlertEngine
	Notifier         *notifications.NotificationService
	SourceManager    *logsources.SourceManager
	NotesManager     *notes.NotesManager
	WorkspaceManager *workspace.WorkspaceManager
	Updater          *updater.Updater
	SyncEngine       *syncpkg.SyncEngine
	DiscoveryManager *discovery.DiscoveryManager
	ReportGenerator  *compliance.ReportGenerator
	Platform         platform.Platform

	// Logic Services
	SecurityService   *SecurityService
	HostService       *HostService
	VaultService      *VaultService
	SSHService        *SSHService
	SessionService    *SessionService
	SettingsService   *SettingsService
	SnippetService    *SnippetService
	MultiExecService  *MultiExecService
	PluginService     *PluginService
	SIEMService       *SIEMService
	TeamService       *TeamService
	LocalService      *LocalService
	ComplianceService *ComplianceService
	TelemetryService  *TelemetryService
	AIService         *AIService
	UEBAEngine        *uebapkg.UEBAService
	UEBAService       *UEBAService

	// Newly Wired Services
	AlertingService  *AlertingService
	HealthService    *HealthService
	MetricsService   *MetricsService
	TunnelService    *TunnelService
	ShareService     *ShareService
	ClusterService   *ClusterService
	RecordingService *RecordingService
	LogSourceService *LogSourceService
	TransferManager  *TransferManager
	Sanitizer        *security.ShellSanitizer
	WorkspaceService *WorkspaceService
	NotesService     *NotesService
	UpdaterService   *UpdaterService
	SyncService      *SyncService
	FileService      *FileService
	DiscoveryService *DiscoveryService
	IngestService    *IngestService
	APIService       *APIService
	AgentService     *AgentService
	SyntheticService *SyntheticService
	TailingService   *TailingService

	// Engines
	SyntheticManager *monitoring.SyntheticManager

	// Phase 6 + Sovereign Meta-Layer Services
	ForensicsService     *ForensicsService
	ObservabilityService *ObservabilityService
	DisasterService      *DisasterService
	GovernanceService    *GovernanceService
	PolicyService        *PolicyService
	IncidentService      *IncidentService
	PlaybookService      *PlaybookService
	RansomwareEngine         *detection.RansomwareEngine
	CanaryService            *security.CanaryService
	CanaryDeploymentService  *CanaryDeploymentService
	SimulationService        *simulation.SimulationService
	NetworkIsolator          *isolation.NetworkIsolator
	NetworkIsolatorService   *NetworkIsolatorService
	RansomwareService        *RansomwareService

	// Phase 9.5 & 11: Strategic Intelligence
	GraphEngine  *graph.GraphEngine
	GraphService *GraphService
	NDRCollector *ndr.FlowCollector
	NDRService   *NDRService

	MatchEngine        *threatintel.MatchEngine
	EnrichmentPipeline *enrich.Pipeline

	RiskEngine  *risk.RiskEngine
	RiskService *RiskService

	TrustService *RuntimeTrustService

	CredentialIntel *CredentialIntelService
	IdentityService *IdentityService
	Lifecycle       *DataLifecycleService

	// Sovereign Hardening & Intelligence
	TemporalService       *TemporalService
	LineageEngine         *lineage.LineageEngine
	LineageService        *LineageService
	DecisionEngine        *decision.DecisionEngine
	DecisionService       *DecisionService
	CounterfactualEngine  *detection.CounterfactualEngine
	CounterfactualService *CounterfactualService
	LedgerService         *LedgerService
	MemorySecurity        *MemorySecurityService
	DeterministicResponse *DeterministicResponseService
	AttestationService    *attestation.AttestationService
}

func NewContainer(log *logger.Logger, v string) *Container {
	return &Container{
		Log:      log,
		Registry: NewServiceRegistry(),
		Platform: platform.Detect(),
	}
}

func (c *Container) Init(ctx context.Context) error {
	c.Log.Info("Initializing application container...")

	// 1. Core Infrastructure
	c.AttestationService = attestation.NewAttestationService()
	c.Bus = eventbus.NewBus(c.Log)
	c.DB = &database.Database{}

	v, err := vault.New(vault.Config{
		StorePath: platform.ConfigDir(),
		Platform:  c.Platform,
	}, c.Log)
	if err != nil {
		return err
	}
	c.Vault = v

	c.FIDO2Manager = security.NewFIDO2Manager()
	c.TelemetryManager = monitoring.NewTelemetryManager()
	c.HealthChecker = monitoring.NewHealthChecker(60 * time.Second)
	c.MetricsCollector = monitoring.NewMetricsCollector()
	c.SyntheticManager = monitoring.NewSyntheticManager(c.Log)
	c.SIEMForwarder = security.NewSIEMForwarder(security.SIEMConfig{Enabled: false}, c.Log)

	c.AnalyticsEngine = analytics.NewAnalyticsEngine(c.Log)
	c.Notifier = notifications.NewNotificationService(c.Log)
	c.AlertEngine = analytics.NewAlertEngine(c.Notifier, c.AnalyticsEngine)
	c.SourceManager = logsources.NewSourceManager(c.Log)
	c.NotesManager = notes.NewNotesManager()
	c.Updater = updater.NewUpdater("https://github.com/kingknull/oblivrashell", "0.1.0", c.Log)
	c.SyncEngine = syncpkg.NewSyncEngine("", "")
	c.DiscoveryManager = discovery.NewDiscoveryManager()
	c.RecordingManager = sharing.NewRecordingManager(c.AnalyticsEngine, c.Vault)

	// 2. Repositories
	hostRepo := database.NewHostRepository(c.DB, c.Vault)
	sessRepo := database.NewSessionRepository(c.DB)
	credRepo := database.NewCredentialRepository(c.DB)
	auditRepo := database.NewAuditRepository(c.DB)

	snippetRepo := database.NewSnippetRepository(c.DB)
	workspaceRepo := database.NewWorkspaceRepository(c.DB)
	evidenceRepo := database.NewEvidenceRepository(c.DB)

	// Initialize BadgerDB HotStore (non-fatal — app runs in degraded mode without it)
	hotStore, err := storage.NewHotStore(platform.DataDir(), c.Log)
	if err != nil {
		c.Log.Error("Failed to initialize BadgerDB HotStore: %v", err)
		c.Log.Warn("SIEM hot storage unavailable — running in degraded mode. Kill any stale processes and restart.")
		// Continue without HotStore — SIEM features will be limited
	}
	c.HotStore = hotStore
	var siemRepo database.SIEMStore
	if hotStore != nil {
		siemRepo = storage.NewBadgerSIEMRepository(hotStore, &c.SearchEngine, c.DB)
	}

	// Initialize Ingestion Pipeline
	wal, err := storage.NewWAL(platform.DataDir(), c.Log)
	if err != nil {
		c.Log.Error("Failed to initialize WAL: %v", err)
	}

	// Initialize Temporal Integrity Service early for pipeline integration
	temporalEngine := temporal.NewIntegrityService(temporal.DefaultPolicy(), c.Bus, c.Log)
	c.TemporalService = NewTemporalService(temporalEngine, c.Bus, c.Log)

	pipeline := ingest.NewPipeline(100000, wal, c.AnalyticsEngine, siemRepo, c.Bus, c.Log, temporalEngine)
	syslogServer := ingest.NewSyslogServer(pipeline, 1514, c.Log)

	// Agent telemetry listener (mTLS)
	agentSrv := ingest.NewAgentServer(pipeline, 8443, "", "", "", c.Log)

	c.WorkspaceManager = workspace.NewWorkspaceManager(workspaceRepo)
	c.ReportGenerator = compliance.NewReportGenerator(auditRepo, sessRepo, hostRepo)

	// 3. Logic Services
	c.SessionService = NewSessionService(sessRepo, auditRepo, c.Bus, c.Log)
	c.SecurityService = NewSecurityService(c.FIDO2Manager, nil, nil, c.Bus, c.Log)
	c.VaultService = NewVaultService(c.Vault, c.DB, c.AnalyticsEngine, &c.SearchEngine, credRepo, auditRepo, c.FIDO2Manager, c.Bus, c.Log)
	c.HostService = NewHostService(c.DB, c.Vault, hostRepo, c.Bus, c.Log)

	// GDPR Data Destruction
	destroyer := gdpr.NewDataDestructionService(c.DB.DB())
	c.SettingsService = NewSettingsService(c.DB, c.Bus, c.Log, destroyer)
	c.TeamService = NewTeamService(nil, c.Bus, c.Log)
	c.AIService = NewAIService(c.Vault, c.Bus, c.Log)

	// 4. Multiplayer & Sharing Bridge
	c.BroadcastService = NewBroadcastService(nil, c.Log)
	c.ShareManager = sharing.NewShareManager(c.BroadcastService, &busShim{bus: c.Bus})
	c.TailingService = NewTailingService(c.Bus, c.Log)
	c.SyntheticService = NewSyntheticService(c.SyntheticManager, c.Log)

	// 5. Advanced Logic Services
	c.Sanitizer = security.NewShellSanitizer()
	c.LocalService = NewLocalService(c.Bus, c.Log, c.SessionService, c.RecordingManager)
	c.SSHService = NewSSHService(c.DB, c.Vault, hostRepo, sessRepo, credRepo, siemRepo, c.Bus, c.Log, c.ShareManager, c.RecordingManager, c.TelemetryManager, nil, c.Sanitizer, c.TailingService)
	c.TransferManager = NewTransferManager(c.SSHService, c.Bus, c.Log)
	c.SSHService.SetTransferManager(c.TransferManager)

	c.SnippetService = NewSnippetService(snippetRepo, c.SSHService, c.TeamService, c.Bus, c.Log)
	c.MultiExecService = NewMultiExecService(hostRepo, c.VaultService, c.Bus, c.Log)

	// Init the Threat Intel MatchEngine
	c.MatchEngine = threatintel.NewMatchEngine(c.Log)

	// Init Enrichment Pipeline
	c.EnrichmentPipeline = enrich.NewPipeline(c.Bus, c.Log)
	geoipEnricher, err := enrich.NewGeoIPEnricher("", "")
	if err == nil {
		c.EnrichmentPipeline.Add(geoipEnricher)
	} else {
		c.Log.Warn("Failed to init GeoIP enricher: %v", err)
	}
	c.EnrichmentPipeline.Add(enrich.NewDNSEnricher())
	c.EnrichmentPipeline.Add(enrich.NewAssetEnricher(hostRepo, sessRepo))
	c.EnrichmentPipeline.Start(ctx)

	// Init SIEM & Analytics
	// NOTE: siemRepo was already constructed above from HotStore. Reuse the same instance.
	// SIEMForwarder is already initialized above — no need to recreate it here.
	c.SIEMService = NewSIEMService(siemRepo, c.SIEMForwarder, c.AIService, c.SnippetService, c.MatchEngine, c.Bus, c.Log)

	// c.SIEMService registration moved to bottom of Init with other services
	c.PluginService = NewPluginService(c.Bus, c.Log)

	// Init Compliance Evaluator
	complianceEval, err := compliance.NewEvaluator()
	if err != nil {
		c.Log.Error("Failed to initialize compliance evaluator: %v", err)
	}

	c.TelemetryService = NewTelemetryService(c.Log, c.TelemetryManager)
	c.UEBAEngine = uebapkg.NewUEBAService(hostRepo, c.Bus, c.HotStore, c.Log)
	c.UEBAService = NewUEBAService(c.UEBAEngine, c.Bus, c.Log)

	// 6. Newly Wired Services
	rulesDir := filepath.Join(platform.DataDir(), "rules")
	evaluator, err := detection.NewEvaluator(rulesDir, c.Log)
	if err != nil {
		c.Log.Error("Failed to initialize detection loop: %v", err)
	}

	c.IncidentService = NewIncidentService(c.DB, auditRepo, evidenceRepo, c.Bus, c.Log)
	c.AlertingService = NewAlertingService(c.AlertEngine, c.Notifier, c.AnalyticsEngine, siemRepo, c.IncidentService, evaluator, c.Bus, c.Log)
	c.HealthService = NewHealthService(c.Log, c.Bus, c.HealthChecker)
	c.MetricsService = NewMetricsService(c.Log, c.MetricsCollector)
	c.TunnelService = NewTunnelService(c.Bus, c.Log)
	c.ShareService = NewShareService(c.ShareManager, c.RecordingManager, c.Bus, c.Log)
	c.ClusterService = NewClusterService(c.DB, c.Bus, c.Log)
	c.RecordingService = NewRecordingService(c.RecordingManager, c.Bus, c.Log)
	c.LogSourceService = NewLogSourceService(c.SourceManager, c.AnalyticsEngine, c.Bus, c.Log)
	c.WorkspaceService = NewWorkspaceService(c.WorkspaceManager, c.Bus, c.Log)
	c.NotesService = NewNotesService(c.NotesManager, c.Bus, c.Log)
	c.UpdaterService = NewUpdaterService(c.Log, c.Updater)
	c.SyncService = NewSyncService(c.SyncEngine, c.Bus, c.Log)
	c.FileService = NewFileService(c.LocalService, c.SSHService, c.Log)
	c.DiscoveryService = NewDiscoveryService(c.Log, c.DiscoveryManager)
	c.IngestService = NewIngestService(pipeline, syslogServer, agentSrv, c.Bus, c.Log)
	c.AgentService = NewAgentService(agentSrv, c.Log)

	// Phase 6 + Sovereign Meta-Layer Services
	c.ForensicsService = NewForensicsService(evidenceRepo, c.Vault, c.Bus, c.Log)
	c.ObservabilityService = NewObservabilityService(c.Bus, c.MetricsCollector, c.HotStore, c.Log)
	c.DisasterService = NewDisasterService(platform.DataDir(), c.Vault, c.Bus, c.Log)
	c.GovernanceService = NewGovernanceService(c.Bus, c.Log)

	// Phase 8: Autonomous Response
	playbookEngine := incident.NewPlaybookEngine(c.SSHService, c.Notifier, c.DB, auditRepo, c.Bus, c.Log)
	c.PlaybookService = NewPlaybookService(playbookEngine)

	// Phase 9: Ransomware Defense
	c.RansomwareEngine = detection.NewRansomwareEngine(c.Bus, c.DB, c.Log)
	c.CanaryService = security.NewCanaryService(c.Log, c.SSHService)
	c.CanaryService.SetBus(c.Bus) // wire event bus for auto-deploy + hit detection
	c.NetworkIsolator = isolation.NewNetworkIsolator(c.SSHService, c.SSHService, "", c.Bus, c.Log)
	c.RansomwareService = NewRansomwareService(c.NetworkIsolator, c.Bus, c.Log)
	// Phase 9 completion: automatic canary deployment on agent registration
	c.CanaryDeploymentService = NewCanaryDeploymentService(c.CanaryService, c.SSHService, c.Bus, c.Log)
	// Phase 9 completion: sovereign NetworkIsolatorService (event-driven, frontend-exposed)
	c.NetworkIsolatorService = NewNetworkIsolatorService(c.SSHService, c.PlaybookService, c.Bus, c.Log)
	c.SimulationService = simulation.NewSimulationService(c.Bus, c.Log)

	// Phase 9.5 & 11: Strategic Intelligence
	c.GraphEngine = graph.NewGraphEngine(c.Bus, c.Log)
	c.GraphService = NewGraphService(c.GraphEngine, c.Log)
	c.NDRCollector = ndr.NewFlowCollector(c.Bus, c.Log)
	c.NDRService = NewNDRService(c.NDRCollector, c.Bus, c.Log)

	// Sovereign Hardening
	c.TrustService = NewRuntimeTrustService(c.Bus, c.Log, c.AttestationService, c.Vault)

	// Phase 7.5: Tactical Edge Extensions
	c.RiskEngine = risk.NewRiskEngine(c.Bus, c.DB, hostRepo, c.Log)
	c.RiskService = NewRiskService(c.RiskEngine, c.Bus, c.Log)

	c.CredentialIntel = NewCredentialIntelService(c.Bus, c.Log)

	// Hardware Rooted Identity
	var hwProvider auth.HardwareRootedIdentity
	if tpm, err := security.NewTPMManager(c.Log); err == nil {
		// handle 0x81000001 is a common persistent handle for AK/EK
		hwProvider = auth.NewTPMIdentityProvider(tpm, 0x81000001)
	}

	// Identity & RBAC
	userRepo := database.NewUserRepository(c.DB)
	roleRepo := database.NewRoleRepository(c.DB)
	rbacEngine := auth.NewRBACEngine(c.Log)
	c.IdentityService = NewIdentityService(userRepo, roleRepo, rbacEngine, hwProvider, c.Bus, c.Log)

	// Data Lifecycle Management
	c.Lifecycle = NewDataLifecycleService(c.DB, c.Bus, c.Log)

	// Data Lineage Tracking
	c.LineageEngine = lineage.NewLineageEngine(c.Bus, c.Log)
	c.LineageService = NewLineageService(c.LineageEngine, c.Bus, c.Log)

	// Decision Traceability
	c.DecisionEngine = decision.NewDecisionEngine(c.Bus, c.Log)
	c.DecisionService = NewDecisionService(c.DecisionEngine, c.Log)

	// Counterfactual Simulation
	c.CounterfactualEngine = detection.NewCounterfactualEngine(evaluator)
	c.CounterfactualService = NewCounterfactualService(c.CounterfactualEngine, evaluator, c.Log)

	// Deterministic Response Engine
	detExec := decision.NewDeterministicExecutor(c.Bus, c.Log)
	c.DeterministicResponse = NewDeterministicResponseService(detExec, c.Log)

	// Sovereign Evidence Cryptographic Ledger
	c.LedgerService = NewLedgerService()

	// Secure Memory Lifecycle Control
	c.MemorySecurity = NewMemorySecurityService(c.Log)

	// Phase 7.5 Feature Governance
	policyEngine := policy.NewEngine(policy.Config{ActiveTier: policy.TierEnterprise}, c.Log)

	// Attempt to load offline cache
	bundlePath := filepath.Join(platform.DataDir(), "policy.obp")
	if _, err := os.Stat(bundlePath); err == nil {
		c.Log.Info("Found offline policy bundle at %s, attempting to load...", bundlePath)
		// Dummy 32-byte public key constraint for demonstration.
		// In production, this is embedded deeply into the compiled binary or HSM.
		dummyPubKey := make([]byte, 32)
		for i := range dummyPubKey {
			dummyPubKey[i] = byte(i)
		}
		if cfg, err := policy.LoadBundle(bundlePath, dummyPubKey); err == nil {
			policyEngine.ApplyOfflineBundle(cfg)
		} else {
			c.Log.Error("Failed to apply offline policy bundle: %v", err)
		}
	}
	macEngine := policy.NewMACEngine(c.Log)
	approvalManager := policy.NewApprovalManager(c.Bus, c.Log)
	c.PolicyService = NewPolicyService(policyEngine, macEngine, approvalManager)

	c.APIService = NewAPIService(8080, siemRepo, pipeline, c.SettingsService, c.AttestationService, c.Bus, c.Log)

	c.ComplianceService = NewComplianceService(c.ReportGenerator, complianceEval, c.Bus, c.Log, c.Vault, c.IdentityService, c.APIService)

	// 7. Final Wiring
	c.BroadcastService.executors = []SessionExecutor{c.LocalService, c.SSHService}
	c.SSHService.SetSnippetService(c.SnippetService)

	// 8. Registry — all services with Startup/Shutdown lifecycle
	c.Registry.Register(c.SecurityService)
	c.Registry.Register(c.HostService)
	c.Registry.Register(c.VaultService)
	c.Registry.Register(c.SSHService)
	c.Registry.Register(c.SessionService)
	c.Registry.Register(c.SettingsService)
	c.Registry.Register(c.SnippetService)
	c.Registry.Register(c.MultiExecService)
	c.Registry.Register(c.PluginService)
	c.Registry.Register(c.SIEMService)
	c.Registry.Register(c.TeamService)
	c.Registry.Register(c.LocalService)
	c.Registry.Register(c.ComplianceService)
	c.Registry.Register(c.TelemetryService)
	c.Registry.Register(c.AIService)
	c.Registry.Register(c.AlertingService)
	c.Registry.Register(c.HealthService)
	c.Registry.Register(c.MetricsService)
	c.Registry.Register(c.TunnelService)
	c.Registry.Register(c.ShareService)
	c.Registry.Register(c.RecordingService)
	c.Registry.Register(c.LogSourceService)
	c.Registry.Register(c.WorkspaceService)
	c.Registry.Register(c.NotesService)
	c.Registry.Register(c.UpdaterService)
	c.Registry.Register(c.SyncService)
	c.Registry.Register(c.FileService)
	c.Registry.Register(c.DiscoveryService)
	c.Registry.Register(c.IngestService)
	c.Registry.Register(c.IncidentService)
	c.Registry.Register(c.PlaybookService)
	c.Registry.Register(c.RansomwareEngine)
	c.Registry.Register(c.RansomwareService)
	c.Registry.Register(c.SimulationService)
	c.Registry.Register(c.CanaryService)
	c.Registry.Register(c.CanaryDeploymentService)
	c.Registry.Register(c.NetworkIsolatorService)
	c.Registry.Register(c.APIService)
	c.Registry.Register(c.ForensicsService)
	c.Registry.Register(c.ObservabilityService)
	c.Registry.Register(c.DisasterService)
	c.Registry.Register(c.GovernanceService)
	c.Registry.Register(c.AgentService)
	c.Registry.Register(c.ClusterService)
	c.Registry.Register(c.SyntheticService)
	c.Registry.Register(c.TailingService)
	c.Registry.Register(c.PolicyService)
	c.Registry.Register(c.GraphService)
	c.Registry.Register(c.NDRService)
	c.Registry.Register(c.UEBAEngine)
	c.Registry.Register(c.UEBAService)
	c.Registry.Register(c.RiskService)
	c.Registry.Register(c.TrustService)
	c.Registry.Register(c.CredentialIntel)
	c.Registry.Register(c.IdentityService)
	c.Registry.Register(c.Lifecycle)
	// TemporalService is registered once here — the early registration above has been removed.
	c.Registry.Register(c.TemporalService)
	c.Registry.Register(c.LineageService)
	c.Registry.Register(c.DecisionService)
	c.Registry.Register(c.CounterfactualService)
	c.Registry.Register(c.LedgerService)
	c.Registry.Register(c.MemorySecurity)
	c.Registry.Register(c.DeterministicResponse)

	return nil
}

func (c *Container) Close() {
	c.Log.Info("Closing application container resources...")
	if c.HotStore != nil {
		c.HotStore.Close()
	}
	if c.Bus != nil {
		c.Bus.Close()
	}
	// Database and Analytics are also handled by VaultService.Shutdown
	// but we can be defensive here.

	// Finally, close the logger to release the file handle
	if c.Log != nil {
		c.Log.Close()
	}
}
