//go:build !windows

package worm

// On unix the cross-platform Lock() already strips write bits from the file
// mode. The proper `chattr +i` requires root and is delegated to ops scripts
// for now. These platform stubs are no-ops so the cross-platform Lock keeps
// the read-only mode change.

func platformLock(_ string) error { return nil }

func platformUnlock(_ string) error { return nil }

func platformIsLocked(_ string) (bool, error) { return true, nil }
