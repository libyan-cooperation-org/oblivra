//go:build linux

// Linux historical backfill — drives the BackfillCollector on Linux.
//
// Strategy:
//   1. Run `journalctl --since "<lookback>" --output=json --no-pager`.
//      Each JSON line is one journald entry. We map PRIORITY → severity,
//      preserve __REALTIME_TIMESTAMP as the original timestamp, and emit.
//      This covers /var/log/journal/* on systemd-managed hosts.
//   2. Walk a default file list: /var/log/syslog, /var/log/auth.log,
//      /var/log/secure, /var/log/messages, /var/log/kern.log. Each line
//      is parsed as syslog (RFC3164 best-effort) and emitted.
//
// Both steps are best-effort — a missing journalctl binary or
// unreadable file is logged at WARN and skipped, never fatal. The
// goal is "best-effort historical reconstruction" per the audit spec.

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// runPlatformBackfill walks Linux log sources and emits historical events.
func (c *BackfillCollector) runPlatformBackfill(ctx context.Context, ch chan<- Event) (int, error) {
	count := 0
	var firstErr error

	// 1. journalctl scan — covers most modern Linux distros.
	if n, err := c.scanJournalctl(ctx, ch); err != nil {
		c.log.Warn("backfill: journalctl scan failed", "error", err)
		if firstErr == nil {
			firstErr = err
		}
	} else {
		count += n
	}

	// 2. Plain text logs under /var/log/* (covers non-systemd distros
	// and apps that don't write to journald).
	textPaths := []string{
		"/var/log/syslog",
		"/var/log/auth.log",
		"/var/log/secure",
		"/var/log/messages",
		"/var/log/kern.log",
		"/var/log/dpkg.log",
		"/var/log/apt/history.log",
	}
	for _, p := range textPaths {
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		case <-c.stop.C():
			return count, nil
		default:
		}
		n, err := c.scanTextLog(ctx, ch, p)
		if err != nil {
			// Don't log every missing file — most systems have at most
			// 2-3 of these. INFO is too loud, just continue silently.
			continue
		}
		count += n
	}

	return count, firstErr
}

// scanJournalctl shells out to `journalctl --since=<lookback>
// --output=json --no-pager` and emits one event per journald entry.
//
// We pipe stdout through a Scanner so we don't buffer the entire
// output in memory — on a server with millions of entries this would
// OOM. Each line is one complete JSON object per journald's NDJSON
// output mode.
func (c *BackfillCollector) scanJournalctl(ctx context.Context, ch chan<- Event) (int, error) {
	if _, err := exec.LookPath("journalctl"); err != nil {
		return 0, fmt.Errorf("journalctl not in PATH: %w", err)
	}

	since := time.Now().Add(-c.lookback).Format("2006-01-02 15:04:05")
	cmd := exec.CommandContext(ctx, "journalctl",
		"--since", since,
		"--output=json",
		"--no-pager",
		"--quiet",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start journalctl: %w", err)
	}

	count := 0
	scanner := bufio.NewScanner(stdout)
	// Default token size is 64KB; bump to 1MB for huge journal entries.
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
			continue // skip malformed lines
		}

		// __REALTIME_TIMESTAMP is microseconds-since-epoch as a string.
		ts := time.Now()
		if rt, ok := entry["__REALTIME_TIMESTAMP"].(string); ok {
			if usec, err := strconv.ParseInt(rt, 10, 64); err == nil {
				ts = time.Unix(usec/1_000_000, (usec%1_000_000)*1000)
			}
		}

		priority, _ := entry["PRIORITY"].(string)
		message, _ := entry["MESSAGE"].(string)
		unit, _ := entry["_SYSTEMD_UNIT"].(string)
		comm, _ := entry["_COMM"].(string)
		if comm == "" {
			comm, _ = entry["SYSLOG_IDENTIFIER"].(string)
		}

		severity := MapSeverity("journald", priority)
		category := CategorizeBySource(unit)
		if category == "system" && comm != "" {
			category = CategorizeBySource(comm)
		}

		extra := map[string]interface{}{
			"systemd_unit": unit,
			"comm":         comm,
			"priority":     priority,
		}

		if err := c.emit(ctx, ch, ts, "journald",
			"log.journald", severity, category, message, extra); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return count, err
		}
		count++
	}

	if err := scanner.Err(); err != nil {
		// non-fatal
		c.log.Debug("backfill: journalctl scanner err", "error", err)
	}
	if err := cmd.Wait(); err != nil {
		// journalctl may exit non-zero with no rows; tolerable.
		c.log.Debug("backfill: journalctl exit", "error", err)
	}
	return count, nil
}

// scanTextLog reads a /var/log file line-by-line and emits each as a
// historical event. Entries older than DefaultLookback are skipped.
func (c *BackfillCollector) scanTextLog(ctx context.Context, ch chan<- Event, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	cutoff := time.Now().Add(-c.lookback)
	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	logSource := filepath.Base(path)
	category := CategorizeBySource(logSource)

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

		// Best-effort RFC3164 parse: "Jan  1 00:00:00 host program: msg"
		ts, severity, msg := parseSyslogLine(line)
		if ts.Before(cutoff) {
			continue
		}

		extra := map[string]interface{}{
			"path":     path,
			"raw_line": line,
		}
		if err := c.emit(ctx, ch, ts, logSource,
			"log.text", severity, category, msg, extra); err != nil {
			return count, err
		}
		count++
	}
	return count, scanner.Err()
}

// parseSyslogLine pulls a timestamp + best-guess severity out of a
// classic syslog line. Does NOT do full RFC3164 — that's `internal/ingest/parsers`'s
// job. The agent only needs enough to filter by lookback and surface
// a timestamp; the server's parser does the proper dissection.
func parseSyslogLine(line string) (time.Time, string, string) {
	// Default: now (so the line at least gets ingested) and INFO.
	ts := time.Now()
	severity := "INFO"
	msg := line

	// "Jan  1 00:00:00 host program: msg" — first 15 chars = timestamp
	if len(line) >= 15 {
		parsed, err := time.Parse("Jan _2 15:04:05", line[:15])
		if err == nil {
			// year is missing in syslog format — assume current year.
			parsed = parsed.AddDate(time.Now().Year()-parsed.Year(), 0, 0)
			ts = parsed
			msg = strings.TrimSpace(line[15:])
		}
	}

	// Quick keyword-based severity guess. The server's normalization
	// pipeline does proper severity mapping; this is just for the
	// historical backfill view.
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
