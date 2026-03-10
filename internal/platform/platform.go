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
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}
