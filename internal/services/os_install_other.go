//go:build !windows

package services

import (
	"os"
	"time"
)

// osInstallDate on non-Windows systems uses the mtime of /etc/machine-id
// (most modern Linux distros) or /var/db/pkg (BSD) as a proxy for the
// install date. Falls back to 1 year ago if nothing is readable.
func osInstallDate() time.Time {
	candidates := []string{
		"/etc/machine-id",       // systemd Linux
		"/var/lib/dbus/machine-id",
		"/var/db/pkg",           // FreeBSD / pkgng
		"/etc/os-release",       // broad Linux fallback
	}
	for _, p := range candidates {
		info, err := os.Stat(p)
		if err == nil {
			return info.ModTime().UTC()
		}
	}
	return time.Now().UTC().AddDate(-1, 0, 0)
}
