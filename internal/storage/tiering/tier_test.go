package tiering

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

func newTestLog(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	if err != nil {
		t.Fatalf("logger.New: %v", err)
	}
	return log
}

// TestMigrator_HotToWarm: events older than HotDuration get promoted.
// Events younger than the threshold stay put.
func TestMigrator_HotToWarm(t *testing.T) {
	hot := NewMemoryTier(TierHot)
	warm := NewMemoryTier(TierWarm)

	now := time.Now()
	// 5 events: 2 old (60d ago — should migrate), 3 new (1d ago — stay).
	for i := 0; i < 2; i++ {
		_, _ = hot.Write(context.Background(), []Event{{
			ID:        fmt.Sprintf("old-%d", i),
			Timestamp: now.Add(-60 * 24 * time.Hour),
			Body:      []byte("ancient"),
		}})
	}
	for i := 0; i < 3; i++ {
		_, _ = hot.Write(context.Background(), []Event{{
			ID:        fmt.Sprintf("new-%d", i),
			Timestamp: now.Add(-24 * time.Hour),
			Body:      []byte("recent"),
		}})
	}

	m := NewMigrator(hot, warm, nil, DefaultRetention(), newTestLog(t))
	stats := m.RunOnce(context.Background())

	if stats.HotToWarm != 2 {
		t.Errorf("HotToWarm: got %d, want 2", stats.HotToWarm)
	}
	if hot.Count() != 3 {
		t.Errorf("hot remaining: got %d, want 3", hot.Count())
	}
	if warm.Count() != 2 {
		t.Errorf("warm count: got %d, want 2", warm.Count())
	}
	if len(stats.Errors) > 0 {
		t.Errorf("expected no errors, got %v", stats.Errors)
	}
}

// TestMigrator_WarmToCold: events older than HotDuration+WarmDuration
// get promoted from warm to cold.
func TestMigrator_WarmToCold(t *testing.T) {
	hot := NewMemoryTier(TierHot)
	warm := NewMemoryTier(TierWarm)
	cold := NewMemoryTier(TierCold)

	now := time.Now()
	// Seed warm with one ancient event (200d) and one merely-old (100d).
	_, _ = warm.Write(context.Background(), []Event{
		{ID: "ancient", Timestamp: now.Add(-200 * 24 * time.Hour)},
		{ID: "old",     Timestamp: now.Add(-100 * 24 * time.Hour)},
	})

	m := NewMigrator(hot, warm, cold, DefaultRetention(), newTestLog(t))
	stats := m.RunOnce(context.Background())

	if stats.WarmToCold != 1 {
		t.Errorf("WarmToCold: got %d, want 1", stats.WarmToCold)
	}
	if warm.Count() != 1 {
		t.Errorf("warm remaining: got %d, want 1", warm.Count())
	}
	if cold.Count() != 1 {
		t.Errorf("cold count: got %d, want 1", cold.Count())
	}
}

// TestMigrator_BatchBudget: the migrator stops at BatchSize per cycle.
func TestMigrator_BatchBudget(t *testing.T) {
	hot := NewMemoryTier(TierHot)
	warm := NewMemoryTier(TierWarm)

	// 100 ancient events.
	now := time.Now()
	for i := 0; i < 100; i++ {
		_, _ = hot.Write(context.Background(), []Event{{
			ID:        fmt.Sprintf("e-%d", i),
			Timestamp: now.Add(-60 * 24 * time.Hour),
		}})
	}

	m := NewMigrator(hot, warm, nil, DefaultRetention(), newTestLog(t))
	m.BatchSize = 10

	stats := m.RunOnce(context.Background())
	if stats.HotToWarm != 10 {
		t.Errorf("expected exactly 10 migrated per cycle, got %d", stats.HotToWarm)
	}
	if hot.Count() != 90 {
		t.Errorf("hot remaining: got %d, want 90", hot.Count())
	}
}

// TestMigrator_PartialTierGraph: nil cold tier means Hot↔Warm only,
// no errors.
func TestMigrator_PartialTierGraph(t *testing.T) {
	hot := NewMemoryTier(TierHot)
	warm := NewMemoryTier(TierWarm)

	now := time.Now()
	for i := 0; i < 5; i++ {
		_, _ = hot.Write(context.Background(), []Event{{
			ID:        fmt.Sprintf("e-%d", i),
			Timestamp: now.Add(-60 * 24 * time.Hour),
		}})
	}

	m := NewMigrator(hot, warm, nil /* no cold */, DefaultRetention(), newTestLog(t))
	stats := m.RunOnce(context.Background())

	if stats.HotToWarm != 5 {
		t.Errorf("expected 5 hot→warm migrations, got %d", stats.HotToWarm)
	}
	if stats.WarmToCold != 0 {
		t.Errorf("expected 0 warm→cold (no cold tier), got %d", stats.WarmToCold)
	}
	if len(stats.Errors) > 0 {
		t.Errorf("nil cold tier should not produce errors, got %v", stats.Errors)
	}
}

// TestMigrator_StartStop: confirm the background loop launches +
// stops cleanly without leaking goroutines.
func TestMigrator_StartStop(t *testing.T) {
	hot := NewMemoryTier(TierHot)
	warm := NewMemoryTier(TierWarm)

	m := NewMigrator(hot, warm, nil, DefaultRetention(), newTestLog(t))
	m.Interval = 50 * time.Millisecond

	ctx := context.Background()
	m.Start(ctx)
	// Second Start is a no-op.
	m.Start(ctx)

	time.Sleep(120 * time.Millisecond)
	m.Stop()
	// Second Stop is a no-op.
	m.Stop()
}
