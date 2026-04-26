package ingest_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/ingest"
	"github.com/kingknull/oblivrashell/internal/logger"
)

func newTestPipeline(t *testing.T) *ingest.Pipeline {
	t.Helper()
	log := logger.NewStdoutLogger()
	bus := eventbus.NewBus(log)
	t.Cleanup(func() { bus.Close() })
	return ingest.NewPipeline(10000, nil, nil, nil, bus, log, nil, nil, nil, nil)
}

func mkEvent(id, eventType, host, rawLine string) *events.SovereignEvent {
	return &events.SovereignEvent{
		Id:        id,
		EventType: eventType,
		Host:      host,
		RawLine:   rawLine,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// ── basic queue and process ───────────────────────────────────────────────────

func TestPipeline_QueueAndProcess(t *testing.T) {
	p := newTestPipeline(t)
	p.Start()
	defer p.Stop()

	e := mkEvent("test-001", "syslog", "host-1", "test log line")
	if err := p.QueueEvent(e); err != nil {
		t.Fatalf("QueueEvent: %v", err)
	}

	// Wait for pipeline to process
	time.Sleep(200 * time.Millisecond)

	m := p.GetMetrics()
	if m.TotalProcessed == 0 {
		t.Error("expected TotalProcessed > 0 after queuing one event")
	}
}

func TestPipeline_DropsBeyondBuffer(t *testing.T) {
	log := logger.NewStdoutLogger()
	bus := eventbus.NewBus(log)
	t.Cleanup(bus.Close) // Phase 25.4: drain bus workers on test exit

	// Tiny buffer of 5 — do NOT start workers so events accumulate.
	// As of the backpressure rework, the pipeline load-sheds via the
	// adaptive controller rather than returning ErrBufferFull from
	// QueueEvent. The observable signal is the metric counter; either
	// path indicates the pipeline correctly refused over-capacity work.
	p := ingest.NewPipeline(5, nil, nil, nil, bus, log, nil, nil, nil, nil)

	directRejects := 0
	for i := 0; i < 10; i++ {
		err := p.QueueEvent(mkEvent(fmt.Sprintf("e-%d", i), "test", "", ""))
		if err == ingest.ErrBufferFull {
			directRejects++
		}
	}

	// Either path is acceptable: direct rejects OR load-shed drops.
	loadShed := p.GetMetrics().DroppedEvents
	if directRejects == 0 && loadShed == 0 {
		t.Errorf(
			"expected at least one event to be refused (direct=%d, load_shed=%d)",
			directRejects, loadShed,
		)
	}
}

// ── metrics ───────────────────────────────────────────────────────────────────

func TestPipeline_MetricsReflectThroughput(t *testing.T) {
	p := newTestPipeline(t)
	p.Start()
	defer p.Stop()

	const count = 50
	for i := 0; i < count; i++ {
		p.QueueEvent(mkEvent(fmt.Sprintf("evt-%d", i), "syslog", "host-1", //nolint:errcheck
			fmt.Sprintf("log line %d", i)))
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if p.GetMetrics().TotalProcessed >= int64(count) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	processed := p.GetMetrics().TotalProcessed
	if processed < int64(count) {
		t.Errorf("expected TotalProcessed ≥ %d, got %d", count, processed)
	}
}

// ── benchmark ─────────────────────────────────────────────────────────────────

func BenchmarkPipeline_Throughput(b *testing.B) {
	log := logger.NewStdoutLogger()
	bus := eventbus.NewBus(log)
	defer bus.Close()
	p := ingest.NewPipeline(100000, nil, nil, nil, bus, log, nil, nil, nil, nil)
	p.Start()
	defer p.Stop()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := mkEvent("bench-1", "syslog", "bench-host",
			"<34>1 2026-01-01T00:00:00Z bench-host sshd - - Failed password for root from 1.2.3.4")
		p.QueueEvent(e) //nolint:errcheck
	}
}

func BenchmarkPipeline_AutoParse(b *testing.B) {
	lines := []string{
		`<34>1 2026-01-01T00:00:00Z myhost sshd - - Failed password for user root from 1.2.3.4 port 51234 ssh2`,
		`{"event":{"type":"process_create"},"process":{"command_line":"powershell.exe -enc SQBFAFgA"}}`,
		`2026-01-01T00:00:00.000Z [ERROR] Exception in thread main: NullPointerException`,
		`<189>Jan  1 00:00:00 router: %SEC-6-IPACCESSLOGP: list 101 denied tcp`,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pCtx := events.EventProcessingContext{
			EventID:  fmt.Sprintf("bench-%d", i),
			TenantID: "GLOBAL",
			Now:      time.Unix(1700000000, 0).UTC(),
		}
		ingest.AutoParse(lines[i%len(lines)], pCtx)
	}
}

// ── context cancellation ──────────────────────────────────────────────────────

func TestPipeline_StopsCleanly(t *testing.T) {
	p := newTestPipeline(t)
	p.Start()

	for i := 0; i < 10; i++ {
		p.QueueEvent(mkEvent(fmt.Sprintf("e-%d", i), "syslog", "", "")) //nolint:errcheck
	}

	done := make(chan struct{})
	go func() {
		p.Stop()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(5 * time.Second):
		t.Error("pipeline.Stop() timed out — possible goroutine leak")
	}
}
