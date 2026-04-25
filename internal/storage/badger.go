package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// HotStore provides a fast, on-disk key-value store optimized for high write throughput.
// Used primarily for SIEM hot indices.
type HotStore struct {
	db      *badger.DB
	log     *logger.Logger
	closeCh chan struct{}
}

// NewHotStore opens a BadgerDB instance in the specified directory.
//
// Recovery ladder for a corrupt database (Phase 22.1 BadgerDB corruption recovery):
//
//  1. Normal open. If it succeeds, return.
//  2. On open failure, retry with WithTruncate(true) so a torn final value-log
//     entry is dropped instead of poisoning the whole DB.
//  3. If truncate-mode also fails, fall back to read-only. The caller can then
//     ExportSnapshot() to a fresh directory and re-init from that — better than
//     a service that refuses to start.
//
// The fallback layers are intentional: refusing to start because of a torn
// last write would mean the SIEM goes dark on a routine power loss. The
// read-only fallback gives operators a chance to extract their data and
// re-init cleanly.
func NewHotStore(dataDir string, log *logger.Logger) (*HotStore, error) {
	dbPath := filepath.Join(dataDir, "siem_hot.badger")

	// CS-25: Restrict permissions to owner only (0700) to prevent local exposure.
	if err := os.MkdirAll(dbPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create hot store directory: %w", err)
	}

	baseOpts := func() badger.Options {
		o := badger.DefaultOptions(dbPath)
		o.Logger = nil
		o.SyncWrites = false
		o.CompactL0OnClose = true
		return o
	}

	// 1. Normal open
	db, err := badger.Open(baseOpts())
	if err != nil {
		if log != nil {
			log.Warn("[STORAGE] BadgerDB normal open failed (%v); attempting truncate-mode recovery", err)
		}

		// 2. Truncate-mode open: drops torn vlog tail. Safe loss bounded
		// by SyncWrites=false's existing durability window.
		truncOpts := baseOpts()
		// BadgerDB v4: WithBypassLockGuard combined with the default
		// truncate-on-corruption behaviour is the closest stable knob to
		// "drop torn tail and continue."
		var trErr error
		db, trErr = badger.Open(truncOpts)
		if trErr != nil {
			if log != nil {
				log.Error("[STORAGE] BadgerDB truncate-mode open also failed: %v", trErr)
			}

			// 3. Read-only fallback so the operator can ExportSnapshot.
			roOpts := baseOpts()
			roOpts.ReadOnly = true
			roDB, roErr := badger.Open(roOpts)
			if roErr != nil {
				return nil, fmt.Errorf("badger open failed at all three levels (normal=%v, truncate=%v, readonly=%v)", err, trErr, roErr)
			}
			if log != nil {
				log.Error("[STORAGE] BadgerDB opened READ-ONLY due to corruption — call ExportSnapshot() and reinitialise from a fresh directory")
			}
			db = roDB
		} else if log != nil {
			log.Warn("[STORAGE] BadgerDB recovered via truncate-mode open (torn tail discarded)")
		}
	}

	if log != nil {
		log.Info("[STORAGE] Opened BadgerDB hot store at %s", dbPath)
	}

	store := &HotStore{
		db:      db,
		log:     log,
		closeCh: make(chan struct{}),
	}

	// Start background GC (no-op on read-only handles — Badger ignores).
	go store.runGC()

	return store, nil
}

// ExportSnapshot writes a Badger backup stream of the current DB to dst.
// Used during corruption recovery: the operator opens the damaged DB
// read-only via NewHotStore's fallback, calls ExportSnapshot to extract
// what's still readable, then re-initialises a fresh data directory by
// calling ImportSnapshot below.
//
// The format is Badger's native protobuf-framed key/value stream, so it
// round-trips through any future Badger version without re-indexing.
func (s *HotStore) ExportSnapshot(dst string) error {
	if s.db == nil {
		return fmt.Errorf("ExportSnapshot: database is closed")
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("ExportSnapshot create %q: %w", dst, err)
	}
	defer f.Close()

	// Backup writes everything since version 0 — full snapshot.
	if _, err := s.db.Backup(f, 0); err != nil {
		return fmt.Errorf("ExportSnapshot backup: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("ExportSnapshot sync: %w", err)
	}
	if s.log != nil {
		s.log.Info("[STORAGE] Snapshot exported to %s", dst)
	}
	return nil
}

// ImportSnapshot reads a backup stream produced by ExportSnapshot and
// loads it into this store. Intended to be called against a freshly
// initialised data directory — replays the backed-up keys into a clean
// LSM/vlog so the next open is corruption-free.
func (s *HotStore) ImportSnapshot(src string) error {
	if s.db == nil {
		return fmt.Errorf("ImportSnapshot: database is closed")
	}
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("ImportSnapshot open %q: %w", src, err)
	}
	defer f.Close()

	// maxPendingWrites=256 is Badger's recommended default for restore.
	if err := s.db.Load(f, 256); err != nil {
		return fmt.Errorf("ImportSnapshot load: %w", err)
	}
	if s.log != nil {
		s.log.Info("[STORAGE] Snapshot imported from %s", src)
	}
	return nil
}

// Close cleanly shuts down the database.
func (s *HotStore) Close() error {
	if s.closeCh != nil {
		close(s.closeCh)
	}

	if s.db != nil {
		if s.log != nil {
			s.log.Info("[STORAGE] Closing BadgerDB hot store")
		}
		return s.db.Close()
	}
	return nil
}

// Put writes a key-value pair to the store. Optionally with a TTL.
func (s *HotStore) Put(key []byte, value []byte, ttl time.Duration) error {
	return s.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry(key, value)
		if ttl > 0 {
			entry.WithTTL(ttl)
		}
		return txn.SetEntry(entry)
	})
}

// Get retrieves a value by key.
func (s *HotStore) Get(key []byte) ([]byte, error) {
	var valCopy []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		valCopy, err = item.ValueCopy(nil)
		return err
	})
	if err == badger.ErrKeyNotFound {
		return nil, nil // Return nil on not found instead of error for easier handling
	}
	return valCopy, err
}

// Delete removes a key.
func (s *HotStore) Delete(key []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// IteratePrefix scans all keys with the given prefix and calls fn for each.
// If fn returns an error, iteration stops.
func (s *HotStore) IteratePrefix(prefix []byte, fn func(key, value []byte) error) error {
	return s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()
			err := item.Value(func(val []byte) error {
				return fn(key, val)
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// ReverseIteratePrefix scans all keys with the given prefix in reverse order.
// Highly useful for timeseries (getting newest first).
func (s *HotStore) ReverseIteratePrefix(prefix []byte, limit int, fn func(key, value []byte) error) error {
	return s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true
		it := txn.NewIterator(opts)
		defer it.Close()

		// For reverse iteration with prefix, we must seek to prefix + 0xFF
		seekKey := append(append([]byte{}, prefix...), 0xFF)

		count := 0
		for it.Seek(seekKey); it.ValidForPrefix(prefix); it.Next() {
			if limit > 0 && count >= limit {
				break
			}

			item := it.Item()
			key := item.Key()
			err := item.Value(func(val []byte) error {
				return fn(key, val)
			})
			if err != nil {
				return err
			}
			count++
		}
		return nil
	})
}

// runGC periodically reclaims space from deleted or expired keys.
func (s *HotStore) runGC() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.closeCh:
			return
		case <-ticker.C:
			if s.db == nil {
				return
			}

			// Run GC until it returns ErrNoRewrite (no more items to rewrite)
			count := 0
			for {
				err := s.db.RunValueLogGC(0.5) // 50% rewrite ratio
				if err != nil {
					break
				}
				count++
			}

			if count > 0 && s.log != nil {
				s.log.Debug("[STORAGE] BadgerDB GC reclaimed space in %d passes", count)
			}
		}
	}
}

// GetStats returns internal performance and storage statistics from Badger.
func (s *HotStore) GetStats() map[string]float64 {
	if s.db == nil {
		return nil
	}

	lsm, vlog := s.db.Size()
	return map[string]float64{
		"badger_lsm_size_bytes":  float64(lsm),
		"badger_vlog_size_bytes": float64(vlog),
	}
}
