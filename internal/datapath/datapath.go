// Package datapath resolves OBLIVRA's on-disk data directory in a way that
// matches the README's documented layout per OS.
package datapath

import (
	"os"
	"path/filepath"
	"runtime"
)

const appDir = "oblivra"

// Resolve returns the data directory, honouring OBLIVRA_DATA_DIR when set.
// Falls back to OS-appropriate defaults:
//
//	Windows: %LOCALAPPDATA%/oblivra
//	macOS:   ~/Library/Application Support/oblivra
//	Linux:   ~/.local/share/oblivra
func Resolve() (string, error) {
	if v := os.Getenv("OBLIVRA_DATA_DIR"); v != "" {
		if err := os.MkdirAll(v, 0o755); err != nil {
			return "", err
		}
		return v, nil
	}

	var base string
	switch runtime.GOOS {
	case "windows":
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Local")
		}
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, "Library", "Application Support")
	default:
		if v := os.Getenv("XDG_DATA_HOME"); v != "" {
			base = v
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, ".local", "share")
		}
	}

	dir := filepath.Join(base, appDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// Sub creates and returns dir/sub.
func Sub(dir, sub string) (string, error) {
	p := filepath.Join(dir, sub)
	if err := os.MkdirAll(p, 0o755); err != nil {
		return "", err
	}
	return p, nil
}
