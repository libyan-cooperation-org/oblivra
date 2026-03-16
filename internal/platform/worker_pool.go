package platform

import (
	"context"
	"runtime"
	"sync"
)

// WorkerPool is a bounded goroutine pool for CPU-heavy operations:
// log parsing, enrichment, detection evaluation, and compression.
//
// Unlike raw goroutines, this prevents goroutine explosion under bursty load:
// if all workers are busy, Submit blocks (backpressure) rather than spawning
// an unbounded number of goroutines that exhaust memory.
//
// Usage:
//
//	pool := NewWorkerPool("enrichment", runtime.NumCPU()*2)
//	pool.Start()
//	defer pool.Stop()
//
//	pool.Submit(func() { enrich(event) })
type WorkerPool struct {
	name    string
	workers int
	jobs    chan func()
	wg      sync.WaitGroup
	once    sync.Once
	cancel  context.CancelFunc
	ctx     context.Context
}

// NewWorkerPool creates a pool with the given number of workers and a job queue
// sized at workers×10 (enough to absorb short bursts without blocking callers).
func NewWorkerPool(name string, workers int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		name:    name,
		workers: workers,
		jobs:    make(chan func(), workers*10),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// NewWorkerPoolDefaults creates a pool sized at NumCPU×2 — the recommended
// setting for I/O-bound enrichment / parsing workloads.
func NewWorkerPoolDefaults(name string) *WorkerPool {
	return NewWorkerPool(name, runtime.NumCPU()*2)
}

// Start launches the worker goroutines. Safe to call multiple times (idempotent).
func (p *WorkerPool) Start() {
	p.once.Do(func() {
		for i := 0; i < p.workers; i++ {
			p.wg.Add(1)
			go func() {
				defer p.wg.Done()
				defer func() {
					if r := recover(); r != nil {
						// Worker panics must not kill the pool
						_ = r
					}
				}()
				for {
					select {
					case job, ok := <-p.jobs:
						if !ok {
							return
						}
						job()
					case <-p.ctx.Done():
						return
					}
				}
			}()
		}
	})
}

// Submit enqueues a job for execution by the pool.
// Blocks if the job queue is full (backpressure).
// Returns immediately if the pool is shutting down.
func (p *WorkerPool) Submit(job func()) {
	select {
	case p.jobs <- job:
	case <-p.ctx.Done():
	}
}

// TrySubmit enqueues a job without blocking. Returns false if queue is full.
func (p *WorkerPool) TrySubmit(job func()) bool {
	select {
	case p.jobs <- job:
		return true
	default:
		return false
	}
}

// Stop drains the job queue and waits for all workers to finish.
func (p *WorkerPool) Stop() {
	p.cancel()
	close(p.jobs)
	p.wg.Wait()
}

// Workers returns the number of worker goroutines in the pool.
func (p *WorkerPool) Workers() int { return p.workers }

// Name returns the pool name (for logging/metrics).
func (p *WorkerPool) Name() string { return p.name }
