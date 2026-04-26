// Package agent — historical log backfill (Phase 30.3)
//
// On first launch, the agent walks pre-existing OS log stores and emits
// each historical record as a normal Event with `Source: "historical"`
// and the line's original timestamp preserved in `Data["original_timestamp"]`.
// This closes the operator-audit gap: "the platform is blind to anything
// that happened before agent install."
//
// One-shot semantics:
//   - The first call to (*BackfillCollector).Start writes all historical
//     events it finds into the channel, then signals "done" by stopping
//     itself. The agent supervisor sees Start return nil and treats this
//     as a successful run.
//   - We persist a marker file `<DataDir>/backfill.complete` so subsequent
//     restarts skip the scan. The file contains the timestamp the scan
//     finished at, which the dashboard can read via the agent metadata
//     endpoint to surface "Since agent install" as a time-range preset
//     (Phase 30.4d resolves this preset by reading the marker).
//
// Platform branches:
//   - Linux:   journalctl since (now - DefaultLookback) + /var/log/* tail
//   - Windows: wevtutil query System / Security / Application
//   - macOS:   /var/log/system.log and unified-log fallback via `log show`
//   - other:   no-op (returns immediately)
//
// Each platform branch lives in `backfill_<os>.go` (this file is the
// orchestrator). The branches MUST be best-effort: any subprocess error
// is logged at WARN and the scan continues — never fail the entire
// agent boot just because one log source can't be read.

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// DefaultLookback is how far back the backfill walks. 30 days is the
// sweet spot: most syslog rotation policies retain 7-30 days, journald
// defaults to "until disk is full," and Windows Event Viewer keeps logs
// effectively forever (we cap explicitly to avoid CPU spikes on busy
// servers with millions of records).
const DefaultLookback = 30 * 24 * time.Hour

// BackfillMarker is the sentinel file that marks "backfill complete."
// Its contents are a JSON object: {"completed_at": "<ISO>", "events": <count>}
const BackfillMarker = "backfill.complete"

// BackfillCollector walks pre-existing OS log stores once on agent
// install. Implements the Collector interface.
type BackfillCollector struct {
	hostname string
	agentID  string
	dataDir  string
	lookback time.Duration
	log      *logger.Logger

	stop stopOnce
	once sync.Once
}

// NewBackfillCollector constructs a backfill collector with default lookback.
func NewBackfillCollector(hostname, agentID, dataDir string, log *logger.Logger) *BackfillCollector {
	return &BackfillCollector{
		hostname: hostname,
		agentID:  agentID,
		dataDir:  dataDir,
		lookback: DefaultLookback,
		log:      log,
		stop:     newStopOnce(),
	}
}

// Name implements Collector.
func (c *BackfillCollector) Name() string { return "backfill" }

// Stop implements Collector.
func (c *BackfillCollector) Stop() { c.stop.stop() }

// Start performs the one-shot historical scan. Returns nil after the
// scan completes (or after an immediate exit if the marker file
// indicates a previous successful run). Errors are logged but not
// returned — backfill failure must NEVER block agent boot.
func (c *BackfillCollector) Start(ctx context.Context, ch chan<- Event) error {
	if c.alreadyComplete() {
		c.log.Info("backfill: already complete, skipping",
			"marker", filepath.Join(c.dataDir, BackfillMarker))
		return nil
	}

	c.log.Info("backfill: starting historical log scan",
		"lookback_hours", int(c.lookback.Hours()),
		"hostname", c.hostname)

	scanStart := time.Now()
	count := 0

	// runPlatformBackfill is implemented in backfill_<os>.go and
	// returns the count of events emitted (best-effort across all
	// log sources for that OS).
	emitted, err := c.runPlatformBackfill(ctx, ch)
	if err != nil {
		// Don't fail — log and proceed. Mark complete anyway so we
		// don't loop on a permanently-broken log source.
		c.log.Warn("backfill: platform scan reported errors",
			"error", err, "events_so_far", emitted)
	}
	count += emitted

	c.markComplete(count, scanStart)
	c.log.Info("backfill: complete",
		"events_emitted", count,
		"duration_seconds", time.Since(scanStart).Seconds())
	return nil
}

// alreadyComplete returns true if the marker file exists. Read errors
// (file unreadable for any reason) fall back to "not complete" so we
// re-run rather than silently skipping.
func (c *BackfillCollector) alreadyComplete() bool {
	path := filepath.Join(c.dataDir, BackfillMarker)
	_, err := os.Stat(path)
	return err == nil
}

// markComplete writes the marker file. Best-effort: a write failure
// only means the scan re-runs on next boot, which is mildly wasteful
// but harmless.
func (c *BackfillCollector) markComplete(events int, started time.Time) {
	path := filepath.Join(c.dataDir, BackfillMarker)
	payload := map[string]interface{}{
		"completed_at": time.Now().UTC().Format(time.RFC3339),
		"started_at":   started.UTC().Format(time.RFC3339),
		"events":       events,
		"agent_id":     c.agentID,
		"hostname":     c.hostname,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		c.log.Warn("backfill: marshal marker failed", "error", err)
		return
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		c.log.Warn("backfill: write marker failed", "path", path, "error", err)
	}
}

// emit is the shared helper used by platform branches to send a
// historical event into the agent's channel. It tags the event with
// `Source: "historical"` and preserves the original log line's
// timestamp under `Data["original_timestamp"]` per the audit spec:
//
//	{
//	  "source": "historical",
//	  "collected_at": "...",
//	  "original_timestamp": "..."
//	}
//
// Backpressure: a slow channel blocks the scan. We use a small select
// with ctx so a cancelled agent shutdown doesn't get stuck.
func (c *BackfillCollector) emit(
	ctx context.Context,
	ch chan<- Event,
	originalTS time.Time,
	logSource string,
	eventType string,
	severity string,
	category string,
	message string,
	extra map[string]interface{},
) error {
	now := time.Now().UTC()

	data := map[string]interface{}{
		"original_timestamp": originalTS.UTC().Format(time.RFC3339Nano),
		"collected_at":       now.Format(time.RFC3339Nano),
		"log_source":         logSource, // /var/log/syslog, journald, eventlog/system, ...
		"severity":           severity,  // unified DEBUG/INFO/WARN/ERROR/CRITICAL
		"category":           category,  // auth, process, network, filesystem, system, security
		"message":            message,
	}
	for k, v := range extra {
		// Don't let extras shadow the standard fields above.
		if _, exists := data[k]; !exists {
			data[k] = v
		}
	}

	ev := Event{
		Timestamp: originalTS.UTC().Format(time.RFC3339Nano),
		Source:    "historical",
		Type:      eventType,
		Host:      c.hostname,
		AgentID:   c.agentID,
		Version:   "v1",
		Data:      data,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stop.C():
		return fmt.Errorf("backfill stopped")
	case ch <- ev:
		return nil
	}
}

// SeverityMapper provides unified severity mapping shared by all
// platform branches. Phase 30.4 / audit-spec gap #3.
//
// Mapping table:
//
//	syslog:    emerg=CRITICAL, alert/crit=CRITICAL, err=ERROR,
//	           warning/notice=WARN, info=INFO, debug=DEBUG
//	journald:  PRIORITY 0-1=CRITICAL, 2-3=ERROR, 4=WARN, 5-6=INFO, 7=DEBUG
//	Windows:   Critical=CRITICAL, Error=ERROR, Warning=WARN,
//	           Information=INFO, Verbose=DEBUG
//
// Unknown values fall back to INFO.
func MapSeverity(source, raw string) string {
	switch source {
	case "syslog":
		switch raw {
		case "emerg", "0":
			return "CRITICAL"
		case "alert", "crit", "1", "2":
			return "CRITICAL"
		case "err", "error", "3":
			return "ERROR"
		case "warning", "warn", "4":
			return "WARN"
		case "notice", "5", "info", "6":
			return "INFO"
		case "debug", "7":
			return "DEBUG"
		}
	case "journald":
		switch raw {
		case "0", "1":
			return "CRITICAL"
		case "2", "3":
			return "ERROR"
		case "4":
			return "WARN"
		case "5", "6":
			return "INFO"
		case "7":
			return "DEBUG"
		}
	case "windows":
		switch raw {
		case "Critical", "1":
			return "CRITICAL"
		case "Error", "2":
			return "ERROR"
		case "Warning", "3":
			return "WARN"
		case "Information", "4":
			return "INFO"
		case "Verbose", "5":
			return "DEBUG"
		}
	}
	return "INFO"
}

// CategorizeBySource provides a default category mapping based on the
// log source (file path or event channel). Best-effort heuristic — the
// dashboard treats it as a hint, not a guarantee.
func CategorizeBySource(source string) string {
	switch {
	case containsAny(source, "auth", "secure", "login", "wtmp", "btmp"):
		return "auth"
	case containsAny(source, "kern", "kernel", "dmesg"):
		return "system"
	case containsAny(source, "mail", "postfix", "dovecot"):
		return "network"
	case containsAny(source, "audit", "audit.log", "Security"):
		return "security"
	case containsAny(source, "fim", "filesystem"):
		return "filesystem"
	}
	return "system"
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && len(s) >= len(sub) {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
