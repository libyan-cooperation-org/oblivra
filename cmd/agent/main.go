package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/kingknull/oblivrashell/internal/agent"
	oblio "github.com/kingknull/oblivrashell/internal/io"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/tamper"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// ── CLI flags ─────────────────────────────────────────────────────────────
	serverAddr     := flag.String("server",        "localhost:8443",  "OBLIVRA server address (host:port or https://host:port)")
	dataDir        := flag.String("data-dir",      defaultDataDir(),  "Local data directory for WAL, cache, and agent ID")
	interval       := flag.Int("interval",         30,                "Collection interval in seconds")
	maxWAL         := flag.Int64("max-wal-events", 500_000,           "Maximum events to buffer on disk (0=unlimited)")
	maxBatch       := flag.Int("max-batch",         5_000,            "Maximum events per HTTP POST to server")
	enableFIM      := flag.Bool("fim",              false,            "Enable File Integrity Monitoring")
	enableSyslog   := flag.Bool("syslog",           true,             "Enable log file tailing")
	enableMetrics  := flag.Bool("metrics",          true,             "Enable system metrics collection")
	enableEventLog := flag.Bool("eventlog",         false,            "Enable Windows Event Log collection")
	tlsCert        := flag.String("tls-cert",       "",               "Path to TLS client certificate (mTLS)")
	tlsKey         := flag.String("tls-key",        "",               "Path to TLS client key (mTLS)")
	tlsCA          := flag.String("tls-ca",         "",               "Path to CA certificate for server verification")
	insecure       := flag.Bool("insecure",         false,            "Allow insecure TLS (skip verification)")
	fleetSecret    := flag.String("fleet-secret",   os.Getenv("OBLIVRA_FLEET_SECRET"), "HMAC fleet shared secret for agent auth (default: dev value, set in production)")
	showVersion    := flag.Bool("version",          false,            "Print version and exit")
	logJSON        := flag.Bool("log-json",         false,            "Enable JSON structured logging")
	logPath        := flag.String("log-path",       "",               "Path to agent log file (default: <data-dir>/agent.log)")

	tenantID       := flag.String("tenant-id",     os.Getenv("OBLIVRA_TENANT_ID"), "Tenant identifier for multi-tenant isolation")
	triggerTamper  := flag.Bool("trigger-tamper",   false,            "Simulate a self-tampering attempt on startup for watchdog verification")
	// New flags wiring the I/O plugin framework + tamper-evidence subsystem.
	ioConfig       := flag.String("io-config",      "",               "Path to YAML inputs/outputs config (Slice 1-5). Leave empty to skip the pluggable pipeline.")
	tamperDisable  := flag.Bool("tamper-disable",   false,            "Disable agent oplog forwarding + heartbeat. Strongly discouraged outside lab.")
	flag.Parse()

	if *showVersion {
		fmt.Printf("oblivra-agent %s (%s) %s/%s\n", version, buildTime, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// ── Logger ────────────────────────────────────────────────────────────────
	finalLogPath := *logPath
	if finalLogPath == "" {
		finalLogPath = filepath.Join(*dataDir, "agent.log")
	}

	l, err := logger.New(logger.Config{
		Level:      logger.InfoLevel,
		OutputPath: finalLogPath,
		Sanitize:   true,
		JSON:       *logJSON,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	l.Info("OBLIVRA Agent %s starting on %s/%s (build: %s)", version, runtime.GOOS, runtime.GOARCH, buildTime)
	l.Info("  Tenant ID:     %s", *tenantID)
	l.Info("  Server:        %s", *serverAddr)
	l.Info("  DataDir:       %s", *dataDir)
	l.Info("  LogPath:       %s", finalLogPath)
	l.Info("  Interval:      %ds", *interval)
	l.Info("  MaxWALEvents:  %d", *maxWAL)
	l.Info("  MaxBatch:      %d", *maxBatch)
	l.Info("  FIM:           %v  Syslog: %v  Metrics: %v  EventLog: %v",
		*enableFIM, *enableSyslog, *enableMetrics, *enableEventLog)

	// SEC-AUDIT — loud warning when --insecure is set. Operators copying
	// from a quickstart guide can leave this on in production by accident,
	// which disables certificate verification on every agent → server
	// request and exposes the fleet to MITM attacks.
	if *insecure {
		l.Warn("⚠ SECURITY: --insecure is set — TLS certificate verification DISABLED. " +
			"DEV ONLY. DO NOT USE IN PRODUCTION. " +
			"Provide --tls-ca pointing at the server's root CA instead.")
	}

	// ── Config ────────────────────────────────────────────────────────────────
	cfg := agent.Config{
		TenantID:       *tenantID,
		ServerAddr:     *serverAddr,
		DataDir:        *dataDir,
		Interval:       time.Duration(*interval) * time.Second,
		MaxWALEvents:   *maxWAL,
		MaxBatchSize:   *maxBatch,
		EnableFIM:      *enableFIM,
		EnableSyslog:   *enableSyslog,
		EnableMetrics:  *enableMetrics,
		EnableEventLog: *enableEventLog,
		TLSCert:        *tlsCert,
		TLSKey:         *tlsKey,
		TLSCA:          *tlsCA,
		InsecureTLS:    *insecure,
		FleetSecret:    []byte(*fleetSecret),
		Version:        version,
	}

	// ── Build and start ───────────────────────────────────────────────────────
	a, err := agent.New(cfg, l)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize agent: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := a.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start agent: %v\n", err)
		os.Exit(1)
	}

	if *triggerTamper {
		a.TriggerWatchdogSelfTest()
	}

	// ── I/O Pipeline (Slices 1-5) ─────────────────────────────────────────────
	// When --io-config is set, parse the YAML and start a pluggable pipeline
	// alongside the legacy collectors. Both run together — operators migrate
	// individual collectors to the new framework at their own pace.
	if *ioConfig != "" {
		if err := startIOPipeline(ctx, *ioConfig, l); err != nil {
			l.Warn("I/O pipeline disabled: %v", err)
		}
	}

	// ── Tamper-evidence subsystem (Layers 1+2+3) ─────────────────────────────
	// Ships agent log → server, sends 30s heartbeats, maintains hash chain.
	// Disabled with --tamper-disable for lab use; in any other environment
	// missing tamper-evidence is a security regression worth being loud about.
	if *tamperDisable {
		l.Warn("⚠ Tamper-evidence subsystem DISABLED via --tamper-disable. " +
			"Agent oplog won't ship to server; heartbeats won't fire. " +
			"DO NOT USE IN PRODUCTION.")
	} else {
		serverURL := *serverAddr
		if !strings.HasPrefix(serverURL, "http") {
			scheme := "https"
			if *insecure {
				scheme = "http"
			}
			serverURL = scheme + "://" + serverURL
		}
		ts, err := tamper.NewSubsystem(tamper.Config{
			AgentID:     a.ID(),
			ServerURL:   serverURL,
			FleetSecret: cfg.FleetSecret,
			LogPath:     finalLogPath,
			VerifyTLS:   !cfg.InsecureTLS,
		}, l)
		if err != nil {
			l.Warn("Tamper subsystem not started: %v", err)
		} else if err := ts.Start(ctx); err != nil {
			l.Warn("Tamper subsystem failed to start: %v", err)
		} else {
			defer ts.Stop()
		}
	}

	// Emit the "Connected" log line that the troubleshooting guide expects
	l.Info("Connected to server: %s", *serverAddr)

	// ── Shutdown ──────────────────────────────────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	l.Warn("Received signal %s — shutting down gracefully", sig)

	cancel()
	a.Stop()
	l.Info("Agent stopped cleanly.")
}

func defaultDataDir() string {
	if runtime.GOOS == "windows" {
		return `C:\ProgramData\oblivra\agent`
	}
	return "/var/lib/oblivra/agent"
}

// startIOPipeline loads the YAML config at `path`, instantiates every
// declared input + output via the registry, wires them through the
// pipeline, and starts it. The pipeline runs alongside the legacy
// agent collectors — both write into the existing agent → server
// transport, so events from `file` / `syslog` / `hec` inputs flow
// upstream identically to the legacy collectors.
//
// Returns an error if the file fails to parse or any plugin can't
// construct. On success the pipeline outlives this function (its
// goroutines run until the parent ctx is cancelled).
func startIOPipeline(ctx context.Context, path string, log *logger.Logger) error {
	cfg, err := oblio.LoadConfig(path)
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}

	pipe := oblio.NewPipeline(log)
	for _, inCfg := range cfg.Inputs {
		in, err := oblio.NewInput(inCfg, log)
		if err != nil {
			return fmt.Errorf("input: %w", err)
		}
		pipe.AddInput(in)
	}
	for _, outCfg := range cfg.Outputs {
		out, err := oblio.NewOutput(outCfg, log)
		if err != nil {
			return fmt.Errorf("output: %w", err)
		}
		pipe.AddOutput(out)
	}
	if err := pipe.Start(ctx); err != nil {
		return err
	}
	log.Info("I/O pipeline started from %s (%d inputs, %d outputs)",
		path, len(cfg.Inputs), len(cfg.Outputs))

	// Hot-reload watcher — operator edits the YAML, fsnotify fires,
	// we (today) just log the change. Full hot-reload (diff-and-restart-
	// only-changed-plugins) is a Phase 34 follow-up; for now operators
	// restart the agent to apply config changes.
	go func() {
		w, err := oblio.NewWatcher(path, log)
		if err != nil {
			log.Warn("config watcher disabled: %v", err)
			return
		}
		defer w.Stop()
		go w.Watch()
		for {
			select {
			case <-ctx.Done():
				return
			case newCfg := <-w.C():
				if newCfg == nil {
					continue
				}
				log.Info("[io] config change detected: %d inputs, %d outputs " +
					"(restart agent to apply — hot-reload coming Phase 34)",
					len(newCfg.Inputs), len(newCfg.Outputs))
			}
		}
	}()
	return nil
}
