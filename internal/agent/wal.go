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
//   - Sequence numbers (Cursor) make the WAL idempotent across server
//     restarts: TruncateUpTo(ackedSeq) drops only the prefix the server
//     has confirmed, preserving any in-flight or post-ack writes.
type WAL struct {
	dir       string
	maxEvents int64
	mu        sync.Mutex
	file      *os.File
	encoder   *json.Encoder
	count     atomic.Int64
	cursor    *Cursor
}

// NewWAL creates a new Write-Ahead Log at the given directory.
// maxEvents is the hard cap on buffered events; 0 means unlimited (not recommended).
func NewWAL(dir string, maxEvents int64) (*WAL, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create WAL dir: %w", err)
	}

	cursor, err := NewCursor(dir)
	if err != nil {
		return nil, fmt.Errorf("open WAL cursor: %w", err)
	}

	w := &WAL{dir: dir, maxEvents: maxEvents, cursor: cursor}
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

// Cursor returns the underlying sequence cursor. Used by the transport
// layer to read LastAckedSeq before flushing and to bump it on server ack.
func (w *WAL) Cursor() *Cursor { return w.cursor }

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

// Write appends a single event to the WAL, assigning a fresh persistent
// sequence number. Returns ErrWALFull when the event cap has been reached.
//
// The seq is reserved through the Cursor before the WAL write, so a crash
// between cursor-reserve and WAL-encode at worst burns a sequence number
// (the server tolerates monotonic gaps). The reverse — WAL-write succeeding
// with a duplicate seq — is what we have to avoid, because that would let
// the server treat fresh events as already-acked replays.
func (w *WAL) Write(event Event) error {
	if w.maxEvents > 0 && w.count.Load() >= w.maxEvents {
		return ErrWALFull
	}

	seq, err := w.cursor.Reserve()
	if err != nil {
		return fmt.Errorf("WAL reserve seq: %w", err)
	}
	event.Seq = seq

	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.encoder.Encode(event); err != nil {
		return fmt.Errorf("WAL encode: %w", err)
	}

	n := w.count.Add(1)
	// Periodic fsync every 100 events — balances durability vs. I/O overhead
	if n%100 == 0 {
		if err := w.file.Sync(); err != nil {
			// Non-fatal but should be known
			fmt.Fprintf(os.Stderr, "[wal] Sync error: %v\n", err)
		}
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
//
// Deprecated: prefer TruncateUpTo(ackedSeq) which preserves writes that
// arrived between ReadAll and the truncate. Plain Truncate erases the
// whole WAL including any post-flush writes — the data-loss vector that
// Phase 22.1's reconnect guarantee was originally meant to fix.
func (w *WAL) Truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		if err := w.file.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "[wal] Truncate sync error: %v\n", err)
		}
		w.file.Close()
		w.file = nil
	}
	if err := w.openTruncate(); err != nil {
		return err
	}
	w.count.Store(0)
	return nil
}

// TruncateUpTo rewrites the WAL keeping only events with Seq > ackedSeq.
// This is the correct partial-truncate semantic for ack-driven flushing:
// records confirmed by the server are dropped; records that arrived in the
// flush-window race (or are still pending retry) are preserved.
//
// Implementation: read the current WAL, filter, atomically replace via
// temp-file + rename. This is O(n) on disk but happens at most once per
// flush cycle (every 5s by default) and only on successful ack.
func (w *WAL) TruncateUpTo(ackedSeq uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	current := filepath.Join(w.dir, "current.wal")
	tmp := filepath.Join(w.dir, "current.wal.tmp")

	// Close the active write handle so we can safely move the file.
	if w.file != nil {
		if err := w.file.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "[wal] TruncateUpTo sync error: %v\n", err)
		}
		w.file.Close()
		w.file = nil
	}

	in, err := os.Open(current)
	if err != nil {
		if os.IsNotExist(err) {
			// Nothing to truncate; reopen append handle and exit clean.
			return w.openAppend()
		}
		_ = w.openAppend()
		return fmt.Errorf("WAL open for filter: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		_ = w.openAppend()
		return fmt.Errorf("WAL temp create: %w", err)
	}

	dec := json.NewDecoder(in)
	enc := json.NewEncoder(out)
	kept := int64(0)
	for dec.More() {
		var ev Event
		if err := dec.Decode(&ev); err != nil {
			// Truncated tail — stop; whatever survived above is safe.
			break
		}
		if ev.Seq > ackedSeq {
			if err := enc.Encode(ev); err != nil {
				out.Close()
				_ = os.Remove(tmp)
				_ = w.openAppend()
				return fmt.Errorf("WAL temp encode: %w", err)
			}
			kept++
		}
	}

	if err := out.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "[wal] TruncateUpTo temp sync error: %v\n", err)
	}
	out.Close()

	if err := os.Rename(tmp, current); err != nil {
		_ = w.openAppend()
		return fmt.Errorf("WAL atomic rename: %w", err)
	}

	// Reopen the append handle so subsequent Writes land in the new file.
	if err := w.openAppend(); err != nil {
		return err
	}
	w.count.Store(kept)
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
		if err := w.file.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "[wal] Close sync error: %v\n", err)
		}
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}
