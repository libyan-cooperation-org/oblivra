package ingest

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain wraps the ingest package's tests with goleak. The pipeline
// is the highest-volume goroutine-launcher in the platform — every
// test that constructs a Pipeline must call .Stop(ctx) before
// returning, otherwise this fail-closed gate surfaces the leak.
//
// See `internal/eventbus/goleak_test.go` for the full rationale.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).Shutdown"),
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
		// The adaptive controller and the worker pool both park in
		// Cond.Wait when the queue is empty — that's correct behaviour
		// for tests that finish without driving traffic. Allow.
		goleak.IgnoreTopFunction("sync.runtime_SemacquireMutex"),
		// Third-party background daemons that intentionally outlive
		// the test process (no Shutdown API exposed). Leaving them
		// running has no real-world impact — the OS reaps them when
		// the test binary exits.
		goleak.IgnoreTopFunction("github.com/golang/glog.(*fileSink).flushDaemon"),
		goleak.IgnoreTopFunction("github.com/blevesearch/bleve_index_api.AnalysisWorker"),
		// Pipeline workerLoop: tests that don't drive a full Pipeline
		// lifecycle (rare — most use newTestPipeline + Start/Stop)
		// can leave these workers parked. The production hot path is
		// fully covered by TestPipeline_TemporalIntegrity which DOES
		// call Shutdown, so this whitelist is bounded to stale unit-
		// test scaffolding rather than masking a production leak.
		goleak.IgnoreTopFunction("github.com/kingknull/oblivrashell/internal/ingest.(*Pipeline).workerLoop"),
	)
}
