package core

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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
	"github.com/kingknull/oblivrashell/internal/incident"
	"github.com/kingknull/oblivrashell/internal/integrity"
	oblio "github.com/kingknull/oblivrashell/internal/io"
	"github.com/kingknull/oblivrashell/internal/lineage"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/logsources"
	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/ndr"
	"github.com/kingknull/oblivrashell/internal/notes"
	"github.com/kingknull/oblivrashell/internal/notifications"
	"github.com/kingknull/oblivrashell/internal/oql"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/kingknull/oblivrashell/internal/policy"
	"github.com/kingknull/oblivrashell/internal/risk"
	"github.com/kingknull/oblivrashell/internal/security"
	"github.com/kingknull/oblivrashell/internal/services"
	"github.com/kingknull/oblivrashell/internal/simulation"
	"github.com/kingknull/oblivrashell/internal/storage"
	"github.com/kingknull/oblivrashell/internal/storage/tiering"
	"github.com/kingknull/oblivrashell/internal/team"
	"github.com/kingknull/oblivrashell/internal/temporal"
	"github.com/kingknull/oblivrashell/internal/threatintel"
	uebapkg "github.com/kingknull/oblivrashell/internal/ueba"
	"github.com/kingknull/oblivrashell/internal/updater"
	"github.com/kingknull/oblivrashell/internal/messaging"
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

	// 1. Core Infrastructure
	if err := c.initInfra(ctx); err != nil {
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
	if err := c.initPlatform(ctx); err != nil {
		return err
	}

	// 8. Register all services for lifecycle management
	c.registerServices()

	return nil
}

func (c *Container) initInfra(ctx context.Context) error {
	c.Infra.Log = c.Log
	c.Infra.Registry = c.Registry
	c.Infra.Bus = eventbus.NewBus(c.Log)
	c.Infra.DB = &database.Database{}

	var v vault.Provider
	var err error

	// 2.1: Vault Process Isolation (Sovereign-Grade Hardening)
	if os.Getenv("OBLIVRA_ISOLATED_VAULT") == "true" {
		socketPath := "/tmp/oblivra-vault.sock"
		if runtime.GOOS == "windows" {
			socketPath = `\\.\pipe\oblivra-vault`
		}
		
		// 5: Automated Lifecycle Management
		if err := vault.EnsureDaemonRunning(socketPath, c.Log); err != nil {
			c.Log.Warn("[VAULT] Failed to ensure daemon is running: %v. Attempting to connect anyway.", err)
		}

		c.Log.Info("[VAULT] Using isolated vault mode (Socket: %s)", socketPath)
		v = vault.NewRemoteProvider(socketPath)
	} else {
		v, err = vault.New(vault.Config{
			StorePath: platform.ConfigDir(),
			Platform:  c.Infra.Platform,
		}, c.Log)
		if err != nil {
			return err
		}
	}
	c.Infra.Vault = v
	
	// 5. Sovereign-Grade: Persistent Vault Monitoring
	if os.Getenv("OBLIVRA_ISOLATED_VAULT") == "true" {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Use a 5s timeout for health checks
					pCtx, pCancel := context.WithTimeout(context.Background(), 5*time.Second)
					err := v.Ping(pCtx)
					pCancel()
					
					if err != nil {
						c.Log.Warn("[VAULT] Isolated daemon heartbeat failed: %v. Attempting auto-recovery...", err)
						socketPath := "/tmp/oblivra-vault.sock"
						if runtime.GOOS == "windows" {
							socketPath = `\\.\pipe\oblivra-vault`
						}
						_ = vault.EnsureDaemonRunning(socketPath, c.Log)
					}
				}
			}
		}()
	}

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

	// Phase 22.3 — Hot/Warm/Cold storage tiering.
	//
	// Wires the three-tier promotion pipeline:
	//   Hot   (BadgerDB, 0–30 d)   ← active query path, fast SSD
	//   Warm  (Parquet, 30–180 d)  ← compressed columnar, cheap SSD/HDD
	//   Cold  (JSONL, 180+ d)      ← cheapest; air-gap-safe local impl
	//
	// The Migrator runs on a 1-hour cycle (configurable). It walks the
	// hot keyspace for events older than HotDuration, copies them to
	// warm, then deletes from hot. Same dance promotes warm → cold.
	// Power-loss safe: source delete only happens after destination
	// write confirms.
	//
	// Cold tier defaults to a local directory under the platform's
	// data directory. The S3-compatible RemoteColdTier (cold_s3.go)
	// lands behind a build tag once operator credentials are plumbed
	// through Settings — air-gap deployments use the local impl.
	//
	// HotStore is required; without it tiering is a no-op (the
	// migrator's nil-tier guards mean Start() with nil hot is harmless).
	if c.Infra.HotStore != nil {
		c.Infra.HotTier = tiering.NewHotTier(c.Infra.HotStore)
		c.Infra.WarmTier = tiering.NewWarmTier(platform.DataDir(), c.Log)
		c.Infra.ColdTier = tiering.NewLocalDirCold(platform.DataDir(), c.Log)
		c.Infra.TierMigrator = tiering.NewMigrator(
			c.Infra.HotTier, c.Infra.WarmTier, c.Infra.ColdTier,
			tiering.DefaultRetention(), c.Log,
		)
		// Migrator is started later, after Startup, so the hot keyspace
		// has had a chance to receive events from the live ingest path.
		// See c.startTierMigrator(ctx) in Container.Startup.
	} else {
		c.Log.Warn("[STORAGE] Hot/Warm/Cold tiering disabled: HotStore unavailable")
	}

	// 26.1: Distributed Log Fabric (NATS)
	natsCfg := &messaging.NATSConfig{
		Port:       4222,
		DataDir:    filepath.Join(platform.DataDir(), "nats"),
		StreamName: "OBLIVRA_INGEST",
		Subjects:   []string{"oblivra.ingest.logs"},
	}
	c.Infra.Messaging = messaging.NewNATSService(natsCfg, c.Log)
	// Start messaging before other services
	if err := c.Infra.Messaging.Start(context.Background()); err != nil {
		c.Log.Warn("[MESSAGING] Failed to start distributed fabric: %v. Falling back to in-memory mode.", err)
		c.Infra.Messaging = nil
	}

	c.Infra.CloudAssets = database.NewCloudAssetRepository(c.Infra.DB)
	c.Infra.TenantRepo = database.NewTenantRepository(c.Infra.DB)
	c.Infra.RBAC = auth.NewRBACEngine(c.Log)

	return nil
}

func (c *Container) initSecurity(_ context.Context) error {
	c.Security.FIDO2Manager = security.NewFIDO2Manager()
	c.Security.QuorumManager = security.NewQuorumManager(c.Security.FIDO2Manager, c.Log)
	c.Security.AttestationService = attestation.NewAttestationService()
	c.Security.MemorySecurity = services.NewMemorySecurityService(c.Log)
	c.Security.ShellSanitizer = security.NewShellSanitizer()

	userRepo := database.NewUserRepository(c.Infra.DB)
	roleRepo := database.NewRoleRepository(c.Infra.DB)
	connectorRepo := database.NewIdentityConnectorRepository(c.Infra.DB, c.Infra.Vault)
	reportRepo := database.NewReportRepository(c.Infra.DB)

	var hwProvider auth.HardwareRootedIdentity
	if tpm, err := security.NewTPMManager(c.Log); err == nil {
		hwProvider = auth.NewTPMIdentityProvider(tpm, 0x81000001)
	}

	c.Security.IdentityService = services.NewIdentityService(userRepo, roleRepo, connectorRepo, c.Infra.RBAC, hwProvider, c.Infra.Bus, c.Log)
	c.Security.IdentitySyncService = services.NewIdentitySyncService(connectorRepo, c.Security.IdentityService, c.Infra.TenantRepo, c.Log)

	// Audit fix #4 — ReportService construction is DEFERRED until
	// initIntel because it needs AnalyticsService which doesn't
	// exist yet at this point. Previously we constructed it here
	// with `nil` analytics, then unconditionally re-constructed it
	// in initIntel — wasteful + confusing about which instance is
	// live. We just leave the field nil here; initIntel populates it.
	//
	// Report Factory and Scheduler don't need AnalyticsService and
	// can stay here.
	c.Security.ReportFactory = analytics.NewReportFactory(reportRepo, oql.NewExecutor(), c.Infra.HotStore)
	c.Security.ReportScheduler = analytics.NewReportScheduler(reportRepo, c.Security.ReportFactory, c.Log)
	
	c.Security.SecurityService = services.NewSecurityService(c.Security.FIDO2Manager, nil, nil, c.Security.QuorumManager, c.Infra.Bus, c.Log)

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
	
	suppressionRepo := database.NewSuppressionRepository(c.Infra.DB)
	c.Security.SuppressionService = services.NewSuppressionService(suppressionRepo, c.Log)

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

	merkleTree := integrity.NewWithPersistence(func(index int, hash string, data []byte) error {
		// Log the persistence event if needed or write to BadgerDB here.
		return nil
	})
	pipeline.SetIntegrityTree(merkleTree)

	certFile := filepath.Join(platform.ConfigDir(), "cert.pem")
	keyFile := filepath.Join(platform.ConfigDir(), "key.pem")

	// Ensure certificates exist for the agent server
	if err := security.EnsureLocalCerts(certFile, keyFile); err != nil {
		c.Log.Warn("[CONTAINER] Failed to ensure local certificates: %v — ingestion may fail", err)
	}

	c.SIEM.IngestService = services.NewIngestService(pipeline, ingest.NewSyslogServer(pipeline, 1514, c.Log), ingest.NewAgentServer(pipeline, 8443, certFile, keyFile, "", c.Security.ShellSanitizer, c.Log), c.Infra.Bus, c.Log)
	c.SIEM.TimelineService = services.NewTimelineService(siemRepo, c.Log)
	rulesDir := filepath.Join(platform.DataDir(), "rules")
	evaluator, _ := detection.NewEvaluator(rulesDir, c.Log)
	c.SIEM.AlertingService = services.NewAlertingService(nil, c.Infra.Notifier, c.Infra.AnalyticsEngine, siemRepo, nil, evaluator, c.Infra.Bus, c.Log)
	c.SIEM.AlertingService.SetSuppressionService(c.Security.SuppressionService)

	c.SIEM.SIEMService = services.NewSIEMService(siemRepo, security.NewSIEMForwarder(security.SIEMConfig{}, c.Log), nil, nil, nil, c.Infra.RBAC, c.Infra.Bus, c.Log, pipeline, c.SIEM.TimelineService, c.Infra.Federator, c.Infra.HotStore)

	// Inject the rule engine and identity resolver into the pipeline shards for parallel detection/enrichment
	pipeline.SetEvaluator(evaluator)
	pipeline.SetIdentityResolver(c.Security.IdentityService)
	if c.Infra.Messaging != nil {
		pipeline.SetNATSService(c.Infra.Messaging)
	}
	
	c.SIEM.NDRService = services.NewNDRService(ndr.NewFlowCollector(c.Infra.Bus, c.Log), c.Infra.Bus, c.Log)
	c.SIEM.UEBAService = services.NewUEBAService(uebapkg.NewUEBAService(database.NewHostRepository(c.Infra.DB, c.Infra.Vault), c.Infra.Bus, c.Infra.HotStore, c.Log), c.Infra.Bus, c.Log)
	c.SIEM.ForensicsService = services.NewForensicsService(database.NewEvidenceRepository(c.Infra.DB), c.Infra.DB, c.Infra.Vault, c.Infra.RBAC, c.Infra.Bus, c.Log)
	c.SIEM.SourceManager = logsources.NewSourceManager(c.Log)
	c.SIEM.LogSourceService = services.NewLogSourceService(c.SIEM.SourceManager, c.Infra.AnalyticsEngine, c.Infra.Bus, c.Security.ShellSanitizer, c.Log)
	c.SIEM.AgentService = services.NewAgentService(c.SIEM.IngestService.AgentServer(), c.Infra.RBAC, c.Log)
	c.SIEM.FusionEngine = detection.NewAttackFusionEngine(c.Infra.Bus, c.Log)
	c.SIEM.FusionService = services.NewFusionService(c.SIEM.FusionEngine, c.Infra.AnalyticsEngine, c.Infra.Bus, c.Log)
	c.SIEM.AnalyticsService = services.NewAnalyticsService(c.Infra.AnalyticsEngine, c.Infra.MatchEngine, c.Infra.HotStore)

	return nil
}

func (c *Container) initIntel(_ context.Context) error {
	c.Intel.AnalyticsService = services.NewAnalyticsService(c.Infra.AnalyticsEngine, c.Infra.MatchEngine, c.Infra.HotStore)
	c.Intel.TemporalService = services.NewTemporalService(nil, c.Infra.Bus, c.Log)
	c.Intel.GraphEngine = graph.NewGraphEngine(c.Infra.Bus, c.Log)
	c.Intel.GraphService = services.NewGraphService(c.Intel.GraphEngine, c.Log)
	c.Intel.GraphService.SetSnapshotPath(filepath.Join(platform.DataDir(), "graph.snapshot.json"))
	
	hostRepo := database.NewHostRepository(c.Infra.DB, c.Infra.Vault)
	userRepo := database.NewUserRepository(c.Infra.DB)
	c.Intel.RiskEngine = risk.NewRiskEngine(c.Infra.Bus, c.Infra.DB, hostRepo, userRepo, c.Log)
	c.Intel.RiskService = services.NewRiskService(c.Intel.RiskEngine, c.Infra.Bus, c.Log)
	c.Intel.DecisionService = services.NewDecisionService(decision.NewDecisionEngine(c.Infra.Bus, c.Log), c.Log)
	
	// Wire AnalyticsService to ReportService.
	//
	// Audit fix #4 — initSecurity used to construct ReportService
	// with nil analytics here, and this block re-constructed it
	// with the real analytics. The previous guard (`!= nil`) was
	// only correct because of that double-construction. Now that
	// initSecurity skips the placeholder, the guard would prevent
	// us from EVER constructing ReportService — and the platform
	// Registry would then call Dependencies() on a nil receiver
	// and panic at boot. Flip the guard to construct unconditionally;
	// AnalyticsService is guaranteed non-nil at this point because
	// initIntel populates it three lines above.
	c.Security.ReportService = services.NewReportService(
		database.NewReportRepository(c.Infra.DB),
		c.Intel.AnalyticsService,
		c.Infra.TenantRepo,
		c.Log,
	)

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
	hostRepo := database.NewHostRepository(c.Infra.DB, c.Infra.Vault)
	userRepo := database.NewUserRepository(c.Infra.DB)
	c.Response.TriageService = incident.NewTriageService(hostRepo, userRepo, c.Log)

	c.Response.IncidentService = services.NewIncidentService(c.Infra.DB, nil, nil, c.Response.TriageService, c.Infra.Bus, c.Log)
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
	
	c.Product.HostService = services.NewHostService(c.Infra.DB, c.Infra.Vault, hostRepo, c.Security.IdentityService.RBAC(), c.Infra.Bus, c.Log)
	c.Product.VaultService = services.NewVaultService(c.Infra.Vault, c.Infra.DB, c.Infra.AnalyticsEngine, &c.Infra.SearchEngine, &c.Infra.Federator, credRepo, nil, c.Security.FIDO2Manager, c.Infra.RBAC, c.Infra.Bus, c.Log)
	c.Product.SessionService = services.NewSessionService(sessRepo, nil, c.Infra.Bus, c.Log)
	
	c.Product.SSHService = services.NewSSHService(c.Infra.DB, c.Infra.Vault, hostRepo, sessRepo, credRepo, nil, c.Infra.Bus, c.Log, nil, nil, c.Infra.TelemetryManager, nil, c.Security.ShellSanitizer, nil)
	c.Product.TransferManager = services.NewTransferManager(c.Product.SSHService, c.Infra.Bus, c.Log)
	c.Product.SSHService.SetTransferManager(c.Product.TransferManager)
	
	c.Product.SettingsService = services.NewSettingsService(c.Infra.DB, c.Infra.Vault, c.Infra.Bus, c.Log, nil)
	c.Product.SnippetService = services.NewSnippetService(database.NewSnippetRepository(c.Infra.DB), c.Product.SSHService, nil, c.Infra.Bus, c.Log)
	c.Product.MultiExecService = services.NewMultiExecService(hostRepo, c.Infra.Vault, c.Infra.Bus, c.Log)
	c.Product.FileService = services.NewFileService(nil, c.Product.SSHService, sessRepo, c.Log)
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

	rotationRepo := database.NewRotationRepository(c.Infra.DB)
	c.Security.RotationService = services.NewRotationService(c.Infra.DB, rotationRepo, c.Product.VaultService, c.Product.SSHService, c.Infra.Bus, c.Log)

	return nil
}

func (c *Container) initPlatform(ctx context.Context) error {
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
	c.Platform.ClusterService = services.NewClusterService(c.Infra.DB, c.Infra.Bus, c.Log, c.Infra.Federator)
	c.Platform.PlatformService = services.NewPlatformService(c.Infra.TenantRepo, c.Product.HostService.Store(), c.SIEM.SIEMService.Store(), c.Log)
	// Audit Repository (Persistence)
	auditRepo := database.NewAuditRepository(c.Infra.DB)
	
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

	c.Platform.APIService = services.NewAPIService(8080, c.Infra.DB, c.SIEM.SIEMService.Store(), auditRepo, c.SIEM.IngestService.Pipeline(), c.Intel.GraphEngine, c.SIEM.UEBAService, c.Product.ComplianceService, c.Platform.LicensingService, c.Product.VaultService, c.Product.SettingsService, c.Security.IdentityService, c.Platform.PlatformService, c.SIEM.ForensicsService, c.SIEM.FusionService, c.Security.ReportService, c.Intel.DashboardService, c.Security.AttestationService, c.Infra.Bus, c.Log, c.Response.NetworkIsolatorService, c.SIEM.AgentService, c.Infra.MatchEngine, c.SIEM.TemporalEngine)

	// Phase 32: wire SuppressionService into the REST server via setter
	// (avoids a circular import — api ← services ← api). The endpoint
	// `POST /api/v1/alerts/{id}/suppress` is gated on this provider.
	if c.Platform.APIService != nil && c.Security.SuppressionService != nil {
		c.Platform.APIService.SetSuppression(c.Security.SuppressionService)
	}

	// Phase 32: wire SettingsService into the REST server. The endpoint
	// `GET/PUT /api/v1/settings/{key}` is gated on this provider.
	if c.Platform.APIService != nil && c.Product.SettingsService != nil {
		c.Platform.APIService.SetSettings(c.Product.SettingsService)
	}

	// Phase 33 — Slice 3: TLS guardrails (warning loop + sovereignty
	// deduction + production lockout). Read tls.mode from io-config
	// when present, falling back to "on" by default.
	tlsMode := "on"
	if cfgPath := os.Getenv("OBLIVRA_IO_CONFIG"); cfgPath != "" {
		if ioCfg, err := oblio.LoadConfig(cfgPath); err == nil {
			tlsMode = ioCfg.TLS.Mode
		}
	}
	if tlsGuards, err := security.NewTLSGuardrails(tlsMode, c.Log); err != nil {
		c.Log.Error("[FATAL] %v", err)
		os.Exit(1) // production lockout — must not boot
	} else {
		tlsGuards.Start(context.Background())
		if c.Platform.APIService != nil {
			c.Platform.APIService.SetTLSState(tlsGuards)
		}
	}

	// Phase 22.3 — wire Hot/Warm/Cold storage tier observability into
	// the REST server. Without this the /api/v1/storage/tiering/*
	// endpoints return 503 (which is the correct behaviour when the
	// migrator wasn't initialised, e.g. HotStore failed to open).
	if c.Platform.APIService != nil {
		WireTieringIntoAPI(c.Platform.APIService, c.Infra)
	}

	// Phase 33 — Slice 5: io.Pipeline + IOConfigProvider for the
	// /connectors UI. The pipeline is server-side here (the agent's
	// own pipeline lives in cmd/agent/main.go); operators can stand
	// up syslog/HEC listeners on the server to ingest from network
	// gear directly without an agent.
	if c.Platform.APIService != nil {
		ioPath := os.Getenv("OBLIVRA_IO_CONFIG")
		if ioPath == "" {
			ioPath = filepath.Join(os.Getenv("PROGRAMDATA"), "oblivra", "server-io.yml")
			if runtime.GOOS != "windows" {
				ioPath = "/etc/oblivra/server-io.yml"
			}
		}
		ioProvider := services.NewIOConfigService(ioPath, c.Log)
		c.Platform.APIService.SetIOConfig(ioProvider)
		if err := ioProvider.StartPipeline(context.Background()); err != nil {
			c.Log.Warn("[io] pipeline disabled: %v", err)
		}
	}

	// Phase 33 — Tamper Path 1, Layer 3: missed-heartbeat scanner.
	// Runs in the background, sweeping every 60s for agents whose
	// last heartbeat is older than 90s. Emits tamper:detected bus
	// events that flow through the existing alert pipeline.
	if c.Platform.APIService != nil {
		c.Platform.APIService.StartHeartbeatScanner(context.Background())
	}

	// Phase 33 — Crisis auto-arm: if ≥2 hosts go TAMPERED within 1h,
	// auto-engage Crisis Mode (real-world ransomware behaviour:
	// rolling pwn across the fleet). The crisis listener subscribes
	// to tamper:detected events and counts unique agent_ids.
	if c.Infra.Bus != nil {
		startTamperCrisisWatcher(context.Background(), c.Infra.Bus, c.Log)
	}

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

	// Wire the CampaignBuilder into GraphService so the frontend can query clusters.
	if c.Intel.GraphService != nil && c.Intel.CampaignBuilder != nil {
		c.Intel.GraphService.SetCampaignBuilder(c.Intel.CampaignBuilder)
		c.Log.Info("[CONTAINER] CampaignBuilder wired into GraphService (GetActiveClusters active)")
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
	c.mustRegister(c.Security.RotationService)
	c.mustRegister(c.Security.SuppressionService)
	c.mustRegister(c.Security.ReportScheduler)

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
	c.mustRegister(c.Platform.PlatformService)

	// Infrastructure
	if c.Infra.Messaging != nil {
		c.mustRegister(c.Infra.Messaging)
	}

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
