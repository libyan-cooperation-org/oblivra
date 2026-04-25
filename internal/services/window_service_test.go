//go:build !server

package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// TestWorkspace_RoundTrip verifies that SaveWorkspace + RestoreWorkspace
// preserve the route+title for every pop-out, even when window handles
// are nil (the test path doesn't have a real Wails app).
//
// We test the JSON layer in isolation by populating WindowService.popouts
// directly with handle=nil records, then snapshotting via the on-disk
// representation that RestoreWorkspace would later read.
func TestWorkspace_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	prev := workspaceFilePathFn
	workspaceFilePathFn = func() string { return filepath.Join(tmp, "workspace.json") }
	t.Cleanup(func() { workspaceFilePathFn = prev })
	dataDir := tmp

	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: "NUL"})
	svc := NewWindowService(log)

	// Populate three "open" pop-outs without real handles. The geometry
	// probe in SaveWorkspace short-circuits on rec.handle == nil.
	svc.popouts[1] = &popoutRecord{Route: "/siem-search", Title: "SIEM"}
	svc.popouts[2] = &popoutRecord{Route: "/alerts", Title: "Alerts"}
	svc.popouts[3] = &popoutRecord{Route: "/fleet", Title: "Fleet"}

	saved, err := svc.SaveWorkspace()
	if err != nil {
		t.Fatalf("SaveWorkspace: %v", err)
	}
	if saved != 3 {
		t.Errorf("expected 3 pop-outs saved, got %d", saved)
	}

	// File should exist.
	if !svc.HasSavedWorkspace() {
		t.Fatal("HasSavedWorkspace returned false right after a successful save")
	}

	// File should contain the routes.
	wsPath := filepath.Join(dataDir, "workspace.json")
	raw, err := os.ReadFile(wsPath)
	if err != nil {
		t.Fatalf("read workspace.json: %v", err)
	}
	var snap WorkspaceSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("workspace.json is not valid JSON: %v", err)
	}
	if snap.Version != workspaceSchemaVersion {
		t.Errorf("expected schema version %d, got %d", workspaceSchemaVersion, snap.Version)
	}
	if len(snap.Popouts) != 3 {
		t.Errorf("expected 3 entries on disk, got %d", len(snap.Popouts))
	}

	// Routes must round-trip 1:1.
	wantRoutes := map[string]bool{"/siem-search": true, "/alerts": true, "/fleet": true}
	for _, p := range snap.Popouts {
		if !wantRoutes[p.Route] {
			t.Errorf("unexpected route on disk: %q", p.Route)
		}
		delete(wantRoutes, p.Route)
	}
	if len(wantRoutes) > 0 {
		t.Errorf("missing routes on disk: %v", wantRoutes)
	}
}

// TestWorkspace_HasSavedWorkspaceEmpty confirms HasSavedWorkspace returns
// false when no workspace.json has been written yet.
func TestWorkspace_HasSavedWorkspaceEmpty(t *testing.T) {
	tmp := t.TempDir()
	prev := workspaceFilePathFn
	workspaceFilePathFn = func() string { return filepath.Join(tmp, "workspace.json") }
	t.Cleanup(func() { workspaceFilePathFn = prev })

	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: "NUL"})
	svc := NewWindowService(log)

	if svc.HasSavedWorkspace() {
		t.Errorf("HasSavedWorkspace returned true on a fresh data dir")
	}
}

// TestWorkspace_RestoreFromMissing confirms RestoreWorkspace silently
// returns 0 when no save exists — a fresh install must not error out.
func TestWorkspace_RestoreFromMissing(t *testing.T) {
	tmp := t.TempDir()
	prev := workspaceFilePathFn
	workspaceFilePathFn = func() string { return filepath.Join(tmp, "workspace.json") }
	t.Cleanup(func() { workspaceFilePathFn = prev })

	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: "NUL"})
	svc := NewWindowService(log)

	n, err := svc.RestoreWorkspace(false)
	if err != nil {
		t.Errorf("RestoreWorkspace on fresh data dir returned error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 restored on missing file, got %d", n)
	}
}

// TestWorkspace_RestoreRejectsWrongVersion ensures a snapshot from a
// future or unknown schema version is refused with a clear error,
// rather than silently re-opening pop-outs the operator never saved.
func TestWorkspace_RestoreRejectsWrongVersion(t *testing.T) {
	tmp := t.TempDir()
	prev := workspaceFilePathFn
	workspaceFilePathFn = func() string { return filepath.Join(tmp, "workspace.json") }
	t.Cleanup(func() { workspaceFilePathFn = prev })

	bogus := WorkspaceSnapshot{
		Version: workspaceSchemaVersion + 100, // future / unknown
		Popouts: []popoutRecord{{Route: "/should-not-restore"}},
	}
	data, _ := json.Marshal(bogus)
	if err := os.WriteFile(filepath.Join(tmp, "workspace.json"), data, 0600); err != nil {
		t.Fatalf("seed workspace.json: %v", err)
	}

	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: "NUL"})
	svc := NewWindowService(log)

	_, err := svc.RestoreWorkspace(false)
	if err == nil {
		t.Fatal("expected RestoreWorkspace to error on future schema version, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported workspace version") {
		t.Errorf("error message should mention 'unsupported workspace version', got: %v", err)
	}
}
