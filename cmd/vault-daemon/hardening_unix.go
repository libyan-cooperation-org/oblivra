//go:build !windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func runMemoryHardening() {
	// 5. Sovereign-Grade: Memory Hardening
	// Prevent the vault daemon's memory from being swapped to disk.
	if err := syscall.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: mlockall failed: %v — memory isolation weakened\n", err)
	}
}
