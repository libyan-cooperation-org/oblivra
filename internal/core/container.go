package core

import (
	"context"
	"path/filepath"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/cloud"
	"github.com/kingknull/oblivrashell/internal/compliance"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/decision"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/discovery"
	"github.com/kingknull/oblivrashell/internal/enrich"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/graph"
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
	"github.com/kingknull/oblivrashell/internal/security"
	"github.com/kingknull/oblivrashell/internal/services"
	"github.com/kingknull/oblivrashell/internal/simulation"
	"github.com/kingknull/oblivrashell/internal/storage"
	"github.com/kingknull/oblivrashell/internal/team"
	"github.com/kingknull/oblivrashell/internal/temporal"
	"github.com/kingknull/oblivrashell/internal/threatintel"
	uebapkg "github.com/kingknull/oblivrashell/internal/ueba"
	"github.com/kingknull/oblivrashell/internal/updater"
	"github.com/kingknull/oblivrashell/internal/vault"
	"github.com/kingknull/oblivrashell/internal/workspace"
)

// licensePubKey is the Ed25519 public key (hex) injected at build time:
//
//	go build -ldflags "-X github.com/kingknull/oblivrashell/internal/core.licensePubKey=<hex>"
//
// Empty string = Community mode (dev builds).
var licensePubKey string

// Container holds all application dependencies and core services, grouped into logical clusters.
type Container struct {
	Log      *logger.Logger
	Registry *platform.Registry
	Kernel   *platform.Kernel
	Infra    *InfrastructureCluster
	Security *SecurityCluster
	SIEM     *SIEMCluster
	Intel    *IntelCluster
	Response *ResponseCluster
	Product  *ProductCluster
	Platform *PlatformCluster
}

// NewContainer creates a new Container instance with initialized empty clusters.
func NewContainer(log *logger.Logger, v string) *Container {
	return &Container{
		Log:      log,
		Registry: platform.NewRegistry(),
		Infra: &InfrastructureCluster{
			Platform: platform.Detect(),
		},
		Security: &SecurityCluster{},
		SIEM:     &SIEMCluster{},
		Intel:    &IntelCluster{},
		Response: &ResponseCluster{},
		Product:  &ProductCluster{},
		Platform: &PlatformCluster{},
	}
}

func (c *Container) Init(ctx context.Context) error {
	c.Log.Info("Initializing application container...")

	// 1. Infrastructure
	if err := c.initInfra(); err != nil {
		return err
	}

	// 2. Security
	if err := c.initSecurity(ctx); err != nil {
		return err
	}

	// 3. SIEM & Analytics
	if err := c.initSIEM(ctx); err != nil {
		return err
	}

	// 4. Intelligence
	if err := c.initIntel(ctx); err != nil {
		return err
	}

	// 5. Response
	if err := c.initResponse(); err != nil {
		return err
	}

	// 6. Product logic
	if err := c.initProduct(); err != nil {
		return err
	}

	// 7. Platform utility
	if err := c.initPlatform(); err != nil {
		return err
	}

	// 8. Register all services for lifecycle management
	c.registerServices()

	return nil
}

func (c *Container) initInfra() error {
	c.Infra.Log = c.Log
	c.Infra.Registry = c.Registry
	c.Infra.Bus = eventbus.NewBus(c.Log)
	c.Infra.DB = &database.Database{}

	v, err := vault.New(vault.Config{
		StorePath: platform.ConfigDir(),
		Platform:  c.Infra.Platform,
	}, c.Log)
	if err != nil {
		return err
	}
	c.Infra.Vault = v

	c.Infra.TelemetryManager = monitoring.NewTelemetryManager()
	c.Infra.HealthChecker = monitoring.NewHealthChecker(60 * time.Second)
	c.Infra.MetricsCollector = monitoring.NewMetricsCollector()
	c.Infra.AnalyticsEngine = analytics.NewAnalyticsEngine(c.Log)
	c.Infra.Notifier = notifications.NewNotificationService(c.Log)
	c.Infra.MatchEngine = threatintel.NewMatchEngine(c.Log)
	c.Infra.Enrichment = enrich.NewPipeline(c.Infra.Bus, c.Log)

	hotStore, err := storage.NewHotStore(platform.DataDir(), c.Log)
	if err != nil {
		c.Log.Error("Failed to initialize BadgerDB HotStore: %v", err)
	}
	c.Infra.HotStore = hotStore

	c.Infra.CloudAssets = database.NewCloudAssetRepository(c.Infra.DB)
	c.Infra.TenantRepo = database.NewTenantRepository(c.Infra.DB)

	return nil
}

func (c *Container) initSecurity(_ context.Context) error {
	c.Security.FIDO2Manager = security.NewFIDO2Manager()
	c.Security.AttestationService = attestation.NewAttestationService()
	c.Security.MemorySecurity = services.NewMemorySecurityService(c.Log)

	userRepo := database.NewUserRepository(c.Infra.DB)
	roleRepo := database.NewRoleRepository(c.Infra.DB)
	connectorRepo := database.NewIdentityConnectorRepository(c.Infra.DB, c.Infra.Vault)
	reportRepo := database.NewReportRepository(c.Infra.DB)
	rbacEngine := auth.NewRBACEngine(c.Log)

	var hwProvider auth.HardwareRootedIdentity
	if tpm, err := security.NewTPMManager(c.Log); err == nil {
		hwProvider = auth.NewTPMIdentityProvider(tpm, 0x81000001)
	}

	c.Security.IdentityService = services.NewIdentityService(userRepo, roleRepo, connectorRepo, rbacEngine, hwProvider, c.Infra.Bus, c.Log)
	c.Security.IdentitySyncService = services.NewIdentitySyncService(connectorRepo, c.Security.IdentityService, c.Infra.TenantRepo, c.Log)
	
	// ReportService depends on AnalyticsService which is initialized in initSIEM/initIntel
	// So we proxy the repo here and finish wiring in initIntel
	c.Security.ReportService = services.NewReportService(reportRepo, nil, c.Infra.TenantRepo, c.Log)
	
	c.Security.SecurityService = services.NewSecurityService(c.Security.FIDO2Manager, nil, nil, c.Infra.Bus, c.Log)

	policyEngine := policy.NewEngine(policy.Config{ActiveTier: policy.TierEnterprise}, c.Log)
	macEngine := policy.NewMACEngine(c.Log)
	approvalManager := policy.NewApprovalManager(c.Infra.Bus, c.Log)
	c.Security.PolicyService = services.NewPolicyService(policyEngine, macEngine, approvalManager)

	c.Security.TrustService = services.NewRuntimeTrustService(c.Infra.Bus, c.Log, c.Security.AttestationService, c.Infra.Vault)
	c.Security.GovernanceService = services.NewGovernanceService(c.Infra.Bus, c.Log)
	c.Security.CredentialIntel = services.NewCredentialIntelService(c.Infra.Bus, c.Log)

	c.Security.CanaryService = security.NewCanaryService(c.Log, nil) // SSH wired later in initProduct
	c.Security.CanaryService.SetBus(c.Infra.Bus)
	c.Security.CanaryDeployment = services.NewCanaryDeploymentService(c.Security.CanaryService, nil, c.Infra.Bus, c.Log)
	c.Security.Sentinel = services.NewSentinel(c.Infra.Vault, c.Log)

	return nil
}

func (c *Container) initSIEM(_ context.Context) error {
	var siemRepo database.SIEMStore
	if c.Infra.HotStore != nil {
		siemRepo = storage.NewBadgerSIEMRepository(c.Infra.HotStore, &c.Infra.SearchEngine, c.Infra.DB)
	}

	wal, err := storage.NewWAL(platform.DataDir(), c.Log)
	if err != nil {
		return err
	}

	temporalEngine := temporal.NewIntegrityService(temporal.DefaultPolicy(), c.Infra.Bus, c.Log)
	c.SIEM.TemporalEngine = temporalEngine
	pipeline := ingest.NewPartitionedPipeline(100000, wal, c.Infra.AnalyticsEngine, siemRepo, c.Infra.Bus, c.Log, temporalEngine, c.Infra.MetricsCollector)

	c.SIEM.IngestService = services.NewIngestService(pipeline, ingest.NewSyslogServer(pipeline, 1514, c.Log), ingest.NewAgentServer(pipeline, 8443, "", "", "", c.Log), c.Infra.Bus, c.Log)
	c.SIEM.SIEMService = services.NewSIEMService(siemRepo, security.NewSIEMForwarder(security.SIEMConfig{}, c.Log), nil, nil, nil, c.Infra.Bus, c.Log)
	
	rulesDir := filepath.Join(platform.DataDir(), "rules")
	evaluator, _ := detection.NewEvaluator(rulesDir, c.Log)
	c.SIEM.AlertingService = services.NewAlertingService(nil, c.Infra.Notifier, c.Infra.AnalyticsEngine, siemRepo, nil, evaluator, c.Infra.Bus, c.Log)

	// Inject the rule engine and identity resolver into the pipeline shards for parallel detection/enrichment
	pipeline.SetEvaluator(evaluator)
	pipeline.SetIdentityResolver(c.Security.IdentityService)
	
	c.SIEM.NDRService = services.NewNDRService(ndr.NewFlowCollector(c.Infra.Bus, c.Log), c.Infra.Bus, c.Log)
	c.SIEM.UEBAService = services.NewUEBAService(uebapkg.NewUEBAService(database.NewHostRepository(c.Infra.DB, c.Infra.Vault), c.Infra.Bus, c.Infra.HotStore, c.Log), c.Infra.Bus, c.Log)
	c.SIEM.ForensicsService = services.NewForensicsService(database.NewEvidenceRepository(c.Infra.DB), c.Infra.Vault, c.Infra.Bus, c.Log)
	c.SIEM.SourceManager = logsources.NewSourceManager(c.Log)
	c.SIEM.LogSourceService = services.NewLogSourceService(c.SIEM.SourceManager, c.Infra.AnalyticsEngine, c.Infra.Bus, c.Log)
	c.SIEM.AgentService = services.NewAgentService(nil, c.Log)
	c.SIEM.FusionEngine = detection.NewAttackFusionEngine(c.Infra.Bus, c.Log)
	c.SIEM.FusionService = services.NewFusionService(c.SIEM.FusionEngine, c.Infra.Bus, c.Log)
	c.SIEM.AnalyticsService = services.NewAnalyticsService(c.Infra.AnalyticsEngine, c.Infra.MatchEngine, c.Infra.HotStore)

	return nil
}

func (c *Container) initIntel(_ context.Context) error {
	c.Intel.AnalyticsService = services.NewAnalyticsService(c.Infra.AnalyticsEngine, c.Infra.MatchEngine, c.Infra.HotStore)
	c.Intel.TemporalService = services.NewTemporalService(nil, c.Infra.Bus, c.Log)
	c.Intel.GraphEngine = graph.NewGraphEngine(c.Infra.Bus, c.Log)
	c.Intel.GraphService = services.NewGraphService(c.Intel.GraphEngine, c.Log)
	
	hostRepo := database.NewHostRepository(c.Infra.DB, c.Infra.Vault)
	userRepo := database.NewUserRepository(c.Infra.DB)
	c.Intel.RiskEngine = risk.NewRiskEngine(c.Infra.Bus, c.Infra.DB, hostRepo, userRepo, c.Log)
	c.Intel.RiskService = services.NewRiskService(c.Intel.RiskEngine, c.Infra.Bus, c.Log)
	c.Intel.DecisionService = services.NewDecisionService(decision.NewDecisionEngine(c.Infra.Bus, c.Log), c.Log)
	
	// Wire AnalyticsService to ReportService
	if c.Security.ReportService != nil {
		// We use the already populated repo from initSecurity
		c.Security.ReportService = services.NewReportService(
			database.NewReportRepository(c.Infra.DB), 
			c.Intel.AnalyticsService, 
			c.Infra.TenantRepo, 
			c.Log,
		)
	}

	c.Intel.CounterfactualService = services.NewCounterfactualService(nil, nil, c.Log)
	c.Intel.LineageService = services.NewLineageService(lineage.NewLineageEngine(c.Infra.Bus, c.Log), c.Infra.Bus, c.Log)
	c.Intel.DashboardService = services.NewDashboardService(database.NewDashboardRepository(c.Infra.DB), c.Intel.AnalyticsService, c.Log)
	c.Intel.AssetIntelService = services.NewAssetIntelService(hostRepo, userRepo, c.Log)

	// CampaignBuilder: bridges graph edge events into the AttackFusionEngine.
	// This closes the audit gap: campaigns are now built from graph relationships
	// (entity clusters, edge types mapped to ATT&CK tactics) rather than flat logs.
	// Requires: GraphEngine (above) + FusionEngine (built in initSIEM, step 3).
	if c.SIEM.FusionEngine != nil {
		c.Intel.CampaignBuilder = detection.NewCampaignBuilder(
			c.SIEM.FusionEngine,
			c.Intel.GraphEngine,
			c.Infra.Bus,
			c.Log,
		)
		c.Log.Info("[CONTAINER] CampaignBuilder wired: graph edges → fusion engine")
	}

	return nil
}

func (c *Container) initResponse() error {
	c.Response.IncidentService = services.NewIncidentService(c.Infra.DB, nil, nil, c.Intel.RiskEngine, c.Infra.Bus, c.Log)
	c.Response.PlaybookService = services.NewPlaybookService(nil)
	c.Response.NetworkIsolatorService = services.NewNetworkIsolatorService(nil, nil, c.Infra.Bus, c.Log)
	c.Response.RansomwareService = services.NewRansomwareService(nil, c.Infra.Bus, c.Log)
	c.Response.DeterministicResponse = services.NewDeterministicResponseService(decision.NewDeterministicExecutor(c.Infra.Bus, c.Log), c.Log)
	c.Response.SimulationService = simulation.NewSimulationService(c.Infra.Bus, c.Log)
	c.Response.LedgerService = services.NewLedgerService()

	return nil
}

func (c *Container) initProduct() error {
	hostRepo := database.NewHostRepository(c.Infra.DB, c.Infra.Vault)
	sessRepo := database.NewSessionRepository(c.Infra.DB)
	credRepo := database.NewCredentialRepository(c.Infra.DB)
	
	c.Product.HostService = services.NewHostService(c.Infra.DB, c.Infra.Vault, hostRepo, c.Infra.Bus, c.Log)
	c.Product.VaultService = services.NewVaultService(c.Infra.Vault, c.Infra.DB, c.Infra.AnalyticsEngine, &c.Infra.SearchEngine, credRepo, nil, c.Security.FIDO2Manager, c.Infra.Bus, c.Log)
	c.Product.SessionService = services.NewSessionService(sessRepo, nil, c.Infra.Bus, c.Log)
	
	c.Product.SSHService = services.NewSSHService(c.Infra.DB, c.Infra.Vault, hostRepo, sessRepo, credRepo, nil, c.Infra.Bus, c.Log, nil, nil, c.Infra.TelemetryManager, nil, security.NewShellSanitizer(), nil)
	c.Product.TransferManager = services.NewTransferManager(c.Product.SSHService, c.Infra.Bus, c.Log)
	c.Product.SSHService.SetTransferManager(c.Product.TransferManager)
	
	c.Product.SettingsService = services.NewSettingsService(c.Infra.DB, c.Infra.Bus, c.Log, nil)
	c.Product.SnippetService = services.NewSnippetService(database.NewSnippetRepository(c.Infra.DB), c.Product.SSHService, nil, c.Infra.Bus, c.Log)
	c.Product.MultiExecService = services.NewMultiExecService(hostRepo, c.Infra.Vault, c.Infra.Bus, c.Log)
	c.Product.FileService = services.NewFileService(nil, c.Product.SSHService, c.Log)
	c.Product.WorkspaceService = services.NewWorkspaceService(workspace.NewWorkspaceManager(database.NewWorkspaceRepository(c.Infra.DB)), c.Infra.Bus, c.Log)
	c.Product.NotesService = services.NewNotesService(notes.NewNotesManager(), c.Infra.Bus, c.Log)
	c.Product.ShareService = services.NewShareService(nil, nil, c.Infra.Bus, c.Log)
	c.Product.RecordingService = services.NewRecordingService(nil, c.Infra.Bus, c.Log)
	complianceGen := compliance.NewReportGenerator(database.NewAuditRepository(c.Infra.DB), sessRepo, hostRepo)
	complianceEval, _ := compliance.NewEvaluator()
	c.Product.ComplianceService = services.NewComplianceService(complianceGen, complianceEval, c.Infra.Bus, c.Log, c.Infra.Vault, c.Security.IdentityService, nil)
	c.Product.TailingService = services.NewTailingService(c.Infra.Bus, c.Log)
	c.Product.TeamService = services.NewTeamService(team.NewTeamVault("Org Vault"), c.Infra.Bus, c.Log)

	// Phase 23: Terminal UX Services
	c.Product.BookmarkService = services.NewBookmarkService(hostRepo, credRepo, c.Infra.Vault, c.Infra.Bus, c.Log)
	c.Product.CommandHistory = services.NewCommandHistoryService(c.Infra.DB, c.Log)
	if c.SIEM != nil && c.SIEM.SIEMService != nil {
		c.Product.OperatorService = services.NewOperatorService(c.SIEM.SIEMService.Store(), hostRepo, c.Log)
	} else {
		// Fallback if SIEM not ready
		c.Product.OperatorService = services.NewOperatorService(nil, hostRepo, c.Log)
	}
	c.Product.SessionPersistence = services.NewSessionPersistence(c.Log)

	// Wire Terminal UX services into SSHService
	c.Product.SSHService.SetCommandHistory(c.Product.CommandHistory)
	c.Product.SSHService.SetSessionPersistence(c.Product.SessionPersistence)

	return nil
}

func (c *Container) initPlatform() error {
	c.Platform.HealthService = services.NewHealthService(c.Log, c.Infra.Bus, c.Infra.HealthChecker, c.Registry)
	c.Platform.MetricsService = services.NewMetricsService(c.Log, c.Infra.MetricsCollector)
	c.Platform.AIService = services.NewAIService(c.Infra.Vault, c.Infra.Bus, c.Log)
	c.Platform.DiscoveryService = services.NewDiscoveryService(c.Log, discovery.NewDiscoveryManager())
	c.Platform.UpdaterService = services.NewUpdaterService(c.Log, updater.NewUpdater("", "", c.Log))
	c.Platform.SyncService = services.NewSyncService(nil, c.Infra.Bus, c.Log)
	c.Platform.TunnelService = services.NewTunnelService(c.Infra.Bus, c.Log)
	c.Platform.PluginService = services.NewPluginService(c.Infra.Bus, c.Log)
	c.Platform.LocalService = services.NewLocalService(c.Infra.Bus, c.Log, c.Product.SessionService, nil)
	c.Platform.LocalService.SetCommandHistory(c.Product.CommandHistory)
	c.Platform.SyntheticService = services.NewSyntheticService(monitoring.NewSyntheticManager(c.Log), c.Log)
	c.Platform.BroadcastService = services.NewBroadcastService(nil, c.Log)
	c.Platform.ResourceMonitor = services.NewResourceMonitor(c.Log, 1024) // 1GB limit before pressure signaling
	c.Platform.ObservabilityService = services.NewObservabilityService(c.Infra.Bus, c.Infra.MetricsCollector, c.Infra.HotStore, c.Log)
	c.Platform.DisasterService = services.NewDisasterService(platform.DataDir(), c.Infra.Vault, c.Infra.Bus, c.Log)
	c.Platform.TelemetryService = services.NewTelemetryService(c.Log, c.Infra.TelemetryManager)
	c.Platform.DataLifecycleService = services.NewDataLifecycleService(c.Infra.DB, c.Infra.Bus, c.Log)
	c.Platform.APIService = services.NewAPIService(8080, c.Infra.DB, c.SIEM.SIEMService.Store(), c.SIEM.IngestService.Pipeline(), c.Product.SettingsService, c.Security.IdentityService, c.Security.ReportService, c.Intel.DashboardService, c.Security.AttestationService, c.Infra.Bus, c.Log, c.Response.NetworkIsolatorService, c.Infra.MatchEngine, c.SIEM.TemporalEngine)

	// DiagnosticsService: wire bus dropped counter from the event bus.
	busDropped := func() uint64 {
		return c.Infra.Bus.DroppedCount()
	}
	c.Platform.DiagnosticsService = services.NewDiagnosticsService(c.Infra.Bus, c.Log, busDropped)

	// Wire the ingest pipeline to push live EPS stats to DiagnosticsService.
	// Both are initialised by this point (SIEM before Platform).
	if c.SIEM.IngestService != nil && c.Platform.DiagnosticsService != nil {
		c.SIEM.IngestService.Pipeline().SetDiagnosticsUpdater(c.Platform.DiagnosticsService)
	}

	// Wire the Graph Engine into the ingest pipeline.
	// This inserts the EntityExtractor DAG node so every event automatically
	// populates the entity graph. Wired here because both Intel (graph) and
	// SIEM (pipeline) are fully initialised by the time initPlatform runs.
	if c.SIEM.IngestService != nil && c.Intel.GraphEngine != nil {
		c.SIEM.IngestService.Pipeline().SetGraphEngine(c.Intel.GraphEngine)
		c.Log.Info("[CONTAINER] Graph engine wired into ingest pipeline (entity extraction active)")
	}

	// LicensingService — pubKeyHex injected at build time via ldflags.
	// In dev/community builds the key is empty; the manager defaults to Community tier.
	if c.Product.SettingsService != nil {
		c.Platform.LicensingService = services.NewLicensingService(
			licensePubKey, // var declared in main.go via -ldflags
			c.Infra.Bus,
			c.Log,
			func(k string) (string, error) { return c.Product.SettingsService.Get(k) },
			func(k, v string) error { return c.Product.SettingsService.Set(k, v) },
		)
	}

	c.Platform.CloudDiscovery = cloud.NewCloudDiscoveryManager(c.Infra.CloudAssets, c.Log)

	return nil
}

// mustRegister registers a service and panics on duplicate or nil service.
// Duplicate registration is a programming error in Init, not a recoverable runtime condition.
func (c *Container) mustRegister(s platform.Service) {
	if s == nil {
		// Skip nil services — some are optional and only wired when hardware is present.
		return
	}
	if err := c.Registry.Register(s); err != nil {
		panic("container: " + err.Error())
	}
}

func (c *Container) registerServices() {
	// Infrastructure (early)
	c.mustRegister(c.Infra.Vault)
	
	// Security
	c.mustRegister(c.Security.SecurityService)
	c.mustRegister(c.Security.IdentityService)
	c.mustRegister(c.Security.IdentitySyncService)
	c.mustRegister(c.Security.ReportService)
	c.mustRegister(c.Security.PolicyService)
	c.mustRegister(c.Security.TrustService)
	c.mustRegister(c.Security.CanaryDeployment)
	c.mustRegister(c.Security.MemorySecurity)
	c.mustRegister(c.Security.GovernanceService)
	c.mustRegister(c.Security.CredentialIntel)
	c.mustRegister(c.Security.Sentinel)

	// SIEM
	c.mustRegister(c.SIEM.SIEMService)
	c.mustRegister(c.SIEM.IngestService)
	c.mustRegister(c.SIEM.AlertingService)
	c.mustRegister(c.SIEM.NDRService)
	c.mustRegister(c.SIEM.UEBAService)
	c.mustRegister(c.SIEM.ForensicsService)
	c.mustRegister(c.SIEM.LogSourceService)
	c.mustRegister(c.SIEM.AgentService)
	c.mustRegister(c.SIEM.FusionService)

	// Product
	c.mustRegister(c.Product.HostService)
	c.mustRegister(c.Product.VaultService)
	c.mustRegister(c.Product.SSHService)
	c.mustRegister(c.Product.SessionService)
	c.mustRegister(c.Product.SettingsService)
	c.mustRegister(c.Product.SnippetService)
	c.mustRegister(c.Product.MultiExecService)
	c.mustRegister(c.Product.FileService)
	c.mustRegister(c.Product.WorkspaceService)
	c.mustRegister(c.Product.NotesService)
	c.mustRegister(c.Product.ShareService)
	c.mustRegister(c.Product.RecordingService)
	c.mustRegister(c.Product.ComplianceService)
	c.mustRegister(c.Product.TailingService)
	c.mustRegister(c.Product.TeamService)
	c.mustRegister(c.Product.BookmarkService)
	c.mustRegister(c.Product.CommandHistory)
	c.mustRegister(c.Product.OperatorService)

	// Platform
	c.mustRegister(c.Platform.HealthService)
	c.mustRegister(c.Platform.MetricsService)
	c.mustRegister(c.Platform.AIService)
	c.mustRegister(c.Platform.DiscoveryService)
	c.mustRegister(c.Platform.UpdaterService)
	c.mustRegister(c.Platform.SyncService)
	c.mustRegister(c.Platform.TunnelService)
	c.mustRegister(c.Platform.PluginService)
	c.mustRegister(c.Platform.LocalService)
	c.mustRegister(c.Platform.SyntheticService)
	c.mustRegister(c.Platform.BroadcastService)
	c.mustRegister(c.Platform.TelemetryService)
	c.mustRegister(c.Platform.DataLifecycleService)
	c.mustRegister(c.Platform.DisasterService)
	c.mustRegister(c.Platform.ResourceMonitor)
	c.mustRegister(c.Platform.ObservabilityService)
	c.mustRegister(c.Platform.DiagnosticsService)
	c.mustRegister(c.Platform.APIService)
	c.mustRegister(c.Platform.LicensingService)

	// Intel
	c.mustRegister(c.Intel.AnalyticsService)
	c.mustRegister(c.Intel.RiskService)
	c.mustRegister(c.Intel.GraphService)
	c.mustRegister(c.Intel.DecisionService)
	c.mustRegister(c.Intel.CounterfactualService)
	c.mustRegister(c.Intel.LineageService)
	c.mustRegister(c.Intel.TemporalService)
	c.mustRegister(c.Intel.DashboardService)
	c.mustRegister(c.Intel.AssetIntelService)

	// Response
	c.mustRegister(c.Response.IncidentService)
	c.mustRegister(c.Response.PlaybookService)
	c.mustRegister(c.Response.NetworkIsolatorService)
	c.mustRegister(c.Response.DeterministicResponse)
	c.mustRegister(c.Response.LedgerService)
	c.mustRegister(c.Response.RansomwareService)
}

func (c *Container) Close() {
	c.Log.Info("Closing application container resources...")
	if c.Infra.HotStore != nil {
		c.Infra.HotStore.Close()
	}
	if c.Infra.Bus != nil {
		c.Infra.Bus.Close()
	}
	// Database and Analytics are also handled by VaultService.Shutdown
	// but we can be defensive here.

	// Finally, close the logger to release the file handle
	if c.Log != nil {
		c.Log.Close()
	}
}
