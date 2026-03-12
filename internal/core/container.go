package core

import (
	"context"
	"path/filepath"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/auth"
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

	return nil
}

func (c *Container) initSecurity(_ context.Context) error {
	c.Security.FIDO2Manager = security.NewFIDO2Manager()
	c.Security.AttestationService = attestation.NewAttestationService()
	c.Security.MemorySecurity = services.NewMemorySecurityService(c.Log)

	userRepo := database.NewUserRepository(c.Infra.DB)
	roleRepo := database.NewRoleRepository(c.Infra.DB)
	rbacEngine := auth.NewRBACEngine(c.Log)

	var hwProvider auth.HardwareRootedIdentity
	if tpm, err := security.NewTPMManager(c.Log); err == nil {
		hwProvider = auth.NewTPMIdentityProvider(tpm, 0x81000001)
	}

	c.Security.IdentityService = services.NewIdentityService(userRepo, roleRepo, rbacEngine, hwProvider, c.Infra.Bus, c.Log)
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
	pipeline := ingest.NewPipeline(100000, wal, c.Infra.AnalyticsEngine, siemRepo, c.Infra.Bus, c.Log, temporalEngine)

	c.SIEM.IngestService = services.NewIngestService(pipeline, ingest.NewSyslogServer(pipeline, 1514, c.Log), ingest.NewAgentServer(pipeline, 8443, "", "", "", c.Log), c.Infra.Bus, c.Log)
	c.SIEM.SIEMService = services.NewSIEMService(siemRepo, security.NewSIEMForwarder(security.SIEMConfig{}, c.Log), nil, nil, nil, c.Infra.Bus, c.Log)
	
	rulesDir := filepath.Join(platform.DataDir(), "rules")
	evaluator, _ := detection.NewEvaluator(rulesDir, c.Log)
	c.SIEM.AlertingService = services.NewAlertingService(nil, c.Infra.Notifier, c.Infra.AnalyticsEngine, siemRepo, nil, evaluator, c.Infra.Bus, c.Log)
	
	c.SIEM.NDRService = services.NewNDRService(ndr.NewFlowCollector(c.Infra.Bus, c.Log), c.Infra.Bus, c.Log)
	c.SIEM.UEBAService = services.NewUEBAService(uebapkg.NewUEBAService(database.NewHostRepository(c.Infra.DB, c.Infra.Vault), c.Infra.Bus, c.Infra.HotStore, c.Log), c.Infra.Bus, c.Log)
	c.SIEM.ForensicsService = services.NewForensicsService(database.NewEvidenceRepository(c.Infra.DB), c.Infra.Vault, c.Infra.Bus, c.Log)
	c.SIEM.SourceManager = logsources.NewSourceManager(c.Log)
	c.SIEM.LogSourceService = services.NewLogSourceService(c.SIEM.SourceManager, c.Infra.AnalyticsEngine, c.Infra.Bus, c.Log)
	c.SIEM.AgentService = services.NewAgentService(nil, c.Log)
	c.SIEM.TailingService = services.NewTailingService(c.Infra.Bus, c.Log)

	return nil
}

func (c *Container) initIntel(_ context.Context) error {
	c.Intel.AnalyticsService = services.NewAnalyticsService(c.Infra.AnalyticsEngine)
	c.Intel.TemporalService = services.NewTemporalService(nil, c.Infra.Bus, c.Log)
	c.Intel.GraphEngine = graph.NewGraphEngine(c.Infra.Bus, c.Log)
	c.Intel.GraphService = services.NewGraphService(c.Intel.GraphEngine, c.Log)
	c.Intel.RiskEngine = risk.NewRiskEngine(c.Infra.Bus, c.Infra.DB, nil, c.Log)
	c.Intel.RiskService = services.NewRiskService(c.Intel.RiskEngine, c.Infra.Bus, c.Log)
	c.Intel.DecisionService = services.NewDecisionService(decision.NewDecisionEngine(c.Infra.Bus, c.Log), c.Log)
	c.Intel.CounterfactualService = services.NewCounterfactualService(nil, nil, c.Log)
	c.Intel.LineageService = services.NewLineageService(lineage.NewLineageEngine(c.Infra.Bus, c.Log), c.Infra.Bus, c.Log)

	return nil
}

func (c *Container) initResponse() error {
	c.Response.IncidentService = services.NewIncidentService(c.Infra.DB, nil, nil, c.Infra.Bus, c.Log)
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
	c.Platform.SyntheticService = services.NewSyntheticService(nil, c.Log)
	c.Platform.BroadcastService = services.NewBroadcastService(nil, c.Log)
	c.Platform.ResourceMonitor = services.NewResourceMonitor(c.Log, 1024) // 1GB limit before pressure signaling
	c.Platform.ObservabilityService = services.NewObservabilityService(c.Infra.Bus, c.Infra.MetricsCollector, c.Infra.HotStore, c.Log)
	c.Platform.DisasterService = services.NewDisasterService(platform.DataDir(), c.Infra.Vault, c.Infra.Bus, c.Log)
	c.Platform.TelemetryService = services.NewTelemetryService(c.Log, c.Infra.TelemetryManager)
	c.Platform.DataLifecycleService = services.NewDataLifecycleService(c.Infra.DB, c.Infra.Bus, c.Log)

	return nil
}

func (c *Container) registerServices() {
	// Infrastructure (early)
	c.Registry.Register(c.Infra.Vault)
	
	// Security
	c.Registry.Register(c.Security.SecurityService)
	c.Registry.Register(c.Security.IdentityService)
	c.Registry.Register(c.Security.PolicyService)
	c.Registry.Register(c.Security.TrustService)
	c.Registry.Register(c.Security.CanaryDeployment)
	c.Registry.Register(c.Security.MemorySecurity)
	c.Registry.Register(c.Security.GovernanceService)
	c.Registry.Register(c.Security.CredentialIntel)
	c.Registry.Register(c.Security.Sentinel)
	
	// SIEM
	c.Registry.Register(c.SIEM.SIEMService)
	c.Registry.Register(c.SIEM.IngestService)
	c.Registry.Register(c.SIEM.AlertingService)
	c.Registry.Register(c.SIEM.NDRService)
	c.Registry.Register(c.SIEM.UEBAService)
	c.Registry.Register(c.SIEM.ForensicsService)
	c.Registry.Register(c.SIEM.LogSourceService)
	c.Registry.Register(c.SIEM.AgentService)
	c.Registry.Register(c.SIEM.TailingService)
	
	// Product
	c.Registry.Register(c.Product.HostService)
	c.Registry.Register(c.Product.VaultService)
	c.Registry.Register(c.Product.SSHService)
	c.Registry.Register(c.Product.SessionService)
	c.Registry.Register(c.Product.SettingsService)
	c.Registry.Register(c.Product.SnippetService)
	c.Registry.Register(c.Product.MultiExecService)
	c.Registry.Register(c.Product.FileService)
	c.Registry.Register(c.Product.WorkspaceService)
	c.Registry.Register(c.Product.NotesService)
	c.Registry.Register(c.Product.ShareService)
	c.Registry.Register(c.Product.RecordingService)
	c.Registry.Register(c.Product.ComplianceService)
	c.Registry.Register(c.Product.TailingService)
	c.Registry.Register(c.Product.TeamService)
	
	// Platform
	c.Registry.Register(c.Platform.HealthService)
	c.Registry.Register(c.Platform.MetricsService)
	c.Registry.Register(c.Platform.AIService)
	c.Registry.Register(c.Platform.DiscoveryService)
	c.Registry.Register(c.Platform.UpdaterService)
	c.Registry.Register(c.Platform.SyncService)
	c.Registry.Register(c.Platform.TunnelService)
	c.Registry.Register(c.Platform.PluginService)
	c.Registry.Register(c.Platform.LocalService)
	c.Registry.Register(c.Platform.SyntheticService)
	c.Registry.Register(c.Platform.BroadcastService)
	c.Registry.Register(c.Platform.TelemetryService)
	c.Registry.Register(c.Platform.DataLifecycleService)
	c.Registry.Register(c.Platform.DisasterService)
	c.Registry.Register(c.Platform.ResourceMonitor)

	// Intel
	c.Registry.Register(c.Intel.AnalyticsService)
	c.Registry.Register(c.Intel.RiskService)
	c.Registry.Register(c.Intel.GraphService)
	c.Registry.Register(c.Intel.DecisionService)
	c.Registry.Register(c.Intel.CounterfactualService)
	c.Registry.Register(c.Intel.LineageService)
	c.Registry.Register(c.Intel.TemporalService)

	// Response
	c.Registry.Register(c.Response.IncidentService)
	c.Registry.Register(c.Response.PlaybookService)
	c.Registry.Register(c.Response.NetworkIsolatorService)
	c.Registry.Register(c.Response.DeterministicResponse)
	c.Registry.Register(c.Response.LedgerService)
	c.Registry.Register(c.Response.RansomwareService)
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
