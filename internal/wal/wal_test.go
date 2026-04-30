package wal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

func TestRoundtripAndReplay(t *testing.T) {
	dir := t.TempDir()
	w, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		ev := &events.Event{Source: events.SourceREST, HostID: "h", Message: "m"}
		if err := ev.Validate(); err != nil {
			t.Fatal(err)
		}
		if err := w.Append(ev); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	w2, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()
	st := w2.Stats()
	if st.Count != 10 {
		t.Errorf("re-open count = %d, want 10", st.Count)
	}

	seen := 0
	if err := w2.Replay(func(ev *events.Event) error {
		seen++
		if !ev.VerifyHash() {
			t.Errorf("event %s lost hash integrity through WAL", ev.ID)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if seen != 10 {
		t.Errorf("replay saw %d, want 10", seen)
	}
}

// TestCrashRecovery simulates a power-loss mid-write: appends a few events,
// truncates the file mid-line, then re-opens. Replay must surface the
// torn-write boundary as a decode error rather than panicking, and the new
// WAL handle must continue accepting writes.
func TestCrashRecovery(t *testing.T) {
	dir := t.TempDir()
	w, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		ev := &events.Event{Source: events.SourceREST, HostID: "h", Message: "x"}
		_ = ev.Validate()
		if err := w.Append(ev); err != nil {
			t.Fatal(err)
		}
	}
	_ = w.Close()

	// Simulate a torn write: truncate the last few bytes of the file.
	path := filepath.Join(dir, "ingest.wal")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) < 50 {
		t.Skip("WAL too small to truncate meaningfully")
	}
	if err := os.WriteFile(path, body[:len(body)-20], 0o600); err != nil {
		t.Fatal(err)
	}

	w2, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()

	// Replay should error somewhere — but Open should not panic.
	err = w2.Replay(func(ev *events.Event) error { return nil })
	if err == nil {
		// Acceptable if the truncation happened to land at a line boundary.
		t.Log("replay succeeded — truncation aligned to boundary")
	} else if !strings.Contains(err.Error(), "decode") && !strings.Contains(err.Error(), "EOF") {
		t.Errorf("expected decode-style error, got %v", err)
	}

	// New writes after recovery must still work.
	ev := &events.Event{Source: events.SourceREST, HostID: "h", Message: "post-crash"}
	_ = ev.Validate()
	if err := w2.Append(ev); err != nil {
		t.Fatalf("post-crash append failed: %v", err)
	}
}

// TestConcurrentAppend stresses the per-WAL mutex by hammering Append from
// many goroutines and confirms the final on-disk count matches.
func TestConcurrentAppend(t *testing.T) {
	dir := t.TempDir()
	w, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}

	const goroutines = 20
	const perG = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				if ctx.Err() != nil {
					return
				}
				ev := &events.Event{
					Source: events.SourceREST, HostID: "h",
					Message: "stress",
				}
				_ = ev.Validate()
				if err := w.Append(ev); err != nil {
					t.Errorf("worker %d: %v", id, err)
				}
			}
		}(g)
	}
	wg.Wait()

	want := int64(goroutines * perG)
	if got := w.Stats().Count; got != want {
		t.Errorf("count = %d, want %d", got, want)
	}
	_ = w.Close()
}
