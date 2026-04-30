// Package events defines the canonical Event model that flows through the
// ingest pipeline, hot store, search index, and detection engine.
package events

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

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
	SourceUnknown Source = "unknown"
)

// Event is the canonical record. Every field is JSON-serialisable so the same
// shape moves over WAL, BadgerDB, REST, and the Wails bridge.
type Event struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenantId"`
	Timestamp  time.Time         `json:"timestamp"`
	ReceivedAt time.Time         `json:"receivedAt"`
	Source     Source            `json:"source"`
	HostID     string            `json:"hostId,omitempty"`
	EventType  string            `json:"eventType,omitempty"`
	Severity   Severity          `json:"severity,omitempty"`
	Message    string            `json:"message"`
	Raw        string            `json:"raw,omitempty"`
	Fields     map[string]string `json:"fields,omitempty"`
}

// Validate fills in derivable defaults and rejects malformed events.
func (e *Event) Validate() error {
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
	return nil
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
