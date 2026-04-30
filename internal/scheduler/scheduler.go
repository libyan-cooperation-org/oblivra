// Package scheduler runs the platform's recurring background jobs — warm-tier
// migration, audit-chain health pings, and anything else that should fire on a
// timer rather than on user demand. It's intentionally a tiny in-process
// ticker manager: no cron syntax, no persistence, no distributed locking.
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Job struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
}

type Scheduler struct {
	log  *slog.Logger
	jobs []Job

	mu     sync.Mutex
	cancel func()
	done   chan struct{}
}

func New(log *slog.Logger) *Scheduler {
	return &Scheduler{log: log}
}

// Add registers a job. Must be called before Start.
func (s *Scheduler) Add(j Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j.Interval <= 0 {
		s.log.Warn("scheduler: ignoring zero-interval job", "name", j.Name)
		return
	}
	s.jobs = append(s.jobs, j)
}

// Start launches one goroutine per job. Idempotent.
func (s *Scheduler) Start(parent context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.done = make(chan struct{})
	wg := &sync.WaitGroup{}
	for _, j := range s.jobs {
		wg.Add(1)
		go s.run(ctx, j, wg)
	}
	go func() {
		wg.Wait()
		close(s.done)
	}()
	s.log.Info("scheduler started", "jobs", len(s.jobs))
}

func (s *Scheduler) run(ctx context.Context, j Job, wg *sync.WaitGroup) {
	defer wg.Done()
	t := time.NewTicker(j.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			start := time.Now()
			if err := j.Run(ctx); err != nil {
				s.log.Warn("scheduled job failed", "name", j.Name, "err", err, "took", time.Since(start))
			} else {
				s.log.Debug("scheduled job ok", "name", j.Name, "took", time.Since(start))
			}
		}
	}
}

// Stop cancels all jobs and waits for them to exit.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.cancel = nil
	s.done = nil
	s.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}
