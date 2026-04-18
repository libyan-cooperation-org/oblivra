//go:build !server

package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// EmitEvent safely wraps wails runtime.EventsEmit to avoid test panics
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
