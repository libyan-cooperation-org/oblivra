package tiering

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// TestWarmTier_TwoDays writes events to two different days and
// asserts both come back via Range. Targets the "-200d events
// disappear after migration" failure mode in the integration test.
func TestWarmTier_TwoDays(t *testing.T) {
	dir := t.TempDir()
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	warm := NewWarmTier(dir, log)
	ctx := context.Background()

	now := time.Now().UTC()
	day1 := now.Add(-200 * 24 * time.Hour)
	day2 := now.Add(-100 * 24 * time.Hour)

	events := []Event{
		{ID: "a1", Timestamp: day1}, {ID: "a2", Timestamp: day1}, {ID: "a3", Timestamp: day1},
		{ID: "o1", Timestamp: day2}, {ID: "o2", Timestamp: day2}, {ID: "o3", Timestamp: day2},
	}
	if _, err := warm.Write(ctx, events); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := map[string]bool{}
	_ = warm.Range(ctx, time.Time{}, time.Time{}, func(e Event) bool {
		got[e.ID] = true
		t.Logf("range hit: %s @ %s", e.ID, e.Timestamp.Format(time.RFC3339))
		return true
	})
	if len(got) != 6 {
		t.Errorf("range hit count: got %d, want 6 (saw: %v)", len(got), got)
	}
}

// TestWarmRange_AncientEvents isolates the warm→cold problem from the
// full migrator test. Writes events at -200d, then queries with the
// 180d cutoff and asserts the events come back.
func TestWarmRange_AncientEvents(t *testing.T) {
	dir := t.TempDir()
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	warm := NewWarmTier(dir, log)
	ctx := context.Background()

	now := time.Now().UTC()
	ancient := now.Add(-200 * 24 * time.Hour)
	events := []Event{
		{ID: "anc-1", Timestamp: ancient, Body: []byte("a")},
		{ID: "anc-2", Timestamp: ancient, Body: []byte("b")},
		{ID: "anc-3", Timestamp: ancient, Body: []byte("c")},
	}
	written, err := warm.Write(ctx, events)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if written != 3 {
		t.Errorf("written: got %d, want 3", written)
	}

	cutoff := now.Add(-180 * 24 * time.Hour)
	t.Logf("ancient=%s cutoff=%s now=%s", ancient.Format(time.RFC3339), cutoff.Format(time.RFC3339), now.Format(time.RFC3339))
	t.Logf("ancient.Before(cutoff) = %v (should be true for these to qualify)", ancient.Before(cutoff))

	got := 0
	if err := warm.Range(ctx, time.Time{}, cutoff, func(e Event) bool {
		got++
		t.Logf("range hit: id=%s ts=%s", e.ID, e.Timestamp.Format(time.RFC3339))
		return true
	}); err != nil {
		t.Fatalf("range: %v", err)
	}
	if got != 3 {
		t.Errorf("expected 3 events in range, got %d", got)
	}
}
