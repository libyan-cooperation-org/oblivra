package quality

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// TestEngineConcurrentObserve confirms per-(host, source) reliability
// accumulators don't lose entries under contention.
func TestEngineConcurrentObserve(t *testing.T) {
	e := New()
	const workers = 12
	const hosts = 5
	const perW = 100

	var wg sync.WaitGroup
	wg.Add(workers)
	for g := 0; g < workers; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perW; i++ {
				ev := events.Event{
					Source:    events.SourceSyslog,
					HostID:    "host-" + strconv.Itoa(i%hosts),
					EventType: "syslog.5424",
					Timestamp: time.Now(),
					ReceivedAt: time.Now(),
					Message:    "m",
				}
				_ = ev.Validate()
				e.Observe(ev)
			}
		}(g)
	}
	wg.Wait()

	cov := e.Coverage()
	if len(cov) != hosts {
		t.Errorf("coverage hosts = %d, want %d", len(cov), hosts)
	}

	var total int64
	for _, p := range e.Profiles() {
		total += p.Total
	}
	if total != int64(workers*perW) {
		t.Errorf("profile total = %d, want %d", total, workers*perW)
	}
}
