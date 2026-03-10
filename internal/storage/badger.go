package storage

import (
	"fmt"
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
func NewHotStore(dataDir string, log *logger.Logger) (*HotStore, error) {
	dbPath := filepath.Join(dataDir, "siem_hot.badger")

	opts := badger.DefaultOptions(dbPath)
	// Optimize for SSDs/NVMe & reduce memory footprint
	opts.Logger = nil       // Disable verbose internal badger logging
	opts.SyncWrites = false // Async writes for throughput (we have a WAL coming anyway)
	opts.CompactL0OnClose = true

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger hot store: %w", err)
	}

	if log != nil {
		log.Info("[STORAGE] Opened BadgerDB hot store at %s", dbPath)
	}

	store := &HotStore{
		db:      db,
		log:     log,
		closeCh: make(chan struct{}),
	}

	// Start background GC
	go store.runGC()

	return store, nil
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
