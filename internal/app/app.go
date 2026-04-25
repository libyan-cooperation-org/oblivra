package app

import (
	"context"
	"fmt"
	"os"
	stdruntime "runtime"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/core"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/kingknull/oblivrashell/internal/services"
	"github.com/kingknull/oblivrashell/internal/simulation"
	"path/filepath"
)


// App is the main application struct that Wails binds to
type App struct {
	ctx       context.Context
	mu        sync.RWMutex
	container *core.Container
	version   string
	ready     bool

	// Bound services (exposed to frontend)
	HostService       *services.HostService
	SSHService        *services.SSHService
	VaultService      *services.VaultService
	SessionService         *services.SessionService
	SettingsService        *services.SettingsService
	SnippetService         *services.SnippetService
	BroadcastService       *services.BroadcastService
	MultiExecService       *services.MultiExecService
	PluginService          *services.PluginService
	SecurityService        *services.SecurityService
	ComplianceService      *services.ComplianceService
	TeamService            *services.TeamService
	SIEMService            *services.SIEMService
	LocalService           *services.LocalService
	AIService              *services.AIService
	TelemetryService       *services.TelemetryService
	IdentityService        *services.IdentityService
	TransferManager        *services.TransferManager
	NetworkIsolatorService *services.NetworkIsolatorService
	RotationService        *services.RotationService
	SuppressionService     *services.SuppressionService

	// Newly wired
	AlertingService       *services.AlertingService
	HealthService         *services.HealthService
	MetricsService        *services.MetricsService
	TunnelService         *services.TunnelService
	ShareService          *services.ShareService
	RecordingService      *services.RecordingService
	LogSourceService      *services.LogSourceService
	WorkspaceService      *services.WorkspaceService
	NotesService          *services.NotesService
	UpdaterService        *services.UpdaterService
	SyncService           *services.SyncService
	FileService           *services.FileService
	DiscoveryService      *services.DiscoveryService
	AgentService          *services.AgentService
	GovernanceService     *services.GovernanceService
	ForensicsService      *services.ForensicsService
	PolicyService         *services.PolicyService
	IncidentService       *services.IncidentService
	PlaybookService       *services.PlaybookService
	SimulationService     *simulation.SimulationService
	ObservabilityService  *services.ObservabilityService
	UEBAService           *services.UEBAService
	GraphService          *services.GraphService
	NDRService            *services.NDRService
	RiskService           *services.RiskService
	TrustService          *services.RuntimeTrustService
	CredentialIntel       *services.CredentialIntelService
	DisasterService       *services.DisasterService
	IngestService         *services.IngestService
	TemporalService       *services.TemporalService
	LineageService        *services.LineageService
	DecisionService       *services.DecisionService
	CounterfactualService *services.CounterfactualService
	LedgerService         *services.LedgerService
	MemorySecurity        *services.MemorySecurityService
	DeterministicResponse *services.DeterministicResponseService
	SyntheticService      *services.SyntheticService
	TailingService        *services.TailingService
	AnalyticsService      *services.AnalyticsService
	DataLifecycleService  *services.DataLifecycleService
	DiagnosticsService    *services.DiagnosticsService
	FusionService         *services.FusionService
	LicensingService      *services.LicensingService
	ReportService         *services.ReportService

	// Terminal UX (Phase 23)
	BookmarkService    *services.BookmarkService
	CommandHistory     *services.CommandHistoryService
	OperatorService    *services.OperatorService
	SessionPersistence *services.SessionPersistence

	// SOC multi-monitor pop-out
	WindowService *services.WindowService
}

// New creates a new App instance with placeholder service structs.
//
// WHY PLACEHOLDERS: Wails binding generation (and the Bind: []interface{}{...}
// list in main.go) reflects on service pointers at process start, before
// Startup() runs. Every pointer in the Bind list must be non-nil at that point
// or Wails panics with "not a pointer to a struct".
//
// These zero-value structs are throwaway — Startup() replaces every pointer
// with the live, fully-initialised container instance via pointer assignment
// (a.X = a.container.X). The frontend always calls through the Wails-bound
// pointer, which by then points at the real service.
// New creates a new App instance and initializes the backend container immediately.
// This ensures that Wails binds to the actual service pointers, avoiding
// nil pointer dereferences and mutex copy warnings.
func New() *App {
	a := &App{
		version: "0.1.0",
	}

	// Ensure platform-specific directories exist (Day Zero bootstrapping)
	if err := platform.EnsureDirectories(); err != nil {
		fmt.Printf("FATAL: Failed to create application directories: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger once for the entire application lifecycle.
	l, err := logger.New(logger.Config{
		Level:      logger.InfoLevel,
		OutputPath: platform.LogPath(),
		Sanitize:   true,
		JSON:       os.Getenv("OBLIVRA_LOG_JSON") == "true",
	})
	if err != nil {
		fmt.Printf("Logger initialization failed: %v\n", err)
		// Fallback to a basic logger if file logging fails.
		l = logger.NewStdoutLogger()
	}

	a.container = core.NewContainer(l, a.version)

	// Initialize the container immediately. This populates all service pointers.
	if err := a.container.Init(context.Background()); err != nil {
		l.Error("Container initialization failed: %v", err)
	}

	// Map container-managed services to App fields.
	// We delegate this to a dedicated method to keep New() clean.
	a.wireServices()

	return a
}

// wireServices maps the deeply nested container services to the flat App struct.
// This flat mapping is required for Wails to correctly generate frontend bindings
// for all platform capabilities.
func (a *App) wireServices() {
	// Cluster: Product
	p := a.container.Product
	a.HostService = p.HostService
	a.SSHService = p.SSHService
	a.VaultService = p.VaultService
	a.SessionService = p.SessionService
	a.SettingsService = p.SettingsService
	a.SnippetService = p.SnippetService
	a.MultiExecService = p.MultiExecService
	a.FileService = p.FileService
	a.WorkspaceService = p.WorkspaceService
	a.NotesService = p.NotesService
	a.ShareService = p.ShareService
	a.RecordingService = p.RecordingService
	a.BookmarkService = p.BookmarkService
	a.CommandHistory = p.CommandHistory
	a.OperatorService = p.OperatorService
	a.SessionPersistence = p.SessionPersistence

	// WindowService doesn't depend on the container — instantiate inline.
	// It only needs the logger and Wails' application.Get() at call time.
	a.WindowService = services.NewWindowService(a.container.Log)
	a.TransferManager = p.TransferManager
	a.ComplianceService = p.ComplianceService
	a.TailingService = p.TailingService
	a.TeamService = p.TeamService

	// Cluster: Security
	s := a.container.Security
	a.SecurityService = s.SecurityService
	a.IdentityService = s.IdentityService
	a.PolicyService = s.PolicyService
	a.TrustService = s.TrustService
	a.MemorySecurity = s.MemorySecurity
	a.CredentialIntel = s.CredentialIntel
	a.GovernanceService = s.GovernanceService
	a.ReportService = s.ReportService
	a.RotationService = s.RotationService
	a.SuppressionService = s.SuppressionService

	// Cluster: SIEM
	siem := a.container.SIEM
	a.SIEMService = siem.SIEMService
	a.IngestService = siem.IngestService
	a.NDRService = siem.NDRService
	a.UEBAService = siem.UEBAService
	a.ForensicsService = siem.ForensicsService
	a.AlertingService = siem.AlertingService
	a.LogSourceService = siem.LogSourceService
	a.AgentService = siem.AgentService
	a.FusionService = siem.FusionService

	// Cluster: Intel
	i := a.container.Intel
	a.AnalyticsService = i.AnalyticsService
	a.RiskService = i.RiskService
	a.GraphService = i.GraphService
	a.DecisionService = i.DecisionService
	a.CounterfactualService = i.CounterfactualService
	a.LineageService = i.LineageService
	a.TemporalService = i.TemporalService
	a.WorkspaceService = p.WorkspaceService // Already in Product

	// Cluster: Response
	r := a.container.Response
	a.IncidentService = r.IncidentService
	a.PlaybookService = r.PlaybookService
	a.NetworkIsolatorService = r.NetworkIsolatorService
	a.SimulationService = r.SimulationService
	a.DeterministicResponse = r.DeterministicResponse
	a.LedgerService = r.LedgerService

	// Cluster: Platform
	plt := a.container.Platform
	a.DiscoveryService = plt.DiscoveryService
	a.UpdaterService = plt.UpdaterService
	a.SyncService = plt.SyncService
	a.TunnelService = plt.TunnelService
	a.HealthService = plt.HealthService
	a.MetricsService = plt.MetricsService
	a.PluginService = plt.PluginService
	a.AIService = plt.AIService
	a.LocalService = plt.LocalService
	a.SyntheticService = plt.SyntheticService
	a.BroadcastService = plt.BroadcastService
	a.ObservabilityService = plt.ObservabilityService
	a.DisasterService = plt.DisasterService
	a.TelemetryService = plt.TelemetryService
	a.DataLifecycleService = plt.DataLifecycleService
	a.DiagnosticsService = plt.DiagnosticsService
	a.LicensingService = plt.LicensingService
}

// Startup is called when the app starts.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize Kernel
	kernel, err := platform.NewKernel(ctx, a.container.Registry)
	if err != nil {
		a.container.Log.Error("Failed to initialize platform kernel: %v", err)
		return
	}
	a.container.Kernel = kernel

	// 3. Start core services
	if err := a.container.Kernel.Start(); err != nil {
		a.container.Log.Error("Failed to start services: %v", err)
		return
	}

	// 4. Propagate the Wails context to VaultService so EventsEmit works correctly
	if a.VaultService != nil {
		a.VaultService.SetContext(ctx)
	}

	a.ready = true
	a.container.Log.Info("Application startup complete")

	// 8. Headless / Server Auto-Unlock
	// Only attempt keychain auto-unlock if the vault was previously set up with
	// remember=true AND has a stored keychain entry. Attempting it unconditionally
	// races with the user typing their password in the UI and produces a spurious
	// "incorrect password" error on every startup
	if a.VaultService != nil {
		go func() {
			// Short pause so the vault UI can render before any backend activity
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}

			// Bail immediately if already unlocked (e.g. fast user who typed password)
			if a.VaultService.IsUnlocked() {
				return
			}

			// 1. Try Keychain Auto-Unlock
			if a.VaultService.HasKeychainEntry() {
				a.container.Log.Info("[AUTO-UNLOCK] Keychain entry found, attempting auto-unlock...")
				if err := a.VaultService.TryAutoUnlock(); err == nil {
					a.container.Log.Info("[AUTO-UNLOCK] Vault unlocked from keychain")
					return
				}
				a.container.Log.Warn("[AUTO-UNLOCK] Keychain unlock failed")
			}

			// 2. Try Dev-Mode Auto-Unlock (if configured)
			devPassword := os.Getenv("OBLIVRA_DEV_PASSWORD")
			if devPassword != "" && os.Getenv("OBLIVRA_ENV") == "development" {
				a.container.Log.Info("[AUTO-UNLOCK] Development mode active, attempting dev password unlock...")

				// Pre-emptive check: If not setup, initialize it
				if !a.container.Infra.Vault.IsSetup() {
					a.container.Log.Info("[AUTO-UNLOCK] Vault not setup, initializing with dev password...")
					if err := a.container.Infra.Vault.Setup(devPassword, ""); err != nil {
						a.container.Log.Warn("[AUTO-UNLOCK] Dev vault setup failed: %v", err)
					}
				}

				if err := a.VaultService.UnlockWithPassword(devPassword, false); err != nil {
					a.container.Log.Warn("[AUTO-UNLOCK] Dev password unlock failed: %v", err)

					// If it failed and we are in dev, re-initialize (aggressive but helpful for dev)
					a.container.Log.Warn("[AUTO-UNLOCK] RE-INITIALIZING vault with dev password (reason: dev-unlock failed)")
					vaultPath := filepath.Join(platform.ConfigDir(), "vault.json")
					os.Remove(vaultPath)
					if err := a.container.Infra.Vault.Setup(devPassword, ""); err == nil {
						if err := a.VaultService.UnlockWithPassword(devPassword, false); err == nil {
							a.container.Log.Info("[AUTO-UNLOCK] Vault re-initialized and unlocked successfully")
						} else {
							a.container.Log.Error("[AUTO-UNLOCK] Second unlock attempt failed: %v", err)
						}
					} else {
						a.container.Log.Error("[AUTO-UNLOCK] Re-initialization setup failed: %v", err)
					}
				} else {
					a.container.Log.Info("[AUTO-UNLOCK] Vault unlocked using dev password")
				}
			}
		}()
	}

	// Start Hardening Telemetry (Goroutine & Resource Watchdog)
	go a.MonitorHardening()
}

// MonitorHardening tracks resource consumption metrics to detect leaks during soak tests.
func (a *App) MonitorHardening() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			// 1. Goroutine Watchdog
			gc := stdruntime.NumGoroutine()
			
			// 2. Memory Metrics
			var ms stdruntime.MemStats
			stdruntime.ReadMemStats(&ms)

			// 3. Ingestion Metrics (if available)
			eps := int64(0)
			total := int64(0)
			drops := int64(0)
			if a.IngestService != nil {
				m := a.IngestService.GetMetrics()
				eps = m["events_per_second"].(int64)
				total = m["total_processed"].(int64)
				drops = m["dropped_events"].(int64)
			}
			
			a.container.Log.Info("[HARDENING] Stats - G:%d | EPS:%d | Total:%d | Drops:%d | Heap:%.2fMB | Obj:%d",
				gc,
				eps,
				total,
				drops,
				float64(ms.HeapAlloc)/1024/1024,
				ms.HeapObjects,
			)

			// Simple threshold alerting for developers
			if gc > 2000 {
				a.container.Log.Warn("[HARDENING] ABNORMAL GOROUTINE SPIKE DETECTED: %d", gc)
			}
		}
	}
}

// DomReady is called after the frontend DOM is ready
func (a *App) DomReady(ctx context.Context) {
	if a.container != nil {
		a.container.Log.Info("Frontend DOM ready")
		a.container.Infra.Bus.Subscribe(eventbus.AllEvents, func(event eventbus.Event) {
			services.EmitEvent(string(event.Type), event.Data)
		})
	}
}

// Shutdown is called at the end of the application lifecycle
func (a *App) Shutdown(ctx context.Context) {
	if a.container != nil {
		if a.container.Kernel != nil {
			a.container.Kernel.Stop()
		}
		a.container.Close()
	}
}

// GetVersion returns the app version
func (a *App) GetVersion() string {
	return a.version
}

// GetObservabilityStatus returns detailed internal health metrics
func (a *App) GetObservabilityStatus() map[string]interface{} {
	if a.ObservabilityService == nil {
		return nil
	}
	return a.ObservabilityService.GetObservabilityStatus()
}

// GetTrustDriftMetrics exposes the rolling slope and anticipated Time-To-Failure
func (a *App) GetTrustDriftMetrics() services.TrustDriftMetrics {
	if a.TrustService == nil {
		return services.TrustDriftMetrics{EstimatedFailureTime: "N/A"}
	}
	return a.container.Security.TrustService.GetTrustDriftMetrics()
}

// GetPlatformInfo returns platform information
func (a *App) GetPlatformInfo() map[string]string {
	if a.container == nil {
		return map[string]string{"version": a.version}
	}
	return map[string]string{
		"os":      a.container.Infra.Platform.Name(),
		"arch":    a.container.Infra.Platform.Arch(),
		"version": a.version,
	}
}

// SearchLogs executes queries against the local SQLite Analytics Engine
func (a *App) SearchLogs(query string, mode string, limit int, offset int) ([]map[string]interface{}, error) {
	if a.container == nil || a.container.Infra.AnalyticsEngine == nil {
		return nil, fmt.Errorf("analytics engine is not initialized")
	}
	return a.container.Infra.AnalyticsEngine.Search(a.ctx, query, mode, limit, offset)
}

// GetRecordingFrames retrieves the full TTY frame sequence for a session
func (a *App) GetRecordingFrames(sessionID string) ([]map[string]interface{}, error) {
	if a.container == nil || a.container.Infra.AnalyticsEngine == nil {
		return nil, fmt.Errorf("analytics engine is not initialized")
	}
	return a.container.Infra.AnalyticsEngine.GetRecordingFrames(a.ctx, sessionID)
}

// SaveDashboard stores a dashboard layout as JSON
func (a *App) SaveDashboard(id string, layoutJSON string) error {
	if a.container == nil || a.container.Infra.AnalyticsEngine == nil {
		return fmt.Errorf("analytics engine is not initialized")
	}
	return a.container.Infra.AnalyticsEngine.SaveConfig(a.ctx, "dashboard_"+id, layoutJSON)
}

// LoadDashboard retrieves a saved dashboard layout
func (a *App) LoadDashboard(id string) (string, error) {
	if a.container == nil || a.container.Infra.AnalyticsEngine == nil {
		return "", fmt.Errorf("analytics engine is not initialized")
	}
	return a.container.Infra.AnalyticsEngine.LoadConfig(a.ctx, "dashboard_" + id)
}

// RunWidgetQuery executes a dashboard widget query
func (a *App) RunWidgetQuery(query string, limit int) ([]map[string]interface{}, error) {
	if a.container == nil || a.container.Infra.AnalyticsEngine == nil {
		return nil, fmt.Errorf("analytics engine is not initialized")
	}
	return a.container.Infra.AnalyticsEngine.Search(a.ctx, query, "sql", limit, 0)
}

// RunOsquery executes an osquery-style query (stub — osquery integration planned for Phase 6)
func (a *App) RunOsquery(query string) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("osquery integration not yet available — planned for Phase 6 Agent Framework")
}

