//go:build !windows

package security

import (
	"github.com/kingknull/oblivrashell/internal/logger"
)

// AntiDebugMonitor is a disabled stub for non-Windows platforms.
type AntiDebugMonitor struct {
	log *logger.Logger
}

func NewAntiDebugMonitor(log *logger.Logger) *AntiDebugMonitor {
	return &AntiDebugMonitor{
		log: log,
	}
}

func (m *AntiDebugMonitor) Start() {
	m.log.Warn("Anti-Debug features are not natively enabled on this OS platform.")
}

func (m *AntiDebugMonitor) Stop() {
	// no-op
}
