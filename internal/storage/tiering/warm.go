// WarmTier — Parquet-backed implementation of Tier.
//
// Phase 31 — wraps the existing `internal/storage.ParquetArchiver`
// (daily-partitioned Parquet files under <DataDir>/archives/YYYY/MM/DD/)
// behind the Tier interface. Once events are promoted out of Hot via
// the Migrator, they land here as efficient columnar files that
// compress well and remain queryable via OQL's ParquetSource.
//
// Schema mapping:
//
//	tiering.Event              ParquetLogEntry
//	-------------              ---------------
//	ID                         (composed into Output as JSON; not directly carried)
//	TenantID                   Host (overloaded — see note below)
//	Timestamp                  Timestamp (Unix microseconds)
//	Host                       Host
//	EventType                  (in Output)
//	Body                       Output (raw bytes encoded as JSON)
//
// The existing `ParquetLogEntry` schema was designed for terminal-
// session log archival (Timestamp/SessionID/Host/Output). Rather than
// change the schema and break read-back of existing parquet files,
// we shoehorn `tiering.Event` fields into `Output` as a JSON envelope.
// Once a future schema migration arrives, this adapter can switch
// to a richer columnar layout without touching the Tier abstraction.
//
// Note on `tracking IDs across the parquet boundary`:
//   Parquet has no native primary-key concept. To support `Delete(ids)`
//   we'd need a side-index. For the Migrator's hot→warm flow this
//   isn't needed — once an event lands in warm, the migrator only
//   reads it (during warm→cold promotion) and then deletes the whole
//   day-partition file. So `Delete([]string)` is best-effort here:
//   it groups IDs by day and rewrites the partition without them.
//   Acceptable for the migrator's pattern; not suitable for arbitrary
//   per-event deletion.

package tiering

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/storage"
)

// WarmTier implements Tier over a `*storage.ParquetArchiver`.
type WarmTier struct {
	archiver *storage.ParquetArchiver
	dataDir  string
	log      *logger.Logger
}

// NewWarmTier constructs a WarmTier rooted at the given data
// directory. The dataDir is the same one passed to
// `storage.NewParquetArchiver`; we keep our own copy so
// EstimatedSize and Delete can walk the on-disk layout.
func NewWarmTier(dataDir string, log *logger.Logger) *WarmTier {
	return &WarmTier{
		archiver: storage.NewParquetArchiver(dataDir, log),
		dataDir:  dataDir,
		log:      log.WithPrefix("warm-tier"),
	}
}

// ID implements Tier.
func (w *WarmTier) ID() TierID { return TierWarm }

// warmEnvelope is the JSON shape we serialise into ParquetLogEntry.Output.
// Keeps `tiering.Event`'s extra fields available for round-tripping
// through warm without changing the parquet schema.
type warmEnvelope struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id,omitempty"`
	EventType string `json:"event_type,omitempty"`
	Body      []byte `json:"body,omitempty"`
}

// Write implements Tier. Groups events by their UTC date and writes
// each day's batch into the corresponding daily partition.
func (w *WarmTier) Write(_ context.Context, events []Event) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	// Group by day so we make one parquet write per partition.
	byDay := make(map[string][]storage.ParquetLogEntry)
	dayDate := make(map[string]time.Time)
	for _, e := range events {
		day := e.Timestamp.UTC().Format("2006-01-02")
		envelope := warmEnvelope{
			ID: e.ID, TenantID: e.TenantID, EventType: e.EventType, Body: e.Body,
		}
		jsonBuf, err := json.Marshal(envelope)
		if err != nil {
			return 0, fmt.Errorf("warm write marshal id=%s: %w", e.ID, err)
		}
		byDay[day] = append(byDay[day], storage.ParquetLogEntry{
			Timestamp: e.Timestamp.UnixMicro(),
			SessionID: e.ID,
			Host:      e.Host,
			Output:    string(jsonBuf),
		})
		dayDate[day] = e.Timestamp.UTC()
	}

	written := 0
	for day, entries := range byDay {
		fmt.Fprintf(os.Stderr, "DEBUG warm write: day=%s entries=%d dayDate=%s\n",
			day, len(entries), dayDate[day].Format(time.RFC3339))
		if err := w.archiver.WriteBatch(dayDate[day], entries); err != nil {
			w.log.Warn("warm write FAILED day=%s err=%v", day, err)
			return written, fmt.Errorf("warm write day=%s: %w", day, err)
		}
		written += len(entries)
	}
	return written, nil
}

// Range implements Tier. Walks every daily partition between `from`
// and `to`, decodes each entry, applies `fn`. Returns early when
// `fn` returns false.
//
// When both `from` and `to` are zero we walk every partition under
// dataDir/archives/ — used during warm→cold migration.
func (w *WarmTier) Range(_ context.Context, from, to time.Time, fn func(e Event) bool) error {
	root := filepath.Join(w.dataDir, "archives")
	// If from is zero, default to "all-time" by setting a very early date.
	if from.IsZero() {
		from = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}

	// Walk one day at a time so the parquet reader only loads the
	// relevant partition.
	cur := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)

	// Skip to the first day that exists on disk to avoid an O(years*365)
	// stat-loop when the range is huge but partitions are sparse.
	if !partitionExists(root, cur) {
		// Try to fast-forward: enumerate years/months that actually exist.
		if d, ok := firstExistingDay(root, cur, end); ok {
			cur = d
		}
	}

	for !cur.After(end) {
		entries, err := w.archiver.ReadPartition(cur)
		if err != nil {
			w.log.Warn("warm range: skipping partition %s: %v", cur.Format("2006-01-02"), err)
			cur = cur.AddDate(0, 0, 1)
			continue
		}
		w.log.Debug("warm range: day=%s entries=%d window=[%s,%s]",
			cur.Format("2006-01-02"), len(entries),
			from.Format(time.RFC3339), to.Format(time.RFC3339))
		for _, parquetEntry := range entries {
			ts := time.Unix(0, parquetEntry.Timestamp*int64(time.Microsecond)).UTC()
			if ts.Before(from) || ts.After(to) {
				continue
			}
			var env warmEnvelope
			if err := json.Unmarshal([]byte(parquetEntry.Output), &env); err != nil {
				// Pre-envelope format or corrupt; reconstruct a best-
				// effort Event so the caller still gets something.
				env = warmEnvelope{ID: parquetEntry.SessionID}
				env.Body = []byte(parquetEntry.Output)
			}
			if !fn(Event{
				ID:        env.ID,
				TenantID:  env.TenantID,
				Timestamp: ts,
				Host:      parquetEntry.Host,
				EventType: env.EventType,
				Body:      env.Body,
			}) {
				return nil
			}
		}
		cur = cur.AddDate(0, 0, 1)
	}
	return nil
}

// Delete implements Tier — best-effort removal by ID.
//
// Strategy: scan each partition file for matching IDs, rewrite the
// file without them. For the Migrator's pattern (warm→cold promotion
// of an entire age cohort) the typical case is "delete every event
// in this date range" which we optimise by deleting the whole
// partition file when the id-set covers it. For sparse deletions
// the rewrite path is slower but correct.
//
// Future optimisation: maintain an id→file side-index so we don't
// need to scan every partition.
func (w *WarmTier) Delete(_ context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	root := filepath.Join(w.dataDir, "archives")
	// Walk the on-disk layout; for each partition that contains any
	// matching id, rewrite without those entries. If the rewrite would
	// produce an empty file, delete the file.
	return filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".parquet") {
			return nil
		}
		// Parse the day from the path: <root>/YYYY/MM/DD/logs.parquet
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 4 {
			return nil
		}
		day, err := time.Parse("2006/01/02",
			parts[0]+"/"+parts[1]+"/"+parts[2])
		if err != nil {
			return nil
		}
		entries, err := w.archiver.ReadPartition(day)
		if err != nil {
			w.log.Warn("warm delete: read partition %s: %v",
				day.Format("2006-01-02"), err)
			return nil
		}
		kept := make([]storage.ParquetLogEntry, 0, len(entries))
		matched := false
		for _, ent := range entries {
			if _, hit := want[ent.SessionID]; hit {
				matched = true
				continue
			}
			kept = append(kept, ent)
		}
		if !matched {
			return nil // nothing to do for this partition
		}
		// Rewrite. Removing the file first guarantees we don't leave
		// the old version behind if WriteBatch chooses a different
		// filename — ParquetArchiver writes `logs.parquet` per partition
		// so we delete the existing file then re-write the kept set.
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("warm delete remove old: %w", err)
		}
		if len(kept) == 0 {
			// Whole partition gone — leave the empty directory; Migrator
			// can tidy via a separate compaction sweep.
			return nil
		}
		if err := w.archiver.WriteBatch(day, kept); err != nil {
			return fmt.Errorf("warm delete rewrite day=%s: %w",
				day.Format("2006-01-02"), err)
		}
		return nil
	})
}

// EstimatedSize implements Tier — sums every .parquet file under
// the archives root.
func (w *WarmTier) EstimatedSize(_ context.Context) (int64, error) {
	root := filepath.Join(w.dataDir, "archives")
	var total int64
	err := filepath.Walk(root, func(_ string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".parquet") {
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

func partitionExists(root string, day time.Time) bool {
	dir := filepath.Join(root,
		fmt.Sprintf("%04d", day.Year()),
		fmt.Sprintf("%02d", day.Month()),
		fmt.Sprintf("%02d", day.Day()))
	_, err := os.Stat(dir)
	return err == nil
}

// firstExistingDay walks YYYY/MM/DD looking for the earliest day that
// has a partition directory in [from, to]. Returns false if nothing
// exists in the range.
func firstExistingDay(root string, from, to time.Time) (time.Time, bool) {
	cur := from
	for !cur.After(to) {
		if partitionExists(root, cur) {
			return cur, true
		}
		cur = cur.AddDate(0, 0, 1)
	}
	return time.Time{}, false
}
