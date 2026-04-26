// HotTier — BadgerDB-backed implementation of the Tier interface.
//
// Phase 31 — wires the existing `internal/storage.HotStore` (the same
// BadgerDB instance the SIEM ingest pipeline already uses) under the
// tier abstraction. Once the ingest pipeline is migrated to write
// through HotTier (planned 22.3 follow-up), this is the only thing
// that touches the hot keyspace — and the Migrator can promote
// events Hot→Warm safely without coordinating with concurrent
// pipeline writes.
//
// Key encoding:
//
//	tier:hot:<unix_nano_padded>/<event_id>
//
// The Unix-nano timestamp is left-zero-padded to 19 chars so
// lexicographic order matches chronological order — Badger's
// IteratePrefix becomes a time-ordered scan for free. Event IDs go
// after the timestamp so multiple events at the same nano don't
// collide.
//
// The value is the JSON-serialised tiering.Event. Compact and
// language-agnostic so a future rewrite in a different language
// can still read the on-disk format.

package tiering

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/storage"
)

// hotKeyPrefix is the byte sequence every hot-tier key starts with.
// Co-locates with other tier prefixes so a future operator browsing
// the BadgerDB raw keyspace can identify tier traffic at a glance.
const hotKeyPrefix = "tier:hot:"

// HotTier implements Tier over a `*storage.HotStore` (BadgerDB).
type HotTier struct {
	store *storage.HotStore
}

// NewHotTier constructs a HotTier over the given BadgerDB store.
// The caller retains ownership of the store — HotTier does NOT
// close it on Drop / Stop so multiple tier instances can share one
// underlying database (e.g. for unit tests).
func NewHotTier(store *storage.HotStore) *HotTier {
	return &HotTier{store: store}
}

// ID implements Tier.
func (h *HotTier) ID() TierID { return TierHot }

// encodeHotKey produces a lexicographically-ordered, time-prefixed
// key. The fixed-width 19-digit padding means events at any time
// from now until ~2262 sort correctly.
func encodeHotKey(ts time.Time, id string) []byte {
	return []byte(fmt.Sprintf("%s%019d/%s", hotKeyPrefix, ts.UnixNano(), id))
}

// decodeHotKey extracts the timestamp + id back out. Returns an
// error if the key doesn't match the expected shape — defensive
// against any process writing under the same prefix.
func decodeHotKey(k []byte) (time.Time, string, error) {
	s := string(k)
	if !strings.HasPrefix(s, hotKeyPrefix) {
		return time.Time{}, "", fmt.Errorf("not a hot-tier key: %q", s)
	}
	rest := s[len(hotKeyPrefix):]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return time.Time{}, "", fmt.Errorf("malformed hot key: %q", s)
	}
	tsPart := rest[:slash]
	id := rest[slash+1:]
	var nanos int64
	if _, err := fmt.Sscanf(tsPart, "%d", &nanos); err != nil {
		return time.Time{}, "", fmt.Errorf("hot key timestamp parse: %w", err)
	}
	return time.Unix(0, nanos).UTC(), id, nil
}

// Write implements Tier. Each event is serialised to JSON and
// stored under its time-prefixed key.
//
// Performance note: Badger's `Put` is synchronous — for high-volume
// writes we'd batch via `db.Update` directly. The Migrator's batch
// size (default 10k) is small enough that one-Put-per-event is fine;
// optimising the hot path comes later.
func (h *HotTier) Write(_ context.Context, events []Event) (int, error) {
	written := 0
	for _, e := range events {
		buf, err := json.Marshal(e)
		if err != nil {
			return written, fmt.Errorf("hot write marshal id=%s: %w", e.ID, err)
		}
		key := encodeHotKey(e.Timestamp, e.ID)
		// TTL=0 means "no auto-expiry" — the Migrator handles GC.
		if err := h.store.Put(key, buf, 0); err != nil {
			return written, fmt.Errorf("hot write put id=%s: %w", e.ID, err)
		}
		written++
	}
	return written, nil
}

// Range implements Tier. Walks events in chronological order and
// applies `fn` to each that falls in [from, to]. Empty `from` /
// `to` means open-ended; both empty walks the whole tier.
//
// Returns early when `fn` returns false (used by the Migrator's
// batch budget).
func (h *HotTier) Range(_ context.Context, from, to time.Time, fn func(e Event) bool) error {
	prefix := []byte(hotKeyPrefix)
	var iterErr error
	err := h.store.IteratePrefix(prefix, func(k, v []byte) error {
		ts, _, err := decodeHotKey(k)
		if err != nil {
			// Skip unparseable keys silently — they're not ours.
			return nil
		}
		if !from.IsZero() && ts.Before(from) {
			return nil
		}
		if !to.IsZero() && ts.After(to) {
			return nil
		}
		var e Event
		if err := json.Unmarshal(v, &e); err != nil {
			// Skip — value is corrupt or pre-format-change. Don't fail
			// the whole iteration.
			return nil
		}
		if !fn(e) {
			// Sentinel for "stop iterating" — we have to surface this
			// via the closure since IteratePrefix doesn't expose a
			// stop signal directly.
			iterErr = errStopIter
			return errStopIter
		}
		return nil
	})
	if err != nil && err != errStopIter && iterErr != errStopIter {
		return err
	}
	return nil
}

// Delete implements Tier. Idempotent — deleting a missing key is
// not an error.
//
// Implementation: we don't know the timestamp from just the ID, so
// we IteratePrefix and Delete matches. For the migrator's typical
// ~10k-batch sizes this is fast enough; a future optimisation would
// keep an id→key index.
func (h *HotTier) Delete(_ context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	prefix := []byte(hotKeyPrefix)
	toDelete := make([][]byte, 0, len(ids))
	if err := h.store.IteratePrefix(prefix, func(k, _ []byte) error {
		_, id, err := decodeHotKey(k)
		if err != nil {
			return nil
		}
		if _, hit := want[id]; hit {
			// Copy the key — Badger reuses iterator buffers.
			cp := make([]byte, len(k))
			copy(cp, k)
			toDelete = append(toDelete, cp)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("hot delete scan: %w", err)
	}
	for _, k := range toDelete {
		if err := h.store.Delete(k); err != nil {
			return fmt.Errorf("hot delete: %w", err)
		}
	}
	return nil
}

// EstimatedSize implements Tier. Sums the LSM + value-log sizes
// reported by Badger's runtime stats. Best-effort; -1 if Badger
// can't supply a number.
func (h *HotTier) EstimatedSize(_ context.Context) (int64, error) {
	stats := h.store.GetStats()
	lsm, _ := stats["lsm_size_bytes"]
	vlog, _ := stats["vlog_size_bytes"]
	if lsm == 0 && vlog == 0 {
		return -1, nil
	}
	return int64(lsm + vlog), nil
}

// errStopIter is the sentinel returned by HotTier.Range's inner
// closure to bail out of an in-progress IteratePrefix call. It
// exists because Badger's iterator API only stops on error and we
// don't want a real error pollutiing the Range return.
var errStopIter = fmt.Errorf("tiering: stop iteration (sentinel)")
