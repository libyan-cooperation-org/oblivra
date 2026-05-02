package services

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/digitorus/timestamp"
)

// TestTSAService_DisabledByDefault makes sure NewTSAService with no
// URLs and no env var returns a service that's a clean no-op rather
// than crashing the platform.
func TestTSAService_DisabledByDefault(t *testing.T) {
	t.Setenv("OBLIVRA_TSA_URLS", "")
	t.Setenv("OBLIVRA_TSA_ANCHOR_DIR", "")
	svc := NewTSAService(slog.Default(), t.TempDir(), TSAOptions{})
	if svc.IsEnabled() {
		t.Fatal("expected disabled service when no URLs configured")
	}
	path, ts, hashHex, err := svc.TimestampDailyAnchor(context.Background(), "2026-01-01", "deadbeef")
	if err != nil {
		t.Fatalf("disabled stamp must not error, got %v", err)
	}
	if path != "" || hashHex != "" || !ts.IsZero() {
		t.Fatalf("disabled stamp should return zero values, got %q/%v/%q", path, ts, hashHex)
	}
}

// TestTSAService_PersistAnchorRoundTrip ensures the .tsr file lifecycle
// (write/read/path) works without involving any network.
func TestTSAService_PersistAnchorRoundTrip(t *testing.T) {
	dir := t.TempDir()
	svc := NewTSAService(slog.Default(), dir, TSAOptions{
		URLs:      []string{"http://example.invalid/tsa"},
		AnchorDir: filepath.Join(dir, "anchors"),
	})
	want := []byte{0xde, 0xad, 0xbe, 0xef}
	path, err := svc.PersistAnchor("2026-01-01", want)
	if err != nil {
		t.Fatalf("persist: %v", err)
	}
	if path != svc.AnchorPath("2026-01-01") {
		t.Fatalf("AnchorPath drift: got %q want %q", svc.AnchorPath("2026-01-01"), path)
	}
	got, err := svc.LoadAnchor("2026-01-01")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("round-trip mismatch: got %x want %x", got, want)
	}
}

// TestTSAService_RoundTrip spins up an in-process fake TSA, generates
// an ephemeral self-signed signing cert, has the fake TSA respond with
// a real PKCS#7-signed token over our digest, then verifies the
// platform parses it, persists it, and that VerifyTSAToken accepts the
// stored sidecar.
//
// This is the closest thing we can run offline to "is the wiring real".
func TestTSAService_RoundTrip(t *testing.T) {
	cert, signer := mustEphemeralTSA(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/timestamp-query" {
			t.Errorf("missing TSA content-type: got %q", got)
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req, err := timestamp.ParseRequest(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ts := &timestamp.Timestamp{
			HashAlgorithm:     req.HashAlgorithm,
			HashedMessage:     req.HashedMessage,
			Time:              time.Now().UTC(),
			Nonce:             req.Nonce,
			Policy:            asn1.ObjectIdentifier{2, 4, 5, 6}, // any OID; verifier ignores
			Ordering:          true,
			Accuracy:          time.Second,
			AddTSACertificate: true,
		}
		resp, err := ts.CreateResponse(cert, signer)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/timestamp-reply")
		_, _ = w.Write(resp)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	svc := NewTSAService(slog.Default(), dir, TSAOptions{
		URLs:      []string{srv.URL},
		AnchorDir: filepath.Join(dir, "anchors"),
	})
	if !svc.IsEnabled() {
		t.Fatal("service should be enabled with one URL")
	}

	rootHex := strings.Repeat("ab", 32) // any 64-char hex
	path, tsaTime, hashHex, err := svc.TimestampDailyAnchor(context.Background(), "2026-01-01", rootHex)
	if err != nil {
		t.Fatalf("TimestampDailyAnchor: %v", err)
	}
	if path == "" {
		t.Fatal("expected sidecar path")
	}
	if tsaTime.IsZero() {
		t.Fatal("expected non-zero tsaTime")
	}
	want := sha256.Sum256([]byte(rootHex))
	if hashHex != hex.EncodeToString(want[:]) {
		t.Fatalf("hashHex mismatch: got %q want %q", hashHex, hex.EncodeToString(want[:]))
	}

	tokenDER, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	got, err := VerifyTSAToken(tokenDER, want[:])
	if err != nil {
		t.Fatalf("VerifyTSAToken: %v", err)
	}
	if got.IsZero() {
		t.Fatal("VerifyTSAToken returned zero time")
	}

	// Tampered digest must reject.
	bad := make([]byte, len(want))
	copy(bad, want[:])
	bad[0] ^= 0xff
	if _, err := VerifyTSAToken(tokenDER, bad); err == nil {
		t.Fatal("expected mismatch error on tampered digest")
	}
}

// TestAudit_TimestampPendingAnchors_Idempotent covers the integration
// between AuditService and a (fake) TSA stamper: anchors get one token
// each, re-running the loop is a no-op.
func TestAudit_TimestampPendingAnchors_Idempotent(t *testing.T) {
	dir := t.TempDir()
	a, err := NewDurable(slog.Default(), dir, []byte("test-hmac-key"))
	if err != nil {
		t.Fatalf("NewDurable: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })

	// Seed two anchor entries on different days.
	a.Append(context.Background(), "system", "audit.daily-anchor", "default", map[string]string{
		"day": "2026-01-01", "root": "rootA", "entries": "1",
	})
	a.Append(context.Background(), "system", "audit.daily-anchor", "default", map[string]string{
		"day": "2026-01-02", "root": "rootB", "entries": "2",
	})

	stamper := &fakeStamper{enabled: true}
	if err := a.TimestampPendingAnchors(context.Background(), stamper); err != nil {
		t.Fatalf("first round: %v", err)
	}
	if got := stamper.callCount(); got != 2 {
		t.Fatalf("expected 2 stamps, got %d", got)
	}

	// Second pass should be a clean no-op.
	if err := a.TimestampPendingAnchors(context.Background(), stamper); err != nil {
		t.Fatalf("second round: %v", err)
	}
	if got := stamper.callCount(); got != 2 {
		t.Fatalf("idempotency broken: stamp count climbed to %d", got)
	}

	// And the audit chain must contain exactly two tsa-token follow-ups,
	// one per anchor day.
	got := map[string]int{}
	for _, e := range a.Recent(0) {
		if e.Action == "audit.tsa-token" {
			got[e.Detail["day"]]++
		}
	}
	if got["2026-01-01"] != 1 || got["2026-01-02"] != 1 {
		t.Fatalf("expected one tsa-token per anchor day, got %v", got)
	}
}

func TestAudit_TimestampPendingAnchors_DisabledStamperIsNoop(t *testing.T) {
	dir := t.TempDir()
	a, err := NewDurable(slog.Default(), dir, nil)
	if err != nil {
		t.Fatalf("NewDurable: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	a.Append(context.Background(), "system", "audit.daily-anchor", "default", map[string]string{
		"day": "2026-01-01", "root": "rootA",
	})
	if err := a.TimestampPendingAnchors(context.Background(), &fakeStamper{enabled: false}); err != nil {
		t.Fatalf("disabled stamper must not error, got %v", err)
	}
	for _, e := range a.Recent(0) {
		if e.Action == "audit.tsa-token" {
			t.Fatal("disabled stamper must not produce tsa-token entries")
		}
	}
}

// ---- helpers ----

type fakeStamper struct {
	mu      sync.Mutex
	enabled bool
	calls   []string
}

func (f *fakeStamper) IsEnabled() bool { return f.enabled }
func (f *fakeStamper) TimestampDailyAnchor(_ context.Context, day, root string) (string, time.Time, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, day)
	return "/tmp/" + day + ".tsr", time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), "hash-of-" + root, nil
}
func (f *fakeStamper) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// mustEphemeralTSA generates a self-signed RSA cert + private key that
// the digitorus library accepts as a TSA signer. Cert has the
// id-kp-timeStamping EKU set, which is required for TSA usage. RSA
// (not ECDSA) so we line up with the upstream library's tested
// signature algorithms.
func mustEphemeralTSA(t *testing.T) (*x509.Certificate, crypto.Signer) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa keygen: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "ephemeral-test-tsa"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse self-signed: %v", err)
	}
	return cert, priv
}

// pemDecodeForTest is here in case we ever want to dump intermediate
// debug artifacts from a failing test — kept unused but documented.
//
//nolint:unused // intentional escape hatch for ad-hoc debugging
func pemDecodeForTest(b []byte) []byte {
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil
	}
	return blk.Bytes
}
