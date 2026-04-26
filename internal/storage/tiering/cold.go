// ColdTier — S3-compatible cold storage implementation.
//
// Phase 31 — the audit's third tier: events older than ~180 days
// (HotDuration + WarmDuration in DefaultRetention) live here. Cold
// storage is optimised for cost, not access speed — multi-second
// query latency is acceptable because cold-tier queries are rare
// (compliance / forensic / DR).
//
// This file ships TWO concrete implementations:
//
//   1. `LocalDirCold` — writes JSON-Lines (one event per line) into
//      a local directory tree partitioned by day. Used by tests and
//      by air-gap deployments without S3 connectivity.
//   2. `RemoteColdTier` — interface stub for an S3-compatible
//      backend (AWS S3, MinIO, Backblaze B2, R2). The stub deliberately
//      DOES NOT pull in the AWS SDK. Real S3 wiring is gated on
//      operator-supplied credentials and lives behind a build tag
//      so air-gap builds stay clean. Concrete S3 adapter ships as
//      a separate file (cold_s3.go) once configuration is plumbed.
//
// Why not import `aws-sdk-go-v2` here:
//   - Air-gap binary commitment — every external dep is weight.
//   - Operators may use MinIO / B2 / R2 / Wasabi instead of AWS,
//     each with a slightly different bucket-policy story.
//   - Pulling cold-tier credentials safely into an agent at scale
//     needs more design (per-tenant scoped keys?) and is its own
//     follow-up.
//
// The Tier interface contract is satisfied by LocalDirCold for now.
// `RemoteColdTier` is a documented placeholder.

package tiering

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ColdEvent is the on-disk JSON-Lines schema. Same shape as
// tiering.Event but with explicit JSON tags so the format is
// language-agnostic — a forensic analyst can `cat *.jsonl | jq`
// to read cold-tier files directly.
type ColdEvent struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Timestamp time.Time `json:"timestamp"`
	Host      string    `json:"host"`
	EventType string    `json:"event_type"`
	Body      []byte    `json:"body"`
}

// LocalDirCold writes events as JSON-Lines under
// `<root>/cold/YYYY/MM/DD/cold-<tenant>.jsonl`. One file per
// (day, tenant) tuple — sized to be friendly to gzip post-write.
type LocalDirCold struct {
	root string
	log  *logger.Logger
}

// NewLocalDirCold constructs the local cold tier. `root` is a
// directory; subdirectories are created on first write. Pass an
// empty `log` for tests; production callers supply the platform
// logger.
func NewLocalDirCold(root string, log *logger.Logger) *LocalDirCold {
	if log == nil {
		log, _ = logger.New(logger.Config{Level: logger.WarnLevel, OutputPath: os.DevNull})
	}
	return &LocalDirCold{
		root: root,
		log:  log.WithPrefix("cold-tier"),
	}
}

// ID implements Tier.
func (c *LocalDirCold) ID() TierID { return TierCold }

// Write implements Tier. Appends to a per-(day, tenant) JSONL file.
// File is opened with O_APPEND so concurrent writes from multiple
// migrators don't clobber each other (atomic per-line on POSIX).
func (c *LocalDirCold) Write(_ context.Context, events []Event) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	// Group by (day, tenant) so we open each output file once.
	type groupKey struct{ day, tenant string }
	grouped := make(map[groupKey][]Event)
	for _, e := range events {
		k := groupKey{
			day:    e.Timestamp.UTC().Format("2006/01/02"),
			tenant: tenantOrGlobal(e.TenantID),
		}
		grouped[k] = append(grouped[k], e)
	}

	written := 0
	for k, batch := range grouped {
		dir := filepath.Join(c.root, "cold", k.day)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return written, fmt.Errorf("cold mkdir: %w", err)
		}
		fname := filepath.Join(dir, fmt.Sprintf("cold-%s.jsonl", k.tenant))
		f, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return written, fmt.Errorf("cold open: %w", err)
		}
		w := bufio.NewWriter(f)
		for _, e := range batch {
			ce := ColdEvent{
				ID: e.ID, TenantID: e.TenantID, Timestamp: e.Timestamp,
				Host: e.Host, EventType: e.EventType, Body: e.Body,
			}
			line, err := json.Marshal(ce)
			if err != nil {
				_ = f.Close()
				return written, fmt.Errorf("cold marshal id=%s: %w", e.ID, err)
			}
			if _, err := w.Write(line); err != nil {
				_ = f.Close()
				return written, fmt.Errorf("cold write: %w", err)
			}
			if err := w.WriteByte('\n'); err != nil {
				_ = f.Close()
				return written, fmt.Errorf("cold write \\n: %w", err)
			}
			written++
		}
		if err := w.Flush(); err != nil {
			_ = f.Close()
			return written, fmt.Errorf("cold flush: %w", err)
		}
		if err := f.Sync(); err != nil {
			_ = f.Close()
			return written, fmt.Errorf("cold fsync: %w", err)
		}
		if err := f.Close(); err != nil {
			return written, fmt.Errorf("cold close: %w", err)
		}
	}
	return written, nil
}

// Range implements Tier. Walks every JSONL file in the cold tree
// whose day falls in [from, to]. Returns early when `fn` returns false.
func (c *LocalDirCold) Range(_ context.Context, from, to time.Time, fn func(e Event) bool) error {
	root := filepath.Join(c.root, "cold")
	if from.IsZero() {
		from = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}
	stop := false
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || stop {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		// Parse YYYY/MM/DD from the path.
		rel, _ := filepath.Rel(root, path)
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 4 {
			return nil
		}
		day, err := time.Parse("2006/01/02",
			parts[0]+"/"+parts[1]+"/"+parts[2])
		if err != nil {
			return nil
		}
		// Cheap pre-filter: skip the whole file if its day is
		// out-of-range. Within-range files still get fully scanned
		// because individual events may straddle midnight.
		if day.Before(from.Truncate(24*time.Hour)) || day.After(to) {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			c.log.Warn("cold range: open %s: %v", path, err)
			return nil
		}
		scanner := bufio.NewScanner(f)
		// Allow up to 16 MB per line — agents can ship big raw logs.
		scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
		for scanner.Scan() {
			if stop {
				break
			}
			var ce ColdEvent
			if err := json.Unmarshal(scanner.Bytes(), &ce); err != nil {
				continue // skip corrupt lines
			}
			if ce.Timestamp.Before(from) || ce.Timestamp.After(to) {
				continue
			}
			if !fn(Event{
				ID: ce.ID, TenantID: ce.TenantID, Timestamp: ce.Timestamp,
				Host: ce.Host, EventType: ce.EventType, Body: ce.Body,
			}) {
				stop = true
			}
		}
		_ = f.Close()
		return nil
	})
	if walkErr != nil && !os.IsNotExist(walkErr) {
		return walkErr
	}
	return nil
}

// Delete implements Tier. Cold-tier deletion is rare (typically only
// on GDPR right-to-erasure or retention-policy expiry); we use the
// same scan-and-rewrite pattern as WarmTier.
func (c *LocalDirCold) Delete(_ context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	root := filepath.Join(c.root, "cold")
	return filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		// Read all lines, drop matches, rewrite.
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

		var kept [][]byte
		matched := false
		for scanner.Scan() {
			line := append([]byte(nil), scanner.Bytes()...) // copy
			var ce ColdEvent
			if err := json.Unmarshal(line, &ce); err != nil {
				kept = append(kept, line) // preserve unparseable
				continue
			}
			if _, hit := want[ce.ID]; hit {
				matched = true
				continue
			}
			kept = append(kept, line)
		}
		_ = f.Close()
		if !matched {
			return nil
		}
		// Atomic rewrite via tmp + rename.
		tmp := path + ".tmp"
		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("cold delete open tmp: %w", err)
		}
		w := bufio.NewWriter(out)
		for _, line := range kept {
			_, _ = w.Write(line)
			_ = w.WriteByte('\n')
		}
		if err := w.Flush(); err != nil {
			_ = out.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("cold delete flush: %w", err)
		}
		if err := out.Sync(); err != nil {
			_ = out.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("cold delete fsync: %w", err)
		}
		if err := out.Close(); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("cold delete close: %w", err)
		}
		if err := os.Rename(tmp, path); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("cold delete rename: %w", err)
		}
		return nil
	})
}

// EstimatedSize implements Tier. Sums every .jsonl file under
// the cold tree.
func (c *LocalDirCold) EstimatedSize(_ context.Context) (int64, error) {
	root := filepath.Join(c.root, "cold")
	var total int64
	err := filepath.Walk(root, func(_ string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".jsonl") {
			total += info.Size()
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return -1, err
	}
	return total, nil
}

// ── Helpers ─────────────────────────────────────────────────────────

func tenantOrGlobal(t string) string {
	if t == "" {
		return "GLOBAL"
	}
	// Sanitise: tenant IDs come from operator config, but defence-
	// in-depth means we strip path separators before they hit the
	// filesystem.
	return strings.NewReplacer("/", "_", "\\", "_", "..", "_").Replace(t)
}

// ── Future: RemoteColdTier ──────────────────────────────────────────
//
// When operator-supplied S3 credentials land, the implementation goes
// in a new file `cold_s3.go` behind a build tag (e.g. `//go:build s3`)
// so air-gap builds compile without the SDK. Outline:
//
//   type RemoteColdTier struct {
//       client *s3.Client
//       bucket string
//       prefix string
//       log    *logger.Logger
//   }
//
//   func (r *RemoteColdTier) Write(ctx context.Context, events []Event) (int, error) {
//       // Group by day; for each group, compose key
//       //   <prefix>/cold/YYYY/MM/DD/cold-<tenant>-<batch>.jsonl.gz
//       // and PutObject with `Content-Encoding: gzip` for over-the-wire compression.
//   }
//
// Same interface; the Tier abstraction is the only surface the
// Migrator and the rest of the platform see.
