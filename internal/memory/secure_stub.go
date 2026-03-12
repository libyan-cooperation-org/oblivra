//go:build !windows

package memory

import (
	"crypto/rand"
	"sync/atomic"
)

// ActiveAllocations tracks how many secure buffers exist in RAM.
var ActiveAllocations int32

// SecureBuffer is a stub implementation for non-Windows platforms.
// It uses standard Go memory and does not provide OS-level locking.
type SecureBuffer struct {
	data  []byte
	wiped bool
}

// NewSecureBuffer allocates a standard buffer.
func NewSecureBuffer(size int) *SecureBuffer {
	data := make([]byte, size)
	sb := &SecureBuffer{
		data:  data,
		wiped: false,
	}
	atomic.AddInt32(&ActiveAllocations, 1)
	return sb
}

// FromString creates a SecureBuffer from a string.
func FromString(s string) *SecureBuffer {
	sb := NewSecureBuffer(len(s))
	copy(sb.data, s)
	return sb
}

// FromBytes creates a SecureBuffer from a byte slice.
func FromBytes(b []byte) *SecureBuffer {
	sb := NewSecureBuffer(len(b))
	copy(sb.data, b)
	return sb
}

// Data returns the underlying byte slice.
func (sb *SecureBuffer) Data() []byte {
	return sb.data
}

// Wipe zeros out the buffer and marks it as wiped.
func (sb *SecureBuffer) Wipe() {
	if sb == nil || sb.wiped {
		return
	}

	// Overwrite with random data then zeros
	_, _ = rand.Read(sb.data)
	for i := range sb.data {
		sb.data[i] = 0
	}

	sb.data = nil
	sb.wiped = true
	atomic.AddInt32(&ActiveAllocations, -1)
}

// IsWiped returns true if Wipe() has been executed.
func (sb *SecureBuffer) IsWiped() bool {
	return sb.wiped
}

// GetActiveCount retrieves the total number of non-wiped buffers currently allocated.
func GetActiveCount() int32 {
	return atomic.LoadInt32(&ActiveAllocations)
}
