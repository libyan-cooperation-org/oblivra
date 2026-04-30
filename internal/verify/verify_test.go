package verify

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/services"
)

func TestVerifyAuditLogClean(t *testing.T) {
	dir := t.TempDir()
	key := []byte("test-key")
	logger := slog.New(slog.NewTextHandler(silentWriter{}, nil))
	a, err := services.NewDurable(logger, dir, key)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		a.Append(context.Background(), "alice", "siem.search", "default", map[string]string{"q": "x"})
	}
	_ = a.Close()

	r, err := File(filepath.Join(dir, "audit.log"), key)
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK {
		t.Fatalf("expected ok, got broken at %d: %s", r.BrokenAt, r.BrokenWhy)
	}
	if r.Kind != "audit" {
		t.Errorf("kind = %q", r.Kind)
	}
	if r.Entries != 5 {
		t.Errorf("entries = %d", r.Entries)
	}
	if r.RootHash == "" {
		t.Error("root hash empty")
	}
}

func TestVerifyAuditLogTampered(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(silentWriter{}, nil))
	a, err := services.NewDurable(logger, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.Append(context.Background(), "x", "first", "default", nil)
	a.Append(context.Background(), "x", "second", "default", nil)
	_ = a.Close()

	path := filepath.Join(dir, "audit.log")
	body, _ := os.ReadFile(path)
	for i := 0; i < len(body)-5; i++ {
		if string(body[i:i+5]) == "first" {
			copy(body[i:], "FIRST")
			break
		}
	}
	_ = os.WriteFile(path, body, 0o600)

	r, err := File(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.OK {
		t.Fatal("verifier should detect tamper")
	}
	if r.BrokenAt != 1 {
		t.Errorf("brokenAt = %d", r.BrokenAt)
	}
}

func TestVerifyWAL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ingest.wal")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		ev := &events.Event{
			Source:  events.SourceREST,
			HostID:  "h",
			Message: "msg",
		}
		if err := ev.Validate(); err != nil {
			t.Fatal(err)
		}
		line, _ := json.Marshal(ev)
		f.Write(line)
		f.Write([]byte("\n"))
	}
	_ = f.Close()

	r, err := File(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != "wal" {
		t.Errorf("kind = %q", r.Kind)
	}
	if !r.OK || r.Entries != 3 {
		t.Errorf("expected 3 ok events, got entries=%d ok=%v", r.Entries, r.OK)
	}
}

func TestVerifyWALTampered(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ingest.wal")
	f, _ := os.Create(path)

	ev := &events.Event{Source: events.SourceREST, HostID: "h", Message: "original"}
	_ = ev.Validate()
	line, _ := json.Marshal(ev)
	f.Write(line)
	f.Write([]byte("\n"))
	_ = f.Close()

	// Corrupt the on-disk message but leave the recorded hash alone.
	body, _ := os.ReadFile(path)
	for i := 0; i < len(body)-8; i++ {
		if string(body[i:i+8]) == "original" {
			copy(body[i:], "tampered")
			break
		}
	}
	_ = os.WriteFile(path, body, 0o600)

	r, _ := File(path, nil)
	if r.OK {
		t.Fatal("verifier should reject mutated event")
	}
}

func TestVerifyEvidencePackage(t *testing.T) {
	dir := t.TempDir()
	key := []byte("ev-key")
	logger := slog.New(slog.NewTextHandler(silentWriter{}, nil))
	a, err := services.NewDurable(logger, dir, key)
	if err != nil {
		t.Fatal(err)
	}
	a.Append(context.Background(), "x", "first", "default", nil)
	a.Append(context.Background(), "x", "second", "default", nil)

	pkg, err := a.GeneratePackage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	pkgPath := filepath.Join(dir, "ev.json")
	body, _ := json.Marshal(pkg)
	_ = os.WriteFile(pkgPath, body, 0o600)
	_ = a.Close()

	r, err := File(pkgPath, key)
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK {
		t.Fatalf("evidence verify failed: %+v", r)
	}
	if r.SignatureOK == nil || !*r.SignatureOK {
		t.Error("signature should be valid")
	}
}

func TestVerifyUnknownArtifact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.txt")
	_ = os.WriteFile(path, []byte("hello plain text"), 0o600)
	if _, err := File(path, nil); err == nil {
		t.Error("expected error on unknown artifact")
	}
}

type silentWriter struct{}

func (silentWriter) Write(p []byte) (int, error) { return len(p), nil }
