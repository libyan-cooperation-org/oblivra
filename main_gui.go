//go:build !server

package main

import (
	"context"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/kingknull/oblivrashell/internal/app"
)

func runGUI() {
	oblivraApp := app.New()

	log.Println("[DEBUG] About to call application.New")
	app := application.New(application.Options{
		Name:        "OblivraShell",
		Description: "OBLIVRA Enterprise Core",
		// Phase 36.12: Wails service registrations pruned to only services with
		// active frontend callers (verified by reverse-import audit, 2026-04-30).
		// Removed services remain LIVE in the Go runtime via container.go +
		// app.go — they're no longer reachable as Wails RPC, which is correct:
		// they have zero `@wailsjs/.../services/X` consumers in the bundle.
		// Use REST or the event bus to interact with them server-side.
		//
		// Removed: AnalyticsService, BroadcastService, CommandHistory,
		// CounterfactualService, CredentialIntel, DataLifecycleService,
		// DiagnosticsService, DiscoveryService, FileService, GovernanceService,
		// LogSourceService, MetricsService, MultiExecService, ObservabilityService,
		// OperatorService, PolicyService, RiskService, SecurityService,
		// SessionPersistence, SessionService, ShareService, SyntheticService,
		// TailingService, TelemetryService, TransferManager, WorkspaceService.
		//
		// Phase 36 prior cuts: AIService, ComplianceService, IncidentService,
		// NetworkIsolatorService, PlaybookService, PluginService.
		Services: []application.Service{
			application.NewService(oblivraApp.HostService),
			application.NewService(oblivraApp.SSHService),
			application.NewService(oblivraApp.VaultService),
			application.NewService(oblivraApp.SettingsService),
			application.NewService(oblivraApp.SnippetService),
			application.NewService(oblivraApp.TeamService),
			application.NewService(oblivraApp.SIEMService),
			application.NewService(oblivraApp.LocalService),
			application.NewService(oblivraApp.AlertingService),
			application.NewService(oblivraApp.HealthService),
			application.NewService(oblivraApp.TunnelService),
			application.NewService(oblivraApp.RecordingService),
			application.NewService(oblivraApp.NotesService),
			application.NewService(oblivraApp.UpdaterService),
			application.NewService(oblivraApp.SyncService),
			application.NewService(oblivraApp.AgentService),
			application.NewService(oblivraApp.ForensicsService),
			application.NewService(oblivraApp.SimulationService),
			application.NewService(oblivraApp.UEBAService),
			application.NewService(oblivraApp.GraphService),
			application.NewService(oblivraApp.NDRService),
			application.NewService(oblivraApp.TrustService),
			application.NewService(oblivraApp.DisasterService),
			application.NewService(oblivraApp.TemporalService),
			application.NewService(oblivraApp.LineageService),
			application.NewService(oblivraApp.DecisionService),
			application.NewService(oblivraApp.IdentityService),
			application.NewService(oblivraApp.LedgerService),
			application.NewService(oblivraApp.FusionService),
			application.NewService(oblivraApp.LicensingService),
			application.NewService(oblivraApp.BookmarkService),
			application.NewService(oblivraApp.RotationService),
			application.NewService(oblivraApp.SuppressionService),
			application.NewService(oblivraApp.WindowService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			oblivraApp.Shutdown(context.Background())
		},
	})

	oblivraApp.Startup(context.Background())
	go func() {
		time.Sleep(1 * time.Second)
		oblivraApp.DomReady(context.Background())
	}()

	// Phase 22.5 + SOC UX — wire the application menu (File / Edit / View /
	// Navigate / Window / Help) before any window is shown. Native OS
	// chrome on macOS picks this up automatically; Windows shows it under
	// the title bar; Linux GTK4 ignores it and the in-app menu is the
	// fallback (see frontend/src/components/layout/TitleBar.svelte).
	app.Menu.Set(oblivraApp.BuildApplicationMenu())

	// SOC ambient awareness — tray icon stays visible while the operator is
	// in a different app, with a quick-action menu (open SIEM / alerts /
	// terminal, pop-out shortcuts, quit).
	oblivraApp.SetupSystemTray()

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "OblivraShell",
		Width:            1280,
		Height:           800,
		MinWidth:         900,
		MinHeight:        600,
		Frameless:        true,
		BackgroundColour: application.NewRGBA(13, 17, 23, 255),
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		Windows: application.WindowsWindow{
			BackdropType: application.Mica,
		},
	})

	log.Println("[DEBUG] application init complete")
	err := app.Run()
	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}
