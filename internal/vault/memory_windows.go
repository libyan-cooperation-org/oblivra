//go:build windows

package vault

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// secureLock locks the memory into the process's working set, preventing it
// from being paged to the swap file.
func secureLock(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	
	// Get the pointer to the underlying array
	ptr := unsafe.Pointer(&data[0])
	size := uintptr(len(data))
	
	return windows.VirtualLock(uintptr(ptr), size)
}

// secureUnlock unlocks the memory from the process's working set.
func secureUnlock(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	
	ptr := unsafe.Pointer(&data[0])
	size := uintptr(len(data))
	
	return windows.VirtualUnlock(uintptr(ptr), size)
}
