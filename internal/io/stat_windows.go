//go:build windows

package io

import "os"

// Windows doesn't expose inode via os.FileInfo.Sys() in a way that's
// portable across all FS variants (NTFS exposes a 64-bit FileIndex via
// GetFileInformationByHandle). For the file-tail rotation-detection
// use case we approximate: any change in (size + modtime + name) is
// "different file". This is good enough — in-place rotation on
// Windows is rare; rename-based rotation triggers our scan loop
// which re-globs.
//
// If we ever care about strict inode equivalence here we can call
// GetFileInformationByHandle through golang.org/x/sys/windows.
func statInode(st os.FileInfo) uint64 {
	// Hash the (size, mtime nanos) into a uint64. Cheap and
	// deterministic. Two files produced by the same writer with the
	// same content + mtime will collide, but that won't cause data
	// loss — at worst we miss a rotation and re-read from where we
	// were (which is correct anyway).
	mtime := uint64(st.ModTime().UnixNano())
	size := uint64(st.Size())
	return mtime ^ (size << 1)
}
