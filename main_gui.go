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
		Services: []application.Service{
			application.NewService(oblivraApp.HostService),
			application.NewService(oblivraApp.SSHService),
			application.NewService(oblivraApp.VaultService),
			application.NewService(oblivraApp.SessionService),
			application.NewService(oblivraApp.SettingsService),
			application.NewService(oblivraApp.SnippetService),
			application.NewService(oblivraApp.BroadcastService),
			application.NewService(oblivraApp.MultiExecService),
			application.NewService(oblivraApp.PluginService),
			application.NewService(oblivraApp.SecurityService),
			application.NewService(oblivraApp.ComplianceService),
			application.NewService(oblivraApp.TeamService),
			application.NewService(oblivraApp.SIEMService),
			application.NewService(oblivraApp.LocalService),
			application.NewService(oblivraApp.TelemetryService),
			application.NewService(oblivraApp.AIService),
			application.NewService(oblivraApp.AlertingService),
			application.NewService(oblivraApp.HealthService),
			application.NewService(oblivraApp.MetricsService),
			application.NewService(oblivraApp.TunnelService),
			application.NewService(oblivraApp.ShareService),
			application.NewService(oblivraApp.RecordingService),
			application.NewService(oblivraApp.LogSourceService),
			application.NewService(oblivraApp.WorkspaceService),
			application.NewService(oblivraApp.NotesService),
			application.NewService(oblivraApp.UpdaterService),
			application.NewService(oblivraApp.SyncService),
			application.NewService(oblivraApp.FileService),
			application.NewService(oblivraApp.DiscoveryService),
			application.NewService(oblivraApp.AgentService),
			application.NewService(oblivraApp.GovernanceService),
			application.NewService(oblivraApp.ForensicsService),
			application.NewService(oblivraApp.PolicyService),
			application.NewService(oblivraApp.IncidentService),
			application.NewService(oblivraApp.PlaybookService),
			application.NewService(oblivraApp.SimulationService),
			application.NewService(oblivraApp.UEBAService),
			application.NewService(oblivraApp.GraphService),
			application.NewService(oblivraApp.NDRService),
			application.NewService(oblivraApp.RiskService),
			application.NewService(oblivraApp.TrustService),
			application.NewService(oblivraApp.CredentialIntel),
			application.NewService(oblivraApp.AnalyticsService),
			application.NewService(oblivraApp.DisasterService),
			application.NewService(oblivraApp.TemporalService),
			application.NewService(oblivraApp.LineageService),
			application.NewService(oblivraApp.DecisionService),
			application.NewService(oblivraApp.CounterfactualService),
			application.NewService(oblivraApp.TailingService),
			application.NewService(oblivraApp.SyntheticService),
			application.NewService(oblivraApp.IdentityService),
			application.NewService(oblivraApp.ObservabilityService),
			application.NewService(oblivraApp.DataLifecycleService),
			application.NewService(oblivraApp.TransferManager),
			application.NewService(oblivraApp.NetworkIsolatorService),
			application.NewService(oblivraApp.LedgerService),
			application.NewService(oblivraApp.DiagnosticsService),
			application.NewService(oblivraApp.FusionService),
			application.NewService(oblivraApp.LicensingService),
			application.NewService(oblivraApp.BookmarkService),
			application.NewService(oblivraApp.CommandHistory),
			application.NewService(oblivraApp.OperatorService),
			application.NewService(oblivraApp.SessionPersistence),
			application.NewService(oblivraApp.RotationService),
			application.NewService(oblivraApp.SuppressionService),
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
