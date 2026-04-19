//go:build windows

package main

func runMemoryHardening() {
	// Memory locking (VirtualLock) on Windows requires different API calls.
	// We skip this for now to ensure compilation on Windows platforms.
}
