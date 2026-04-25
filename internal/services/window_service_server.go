//go:build server

package services

import (
	"github.com/kingknull/oblivrashell/internal/logger"
)

// WindowService stub for the headless server build. The headless mode has
// no Wails application instance, so pop-out is a no-op. Keeping the type
// surface identical to the desktop build so internal/app/app.go can wire
// the service unconditionally.
type WindowService struct {
	BaseService
	log *logger.Logger
}

func NewWindowService(log *logger.Logger) *WindowService {
	return &WindowService{log: log.WithPrefix("window")}
}

func (s *WindowService) Name() string { return "window-service" }

func (s *WindowService) PopOut(route string, title string) (int64, error)     { return 0, nil }
func (s *WindowService) ClosePopout(id int64) error                            { return nil }
func (s *WindowService) CloseAllPopouts() int                                  { return 0 }
func (s *WindowService) ListPopouts() []int64                                  { return nil }
func (s *WindowService) SaveWorkspace() (int, error)                           { return 0, nil }
func (s *WindowService) RestoreWorkspace(closeExisting bool) (int, error)      { return 0, nil }
func (s *WindowService) HasSavedWorkspace() bool                               { return false }
