package agent

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain wraps the agent package's tests with goleak. Every collector
// (FileTail, EventLog, Metrics, FIM, Backfill, eBPF) launches its own
// goroutine on Start; tests that don't call Stop on shutdown leak,
// and this gate now catches that fail-closed.
//
// See `internal/eventbus/goleak_test.go` for the full rationale.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).Shutdown"),
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	)
}
