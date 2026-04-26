package tiering

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/storage"
)

// newTestStore opens a fresh BadgerDB in t.TempDir() and registers
// cleanup so the goroutine-leak gate doesn't fire.
func newTestStore(t *testing.T) *storage.HotStore {
	t.Helper()
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	store, err := storage.NewHotStore(t.TempDir(), log)
	if err != nil {
		t.Fatalf("NewHotStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// TestHotTier_RoundTrip verifies the encode-write-range-decode cycle.
func TestHotTier_RoundTrip(t *testing.T) {
	hot := NewHotTier(newTestStore(t))
	ctx := context.Background()

	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	events := []Event{
		{ID: "e1", Timestamp: now, Host: "h1", Body: []byte("first")},
		{ID: "e2", Timestamp: now.Add(time.Minute), Host: "h2", Body: []byte("second")},
		{ID: "e3", Timestamp: now.Add(2 * time.Minute), Host: "h3", Body: []byte("third")},
	}
	written, err := hot.Write(ctx, events)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if written != 3 {
		t.Errorf("written: got %d, want 3", written)
	}

	// Range full window.
	var got []Event
	if err := hot.Range(ctx, time.Time{}, time.Time{}, func(e Event) bool {
		got = append(got, e)
		return true
	}); err != nil {
		t.Fatalf("range: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("range count: got %d, want 3", len(got))
	}
	// Confirm chronological order (HotTier's key encoding guarantees this).
	for i := 1; i < len(got); i++ {
		if got[i].Timestamp.Before(got[i-1].Timestamp) {
			t.Errorf("range not chronological at i=%d", i)
		}
	}
}

// TestHotTier_RangeFiltering: Range honours from/to bounds.
func TestHotTier_RangeFiltering(t *testing.T) {
	hot := NewHotTier(newTestStore(t))
	ctx := context.Background()
	base := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	// 10 events at 1-hour intervals.
	for i := 0; i < 10; i++ {
		_, _ = hot.Write(ctx, []Event{{
			ID: fmt.Sprintf("e-%d", i), Timestamp: base.Add(time.Duration(i) * time.Hour),
		}})
	}
	from := base.Add(3 * time.Hour)
	to := base.Add(7 * time.Hour)
	count := 0
	_ = hot.Range(ctx, from, to, func(Event) bool {
		count++
		return true
	})
	// Events at hours 3,4,5,6,7 = 5 events.
	if count != 5 {
		t.Errorf("filtered range: got %d, want 5", count)
	}
}

// TestHotTier_RangeEarlyStop: returning false from the visitor stops iteration.
func TestHotTier_RangeEarlyStop(t *testing.T) {
	hot := NewHotTier(newTestStore(t))
	ctx := context.Background()
	base := time.Now().UTC()
	for i := 0; i < 100; i++ {
		_, _ = hot.Write(ctx, []Event{{
			ID: fmt.Sprintf("e-%d", i), Timestamp: base.Add(time.Duration(i) * time.Second),
		}})
	}
	count := 0
	_ = hot.Range(ctx, time.Time{}, time.Time{}, func(Event) bool {
		count++
		return count < 10
	})
	if count != 10 {
		t.Errorf("early stop: got %d, want 10", count)
	}
}

// TestHotTier_Delete: events deleted by ID disappear from subsequent
// Range calls.
func TestHotTier_Delete(t *testing.T) {
	hot := NewHotTier(newTestStore(t))
	ctx := context.Background()
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		_, _ = hot.Write(ctx, []Event{{
			ID: fmt.Sprintf("e-%d", i), Timestamp: now.Add(time.Duration(i) * time.Minute),
		}})
	}
	if err := hot.Delete(ctx, []string{"e-1", "e-3"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	count := 0
	_ = hot.Range(ctx, time.Time{}, time.Time{}, func(e Event) bool {
		count++
		return true
	})
	if count != 3 {
		t.Errorf("after delete: got %d, want 3", count)
	}
}

// TestWarmTier_RoundTrip writes events to Parquet and reads them back.
func TestWarmTier_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	warm := NewWarmTier(dir, log)
	ctx := context.Background()

	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	events := []Event{
		{ID: "w1", Timestamp: now, Host: "h-east", TenantID: "t1", Body: []byte("alpha")},
		{ID: "w2", Timestamp: now.Add(time.Hour), Host: "h-west", TenantID: "t1", Body: []byte("beta")},
	}
	written, err := warm.Write(ctx, events)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if written != 2 {
		t.Errorf("written: got %d, want 2", written)
	}

	var got []Event
	_ = warm.Range(ctx, now.Add(-time.Hour), now.Add(2*time.Hour), func(e Event) bool {
		got = append(got, e)
		return true
	})
	if len(got) != 2 {
		t.Errorf("range count: got %d, want 2", len(got))
	}
	// The envelope must round-trip the ID + body.
	for _, e := range got {
		if e.ID == "" {
			t.Errorf("event lost ID through warm round-trip: %+v", e)
		}
	}
}

// TestColdTier_RoundTrip writes events to JSONL and reads them back.
func TestColdTier_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cold := NewLocalDirCold(dir, nil)
	ctx := context.Background()

	now := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	events := []Event{
		{ID: "c1", Timestamp: now, Host: "old-host", TenantID: "t1", Body: []byte("ancient1")},
		{ID: "c2", Timestamp: now.Add(time.Hour), Host: "old-host", TenantID: "t2", Body: []byte("ancient2")},
	}
	written, err := cold.Write(ctx, events)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if written != 2 {
		t.Errorf("written: got %d, want 2", written)
	}

	var got []Event
	_ = cold.Range(ctx, now.Add(-time.Hour), now.Add(2*time.Hour), func(e Event) bool {
		got = append(got, e)
		return true
	})
	if len(got) != 2 {
		t.Errorf("range count: got %d, want 2", len(got))
	}
}

// TestColdTier_PerTenantSplit: events with different tenants land in
// different files (so a GDPR delete-tenant scan only touches one).
func TestColdTier_PerTenantSplit(t *testing.T) {
	dir := t.TempDir()
	cold := NewLocalDirCold(dir, nil)
	ctx := context.Background()
	now := time.Now().UTC()

	_, _ = cold.Write(ctx, []Event{
		{ID: "x", Timestamp: now, TenantID: "tenant-a"},
		{ID: "y", Timestamp: now, TenantID: "tenant-b"},
	})
	// We expect exactly two files under cold/<day>/.
	day := now.UTC().Format("2006/01/02")
	d, err := os.ReadDir(fmt.Sprintf("%s/cold/%s", dir, day))
	if err != nil {
		t.Fatalf("read cold dir: %v", err)
	}
	if len(d) != 2 {
		t.Errorf("expected 2 per-tenant files, got %d", len(d))
	}
}

// TestColdTier_Delete removes specific IDs from the JSONL file.
func TestColdTier_Delete(t *testing.T) {
	dir := t.TempDir()
	cold := NewLocalDirCold(dir, nil)
	ctx := context.Background()
	now := time.Now().UTC()
	_, _ = cold.Write(ctx, []Event{
		{ID: "k1", Timestamp: now}, {ID: "k2", Timestamp: now},
		{ID: "k3", Timestamp: now}, {ID: "k4", Timestamp: now},
	})
	if err := cold.Delete(ctx, []string{"k2", "k4"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got := 0
	_ = cold.Range(ctx, time.Time{}, time.Time{}, func(Event) bool {
		got++
		return true
	})
	if got != 2 {
		t.Errorf("after delete: got %d, want 2", got)
	}
}

// TestMigrator_FullStack drives Hot → Warm → Cold through real
// adapters to prove the pipeline works end-to-end.
func TestMigrator_FullStack(t *testing.T) {
	dir := t.TempDir()
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})

	hot := NewHotTier(newTestStore(t))
	warm := NewWarmTier(dir, log)
	cold := NewLocalDirCold(dir, log)

	ctx := context.Background()
	now := time.Now().UTC()
	// 3 ancient (200d), 3 old (100d), 3 fresh (1d) — should land cold/warm/hot respectively.
	seed := []Event{
		{ID: "a1", Timestamp: now.Add(-200 * 24 * time.Hour), Body: []byte("anc")},
		{ID: "a2", Timestamp: now.Add(-200 * 24 * time.Hour), Body: []byte("anc")},
		{ID: "a3", Timestamp: now.Add(-200 * 24 * time.Hour), Body: []byte("anc")},
		{ID: "o1", Timestamp: now.Add(-100 * 24 * time.Hour), Body: []byte("old")},
		{ID: "o2", Timestamp: now.Add(-100 * 24 * time.Hour), Body: []byte("old")},
		{ID: "o3", Timestamp: now.Add(-100 * 24 * time.Hour), Body: []byte("old")},
		{ID: "f1", Timestamp: now.Add(-24 * time.Hour), Body: []byte("fresh")},
		{ID: "f2", Timestamp: now.Add(-24 * time.Hour), Body: []byte("fresh")},
		{ID: "f3", Timestamp: now.Add(-24 * time.Hour), Body: []byte("fresh")},
	}
	if _, err := hot.Write(ctx, seed); err != nil {
		t.Fatalf("seed hot: %v", err)
	}

	m := NewMigrator(hot, warm, cold, DefaultRetention(), log)
	// Cycle 1 does BOTH stages in sequence:
	//   hot → warm: promotes everything ≥30d from hot.
	//                That's 6 events (3 at -100d, 3 at -200d).
	//   warm → cold: promotes everything ≥(30d+150d)=180d from warm.
	//                Right after the previous step, the 3 events at
	//                -200d are now in warm AND older than 180d, so
	//                they get promoted again to cold within the same
	//                cycle. The 3 events at -100d stay in warm.
	stats := m.RunOnce(ctx)
	if stats.HotToWarm < 6 {
		t.Errorf("HotToWarm: got %d, want ≥6 (3 old + 3 ancient should leave hot)", stats.HotToWarm)
	}
	if stats.WarmToCold < 3 {
		t.Errorf("WarmToCold (cycle 1): got %d, want ≥3 (-200d events should land in cold same cycle)", stats.WarmToCold)
	}

	// Final tier accounting.
	hotCount := 0
	_ = hot.Range(ctx, time.Time{}, time.Time{}, func(Event) bool {
		hotCount++
		return true
	})
	if hotCount != 3 {
		t.Errorf("hot final: got %d, want 3 (the fresh ones)", hotCount)
	}

	coldCount := 0
	_ = cold.Range(ctx, time.Time{}, time.Time{}, func(Event) bool {
		coldCount++
		return true
	})
	if coldCount < 3 {
		t.Errorf("cold final: got %d, want ≥3 (the ancient ones)", coldCount)
	}
}
