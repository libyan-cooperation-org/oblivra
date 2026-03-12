package app

import (
	"context"
	"fmt"
	"os"
	stdruntime "runtime"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/kingknull/oblivrashell/internal/simulation"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Service defines a standard interface for application services
type Service interface {
	Name() string
	Startup(ctx context.Context)
	Shutdown()
}

// BaseService provides a default implementation for optional methods
type BaseService struct{}

func (s *BaseService) Startup(ctx context.Context) {}
func (s *BaseService) Shutdown()                   {}

// App is the main application struct that Wails binds to
type App struct {
	ctx       context.Context
	mu        sync.RWMutex
	container *Container
	version   string
	ready     bool

	// Bound services (exposed to frontend)
	HostService       *HostService
	SSHService        *SSHService
	VaultService      *VaultService
	SessionService         *SessionService
	SettingsService        *SettingsService
	SnippetService         *SnippetService
	BroadcastService       *BroadcastService
	MultiExecService       *MultiExecService
	PluginService          *PluginService
	SecurityService        *SecurityService
	ComplianceService      *ComplianceService
	TeamService            *TeamService
	SIEMService            *SIEMService
	LocalService           *LocalService
	AIService              *AIService
	TelemetryService       *TelemetryService
	IdentityService        *IdentityService
	TransferManager        *TransferManager
	NetworkIsolatorService *NetworkIsolatorService

	// Newly wired
	AlertingService       *AlertingService
	HealthService         *HealthService
	MetricsService        *MetricsService
	TunnelService         *TunnelService
	ShareService          *ShareService
	RecordingService      *RecordingService
	LogSourceService      *LogSourceService
	WorkspaceService      *WorkspaceService
	NotesService          *NotesService
	UpdaterService        *UpdaterService
	SyncService           *SyncService
	FileService           *FileService
	DiscoveryService      *DiscoveryService
	AgentService          *AgentService
	GovernanceService     *GovernanceService
	ForensicsService      *ForensicsService
	PolicyService         *PolicyService
	IncidentService       *IncidentService
	PlaybookService       *PlaybookService
	SimulationService     *simulation.SimulationService
	ObservabilityService  *ObservabilityService
	UEBAService           *UEBAService
	GraphService          *GraphService
	NDRService            *NDRService
	RiskService           *RiskService
	TrustService          *RuntimeTrustService
	CredentialIntel       *CredentialIntelService
	DisasterService       *DisasterService
	IngestService         *IngestService
	TemporalService       *TemporalService
	LineageService        *LineageService
	DecisionService       *DecisionService
	CounterfactualService *CounterfactualService
	LedgerService         *LedgerService
	MemorySecurity        *MemorySecurityService
	DeterministicResponse *DeterministicResponseService
	SyntheticService      *SyntheticService
	TailingService        *TailingService
	AnalyticsService      *AnalyticsService
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

	a.container = NewContainer(l, a.version)

	// Initialize the container immediately. This populates all service pointers.
	// We use the background context here; Startup will later pass the Wails context.
	if err := a.container.Init(context.Background()); err != nil {
		l.Error("Container initialization failed: %v", err)
	}

	// Map container-managed services to App fields.
	// These pointers are now STABLE and LIVE. Wails will bind to these addresses.
	a.HostService = a.container.HostService
	a.SSHService = a.container.SSHService
	a.VaultService = a.container.VaultService
	a.SessionService = a.container.SessionService
	a.SettingsService = a.container.SettingsService
	a.SnippetService = a.container.SnippetService
	a.BroadcastService = a.container.BroadcastService
	a.MultiExecService = a.container.MultiExecService
	a.PluginService = a.container.PluginService
	a.SecurityService = a.container.SecurityService
	a.ComplianceService = a.container.ComplianceService
	a.TeamService = a.container.TeamService
	a.SIEMService = a.container.SIEMService
	a.LocalService = a.container.LocalService
	a.AIService = a.container.AIService
	a.TelemetryService = a.container.TelemetryService
	a.IdentityService = a.container.IdentityService
	a.TransferManager = a.container.TransferManager
	a.NetworkIsolatorService = a.container.NetworkIsolatorService

	a.AlertingService = a.container.AlertingService
	a.HealthService = a.container.HealthService
	a.MetricsService = a.container.MetricsService
	a.TunnelService = a.container.TunnelService
	a.ShareService = a.container.ShareService
	a.RecordingService = a.container.RecordingService
	a.LogSourceService = a.container.LogSourceService
	a.WorkspaceService = a.container.WorkspaceService
	a.NotesService = a.container.NotesService
	a.UpdaterService = a.container.UpdaterService
	a.SyncService = a.container.SyncService
	a.FileService = a.container.FileService
	a.DiscoveryService = a.container.DiscoveryService
	a.AgentService = a.container.AgentService
	a.GovernanceService = a.container.GovernanceService
	a.ForensicsService = a.container.ForensicsService
	a.PolicyService = a.container.PolicyService
	a.IncidentService = a.container.IncidentService
	a.PlaybookService = a.container.PlaybookService
	a.SimulationService = a.container.SimulationService
	a.ObservabilityService = a.container.ObservabilityService
	a.UEBAService = a.container.UEBAService
	a.GraphService = a.container.GraphService
	a.NDRService = a.container.NDRService
	a.RiskService = a.container.RiskService
	a.TrustService = a.container.TrustService
	a.CredentialIntel = a.container.CredentialIntel
	a.DisasterService = a.container.DisasterService
	a.IngestService = a.container.IngestService
	a.TemporalService = a.container.TemporalService
	a.LineageService = a.container.LineageService
	a.DecisionService = a.container.DecisionService
	a.CounterfactualService = a.container.CounterfactualService
	a.LedgerService = a.container.LedgerService
	a.MemorySecurity = a.container.MemorySecurity
	a.DeterministicResponse = a.container.DeterministicResponse
	a.SyntheticService = a.container.SyntheticService
	a.TailingService = a.container.TailingService
	a.AnalyticsService = a.container.AnalyticsService

	return a
}

// Startup is called when the app starts.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Logic services and heavy background loops start here.
	// Since the container was initialized in New(), we just trigger the StartAll.
	a.container.Registry.StartAll(ctx)

	a.ready = true
	a.container.Log.Info("Application startup complete")

	// 8. Headless / Server Auto-Unlock
	// In sovereign server deployments, we attempt to auto-unlock via the OS keychain
	// if the user has previously 'remembered' the credential.
	if a.VaultService != nil {
		go func() {
			time.Sleep(2 * time.Second) // Give services a moment to settle
			if !a.VaultService.IsUnlocked() {
				a.container.Log.Info("[HARDENING] Attempting headless auto-unlock...")
				if err := a.VaultService.TryAutoUnlock(); err != nil {
					a.container.Log.Warn("[HARDENING] Headless auto-unlock failed: %v. Database-dependent features may remain locked.", err)
				} else {
					a.container.Log.Info("[HARDENING] Headless vault successfully unlocked.")
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
		a.container.Bus.Subscribe(eventbus.AllEvents, func(event eventbus.Event) {
			EmitEvent(a.ctx, string(event.Type), event.Data)
		})
	}
}

// Shutdown is called at the end of the application lifecycle
func (a *App) Shutdown(ctx context.Context) {
	if a.container != nil && a.container.Registry != nil {
		a.container.Registry.StopAll()
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
func (a *App) GetTrustDriftMetrics() TrustDriftMetrics {
	if a.TrustService == nil {
		return TrustDriftMetrics{EstimatedFailureTime: "N/A"}
	}
	return a.container.TrustService.GetTrustDriftMetrics()
}

// GetPlatformInfo returns platform information
func (a *App) GetPlatformInfo() map[string]string {
	if a.container == nil {
		return map[string]string{"version": a.version}
	}
	return map[string]string{
		"os":      a.container.Platform.Name(),
		"arch":    a.container.Platform.Arch(),
		"version": a.version,
	}
}

// SearchLogs executes queries against the local SQLite Analytics Engine
func (a *App) SearchLogs(query string, mode string, limit int, offset int) ([]map[string]interface{}, error) {
	if a.container == nil || a.container.AnalyticsEngine == nil {
		return nil, fmt.Errorf("analytics engine is not initialized")
	}
	return a.container.AnalyticsEngine.Search(query, mode, limit, offset)
}

// GetRecordingFrames retrieves the full TTY frame sequence for a session
func (a *App) GetRecordingFrames(sessionID string) ([]map[string]interface{}, error) {
	if a.container == nil || a.container.AnalyticsEngine == nil {
		return nil, fmt.Errorf("analytics engine is not initialized")
	}
	return a.container.AnalyticsEngine.GetRecordingFrames(sessionID)
}

// SaveDashboard stores a dashboard layout as JSON
func (a *App) SaveDashboard(id string, layoutJSON string) error {
	if a.container == nil || a.container.AnalyticsEngine == nil {
		return fmt.Errorf("analytics engine is not initialized")
	}
	return a.container.AnalyticsEngine.SaveConfig("dashboard_"+id, layoutJSON)
}

// LoadDashboard retrieves a saved dashboard layout
func (a *App) LoadDashboard(id string) (string, error) {
	if a.container == nil || a.container.AnalyticsEngine == nil {
		return "", fmt.Errorf("analytics engine is not initialized")
	}
	return a.container.AnalyticsEngine.LoadConfig("dashboard_" + id)
}

// RunWidgetQuery executes a dashboard widget query
func (a *App) RunWidgetQuery(query string, limit int) ([]map[string]interface{}, error) {
	if a.container == nil || a.container.AnalyticsEngine == nil {
		return nil, fmt.Errorf("analytics engine is not initialized")
	}
	return a.container.AnalyticsEngine.Search(query, "sql", limit, 0)
}

// RunOsquery executes an osquery-style query (stub — osquery integration planned for Phase 6)
func (a *App) RunOsquery(query string) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("osquery integration not yet available — planned for Phase 6 Agent Framework")
}

// EmitEvent safely wraps wails runtime.EventsEmit to avoid test panics
func EmitEvent(ctx context.Context, eventName string, optionalData ...interface{}) {
	if ctx == nil {
		return
	}
	if ctx.Value("test") != nil {
		return
	}
	// Defensively catch Wails panics if given context lacks expected lifecycle flags
	defer func() {
		if r := recover(); r != nil {
			// Do nothing on panic, it's just a test context lacking Wails bindings
		}
	}()
	runtime.EventsEmit(ctx, eventName, optionalData...)
}
