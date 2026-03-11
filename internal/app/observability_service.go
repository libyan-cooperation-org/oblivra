package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/storage"
)

// ObservabilityService provides self-monitoring for OBLIVRA.
// It exposes pprof endpoints, watches goroutine counts, and
// detects potential deadlocks and resource anomalies.
type ObservabilityService struct {
	BaseService
	ctx      context.Context
	bus      *eventbus.Bus
	metrics  *monitoring.MetricsCollector
	hotStore *storage.HotStore
	log      *logger.Logger

	// Configuration
	GoroutineThreshold int           // Alert if goroutines exceed this
	CheckInterval      time.Duration // How often to check
	PprofAddr          string        // Address for pprof HTTP server (e.g. "127.0.0.1:6060")

	// State
	pprofServer      *http.Server
	goroutinePeak    int64
	alertSuppressed  atomic.Bool
	startTime        time.Time
	initialHeap      uint64
	initialGoroutine int
}

// Name returns the service name.
func (s *ObservabilityService) Name() string { return "ObservabilityService" }

// NewObservabilityService creates a new self-monitoring service.
func NewObservabilityService(bus *eventbus.Bus, metrics *monitoring.MetricsCollector, hotStore *storage.HotStore, log *logger.Logger) *ObservabilityService {
	return &ObservabilityService{
		bus:                bus,
		metrics:            metrics,
		hotStore:           hotStore,
		log:                log.WithPrefix("observability"),
		GoroutineThreshold: 1000,
		CheckInterval:      30 * time.Second,
		PprofAddr:          "127.0.0.1:6060",
	}
}

// Startup initialises pprof and starts background monitoring.
func (s *ObservabilityService) Startup(ctx context.Context) {
	s.ctx = ctx

	// Start pprof HTTP server (localhost only — never exposed externally)
	// Skip in test mode to avoid port conflicts/hangs
	if s.ctx != nil && s.ctx.Value("test") == "true" {
		s.log.Info("Skipping pprof server in test mode")
	} else {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		mux.HandleFunc("/debug/health", s.healthHandler)
		mux.HandleFunc("/debug/metrics", s.metricsHandler)

		s.pprofServer = &http.Server{
			Addr:    s.PprofAddr,
			Handler: mux,
		}

		go func() {
			s.log.Info("pprof server listening on %s", s.PprofAddr)
			if err := s.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.log.Warn("pprof server error: %v", err)
			}
		}()
	}

	// Enable mutex profiling for deadlock detection
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(1000)

	// Initialize stability baselines
	s.startTime = time.Now()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	s.initialHeap = m.HeapAlloc
	s.initialGoroutine = runtime.NumGoroutine()

	s.log.Info("ObservabilityService started — pprof on %s, goroutine threshold %d",
		s.PprofAddr, s.GoroutineThreshold)

	// Start background watchdog
	if s.ctx != nil {
		go s.watchdog()
	}
}

// Shutdown stops the pprof server and watchdog.
func (s *ObservabilityService) Shutdown() {
	if s.pprofServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.pprofServer.Shutdown(ctx)
	}
	s.log.Info("ObservabilityService stopped")
}

// GetStatus returns current system health metrics.
func (s *ObservabilityService) GetStatus() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"goroutines":      runtime.NumGoroutine(),
		"goroutine_peak":  atomic.LoadInt64(&s.goroutinePeak),
		"goroutine_limit": s.GoroutineThreshold,
		"heap_alloc_mb":   float64(m.HeapAlloc) / 1024 / 1024,
		"heap_sys_mb":     float64(m.HeapSys) / 1024 / 1024,
		"stack_sys_mb":    float64(m.StackSys) / 1024 / 1024,
		"gc_pause_ns":     m.PauseNs[(m.NumGC+255)%256],
		"gc_count":        m.NumGC,
		"num_cpu":         runtime.NumCPU(),
		"go_version":      runtime.Version(),
		"pprof_addr":      s.PprofAddr,
		"start_time":      s.startTime.Format(time.RFC3339),
	}
}

// ─── Internal ────────────────────────────────────────────

func (s *ObservabilityService) watchdog() {
	ticker := time.NewTicker(s.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkGoroutines()
			s.checkMemory()
			s.updateMetrics()
		}
	}
}

func (s *ObservabilityService) updateMetrics() {
	if s.metrics == nil {
		return
	}

	// Runtime metrics
	currGoroutines := runtime.NumGoroutine()
	s.metrics.SetGauge("goroutine_count", float64(currGoroutines), nil)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	s.metrics.SetGauge("memory_usage_bytes", float64(m.HeapAlloc), nil)

	// Stability Slope Calculation
	elapsedMin := time.Since(s.startTime).Minutes()
	if elapsedMin > 1 { // Only calculate slope after 1 minute of stabilization
		rssSlope := float64(int64(m.HeapAlloc)-int64(s.initialHeap)) / elapsedMin
		goSlope := float64(currGoroutines-s.initialGoroutine) / elapsedMin

		s.metrics.SetGauge("stability_rss_slope_bytes_min", rssSlope, nil)
		s.metrics.SetGauge("stability_goroutine_slope_min", goSlope, nil)
	}

	// Badger metrics
	if s.hotStore != nil {
		stats := s.hotStore.GetStats()
		for name, val := range stats {
			s.metrics.SetGauge(name, val, nil)
		}
	}
}

func (s *ObservabilityService) checkGoroutines() {
	count := runtime.NumGoroutine()
	current := int64(count)

	// Track peak
	for {
		peak := atomic.LoadInt64(&s.goroutinePeak)
		if current <= peak || atomic.CompareAndSwapInt64(&s.goroutinePeak, peak, current) {
			break
		}
	}

	if count > s.GoroutineThreshold {
		if !s.alertSuppressed.Load() {
			s.log.Warn("⚠ Goroutine watchdog: %d goroutines (threshold: %d)", count, s.GoroutineThreshold)
			s.bus.Publish("observability:goroutine_alert", map[string]interface{}{
				"count":     count,
				"threshold": s.GoroutineThreshold,
				"peak":      atomic.LoadInt64(&s.goroutinePeak),
			})
			s.alertSuppressed.Store(true)

			// Context-aware suppression reset (no goroutine leak on shutdown)
			go func() {
				timer := time.NewTimer(5 * time.Minute)
				defer timer.Stop()
				select {
				case <-timer.C:
					s.alertSuppressed.Store(false)
				case <-s.ctx.Done():
					return
				}
			}()
		}
	}
}

func (s *ObservabilityService) checkMemory() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	heapMB := float64(m.HeapAlloc) / 1024 / 1024

	// Alert on excessive heap usage (> 1GB)
	if heapMB > 1024 {
		s.log.Warn("⚠ Memory watchdog: heap at %.0fMB", heapMB)
		s.bus.Publish("observability:memory_alert", map[string]interface{}{
			"heap_mb":  heapMB,
			"gc_count": m.NumGC,
		})
	}
}

func (s *ObservabilityService) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := s.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","goroutines":%d,"heap_mb":%.1f}`,
		status["goroutines"], status["heap_alloc_mb"])
}

func (s *ObservabilityService) metricsHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "# HELP oblivra_goroutines Current goroutine count\n")
	fmt.Fprintf(w, "# TYPE oblivra_goroutines gauge\n")
	fmt.Fprintf(w, "oblivra_goroutines %d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "# HELP oblivra_goroutine_peak Peak goroutine count since start\n")
	fmt.Fprintf(w, "# TYPE oblivra_goroutine_peak gauge\n")
	fmt.Fprintf(w, "oblivra_goroutine_peak %d\n", atomic.LoadInt64(&s.goroutinePeak))
	fmt.Fprintf(w, "# HELP oblivra_heap_bytes Current heap allocation in bytes\n")
	fmt.Fprintf(w, "# TYPE oblivra_heap_bytes gauge\n")
	fmt.Fprintf(w, "oblivra_heap_bytes %d\n", m.HeapAlloc)
	fmt.Fprintf(w, "# HELP oblivra_gc_count Total GC runs\n")
	fmt.Fprintf(w, "# TYPE oblivra_gc_count counter\n")
	fmt.Fprintf(w, "oblivra_gc_count %d\n", m.NumGC)
	fmt.Fprintf(w, "# HELP oblivra_heap_sys_bytes Total heap memory obtained from OS\n")
	fmt.Fprintf(w, "# TYPE oblivra_heap_sys_bytes gauge\n")
	fmt.Fprintf(w, "oblivra_heap_sys_bytes %d\n", m.HeapSys)
}
