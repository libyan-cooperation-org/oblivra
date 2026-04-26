package eventbus

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestBusPublishSubscribe(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	bus := NewBus(log)
	t.Cleanup(bus.Close) // Phase 25.4: ensure worker goroutines drain on test exit
	var wg sync.WaitGroup
	wg.Add(1)

	var received Event
	bus.Subscribe(EventAppReady, func(event Event) {
		received = event
		wg.Done()
	})

	bus.Publish(EventAppReady, map[string]string{"status": "ok"})

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
		if received.Type != EventAppReady {
			t.Errorf("expected %s, got %s", EventAppReady, received.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestBusWildcardSubscriber(t *testing.T) {
	log, _ := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	bus := NewBus(log)
	t.Cleanup(bus.Close) // Phase 25.4: ensure worker goroutines drain on test exit
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(AllEvents, func(event Event) { wg.Done() })
	bus.Publish(EventHostCreated, nil)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("wildcard subscriber did not receive event")
	}
}
