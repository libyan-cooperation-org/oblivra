//go:build !server

package services

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// popoutRecord tracks a pop-out window plus the metadata needed to restore
// it later (workspace persistence). The window handle is short-lived; the
// metadata survives a process restart via WorkspaceFile().
type popoutRecord struct {
	handle *application.WebviewWindow
	Route  string `json:"route"`
	Title  string `json:"title,omitempty"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// WindowService is a Wails-bound service that lets the frontend pop a panel
// out into its own native window — the SOC operator pattern of "drag the
// SIEM search to monitor 2, drag the alerts board to monitor 3, keep the
// terminal on monitor 1." Each pop-out is a real Wails window backed by the
// same Go process and the same in-memory data, so there's zero IPC round
// trip between the panel views.
//
// SOC multi-monitor support + workspace save/restore.
type WindowService struct {
	BaseService
	log *logger.Logger

	mu      sync.Mutex
	popouts map[int64]*popoutRecord // window-id → record
	nextID  atomic.Int64
}

func NewWindowService(log *logger.Logger) *WindowService {
	return &WindowService{
		log:     log.WithPrefix("window"),
		popouts: make(map[int64]*popoutRecord),
	}
}

func (s *WindowService) Name() string { return "window-service" }

// PopOut spawns a new Wails window pointed at the given route. Always
// creates a new window — multiple pop-outs of the same route are a valid
// SOC pattern (e.g. one filtered to host A on monitor 2, another filtered
// to host B on monitor 3).
//
// The new window starts at "/?popout=1&route=<route>"; App.svelte detects
// the popout query param to hide the sidebar and render only the requested
// route.
//
// Returns the integer window ID, which the caller can later pass to
// FocusPopout(id) or ClosePopout(id).
func (s *WindowService) PopOut(route string, title string) (int64, error) {
	if route == "" {
		return 0, fmt.Errorf("PopOut: route is required")
	}
	// Defence against arbitrary URL injection: only allow known-shape paths.
	if !strings.HasPrefix(route, "/") || strings.ContainsAny(route, " \t\r\n#?") {
		return 0, fmt.Errorf("PopOut: invalid route %q", route)
	}

	app := application.Get()
	if app == nil {
		return 0, fmt.Errorf("PopOut: Wails application not initialised")
	}

	if title == "" {
		title = "OBLIVRA " + strings.TrimPrefix(route, "/")
	}

	startURL := "/?popout=1&route=" + url.QueryEscape(route)

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            title,
		URL:              startURL,
		Width:            1024,
		Height:           700,
		MinWidth:         640,
		MinHeight:        420,
		Frameless:        true,
		BackgroundColour: application.NewRGBA(13, 17, 23, 255),
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 28,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		Windows: application.WindowsWindow{
			BackdropType: application.Mica,
		},
	})

	id := s.nextID.Add(1)
	s.mu.Lock()
	s.popouts[id] = &popoutRecord{
		handle: win,
		Route:  route,
		Title:  title,
	}
	s.mu.Unlock()
	s.log.Info("popped out route=%s as window id=%d title=%q", route, id, title)
	return id, nil
}

// ClosePopout closes a previously-opened pop-out window by ID. Idempotent —
// returns nil if the ID was already closed/unknown.
func (s *WindowService) ClosePopout(id int64) error {
	s.mu.Lock()
	rec, ok := s.popouts[id]
	delete(s.popouts, id)
	s.mu.Unlock()
	if !ok || rec == nil || rec.handle == nil {
		return nil
	}
	rec.handle.Close()
	return nil
}

// CloseAllPopouts is a convenience for "reset my workspace" — closes every
// pop-out without touching the main window. Useful when an operator has 7
// windows scattered across 4 monitors and wants to start over.
func (s *WindowService) CloseAllPopouts() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	closed := 0
	for id, rec := range s.popouts {
		if rec != nil && rec.handle != nil {
			rec.handle.Close()
			closed++
		}
		delete(s.popouts, id)
	}
	if closed > 0 {
		s.log.Info("closed %d pop-outs", closed)
	}
	return closed
}

// ListPopouts returns IDs of currently-tracked pop-outs. Stale entries
// (windows the user closed via OS chrome) may persist until ClosePopout
// is called explicitly — Wails v3 doesn't expose a portable "is this
// window still alive" check we trust enough to use here.
func (s *WindowService) ListPopouts() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]int64, 0, len(s.popouts))
	for id := range s.popouts {
		out = append(out, id)
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// Workspace save / restore
//
// Captures the route + position + size of every open pop-out and the main
// window, persists to <DataDir>/workspace.json, and restores them on the
// next launch (or on demand via the menu's "Restore Workspace" item).
// ─────────────────────────────────────────────────────────────────────────────

// WorkspaceSnapshot is the on-disk layout — versioned so future schema
// changes can detect and migrate old files.
type WorkspaceSnapshot struct {
	Version int            `json:"version"`
	Popouts []popoutRecord `json:"popouts"`
}

const workspaceSchemaVersion = 1

func workspaceFilePath() string {
	return filepath.Join(platform.DataDir(), "workspace.json")
}

// SaveWorkspace captures every currently-open pop-out's route + geometry
// to <DataDir>/workspace.json. Returns the number of pop-outs saved.
//
// Geometry is best-effort: if a window's Position()/Size() call panics or
// returns zero (some Wails platforms haven't initialised the surface yet),
// we fall back to defaults — the route is the important bit.
func (s *WindowService) SaveWorkspace() (int, error) {
	s.mu.Lock()
	snapshot := WorkspaceSnapshot{Version: workspaceSchemaVersion}
	for _, rec := range s.popouts {
		if rec == nil {
			continue
		}
		entry := popoutRecord{
			Route: rec.Route,
			Title: rec.Title,
		}
		if rec.handle != nil {
			// Wails v3 panics defensively if a window is in an odd state
			// during shutdown; guard against that so SaveWorkspace never
			// kills the host process.
			func() {
				defer func() {
					if r := recover(); r != nil {
						s.log.Warn("SaveWorkspace: window geometry probe panicked: %v", r)
					}
				}()
				if x, y := rec.handle.Position(); x != 0 || y != 0 {
					entry.X, entry.Y = x, y
				}
				if w, h := rec.handle.Size(); w > 0 && h > 0 {
					entry.Width, entry.Height = w, h
				}
			}()
		}
		snapshot.Popouts = append(snapshot.Popouts, entry)
	}
	s.mu.Unlock()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("marshal workspace: %w", err)
	}
	path := workspaceFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return 0, fmt.Errorf("workspace dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return 0, fmt.Errorf("write workspace temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return 0, fmt.Errorf("rename workspace: %w", err)
	}

	s.log.Info("saved workspace with %d pop-out(s) to %s", len(snapshot.Popouts), path)
	return len(snapshot.Popouts), nil
}

// RestoreWorkspace re-opens the pop-outs captured by the most recent
// SaveWorkspace. Idempotent in the sense that calling it twice produces
// duplicate windows — by design, since multiple pop-outs of the same
// route is a valid SOC pattern. Closes existing pop-outs first if
// `closeExisting` is true, so the operator can use it as a "reset to my
// saved layout" action.
//
// Returns the number of pop-outs restored.
func (s *WindowService) RestoreWorkspace(closeExisting bool) (int, error) {
	data, err := os.ReadFile(workspaceFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // nothing saved — no-op, not an error
		}
		return 0, fmt.Errorf("read workspace: %w", err)
	}
	var snap WorkspaceSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return 0, fmt.Errorf("parse workspace: %w", err)
	}
	if snap.Version != workspaceSchemaVersion {
		return 0, fmt.Errorf("unsupported workspace version %d (this build expects %d)",
			snap.Version, workspaceSchemaVersion)
	}

	if closeExisting {
		s.CloseAllPopouts()
	}

	restored := 0
	for _, entry := range snap.Popouts {
		id, err := s.PopOut(entry.Route, entry.Title)
		if err != nil {
			s.log.Warn("RestoreWorkspace: skipping %q: %v", entry.Route, err)
			continue
		}
		// Re-apply geometry if we have it. SetPosition/SetSize are no-ops
		// when the values are zero or invalid for the current screen layout.
		s.mu.Lock()
		rec := s.popouts[id]
		s.mu.Unlock()
		if rec != nil && rec.handle != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						s.log.Warn("RestoreWorkspace: geometry restore panicked: %v", r)
					}
				}()
				if entry.X != 0 || entry.Y != 0 {
					rec.handle.SetPosition(entry.X, entry.Y)
				}
				if entry.Width > 0 && entry.Height > 0 {
					rec.handle.SetSize(entry.Width, entry.Height)
				}
			}()
		}
		restored++
	}
	s.log.Info("restored %d pop-out(s) from workspace.json", restored)
	return restored, nil
}

// HasSavedWorkspace returns true when a workspace.json exists. Used by the
// frontend to decide whether to show "Restore Workspace?" prompts on cold
// boot.
func (s *WindowService) HasSavedWorkspace() bool {
	_, err := os.Stat(workspaceFilePath())
	return err == nil
}
