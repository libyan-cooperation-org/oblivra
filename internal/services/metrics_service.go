package services

import (
	"context"
	"net/http"
	"os"

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
func (s *MetricsService) Name() string { return "metrics-service" }

// Dependencies returns service dependencies
func (s *MetricsService) Dependencies() []string {
	return []string{}
}

// NewMetricsService creates a new MetricsService
func NewMetricsService(log *logger.Logger, metrics *monitoring.MetricsCollector) *MetricsService {
	return &MetricsService{
		log:              log.WithPrefix("metricsservice"),
		metricsCollector: metrics,
	}
}

// Startup is called at application startup
func (s *MetricsService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("Metrics Service started")

	if s.metricsCollector == nil {
		return nil
	}

	// Start a local HTTP server for Prometheus scraping.
	// Allow overriding the port via environment variable (useful for tests to avoid port 9090 contention).
	port := os.Getenv("OBLIVRA_METRICS_PORT")
	if port == "" {
		port = "9090"
	}
	addr := "127.0.0.1:" + port

	mux := http.NewServeMux()
	mux.Handle("/metrics", s.metricsCollector.PrometheusHandler())

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		s.log.Info("Starting Prometheus metrics server on http://%s/metrics", addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("Metrics server error: %v", err)
		}
	}()
	return nil
}

func (s *MetricsService) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
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
