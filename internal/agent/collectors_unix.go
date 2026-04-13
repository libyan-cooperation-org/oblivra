//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package agent

import "syscall"

// statfsDiskUsage returns (free, total) bytes for the given path using Statfs.
func statfsDiskUsage(path string) (free, total uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	// Bavail = blocks available to unprivileged processes
	free = stat.Bavail * uint64(stat.Bsize)
	total = stat.Blocks * uint64(stat.Bsize)
	return
}
