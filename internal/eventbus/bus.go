package eventbus

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
	"golang.org/x/time/rate"
)

type EventType string

const (
	AllEvents EventType = "*"

	EventConnectionOpened EventType = "session:started"
	EventConnectionClosed EventType = "session:closed"
	EventConnectionError  EventType = "session:error"
	EventConnectionRetry  EventType = "session:retry"

	EventSessionCreated   EventType = "session:created"
	EventSessionDestroyed EventType = "session:destroyed"
	EventSessionOutput    EventType = "session:output"

	EventVaultUnlocked EventType = "vault:unlocked"
	EventVaultLocked   EventType = "vault:locked"
	EventVaultTimeout  EventType = "vault:timeout"

	EventHostCreated EventType = "host:list_updated"
	EventHostUpdated EventType = "host:list_updated"
	EventHostDeleted EventType = "host:list_updated"

	EventCredentialCreated  EventType = "credential:created"
	EventCredentialDeleted  EventType = "credential:deleted"
	EventCredentialAccessed EventType = "credential:accessed"

	EventAppReady        EventType = "app:ready"
	EventAppError        EventType = "app:error"
	EventThemeChanged    EventType = "theme:changed"
	EventSettingsChanged EventType = "settings:changed"

	EventSIEMAlert EventType = "siem:alert"

	EventPolicyApprovalRequested EventType = "policy:approval_requested"
	EventPolicyApprovalGranted   EventType = "policy:approval_granted"
	EventPolicyApprovalDenied    EventType = "policy:approval_denied"

	EventFIMCreated     EventType = "fim:created"
	EventFIMModified    EventType = "fim:modified"
	EventFIMDeleted     EventType = "fim:deleted"
	EventFIMRenamed     EventType = "fim:renamed"
	EventSSHLoginFailed EventType = "ssh:login_failed"
)

type Event struct {
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type Handler func(event Event)

type subscription struct {
	ch      chan Event
	handler Handler
}

type Bus struct {
	mu          sync.RWMutex
	handlers    map[EventType][]*subscription // Changed from subscribers
	log         *logger.Logger
	ingestLimit *rate.Limiter
	dropped     uint64
	closing     chan struct{}
}

// NewBus creates a new central event bus.
// To survive Economic DoS, we strictly limit ingestion capacity.
func NewBus(log *logger.Logger) *Bus {
	return &Bus{
		handlers: make(map[EventType][]*subscription), // Changed from subscribers
		log:      log,
		// Max bursts of 5000 events, refilling at 1000 items/second.
		// Exceeding this triggers immediate load-shedding before GC trashing.
		ingestLimit: rate.NewLimiter(rate.Limit(1000), 5000),
		closing:     make(chan struct{}),
	}
}

func (b *Bus) Subscribe(eventType EventType, handler Handler) {
	if handler == nil {
		b.log.Error("Bus.Subscribe called with nil handler for topic: %s", eventType)
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create bounded buffer for this specific handler
	sub := &subscription{
		ch:      make(chan Event, 2000), // Buffer up to 2000 events per subscriber
		handler: handler,
	}

	b.handlers[eventType] = append(b.handlers[eventType], sub) // Changed from subscribers

	// Start the robust worker for this subscription
	go func(s *subscription) {
		defer func() {
			if r := recover(); r != nil {
				b.log.Error("[SOVEREIGN-BUS] PANIC RECOVERED in handler for %s: %v", eventType, r)
			}
		}()
		for {
			select {
			case event, ok := <-s.ch:
				if !ok {
					return
				}
				s.handler(event)
			case <-b.closing:
				return
			}
		}
	}(sub)
}

// Publish broadcasts an event to all subscribers of its exact topic.
// Silently drops the event if rate limit is exceeded.
func (b *Bus) Publish(eventType EventType, data interface{}) {
	// 1. Defend the stack: strict backpressure evaluation
	if !b.ingestLimit.Allow() {
		// Event cannot be processed. Drop it immediately to protect the memory heap.
		atomic.AddUint64(&b.dropped, 1)

		// Only log occasionally to avoid I/O bottlenecks during DoS
		if atomic.LoadUint64(&b.dropped)%1000 == 0 {
			b.log.Error("[SOVEREIGN-PANIC] ECONOMIC DOS DETECTED. DROPPED %d EVENTS DUE TO INGESTION RATELIMIT", atomic.LoadUint64(&b.dropped))
		}
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	event := Event{
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}

	// Dispatch to specific handlers
	if subs, ok := b.handlers[eventType]; ok {
		for _, sub := range subs {
			select {
			case sub.ch <- event:
				// Successfully queued
			default:
				// Buffer full, drop event to protect system memory
			}
		}
	}

	// Dispatch to global catch-all handlers
	if subs, ok := b.handlers[AllEvents]; ok {
		for _, sub := range subs {
			select {
			case sub.ch <- event:
			default:
			}
		}
	}
}

// Close gracefully shuts down all persistent event worker loops
func (b *Bus) Close() {
	close(b.closing)
}
