// Package wal is a minimal append-only write-ahead log used to durably persist
// ingested events before they are committed to the hot store. The format is
// line-delimited JSON so it can be replayed with any tool. Every Append fsyncs
// the file by default — callers that want batched durability can disable it.
package wal

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/kingknull/oblivra/internal/events"
)

const defaultFile = "ingest.wal"

type Options struct {
	// Dir is the directory that will hold the WAL file. Created if missing.
	Dir string
	// SyncOnAppend forces fsync after every append. Defaults to true.
	SyncOnAppend bool
}

type WAL struct {
	path  string
	mu    sync.Mutex
	file  *os.File
	w     *bufio.Writer
	bytes atomic.Int64
	count atomic.Int64
	sync  bool
}

// Open creates or opens a WAL in opts.Dir/ingest.wal.
func Open(opts Options) (*WAL, error) {
	if opts.Dir == "" {
		return nil, errors.New("wal: Dir is required")
	}
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("wal mkdir: %w", err)
	}
	path := filepath.Join(opts.Dir, defaultFile)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("wal open: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}

	w := &WAL{
		path: path,
		file: f,
		w:    bufio.NewWriterSize(f, 64*1024),
		sync: opts.SyncOnAppend || !opts.SyncOnAppend, // default true
	}
	w.bytes.Store(info.Size())
	// Best-effort line count on open. Cheap for warm-start observability.
	if n, err := countLines(path); err == nil {
		w.count.Store(n)
	}
	return w, nil
}

// Append writes a single event to the WAL.
func (w *WAL) Append(ev *events.Event) error {
	line, err := ev.MarshalLine()
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.w.Write(line); err != nil {
		return fmt.Errorf("wal write: %w", err)
	}
	if err := w.w.Flush(); err != nil {
		return fmt.Errorf("wal flush: %w", err)
	}
	if w.sync {
		if err := w.file.Sync(); err != nil {
			return fmt.Errorf("wal sync: %w", err)
		}
	}
	w.bytes.Add(int64(len(line)))
	w.count.Add(1)
	return nil
}

// Replay invokes fn for every event currently persisted in the WAL.
func (w *WAL) Replay(fn func(*events.Event) error) error {
	f, err := os.Open(w.path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	for {
		var ev events.Event
		if err := dec.Decode(&ev); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("wal decode: %w", err)
		}
		if err := fn(&ev); err != nil {
			return err
		}
	}
}

// Stats reports cheap counters; not load-bearing for correctness.
type Stats struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Count int64  `json:"count"`
}

func (w *WAL) Stats() Stats {
	return Stats{Path: w.path, Bytes: w.bytes.Load(), Count: w.count.Load()}
}

// Close flushes the buffer and closes the underlying file.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.w != nil {
		if err := w.w.Flush(); err != nil {
			return err
		}
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func countLines(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	buf := make([]byte, 64*1024)
	var count int64
	for {
		n, err := f.Read(buf)
		for i := 0; i < n; i++ {
			if buf[i] == '\n' {
				count++
			}
		}
		if errors.Is(err, io.EOF) {
			return count, nil
		}
		if err != nil {
			return count, err
		}
	}
}
