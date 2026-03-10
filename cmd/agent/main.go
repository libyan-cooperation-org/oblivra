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
	// Agent configuration flags
	serverAddr := flag.String("server", "localhost:8443", "OBLIVRA server address (host:port)")
	dataDir := flag.String("data-dir", agentDefaultDataDir(), "Local data directory for WAL and cache")
	interval := flag.Int("interval", 30, "Collection interval in seconds")
	enableFIM := flag.Bool("fim", false, "Enable File Integrity Monitoring")
	enableSyslog := flag.Bool("syslog", true, "Enable syslog forwarding")
	enableMetrics := flag.Bool("metrics", true, "Enable system metrics collection")
	enableEventLog := flag.Bool("eventlog", false, "Enable Windows Event Log collection")
	tlsCert := flag.String("tls-cert", "", "Path to TLS client certificate (mTLS)")
	tlsKey := flag.String("tls-key", "", "Path to TLS client key (mTLS)")
	tlsCA := flag.String("tls-ca", "", "Path to CA certificate for server verification")
	showVersion := flag.Bool("version", false, "Print version and exit")
	logJSON := flag.Bool("log-json", false, "Enable JSON structured logging")
	logPath := flag.String("log-path", "", "Path to agent log file (default: <data-dir>/agent.log)")

	flag.Parse()

	// Default log path if not provided
	finalLogPath := *logPath
	if finalLogPath == "" {
		finalLogPath = filepath.Join(*dataDir, "agent.log")
	}

	if *showVersion {
		fmt.Printf("oblivra-agent %s (%s) %s/%s\n", version, buildTime, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// Initialize logger
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

	l.Info("OBLIVRA Agent %s starting on %s/%s", version, runtime.GOOS, runtime.GOARCH)
	l.Info("  Server:   %s", *serverAddr)
	l.Info("  DataDir:  %s", *dataDir)
	l.Info("  LogPath:  %s", finalLogPath)

	// Create agent configuration
	cfg := agent.Config{
		ServerAddr:     *serverAddr,
		DataDir:        *dataDir,
		Interval:       time.Duration(*interval) * time.Second,
		EnableFIM:      *enableFIM,
		EnableSyslog:   *enableSyslog,
		EnableMetrics:  *enableMetrics,
		EnableEventLog: *enableEventLog,
		TLSCert:        *tlsCert,
		TLSKey:         *tlsKey,
		TLSCA:          *tlsCA,
		Version:        version,
	}

	// Build and start agent
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

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	l.Warn("Received %s, shutting down...", sig)

	cancel()
	a.Stop()
	l.Info("Agent stopped.")
}

func agentDefaultDataDir() string {
	if runtime.GOOS == "windows" {
		return `C:\ProgramData\oblivra\agent`
	}
	return "/var/lib/oblivra/agent"
}
