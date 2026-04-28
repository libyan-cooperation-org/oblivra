//go:build !windows

package services

import "golang.org/x/sys/unix"

// diskFreeBytes returns the number of free bytes available on the
// filesystem containing `path`. Uses statfs(2) directly so the result
// reflects what the kernel sees (matches `df -B1 path`'s "Available"
// column for the running user, including reserved-blocks reduction).
//
// Returns 0 if the path can't be statted — caller treats 0 as
// "unreadable" and skips the kill-switch trigger.
func diskFreeBytes(path string) (int64, error) {
	if path == "" {
		return 0, nil
	}
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// Bavail (available to non-root) × Bsize (block size).
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}
