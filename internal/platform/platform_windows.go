//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

func Detect() Platform { return &WindowsPlatform{} }

type WindowsPlatform struct{}

func (p *WindowsPlatform) Name() string { return "windows" }
func (p *WindowsPlatform) Arch() string { return runtime.GOARCH }

func (p *WindowsPlatform) ConfigDir() string {
	return filepath.Join(os.Getenv("APPDATA"), appName)
}

func (p *WindowsPlatform) DataDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), appName, "data")
}

func (p *WindowsPlatform) LogDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), appName, "logs")
}

func (p *WindowsPlatform) KeychainAvailable() bool { return true }

func (p *WindowsPlatform) TotalMemoryMB() uint64 {
	// Baseline for Windows hosts — 4GB is a safe assumption for modern OS.
	// Future: use windows.GlobalMemoryStatusEx to detect actual physical RAM.
	return 4096
}
