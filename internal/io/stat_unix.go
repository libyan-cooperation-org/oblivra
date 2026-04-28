//go:build !windows

package io

import (
	"os"
	"syscall"
)

// statInode returns the file's inode on Unix. Used by file input to
// detect rotation (when the inode changes, the file was rotated).
func statInode(st os.FileInfo) uint64 {
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		return sys.Ino
	}
	return 0
}
