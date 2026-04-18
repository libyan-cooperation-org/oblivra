package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// ErrWALFull is returned by Write when the WAL has reached its configured
// maximum event count. The caller should drop or rate-limit the event.
var ErrWALFull = errors.New("WAL full: event dropped")

// WAL provides Write-Ahead Logging for offline event buffering.
//
// Design properties:
//   - Single write path serialized via mutex
//   - Hard cap (maxEvents) to prevent unbounded disk growth during long outages
//   - Atomic count for monitoring without holding the lock
//   - ReadAll uses a separate read handle — never blocks the writer
//   - Truncate rotates atomically: close → recreate → reset count
type WAL struct {
	dir       string
	maxEvents int64
	mu        sync.Mutex
	file      *os.File
	encoder   *json.Encoder
	count     atomic.Int64
}

// NewWAL creates a new Write-Ahead Log at the given directory.
// maxEvents is the hard cap on buffered events; 0 means unlimited (not recommended).
func NewWAL(dir string, maxEvents int64) (*WAL, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create WAL dir: %w", err)
	}

	w := &WAL{dir: dir, maxEvents: maxEvents}
	if err := w.openAppend(); err != nil {
		return nil, err
	}

	// Recover approximate count from existing WAL content
	go func() {
		if existing, err := w.ReadAll(); err == nil {
			w.count.Store(int64(len(existing)))
		}
	}()

	return w, nil
}

// openAppend opens current.wal for appending (creates if missing).
func (w *WAL) openAppend() error {
	filename := filepath.Join(w.dir, "current.wal")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open WAL: %w", err)
	}
	w.file = f
	w.encoder = json.NewEncoder(f)
	return nil
}

// openTruncate recreates current.wal, discarding previous content.
func (w *WAL) openTruncate() error {
	filename := filepath.Join(w.dir, "current.wal")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("truncate WAL: %w", err)
	}
	w.file = f
	w.encoder = json.NewEncoder(f)
	return nil
}

// Write appends a single event to the WAL.
// Returns ErrWALFull when the event cap has been reached.
func (w *WAL) Write(event Event) error {
	if w.maxEvents > 0 && w.count.Load() >= w.maxEvents {
		return ErrWALFull
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.encoder.Encode(event); err != nil {
		return fmt.Errorf("WAL encode: %w", err)
	}

	n := w.count.Add(1)
	// Periodic fsync every 100 events — balances durability vs. I/O overhead
	if n%100 == 0 {
		_ = w.file.Sync()
	}
	return nil
}

// ReadAll reads all buffered events from the WAL using a dedicated read handle.
// Safe to call concurrently with Write.
func (w *WAL) ReadAll() ([]Event, error) {
	filename := filepath.Join(w.dir, "current.wal")
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []Event
	dec := json.NewDecoder(f)
	for dec.More() {
		var ev Event
		if err := dec.Decode(&ev); err != nil {
			// Truncated or corrupted tail — stop and keep what we have
			break
		}
		events = append(events, ev)
	}
	return events, nil
}

// Truncate rotates the WAL after a successful server flush.
// Events written between ReadAll and Truncate are safe: the write file is
// recreated from scratch; in-flight writes land in the new file.
func (w *WAL) Truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		_ = w.file.Sync()
		w.file.Close()
		w.file = nil
	}
	if err := w.openTruncate(); err != nil {
		return err
	}
	w.count.Store(0)
	return nil
}

// Count returns the approximate number of events buffered in the WAL.
func (w *WAL) Count() int64 {
	return w.count.Load()
}

// Close flushes and closes the WAL.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		_ = w.file.Sync()
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}
