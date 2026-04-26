// Multi-output routing for the agent transport.
//
// Closes the "Multi-output routing" gap from the agent feature audit:
// production deployments commonly want a primary SIEM endpoint plus
// a backup endpoint (different region, different rack, different
// cloud) that the agent automatically falls over to when the primary
// is unreachable. Splunk Forwarder doesn't do this natively — they
// punt to a separate load-balancing tier.
//
// Design:
//
//   Config carries a slice of endpoints in priority order. The router
//   walks them on every Send and:
//
//     1. tries each endpoint in priority order until one succeeds.
//     2. tracks consecutive failures per endpoint.
//     3. when an endpoint exceeds `MaxConsecutiveFailures`, it's
//        demoted to the back of the rotation for the duration of
//        `DemotionWindow` so subsequent calls hit the healthy
//        endpoint first.
//     4. recovery is fast — a single successful Send resets the
//        failure counter, and a demoted endpoint is rehabilitated
//        once its window expires.
//
// Backwards-compat: when only one endpoint is configured (the legacy
// `ServerAddr` path), the router still works — it just has nothing
// to fall over to. Existing single-endpoint deployments behave
// identically.

package agent

import (
	"sort"
	"sync"
	"time"
)

// AgentOutput is one configured destination for an agent's events.
type AgentOutput struct {
	// URL is the base URL (e.g. https://siem-east.example.com).
	URL string
	// Priority — lower number wins. Ties broken by config order.
	Priority int
	// Label is a human-readable name for logging ("primary",
	// "backup-east"); optional, falls back to URL.
	Label string
}

// outputState carries runtime health for a single output.
type outputState struct {
	out                  AgentOutput
	consecutiveFailures  int
	lastFailureAt        time.Time
	demotedUntil         time.Time
}

// OutputRouter walks a configured list of outputs and tries them in
// priority order on every send. Failures are tracked so a flapping
// endpoint demotes itself out of the way of the healthy one.
type OutputRouter struct {
	mu                     sync.Mutex
	outputs                []*outputState
	MaxConsecutiveFailures int
	DemotionWindow         time.Duration
}

// NewOutputRouter returns a router over the given outputs, sorted
// by priority. An empty input is illegal (caller must always have
// at least the primary).
func NewOutputRouter(outs []AgentOutput) *OutputRouter {
	if len(outs) == 0 {
		return nil
	}
	sorted := make([]AgentOutput, len(outs))
	copy(sorted, outs)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})
	r := &OutputRouter{
		outputs:                make([]*outputState, len(sorted)),
		MaxConsecutiveFailures: 3,
		DemotionWindow:         60 * time.Second,
	}
	for i, o := range sorted {
		r.outputs[i] = &outputState{out: o}
	}
	return r
}

// Send tries every output in priority order, stopping on first
// success. Returns the URL that succeeded + any error from the LAST
// attempted output. Empty `try` is treated as a no-op.
//
// `try` is the per-endpoint operation — typically the existing HTTP
// post against `out.URL + "/api/v1/events"`. The router only wraps
// it with priority + failure tracking; encoding, compression, and
// signing stay in the caller.
func (r *OutputRouter) Send(try func(out AgentOutput) error) (string, error) {
	if r == nil {
		return "", nil
	}
	r.mu.Lock()
	candidates := r.orderForSend()
	r.mu.Unlock()

	var lastErr error
	for _, st := range candidates {
		err := try(st.out)
		r.mu.Lock()
		if err == nil {
			st.consecutiveFailures = 0
			st.demotedUntil = time.Time{}
			r.mu.Unlock()
			return st.out.URL, nil
		}
		st.consecutiveFailures++
		st.lastFailureAt = time.Now()
		if st.consecutiveFailures >= r.MaxConsecutiveFailures {
			st.demotedUntil = time.Now().Add(r.DemotionWindow)
		}
		r.mu.Unlock()
		lastErr = err
	}
	return "", lastErr
}

// orderForSend returns the outputs in the order they should be tried
// for the next Send. Demoted endpoints (recently flapping) are
// pushed to the back; once a demotion window expires, the endpoint
// is rehabilitated to its configured priority.
func (r *OutputRouter) orderForSend() []*outputState {
	now := time.Now()
	healthy := make([]*outputState, 0, len(r.outputs))
	demoted := make([]*outputState, 0)
	for _, st := range r.outputs {
		if !st.demotedUntil.IsZero() && now.Before(st.demotedUntil) {
			demoted = append(demoted, st)
			continue
		}
		// Demotion window expired — rehabilitate.
		if !st.demotedUntil.IsZero() && !now.Before(st.demotedUntil) {
			st.demotedUntil = time.Time{}
			st.consecutiveFailures = 0
		}
		healthy = append(healthy, st)
	}
	// Already sorted by priority within each bucket because the
	// underlying slice was sorted at construction; preserve that.
	return append(healthy, demoted...)
}

// Health returns a snapshot of per-output health for diagnostic UIs.
type OutputHealth struct {
	URL                 string
	Label               string
	Priority            int
	ConsecutiveFailures int
	Demoted             bool
	DemotedUntil        time.Time
}

func (r *OutputRouter) Health() []OutputHealth {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	out := make([]OutputHealth, 0, len(r.outputs))
	for _, st := range r.outputs {
		out = append(out, OutputHealth{
			URL:                 st.out.URL,
			Label:               st.out.Label,
			Priority:            st.out.Priority,
			ConsecutiveFailures: st.consecutiveFailures,
			Demoted:             !st.demotedUntil.IsZero() && now.Before(st.demotedUntil),
			DemotedUntil:        st.demotedUntil,
		})
	}
	return out
}
