// Package hot is the BadgerDB-backed hot store for ingested events.
//
// Key layout:
//
//	tenant:{tenantID}:event:{nanoTimestamp}:{eventID}
//
// Big-endian nano timestamps make BadgerDB's ordered iteration give us free
// chronological scans and efficient time-range queries. Per-tenant prefixes
// make it structurally impossible to leak events across tenants — there is no
// shared keyspace to escape.
package hot

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/kingknull/oblivra/internal/events"
)

type Options struct {
	Dir      string
	InMemory bool
}

type Store struct {
	db    *badger.DB
	count atomic.Int64
}

func Open(opts Options) (*Store, error) {
	if opts.Dir == "" && !opts.InMemory {
		return nil, errors.New("hot: Dir is required unless InMemory is set")
	}
	bopts := badger.DefaultOptions(opts.Dir)
	bopts.Logger = nil // silence Badger's own logger; we have slog
	if opts.InMemory {
		bopts = bopts.WithInMemory(true).WithDir("").WithValueDir("")
	}
	db, err := badger.Open(bopts)
	if err != nil {
		return nil, fmt.Errorf("hot: open badger: %w", err)
	}

	s := &Store{db: db}
	// Best-effort initial count for warm-start observability.
	_ = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		var n int64
		for it.Rewind(); it.Valid(); it.Next() {
			n++
		}
		s.count.Store(n)
		return nil
	})
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Count() int64 { return s.count.Load() }

// Delete removes a batch of events by (tenantID, eventID). Used by the
// tiering migrator after a successful warm-tier write to reclaim hot space.
// Missing IDs are silently skipped.
func (s *Store) Delete(tenantID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	prefix := tenantPrefix(tenantID)
	deleted := int64(0)
	err := s.db.Update(func(txn *badger.Txn) error {
		iopts := badger.DefaultIteratorOptions
		iopts.Prefix = prefix
		it := txn.NewIterator(iopts)
		defer it.Close()
		var keys [][]byte
		for it.Rewind(); it.Valid(); it.Next() {
			k := it.Item().KeyCopy(nil)
			lastColon := -1
			for i := len(k) - 1; i >= 0; i-- {
				if k[i] == ':' {
					lastColon = i
					break
				}
			}
			if lastColon < 0 {
				continue
			}
			id := string(k[lastColon+1:])
			if _, hit := want[id]; hit {
				keys = append(keys, k)
			}
		}
		// Iterator is closed before mutations — Badger requires it.
		it.Close()
		for _, k := range keys {
			if err := txn.Delete(k); err != nil {
				return err
			}
			deleted++
		}
		return nil
	})
	if err == nil {
		// keep counter approximately accurate
		newCount := s.count.Add(-deleted)
		if newCount < 0 {
			s.count.Store(0)
		}
	}
	return err
}

// Lookup hydrates a slice of events by (tenantID, eventID). Missing IDs are
// silently skipped — callers are expected to tolerate index/store skew.
func (s *Store) Lookup(tenantID string, ids []string) ([]events.Event, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := make([]events.Event, 0, len(ids))
	err := s.db.View(func(txn *badger.Txn) error {
		iopts := badger.DefaultIteratorOptions
		iopts.Prefix = tenantPrefix(tenantID)
		it := txn.NewIterator(iopts)
		defer it.Close()

		want := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			want[id] = struct{}{}
		}
		// Simple full-prefix scan. Phase 1b acceptable; Phase 2 will add a
		// secondary id→key index so this is O(N) per lookup batch instead of
		// O(events).
		for it.Rewind(); it.Valid() && len(out) < len(ids); it.Next() {
			k := it.Item().Key()
			// Trailing segment after the last ':' is the event ID.
			lastColon := -1
			for i := len(k) - 1; i >= 0; i-- {
				if k[i] == ':' {
					lastColon = i
					break
				}
			}
			if lastColon < 0 {
				continue
			}
			id := string(k[lastColon+1:])
			if _, hit := want[id]; !hit {
				continue
			}
			err := it.Item().Value(func(v []byte) error {
				var ev events.Event
				if err := json.Unmarshal(v, &ev); err != nil {
					return err
				}
				out = append(out, ev)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

// Put writes a single event. The event's ID and Timestamp must already be set.
func (s *Store) Put(ev *events.Event) error {
	key := buildKey(ev.TenantID, ev.Timestamp, ev.ID)
	val, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("hot: marshal: %w", err)
	}
	if err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	}); err != nil {
		return fmt.Errorf("hot: put: %w", err)
	}
	s.count.Add(1)
	return nil
}

// Range query options.
type RangeOpts struct {
	TenantID string
	From     time.Time // inclusive
	To       time.Time // inclusive (defaults to now if zero)
	Limit    int       // 0 = no limit, but caller should set one
	Reverse  bool      // newest-first
}

// Range returns events within [From, To] for the given tenant.
func (s *Store) Range(ctx context.Context, opts RangeOpts) ([]events.Event, error) {
	if opts.TenantID == "" {
		opts.TenantID = "default"
	}
	if opts.To.IsZero() {
		opts.To = time.Now().UTC()
	}
	if opts.Limit == 0 {
		opts.Limit = 1000
	}

	prefix := tenantPrefix(opts.TenantID)
	fromKey := buildKey(opts.TenantID, opts.From, "")
	toKey := buildKeyAfter(opts.TenantID, opts.To)

	out := make([]events.Event, 0, min(opts.Limit, 256))
	err := s.db.View(func(txn *badger.Txn) error {
		iopts := badger.DefaultIteratorOptions
		iopts.Prefix = prefix
		iopts.Reverse = opts.Reverse
		it := txn.NewIterator(iopts)
		defer it.Close()

		if opts.Reverse {
			// For reverse iteration with a prefix, BadgerDB requires the seek
			// key to be >= the last key in the prefix. We build a key that is
			// strictly higher than any real key by appending 0xFF bytes after
			// the prefix — this places us past all events and lets Badger walk
			// backwards from the most recent one.
			//
			// We then manually skip events newer than opts.To (they are between
			// our synthetic seek position and toKey) and stop when we fall below
			// opts.From.
			revSeek := make([]byte, len(prefix)+8)
			copy(revSeek, prefix)
			for i := len(prefix); i < len(revSeek); i++ {
				revSeek[i] = 0xFF
			}
			for it.Seek(revSeek); it.Valid(); it.Next() {
				if err := ctx.Err(); err != nil {
					return err
				}
				k := it.Item().Key()
				// Stop once we pass below the from boundary.
				if len(k) >= len(fromKey) && string(k[:len(fromKey)]) < string(fromKey) {
					break
				}
				// Skip events that are strictly after opts.To.
				if len(k) >= len(prefix)+8 {
					var stamp [8]byte
					copy(stamp[:], k[len(prefix):])
					kNano := int64(binary.BigEndian.Uint64(stamp[:]))
					if kNano > opts.To.UnixNano() {
						continue
					}
					if !opts.From.IsZero() && kNano < opts.From.UnixNano() {
						break
					}
				}
				err := it.Item().Value(func(v []byte) error {
					var ev events.Event
					if err := json.Unmarshal(v, &ev); err != nil {
						return err
					}
					out = append(out, ev)
					return nil
				})
				if err != nil {
					return err
				}
				if len(out) >= opts.Limit {
					break
				}
			}
			return nil
		}

		// Forward iteration.
		for it.Seek(fromKey); it.Valid(); it.Next() {
			if err := ctx.Err(); err != nil {
				return err
			}
			k := it.Item().Key()
			if string(k) > string(toKey) {
				break
			}
			err := it.Item().Value(func(v []byte) error {
				var ev events.Event
				if err := json.Unmarshal(v, &ev); err != nil {
					return err
				}
				out = append(out, ev)
				return nil
			})
			if err != nil {
				return err
			}
			if len(out) >= opts.Limit {
				break
			}
		}
		return nil
	})
	return out, err
}

func tenantPrefix(tenantID string) []byte {
	return []byte("tenant:" + tenantID + ":event:")
}

func buildKey(tenantID string, ts time.Time, eventID string) []byte {
	prefix := tenantPrefix(tenantID)
	var stamp [8]byte
	if !ts.IsZero() {
		binary.BigEndian.PutUint64(stamp[:], uint64(ts.UnixNano()))
	}
	out := make([]byte, 0, len(prefix)+8+1+len(eventID))
	out = append(out, prefix...)
	out = append(out, stamp[:]...)
	out = append(out, ':')
	out = append(out, []byte(eventID)...)
	return out
}

// buildKeyAfter returns a key strictly greater than every event at ts. We do
// this by encoding ts then appending a high byte so range scans include all
// events whose nano-timestamp equals ts.
func buildKeyAfter(tenantID string, ts time.Time) []byte {
	prefix := tenantPrefix(tenantID)
	var stamp [8]byte
	binary.BigEndian.PutUint64(stamp[:], uint64(ts.UnixNano()))
	out := make([]byte, 0, len(prefix)+8+1)
	out = append(out, prefix...)
	out = append(out, stamp[:]...)
	out = append(out, 0xFF)
	return out
}
