package eventbus

import (
	"sync"
)

// CollectingBus is a specialized event bus for deterministic replay.
// It records all published events in memory rather than broadcasting to async handlers.
type CollectingBus struct {
	mu     sync.RWMutex
	Events []Event
}

func NewCollectingBus() *CollectingBus {
	return &CollectingBus{
		Events: make([]Event, 0),
	}
}

func (b *CollectingBus) Publish(eventType EventType, data interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Events = append(b.Events, Event{
		Type: eventType,
		Data: data,
	})
}

// GetEvents returns a snapshot of all events filtered by type.
func (b *CollectingBus) GetEvents(filter EventType) []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if filter == AllEvents {
		res := make([]Event, len(b.Events))
		copy(res, b.Events)
		return res
	}

	var res []Event
	for _, e := range b.Events {
		if e.Type == filter {
			res = append(res, e)
		}
	}
	return res
}
