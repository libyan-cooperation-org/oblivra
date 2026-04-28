//go:build windows

package services

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// diskFreeBytes returns the number of free bytes available on the
// volume containing `path` for the calling user (respects per-user
// disk quotas, matches what File Explorer shows in the volume's
// properties dialog).
//
// Implementation calls GetDiskFreeSpaceExW directly; the third
// argument it returns is `lpFreeBytesAvailableToCaller` which is
// what we want for kill-switch purposes (NOT the total free bytes
// system-wide, which would be misleading on quota-bound volumes).
func diskFreeBytes(path string) (int64, error) {
	if path == "" {
		return 0, nil
	}
	utf16Path, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var freeAvail, totalBytes, totalFreeBytes uint64
	err = windows.GetDiskFreeSpaceEx(
		utf16Path,
		(*uint64)(unsafe.Pointer(&freeAvail)),
		(*uint64)(unsafe.Pointer(&totalBytes)),
		(*uint64)(unsafe.Pointer(&totalFreeBytes)),
	)
	if err != nil {
		return 0, err
	}
	if freeAvail > uint64(1)<<62 {
		// guard against improbable overflow
		return int64(1) << 62, nil
	}
	return int64(freeAvail), nil
}
