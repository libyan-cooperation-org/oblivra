//go:build !server

package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// EmitEvent safely wraps the Wails v3 application.Event.Emit call so unit
// tests (which run without a live Wails application instance) don't panic.
// Callers can fire-and-forget events even when application.Get() is nil.
func EmitEvent(eventName string, data interface{}) {
	// Defensively catch Wails panics if given context lacks expected lifecycle flags
	defer func() {
		if r := recover(); r != nil {
			// Do nothing on panic
		}
	}()
	if app := application.Get(); app != nil {
		app.Event.Emit(eventName, data)
	}
}
