package io

// The pipeline glues inputs → filters → outputs. One pipeline per
// process; the agent and the server each own one.
//
// Channel topology:
//
//   input1 ─┐
//   input2 ─┼─→ events  (single bounded chan, 4096 deep)
//   inputN ─┘     │
//                 ↓ (per-event)
//             [filters]   (synchronous, ordered, drop-allowed)
//                 │
//                 ↓ (fan-out — each output has its own bounded chan)
//           ┌────┼────┐
//        output1 …  outputN
//
// Backpressure: when an output's per-output channel is full, the
// pipeline drops the event for THAT output (records a metric) but
// keeps fanning out to others. Slow output → no head-of-line blocking
// across the rest of the system.

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

const (
	// inputBufferSize bounds the per-input → fan-in channel. 4096
	// events is ~10s of buffer at 400 EPS per input — enough to
	// absorb a brief output stall, small enough that operators
	// notice if outputs are persistently behind.
	inputBufferSize = 4096

	// outputBufferSize is per-output. Smaller than the fan-in buffer
	// because the pipeline can drop here without affecting other
	// outputs.
	outputBufferSize = 1024

	// flushInterval is how often the pipeline calls Flush() on every
	// output, regardless of buffer state.
	flushInterval = 5 * time.Second
)

// Pipeline runs a set of inputs through a chain of filters, fanning
// the result out to a set of outputs. Use NewPipeline to construct,
// then Add* methods to populate, then Start.
type Pipeline struct {
	inputs  []Input
	outputs []Output
	filters []Filter

	log *logger.Logger

	// Per-output queues. Built in Start so we know the output count.
	outputQueues []chan Event

	// Metrics — atomic so they're safe to read from any goroutine.
	// `eventsIn`, `eventsOut` are coarse; per-output drops are fine
	// for surfacing "output X is wedged" without a full Prometheus
	// dance. A future enrichment exposes these via `/api/v1/io/metrics`.
	eventsIn   atomic.Uint64
	eventsOut  atomic.Uint64
	eventsDrop atomic.Uint64

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewPipeline(log *logger.Logger) *Pipeline {
	return &Pipeline{
		log: log.WithPrefix("io"),
	}
}

func (p *Pipeline) AddInput(in Input)    { p.inputs = append(p.inputs, in) }
func (p *Pipeline) AddOutput(out Output) { p.outputs = append(p.outputs, out) }
func (p *Pipeline) AddFilter(f Filter)   { p.filters = append(p.filters, f) }

// Start launches every plugin and the fan-out goroutine. Returns
// after every input's Start() has returned successfully (which they
// do quickly because actual work is in the input's own goroutines).
func (p *Pipeline) Start(ctx context.Context) error {
	if len(p.outputs) == 0 {
		return fmt.Errorf("pipeline: no outputs configured")
	}
	pipelineCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	// Provision per-output queues.
	p.outputQueues = make([]chan Event, len(p.outputs))
	for i := range p.outputs {
		p.outputQueues[i] = make(chan Event, outputBufferSize)
	}

	// Single fan-in channel; every input writes here.
	fanIn := make(chan Event, inputBufferSize)

	// Start every input.
	for _, in := range p.inputs {
		in := in
		if err := in.Start(pipelineCtx, fanIn); err != nil {
			cancel()
			return fmt.Errorf("pipeline: input %q (%s) failed to start: %w", in.Name(), in.Type(), err)
		}
		p.log.Info("[input %s] %s started", in.Type(), in.Name())
	}

	// Fan-out goroutine: read from fanIn, run filters, write to each
	// output's queue.
	p.wg.Add(1)
	go p.fanOutLoop(pipelineCtx, fanIn)

	// Per-output writer goroutines.
	for i, out := range p.outputs {
		out := out
		queue := p.outputQueues[i]
		p.wg.Add(1)
		go p.outputLoop(pipelineCtx, out, queue)
	}

	// Periodic flush.
	p.wg.Add(1)
	go p.flushLoop(pipelineCtx)

	p.log.Info("Pipeline started: %d input(s), %d filter(s), %d output(s)",
		len(p.inputs), len(p.filters), len(p.outputs))
	return nil
}

// Stop cancels every plugin context, waits for goroutines to drain,
// then calls each output's Close().
func (p *Pipeline) Stop(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	// Wait with a timeout — pathological plugins shouldn't hang shutdown.
	done := make(chan struct{})
	go func() { p.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		p.log.Warn("Pipeline shutdown timed out — some goroutines still alive")
	}

	// Stop inputs first (they'd otherwise keep writing to the closed channels).
	for _, in := range p.inputs {
		_ = in.Stop()
	}
	// Then close outputs.
	for _, out := range p.outputs {
		_ = out.Close()
	}

	p.log.Info("Pipeline stopped: in=%d out=%d drop=%d",
		p.eventsIn.Load(), p.eventsOut.Load(), p.eventsDrop.Load())
	return nil
}

// Stats returns a snapshot for diagnostic surfaces.
func (p *Pipeline) Stats() (in, out, drop uint64) {
	return p.eventsIn.Load(), p.eventsOut.Load(), p.eventsDrop.Load()
}

func (p *Pipeline) fanOutLoop(ctx context.Context, fanIn <-chan Event) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-fanIn:
			if !ok {
				return
			}
			p.eventsIn.Add(1)
			if ev.Timestamp.IsZero() {
				ev.Timestamp = time.Now().UTC()
			}

			// Run filters. Any filter returning false drops the event.
			drop := false
			for _, f := range p.filters {
				next, keep := f(ev)
				if !keep {
					drop = true
					break
				}
				ev = next
			}
			if drop {
				p.eventsDrop.Add(1)
				continue
			}

			// Fan out — non-blocking. A wedged output drops, doesn't
			// block its peers.
			for i, q := range p.outputQueues {
				select {
				case q <- ev:
				default:
					// Output's queue is full → drop and metric.
					p.eventsDrop.Add(1)
					p.log.Debug("[output %s] dropped event (queue full)", p.outputs[i].Name())
				}
			}
		}
	}
}

func (p *Pipeline) outputLoop(ctx context.Context, out Output, q <-chan Event) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-q:
			if !ok {
				return
			}
			if err := out.Write(ctx, ev); err != nil {
				// Don't drop — Write is responsible for buffering /
				// retrying. Log the surface error so operators see
				// the trail.
				p.log.Debug("[output %s] write failed: %v", out.Name(), err)
				continue
			}
			p.eventsOut.Add(1)
		}
	}
}

func (p *Pipeline) flushLoop(ctx context.Context) {
	defer p.wg.Done()
	t := time.NewTicker(flushInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			// Final flush before shutdown.
			fctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			for _, out := range p.outputs {
				_ = out.Flush(fctx)
			}
			cancel()
			return
		case <-t.C:
			for _, out := range p.outputs {
				if err := out.Flush(ctx); err != nil {
					p.log.Debug("[output %s] flush failed: %v", out.Name(), err)
				}
			}
		}
	}
}
