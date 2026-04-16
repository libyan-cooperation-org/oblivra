//go:build server

package services

// EmitEvent is a no-op in server mode.
func EmitEvent(eventName string, data interface{}) {
	// Silent drop in server mode
}
