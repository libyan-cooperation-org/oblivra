package events

import (
	"context"
	"time"
)

// EventProcessingContext enforces deterministic execution across all pipeline stages.
// Instead of calling time.Now() or rand.Uint64() inside pipeline processors,
// nodes must rely strictly on the context provided at ingestion.
type EventProcessingContext struct {
	EventID  string
	TenantID string
	Seed     uint64
	Now      time.Time
}

// TimeConfidence classifies how trustworthy an event's claimed timestamp is
// relative to server-side time at ingest. The pipeline tags every event with
// one of these values so detection rules and forensic queries can decide
// whether late-arriving or skewed events should be promoted, suppressed, or
// flagged for human review.
//
// Wire-format strings are stable — they appear in indexed event records and
// the audit log. Do not rename without a migration.
type TimeConfidence string

const (
	// TimeConfidenceNormal — timestamp is within ±60s of server time at ingest.
	TimeConfidenceNormal TimeConfidence = "normal"

	// TimeConfidenceLate — event timestamp is more than 60s in the past.
	// Could be a backfill / replayed batch, or evidence of a clock-skewed agent.
	TimeConfidenceLate TimeConfidence = "late"

	// TimeConfidenceSkewed — event timestamp is more than 60s in the future,
	// or more than 5min in the past. Strong indicator the agent's clock is wrong;
	// detection rules that key on time-windows should treat these with care.
	TimeConfidenceSkewed TimeConfidence = "skewed"

	// TimeConfidenceUnknown — server could not parse the timestamp; falls back
	// to ingest-time. Anything keyed on the original time is unreliable.
	TimeConfidenceUnknown TimeConfidence = "unknown"
)

// SovereignEvent is the universal event format for the Sovereign Terminal ingestion pipeline.
// It includes metadata, raw logs, and processing context.
type SovereignEvent struct {
	Id        string            `json:"id"`
	TenantID  string            `json:"tenant_id"`
	Timestamp string            `json:"timestamp"`
	Host      string            `json:"host"`
	EventType string            `json:"event_type"`
	SourceIp  string            `json:"source_ip"`
	User      string            `json:"user"`
	SessionId string            `json:"session_id"`
	RawLine   string            `json:"raw_line"`
	Version   int32             `json:"version"`
	Metadata  map[string]string `json:"metadata"`
	Signature string            `json:"signature"`
	IntegrityHash string       `json:"integrity_hash,omitempty"`
	IntegrityIndex int32       `json:"integrity_index,omitempty"`

	// EventTimeConfidence is set by the pipeline at ingest based on the
	// difference between Timestamp and server-side now(). See TimeConfidence.
	EventTimeConfidence TimeConfidence `json:"event_time_confidence,omitempty"`

	// SkewSeconds is the signed delta between the event's claimed timestamp
	// and server time at ingest. Positive = event in the future; negative =
	// event in the past. 0 if Timestamp could not be parsed.
	SkewSeconds int64 `json:"skew_seconds,omitempty"`

	// Context for tracing and cancellation across the pipeline
	Ctx context.Context `json:"-"`

	// Determinism and audit context
	ProcessingCtx EventProcessingContext `json:"processing_context"`
}

// ClassifyTime computes EventTimeConfidence + SkewSeconds for an event by
// comparing its Timestamp string against now. Pure function — safe to call
// from any pipeline stage without touching the wall clock.
//
// The thresholds match the ones documented on TimeConfidence:
//   - >60s in the future                 → skewed
//   - >5min in the past                  → skewed
//   - >60s in the past, ≤5min            → late
//   - within ±60s                        → normal
//   - unparseable timestamp              → unknown (skew=0)
func ClassifyTime(timestamp string, now time.Time) (TimeConfidence, int64) {
	if timestamp == "" {
		return TimeConfidenceUnknown, 0
	}
	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		// Fall back to RFC3339 (no fractional seconds)
		t, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return TimeConfidenceUnknown, 0
		}
	}
	delta := t.Sub(now)
	skew := int64(delta.Seconds())
	switch {
	case skew > 60:
		return TimeConfidenceSkewed, skew
	case skew < -300:
		return TimeConfidenceSkewed, skew
	case skew < -60:
		return TimeConfidenceLate, skew
	default:
		return TimeConfidenceNormal, skew
	}
}

func (e *SovereignEvent) GetId() string { return e.Id }
func (e *SovereignEvent) GetHost() string { return e.Host }
func (e *SovereignEvent) GetRawLine() string { return e.RawLine }
