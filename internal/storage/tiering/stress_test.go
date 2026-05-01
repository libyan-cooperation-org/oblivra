package tiering

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/storage/hot"
)

// TestMigratorWithConcurrentIngest runs warm-tier migration while another
// goroutine is hammering ingest. The migrator must:
//   - never delete an event whose warm-tier write hasn't fsynced
//   - never produce a half-written Parquet file the verifier can't replay
//   - finish without panicking under contention
func TestMigratorWithConcurrentIngest(t *testing.T) {
	store, err := hot.Open(hot.Options{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(silent{}, nil))
	m, err := New(logger, store, Options{
		WarmDir:    dir,
		MaxAge:     1 * time.Millisecond, // every ingested event is "old enough"
		ResolveAge: nil,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Pre-load a chunk so the first migration has work to do.
	for i := 0; i < 100; i++ {
		ev := &events.Event{Source: events.SourceREST, HostID: "h", Message: "m"}
		_ = ev.Validate()
		if err := store.Put(ev); err != nil {
			t.Fatal(err)
		}
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				ev := &events.Event{Source: events.SourceREST, HostID: "h", Message: "live"}
				_ = ev.Validate()
				_ = store.Put(ev)
			}
		}
	}()

	// Run migrate twice while ingest is racing.
	for i := 0; i < 2; i++ {
		if _, err := m.Run(context.Background()); err != nil {
			close(stop)
			wg.Wait()
			t.Fatalf("migrate failed under load: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	close(stop)
	wg.Wait()

	// Verifier must report ok across whatever was written.
	res, err := m.Verify(50)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Errorf("warm-tier verifier failed: bad=%d events=%d", res.BadEvents, res.EventsSeen)
	}
}

type silent struct{}

func (silent) Write(p []byte) (int, error) { return len(p), nil }
