package dag

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/dlp"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// DLPNode runs the centralized DLP redactor against every event
// flowing through the ingest DAG, BEFORE it reaches the SIEM index
// or analytics sinks. Phase 27.2.3 — closes the "cloud logs bypass
// agent redaction" gap.
//
// Placement: insert this node between identity enrichment and the
// fanout (SIEM / analytics) so every downstream consumer sees the
// redacted form. The agent-side redactor still runs at the edge
// for events that originated on agents — DLPNode catches everything
// else (cloud connectors, manual ingest, REST API, etc.).
//
// Fields scrubbed (in place on the event):
//   - RawLine                — the canonical log line text
//   - User                   — sometimes carries email PII
//   - Metadata[*]            — arbitrary string metadata
//
// EventID, TenantID, Timestamp, Host, EventType, SourceIp, SessionId
// are NOT scrubbed — they're security signals (correlation, geo,
// integrity), and redacting them would break detection.
type DLPNode struct {
	BaseNode
	redactor *dlp.Redactor
	log      *logger.Logger
}

// NewDLPNode constructs a DLP node with the given redactor. Pass
// `nil` redactor to make the node a no-op (useful in tests / when a
// tenant disables DLP entirely from the Settings UI).
func NewDLPNode(r *dlp.Redactor, log *logger.Logger) *DLPNode {
	return &DLPNode{
		BaseNode: BaseNode{nodeName: "DLP_Redactor"},
		redactor: r,
		log:      log,
	}
}

// Process scrubs the event in place and forwards it. Errors are
// non-fatal — a redaction problem on one event must not poison the
// pipeline. We log at WARN and pass the event through untouched.
func (n *DLPNode) Process(_ context.Context, evt *Event) ([]*Event, error) {
	if n.redactor == nil || evt == nil {
		return []*Event{evt}, nil
	}

	if evt.RawLine != "" {
		evt.RawLine = n.redactor.Scrub(evt.RawLine)
	}
	if evt.User != "" {
		evt.User = n.redactor.Scrub(evt.User)
	}
	if evt.Metadata != nil {
		for k, v := range evt.Metadata {
			evt.Metadata[k] = n.redactor.Scrub(v)
		}
	}
	return []*Event{evt}, nil
}
