// Package worm makes a file as immutable as the host OS will allow.
//
// On Windows we set the read-only attribute. On Linux a future build-tagged
// implementation will call `chattr +i` (root-only); without root we degrade
// to read-only mode bits, which is the same posture as macOS. The function
// is best-effort — call it after fsync of a finalized warm-tier file.
package worm

import (
	"errors"
	"os"
)

// Lock makes path read-only at the highest privilege level the runtime can
// reach without escalating. Returns nil on success, or a wrapped error.
func Lock(path string) error {
	if path == "" {
		return errors.New("worm: empty path")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	// Strip write bits from the mode. This is the cross-platform floor; the
	// Linux/Windows-specific files do additional work.
	mode := info.Mode().Perm() &^ 0o222
	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	return platformLock(path)
}

// IsLocked reports whether the file appears to be WORM-locked according to
// the same set of checks Lock applies.
func IsLocked(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.Mode().Perm()&0o222 != 0 {
		return false, nil
	}
	return platformIsLocked(path)
}

// Unlock reverts the OS-level immutability flags so a file can be deleted —
// kept available for retention purges. Operators are responsible for the
// audit trail of unlocks (the audit chain records every storage.unlock).
func Unlock(path string) error {
	if err := platformUnlock(path); err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.Chmod(path, info.Mode().Perm()|0o200)
}
