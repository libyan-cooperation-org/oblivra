//go:build windows
// +build windows

package memory

import (
	"crypto/rand"
	"runtime"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ActiveAllocations tracks how many secure buffers exist in RAM.
// Useful for leak-detection metrics in the MemorySecurityService.
var ActiveAllocations int32

// SecureBuffer is a wrapper around a byte slice that guarantees
// its contents will be zeroed out when Wipe() is called, or when
// the garbage collector reclaims the object (via SetFinalizer).
type SecureBuffer struct {
	addr     uintptr
	size     uintptr
	data     []byte
	wiped    bool
	fallback bool
}

// NewSecureBuffer allocates a buffer safely using OS-level VirtualAlloc,
// outside the reach of the Go Garbage Collector, and locks it to RAM
// to prevent it from being paged out to disk.
func NewSecureBuffer(size int) *SecureBuffer {
	// 1. Allocate committed, read-write memory pages
	addr, err := windows.VirtualAlloc(
		0,
		uintptr(size),
		windows.MEM_COMMIT|windows.MEM_RESERVE,
		windows.PAGE_READWRITE,
	)
	if err != nil {
		// Fallback to standard golang memory if virtual allocation fails
		slice := make([]byte, size)
		sb := &SecureBuffer{
			data:     slice,
			wiped:    false,
			fallback: true,
		}
		atomic.AddInt32(&ActiveAllocations, 1)
		runtime.SetFinalizer(sb, func(b *SecureBuffer) { b.Wipe() })
		return sb
	}

	// 2. Lock the memory into physical RAM to prevent swapping/paging
	err = windows.VirtualLock(addr, uintptr(size))
	if err != nil {
		// If locking fails, we free the memory before falling back
		_ = windows.VirtualFree(addr, 0, windows.MEM_RELEASE)
		slice := make([]byte, size)
		sb := &SecureBuffer{
			data:     slice,
			wiped:    false,
			fallback: true,
		}
		atomic.AddInt32(&ActiveAllocations, 1)
		runtime.SetFinalizer(sb, func(b *SecureBuffer) { b.Wipe() })
		return sb
	}

	// 3. Construct a Go slice backed by the raw pointer
	slice := unsafe.Slice((*byte)(unsafe.Pointer(addr)), size)

	sb := &SecureBuffer{
		addr:     addr,
		size:     uintptr(size),
		data:     slice,
		wiped:    false,
		fallback: false,
	}
	atomic.AddInt32(&ActiveAllocations, 1)

	// Attach a finalizer to ensure memory wiping if the developer
	// forgets to call Wipe() explicitly over the lifecycle.
	runtime.SetFinalizer(sb, func(b *SecureBuffer) {
		b.Wipe()
	})

	return sb
}


// FromString creates a SecureBuffer from a string, copying the bytes.
func FromString(s string) *SecureBuffer {
	sb := NewSecureBuffer(len(s))
	copy(sb.data, s)
	return sb
}

// FromBytes creates a SecureBuffer from an existing slice, copying the bytes
// so the original slice can be zeroed out independently if needed.
func FromBytes(b []byte) *SecureBuffer {
	sb := NewSecureBuffer(len(b))
	copy(sb.data, b)
	return sb
}

// Data returns the underlying byte slice.
// Modifying this slice modifies the SecureBuffer directly.
func (sb *SecureBuffer) Data() []byte {
	return sb.data
}

// Wipe actively zeros out the underlying byte slice array in memory,
// replacing bits with randomized noise first, and then zero vectors,
// to mitigate cold-boot data remanence vulnerabilities.
// It then unlocks the RAM and frees the virtual pages back to the OS.
func (sb *SecureBuffer) Wipe() {
	if sb == nil || sb.wiped {
		return
	}

	// Double-pass wiping strategy:
	// Pass 1: Cryptographic noise
	_, _ = rand.Read(sb.data)

	// Pass 2: Zeroing
	for i := range sb.data {
		sb.data[i] = 0
	}

	// Barrier: ensure the compiler doesn't elide the zeroing if it determines
	// the buffer is about to be released/freed.
	runtime.KeepAlive(sb.data)

	// Remove slice reference
	sb.data = nil

	if !sb.fallback {
		// 1. Unlock memory from physical RAM
		_ = windows.VirtualUnlock(sb.addr, sb.size)

		// 2. Free virtual memory pages
		_ = windows.VirtualFree(sb.addr, 0, windows.MEM_RELEASE)
	}

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
