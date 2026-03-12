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

	log.Println("[DEBUG] About to call wails.Run")
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
			// application, // REMOVED: Redundant with individual service bindings and causes type resolution bloat
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
			application.AnalyticsService,
			application.DisasterService,
			application.TemporalService,
			application.LineageService,
			application.DecisionService,
			application.CounterfactualService,
			application.TailingService,
			application.SyntheticService,
			application.IdentityService,
			application.ObservabilityService,
			application.TransferManager,
			application.NetworkIsolatorService,
			application.LedgerService,
			// NOTE: MemorySecurity and DeterministicResponse are intentionally
			// not exposed to the Wails frontend (no UI binding needed), but they ARE
			// initialized and registered in the ServiceRegistry for internal lifecycle management.
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
	log.Println("[DEBUG] wails.Run completed")

	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}

// WorkerDetectEngine is the RPC receiver for the detection worker subprocess.
type WorkerDetectEngine struct{}

func (e *WorkerDetectEngine) EvaluateEvent(args *isolation.EvaluateEventArgs, reply *isolation.EvaluateEventResponse) error {
	// Stateless evaluation: parse the raw event JSON and apply basic rule matching.
	// A full implementation would load rules from a shared config path passed via flag.
	// For now, we return an empty match set — the subprocess is correctly wired and
	// ready to receive events from the parent via JSON-RPC over stdin/stdout.
	reply.Matches = []string{}
	return nil
}

// WorkerPolicyEngine is the RPC receiver for the policy worker subprocess.
type WorkerPolicyEngine struct{}

type PolicyCheckArgs struct {
	Tier   string
	Action string
}

type PolicyCheckReply struct {
	Allowed bool
	Reason  string
}

func (e *WorkerPolicyEngine) CheckPolicy(args *PolicyCheckArgs, reply *PolicyCheckReply) error {
	// Isolated policy decisions run outside the main process address space.
	// Default: allow. A real implementation loads a policy bundle from disk.
	reply.Allowed = true
	reply.Reason = "default-allow"
	return nil
}

// WorkerEnrichEngine is the RPC receiver for the enrichment worker subprocess.
type WorkerEnrichEngine struct{}

type EnrichArgs struct {
	RawEventJSON []byte
}

type EnrichReply struct {
	EnrichedEventJSON []byte
}

func (e *WorkerEnrichEngine) Enrich(args *EnrichArgs, reply *EnrichReply) error {
	// Pass-through stub: returns the original event unchanged.
	// A real implementation would apply GeoIP, DNS, and asset mapping.
	reply.EnrichedEventJSON = args.RawEventJSON
	return nil
}

func workerCmd(args []string) {
	fs := flag.NewFlagSet("worker", flag.ExitOnError)
	wType := fs.String("type", "", "Type of worker to start (detect, policy, enrich)")
	fs.Parse(args)

	if *wType == "" {
		log.Fatal("Worker error: --type is required")
	}

	worker := isolation.NewIsolatedWorker(*wType)

	switch *wType {
	case "detect":
		if err := worker.Register("DetectEngine", &WorkerDetectEngine{}); err != nil {
			log.Fatalf("Worker: failed to register DetectEngine: %v", err)
		}
	case "policy":
		if err := worker.Register("PolicyEngine", &WorkerPolicyEngine{}); err != nil {
			log.Fatalf("Worker: failed to register PolicyEngine: %v", err)
		}
	case "enrich":
		if err := worker.Register("EnrichEngine", &WorkerEnrichEngine{}); err != nil {
			log.Fatalf("Worker: failed to register EnrichEngine: %v", err)
		}
	default:
		log.Fatalf("Worker error: unknown worker type %q (valid: detect, policy, enrich)", *wType)
	}

	// Serve forever on standard I/O
	worker.ServeStdinStdout(os.Stdin, os.Stdout)
}
