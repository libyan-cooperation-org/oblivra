//go:build !windows

package tamper

import (
	"os"
	"syscall"
)

// statInode returns the file's Unix inode. The server compares
// successive heartbeats: a changing inode = legitimate rotation,
// a shrinking size at the same inode = log_truncated.
func statInode(st os.FileInfo) uint64 {
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		return sys.Ino
	}
	return 0
}
