package detection

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/security"
)

// ── Signal types ─────────────────────────────────────────────────────────────

// ransomSignal is one confirmed behavioural indicator on a host.
type ransomSignal struct {
	kind      string    // "entropy", "canary", "shadow_delete", "ext_rename", "mass_delete"
	detail    string
	seenAt    time.Time
	weight    int // contribution to threat score
}

// hostState tracks all signals observed for a single host within the rolling window.
type hostState struct {
	mu      sync.Mutex
	signals []ransomSignal
	alerted bool // prevent repeated full-isolation alerts for the same incident
}

func (h *hostState) add(s ransomSignal) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.signals = append(h.signals, s)
}

// score returns the sum of signal weights within the last windowDur.
func (h *hostState) score(windowDur time.Duration) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	cutoff := time.Now().Add(-windowDur)
	total := 0
	for _, s := range h.signals {
		if s.seenAt.After(cutoff) {
			total += s.weight
		}
	}
	return total
}

// prune removes signals older than the window.
func (h *hostState) prune(windowDur time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	cutoff := time.Now().Add(-windowDur)
	fresh := h.signals[:0]
	for _, s := range h.signals {
		if s.seenAt.After(cutoff) {
			fresh = append(fresh, s)
		}
	}
	h.signals = fresh
}

// ── Ransomware-specific file extensions ──────────────────────────────────────

// knownRansomExts is a non-exhaustive list of extensions appended by common ransomware families.
var knownRansomExts = map[string]bool{
	".locked": true, ".crypto": true, ".enc": true, ".encrypted": true,
	".crypt": true, ".lck": true, ".pays": true, ".wncry": true,
	".wnry": true, ".wcry": true, ".crypz": true, ".cryp1": true,
	".micro": true, ".zepto": true, ".locky": true, ".cerber": true,
	".cerber2": true, ".cerber3": true, ".vvv": true, ".ecc": true,
	".ezz": true, ".exx": true, ".xyz": true, ".zzz": true,
	".aaa": true, ".abc": true, ".bad": true, ".fuck": true,
	".r5a": true, ".ransom": true, ".raid10": true, ".exploit": true,
}

// shadowCopyIndicators are commands/events that indicate VSS deletion.
var shadowCopyIndicators = []string{
	"vssadmin delete shadows",
	"wmic shadowcopy delete",
	"bcdedit /set recoveryenabled no",
	"wbadmin delete catalog",
	"diskshadow /s",
}

// ── Thresholds ────────────────────────────────────────────────────────────────

const (
	// A threat score at or above this level triggers a CRITICAL alert + auto-isolation
	isolationThreshold = 60

	// A threat score at or above this level triggers a HIGH alert only
	alertThreshold = 30

	// Rolling window for signal accumulation
	detectionWindow = 2 * time.Minute

	// Minimum interval between full-isolation actions per host
	isolationCooldown = 5 * time.Minute
)

// ── Signal weights ────────────────────────────────────────────────────────────

const (
	weightEntropyWrite   = 5  // single high-entropy write — common, low individual weight
	weightCanaryHit      = 50 // canary touched — near-certain indicator
	weightShadowDelete   = 40 // VSS deletion — hallmark of ransomware
	weightExtRename      = 15 // known ransomware extension added
	weightMassDelete     = 20 // large number of deletes in window
	weightMassModify     = 10 // large number of modifications in window
)

// ── RansomwareEngine ─────────────────────────────────────────────────────────

// RansomwareEngine correlates multiple behavioural signals to detect ransomware
// with high confidence and trigger automatic network isolation.
type RansomwareEngine struct {
	bus       *eventbus.Bus
	incidents database.IncidentStore
	log       *logger.Logger

	mu         sync.Mutex
	hostStates map[string]*hostState // hostID -> accumulated signals
	lastAlert  map[string]time.Time  // hostID -> last isolation timestamp (cooldown)
}

func NewRansomwareEngine(bus *eventbus.Bus, incidents database.IncidentStore, log *logger.Logger) *RansomwareEngine {
	return &RansomwareEngine{
		bus:        bus,
		incidents:  incidents,
		log:        log.WithPrefix("ransomware"),
		hostStates: make(map[string]*hostState),
		lastAlert:  make(map[string]time.Time),
	}
}

func (e *RansomwareEngine) Name() string { return "RansomwareEngine" }

func (e *RansomwareEngine) Startup(ctx context.Context) {
	e.log.Info("Ransomware multi-signal detection engine starting...")

	// FIM events — the primary signal source
	e.bus.Subscribe(eventbus.EventFIMModified, e.handleFIMModified)
	e.bus.Subscribe(eventbus.EventFIMRenamed, e.handleFIMRenamed)
	e.bus.Subscribe(eventbus.EventFIMDeleted, e.handleFIMDeleted)
	e.bus.Subscribe(eventbus.EventFIMCreated, e.handleFIMCreated)

	// Canary hit (published by CanaryService when a canary path is accessed)
	e.bus.Subscribe("ransomware.canary_hit", e.handleCanaryHit)

	// Shadow copy deletion (published by agent or SIEM ingestion on matching events)
	e.bus.Subscribe("ransomware.shadow_copy_deleted", e.handleShadowCopyDeleted)

	// Raw command execution events from agents
	e.bus.Subscribe("agent.command_executed", e.handleCommandExecution)

	go e.gcLoop(ctx)
}

func (e *RansomwareEngine) Shutdown() {
	e.log.Info("Ransomware detection engine shutting down...")
}

// ── Signal handlers ───────────────────────────────────────────────────────────

func (e *RansomwareEngine) handleFIMModified(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}
	hostID, _ := data["host_id"].(string)
	filePath, _ := data["path"].(string)
	content, _ := data["content"].([]byte)

	if hostID == "" {
		return
	}

	// Signal 1: High-entropy content write (potential encryption in progress)
	if len(content) > 512 && security.IsHighEntropy(content) {
		e.addSignal(hostID, ransomSignal{
			kind:   "entropy",
			detail: fmt.Sprintf("high-entropy write: %s (entropy>7.5)", filePath),
			seenAt: time.Now(),
			weight: weightEntropyWrite,
		})
	}

	// Signal 2: Known ransomware extension added
	ext := strings.ToLower(filepath.Ext(filePath))
	if knownRansomExts[ext] {
		e.addSignal(hostID, ransomSignal{
			kind:   "ext_rename",
			detail: fmt.Sprintf("known ransomware extension written: %s", filePath),
			seenAt: time.Now(),
			weight: weightExtRename,
		})
	}

	e.evaluate(hostID)
}

func (e *RansomwareEngine) handleFIMRenamed(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}
	hostID, _ := data["host_id"].(string)
	newPath, _ := data["new_path"].(string)
	if newPath == "" {
		newPath, _ = data["path"].(string)
	}

	if hostID == "" {
		return
	}

	// Signal: file renamed to a known ransomware extension
	ext := strings.ToLower(filepath.Ext(newPath))
	if knownRansomExts[ext] {
		e.addSignal(hostID, ransomSignal{
			kind:   "ext_rename",
			detail: fmt.Sprintf("file renamed to ransomware extension: %s", newPath),
			seenAt: time.Now(),
			weight: weightExtRename,
		})
		e.evaluate(hostID)
	}
}

func (e *RansomwareEngine) handleFIMDeleted(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}
	hostID, _ := data["host_id"].(string)
	filePath, _ := data["path"].(string)
	if hostID == "" {
		return
	}

	// Signal: canary file deleted
	if isCanaryPath(filePath) {
		e.addSignal(hostID, ransomSignal{
			kind:   "canary",
			detail: fmt.Sprintf("canary file deleted: %s", filePath),
			seenAt: time.Now(),
			weight: weightCanaryHit,
		})
		e.log.Warn("[RANSOMWARE] CANARY DELETED on %s: %s — HIGH CONFIDENCE INDICATOR", hostID, filePath)
		e.evaluate(hostID)
		return
	}

	// Signal: mass deletion accumulation
	e.addSignal(hostID, ransomSignal{
		kind:   "mass_delete",
		detail: fmt.Sprintf("file deleted: %s", filePath),
		seenAt: time.Now(),
		weight: 1, // individual deletes are low weight; mass accumulation matters
	})

	// Check mass deletion threshold (50+ deletes in window = mass_delete signal)
	state := e.getOrCreateState(hostID)
	state.mu.Lock()
	deleteCount := 0
	cutoff := time.Now().Add(-detectionWindow)
	for _, s := range state.signals {
		if s.kind == "mass_delete" && s.seenAt.After(cutoff) {
			deleteCount++
		}
	}
	state.mu.Unlock()

	if deleteCount == 50 { // trigger once at exactly 50 to avoid repeated signals
		e.addSignal(hostID, ransomSignal{
			kind:   "mass_delete",
			detail: fmt.Sprintf("mass file deletion: %d files in %s", deleteCount, detectionWindow),
			seenAt: time.Now(),
			weight: weightMassDelete,
		})
		e.evaluate(hostID)
	}
}

func (e *RansomwareEngine) handleFIMCreated(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}
	hostID, _ := data["host_id"].(string)
	filePath, _ := data["path"].(string)
	if hostID == "" {
		return
	}

	// Ransom note detection: ransomware always drops a README/DECRYPT/HOW_TO file
	base := strings.ToUpper(filepath.Base(filePath))
	ransomNoteKeywords := []string{"README", "DECRYPT", "HOW_TO", "RESTORE", "RECOVER", "RANSOM", "HELP_DECRYPT", "!HELP"}
	for _, kw := range ransomNoteKeywords {
		if strings.Contains(base, kw) {
			e.addSignal(hostID, ransomSignal{
				kind:   "ransom_note",
				detail: fmt.Sprintf("ransom note created: %s", filePath),
				seenAt: time.Now(),
				weight: 45, // near-certain indicator
			})
			e.log.Warn("[RANSOMWARE] RANSOM NOTE DETECTED on %s: %s", hostID, filePath)
			e.evaluate(hostID)
			return
		}
	}
}

func (e *RansomwareEngine) handleCanaryHit(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}
	hostID, _ := data["host_id"].(string)
	filePath, _ := data["path"].(string)
	action, _ := data["action"].(string)
	if hostID == "" {
		return
	}

	e.addSignal(hostID, ransomSignal{
		kind:   "canary",
		detail: fmt.Sprintf("canary %s: %s", action, filePath),
		seenAt: time.Now(),
		weight: weightCanaryHit,
	})
	e.log.Warn("[RANSOMWARE] CANARY HIT on %s (%s): %s — TRIGGERING IMMEDIATE EVALUATION", hostID, action, filePath)
	e.evaluate(hostID)
}

func (e *RansomwareEngine) handleShadowCopyDeleted(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}
	hostID, _ := data["host_id"].(string)
	detail, _ := data["detail"].(string)
	if hostID == "" {
		return
	}

	e.addSignal(hostID, ransomSignal{
		kind:   "shadow_delete",
		detail: fmt.Sprintf("VSS shadow copy deleted: %s", detail),
		seenAt: time.Now(),
		weight: weightShadowDelete,
	})
	e.log.Warn("[RANSOMWARE] SHADOW COPY DELETION on %s — CRITICAL RANSOMWARE INDICATOR", hostID)
	e.evaluate(hostID)
}

func (e *RansomwareEngine) handleCommandExecution(event eventbus.Event) {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}
	hostID, _ := data["host_id"].(string)
	command, _ := data["command"].(string)
	if hostID == "" || command == "" {
		return
	}

	cmdLower := strings.ToLower(command)
	for _, indicator := range shadowCopyIndicators {
		if strings.Contains(cmdLower, indicator) {
			// Emit the shadow_copy_deleted event so our handler fires
			e.bus.Publish("ransomware.shadow_copy_deleted", map[string]interface{}{
				"host_id": hostID,
				"detail":  command,
			})
			return
		}
	}
}

// ── Evaluation & response ─────────────────────────────────────────────────────

// evaluate scores the current threat level for a host and fires the appropriate response.
func (e *RansomwareEngine) evaluate(hostID string) {
	state := e.getOrCreateState(hostID)
	score := state.score(detectionWindow)

	e.log.Debug("[RANSOMWARE] Host %s threat score: %d", hostID, score)

	if score >= isolationThreshold {
		e.triggerIsolation(hostID, state, score)
	} else if score >= alertThreshold {
		e.triggerAlert(hostID, score, "MEDIUM", "Suspicious ransomware-like behaviour detected — monitoring escalated.")
	}
}

// triggerIsolation fires a CRITICAL alert and publishes an isolation request.
func (e *RansomwareEngine) triggerIsolation(hostID string, state *hostState, score int) {
	e.mu.Lock()
	lastTime, alerted := e.lastAlert[hostID]
	if alerted && time.Since(lastTime) < isolationCooldown {
		e.mu.Unlock()
		return // still within cooldown — don't re-isolate
	}
	e.lastAlert[hostID] = time.Now()
	state.alerted = true
	e.mu.Unlock()

	// Collect signal summary for evidence
	state.mu.Lock()
	var signalSummary []string
	cutoff := time.Now().Add(-detectionWindow)
	signalCounts := make(map[string]int)
	for _, s := range state.signals {
		if s.seenAt.After(cutoff) {
			signalCounts[s.kind]++
		}
	}
	for kind, count := range signalCounts {
		signalSummary = append(signalSummary, fmt.Sprintf("%s×%d", kind, count))
	}
	state.mu.Unlock()

	reason := fmt.Sprintf(
		"Ransomware behavioral chain confirmed. Threat score: %d/%d. Signals: [%s]",
		score, isolationThreshold, strings.Join(signalSummary, ", "),
	)

	e.log.Error("[RANSOMWARE] 🔴 ISOLATION TRIGGERED for host %s — %s", hostID, reason)

	// 1. Persist a CRITICAL incident
	e.triggerAlert(hostID, score, "Critical", reason)

	// 2. Publish isolation request — NetworkIsolator subscribes to this
	e.bus.Publish("ransomware.isolation_requested", map[string]interface{}{
		"host_id":     hostID,
		"reason":      reason,
		"threat_score": score,
		"auto":        true,
	})

	// 3. Publish to SIEM alert stream
	e.bus.Publish("siem.alert_fired", map[string]interface{}{
		"type":        "RANSOMWARE_AUTO_ISOLATION",
		"severity":    "CRITICAL",
		"host_id":     hostID,
		"description": reason,
		"technique":   "T1486", // Data Encrypted for Impact
		"auto_action": "network_isolation",
	})
}

// triggerAlert persists an incident record without triggering isolation.
func (e *RansomwareEngine) triggerAlert(hostID string, score int, severity, reason string) {
	incident := &database.Incident{
		ID:              fmt.Sprintf("RANS-%s-%d", hostID, time.Now().Unix()),
		RuleID:          "Ransomware.Behavioral.MultiSignal",
		GroupKey:        hostID,
		Status:          "Active",
		Severity:        severity,
		Title:           "Ransomware Activity Detected",
		Description:     reason,
		FirstSeenAt:     time.Now(),
		LastSeenAt:      time.Now(),
		EventCount:      score,
		MitreTactics:    []string{"Impact", "Defense Evasion"},
		MitreTechniques: []string{"T1486", "T1490"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.incidents.Upsert(ctx, incident); err != nil {
		e.log.Error("Failed to persist ransomware incident: %v", err)
	}

	e.bus.Publish(eventbus.EventSIEMAlert, incident)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (e *RansomwareEngine) addSignal(hostID string, s ransomSignal) {
	state := e.getOrCreateState(hostID)
	state.add(s)
}

func (e *RansomwareEngine) getOrCreateState(hostID string) *hostState {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.hostStates[hostID] == nil {
		e.hostStates[hostID] = &hostState{}
	}
	return e.hostStates[hostID]
}

// gcLoop periodically prunes old signals to prevent unbounded memory growth.
func (e *RansomwareEngine) gcLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			e.mu.Lock()
			for _, state := range e.hostStates {
				state.prune(detectionWindow * 3)
			}
			e.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// isCanaryPath returns true if the path matches known canary file patterns.
func isCanaryPath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	canaryNames := []string{".oblivra_canary", "secrets.txt", "passwords.db", "backups.zip", ".config"}
	for _, name := range canaryNames {
		if base == name {
			return true
		}
	}
	return strings.Contains(path, "oblivra_canary")
}
