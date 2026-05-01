package httpserver

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	runtimemetrics "runtime/metrics"
	"strings"

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

		writeRuntimeMetrics(w)
	}
}

// writeRuntimeMetrics surfaces a curated set of Go-runtime metrics. We
// pick the ones that most often explain a latency regression: scheduler
// latency, GC pause distribution, heap headroom, and goroutine stack
// usage. The full set is much larger but adds noise; the operator can
// hit /debug/pprof for deeper digging.
func writeRuntimeMetrics(w io.Writer) {
	wanted := []struct {
		key, name, help, kind string
	}{
		{"/sched/latency:seconds", "oblivra_runtime_sched_latency_p99_seconds", "scheduler latency p99 (seconds spent in run queue)", "gauge"},
		{"/gc/pauses:seconds", "oblivra_runtime_gc_pause_p99_seconds", "GC stop-the-world pause p99 (seconds)", "gauge"},
		{"/memory/classes/heap/free:bytes", "oblivra_runtime_heap_free_bytes", "memory in heap that is free for reuse", "gauge"},
		{"/memory/classes/heap/objects:bytes", "oblivra_runtime_heap_objects_bytes", "memory in live heap objects", "gauge"},
		{"/memory/classes/heap/released:bytes", "oblivra_runtime_heap_released_bytes", "memory returned to the OS", "gauge"},
		{"/gc/heap/allocs:bytes", "oblivra_runtime_heap_allocs_bytes_total", "cumulative heap allocations in bytes", "counter"},
		{"/gc/heap/frees:bytes", "oblivra_runtime_heap_frees_bytes_total", "cumulative heap frees in bytes", "counter"},
		{"/sched/goroutines:goroutines", "oblivra_runtime_goroutines", "current goroutine count (runtime accounting)", "gauge"},
	}
	samples := make([]runtimemetrics.Sample, len(wanted))
	for i, w := range wanted {
		samples[i].Name = w.key
	}
	runtimemetrics.Read(samples)
	for i, s := range samples {
		val, ok := runtimeMetricValue(s)
		if !ok {
			continue
		}
		writeMetric(w, wanted[i].name, wanted[i].help, wanted[i].kind, val)
	}
}

// runtimeMetricValue extracts a single float from a metric sample —
// histograms collapse to their p99 bucket, scalars pass through. The
// runtime/metrics API exposes histograms as cumulative counts per bucket;
// p99 is the smallest bucket whose cumulative fraction reaches 0.99.
func runtimeMetricValue(s runtimemetrics.Sample) (float64, bool) {
	switch s.Value.Kind() {
	case runtimemetrics.KindUint64:
		return float64(s.Value.Uint64()), true
	case runtimemetrics.KindFloat64:
		return s.Value.Float64(), true
	case runtimemetrics.KindFloat64Histogram:
		h := s.Value.Float64Histogram()
		var total uint64
		for _, c := range h.Counts {
			total += c
		}
		if total == 0 {
			return 0, true
		}
		var cum uint64
		threshold := uint64(float64(total) * 0.99)
		for i, c := range h.Counts {
			cum += c
			if cum >= threshold {
				if i+1 < len(h.Buckets) {
					return h.Buckets[i+1], true
				}
				return h.Buckets[i], true
			}
		}
		return h.Buckets[len(h.Buckets)-1], true
	}
	return 0, false
}

var _ = strings.HasPrefix // reserved for future label dimensions

func writeMetric(w io.Writer, name, help, kind string, value float64) {
	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s %s\n", name, kind)
	fmt.Fprintf(w, "%s %g\n", name, value)
}
