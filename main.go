package main

import (
	"context"
	"embed"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/isolation"
	"github.com/wailsapp/wails/v3/pkg/application"
						
	"github.com/kingknull/oblivrashell/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "worker" {
		workerCmd(os.Args[2:])
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "server" {
		serverCmd(os.Args[2:])
		return
	}

	// Sovereign Core: Runtime Binary Attestation
	attestSvc := attestation.NewAttestationService()
	if err := attestSvc.VerifyOnStartup(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}

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
Title: "OblivraShell",
Width: 1280,
Height: 800,
MinWidth: 900,
MinHeight: 600,
Frameless: true,
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

func serverCmd(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	port := fs.Int("port", 8080, "Port for the headless API server")
	fs.Parse(args)

	// Runtime Binary Attestation
	attestSvc := attestation.NewAttestationService()
	if err := attestSvc.VerifyOnStartup(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	oblivraApp := app.New()
	
	// Create context that listens for interrupt signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// In server mode, we don't have Wails calling Startup, so we do it manually.
	oblivraApp.Startup(ctx)

	log.Printf("[SERVER] OBLIVRA Headless Core active on port %d\n", *port)
	log.Println("[SERVER] Press Ctrl+C to shut down")

	<-ctx.Done()
	log.Println("[SERVER] Shutting down...")
	
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oblivraApp.Shutdown(shutdownCtx)
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
