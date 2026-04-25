package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Cursor persists the agent's WAL sequence state across restarts.
//
// Two values matter:
//
//   - NextSeq    — the next sequence number to assign on Write. Persisted
//     so a crash during a batch does not cause seq reuse, which would
//     otherwise let the server treat fresh events as already-seen
//     duplicates.
//
//   - LastAckedSeq — the highest sequence number the server has confirmed
//     ingest for. Drives partial WAL truncation (`TruncateUpTo`) so an
//     ack for the first half of a batch frees that half on disk even if
//     the second half still needs retry.
//
// The file is written via temp-file + rename to avoid torn writes during
// crashes.
type Cursor struct {
	path string
	mu   sync.Mutex

	NextSeq       uint64 `json:"next_seq"`
	LastAckedSeq  uint64 `json:"last_acked_seq"`
}

// NewCursor opens or creates the cursor file at <dir>/cursor.json. If the
// file is missing, both counters start at zero. If the file is corrupt
// (parse failure), we fail loudly — silently zeroing would re-issue
// already-sent sequences and cause the server to drop fresh events as
// duplicates.
func NewCursor(dir string) (*Cursor, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create cursor dir: %w", err)
	}

	c := &Cursor{path: filepath.Join(dir, "cursor.json")}

	data, err := os.ReadFile(c.path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read cursor: %w", err)
		}
		// First boot — leave at zero
		return c, nil
	}

	if err := json.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("cursor parse (refusing to zero counters — manual recovery needed): %w", err)
	}
	return c, nil
}

// Reserve atomically allocates the next sequence number, advances NextSeq,
// and persists the new state. Callers must use the returned seq for the
// event being written.
func (c *Cursor) Reserve() (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.NextSeq++
	if err := c.persistLocked(); err != nil {
		// Roll back the in-memory advance — we'd rather refuse the write
		// than risk a fsync gap that could lose the cursor while the WAL
		// already has the event.
		c.NextSeq--
		return 0, err
	}
	return c.NextSeq, nil
}

// MarkAcked records the highest sequence number the server has confirmed.
// Idempotent and monotonic — out-of-order or stale acks are dropped.
func (c *Cursor) MarkAcked(seq uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if seq <= c.LastAckedSeq {
		return nil
	}
	prev := c.LastAckedSeq
	c.LastAckedSeq = seq
	if err := c.persistLocked(); err != nil {
		c.LastAckedSeq = prev
		return err
	}
	return nil
}

// Snapshot returns a point-in-time copy of (NextSeq, LastAckedSeq).
func (c *Cursor) Snapshot() (uint64, uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.NextSeq, c.LastAckedSeq
}

// persistLocked writes the cursor to disk via temp-file + rename. Caller
// must hold c.mu.
func (c *Cursor) persistLocked() error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("cursor marshal: %w", err)
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("cursor temp write: %w", err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return fmt.Errorf("cursor rename: %w", err)
	}
	return nil
}
