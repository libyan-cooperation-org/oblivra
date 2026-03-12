//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

func Detect() Platform { return &LinuxPlatform{} }

type LinuxPlatform struct{}

func (p *LinuxPlatform) Name() string { return "linux" }
func (p *LinuxPlatform) Arch() string { return runtime.GOARCH }

func (p *LinuxPlatform) ConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", appName)
}

func (p *LinuxPlatform) DataDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", appName)
}

func (p *LinuxPlatform) LogDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, appName, "logs")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", appName, "logs")
}

func (p *LinuxPlatform) KeychainAvailable() bool {
	_, err := os.Stat("/usr/bin/secret-tool")
	return err == nil
}

func (p *LinuxPlatform) TotalMemoryMB() uint64 {
	// Baseline for Linux hosts — 2GB safe minimum for modern distros.
	// Future: read from /proc/meminfo.
	return 2048
}
