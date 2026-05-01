package trust

import (
	"sync"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// TestEngineConcurrentObserve hammers the trust engine from many goroutines
// — fingerprint corroboration + sequence watermarks all share state, so
// this is what catches a future refactor that drops a lock.
func TestEngineConcurrentObserve(t *testing.T) {
	e := New()
	const workers = 20
	const perW = 100
	now := time.Now().UTC().Truncate(time.Minute)

	var wg sync.WaitGroup
	wg.Add(workers)
	for g := 0; g < workers; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perW; i++ {
				ev := events.Event{
					Source: events.SourceSyslog, HostID: "h",
					Message:   "msg",
					Timestamp: now.Add(time.Duration(i) * time.Second),
					Provenance: events.Provenance{
						IngestPath: "syslog-udp",
					},
				}
				_ = ev.Validate()
				e.Observe(ev)
			}
		}(g)
	}
	wg.Wait()

	s := e.Summary()
	total := s.Verified + s.Consistent + s.Suspicious + s.Untrusted
	if total != workers*perW {
		t.Errorf("summary total = %d, want %d", total, workers*perW)
	}
}
