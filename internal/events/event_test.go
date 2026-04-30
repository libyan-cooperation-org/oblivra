package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateSetsSchemaVersionAndHash(t *testing.T) {
	ev := &Event{Message: "hello"}
	if err := ev.Validate(); err != nil {
		t.Fatal(err)
	}
	if ev.SchemaVersion != SchemaVersion {
		t.Errorf("schemaVersion = %d", ev.SchemaVersion)
	}
	if ev.Hash == "" {
		t.Error("hash empty")
	}
	if !ev.VerifyHash() {
		t.Error("VerifyHash false right after Validate")
	}
}

func TestHashIsDeterministic(t *testing.T) {
	now := time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)
	mk := func() Event {
		return Event{
			TenantID:  "default",
			Timestamp: now,
			ReceivedAt: now,
			Source:    SourceREST,
			HostID:    "web-01",
			EventType: "failed_login",
			Severity:  SeverityWarn,
			Message:   "sshd Failed password for root",
			Fields:    map[string]string{"user": "root", "src": "10.0.0.1"},
			Provenance: Provenance{IngestPath: "rest", Peer: "127.0.0.1"},
			ID:        "fixed-id",
		}
	}
	a := mk()
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	b := mk()
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	if a.Hash != b.Hash {
		t.Errorf("hashes differ: %s vs %s", a.Hash, b.Hash)
	}
}

func TestHashSurvivesJSONRoundtrip(t *testing.T) {
	ev := &Event{Message: "hi", HostID: "x", Fields: map[string]string{"a": "1", "b": "2"}}
	if err := ev.Validate(); err != nil {
		t.Fatal(err)
	}
	original := ev.Hash

	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	var ev2 Event
	if err := json.Unmarshal(raw, &ev2); err != nil {
		t.Fatal(err)
	}
	if ev2.Hash != original {
		t.Errorf("stored hash drift: %s vs %s", ev2.Hash, original)
	}
	if !ev2.VerifyHash() {
		t.Errorf("VerifyHash false after roundtrip; computed %s, stored %s",
			ev2.ContentHash(), ev2.Hash)
	}
}

func TestMutationBreaksIdentity(t *testing.T) {
	ev := &Event{Message: "original"}
	if err := ev.Validate(); err != nil {
		t.Fatal(err)
	}
	if !ev.VerifyHash() {
		t.Fatal("VerifyHash false right after Validate")
	}
	// Tamper.
	ev.Message = "mutated"
	if ev.VerifyHash() {
		t.Fatal("VerifyHash should be false after mutation")
	}
}

func TestFieldOrderIndependence(t *testing.T) {
	// Two events with the same fields in different insertion orders must
	// produce the same hash because canonicalView sorts keys.
	a := &Event{
		Message: "x",
		Fields:  map[string]string{"alpha": "1", "beta": "2", "gamma": "3"},
		ID:      "same",
		Timestamp:  time.Unix(0, 0).UTC(),
		ReceivedAt: time.Unix(0, 0).UTC(),
		Source:     SourceREST,
		TenantID:   "default",
		Severity:   SeverityInfo,
	}
	b := &Event{
		Message: "x",
		Fields:  map[string]string{"gamma": "3", "alpha": "1", "beta": "2"},
		ID:      "same",
		Timestamp:  time.Unix(0, 0).UTC(),
		ReceivedAt: time.Unix(0, 0).UTC(),
		Source:     SourceREST,
		TenantID:   "default",
		Severity:   SeverityInfo,
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
	if a.Hash != b.Hash {
		t.Errorf("field-order changed hash: %s vs %s", a.Hash, b.Hash)
	}
}

func TestProvenanceMutationDetected(t *testing.T) {
	ev := &Event{Message: "x", Provenance: Provenance{IngestPath: "rest", Peer: "real-peer"}}
	if err := ev.Validate(); err != nil {
		t.Fatal(err)
	}
	ev.Provenance.Peer = "spoofed-peer"
	if ev.VerifyHash() {
		t.Fatal("provenance mutation should break hash")
	}
}

func TestRejectsEmptyEvent(t *testing.T) {
	ev := &Event{}
	if err := ev.Validate(); err == nil {
		t.Fatal("expected error for event without message or raw")
	}
}
