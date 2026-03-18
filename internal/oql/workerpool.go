package oql

import (
	"context"
	"sync"
	"sync/atomic"
)

type WorkerPool struct {
	sem    chan struct{}
	wg     sync.WaitGroup
	size   int
	active atomic.Int64
}

func NewWorkerPool(size, reservePerQuery int) *WorkerPool {
	if reservePerQuery < 1 {
		reservePerQuery = 1
	}
	return &WorkerPool{sem: make(chan struct{}, size), size: size}
}

func (wp *WorkerPool) Submit(ctx context.Context, fn func()) error {
	select {
	case wp.sem <- struct{}{}:
		wp.wg.Add(1)
		wp.active.Add(1)
		go func() {
			defer func() { <-wp.sem; wp.active.Add(-1); wp.wg.Done() }()
			fn()
		}()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (wp *WorkerPool) SubmitGuaranteed(fn func()) {
	select {
	case wp.sem <- struct{}{}:
		wp.wg.Add(1)
		wp.active.Add(1)
		go func() {
			defer func() { <-wp.sem; wp.active.Add(-1); wp.wg.Done() }()
			fn()
		}()
	default:
		wp.wg.Add(1)
		wp.active.Add(1)
		go func() {
			defer func() { wp.active.Add(-1); wp.wg.Done() }()
			fn()
		}()
	}
}

func (wp *WorkerPool) Wait()       { wp.wg.Wait() }
func (wp *WorkerPool) Active() int { return int(wp.active.Load()) }
func (wp *WorkerPool) Available() int {
	a := wp.size - int(wp.active.Load())
	if a < 0 {
		return 0
	}
	return a
}
