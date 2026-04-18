//go:build windows
// +build windows

package agent

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// statfsDiskUsage returns (free, total) bytes for the given path using
// GetDiskFreeSpaceExW on Windows.
func statfsDiskUsage(path string) (free, total uint64) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0
	}
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	err = windows.GetDiskFreeSpaceEx(
		pathPtr,
		(*uint64)(unsafe.Pointer(&freeBytesAvailable)),
		(*uint64)(unsafe.Pointer(&totalNumberOfBytes)),
		(*uint64)(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	if err != nil {
		return 0, 0
	}
	return freeBytesAvailable, totalNumberOfBytes
}
