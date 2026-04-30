package services

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestDailyAnchor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()
	a, err := NewDurable(logger, dir, []byte("k"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	ctx := context.Background()
	day := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)

	// Inject 3 entries dated within `day` by manipulating the entries slice
	// directly — anchor logic only cares about timestamps.
	a.mu.Lock()
	parent := ""
	for i := 0; i < 3; i++ {
		e := AuditEntry{
			Seq:        int64(len(a.entries) + 1),
			Timestamp:  day.Add(time.Duration(i) * time.Hour),
			Actor:      "x", Action: "test", TenantID: "default",
			ParentHash: parent,
		}
		e.Hash = hashEntry(canonical(e, parent))
		a.entries = append(a.entries, e)
		parent = e.Hash
	}
	a.mu.Unlock()

	entry, isNew, err := a.AnchorDaily(ctx, day)
	if err != nil {
		t.Fatal(err)
	}
	if !isNew {
		t.Fatal("first anchor should be marked new")
	}
	if entry.Action != "audit.daily-anchor" {
		t.Errorf("anchor action = %q", entry.Action)
	}
	if entry.Detail["entries"] != "3" {
		t.Errorf("anchored count = %q (want 3)", entry.Detail["entries"])
	}
	if entry.Detail["root"] == "" {
		t.Error("daily root empty")
	}

	// Second call same day should be a no-op.
	_, isNew2, err := a.AnchorDaily(ctx, day)
	if err != nil {
		t.Fatal(err)
	}
	if isNew2 {
		t.Error("second anchor should be idempotent (not new)")
	}

	// Verify the chain is still good after anchoring.
	if !a.Verify().OK {
		t.Error("chain broke after anchor")
	}
}

func TestDailyAnchorEmptyDay(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(testWriter{}, nil))
	dir := t.TempDir()
	a, err := NewDurable(logger, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	day := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC) // future, no entries
	entry, isNew, err := a.AnchorDaily(context.Background(), day)
	if err != nil {
		t.Fatal(err)
	}
	if isNew {
		t.Error("anchoring an empty day should produce no entry")
	}
	if entry.Hash != "" {
		t.Error("entry should be zero-value when no anchor written")
	}
}
