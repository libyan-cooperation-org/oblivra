package vault

import (
	"runtime"
)

// SecureBytes is a byte slice that zeros itself when no longer needed
type SecureBytes struct {
	data []byte
}

// NewSecureBytes creates a new secure byte slice
func NewSecureBytes(size int) *SecureBytes {
	sb := &SecureBytes{
		data: make([]byte, size),
	}
	runtime.SetFinalizer(sb, func(s *SecureBytes) {
		s.Release()
	})
	return sb
}

// Release explicitly zeros the data and marks it as unusable
func (s *SecureBytes) Release() {
	if s == nil || s.data == nil {
		return
	}
	s.Zero()
	s.data = nil
}

// NewSecureBytesFromSlice creates a SecureBytes from existing data
func NewSecureBytesFromSlice(data []byte) *SecureBytes {
	sb := &SecureBytes{
		data: make([]byte, len(data)),
	}
	copy(sb.data, data)
	runtime.SetFinalizer(sb, func(s *SecureBytes) {
		s.Zero()
	})
	return sb
}

// Bytes returns the underlying byte slice
func (s *SecureBytes) Bytes() []byte {
	if s == nil {
		return nil
	}
	return s.data
}

// Len returns the length
func (s *SecureBytes) Len() int {
	if s == nil {
		return 0
	}
	return len(s.data)
}

// Zero overwrites the data with zeros. It uses a memory barrier to prevent
// compiler elision of the loop if the buffer is immediately freed.
func (s *SecureBytes) Zero() {
	if s == nil || s.data == nil {
		return
	}
	for i := range s.data {
		s.data[i] = 0
	}
	// Barrier: ensure the compiler doesn't elide the zeroing if it determines
	// the buffer is no longer reachable.
	runtime.KeepAlive(s.data)
}

// ZeroSlice zeros a byte slice in place.
func ZeroSlice(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}

// SecureCompare performs constant-time comparison
func SecureCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
