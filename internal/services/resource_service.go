package services

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ResourceMonitor monitors system resource usage (CPU/Memory)
// and reports pressure to the health system.
type ResourceMonitor struct {
	log           *logger.Logger
	stop          context.CancelFunc
	memThreshold  uint64 // Max allowable heap MB before signaling pressure
	pressure      bool
}

func NewResourceMonitor(log *logger.Logger, maxHeapMB uint64) *ResourceMonitor {
	return &ResourceMonitor{
		log:          log.WithPrefix("res_mon"),
		memThreshold: maxHeapMB,
	}
}

func (s *ResourceMonitor) Name() string { return "ResourceMonitor" }

func (s *ResourceMonitor) Dependencies() []string { return nil }

func (s *ResourceMonitor) Start(ctx context.Context) error {
	innerCtx, cancel := context.WithCancel(ctx)
	s.stop = cancel

	go s.monitorLoop(innerCtx)
	return nil
}

func (s *ResourceMonitor) Stop(ctx context.Context) error {
	if s.stop != nil {
		s.stop()
	}
	return nil
}

func (s *ResourceMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var m runtime.MemStats
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runtime.ReadMemStats(&m)
			heapMB := m.Alloc / 1024 / 1024

			if heapMB > s.memThreshold {
				if !s.pressure {
					s.log.Warn("MEMORY PRESSURE DETECTED: %dMB > %dMB", heapMB, s.memThreshold)
					s.pressure = true
				}
			} else {
				if s.pressure {
					s.log.Info("Memory pressure relieved: %dMB", heapMB)
					s.pressure = false
				}
			}
		}
	}
}

func (s *ResourceMonitor) Health(ctx context.Context) error {
	if s.pressure {
		return fmt.Errorf("high memory pressure detected")
	}
	return nil
}
