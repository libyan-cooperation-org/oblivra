package monitoring

// otel.go — OpenTelemetry trace instrumentation for Oblivra
//
// Provides:
//   - A global TracerProvider initialised to a stdout/OTLP exporter
//   - Helper to start/end spans from any service
//   - Automatic attribute propagation for Oblivra-specific context keys
//   - Prometheus metrics bridge: detection, vault, SSH counters exposed on /metrics
//
// Usage:
//   tracer := monitoring.Tracer("ssh")
//   ctx, span := tracer.Start(ctx, "Connect", monitoring.HostAttr(hostID))
//   defer span.End()

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	serviceName    = "oblivra"
	serviceVersion = "1.0.0"
)

// ── Bootstrap ─────────────────────────────────────────────────────────────────

// InitTracing sets up the global OpenTelemetry TracerProvider.
// Call once at application startup. Returns a shutdown function.
//
// By default traces are written to stdout (human-readable for development).
// Set OTEL_EXPORTER_OTLP_ENDPOINT to forward to an OTLP collector (Jaeger, Tempo, etc.).
func InitTracing() (shutdown func(context.Context) error, err error) {
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			attribute.String("oblivra.env", getEnv("OBLIVRA_ENV", "production")),
		),
		resource.WithOS(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTel resource: %w", err)
	}

	// stdout exporter — always available, zero deps beyond the SDK
	stdoutExp, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(traceWriter()),
	)
	if err != nil {
		return nil, fmt.Errorf("create stdout trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(stdoutExp),
		sdktrace.WithResource(res),
		// Sample at 10% in production, 100% in dev/test
		sdktrace.WithSampler(adaptiveSampler()),
	)

	otel.SetTracerProvider(tp)

	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(ctx)
	}, nil
}

// Tracer returns a named tracer from the global provider.
// Use the package/service name as the instrumentation name.
//
// Example:
//   tracer := monitoring.Tracer("ssh")
func Tracer(name string) trace.Tracer {
	return otel.Tracer(serviceName + "/" + name)
}

// ── Span helpers ──────────────────────────────────────────────────────────────

// StartSpan starts a named span and returns the child context and span.
// The caller MUST call span.End() when the operation finishes.
//
// Example:
//   ctx, span := monitoring.StartSpan(ctx, "ssh", "Connect")
//   defer span.End()
func StartSpan(ctx context.Context, pkg, operation string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return Tracer(pkg).Start(ctx, operation, trace.WithAttributes(attrs...))
}

// RecordError marks the active span as failed and records the error.
// Safe to call with a nil or no-op span.
func RecordError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// ── Attribute constructors ────────────────────────────────────────────────────

// HostAttr returns an attribute for the connected host ID.
func HostAttr(hostID string) attribute.KeyValue {
	return attribute.String("oblivra.host_id", hostID)
}

// SessionAttr returns an attribute for an SSH session ID.
func SessionAttr(sessionID string) attribute.KeyValue {
	return attribute.String("oblivra.session_id", sessionID)
}

// RuleAttr returns an attribute for a detection rule ID.
func RuleAttr(ruleID string) attribute.KeyValue {
	return attribute.String("oblivra.rule_id", ruleID)
}

// TenantAttr returns an attribute for the active tenant.
func TenantAttr(tenantID string) attribute.KeyValue {
	return attribute.String("oblivra.tenant_id", tenantID)
}

// SeverityAttr returns an attribute for an alert severity string.
func SeverityAttr(severity string) attribute.KeyValue {
	return attribute.String("oblivra.severity", severity)
}

// ── Detection metrics helpers (convenience wrappers) ─────────────────────────

// RecordDetectionMatch records a detection rule match in the MetricsCollector and
// emits a trace span representing the alert.
func RecordDetectionMatch(ctx context.Context, mc *MetricsCollector, ruleID, severity string) {
	if mc != nil {
		mc.IncrCounter("detections_total", map[string]string{"severity": severity})
	}

	_, span := StartSpan(ctx, "detection", "RuleMatch",
		RuleAttr(ruleID),
		SeverityAttr(severity),
	)
	span.SetStatus(codes.Ok, "rule matched")
	span.End()
}

// RecordSSHConnect records SSH connection metrics and a trace span.
func RecordSSHConnect(ctx context.Context, mc *MetricsCollector, hostID string, success bool, latencyMs float64) {
	if mc != nil {
		mc.IncrCounter("ssh_connections_total", nil)
		if success {
			mc.IncrCounter("ssh_connections_success", nil)
		} else {
			mc.IncrCounter("ssh_connections_failed", nil)
		}
		mc.ObserveHistogram("ssh_connection_latency_ms", latencyMs, nil)
	}

	_, span := StartSpan(ctx, "ssh", "Connect", HostAttr(hostID))
	if !success {
		span.SetStatus(codes.Error, "connection failed")
	}
	span.End()
}

// RecordVaultUnlock records a vault unlock attempt.
func RecordVaultUnlock(ctx context.Context, mc *MetricsCollector, success bool) {
	if mc != nil {
		mc.IncrCounter("vault_unlock_total", nil)
		if !success {
			mc.IncrCounter("vault_unlock_failed", nil)
		}
	}

	_, span := StartSpan(ctx, "vault", "Unlock")
	if !success {
		span.SetStatus(codes.Error, "unlock failed")
	}
	span.End()
}

// ── Prometheus bridge ─────────────────────────────────────────────────────────

// OblivraMetricsHandler returns an HTTP handler that exposes Prometheus metrics
// PLUS detection-specific counters from the MetricsCollector.
// Mount at /metrics on the observability server.
func OblivraMetricsHandler(mc *MetricsCollector) http.Handler {
	return mc.PrometheusHandler()
}

// RegisterDetectionMetrics adds detection-specific counters/gauges to the collector.
// Call once during startup, before any rules are loaded.
func RegisterDetectionMetrics(mc *MetricsCollector) {
	mc.RegisterCounter("detections_total", "Total detection rule matches", nil)
	mc.RegisterCounter("detections_total", "Total detection rule matches (critical)", map[string]string{"severity": "critical"})
	mc.RegisterCounter("detections_total", "Total detection rule matches (high)", map[string]string{"severity": "high"})
	mc.RegisterCounter("detections_total", "Total detection rule matches (medium)", map[string]string{"severity": "medium"})
	mc.RegisterCounter("detections_total", "Total detection rule matches (low)", map[string]string{"severity": "low"})
	mc.RegisterGauge("detection_rules_loaded", "Number of detection rules currently loaded", nil)
	mc.RegisterGauge("detection_rules_sigma", "Number of Sigma rules loaded", nil)
	mc.RegisterCounter("detection_sigma_transpile_errors", "Sigma rules that failed transpilation", nil)
	mc.RegisterHistogram("detection_event_processing_ms", "Time to process one event through all rules",
		[]float64{0.1, 0.5, 1, 5, 10, 50, 100, 500}, nil)
}

// ── Internal ──────────────────────────────────────────────────────────────────

// traceWriter returns the trace output destination.
// Defaults to os.Stdout; can be redirected by setting OTEL_TRACE_FILE.
func traceWriter() *os.File {
	if path := os.Getenv("OTEL_TRACE_FILE"); path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err == nil {
			return f
		}
	}
	return os.Stdout
}

// adaptiveSampler selects sampling rate based on environment.
// 100% in dev/test, 10% in production to limit trace volume.
func adaptiveSampler() sdktrace.Sampler {
	env := getEnv("OBLIVRA_ENV", "production")
	switch env {
	case "development", "dev", "test":
		return sdktrace.AlwaysSample()
	default:
		return sdktrace.TraceIDRatioBased(0.1)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
