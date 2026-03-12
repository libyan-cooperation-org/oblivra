//go:build !windows

package vault

// secureLock is a no-op on non-Windows platforms in this specific implementation.
// In a full cross-platform app, this would use unix.Mlock.
func secureLock(data []byte) error {
	return nil
}

// secureUnlock is a no-op on non-Windows platforms.
func secureUnlock(data []byte) error {
	return nil
}
