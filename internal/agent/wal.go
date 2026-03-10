package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// WAL provides Write-Ahead Logging for offline event buffering.
// Events are written to disk when the server is unreachable and
// replayed when connectivity is restored.
type WAL struct {
	dir     string
	mu      sync.Mutex
	file    *os.File
	encoder *json.Encoder
	count   int64
}

// NewWAL creates a new Write-Ahead Log at the given directory.
func NewWAL(dir string) (*WAL, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create WAL dir: %w", err)
	}

	filename := filepath.Join(dir, "current.wal")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("open WAL: %w", err)
	}

	return &WAL{
		dir:     dir,
		file:    f,
		encoder: json.NewEncoder(f),
	}, nil
}

// Write appends an event to the WAL.
func (w *WAL) Write(event Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.encoder.Encode(event); err != nil {
		return fmt.Errorf("WAL write: %w", err)
	}
	w.count++
	return nil
}

// ReadAll reads all buffered events from the WAL for replay.
func (w *WAL) ReadAll() ([]Event, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

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
			break
		}
		events = append(events, ev)
	}

	return events, nil
}

// Truncate clears the WAL after successful flush.
func (w *WAL) Truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		w.file.Close()
	}

	filename := filepath.Join(w.dir, "current.wal")
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("truncate WAL: %w", err)
	}

	w.file = f
	w.encoder = json.NewEncoder(f)
	w.count = 0
	return nil
}

// Count returns the number of buffered events.
func (w *WAL) Count() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.count
}

// Close closes the WAL file.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
