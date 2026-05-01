package httpserver

import (
	"net/http"
	"net/http/pprof"
	"strings"
)

// Phase 47 — runtime profiling.
//
// pprof is invaluable for diagnosing latency / leak / goroutine-storm
// incidents in production. We expose the standard handlers under
// /debug/pprof/* but only when the caller carries operator-level auth.
// The auth middleware sits in front of the mux, so any request that
// reaches these handlers has already been authenticated.
//
// We route through one wrapper rather than registering each handler
// individually because the stdlib's pprof.Index dispatches by URL path
// to the per-profile handlers (heap, goroutine, allocs, etc.), and the
// router needs the full /debug/pprof/* prefix preserved.
func registerPprof(mux *http.ServeMux) {
	mux.Handle("GET /debug/pprof/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip our prefix so pprof's internal switch sees /<profile>.
		switch {
		case strings.HasSuffix(r.URL.Path, "/cmdline"):
			pprof.Cmdline(w, r)
		case strings.HasSuffix(r.URL.Path, "/profile"):
			pprof.Profile(w, r)
		case strings.HasSuffix(r.URL.Path, "/symbol"):
			pprof.Symbol(w, r)
		case strings.HasSuffix(r.URL.Path, "/trace"):
			pprof.Trace(w, r)
		default:
			pprof.Index(w, r)
		}
	}))
}
