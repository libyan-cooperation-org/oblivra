package monitoring

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MetricType defines the kind of metric
type MetricType string

const (
	MetricCounter   MetricType = "counter"
	MetricGauge     MetricType = "gauge"
	MetricHistogram MetricType = "histogram"
)

// Metric represents a single metric reading
type Metric struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels,omitempty"`
	Description string            `json:"description,omitempty"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type metricMeta struct {
	description string
	metricType  MetricType
	labels      map[string]string
}

// Histogram tracks a distribution of values
type Histogram struct {
	mu      sync.Mutex
	buckets []float64
	counts  []int64
	sum     float64
	count   int64
}

// MetricsCollector centralized metric gathering
type MetricsCollector struct {
	mu         sync.RWMutex
	counters   map[string]*int64
	gauges     map[string]*float64
	histograms map[string]*Histogram
	metadata   map[string]metricMeta
}

func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		counters:   make(map[string]*int64),
		gauges:     make(map[string]*float64),
		histograms: make(map[string]*Histogram),
		metadata:   make(map[string]metricMeta),
	}
	mc.registerDefaultMetrics()
	return mc
}

func (mc *MetricsCollector) registerDefaultMetrics() {
	// Connection metrics
	mc.RegisterCounter("ssh_connections_total", "Total SSH connections attempted", nil)
	mc.RegisterCounter("ssh_connections_success", "Successful SSH connections", nil)
	mc.RegisterCounter("ssh_connections_failed", "Failed SSH connections", nil)
	mc.RegisterGauge("ssh_connections_active", "Currently active SSH connections", nil)

	// Session metrics
	mc.RegisterCounter("sessions_total", "Total sessions created", nil)
	mc.RegisterGauge("sessions_active", "Currently active sessions", nil)
	mc.RegisterHistogram("session_duration_seconds", "Session duration distribution",
		[]float64{60, 300, 900, 1800, 3600, 7200, 14400, 28800}, nil)

	// Tunnel metrics
	mc.RegisterCounter("tunnels_total", "Total tunnels created", nil)
	mc.RegisterGauge("tunnels_active", "Currently active tunnels", nil)
	mc.RegisterCounter("tunnel_bytes_transferred", "Total bytes through tunnels", nil)

	// Vault metrics
	mc.RegisterCounter("vault_unlock_total", "Total vault unlock attempts", nil)
	mc.RegisterCounter("vault_unlock_failed", "Failed vault unlock attempts", nil)
	mc.RegisterGauge("vault_entries_count", "Number of vault entries", nil)

	// Performance metrics
	mc.RegisterHistogram("ssh_connection_latency_ms", "SSH connection latency",
		[]float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000}, nil)
	mc.RegisterGauge("memory_usage_bytes", "Application memory usage", nil)
	mc.RegisterGauge("goroutine_count", "Number of active goroutines", nil)

	// Badger metrics
	mc.RegisterGauge("badger_lsm_size_bytes", "Size of the LSM tree in bytes", nil)
	mc.RegisterGauge("badger_vlog_size_bytes", "Size of the value log in bytes", nil)

	// Stability Slope metrics
	mc.RegisterGauge("stability_rss_slope_bytes_min", "Average heap memory growth per minute", nil)
	mc.RegisterGauge("stability_goroutine_slope_min", "Average goroutine growth per minute", nil)
}

// RegisterCounter registers a new counter metric
func (mc *MetricsCollector) RegisterCounter(name, description string, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := metricKey(name, labels)
	v := int64(0)
	mc.counters[key] = &v
	mc.metadata[key] = metricMeta{
		description: description,
		metricType:  MetricCounter,
		labels:      labels,
	}
}

// RegisterGauge registers a new gauge metric
func (mc *MetricsCollector) RegisterGauge(name, description string, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := metricKey(name, labels)
	v := float64(0)
	mc.gauges[key] = &v
	mc.metadata[key] = metricMeta{
		description: description,
		metricType:  MetricGauge,
		labels:      labels,
	}
}

// RegisterHistogram registers a new histogram metric
func (mc *MetricsCollector) RegisterHistogram(name, description string, buckets []float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := metricKey(name, labels)
	sort.Float64s(buckets)
	mc.histograms[key] = &Histogram{
		buckets: buckets,
		counts:  make([]int64, len(buckets)+1), // +1 for +Inf
	}
	mc.metadata[key] = metricMeta{
		description: description,
		metricType:  MetricHistogram,
		labels:      labels,
	}
}

// IncrCounter increments a counter
func (mc *MetricsCollector) IncrCounter(name string, labels map[string]string) {
	key := metricKey(name, labels)
	mc.mu.RLock()
	counter, ok := mc.counters[key]
	mc.mu.RUnlock()
	if ok {
		atomic.AddInt64(counter, 1)
	}
}

// AddCounter adds a value to a counter
func (mc *MetricsCollector) AddCounter(name string, value int64, labels map[string]string) {
	key := metricKey(name, labels)
	mc.mu.RLock()
	counter, ok := mc.counters[key]
	mc.mu.RUnlock()
	if ok {
		atomic.AddInt64(counter, value)
	}
}

// SetGauge sets a gauge value
func (mc *MetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	key := metricKey(name, labels)
	mc.mu.Lock()
	if gauge, ok := mc.gauges[key]; ok {
		*gauge = value
	}
	mc.mu.Unlock()
}

// ObserveHistogram records a value in a histogram
func (mc *MetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	key := metricKey(name, labels)
	mc.mu.RLock()
	hist, ok := mc.histograms[key]
	mc.mu.RUnlock()
	if !ok {
		return
	}

	hist.mu.Lock()
	defer hist.mu.Unlock()

	hist.sum += value
	hist.count++

	for i, bucket := range hist.buckets {
		if value <= bucket {
			hist.counts[i]++
			return
		}
	}
	hist.counts[len(hist.buckets)]++ // +Inf bucket
}

// GetAll returns all current metric values
func (mc *MetricsCollector) GetAll() []Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var metrics []Metric

	for key, counter := range mc.counters {
		meta := mc.metadata[key]
		metrics = append(metrics, Metric{
			Name:        extractName(key),
			Type:        MetricCounter,
			Value:       float64(atomic.LoadInt64(counter)),
			Labels:      meta.labels,
			Description: meta.description,
			UpdatedAt:   time.Now(),
		})
	}

	for key, gauge := range mc.gauges {
		meta := mc.metadata[key]
		metrics = append(metrics, Metric{
			Name:        extractName(key),
			Type:        MetricGauge,
			Value:       *gauge,
			Labels:      meta.labels,
			Description: meta.description,
			UpdatedAt:   time.Now(),
		})
	}

	return metrics
}

// PrometheusHandler returns an HTTP handler that serves Prometheus metrics
func (mc *MetricsCollector) PrometheusHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mc.mu.RLock()
		defer mc.mu.RUnlock()

		var sb strings.Builder

		// Counters
		for key, counter := range mc.counters {
			meta := mc.metadata[key]
			name := extractName(key)
			sb.WriteString(fmt.Sprintf("# HELP %s %s\n", name, meta.description))
			sb.WriteString(fmt.Sprintf("# TYPE %s counter\n", name))
			sb.WriteString(fmt.Sprintf("%s%s %d\n", name, formatLabels(meta.labels), atomic.LoadInt64(counter)))
		}

		// Gauges
		for key, gauge := range mc.gauges {
			meta := mc.metadata[key]
			name := extractName(key)
			sb.WriteString(fmt.Sprintf("# HELP %s %s\n", name, meta.description))
			sb.WriteString(fmt.Sprintf("# TYPE %s gauge\n", name))
			sb.WriteString(fmt.Sprintf("%s%s %g\n", name, formatLabels(meta.labels), *gauge))
		}

		// Histograms
		for key, hist := range mc.histograms {
			meta := mc.metadata[key]
			name := extractName(key)
			hist.mu.Lock()

			sb.WriteString(fmt.Sprintf("# HELP %s %s\n", name, meta.description))
			sb.WriteString(fmt.Sprintf("# TYPE %s histogram\n", name))

			cumulative := int64(0)
			for i, bucket := range hist.buckets {
				cumulative += hist.counts[i]
				sb.WriteString(fmt.Sprintf("%s_bucket{le=\"%g\"}%s %d\n", name, bucket, formatLabels(meta.labels), cumulative))
			}
			cumulative += hist.counts[len(hist.buckets)]
			sb.WriteString(fmt.Sprintf("%s_bucket{le=\"+Inf\"}%s %d\n", name, formatLabels(meta.labels), cumulative))
			sb.WriteString(fmt.Sprintf("%s_sum%s %g\n", name, formatLabels(meta.labels), hist.sum))
			sb.WriteString(fmt.Sprintf("%s_count%s %d\n", name, formatLabels(meta.labels), hist.count))

			hist.mu.Unlock()
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Write([]byte(sb.String()))
	})
}

func metricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(parts)
	return name + "{" + strings.Join(parts, ",") + "}"
}

func extractName(key string) string {
	if idx := strings.Index(key, "{"); idx > 0 {
		return key[:idx]
	}
	return key
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, v))
	}
	sort.Strings(parts)
	return "{" + strings.Join(parts, ",") + "}"
}
