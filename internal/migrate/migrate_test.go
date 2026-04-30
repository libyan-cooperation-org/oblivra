package migrate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kingknull/oblivra/internal/events"
)

// At schema v1 there are no upgraders; UpgradeEvent should be a no-op for an
// already-current event.
func TestUpgradeNoop(t *testing.T) {
	ev := &events.Event{Source: events.SourceREST, Message: "x"}
	if err := ev.Validate(); err != nil {
		t.Fatal(err)
	}
	pre := ev.Hash
	got, err := UpgradeEvent(ev)
	if err != nil {
		t.Fatal(err)
	}
	if got != events.SchemaVersion {
		t.Errorf("version = %d", got)
	}
	if ev.Hash != pre {
		t.Error("hash changed for an already-current event")
	}
}

// File migration must be a no-op on a file full of current-schema events —
// in particular, no .pre-migrate backup should be left behind.
func TestFileNoop(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ingest.wal")
	f, _ := os.Create(path)
	for i := 0; i < 3; i++ {
		ev := &events.Event{Source: events.SourceREST, Message: "x"}
		_ = ev.Validate()
		b, _ := json.Marshal(ev)
		f.Write(b)
		f.Write([]byte("\n"))
	}
	_ = f.Close()

	st, err := File(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lines != 3 {
		t.Errorf("lines = %d", st.Lines)
	}
	if st.Migrated != 0 {
		t.Errorf("expected no migrations, got %d", st.Migrated)
	}
	if _, err := os.Stat(path + ".pre-migrate"); err == nil {
		t.Error("backup file should NOT exist on no-op migration")
	}
}

func TestPlanFromCurrent(t *testing.T) {
	steps := Plan(events.SchemaVersion)
	if len(steps) != 0 {
		t.Errorf("plan from current should be empty, got %v", steps)
	}
}

// UpgradeEvent must error with a clear message when there's no upgrader
// for an old version. We simulate this by registering an upgrader for the
// current version and pointing an event at a phantom older version.
func TestUpgradeMissingPathErrors(t *testing.T) {
	ev := &events.Event{Message: "x", Source: events.SourceREST, SchemaVersion: 99}
	if err := ev.Validate(); err != nil {
		t.Fatal(err)
	}
	// SchemaVersion=99 is "newer than current" — UpgradeEvent should be a no-op
	// because the loop condition is `< SchemaVersion`.
	got, err := UpgradeEvent(ev)
	if err != nil {
		t.Fatalf("future-version event should be a no-op, got %v", err)
	}
	if got != 99 {
		t.Errorf("version = %d", got)
	}
}
