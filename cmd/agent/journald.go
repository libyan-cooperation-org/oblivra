package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// runJournald tails systemd-journal via the `journalctl --follow
// --output=json` subprocess. Each line is one JSON-encoded journal
// record — we extract the human-readable MESSAGE plus a curated set
// of metadata fields and push the result through the standard
// enqueue() path.
//
// We deliberately shell out instead of linking libsystemd: this keeps
// the agent a single static binary and avoids CGO. Cost is one extra
// subprocess and a fork-on-restart; on a busy server that's negligible
// next to the IO journalctl is already doing.
//
// Config knobs (yaml):
//   - type: journald
//   - units: ["sshd.service", "nginx.service"]   # --unit (zero+ entries)
//   - matches: ["_SYSTEMD_UNIT=sshd.service"]    # raw journalctl match
//   - priority: "warning"                         # --priority (default: omit)
//   - sinceBoot: true                             # --boot (this boot only)
//   - cursorFile: "/var/lib/oblivra-agent/journald.cursor" # checkpointed
//
// `path` is reused as cursorFile when cursorFile isn't set explicitly,
// so the simplest config (`type: journald`) just works.
func (t *Tailer) runJournald(ctx context.Context) error {
	if runtime.GOOS != "linux" {
		return errors.New("journald: requires Linux")
	}
	if _, err := exec.LookPath("journalctl"); err != nil {
		return errors.New("journald: journalctl not found in PATH")
	}

	args := []string{"--follow", "--output=json", "--no-pager", "--all"}

	cursorFile := t.in.Path
	if cursorFile == "" {
		dir := t.stateDir
		if dir == "" {
			dir = "."
		}
		cursorFile = filepath.Join(dir, "journald.cursor")
	}

	// Resume from the checkpoint if one exists; otherwise start from now
	// (don't replay days of backlog by default).
	cursor := readCursor(cursorFile)
	if cursor != "" {
		args = append(args, "--after-cursor", cursor)
	} else if t.in.StartFrom == "beginning" {
		args = append(args, "--no-tail")
	} else {
		// Default: only events written from now on.
		args = append(args, "--since", "now")
	}

	for _, u := range t.in.JournaldUnits {
		args = append(args, "--unit", u)
	}
	for _, m := range t.in.JournaldMatches {
		args = append(args, m)
	}
	if t.in.JournaldPriority != "" {
		args = append(args, "--priority", t.in.JournaldPriority)
	}
	if t.in.JournaldSinceBoot {
		args = append(args, "--boot")
	}

	for ctx.Err() == nil {
		cmd := exec.CommandContext(ctx, "journalctl", args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("journald stdout pipe: %w", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("journald stderr pipe: %w", err)
		}
		if err := cmd.Start(); err != nil {
			log.Printf("journald: start: %v; retrying in 2s", err)
			sleep(ctx, 2*time.Second)
			continue
		}

		go drainStderr(stderr)
		t.consumeJournald(ctx, stdout, cursorFile)

		_ = cmd.Wait()
		if ctx.Err() != nil {
			return nil
		}
		log.Printf("journald: subprocess exited; restarting in 1s")
		sleep(ctx, 1*time.Second)

		// On restart, switch to "after-cursor" mode so we resume cleanly.
		if cursor := readCursor(cursorFile); cursor != "" {
			args = stripFlag(args, "--since")
			args = stripFlag(args, "--no-tail")
			args = append(args, "--after-cursor", cursor)
		}
	}
	return nil
}

func (t *Tailer) consumeJournald(ctx context.Context, r io.Reader, cursorFile string) {
	scan := bufio.NewScanner(r)
	scan.Buffer(make([]byte, 1<<20), 16<<20)
	checkpointEvery := 100
	since := 0

	for scan.Scan() {
		if ctx.Err() != nil {
			return
		}
		line := scan.Bytes()
		if len(line) == 0 {
			continue
		}
		ev, cursor, ok := parseJournaldRecord(line)
		if !ok {
			continue
		}
		t.enqueue("journald", ev)
		since++
		if since >= checkpointEvery && cursor != "" {
			writeCursor(cursorFile, cursor)
			since = 0
		}
	}
	if err := scan.Err(); err != nil {
		log.Printf("journald: scan: %v", err)
	}
}

func drainStderr(r io.Reader) {
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		log.Printf("journalctl: %s", scan.Text())
	}
}

// parseJournaldRecord pulls the human-readable message out of a journal
// JSON line and returns it as the event raw text. The cursor is
// returned separately so the caller can checkpoint progress.
//
// A journal record carries dozens of fields; we render the most useful
// ones inline ("Mar 02 12:34:56 host unit[pid]: message") so the
// platform's existing syslog parser handles the line natively. Extra
// metadata is preserved in fields the enqueue() path stamps on top.
func parseJournaldRecord(line []byte) (string, string, bool) {
	var rec map[string]any
	if err := json.Unmarshal(line, &rec); err != nil {
		return "", "", false
	}

	msg := stringOf(rec["MESSAGE"])
	if msg == "" {
		return "", "", false
	}

	host := stringOf(rec["_HOSTNAME"])
	unit := stringOf(rec["_SYSTEMD_UNIT"])
	if unit == "" {
		unit = stringOf(rec["SYSLOG_IDENTIFIER"])
	}
	pid := stringOf(rec["_PID"])
	cursor := stringOf(rec["__CURSOR"])

	// Synthesise a syslog-RFC3164-shape line so server-side parsers can
	// pick up host/process/PID without us hardcoding the schema.
	ts := journaldTimestamp(rec)
	var b strings.Builder
	if !ts.IsZero() {
		b.WriteString(ts.Format("Jan _2 15:04:05"))
		b.WriteByte(' ')
	}
	if host != "" {
		b.WriteString(host)
		b.WriteByte(' ')
	}
	if unit != "" {
		b.WriteString(unit)
		if pid != "" {
			b.WriteByte('[')
			b.WriteString(pid)
			b.WriteByte(']')
		}
		b.WriteString(": ")
	}
	b.WriteString(msg)
	return b.String(), cursor, true
}

// journaldTimestamp parses the __REALTIME_TIMESTAMP microseconds field.
func journaldTimestamp(rec map[string]any) time.Time {
	v := stringOf(rec["__REALTIME_TIMESTAMP"])
	if v == "" {
		return time.Time{}
	}
	usec, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(usec/1_000_000, (usec%1_000_000)*1000).UTC()
}

func stringOf(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

// readCursor returns the last-checkpointed cursor or "" if none.
// Empty / missing file is normal on first run.
func readCursor(path string) string {
	body, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// writeCursor atomically replaces the cursor file. Uses temp + rename
// so a crash mid-write doesn't truncate to zero bytes (which would
// trigger an unwanted rewind on restart).
func writeCursor(path, cursor string) {
	if cursor == "" {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(cursor+"\n"), 0o600); err != nil {
		log.Printf("journald cursor write: %v", err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		log.Printf("journald cursor rename: %v", err)
		_ = os.Remove(tmp)
	}
}

func stripFlag(args []string, flag string) []string {
	out := args[:0]
	skip := false
	for _, a := range args {
		if skip {
			skip = false
			continue
		}
		if a == flag {
			skip = true
			continue
		}
		out = append(out, a)
	}
	return out
}
