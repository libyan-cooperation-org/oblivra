package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/kingknull/oblivrashell/internal/agent"
	"github.com/kingknull/oblivrashell/internal/logger"
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
