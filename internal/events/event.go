// Package events defines the canonical Event model that flows through the
// ingest pipeline, hot store, search index, and detection engine.
package events

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// SchemaVersion is bumped any time the Event struct changes shape on the wire
// or in the WAL. Migrators (cmd/migrate, future) read this to decide whether
// to upgrade in-place.
const SchemaVersion = 1

// Severity matches RFC 5424 numeric ranges loosely.
type Severity string

const (
	SeverityDebug    Severity = "debug"
	SeverityInfo     Severity = "info"
	SeverityNotice   Severity = "notice"
	SeverityWarn     Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
	SeverityAlert    Severity = "alert"
)

// Source classifies how the event entered the platform.
type Source string

const (
	SourceREST    Source = "rest"
	SourceSyslog  Source = "syslog"
	SourceAgent   Source = "agent"
	SourceFile    Source = "file"
	SourceImport  Source = "import"
	SourceUnknown Source = "unknown"
)

// Provenance records *how* the event entered the platform. It's hashed into
// the event's content hash so any retroactive mutation of provenance breaks
// identity. Fields are deliberately string-only so they can survive WAL
// replay across schema versions.
type Provenance struct {
	IngestPath     string `json:"ingestPath,omitempty"` // "rest", "syslog-udp", "agent", "raw", "import"
	Peer           string `json:"peer,omitempty"`       // remote address for REST/syslog
	AgentID        string `json:"agentId,omitempty"`
	Parser         string `json:"parser,omitempty"`         // "rfc5424" / "json" / "cef" / "rfc3164" / "" (none)
	TLSFingerprint string `json:"tlsFingerprint,omitempty"` // sha256 of peer cert (when mTLS in use)
	Format         string `json:"format,omitempty"`         // raw on-wire format ("syslog", "json", ...)
}

// Event is the canonical record. Every field is JSON-serialisable so the same
// shape moves over WAL, BadgerDB, REST, and the Wails bridge.
type Event struct {
	SchemaVersion int               `json:"schemaVersion"`
	ID            string            `json:"id"`
	Hash          string            `json:"hash"` // sha256 over canonical(event-with-empty-hash)
	TenantID      string            `json:"tenantId"`
	Timestamp     time.Time         `json:"timestamp"`
	ReceivedAt    time.Time         `json:"receivedAt"`
	Source        Source            `json:"source"`
	HostID        string            `json:"hostId,omitempty"`
	EventType     string            `json:"eventType,omitempty"`
	Severity      Severity          `json:"severity,omitempty"`
	Message       string            `json:"message"`
	Raw           string            `json:"raw,omitempty"`
	Fields        map[string]string `json:"fields,omitempty"`
	Provenance    Provenance        `json:"provenance,omitempty"`
}

// Validate fills in derivable defaults, rejects malformed events, and seals
// the event with a deterministic content hash. Calling Validate twice on the
// same event yields the same Hash (content hash is deterministic).
func (e *Event) Validate() error {
	if e.SchemaVersion == 0 {
		e.SchemaVersion = SchemaVersion
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = time.Now().UTC()
	}
	if e.Source == "" {
		e.Source = SourceUnknown
	}
	if e.TenantID == "" {
		e.TenantID = "default"
	}
	if e.Severity == "" {
		e.Severity = SeverityInfo
	}
	if e.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		e.ID = id
	}
	if e.Message == "" && e.Raw == "" {
		return errors.New("event: message or raw must be present")
	}
	e.Hash = e.ContentHash()
	return nil
}

// ContentHash returns the deterministic sha256 over a canonicalised view of
// the event with `Hash` cleared. The same event always produces the same
// hash, even after marshal/unmarshal round-trips.
func (e *Event) ContentHash() string {
	canon := canonicalView(e)
	b, _ := json.Marshal(canon)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// VerifyHash returns true if the event's stored Hash matches the recomputed
// one. A `false` return means someone (or something) mutated a field after
// the event was sealed.
func (e *Event) VerifyHash() bool {
	if e.Hash == "" {
		return false
	}
	return e.Hash == e.ContentHash()
}

// canonicalView produces a stable shape for hashing — Hash and any mutable
// derivative fields are cleared, and Fields is sorted by key so JSON's
// map-iteration order doesn't affect the digest.
type canonEvent struct {
	SchemaVersion int            `json:"schemaVersion"`
	ID            string         `json:"id"`
	TenantID      string         `json:"tenantId"`
	Timestamp     string         `json:"timestamp"`
	ReceivedAt    string         `json:"receivedAt"`
	Source        Source         `json:"source"`
	HostID        string         `json:"hostId"`
	EventType     string         `json:"eventType"`
	Severity      Severity       `json:"severity"`
	Message       string         `json:"message"`
	Raw           string         `json:"raw"`
	Fields        []canonField   `json:"fields"`
	Provenance    Provenance     `json:"provenance"`
}

type canonField struct {
	K string `json:"k"`
	V string `json:"v"`
}

func canonicalView(e *Event) canonEvent {
	keys := make([]string, 0, len(e.Fields))
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]canonField, 0, len(keys))
	for _, k := range keys {
		out = append(out, canonField{K: k, V: e.Fields[k]})
	}
	return canonEvent{
		SchemaVersion: e.SchemaVersion,
		ID:            e.ID,
		TenantID:      e.TenantID,
		// RFC3339Nano is round-trip stable through JSON marshal/unmarshal.
		Timestamp:  e.Timestamp.UTC().Format(time.RFC3339Nano),
		ReceivedAt: e.ReceivedAt.UTC().Format(time.RFC3339Nano),
		Source:     e.Source,
		HostID:     e.HostID,
		EventType:  e.EventType,
		Severity:   e.Severity,
		Message:    e.Message,
		Raw:        e.Raw,
		Fields:     out,
		Provenance: e.Provenance,
	}
}

// MarshalLine serialises the event as a single newline-terminated JSON line.
// Used by the WAL.
func (e *Event) MarshalLine() ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}
	return append(b, '\n'), nil
}

func newID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
