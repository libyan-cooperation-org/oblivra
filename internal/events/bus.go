package events

import (
	"sync"
)

// Bus is a fan-out broadcaster of events to N subscribers. Each subscriber
// gets its own buffered channel; slow subscribers drop events rather than
// blocking the producer (ingest must never stall behind a UI tab).
type Bus struct {
	mu   sync.RWMutex
	next int
	subs map[int]chan Event
	cap  int
}

func NewBus(perSubBuffer int) *Bus {
	if perSubBuffer <= 0 {
		perSubBuffer = 256
	}
	return &Bus{subs: make(map[int]chan Event), cap: perSubBuffer}
}

// Subscribe returns a channel and a cancel func.
func (b *Bus) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, b.cap)
	b.mu.Lock()
	id := b.next
	b.next++
	b.subs[id] = ch
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		if c, ok := b.subs[id]; ok {
			close(c)
			delete(b.subs, id)
		}
		b.mu.Unlock()
	}
}

// Publish fans an event out to every subscriber. Drops on full buffers.
func (b *Bus) Publish(ev Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- ev:
		default:
			// subscriber too slow — drop rather than block.
		}
	}
}

// Count returns the current subscriber count.
func (b *Bus) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}
