//go:build darwin

package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

func Detect() Platform { return &DarwinPlatform{} }

type DarwinPlatform struct{}

func (p *DarwinPlatform) Name() string { return "darwin" }
func (p *DarwinPlatform) Arch() string { return runtime.GOARCH }

func (p *DarwinPlatform) ConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", appName)
}

func (p *DarwinPlatform) DataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", appName, "data")
}

func (p *DarwinPlatform) LogDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Logs", appName)
}

func (p *DarwinPlatform) KeychainAvailable() bool { return true }
