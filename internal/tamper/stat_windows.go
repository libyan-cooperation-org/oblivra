//go:build windows

package tamper

import "os"

// statInode on Windows. Same trick as internal/io/stat_windows.go —
// hash (size, mtime) into a uint64 because os.FileInfo doesn't expose
// NTFS file index portably. Two heartbeats with identical (size,
// mtime) collide, but heartbeats fire 30s apart and the server's
// detection rules tolerate identical inodes — they only alarm on
// inode CHANGE, never inode equality.
func statInode(st os.FileInfo) uint64 {
	mtime := uint64(st.ModTime().UnixNano())
	size := uint64(st.Size())
	return mtime ^ (size << 1)
}
