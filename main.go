package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/isolation"
	"github.com/kingknull/oblivrashell/internal/app"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
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

	if len(os.Args) >= 2 && os.Args[1] == "replay" {
		replayCmd(os.Args[2:])
		return
	}

	runGUI()
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
	// SA-02: The policy engine MUST NOT default to allow.
	// This subprocess is designed to enforce tier/action policy in an isolated address space.
	// Until a real policy bundle is loaded from disk, all requests are denied with a clear reason
	// so callers fail closed rather than silently passing.
	//
	// TODO: Load policy bundle from the path provided via --policy-bundle flag and evaluate args.
	reply.Allowed = false
	reply.Reason = "policy-engine-not-configured: no policy bundle loaded; all actions denied by default"
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
func replayCmd(args []string) {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	walPath := fs.String("from", "", "Path to the WAL file to replay")
	fs.Parse(args)

	if *walPath == "" {
		log.Fatal("Replay error: --from <path> is required")
	}

	// Initialize isolated logger for replay
	l := logger.NewStdoutLogger()
	l.Info("╔══════════════════════════════════════════╗")
	l.Info("║      OBLIVRA EVENT REPLAY ENGINE         ║")
	l.Info("╚══════════════════════════════════════════╝")
	l.Info("Replaying from: %s", *walPath)

	replayer := ingest.NewEventReplayer(l)
	result, err := replayer.ReplayWAL(context.Background(), *walPath)
	if err != nil {
		l.Fatal("Replay failed: %v", err)
	}

	// Output summary
	l.Info("Replay complete!")
	l.Info("Total Processed: %d", result.TotalEvents)
	l.Info("Alerts Generated: %d", len(result.Alerts))
	l.Info("Duration: %v", result.Duration)

	// Print alerts in NDJSON for piping
	if len(result.Alerts) > 0 {
		fmt.Println("\n--- ALERTS (NDJSON) ---")
		for _, alert := range result.Alerts {
			data, _ := json.Marshal(alert)
			fmt.Println(string(data))
		}
	}
}
