// oblivra-agent — Splunk-Forwarder-grade log shipper.
//
// Subcommands:
//
//	oblivra-agent init       — write a sample config to the default path
//	oblivra-agent run        — start tailing inputs and forwarding events
//	oblivra-agent status     — show queue depth, tail positions, last delivery
//	oblivra-agent reload     — re-read config (or send SIGHUP to a running agent)
//	oblivra-agent version    — print build version
//
// Configuration:
//
//   - Config file (YAML) at /etc/oblivra/agent.yml or %PROGRAMDATA%\oblivra\agent.yml
//   - --config FILE overrides the default path
//   - Token can be inlined OR loaded from a 0600-mode tokenFile
//   - mTLS supported (clientCertFile + clientKeyFile)
//   - Server-cert pinning via SHA-256 of the public key (air-gap deployments)
//
// Compatibility highlights:
//
//   - Multiple typed inputs per agent (file / stdin / winlog / syslog-udp)
//   - Multiline event stitching via regex startPattern
//   - Per-input field injection (sourcetype, env tags, etc.)
//   - Position tracking — restart resumes where we left off, log-rotate aware
//   - On-disk spill+replay if the server is unreachable
//   - gzip compression on the wire
//   - Heartbeat to /api/v1/agent/ingest so the Fleet view shows live agents
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Build-time version. Set via `-ldflags "-X main.version=v0.1.0"`.
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]

	switch cmd {
	case "init":
		runInit(os.Args[2:])
	case "run":
		runForward(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "reload":
		runReload(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("oblivra-agent %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `oblivra-agent — log forwarder

Usage:
  oblivra-agent init       Write a commented sample config to the default path
  oblivra-agent run        Start forwarding events to the platform
  oblivra-agent status     Show tail positions + queue depth
  oblivra-agent reload     Re-read config (or signal a running agent via SIGHUP)
  oblivra-agent version    Print build version

Common flags:
  --config FILE            Override the default config path
  --pipe                   (run) read newline-delimited events from stdin

Environment:
  OBLIVRA_AGENT_CONFIG     Override the default config path

Default config path:
  Linux/macOS: /etc/oblivra/agent.yml
  Windows:     %PROGRAMDATA%\oblivra\agent.yml`)
}

// ---- init ---------------------------------------------------------------

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	configPath := fs.String("config", DefaultConfigPath(), "config file path")
	force := fs.Bool("force", false, "overwrite existing config")
	_ = fs.Parse(args)

	if _, err := os.Stat(*configPath); err == nil && !*force {
		fmt.Fprintf(os.Stderr, "%s exists; use --force to overwrite\n", *configPath)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*configPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}
	host, _ := os.Hostname()
	if err := os.WriteFile(*configPath, []byte(SampleConfigYAML(host)), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote sample config to %s\n", *configPath)
	fmt.Println("Edit it (server.url + token + inputs) then run: oblivra-agent run")
}

// ---- run ----------------------------------------------------------------

func runForward(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", DefaultConfigPath(), "config file path")
	pipeMode := fs.Bool("pipe", false, "ignore inputs in config and read stdin instead")
	_ = fs.Parse(args)

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	if *pipeMode {
		cfg.Inputs = []Input{{Type: "stdin", Label: "pipe", HostID: cfg.Hostname}}
	}

	posStore, err := NewPositionStore(cfg.StateDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "state dir:", err)
		os.Exit(1)
	}

	client, err := NewClient(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "client:", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// SIGHUP reload — re-spawn run subcommand. Simpler than dynamic reconfiguration.
	if hupCh := setupHUP(); hupCh != nil {
		go func() {
			for range hupCh {
				log.Println("SIGHUP — exiting; service supervisor will restart with new config")
				stop()
				return
			}
		}()
	}

	// Register with the server (best-effort; don't block on registration).
	go func() {
		regCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if id, err := client.RegisterAgent(regCtx, cfg.Hostname, runtime.GOOS, runtime.GOARCH, version, cfg.Tags); err != nil {
			log.Printf("register: %v", err)
		} else {
			log.Printf("registered as %s", id)
		}
	}()

	if err := os.MkdirAll(cfg.Buffer.Dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "buffer dir:", err)
		os.Exit(1)
	}

	queue := make(chan string, 8192)

	// Spawn one tailer per input.
	var wg sync.WaitGroup
	for _, in := range cfg.Inputs {
		t, err := NewTailer(cfg, in, queue, posStore)
		if err != nil {
			log.Printf("input %q: %v", in.Label, err)
			continue
		}
		wg.Add(1)
		go func(t *Tailer) {
			defer wg.Done()
			if err := t.Run(ctx); err != nil {
				log.Printf("tailer: %v", err)
			}
		}(t)
	}

	// Heartbeat goroutine (optional).
	if cfg.Heartbeat.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tick := time.NewTicker(cfg.Heartbeat.Interval)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					ctx2, cancel := context.WithTimeout(ctx, cfg.Server.RequestTimeout)
					if err := client.Heartbeat(ctx2, cfg.Hostname, ""); err != nil {
						log.Printf("heartbeat: %v", err)
					}
					cancel()
				}
			}
		}()
	}

	// Forwarder goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		runBatcher(ctx, cfg, client, queue)
	}()

	log.Printf("oblivra-agent %s started — %d inputs → %s",
		version, len(cfg.Inputs), cfg.Server.URL)
	wg.Wait()
	log.Printf("oblivra-agent: shutdown")
}

// runBatcher is the outbound side: drain queue, batch, post, spill+replay
// on failure, respect MaxBytes disk-buffer cap.
func runBatcher(ctx context.Context, cfg *Config, client *Client, queue <-chan string) {
	var pending []string
	timer := time.NewTimer(cfg.FlushEvery)
	defer timer.Stop()

	send := func() {
		if len(pending) == 0 {
			return
		}
		ctx2, cancel := context.WithTimeout(ctx, cfg.Server.RequestTimeout)
		defer cancel()
		if err := client.PostBatch(ctx2, pending); err != nil {
			spillToDisk(cfg, pending)
			log.Printf("send failed (%d events spilled to disk): %v", len(pending), err)
		} else {
			replaySpilled(ctx, cfg, client)
		}
		pending = pending[:0]
	}

	for {
		select {
		case <-ctx.Done():
			send()
			return
		case ev := <-queue:
			pending = append(pending, ev)
			if len(pending) >= cfg.BatchSize {
				send()
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(cfg.FlushEvery)
			}
		case <-timer.C:
			send()
			timer.Reset(cfg.FlushEvery)
		}
	}
}

// spillToDisk persists a batch under the buffer dir. Honors the MaxBytes
// cap by deleting the oldest spill files first.
func spillToDisk(cfg *Config, items []string) {
	name := filepath.Join(cfg.Buffer.Dir, fmt.Sprintf("spill-%d.jsonl", time.Now().UnixNano()))
	f, err := os.Create(name)
	if err != nil {
		log.Printf("spill: %v", err)
		return
	}
	for _, it := range items {
		fmt.Fprintln(f, it)
	}
	_ = f.Sync()
	_ = f.Close()
	enforceBufferCap(cfg)
}

func enforceBufferCap(cfg *Config) {
	entries, err := os.ReadDir(cfg.Buffer.Dir)
	if err != nil {
		return
	}
	type fi struct {
		name string
		size int64
		mod  time.Time
	}
	var spills []fi
	var total int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "spill-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		spills = append(spills, fi{name: e.Name(), size: info.Size(), mod: info.ModTime()})
		total += info.Size()
	}
	if total <= cfg.Buffer.MaxBytes {
		return
	}
	// Sort oldest-first, delete until under cap.
	for i := 0; i < len(spills)-1; i++ {
		for j := i + 1; j < len(spills); j++ {
			if spills[j].mod.Before(spills[i].mod) {
				spills[i], spills[j] = spills[j], spills[i]
			}
		}
	}
	for _, s := range spills {
		if total <= cfg.Buffer.MaxBytes {
			break
		}
		if err := os.Remove(filepath.Join(cfg.Buffer.Dir, s.name)); err == nil {
			total -= s.size
			log.Printf("spill cap exceeded; dropped %s (%d bytes)", s.name, s.size)
		}
	}
}

func replaySpilled(ctx context.Context, cfg *Config, client *Client) {
	entries, err := os.ReadDir(cfg.Buffer.Dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "spill-") {
			continue
		}
		path := filepath.Join(cfg.Buffer.Dir, e.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(body)), "\n")
		ctx2, cancel := context.WithTimeout(ctx, cfg.Server.RequestTimeout)
		err = client.PostBatch(ctx2, lines)
		cancel()
		if err != nil {
			return
		}
		_ = os.Remove(path)
		log.Printf("replayed spill %s (%d events)", e.Name(), len(lines))
	}
}

// ---- status -------------------------------------------------------------

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	configPath := fs.String("config", DefaultConfigPath(), "config file path")
	jsonOut := fs.Bool("json", false, "machine-readable JSON")
	_ = fs.Parse(args)

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	posStore, _ := NewPositionStore(cfg.StateDir)
	positions := posStore.All()

	type bufInfo struct {
		File string `json:"file"`
		Size int64  `json:"size"`
	}
	var spills []bufInfo
	if entries, err := os.ReadDir(cfg.Buffer.Dir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasPrefix(e.Name(), "spill-") {
				continue
			}
			info, _ := e.Info()
			if info != nil {
				spills = append(spills, bufInfo{File: e.Name(), Size: info.Size()})
			}
		}
	}

	report := map[string]any{
		"version":   version,
		"hostname":  cfg.Hostname,
		"server":    cfg.Server.URL,
		"tenant":    cfg.Tenant,
		"inputs":    len(cfg.Inputs),
		"positions": positions,
		"spills":    spills,
		"runtime":   hostUname(),
		"pid":       pid(),
		"checkedAt": time.Now().UTC(),
	}

	// Optional live ping.
	client, err := NewClient(cfg)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if perr := client.HealthCheck(ctx); perr == nil {
			report["server.healthz"] = "ok"
		} else {
			report["server.healthz"] = perr.Error()
		}
		cancel()
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return
	}

	fmt.Printf("oblivra-agent %s on %s\n", version, cfg.Hostname)
	fmt.Printf("server:    %s\n", cfg.Server.URL)
	fmt.Printf("tenant:    %s\n", cfg.Tenant)
	fmt.Printf("inputs:    %d configured\n", len(cfg.Inputs))
	fmt.Printf("state dir: %s\n", cfg.StateDir)
	fmt.Printf("buffer:    %d spill files\n", len(spills))
	fmt.Println()
	if h, ok := report["server.healthz"]; ok {
		fmt.Printf("server /healthz: %v\n", h)
	}
	fmt.Println()
	fmt.Println("tail positions:")
	if len(positions) == 0 {
		fmt.Println("  (none yet — run `oblivra-agent run` first)")
	}
	for _, p := range positions {
		fmt.Printf("  %s\n    offset=%d size=%d\n", p.Path, p.Off, p.Size)
	}
}

// ---- reload -------------------------------------------------------------

func runReload(_ []string) {
	if runtime.GOOS == "windows" {
		fmt.Fprintln(os.Stderr, "reload not supported on Windows; use service restart")
		os.Exit(1)
	}
	// On Unix we look for the agent's PID file; absent that, tell the user
	// to send SIGHUP themselves. Keeping this simple — if you have the PID,
	// `kill -HUP` is the documented mechanism.
	fmt.Println("Send SIGHUP to the running oblivra-agent process to reload:")
	fmt.Println("  pkill -HUP oblivra-agent")
}

// setupHUP returns nil on Windows.
func setupHUP() chan os.Signal { return setupHUPPlatform() }

// silenceUnused keeps imports we want available across platforms.
var _ = io.Discard
