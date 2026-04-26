//go:build darwin

// macOS historical backfill — drives the BackfillCollector on macOS.
//
// Strategy:
//   1. Run `log show --predicate 'eventType == logEvent' --last <duration>
//      --style ndjson` to dump unified-log entries as one JSON object
//      per line. This is Apple's modern log API; it covers everything
//      from kernel through application-level os_log() calls.
//   2. Walk classic /var/log/system.log if present (Apple still writes
//      sudo / kernel panic / wifi roaming events here on some
//      configurations).
//
// Both steps are best-effort.

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// runPlatformBackfill walks macOS log sources.
func (c *BackfillCollector) runPlatformBackfill(ctx context.Context, ch chan<- Event) (int, error) {
	count := 0
	var firstErr error

	if n, err := c.scanUnifiedLog(ctx, ch); err != nil {
		c.log.Warn("backfill: macOS unified log scan failed: %v", err)
		if firstErr == nil {
			firstErr = err
		}
	} else {
		count += n
	}

	for _, p := range []string{"/var/log/system.log", "/var/log/wifi.log"} {
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		case <-c.stop.C():
			return count, nil
		default:
		}
		if n, err := c.scanMacTextLog(ctx, ch, p); err == nil {
			count += n
		}
	}

	return count, firstErr
}

// scanUnifiedLog runs `log show --style ndjson` and emits each entry.
//
// The CLI requires the duration in `<n>[hms]` form. We use the highest
// granularity that fits, which is hours for our 30-day default.
func (c *BackfillCollector) scanUnifiedLog(ctx context.Context, ch chan<- Event) (int, error) {
	if _, err := exec.LookPath("log"); err != nil {
		return 0, fmt.Errorf("log binary not in PATH: %w", err)
	}

	hours := int(c.lookback.Hours())
	if hours < 1 {
		hours = 1
	}

	cmd := exec.CommandContext(ctx, "log", "show",
		"--last", fmt.Sprintf("%dh", hours),
		"--style", "ndjson",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start log: %w", err)
	}

	count := 0
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return count, ctx.Err()
		case <-c.stop.C():
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return count, nil
		default:
		}

		var entry map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		ts := time.Now()
		if tstr, ok := entry["timestamp"].(string); ok {
			if parsed, err := time.Parse("2006-01-02 15:04:05.000000-0700", tstr); err == nil {
				ts = parsed
			}
		}

		// Unified log entries carry a `messageType` ("Default", "Info",
		// "Debug", "Error", "Fault"). Map to our unified severity.
		mtype, _ := entry["messageType"].(string)
		severity := "INFO"
		switch mtype {
		case "Fault":
			severity = "CRITICAL"
		case "Error":
			severity = "ERROR"
		case "Debug":
			severity = "DEBUG"
		}

		subsystem, _ := entry["subsystem"].(string)
		category, _ := entry["category"].(string)
		if category == "" {
			category = CategorizeBySource(subsystem)
		}

		message, _ := entry["eventMessage"].(string)
		process, _ := entry["processImagePath"].(string)

		extra := map[string]interface{}{
			"subsystem": subsystem,
			"process":   process,
			"raw":       string(scanner.Bytes()),
		}
		if err := c.emit(ctx, ch, ts, "macos.unified",
			"log.unified", severity, category, message, extra); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return count, err
		}
		count++
	}

	if err := cmd.Wait(); err != nil {
		c.log.Debug("backfill: log show exit: %v", err)
	}
	return count, nil
}

// scanMacTextLog reads /var/log/system.log style files (BSD syslog
// format). Same approach as the Linux text-log scanner — best-effort
// timestamp parsing + lookback filter.
func (c *BackfillCollector) scanMacTextLog(ctx context.Context, ch chan<- Event, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	cutoff := time.Now().Add(-c.lookback)
	count := 0
	logSource := filepath.Base(path)
	category := CategorizeBySource(logSource)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		case <-c.stop.C():
			return count, nil
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}
		ts, severity, msg := parseSyslogLine(line)
		if ts.Before(cutoff) {
			continue
		}
		extra := map[string]interface{}{
			"path":     path,
			"raw_line": line,
		}
		if err := c.emit(ctx, ch, ts, logSource, "log.text", severity, category, msg, extra); err != nil {
			return count, err
		}
		count++
	}
	return count, scanner.Err()
}

// parseSyslogLine — same as Linux's helper. We keep one copy here
// (build-tagged) so the Linux file's helper isn't visible on macOS
// builds and vice versa. The implementation is identical.
func parseSyslogLine(line string) (time.Time, string, string) {
	ts := time.Now()
	severity := "INFO"
	msg := line

	if len(line) >= 15 {
		parsed, err := time.Parse("Jan _2 15:04:05", line[:15])
		if err == nil {
			parsed = parsed.AddDate(time.Now().Year()-parsed.Year(), 0, 0)
			ts = parsed
			msg = strings.TrimSpace(line[15:])
		}
	}

	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "panic"), strings.Contains(lower, "fatal"):
		severity = "CRITICAL"
	case strings.Contains(lower, "error"), strings.Contains(lower, "fail"):
		severity = "ERROR"
	case strings.Contains(lower, "warn"):
		severity = "WARN"
	case strings.Contains(lower, "debug"):
		severity = "DEBUG"
	}
	return ts, severity, msg
}
