// Package migrate is the schema-migration framework for OBLIVRA's persisted
// artifacts. Today the on-wire/on-disk Event schema is at version 1, so
// migrations are no-ops. This file exists so the *next* schema bump has a
// well-tested upgrade path rather than ad-hoc field-rename scripts.
//
// Migrations are pure functions over a single event line: `Upgrade(line)`
// returns the upgraded line and the post-upgrade schema version. Files are
// migrated by streaming line-by-line with an atomic rename on success.
package migrate

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kingknull/oblivra/internal/events"
)

// Upgrader takes one parsed event and returns the upgraded form. It must be
// idempotent — running it twice should produce the same output as running it
// once.
type Upgrader func(*events.Event) error

// upgraders maps fromVersion → upgrader. The chain runs in order until the
// event reaches events.SchemaVersion.
var upgraders = map[int]Upgrader{
	// Example for when we bump v1 → v2:
	//
	// 1: func(e *events.Event) error { e.NewField = "default"; return nil },
}

// Plan reports which versions an event would step through when migrated.
func Plan(from int) []int {
	steps := []int{}
	v := from
	for v < events.SchemaVersion {
		if _, ok := upgraders[v]; !ok {
			return steps
		}
		steps = append(steps, v+1)
		v++
	}
	return steps
}

// UpgradeEvent walks an event up to events.SchemaVersion and re-seals it.
// Returns the new version and an error if any step in the chain fails.
func UpgradeEvent(e *events.Event) (int, error) {
	if e.SchemaVersion == 0 {
		e.SchemaVersion = 1
	}
	for e.SchemaVersion < events.SchemaVersion {
		up, ok := upgraders[e.SchemaVersion]
		if !ok {
			return e.SchemaVersion, fmt.Errorf("migrate: no upgrader from v%d", e.SchemaVersion)
		}
		if err := up(e); err != nil {
			return e.SchemaVersion, fmt.Errorf("migrate: v%d→v%d: %w", e.SchemaVersion, e.SchemaVersion+1, err)
		}
		e.SchemaVersion++
	}
	// Re-seal hash so VerifyHash() passes on the upgraded event.
	if err := e.Validate(); err != nil {
		return e.SchemaVersion, err
	}
	return e.SchemaVersion, nil
}

// Stats summarises a file migration.
type Stats struct {
	Path        string `json:"path"`
	Lines       int    `json:"lines"`
	Migrated    int    `json:"migrated"`
	UntouchedAt int    `json:"untouchedAt,omitempty"` // current schema, no upgrade needed
}

// File migrates a WAL or event-log file in place. The original is renamed to
// `<path>.pre-migrate` so a corrupted run is recoverable.
func File(path string) (Stats, error) {
	st := Stats{Path: path}
	in, err := os.Open(path)
	if err != nil {
		return st, err
	}
	defer in.Close()

	tmp := path + ".migrating"
	out, err := os.Create(tmp)
	if err != nil {
		return st, err
	}
	bw := bufio.NewWriter(out)

	br := bufio.NewReader(in)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			st.Lines++
			var ev events.Event
			if jerr := json.Unmarshal(line, &ev); jerr != nil {
				_ = out.Close()
				_ = os.Remove(tmp)
				return st, fmt.Errorf("migrate: bad json at line %d: %w", st.Lines, jerr)
			}
			before := ev.SchemaVersion
			if before >= events.SchemaVersion {
				st.UntouchedAt++
			} else {
				if _, uerr := UpgradeEvent(&ev); uerr != nil {
					_ = out.Close()
					_ = os.Remove(tmp)
					return st, uerr
				}
				st.Migrated++
			}
			b, _ := json.Marshal(&ev)
			if _, werr := bw.Write(b); werr != nil {
				_ = out.Close()
				_ = os.Remove(tmp)
				return st, werr
			}
			if _, werr := bw.WriteString("\n"); werr != nil {
				_ = out.Close()
				_ = os.Remove(tmp)
				return st, werr
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			_ = out.Close()
			_ = os.Remove(tmp)
			return st, err
		}
	}
	if err := bw.Flush(); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return st, err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return st, err
	}
	_ = out.Close()

	// If nothing actually moved versions there's no point shuffling files.
	if st.Migrated == 0 {
		_ = os.Remove(tmp)
		return st, nil
	}

	// Atomic-ish swap: keep the original under .pre-migrate so an operator
	// can roll back, then promote the new file.
	backup := path + ".pre-migrate"
	if err := os.Rename(path, backup); err != nil {
		_ = os.Remove(tmp)
		return st, err
	}
	if err := os.Rename(tmp, path); err != nil {
		// Try to restore the original on failure.
		_ = os.Rename(backup, path)
		return st, err
	}
	return st, nil
}

// Dir migrates every *.wal / *.log file under root.
func Dir(root string) ([]Stats, error) {
	out := []Stats{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		switch filepath.Ext(p) {
		case ".wal", ".log":
			s, mErr := File(p)
			out = append(out, s)
			return mErr
		}
		return nil
	})
	return out, err
}
