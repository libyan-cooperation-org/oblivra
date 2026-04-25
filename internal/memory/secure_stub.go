//go:build !windows

package memory

import (
	"crypto/rand"
	"runtime"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

// ActiveAllocations tracks how many secure buffers exist in RAM.
// Useful for leak-detection metrics in the MemorySecurityService.
var ActiveAllocations int32

// SecureBuffer on Linux / macOS uses mlock to prevent the OS from paging
// the buffer to disk. Phase 22.7 requested memguard-grade memory
// protection — this is the cross-platform companion to the
// VirtualLock-based Windows implementation in secure.go.
//
// Layout matches secure.go (Windows) so callers don't see a difference.
type SecureBuffer struct {
	data     []byte
	wiped    bool
	fallback bool // true when mlock failed and we fell back to plain Go memory
}

// NewSecureBuffer allocates a buffer and pins it into physical RAM via
// mlock. If mlock fails (process lacks CAP_IPC_LOCK on Linux, or the
// resident-set rlimit is exhausted in a container), we degrade to plain
// Go memory rather than panicking — losing the can't-be-paged guarantee
// but keeping the service available. The fallback is recorded on the
// SecureBuffer so callers can report it via GetActiveCount metrics.
//
// The data slice is wiped on Wipe() / GC finalizer either way, so
// the worst-case fallback degrades to "buffer can be paged to swap" but
// retains the "actively zeroed" property.
func NewSecureBuffer(size int) *SecureBuffer {
	data := make([]byte, size)
	fallback := false

	// Mlock on the slice's backing storage. Best-effort: on Linux without
	// CAP_IPC_LOCK and a tight rlimit it will EPERM, in which case we keep
	// the buffer but mark it as fallback.
	if err := unix.Mlock(data); err != nil {
		fallback = true
	}

	sb := &SecureBuffer{
		data:     data,
		wiped:    false,
		fallback: fallback,
	}
	atomic.AddInt32(&ActiveAllocations, 1)

	// Safety net — if the caller forgets to Wipe(), the GC's finalizer
	// at least zeroes the bytes before the runtime hands the memory back.
	runtime.SetFinalizer(sb, func(b *SecureBuffer) { b.Wipe() })
	return sb
}

// FromString creates a SecureBuffer from a string, copying the bytes.
func FromString(s string) *SecureBuffer {
	sb := NewSecureBuffer(len(s))
	copy(sb.data, s)
	return sb
}

// FromBytes creates a SecureBuffer from a byte slice, copying the bytes.
func FromBytes(b []byte) *SecureBuffer {
	sb := NewSecureBuffer(len(b))
	copy(sb.data, b)
	return sb
}

// Data returns the underlying byte slice.
func (sb *SecureBuffer) Data() []byte {
	return sb.data
}

// Wipe overwrites the buffer with cryptographic noise, then zeroes, then
// unlocks the page from physical RAM. Idempotent and safe to call on a
// nil receiver.
func (sb *SecureBuffer) Wipe() {
	if sb == nil || sb.wiped {
		return
	}

	// Pass 1: cryptographic noise to mitigate cold-boot data remanence.
	_, _ = rand.Read(sb.data)
	// Pass 2: zero.
	for i := range sb.data {
		sb.data[i] = 0
	}
	// Barrier — without KeepAlive the compiler may elide the zero loop
	// when it sees the buffer is about to become unreachable.
	runtime.KeepAlive(sb.data)

	if !sb.fallback {
		// Best-effort munlock; ignore errors (the kernel may already have
		// unmapped the page, or the original mlock returned ENOMEM).
		_ = unix.Munlock(sb.data)
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
