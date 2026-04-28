package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"crypto/ed25519"
)

func resolveIdentityKey(dataDir string) (ed25519.PrivateKey, error) {
	path := filepath.Join(dataDir, "identity.key")
	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) == ed25519.PrivateKeySize {
			return ed25519.PrivateKey(data), nil
		}
	}

	// Generate new key
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, priv, 0600); err != nil {
		// Log but don't fail boot; agent will use the key in memory but will regenerate next time
		fmt.Fprintf(os.Stderr, "[agent] CRITICAL: failed to save identity key to %s: %v\n", path, err)
	}
	return priv, nil
}

// Config defines the agent configuration.
type Config struct {
	// AgentID is a stable, cryptographically random 16-byte identifier written
	// to <DataDir>/agent_id on first boot and reused on every subsequent start.
	// This ensures each agent is uniquely identifiable even when multiple agents
	// share the same hostname (containers, VMs).
	AgentID        string
	ServerAddr     string
	DataDir        string
	Interval       time.Duration
	EnableFIM      bool
	EnableSyslog   bool
	EnableMetrics  bool
	EnableEventLog bool
	TLSCert        string
	TLSKey         string
	TLSCA          string
	InsecureTLS    bool
	// FleetSecret is the shared HMAC secret the server validates
	// every agent request against (`internal/api/middleware.go:VerifyHMAC`).
	// Without this, the server returns 401 on `/api/v1/agent/ingest`
	// with "missing authentication headers (X-Timestamp/X-Signature)".
	// The default `oblivra-fleet-secret-v1` matches the dev value
	// hardcoded in `internal/services/api_service.go:142`. In
	// production this MUST be operator-supplied and identical on
	// every agent + the server.
	FleetSecret    []byte
	// TenantID is used for multi-tenant isolation.
	// Defaults to "GLOBAL" if not provided.
	TenantID string
	Version  string

	// MaxWALEvents caps the number of events buffered on disk before new events
	// are dropped. Prevents unbounded disk growth during prolonged server outage.
	MaxWALEvents int64
	// MaxBatchSize caps the number of events sent in a single HTTP POST.
	MaxBatchSize int
}

// FleetConfig is the structure received from the server for remote configuration updates.
type FleetConfig struct {
	Interval       time.Duration `json:"interval"`
	EnableFIM      bool          `json:"enable_fim"`
	EnableSyslog   bool          `json:"enable_syslog"`
	EnableMetrics  bool          `json:"enable_metrics"`
	EnableEventLog bool          `json:"enable_event_log"`
	Quarantine     bool          `json:"quarantine"`
}

// ActionType defines forensic or response operations the agent can perform.
type ActionType string

const (
	ActionKillProcess     ActionType = "kill_process"
	ActionProcessSnapshot ActionType = "process_snapshot"
	ActionProcessInventory ActionType = "process_inventory"
	ActionIsolateNetwork  ActionType = "isolate_network"
	ActionRestoreNetwork  ActionType = "restore_network"

	// ── Phase 30.5 / Phase 31 — operator remote-control actions ──
	// These are the three actions the HostDetail "Agent Control"
	// panel surfaces in the UI. They're routed through the same
	// PendingAction queue + handleAction switch as response-action
	// commands so there's one consistent server→agent control plane.
	ActionTriggerScan  ActionType = "trigger_scan"
	ActionToggleDebug  ActionType = "toggle_debug"
	ActionRestartAgent ActionType = "restart_agent"
)

// PendingAction represents a command waiting for an agent to pull.
type PendingAction struct {
	ID      string            `json:"id"`
	Type    ActionType        `json:"type"`
	Payload map[string]string `json:"payload"`
}

// Agent is the main agent process that manages collectors and transport.
type Agent struct {
	cfg      Config
	eventCh  chan Event // shared channel across all collectors
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc

	collectors []Collector
	transport  *Transport
	wal        *WAL
	redactor   *PIIRedactor
	response   *ResponseActionExecutor
	wg         sync.WaitGroup
	log        *logger.Logger
	hostname   string // Host identifier resolved at startup
	privKey    ed25519.PrivateKey // 1.4: Sovereign identity key
	watchdog   *Watchdog // 5: Sovereign self-protection

	// Phase 30.5 / Phase 31 operator-control plumbing.
	// `detector` runs the agent-side local rules (SSH brute-force,
	// suspicious sudo, discovery commands). nil-safe — when unset,
	// local detection is disabled silently.
	// `restartMgr` orchestrates the restart-with-WAL-drain shutdown
	// triggered by ActionRestartAgent or by the watchdog on tamper.
	detector   *Detector
	restartMgr *RestartManager
}

// ID returns the agent's stable identifier. Resolved once at startup
// from the data directory's agentid file (or generated and persisted
// if missing). Used by Tamper subsystem (oplog + heartbeat) and by
// the I/O pipeline as the default `host` field on every event.
func (a *Agent) ID() string { return a.cfg.AgentID }

// Collector defines the interface for data collection plugins.
type Collector interface {
	Name() string
	Start(ctx context.Context, ch chan<- Event) error
	Stop()
}

// Event represents a collected data point sent to the server.
//
// Seq is a per-agent monotonically increasing sequence number assigned by
// the WAL on Write. It enables idempotent replay: a server that restarts
// mid-flush can deduplicate by tracking the highest Seq it has seen, and
// the agent only truncates WAL records up to the server's acked_seq.
// See internal/agent/cursor.go for the persistence model.
type Event struct {
	Seq       uint64                 `json:"seq"`
	Timestamp string                 `json:"timestamp"`
	Source    string                 `json:"source"`
	Type      string                 `json:"type"`
	Host      string                 `json:"host"`
	AgentID   string                 `json:"agent_id"`
	Version   string                 `json:"version"`
	Data      map[string]interface{} `json:"data"`
}

const (
	defaultMaxWALEvents = 500_000
	defaultMaxBatch     = 5_000
	eventChannelBuf     = 20_000 // large enough to absorb short bursts
)

// New creates and initialises a new agent.
func New(cfg Config, log *logger.Logger) (*Agent, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Resolve or generate a stable agent ID
	agentID, err := resolveAgentID(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve agent ID: %w", err)
	}
	cfg.AgentID = agentID

	// Resolve or generate sovereign identity key
	privKey, err := resolveIdentityKey(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve identity key: %w", err)
	}

	if cfg.TenantID == "" {
		cfg.TenantID = "GLOBAL"
	}

	// Apply defaults
	if cfg.MaxWALEvents <= 0 {
		cfg.MaxWALEvents = defaultMaxWALEvents
	}
	if cfg.MaxBatchSize <= 0 {
		cfg.MaxBatchSize = defaultMaxBatch
	}

	// WAL for offline buffering
	wal, err := NewWAL(filepath.Join(cfg.DataDir, "wal"), cfg.MaxWALEvents)
	if err != nil {
		return nil, fmt.Errorf("init WAL: %w", err)
	}

	// Transport
	transport, err := NewTransport(cfg, log.WithPrefix("transport"))
	if err != nil {
		wal.Close()
		return nil, fmt.Errorf("init transport: %w", err)
	}
	transport.SetIdentityKey(privKey) // 1.4: Enable batch signing

	a := &Agent{
		cfg:      cfg,
		eventCh:  make(chan Event, eventChannelBuf),
		wal:      wal,
		transport: transport,
		redactor:  NewPIIRedactor(),
		response:  NewResponseActionExecutor(log),
		log:       log.WithPrefix("agent"),
		privKey:   privKey,
		watchdog:  NewWatchdog(log),
	}

	a.hostname, err = os.Hostname()
	if err != nil {
		a.hostname = "unknown-host"
		a.log.Warn("failed to resolve hostname: %v (using 'unknown-host')", err)
	}

	if cfg.EnableMetrics {
		a.collectors = append(a.collectors,
			NewMetricsCollector(a.hostname, agentID, cfg.Interval, log.WithPrefix("metrics")))
	}
	if cfg.EnableSyslog {
		a.collectors = append(a.collectors,
			NewFileTailCollector(a.hostname, agentID, defaultLogPaths(), log.WithPrefix("file_tail")))
	}
	if cfg.EnableFIM {
		a.collectors = append(a.collectors,
			NewFIMCollector(a.hostname, agentID, defaultFIMPaths(), log.WithPrefix("fim")))
	}
	if cfg.EnableEventLog && runtime.GOOS == "windows" {
		a.collectors = append(a.collectors,
			NewEventLogCollector(a.hostname, agentID, log.WithPrefix("eventlog")))
	}
	if runtime.GOOS == "linux" {
		a.collectors = append(a.collectors,
			NewEBPFCollector(a.hostname, log.WithPrefix("ebpf")))
	}

	// Historical backfill (Phase 30.3) — one-shot scan of OS log
	// stores on first launch. Runs to completion, writes a marker
	// file, and is skipped on subsequent boots. We register it
	// unconditionally because it's a no-op on already-backfilled
	// agents and a no-op fallback on unsupported GOOS values.
	a.collectors = append(a.collectors,
		NewBackfillCollector(a.hostname, agentID, cfg.DataDir, log.WithPrefix("backfill")))

	log.Info("Agent ID: %s  a.hostname: %s  collectors: %d", agentID, a.hostname, len(a.collectors))
	return a, nil
}

// Start begins all collectors and the transport loop.
func (a *Agent) Start(ctx context.Context) error {
	a.ctx, a.cancel = context.WithCancel(ctx)
	ctx = a.ctx

	// Register with server before starting data flow
	collectorNames := make([]string, len(a.collectors))
	for i, c := range a.collectors {
		collectorNames[i] = c.Name()
	}
	if err := a.transport.Register(ctx, collectorNames); err != nil {
		a.log.Warn("[agent] Registration failed (will retry): %v", err)
		// Non-fatal — FlushLoop heartbeats carry registration data too
	}

	// Start all registered collectors
	for _, c := range a.collectors {
		a.startCollector(ctx, c)
	}

	// processEvents: WAL writer (bounded by channel depth + WAL size cap)
	a.wg.Add(1)
	go func() {
		defer a.safeRecover("event-processor")
		defer a.wg.Done()
		a.processEvents(ctx)
	}()

	// FlushLoop: WAL → Transport
	a.wg.Add(1)
	go func() {
		defer a.safeRecover("flush-loop")
		defer a.wg.Done()
		a.transport.FlushLoop(ctx, a.wal, a.cfg.MaxBatchSize, a.ApplyConfig)
	}()

	a.log.Info("Agent started — %d collectors active", len(a.collectors))
	return nil
}

// startCollector launches a single collector in a managed goroutine.
// Safe to call after Start().
func (a *Agent) startCollector(ctx context.Context, c Collector) {
	a.wg.Add(1)
	go func() {
		defer a.safeRecover("collector-" + c.Name())
		defer a.wg.Done()
		a.log.Info("Collector %s started", c.Name())
		if err := c.Start(ctx, a.eventCh); err != nil && ctx.Err() == nil {
			a.log.Error("Collector %s exited with error: %v", c.Name(), err)
		}
	}()
}

// ApplyConfig applies configuration received from the server and dispatches
// any pending response actions.
func (a *Agent) ApplyConfig(cfg FleetConfig, actions []PendingAction) {
	// Execute response actions OUTSIDE the config lock — actions can be slow
	// (e.g. process kill + snapshot) and should not block config application.
	for _, action := range actions {
		a.handleAction(action)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	changed := false
	if cfg.Interval > 0 && a.cfg.Interval != cfg.Interval {
		a.cfg.Interval = cfg.Interval
		changed = true
	}
	if a.cfg.EnableFIM != cfg.EnableFIM {
		a.cfg.EnableFIM = cfg.EnableFIM
		changed = true
		a.hotToggle("fim", cfg.EnableFIM)
	}
	if a.cfg.EnableSyslog != cfg.EnableSyslog {
		a.cfg.EnableSyslog = cfg.EnableSyslog
		changed = true
		a.hotToggle("file_tail", cfg.EnableSyslog)
	}
	if a.cfg.EnableMetrics != cfg.EnableMetrics {
		a.cfg.EnableMetrics = cfg.EnableMetrics
		changed = true
		a.hotToggle("metrics", cfg.EnableMetrics)
	}
	if a.cfg.EnableEventLog != cfg.EnableEventLog {
		a.cfg.EnableEventLog = cfg.EnableEventLog
		changed = true
		if runtime.GOOS == "windows" {
			a.hotToggle("eventlog", cfg.EnableEventLog)
		}
	}
	if changed {
		a.log.Info("[config] Applied fleet config: interval=%v collectors=%d", a.cfg.Interval, len(a.collectors))
	}
}

// hotToggle adds or removes a named collector while the agent is running.
// Called with a.mu held.
func (a *Agent) hotToggle(name string, enable bool) {
	if !enable {
		for i, c := range a.collectors {
			if c.Name() == name {
				c.Stop()
				a.collectors = append(a.collectors[:i], a.collectors[i+1:]...)
				a.log.Info("[hot-toggle] Stopped collector: %s", name)
				return
			}
		}
		return
	}

	// Already running?
	for _, c := range a.collectors {
		if c.Name() == name {
			return
		}
	}

	var c Collector
	switch name {
	case "metrics":
		c = NewMetricsCollector(a.hostname, a.cfg.AgentID, a.cfg.Interval, a.log.WithPrefix("metrics"))
	case "file_tail":
		c = NewFileTailCollector(a.hostname, a.cfg.AgentID, defaultLogPaths(), a.log.WithPrefix("file_tail"))
	case "fim":
		c = NewFIMCollector(a.hostname, a.cfg.AgentID, defaultFIMPaths(), a.log.WithPrefix("fim"))
	case "eventlog":
		c = NewEventLogCollector(a.hostname, a.cfg.AgentID, a.log.WithPrefix("eventlog"))
	}

	if c == nil {
		a.log.Warn("[hot-toggle] Unknown collector name: %s", name)
		return
	}

	a.collectors = append(a.collectors, c)
	
	ctx := context.Background()
	if a.ctx != nil {
		ctx = a.ctx
	}
	
	a.startCollector(ctx, c)
	a.log.Info("[hot-toggle] Started collector: %s", name)
}

func (a *Agent) handleAction(action PendingAction) {
	a.log.Info("[c2] Executing action %s type=%s", action.ID, action.Type)
	var err error
	switch action.Type {
	case ActionKillProcess:
		var pid int
		fmt.Sscanf(action.Payload["pid"], "%d", &pid)
		err = a.response.KillProcess(pid)
	case ActionProcessSnapshot:
		var pid int
		fmt.Sscanf(action.Payload["pid"], "%d", &pid)
		_, err = a.response.CollectProcessSnapshot(pid)
	case ActionProcessInventory:
		snaps := a.response.CollectProcessInventory()
		evt := Event{
			Timestamp: time.Now().Format(time.RFC3339),
			Source:    "forensics",
			Type:      "process_inventory",
			Host:      a.hostname,
			AgentID:   a.cfg.AgentID,
			Data:      map[string]interface{}{"processes": snaps},
		}
		a.eventCh <- evt
	case ActionIsolateNetwork:
		a.log.Warn("[containment] Network isolation requested by SOAR")
		err = applyNetworkIsolation(true, a.log)
	case ActionRestoreNetwork:
		a.log.Info("[recovery] Network restore requested by SOAR")
		err = applyNetworkIsolation(false, a.log)

	// ── Phase 30.5 — operator remote control ────────────────────────
	case ActionTriggerScan:
		// On-demand scan = re-emit a fresh metrics + FIM sweep
		// regardless of the configured Interval. Synthesised as a
		// single event so the operator's UI gets immediate feedback.
		a.log.Info("[c2] Triggered on-demand scan")
		evt := Event{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Source:    "agent",
			Type:      "scan.triggered",
			Host:      a.hostname,
			AgentID:   a.cfg.AgentID,
			Data:      map[string]interface{}{"action_id": action.ID, "requested_by": action.Payload["requested_by"]},
		}
		select {
		case a.eventCh <- evt:
		default:
			a.log.Warn("[c2] event channel full, scan-triggered event dropped")
		}
	case ActionToggleDebug:
		// Toggle the local-detection rule pack on/off. We can't
		// flip the global zerolog level at runtime without
		// rebuilding the logger, so this action's scope is bounded
		// to detection-rule enablement — which is the operator-
		// visible behaviour anyway. Emit a debug-mode-changed event
		// so the dashboard reflects the new state.
		desired := strings.ToLower(action.Payload["state"])
		debugOn := desired == "on" || desired == "true" || desired == "1"
		if a.detector != nil {
			a.detector.SetEnabled(!debugOn) // debug-on disables noisy local rules
		}
		a.log.Info("[c2] Debug mode set to %v (local detection %s)", debugOn,
			map[bool]string{true: "disabled", false: "enabled"}[debugOn])
		evt := Event{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Source:    "agent",
			Type:      "debug.toggled",
			Host:      a.hostname,
			AgentID:   a.cfg.AgentID,
			Data:      map[string]interface{}{"debug_on": debugOn, "action_id": action.ID},
		}
		select {
		case a.eventCh <- evt:
		default:
		}
	case ActionRestartAgent:
		// Hand off to the RestartManager, which flushes WAL, closes
		// collectors, and exits with code 75 so the OS service
		// manager (systemd / launchd / SCM) auto-respawns us.
		a.log.Warn("[c2] Operator-initiated agent restart requested")
		if a.restartMgr != nil {
			go a.restartMgr.RequestRestart(RestartReasonUIRequest, 10*time.Second)
		} else {
			a.log.Error("[c2] No RestartManager configured — restart action ignored")
		}

	default:
		a.log.Warn("[c2] Unknown action type: %s", action.Type)
		return
	}
	if err != nil {
		a.log.Error("[c2] Action %s failed: %v", action.ID, err)
	} else {
		a.log.Info("[c2] Action %s completed", action.ID)
	}
}

func (a *Agent) TriggerWatchdogSelfTest() {
	if a.watchdog != nil {
		evt := a.watchdog.TriggerSelfTest()
		if evt != nil {
			a.eventCh <- *evt
		}
	}
}

// Stop gracefully shuts down the agent.
func (a *Agent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	// Stop all collectors so their Start() loops exit cleanly
	a.mu.Lock()
	for _, c := range a.collectors {
		c.Stop()
	}
	a.mu.Unlock()

	a.wg.Wait()
	a.wal.Close()
	a.transport.Close()
	a.log.Info("Agent stopped cleanly")
}

// processEvents reads from the shared collector channel, applies PII redaction,
// and writes to the WAL. Non-blocking drop when WAL is full.
func (a *Agent) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Drain remaining events before exiting
			for {
				select {
				case event := <-a.eventCh:
					a.writeEvent(event)
				default:
					return
				}
			}
		case event := <-a.eventCh:
			a.writeEvent(event)
		}
	}
}

func (a *Agent) writeEvent(event Event) {
	// 5. Sovereign Self-Protection Watchdog
	if a.watchdog != nil {
		if alert := a.watchdog.Inspect(event); alert != nil {
			// Enqueue alert asynchronously to avoid blocking the current event's write
			go func(ea Event) {
				a.eventCh <- ea
			}(*alert)
		}
	}

	// Stamp agent ID if not set by collector
	if event.AgentID == "" {
		event.AgentID = a.cfg.AgentID
	}
	a.redactor.RedactEvent(&event)
	a.redactor.RedactSensitiveFields(&event)
	if err := a.wal.Write(event); err != nil {
		if err == ErrWALFull {
			a.log.Warn("[wal] Buffer full — dropping event type=%s host=%s", event.Type, event.Host)
		} else {
			a.log.Error("[wal] Write error: %v", err)
		}
	}
}

// safeRecover is a panic handler for goroutines — logs and continues.
func (a *Agent) safeRecover(name string) {
	if r := recover(); r != nil {
		a.log.Error("[panic] goroutine %s panicked: %v", name, r)
	}
}

// resolveAgentID reads a persistent agent ID from disk, generating one on first boot.
func resolveAgentID(dataDir string) (string, error) {
	idPath := filepath.Join(dataDir, "agent_id")
	if data, err := os.ReadFile(idPath); err == nil {
		id := string(data)
		if len(id) == 32 { // 16 bytes hex-encoded
			return id, nil
		}
	}
	// Generate a new one
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(raw[:])
	if err := os.WriteFile(idPath, []byte(id), 0600); err != nil {
		return "", err
	}
	return id, nil
}

// defaultLogPaths returns OS-appropriate log file paths to tail.
func defaultLogPaths() []string {
	if runtime.GOOS == "windows" {
		return []string{`C:\Windows\System32\LogFiles\`}
	}
	return []string{
		"/var/log/syslog",
		"/var/log/auth.log",
		"/var/log/secure",
		"/var/log/messages",
	}
}

// defaultFIMPaths returns critical files and directories to monitor.
func defaultFIMPaths() []string {
	if runtime.GOOS == "windows" {
		return []string{
			`C:\Windows\System32\drivers\etc\hosts`,
			`C:\Windows\System32\`,
		}
	}
	return []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/etc/ssh/sshd_config",
		"/etc/hosts",
		"/etc/crontab",
		"/etc/ld.so.preload",
	}
}

// applyNetworkIsolation is a platform-specific stub for host isolation.
// A real implementation would use iptables / Windows Firewall APIs.
func applyNetworkIsolation(isolate bool, log *logger.Logger) error {
	action := "restoring"
	if isolate {
		action = "isolating"
	}
	log.Warn("[isolation] Network %s — full implementation requires platform-specific firewall API", action)
	return nil
}
