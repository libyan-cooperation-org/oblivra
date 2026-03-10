package main

import (
	"embed"
	"flag"
	"log"
	"os"

	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/isolation"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/kingknull/oblivrashell/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "worker" {
		workerCmd(os.Args[2:])
		return
	}

	// Sovereign Core: Runtime Binary Attestation
	attestSvc := attestation.NewAttestationService()
	if err := attestSvc.VerifyOnStartup(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	application := app.New()

	err := wails.Run(&options.App{
		Title:     "OblivraShell",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,

		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		BackgroundColour: &options.RGBA{R: 13, G: 17, B: 23, A: 1},
		Frameless:        true,

		OnStartup:  application.Startup,
		OnShutdown: application.Shutdown,
		OnDomReady: application.DomReady,

		Bind: []interface{}{
			application,
			application.HostService,
			application.SSHService,
			application.VaultService,
			application.SessionService,
			application.SettingsService,
			application.SnippetService,
			application.BroadcastService,
			application.MultiExecService,
			application.PluginService,
			application.SecurityService,
			application.ComplianceService,
			application.TeamService,
			application.SIEMService,
			application.LocalService,
			application.TelemetryService,
			application.AIService,
			application.AlertingService,
			application.HealthService,
			application.MetricsService,
			application.TunnelService,
			application.ShareService,
			application.RecordingService,
			application.LogSourceService,
			application.WorkspaceService,
			application.NotesService,
			application.UpdaterService,
			application.SyncService,
			application.FileService,
			application.DiscoveryService,
			application.AgentService,
			application.GovernanceService,
			application.ForensicsService,
			application.PolicyService,
			application.IncidentService,
			application.PlaybookService,
			application.SimulationService,
			application.UEBAService,
			application.GraphService,
			application.NDRService,
			application.RiskService,
			application.TrustService,
			application.CredentialIntel,
			application.DisasterService,
			application.TemporalService,
			application.LineageService,
			application.DecisionService,
			application.CounterfactualService,
			application.TailingService,
			application.SyntheticService,
			// Removed: LedgerService, MemorySecurity, DeterministicResponse
		},

		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			Theme:                windows.Dark,
			BackdropType:         windows.Mica,
		},

		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},

		Linux: &linux.Options{
			WindowIsTranslucent: false,
		},

		Debug: options.Debug{
			OpenInspectorOnStartup: false,
		},
	})

	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}

func workerCmd(args []string) {
	fs := flag.NewFlagSet("worker", flag.ExitOnError)
	wType := fs.String("type", "", "Type of worker to start (detect, policy, enrich)")
	fs.Parse(args)

	if *wType == "" {
		log.Fatal("Worker error: --type is required")
	}

	worker := isolation.NewIsolatedWorker(*wType)

	// Here we would configure specific rpc receiver based on the worker type
	// For example:
	// if *wType == "detect" {
	// 	 worker.Register("DetectEngine", newIsolatedDetectEngine())
	// }

	// Serve forever on standard I/O
	worker.ServeStdinStdout(os.Stdin, os.Stdout)
}
