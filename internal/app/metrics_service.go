package app

import (
	"context"
	"net/http"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/monitoring"
)

// MetricsService exposes observability metrics to the frontend and acts as a Prometheus endpoint
type MetricsService struct {
	BaseService
	ctx              context.Context
	log              *logger.Logger
	metricsCollector *monitoring.MetricsCollector
	server           *http.Server
}

// Name returns the name of the service.
func (s *MetricsService) Name() string { return "MetricsService" }

// NewMetricsService creates a new MetricsService
func NewMetricsService(log *logger.Logger, metrics *monitoring.MetricsCollector) *MetricsService {
	return &MetricsService{
		log:              log.WithPrefix("metricsservice"),
		metricsCollector: metrics,
	}
}

// Startup is called at application startup
func (s *MetricsService) Startup(ctx context.Context) {
	s.ctx = ctx
	s.log.Info("Metrics Service started")

	if s.metricsCollector == nil {
		return
	}

	// Start a local HTTP server for Prometheus scraping on port 9090
	mux := http.NewServeMux()
	mux.Handle("/metrics", s.metricsCollector.PrometheusHandler())

	s.server = &http.Server{
		Addr:    "127.0.0.1:9090",
		Handler: mux,
	}

	go func() {
		s.log.Info("Starting Prometheus metrics server on http://127.0.0.1:9090/metrics")
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("Metrics server error: %v", err)
		}
	}()
}

func (s *MetricsService) Shutdown() {
	if s.server != nil {
		s.server.Close()
	}
}

// GetAllMetrics returns all current metrics formatted for the frontend
func (s *MetricsService) GetAllMetrics() []monitoring.Metric {
	if s.metricsCollector == nil {
		return nil
	}
	return s.metricsCollector.GetAll()
}

// Helper methods to let frontend trigger arbitrary metric bumps
func (s *MetricsService) IncrCounter(name string, labels map[string]string) {
	if s.metricsCollector == nil {
		return
	}
	s.metricsCollector.IncrCounter(name, labels)
}

func (s *MetricsService) RecordLatency(name string, ms float64, labels map[string]string) {
	if s.metricsCollector == nil {
		return
	}
	s.metricsCollector.ObserveHistogram(name, ms, labels)
}
