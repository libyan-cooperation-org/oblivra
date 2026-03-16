package monitoring

// otel.go — OpenTelemetry trace instrumentation for Oblivra
//
// Provides lightweight tracing helpers using only the OTel API packages
// (go.opentelemetry.io/otel and go.opentelemetry.io/otel/trace) which are
// already present as transitive dependencies via Wails.
//
// The full SDK (TracerProvider, stdout exporter, semconv) is wired in
// otel_sdk.go which is only compiled when the `otel_sdk` build tag is set:
//
//   go build -tags otel_sdk ./...
//
// Default production builds use the no-op provider from the OTel API.

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const serviceName = "oblivra"

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(serviceName + "/" + name)
}

// StartSpan starts a named span and returns the child context and span.
// The caller MUST call span.End() when the operation finishes.
func StartSpan(ctx context.Context, pkg, operation string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return Tracer(pkg).Start(ctx, operation, trace.WithAttributes(attrs...))
}

// RecordError marks the active span as failed and records the error.
func RecordError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// ── Attribute constructors ────────────────────────────────────────────────────

func HostAttr(hostID string) attribute.KeyValue     { return attribute.String("oblivra.host_id", hostID) }
func SessionAttr(id string) attribute.KeyValue      { return attribute.String("oblivra.session_id", id) }
func RuleAttr(ruleID string) attribute.KeyValue     { return attribute.String("oblivra.rule_id", ruleID) }
func TenantAttr(tenantID string) attribute.KeyValue { return attribute.String("oblivra.tenant_id", tenantID) }
func SeverityAttr(sev string) attribute.KeyValue    { return attribute.String("oblivra.severity", sev) }

// ── Convenience recorders ─────────────────────────────────────────────────────

func RecordDetectionMatch(ctx context.Context, mc *MetricsCollector, ruleID, severity string) {
	if mc != nil {
		mc.IncrCounter("detections_total", map[string]string{"severity": severity})
	}
	_, span := StartSpan(ctx, "detection", "RuleMatch", RuleAttr(ruleID), SeverityAttr(severity))
	span.SetStatus(codes.Ok, "rule matched")
	span.End()
}

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

func OblivraMetricsHandler(mc *MetricsCollector) http.Handler {
	return mc.PrometheusHandler()
}

func RegisterDetectionMetrics(mc *MetricsCollector) {
	mc.RegisterCounter("detections_total", "Total detection rule matches", nil)
	mc.RegisterCounter("detections_total", "Critical detections", map[string]string{"severity": "critical"})
	mc.RegisterCounter("detections_total", "High detections", map[string]string{"severity": "high"})
	mc.RegisterCounter("detections_total", "Medium detections", map[string]string{"severity": "medium"})
	mc.RegisterCounter("detections_total", "Low detections", map[string]string{"severity": "low"})
	mc.RegisterGauge("detection_rules_loaded", "Number of detection rules currently loaded", nil)
	mc.RegisterGauge("detection_rules_sigma", "Number of Sigma rules loaded", nil)
	mc.RegisterCounter("detection_sigma_transpile_errors", "Sigma rules that failed transpilation", nil)
	mc.RegisterHistogram("detection_event_processing_ms", "Time to process one event through all rules",
		[]float64{0.1, 0.5, 1, 5, 10, 50, 100, 500}, nil)
}

// InitTracing is a no-op in the default build (no SDK compiled in).
// Use build tag `otel_sdk` to enable the full TracerProvider + exporter.
func InitTracing() (func(interface{}) error, error) {
	return func(interface{}) error { return nil }, nil
}
