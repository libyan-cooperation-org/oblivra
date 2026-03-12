package services

import (
	"github.com/kingknull/oblivrashell/internal/eventbus"
)

// busShim bridges the eventbus with sharing package
type busShim struct {
	bus *eventbus.Bus
}

func (s *busShim) Publish(topic string, data interface{}) {
	s.bus.Publish(eventbus.EventType(topic), data)
}
