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
	"runtime/debug"
	"syscall"
	"time"

	"github.com/kingknull/oblivrashell/internal/attestation"
	"github.com/kingknull/oblivrashell/internal/database"
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

	if len(os.Args) >= 2 && os.Args[1] == "bootcheck" {
		bootcheckCmd(os.Args[2:])
		return
	}

	runGUI()
}

// bootcheckCmd runs the same container Init + service Start sequence as
// the GUI binary, then exits 0 if no panic / no Start() error fired.
// Designed to be the canonical CI smoke-test command — catches the class
// of regression where a service's Start() panics on nil dependencies
// (e.g. db.DB() == nil at boot time, observed 2026-04-28).
//
// Usage: oblivrashell bootcheck [--timeout=15s]
func bootcheckCmd(args []string) {
	fs := flag.NewFlagSet("bootcheck", flag.ExitOnError)
	timeout := fs.Duration("timeout", 15*time.Second, "Max time to wait for services to start")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "[bootcheck] flag parse: %v\n", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Same construction path as runGUI — but no Wails window, no UI loop.
	// We just exercise app.New() (which calls container.Init()) and then
	// run Startup() to fire every Service.Start(). Any nil-deref or
	// returned error from a service's Start surfaces here.
	done := make(chan struct{})
	var bootErr error
	go func() {
		defer close(done)
		defer func() {
			if rec := recover(); rec != nil {
				// Include the stack trace so an operator running
				// bootcheck in CI can diagnose nil-receiver panics
				// without re-running with GOTRACEBACK=all.
				bootErr = fmt.Errorf("panic during boot: %v\n%s", rec, debug.Stack())
			}
		}()
		oblivraApp := app.New()
		oblivraApp.Startup(ctx)
		// Brief settle window for late-init goroutines (NATS, BadgerDB
		// compaction, agent registry hydration, etc.). If they panic
		// inside this window, the deferred recover catches it.
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
		}
		oblivraApp.Shutdown(context.Background())
	}()

	select {
	case <-done:
		if bootErr != nil {
			fmt.Fprintf(os.Stderr, "[bootcheck] FAILED: %v\n", bootErr)
			os.Exit(1)
		}
		fmt.Println("[bootcheck] OK — container Init + service Start completed without panic")
		os.Exit(0)
	case <-ctx.Done():
		fmt.Fprintf(os.Stderr, "[bootcheck] FAILED: timeout after %v — services hung during start\n", *timeout)
		os.Exit(1)
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
	database.EnforceStrictIsolation = true
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
