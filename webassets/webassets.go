// Package webassets exposes the compiled Svelte frontend so that both the
// Wails desktop shell and the headless HTTP server can serve the same UI.
package webassets

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var raw embed.FS

// FS returns the embedded frontend rooted at the dist directory.
func FS() (fs.FS, error) {
	return fs.Sub(raw, "dist")
}

// Raw returns the underlying embed.FS (used by Wails AssetFileServerFS which
// expects the embed.FS itself with its original prefix).
func Raw() embed.FS {
	return raw
}
