package platform

import (
	"os"
	"path/filepath"
)

type Platform interface {
	Name() string
	Arch() string
	ConfigDir() string
	DataDir() string
	LogDir() string
	KeychainAvailable() bool
	TotalMemoryMB() uint64
}

const appName = "sovereign-terminal"

func ConfigDir() string {
	return Detect().ConfigDir()
}

func DataDir() string {
	return Detect().DataDir()
}

func LogPath() string {
	return filepath.Join(Detect().LogDir(), "app.log")
}

func DatabasePath() string {
	return filepath.Join(Detect().DataDir(), "sovereign.db")
}

func VaultPath() string {
	return filepath.Join(Detect().DataDir(), "vault.enc")
}

func EnsureDirectories() error {
	p := Detect()
	dirs := []string{
		p.ConfigDir(),
		p.DataDir(),
		p.LogDir(),
		filepath.Join(p.DataDir(), "sigma"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}

// ValidateSafePath ensures a path is within the allowed application directories or the user's home.
// This is the primary defense against G304/G703 Path Traversal vulnerabilities.
func ValidateSafePath(path string) (string, error) {
	if path == "" {
		return "", os.ErrNotExist
	}
	
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	home, _ := os.UserHomeDir()
	p := Detect()
	
	allowedPrefixes := []string{
		p.ConfigDir(),
		p.DataDir(),
		p.LogDir(),
		home,
	}

	for _, prefix := range allowedPrefixes {
		cleanPrefix := filepath.Clean(prefix)
		if abs == cleanPrefix || (len(abs) > len(cleanPrefix) && abs[len(cleanPrefix)] == filepath.Separator && abs[:len(cleanPrefix)] == cleanPrefix) {
			return abs, nil
		}
	}

	return "", os.ErrPermission
}
