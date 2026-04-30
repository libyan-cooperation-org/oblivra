package httpserver

import (
	"fmt"
	"io"
	"net/http"
	"runtime"

	"github.com/kingknull/oblivra/internal/services"
)

// metricsHandler renders OBLIVRA's runtime + ingest counters in Prometheus
// text exposition format. We don't pull in prometheus/client_golang to keep
// the headless binary tight — the format is simple enough to write by hand.
func metricsHandler(siem *services.SiemService, alerts *services.AlertService, fleet *services.FleetService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)

		writeMetric(w, "oblivra_goroutines", "current goroutine count", "gauge", float64(runtime.NumGoroutine()))
		writeMetric(w, "oblivra_heap_bytes", "process heap allocation in bytes", "gauge", float64(ms.HeapAlloc))
		writeMetric(w, "oblivra_gc_pause_ns_total", "cumulative GC pause time", "counter", float64(ms.PauseTotalNs))

		if siem != nil {
			s := siem.Stats()
			writeMetric(w, "oblivra_events_total", "total events ingested", "counter", float64(s.Total))
			writeMetric(w, "oblivra_events_eps", "current events-per-second (rolling 1s)", "gauge", float64(s.EPS))
			writeMetric(w, "oblivra_hot_events", "events currently in hot store", "gauge", float64(s.HotCount))
			writeMetric(w, "oblivra_wal_bytes", "WAL size in bytes", "gauge", float64(s.WAL.Bytes))
			writeMetric(w, "oblivra_wal_count", "WAL line count", "counter", float64(s.WAL.Count))
		}
		if alerts != nil {
			writeMetric(w, "oblivra_alerts_total", "total alerts raised", "counter", float64(alerts.Count()))
		}
		if fleet != nil {
			writeMetric(w, "oblivra_agents_registered", "total agents registered", "gauge", float64(len(fleet.List())))
		}
	}
}

func writeMetric(w io.Writer, name, help, kind string, value float64) {
	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s %s\n", name, kind)
	fmt.Fprintf(w, "%s %g\n", name, value)
}
