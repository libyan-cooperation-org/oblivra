//go:build windows

package services

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// osInstallDate reads HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion!InstallDate
// using the built-in `reg query` command so no extra dependencies are needed.
// Falls back to 1 year ago on any failure.
func osInstallDate() time.Time {
	out, err := exec.Command(
		"reg", "query",
		`HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		"/v", "InstallDate",
	).Output()
	if err != nil {
		return time.Now().UTC().AddDate(-1, 0, 0)
	}
	// Example output line:
	//     InstallDate    REG_DWORD    0x5f3b2c10
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "InstallDate") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			break
		}
		hexStr := strings.TrimPrefix(fields[len(fields)-1], "0x")
		v, err := strconv.ParseInt(hexStr, 16, 64)
		if err != nil {
			break
		}
		return time.Unix(v, 0).UTC()
	}
	return time.Now().UTC().AddDate(-1, 0, 0)
}
