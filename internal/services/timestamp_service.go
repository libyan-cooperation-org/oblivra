// Package services — RFC 3161 Time-Stamp Protocol client for the daily
// Merkle anchor. The platform's audit chain is already cryptographically
// linked + HMAC-signed; what RFC 3161 adds is a third-party witness:
// every daily root is sent to one (or more) public Time Stamping
// Authorities, each replies with a PKCS#7-signed TSTInfo asserting
// "this hash existed before <UTC time>", and we persist that token as
// a sidecar file (and a hex-encoded reference inside the audit entry).
//
// Why we want it:
//
//   - the HMAC root only proves that whoever holds the audit key signed
//     this root, not WHEN it was signed. A motivated insider with key
//     access could backdate the system clock and re-sign.
//   - an RFC 3161 token from FreeTSA, DigiCert, Sectigo, etc. is a
//     PKCS#7 envelope signed by a publicly-known TSA cert chain. The
//     token's TSTInfo carries the TSA's own time, signed by the TSA's
//     private key. To forge the token an attacker would need the TSA's
//     private key — not just OBLIVRA's audit key.
//   - this turns daily anchors into court-grade evidence: a forensic
//     examiner re-running oblivra-verify can show that the daily root
//     and every event hashed underneath it were committed before the
//     timestamp on the TSA's PKCS#7 signature.
//
// Design notes:
//
//   - service is fire-and-forget — TSA reachability isn't required for
//     the platform to work. A daily anchor whose TSA round-trip fails
//     is still a valid anchor; the audit entry just won't carry a
//     `tsaToken` field for that day. Recoverable: the TSA fetcher will
//     retry the most recent anchor without a token on the next tick.
//   - tokens land at <dataDir>/audit/anchors/<YYYY-MM-DD>.tsr (DER bytes)
//     so they live alongside the audit log itself in evidence packages.
//   - URL list is configured via OBLIVRA_TSA_URLS (comma-separated). If
//     empty, the service is disabled (default).
//   - we deliberately do NOT verify the TSA cert chain at request time.
//     Verification happens at evidence-package build / oblivra-verify
//     time, against a CA bundle the operator chose to trust. This keeps
//     the request path simple and air-gap-survivable (you can ship the
//     token off-host even if your CA bundle here is stale).
//
// References: RFC 3161 §2.4 (TimeStampReq), §2.4.2 (TimeStampResp).
package services

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/digitorus/timestamp"
)

// TSAOptions configures the timestamp service. Pass an empty URLs slice
// to disable; the rest of the platform behaves identically — anchors
// just won't carry a tsaToken field.
type TSAOptions struct {
	// URLs is the ordered list of TSA endpoints to try. The first one
	// that returns a valid token wins. RFC 3161 supports HTTP POST with
	// Content-Type: application/timestamp-query.
	URLs []string

	// HTTPTimeout bounds the whole TSA request (DNS + connect + body).
	// Default 30s — TSAs are sometimes slow but rarely terrible.
	HTTPTimeout time.Duration

	// AnchorDir is where .tsr sidecar files are written. The verifier
	// looks for them at the same path. Default <dataDir>/audit/anchors.
	AnchorDir string

	// IncludeCerts asks the TSA to embed its certificate chain in the
	// PKCS#7 token. We default this to true so offline verification
	// has the cert it needs without a separate fetch.
	IncludeCerts bool
}

// TSAService is the public interface. Methods are safe for concurrent
// use. When URLs is empty, every method is a no-op except IsEnabled().
type TSAService struct {
	log    *slog.Logger
	opts   TSAOptions
	client *http.Client
	mu     sync.Mutex
}

// NewTSAService returns a configured service. Reads TSA URLs and
// AnchorDir from the explicit opts first, then falls back to env vars
// (OBLIVRA_TSA_URLS, OBLIVRA_TSA_ANCHOR_DIR) so operators can flip it on
// without code changes.
func NewTSAService(log *slog.Logger, dataDir string, opts TSAOptions) *TSAService {
	if len(opts.URLs) == 0 {
		if v := strings.TrimSpace(os.Getenv("OBLIVRA_TSA_URLS")); v != "" {
			for _, p := range strings.Split(v, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					opts.URLs = append(opts.URLs, p)
				}
			}
		}
	}
	if opts.AnchorDir == "" {
		if v := strings.TrimSpace(os.Getenv("OBLIVRA_TSA_ANCHOR_DIR")); v != "" {
			opts.AnchorDir = v
		} else if dataDir != "" {
			opts.AnchorDir = filepath.Join(dataDir, "audit", "anchors")
		}
	}
	if opts.HTTPTimeout == 0 {
		opts.HTTPTimeout = 30 * time.Second
	}
	if !opts.IncludeCerts {
		// default true — see comment on field.
		opts.IncludeCerts = true
	}
	return &TSAService{
		log:    log,
		opts:   opts,
		client: &http.Client{Timeout: opts.HTTPTimeout},
	}
}

// IsEnabled returns true if at least one TSA URL is configured.
func (t *TSAService) IsEnabled() bool {
	return t != nil && len(t.opts.URLs) > 0
}

// URLs exposes the configured endpoints (read-only) for the /status
// endpoint — operators want to see at a glance which TSAs the box has
// been wired to.
func (t *TSAService) URLs() []string {
	if t == nil {
		return nil
	}
	out := make([]string, len(t.opts.URLs))
	copy(out, t.opts.URLs)
	return out
}

// RequestToken takes a SHA-256 digest (raw bytes, not hex) and walks
// the configured TSA URLs in order. Returns the first DER-encoded
// PKCS#7 token returned by a TSA along with the TSA's asserted time.
//
// Each request carries a 128-bit cryptographic nonce so a replay-only
// attack against the TSA can't fool a verifier into accepting a stale
// timestamp.
func (t *TSAService) RequestToken(ctx context.Context, sha256Digest []byte) ([]byte, time.Time, error) {
	if !t.IsEnabled() {
		return nil, time.Time{}, errors.New("tsa: no URLs configured")
	}
	if len(sha256Digest) != sha256.Size {
		return nil, time.Time{}, fmt.Errorf("tsa: digest must be %d bytes (got %d)", sha256.Size, len(sha256Digest))
	}

	nonce, err := newNonce()
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("tsa: nonce: %w", err)
	}
	req := timestamp.Request{
		HashAlgorithm: crypto.SHA256,
		HashedMessage: sha256Digest,
		Certificates:  t.opts.IncludeCerts,
		Nonce:         nonce,
	}
	der, err := req.Marshal()
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("tsa: marshal request: %w", err)
	}

	var lastErr error
	for _, u := range t.opts.URLs {
		token, ts, err := t.postOne(ctx, u, der, nonce, sha256Digest)
		if err == nil {
			t.log.Debug("tsa: token acquired", "url", u, "tsaTime", ts.Format(time.RFC3339Nano))
			return token, ts, nil
		}
		t.log.Warn("tsa: request failed; trying next", "url", u, "err", err)
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("tsa: all URLs failed")
	}
	return nil, time.Time{}, lastErr
}

// postOne dispatches a single TSA POST and validates the round-trip
// (nonce echo + digest match). Returns the raw DER token bytes ready
// to persist as `audit.<day>.tsr`.
func (t *TSAService) postOne(ctx context.Context, url string, body []byte, nonce *big.Int, digest []byte) ([]byte, time.Time, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, time.Time{}, err
	}
	httpReq.Header.Set("Content-Type", "application/timestamp-query")
	httpReq.Header.Set("Accept", "application/timestamp-reply")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("tsa: read response: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, time.Time{}, fmt.Errorf("tsa: http %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	parsed, err := timestamp.ParseResponse(respBody)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("tsa: parse response: %w", err)
	}
	if !bytes.Equal(parsed.HashedMessage, digest) {
		return nil, time.Time{}, errors.New("tsa: response hash does not match request digest")
	}
	if parsed.Nonce != nil && nonce != nil && parsed.Nonce.Cmp(nonce) != 0 {
		return nil, time.Time{}, errors.New("tsa: response nonce mismatch (possible replay)")
	}
	return respBody, parsed.Time.UTC(), nil
}

// VerifyToken parses a previously-stored .tsr blob and confirms the
// embedded HashedMessage matches the supplied digest. The PKCS#7
// signature on the token is verified against whatever certificates the
// token itself carries (digitorus/timestamp does this in Parse). For
// air-gap-grade verification, callers should additionally pin the
// expected TSA cert/CA and check it via the returned Certificates
// slice.
//
// Returns the asserted TSA time on success.
func VerifyTSAToken(tokenDER, sha256Digest []byte) (time.Time, error) {
	if len(tokenDER) == 0 {
		return time.Time{}, errors.New("tsa: empty token")
	}
	parsed, err := timestamp.ParseResponse(tokenDER)
	if err != nil {
		// Some sidecars store just the inner PKCS#7 (Parse, not the
		// outer Response). Fall back to that.
		var perr error
		parsed, perr = timestamp.Parse(tokenDER)
		if perr != nil {
			return time.Time{}, fmt.Errorf("tsa: parse: %w", err)
		}
	}
	if !bytes.Equal(parsed.HashedMessage, sha256Digest) {
		return time.Time{}, errors.New("tsa: token hash does not match supplied digest")
	}
	return parsed.Time.UTC(), nil
}

// PersistAnchor writes the DER token to <AnchorDir>/<day>.tsr and
// returns the file path. Mode 0644 — the token is not secret and
// readability is what matters for offline verification.
func (t *TSAService) PersistAnchor(day string, tokenDER []byte) (string, error) {
	if t.opts.AnchorDir == "" {
		return "", errors.New("tsa: no anchor dir configured")
	}
	if err := os.MkdirAll(t.opts.AnchorDir, 0o750); err != nil {
		return "", fmt.Errorf("tsa: mkdir anchor: %w", err)
	}
	path := filepath.Join(t.opts.AnchorDir, day+".tsr")
	if err := os.WriteFile(path, tokenDER, 0o644); err != nil {
		return "", fmt.Errorf("tsa: write anchor: %w", err)
	}
	return path, nil
}

// LoadAnchor returns the DER bytes for a given day, or os.ErrNotExist.
func (t *TSAService) LoadAnchor(day string) ([]byte, error) {
	if t.opts.AnchorDir == "" {
		return nil, errors.New("tsa: no anchor dir configured")
	}
	path := filepath.Join(t.opts.AnchorDir, day+".tsr")
	return os.ReadFile(path)
}

// AnchorPath returns the path where an anchor for `day` is/would be.
// Used by the verifier and the evidence packager to find sidecars.
func (t *TSAService) AnchorPath(day string) string {
	if t.opts.AnchorDir == "" {
		return ""
	}
	return filepath.Join(t.opts.AnchorDir, day+".tsr")
}

// TimestampDailyAnchor is the high-level call the scheduler invokes
// after AuditService.AnchorYesterday writes its entry. It:
//
//   1. computes SHA-256 over the daily root hex string (the canonical
//      hash form an external auditor will check)
//   2. POSTs to a TSA, persists the token to <day>.tsr
//   3. returns (path, tsaTime, hexHashOfRoot) so the caller can append
//      a tsaToken / tsaTime / tsaPath entry to the audit chain.
//
// The audit entry mutation is done by the caller (we don't reach back
// into AuditService here to avoid a cyclic dependency).
//
// Returns (path, tsaTime, hashHex, err). All zero values when the TSA
// service is disabled — that's a soft-fail by design.
func (t *TSAService) TimestampDailyAnchor(ctx context.Context, day string, dailyRootHex string) (string, time.Time, string, error) {
	if !t.IsEnabled() {
		return "", time.Time{}, "", nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	// We hash the canonical hex form of the root so a verifier can
	// reproduce the digest from just the audit log without us having
	// to publish a separate "what we sent to the TSA" blob.
	sum := sha256.Sum256([]byte(dailyRootHex))
	tokenDER, ts, err := t.RequestToken(ctx, sum[:])
	if err != nil {
		return "", time.Time{}, "", err
	}
	path, err := t.PersistAnchor(day, tokenDER)
	if err != nil {
		return "", time.Time{}, "", err
	}
	return path, ts, hex.EncodeToString(sum[:]), nil
}

func newNonce() (*big.Int, error) {
	// 128 bits is more than enough — TSAs typically accept up to 160.
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, max)
}
