//go:build !server

package services

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// WindowService is a Wails-bound service that lets the frontend pop a panel
// out into its own native window — the SOC operator pattern of "drag the
// SIEM search to monitor 2, drag the alerts board to monitor 3, keep the
// terminal on monitor 1." Each pop-out is a real Wails window backed by the
// same Go process and the same in-memory data, so there's zero IPC round
// trip between the panel views.
//
// SOC multi-monitor support.
type WindowService struct {
	BaseService
	log *logger.Logger

	mu       sync.Mutex
	popouts  map[int64]*application.WebviewWindow // window-id → window handle
	nextID   atomic.Int64
}

func NewWindowService(log *logger.Logger) *WindowService {
	return &WindowService{
		log:     log.WithPrefix("window"),
		popouts: make(map[int64]*application.WebviewWindow),
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
	s.popouts[id] = win
	s.mu.Unlock()
	s.log.Info("popped out route=%s as window id=%d title=%q", route, id, title)
	return id, nil
}

// ClosePopout closes a previously-opened pop-out window by ID. Idempotent —
// returns nil if the ID was already closed/unknown.
func (s *WindowService) ClosePopout(id int64) error {
	s.mu.Lock()
	win, ok := s.popouts[id]
	delete(s.popouts, id)
	s.mu.Unlock()
	if !ok || win == nil {
		return nil
	}
	win.Close()
	return nil
}

// CloseAllPopouts is a convenience for "reset my workspace" — closes every
// pop-out without touching the main window. Useful when an operator has 7
// windows scattered across 4 monitors and wants to start over.
func (s *WindowService) CloseAllPopouts() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	closed := 0
	for id, win := range s.popouts {
		if win != nil {
			win.Close()
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
