package tiering

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"

	"github.com/kingknull/oblivra/internal/events"
)

// VerifyResult reports the outcome of a cross-tier integrity check on the
// warm Parquet directory.
type VerifyResult struct {
	WarmDir     string    `json:"warmDir"`
	FilesSeen   int       `json:"filesSeen"`
	EventsSeen  int       `json:"eventsSeen"`
	BadEvents   int       `json:"badEvents"`
	OK          bool      `json:"ok"`
	GeneratedAt time.Time `json:"generatedAt"`
	Notes       []string  `json:"notes,omitempty"`
}

// Verify reads up to N most recent Parquet files in the warm dir, rebuilds
// the event for every row, and re-derives its content hash. Any divergence
// proves the warm tier was tampered with on disk.
//
// We don't store the original `Hash` in the Parquet schema (yet — Phase 39),
// so what this currently checks is "the on-disk row is parseable and the
// fields it carries hash to *something* deterministic." Once `Hash` lands in
// the Parquet schema this becomes a stronger end-to-end identity check.
func (m *Migrator) Verify(maxFiles int) (VerifyResult, error) {
	r := VerifyResult{WarmDir: m.warmDir, GeneratedAt: time.Now().UTC()}
	if maxFiles <= 0 {
		maxFiles = 50
	}
	entries, err := os.ReadDir(m.warmDir)
	if err != nil {
		if os.IsNotExist(err) {
			r.OK = true
			r.Notes = append(r.Notes, "warm dir does not exist yet")
			return r, nil
		}
		return r, err
	}

	type fi struct {
		name string
		mod  time.Time
	}
	var paths []fi
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".parquet") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		paths = append(paths, fi{name: e.Name(), mod: info.ModTime()})
	}
	sort.Slice(paths, func(i, j int) bool { return paths[i].mod.After(paths[j].mod) })
	if len(paths) > maxFiles {
		paths = paths[:maxFiles]
	}

	for _, p := range paths {
		full := filepath.Join(m.warmDir, p.name)
		if err := m.verifyFile(full, &r); err != nil {
			return r, err
		}
	}
	r.OK = r.BadEvents == 0
	return r, nil
}

func (m *Migrator) verifyFile(full string, r *VerifyResult) error {
	f, err := os.Open(full)
	if err != nil {
		return err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	preader, err := parquet.OpenFile(f, stat.Size())
	if err != nil {
		return fmt.Errorf("open parquet %s: %w", filepath.Base(full), err)
	}
	reader := parquet.NewGenericReader[ParquetEvent](preader)
	defer reader.Close()

	buf := make([]ParquetEvent, 256)
	for {
		n, err := reader.Read(buf)
		for i := 0; i < n; i++ {
			row := buf[i]
			ev := events.Event{
				SchemaVersion: row.SchemaVersion,
				ID:            row.ID,
				Hash:          row.Hash,
				TenantID:      row.TenantID,
				Timestamp:     time.Unix(0, row.Timestamp).UTC(),
				ReceivedAt:    time.Unix(0, row.ReceivedAt).UTC(),
				Source:        events.Source(row.Source),
				HostID:        row.HostID,
				EventType:     row.EventType,
				Severity:      events.Severity(row.Severity),
				Message:       row.Message,
				Raw:           row.Raw,
				Provenance: events.Provenance{
					IngestPath: row.IngestPath,
					Peer:       row.Peer,
					AgentID:    row.AgentID,
					Parser:     row.Parser,
				},
			}
			if ev.SchemaVersion == 0 || ev.Hash == "" {
				// v1 row — no embedded hash to verify against. Just confirm
				// structural parse worked.
				r.EventsSeen++
				continue
			}
			if !ev.VerifyHash() {
				r.BadEvents++
			}
			r.EventsSeen++
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read parquet %s: %w", filepath.Base(full), err)
		}
	}
	r.FilesSeen++
	return nil
}
