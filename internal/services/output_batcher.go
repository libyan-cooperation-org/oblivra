package services

import (
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"golang.org/x/net/context"
)

// OutputBatcher aggregates small terminal output chunks into larger events
// to reduce the total number of IPC calls and improve UI performance.
type OutputBatcher struct {
	ctx       context.Context
	sessionID string
	buffer    []byte
	mu        sync.Mutex
	timer     *time.Timer
	maxDelay  time.Duration
	maxSize   int
}

func NewOutputBatcher(ctx context.Context, sessionID string) *OutputBatcher {
	return &OutputBatcher{
		ctx:       ctx,
		sessionID: sessionID,
		buffer:    make([]byte, 0, 65536), // Pre-allocate up to 64KB
		maxDelay:  16 * time.Millisecond,  // 60 FPS target
		maxSize:   65536,                  // 64KB max batch size
	}
}

func (b *OutputBatcher) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buffer = append(b.buffer, p...)

	// If buffer is large enough, flush immediately
	if len(b.buffer) >= b.maxSize {
		b.flushLocked()
		return len(p), nil
	}

	// Otherwise, start/reset the flush timer
	if b.timer == nil {
		b.timer = time.AfterFunc(b.maxDelay, b.Flush)
	}

	return len(p), nil
}

func (b *OutputBatcher) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flushLocked()
}

func (b *OutputBatcher) flushLocked() {
	if len(b.buffer) == 0 {
		return
	}

	// Reset timer
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	// Encode as plain base64 (frontend decodes via atob)
	encoded := base64.StdEncoding.EncodeToString(b.buffer)
	
	// Primary event for Svelte 5 Terminal component
	EmitEvent(fmt.Sprintf("terminal:out:%s", b.sessionID), encoded)
	
	// Legacy / internal events
	EmitEvent(fmt.Sprintf("terminal-output-%s", b.sessionID), encoded)
	EmitEvent(fmt.Sprintf("session.output.%s", b.sessionID), encoded)

	// Clear buffer
	b.buffer = b.buffer[:0]
}
