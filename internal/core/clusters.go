package core

import (
	"github.com/kingknull/oblivrashell/internal/services"
	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/cloud"
	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/enrich"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/graph"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/logsources"
	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/notes"
	"github.com/kingknull/oblivrashell/internal/notifications"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/kingknull/oblivrashell/internal/risk"
	"github.com/kingknull/oblivrashell/internal/search"
	"github.com/kingknull/oblivrashell/internal/security"
	"github.com/kingknull/oblivrashell/internal/sharing"
	"github.com/kingknull/oblivrashell/internal/simulation"
	"github.com/kingknull/oblivrashell/internal/storage"
	"github.com/kingknull/oblivrashell/internal/threatintel"
	"github.com/kingknull/oblivrashell/internal/temporal"
	"github.com/kingknull/oblivrashell/internal/vault"
	"github.com/kingknull/oblivrashell/internal/workspace"
)

// InfrastructureCluster holds core runtime and storage dependencies.
type InfrastructureCluster struct {
	DB               *database.Database
	Vault            *vault.Vault
	Bus              *eventbus.Bus
	Log              *logger.Logger
	Registry         *platform.Registry
	Platform         platform.Platform
	SearchEngine     *search.SearchEngine
	HotStore         *storage.HotStore
	TelemetryManager *monitoring.TelemetryManager
	HealthChecker    *monitoring.HealthChecker
	MetricsCollector *monitoring.MetricsCollector
	AnalyticsEngine  *analytics.AnalyticsEngine
	Notifier         *notifications.NotificationService
	MatchEngine      *threatintel.MatchEngine
	Enrichment       *enrich.Pipeline
	CloudAssets      database.CloudAssetStore
	TenantRepo       *database.TenantRepository
	RBAC             *auth.RBACEngine
}

// SecurityCluster holds identity, hardening, and trust services.
type SecurityCluster struct {
	SecurityService    *services.SecurityService
	IdentityService    *services.IdentityService
	TrustService       *services.RuntimeTrustService
	CanaryService      *security.CanaryService
	PolicyService      *services.PolicyService
	CredentialIntel    *services.CredentialIntelService
	GovernanceService  *services.GovernanceService
	MemorySecurity     *services.MemorySecurityService
	FIDO2Manager       *security.FIDO2Manager
	AttestationService *attestation.AttestationService
	CanaryDeployment   *services.CanaryDeploymentService
	Sentinel           *services.Sentinel
	IdentitySyncService *services.IdentitySyncService
	ReportService      *services.ReportService
}

// SIEMCluster holds ingestion, detection, and alerting services.
type SIEMCluster struct {
	AnalyticsService *services.AnalyticsService
	SIEMService      *services.SIEMService
	IngestService    *services.IngestService
	AlertingService  *services.AlertingService
	LogSourceService *services.LogSourceService
	Enrichment       *enrich.Pipeline
	AgentService     *services.AgentService
	NDRService       *services.NDRService
	UEBAService      *services.UEBAService
	ForensicsService *services.ForensicsService
	SourceManager    *logsources.SourceManager
	FusionEngine     *detection.AttackFusionEngine
	FusionService    *services.FusionService
	TemporalEngine   *temporal.IntegrityService
}

// IntelCluster holds analytics, risk, and graph correlation services.
type IntelCluster struct {
	AnalyticsService      *services.AnalyticsService
	RiskService           *services.RiskService
	GraphService          *services.GraphService
	DecisionService       *services.DecisionService
	CounterfactualService *services.CounterfactualService
	LineageService        *services.LineageService
	TemporalService       *services.TemporalService
	GraphEngine           *graph.GraphEngine
	RiskEngine            *risk.RiskEngine
	DashboardService      *services.DashboardService
	AssetIntelService     *services.AssetIntelService
	CampaignBuilder       *detection.CampaignBuilder // graph → fusion bridge
}

// ResponseCluster holds incident management and automated response logic.
type ResponseCluster struct {
	IncidentService        *services.IncidentService
	PlaybookService        *services.PlaybookService
	NetworkIsolatorService *services.NetworkIsolatorService
	SimulationService      *simulation.SimulationService
	DeterministicResponse  *services.DeterministicResponseService
	LedgerService          *services.LedgerService
	RansomwareService      *services.RansomwareService
}

// ProductCluster holds end-user logic services (SSH, Host, snippets, etc).
type ProductCluster struct {
	HostService       *services.HostService
	SSHService        *services.SSHService
	VaultService      *services.VaultService
	SessionService    *services.SessionService
	SettingsService   *services.SettingsService
	SnippetService    *services.SnippetService
	MultiExecService  *services.MultiExecService
	FileService       *services.FileService
	WorkspaceService  *services.WorkspaceService
	NotesService      *services.NotesService
	ShareService      *services.ShareService
	RecordingService  *services.RecordingService
	TransferManager   *services.TransferManager
	ComplianceService *services.ComplianceService
	TailingService    *services.TailingService
	RecordingManager  *sharing.RecordingManager
	ShareManager      *sharing.ShareManager
	NotesManager      *notes.NotesManager
	WorkspaceManager  *workspace.WorkspaceManager
	TeamService       *services.TeamService
	SnippetRepo       database.SnippetStore
	BookmarkService   *services.BookmarkService
	CommandHistory    *services.CommandHistoryService
	OperatorService   *services.OperatorService
	SessionPersistence *services.SessionPersistence
}

// PlatformCluster holds utility and system services.
type PlatformCluster struct {
	DiscoveryService     *services.DiscoveryService
	UpdaterService       *services.UpdaterService
	SyncService          *services.SyncService
	TunnelService        *services.TunnelService
	HealthService        *services.HealthService
	MetricsService       *services.MetricsService
	PluginService        *services.PluginService
	AIService            *services.AIService
	LocalService         *services.LocalService
	BroadcastService     *services.BroadcastService
	ObservabilityService *services.ObservabilityService
	DisasterService      *services.DisasterService
	SyntheticService     *services.SyntheticService
	TelemetryService     *services.TelemetryService
	DataLifecycleService *services.DataLifecycleService
	ResourceMonitor      *services.ResourceMonitor
	DiagnosticsService   *services.DiagnosticsService
	APIService           *services.APIService
	LicensingService     *services.LicensingService
	CloudDiscovery       *cloud.CloudDiscoveryManager
}
