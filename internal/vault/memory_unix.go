//go:build !windows

package vault

import (
	"golang.org/x/sys/unix"
)

// secureLock uses unix.Mlock to prevent memory from being swapped to disk.
func secureLock(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return unix.Mlock(data)
}

// secureUnlock uses unix.Munlock to release the memory lock.
func secureUnlock(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return unix.Munlock(data)
}
