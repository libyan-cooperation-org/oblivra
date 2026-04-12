package events

import (
	"context"
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

	// Context for tracing and cancellation across the pipeline
	Ctx context.Context `json:"-"`
}

func (e *SovereignEvent) GetId() string { return e.Id }
func (e *SovereignEvent) GetHost() string { return e.Host }
func (e *SovereignEvent) GetRawLine() string { return e.RawLine }
