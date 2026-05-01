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
	case "test":
		runTest(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "reload":
		runReload(os.Args[2:])
	case "service":
		runService(os.Args[2:])
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
  oblivra-agent test       Parse a sample of inputs and print what would be sent (no shipping)
  oblivra-agent status     Show tail positions + queue depth
  oblivra-agent reload     Re-read config (or signal a running agent via SIGHUP)
  oblivra-agent service    Install / remove the agent as an OS service (systemd / Windows)
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

	// Per-agent ed25519 keypair for at-the-edge signing. Operator copies the
	// generated *.pub file into the platform's allow-list.
	var signer *Signer
	if cfg.SignEvents {
		s, err := LoadOrCreateSigner(cfg.StateDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "signer:", err)
			os.Exit(1)
		}
		signer = s
		log.Printf("signing enabled — pubkey %s (in %s/agent.ed25519.pub)",
			signer.FingerprintShort(), cfg.StateDir)
	}

	spill, err := NewSpillEncryption(cfg.SpillSecret, cfg.Hostname)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spill encryption:", err)
		os.Exit(1)
	}
	log.Printf("disk spill: %s (key fingerprint %s)",
		map[bool]string{true: "encrypted", false: "plaintext"}[spill.enabled],
		spill.SpillFingerprint())

	client, err := NewClient(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "client:", err)
		os.Exit(1)
	}

	// Optional dual-egress clients — same payload, multiple destinations
	// for federation / redundancy.
	var extraClients []*Client
	for _, alt := range cfg.DualEgress {
		clone := *cfg
		clone.Server = alt
		c, err := NewClient(&clone)
		if err != nil {
			log.Printf("dualEgress %s: %v", alt.URL, err)
			continue
		}
		extraClients = append(extraClients, c)
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
	hiQueue := make(chan string, 1024)

	var rules []LocalRule
	if cfg.LocalRules {
		rules = DefaultLocalRules()
	}

	// Spawn one tailer per input.
	var wg sync.WaitGroup
	for _, in := range cfg.Inputs {
		t, err := NewTailer(cfg, in, TailerDeps{
			Queue: queue, HiQueue: hiQueue, Signer: signer, Rules: rules,
		}, posStore)
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
		runBatcher(ctx, cfg, client, extraClients, queue, hiQueue, spill)
	}()

	log.Printf("oblivra-agent %s started — %d inputs → %s",
		version, len(cfg.Inputs), cfg.Server.URL)
	wg.Wait()
	log.Printf("oblivra-agent: shutdown")
}

// runBatcher drains the queue, batches, posts, spills+replays on failure,
// respects MaxBytes, fans out to dual-egress targets, and (when enabled)
// adapts batch size to observed p99 latency.
//
// Priority queue (`hi`) is drained first so high-severity events ship
// ahead of routine events under backpressure.
func runBatcher(
	ctx context.Context, cfg *Config,
	primary *Client, extras []*Client,
	queue, hi <-chan string,
	spill *SpillEncryption,
) {
	var pending []string
	timer := time.NewTimer(cfg.FlushEvery)
	defer timer.Stop()

	currentBatch := cfg.BatchSize
	currentFlush := cfg.FlushEvery

	send := func() {
		if len(pending) == 0 {
			return
		}
		ctx2, cancel := context.WithTimeout(ctx, cfg.Server.RequestTimeout)
		start := time.Now()
		err := primary.PostBatch(ctx2, pending)
		elapsed := time.Since(start)
		cancel()
		if err != nil {
			path, werr := spill.WriteSpill(cfg.Buffer.Dir, pending)
			if werr != nil {
				log.Printf("spill write failed: %v", werr)
			} else {
				log.Printf("send failed (%d events → %s): %v", len(pending), filepath.Base(path), err)
				enforceBufferCap(cfg)
			}
		} else {
			replaySpilled(ctx, cfg, primary, spill)
			// Mirror to dual-egress targets — best-effort, errors logged only.
			for _, c := range extras {
				ec, can := context.WithTimeout(ctx, cfg.Server.RequestTimeout)
				if err := c.PostBatch(ec, pending); err != nil {
					log.Printf("dualEgress %s: %v", c.server, err)
				}
				can()
			}
		}
		pending = pending[:0]

		// Adaptive batching — if a send took >2× the flush interval, shrink
		// the batch (reduces server-side queue depth). If consistently fast,
		// grow the batch (better throughput).
		if cfg.AdaptiveBatch {
			switch {
			case elapsed > currentFlush*2 && currentBatch > 25:
				currentBatch = currentBatch * 3 / 4
				log.Printf("adaptive: shrinking batch → %d (last send %s)", currentBatch, elapsed)
			case elapsed < currentFlush/3 && currentBatch < 500:
				currentBatch += 25
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			send()
			return
		case ev := <-hi:
			// Priority event — flush immediately to minimise time-to-server.
			pending = append(pending, ev)
			send()
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(currentFlush)
		case ev := <-queue:
			pending = append(pending, ev)
			if len(pending) >= currentBatch {
				send()
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(currentFlush)
			}
		case <-timer.C:
			send()
			timer.Reset(currentFlush)
		}
	}
}

// (spillToDisk replaced by SpillEncryption.WriteSpill — kept here as a
// no-op stub so older imports compile if anyone re-introduces a caller.)
//
//nolint:unused
func spillToDisk(cfg *Config, items []string) {
	if cfg == nil || items == nil {
		return
	}
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
		if e.IsDir() || !isSpillFile(e.Name()) {
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
	for i := 0; i < len(spills)-1; i++ {
		for j := i + 1; j < len(spills); j++ {
			if spills[j].mod.Before(spills[i].mod) {
				spills[i], spills[j] = spills[j], spills[i]
			}
		}
	}
	dropped := 0
	for _, s := range spills {
		if total <= cfg.Buffer.MaxBytes {
			break
		}
		if err := os.Remove(filepath.Join(cfg.Buffer.Dir, s.name)); err == nil {
			total -= s.size
			dropped++
		}
	}
	if dropped > 0 {
		log.Printf("spill cap exceeded — evicted %d oldest spill files", dropped)
	}
}

func isSpillFile(name string) bool {
	return strings.HasPrefix(name, "spill-") || strings.HasPrefix(name, "spill.enc-")
}

func replaySpilled(ctx context.Context, cfg *Config, client *Client, spill *SpillEncryption) {
	entries, err := os.ReadDir(cfg.Buffer.Dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !isSpillFile(e.Name()) {
			continue
		}
		path := filepath.Join(cfg.Buffer.Dir, e.Name())
		lines, err := spill.ReadSpill(path)
		if err != nil {
			log.Printf("spill read %s: %v", e.Name(), err)
			continue
		}
		if len(lines) == 0 {
			_ = os.Remove(path)
			continue
		}
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

// ---- test ---------------------------------------------------------------

// runTest parses N lines from a sample input and prints exactly what would
// be sent — including which local rule fired. Use it before deploying to
// validate regex extracts, multiline patterns, and tagging.
func runTest(args []string) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	configPath := fs.String("config", DefaultConfigPath(), "config file path")
	inputLabel := fs.String("input", "", "filter to a specific input label (optional)")
	count := fs.Int("count", 10, "max events to render per input")
	verify := fs.Bool("verify-sig", false, "if signing is enabled, verify each signed event locally")
	_ = fs.Parse(args)

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	var signer *Signer
	if cfg.SignEvents {
		s, err := LoadOrCreateSigner(cfg.StateDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "signer:", err)
			os.Exit(1)
		}
		signer = s
	}

	rules := []LocalRule{}
	if cfg.LocalRules {
		rules = DefaultLocalRules()
	}

	queue := make(chan string, *count*4)
	hi := make(chan string, *count*4)

	posDir, _ := os.MkdirTemp("", "agent-test-")
	defer os.RemoveAll(posDir)
	pos, _ := NewPositionStore(posDir)

	for _, in := range cfg.Inputs {
		if *inputLabel != "" && in.Label != *inputLabel {
			continue
		}
		// Force start-from-beginning for the test so we see real samples,
		// not just the tail of an idle file.
		in.StartFrom = "beginning"
		t, err := NewTailer(cfg, in, TailerDeps{
			Queue: queue, HiQueue: hi, Signer: signer, Rules: rules,
		}, pos)
		if err != nil {
			log.Printf("input %q: %v", in.Label, err)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		go t.Run(ctx)

		seen := 0
		fmt.Printf("\n=== input: %s (%s) ===\n", in.Label, in.Path)
	loop:
		for seen < *count {
			select {
			case ev := <-hi:
				renderEvent(ev, signer, *verify, true)
				seen++
			case ev := <-queue:
				renderEvent(ev, signer, *verify, false)
				seen++
			case <-ctx.Done():
				break loop
			}
		}
		cancel()
	}
}

func renderEvent(raw string, signer *Signer, verify, hi bool) {
	prefix := "  "
	if hi {
		prefix = "↑ "
	}
	fmt.Printf("%s%s\n", prefix, raw)
	if signer != nil && verify {
		if err := Verify(signer.PublicKeyB64(), []byte(raw)); err != nil {
			fmt.Printf("    SIGNATURE INVALID: %v\n", err)
		} else {
			fmt.Printf("    signature ✓ (key %s)\n", signer.FingerprintShort())
		}
	}
}

// ---- service install ----------------------------------------------------

func runService(args []string) {
	if len(args) == 0 || args[0] == "help" {
		fmt.Println(`oblivra-agent service — manage the agent as an OS service

Subcommands:
  install     Install systemd unit (Linux) or Windows service entry
  uninstall   Remove the service
  print       Print the unit/service definition without installing

Linux:   systemd unit at /etc/systemd/system/oblivra-agent.service
Windows: service registered via sc.exe`)
		return
	}
	switch args[0] {
	case "print":
		printServiceUnit()
	case "install":
		installService()
	case "uninstall":
		uninstallService()
	default:
		fmt.Fprintf(os.Stderr, "unknown service subcommand: %s\n", args[0])
		os.Exit(2)
	}
}

func printServiceUnit() {
	exe, _ := os.Executable()
	cfg := DefaultConfigPath()
	switch runtime.GOOS {
	case "linux":
		fmt.Printf(`# /etc/systemd/system/oblivra-agent.service
[Unit]
Description=OBLIVRA log forwarder
After=network.target

[Service]
Type=simple
User=oblivra
Group=oblivra
ExecStart=%s run --config %s
Restart=always
RestartSec=5
LimitNOFILE=65536
ProtectSystem=strict
ProtectHome=true
NoNewPrivileges=true
ReadWritePaths=%s

[Install]
WantedBy=multi-user.target
`, exe, cfg, defaultStateDir())
	case "windows":
		fmt.Printf(`REM Run as Administrator:
sc.exe create OblivraAgent binPath= "\"%s\" run --config \"%s\"" start= auto
sc.exe description OblivraAgent "OBLIVRA log forwarder"
sc.exe start OblivraAgent
`, exe, cfg)
	default:
		fmt.Println("Service install not supported on this OS — run oblivra-agent under your supervisor")
	}
}

func installService() {
	switch runtime.GOOS {
	case "linux":
		path := "/etc/systemd/system/oblivra-agent.service"
		exe, _ := os.Executable()
		body := fmt.Sprintf(`[Unit]
Description=OBLIVRA log forwarder
After=network.target

[Service]
Type=simple
User=oblivra
Group=oblivra
ExecStart=%s run --config %s
Restart=always
RestartSec=5
LimitNOFILE=65536
ProtectSystem=strict
ProtectHome=true
NoNewPrivileges=true
ReadWritePaths=%s

[Install]
WantedBy=multi-user.target
`, exe, DefaultConfigPath(), defaultStateDir())
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "install:", err)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s\n", path)
		fmt.Println("Run: systemctl daemon-reload && systemctl enable --now oblivra-agent")
	default:
		fmt.Println("Use 'oblivra-agent service print' to see the recipe for this OS, then install manually.")
	}
}

func uninstallService() {
	switch runtime.GOOS {
	case "linux":
		path := "/etc/systemd/system/oblivra-agent.service"
		_ = os.Remove(path)
		fmt.Println("Removed " + path + "; run: systemctl daemon-reload")
	default:
		fmt.Println("Use 'sc.exe delete OblivraAgent' on Windows; on others, remove the unit file you installed manually.")
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
