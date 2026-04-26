package eventbus

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain wraps the package's tests with goleak so any goroutine left
// running after a test exits fails the suite. Closes Phase 25.4
// "132 untracked goroutine launches" by making goroutine leaks visible
// at the test boundary instead of letting them accumulate silently.
//
// Ignored goroutines: the Go runtime spawns a few well-known background
// routines (proc.gcBgMarkWorker, runtime.timeBeginPeriod on Windows,
// the OpenTelemetry tracer's batch span processor) that are NOT leaks.
// goleak.IgnoreTopFunction lets us whitelist them so genuine leaks
// surface clearly.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).Shutdown"),
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	)
}
