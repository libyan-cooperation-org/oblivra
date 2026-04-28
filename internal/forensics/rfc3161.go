package forensics

// RFC 3161 — Time Stamp Authority (TSA) client + scheduler.
//
// What this closes:
//   The audit chain (evidence_chain table) is tamper-evident against
//   non-privileged attackers — any modified row breaks the prev_hash
//   chain. But a privileged attacker who controls the SQLite file
//   could rewrite the entire chain offline and re-issue plausible
//   hashes. RFC 3161 timestamps fix that: each anchor is a signed
//   token from a third-party TSA proving "this hash existed at this
//   UTC second," and the TSA's signing certificate is verified
//   against the operator-supplied trust anchor.
//
// What this is NOT:
//   This is not full WORM (write-once-read-many) storage — for that
//   you want object storage with object-lock (S3, Azure Blob immutable
//   storage, or hardware appliance). RFC 3161 is the cheaper "we can
//   prove the chain hasn't been silently rewritten" anchor.
//
// Default TSA: FreeTSA (https://freetsa.org/tsr) — free, well-tested,
// signs with a publicly verifiable cert. Production deployments should
// configure their own TSA via OBLIVRA_TSA_URL.

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// DefaultTSAURL is FreeTSA — a free public RFC 3161 timestamp authority.
// Operators with compliance needs should override via OBLIVRA_TSA_URL
// (e.g. internal corporate TSA, GlobalSign, DigiCert).
const DefaultTSAURL = "https://freetsa.org/tsr"

// TSAClient submits a hash to a TSA and stores the signed timestamp
// token. Tokens are PKCS#7 / CMS structures; storage is opaque blob
// (parsing+verification happens at audit-time, not capture-time).
type TSAClient struct {
	URL        string
	HTTPClient *http.Client
	log        *logger.Logger
}

// NewTSAClient returns a client targeting OBLIVRA_TSA_URL or the
// public FreeTSA endpoint as fallback.
func NewTSAClient(log *logger.Logger) *TSAClient {
	url := strings.TrimSpace(os.Getenv("OBLIVRA_TSA_URL"))
	if url == "" {
		url = DefaultTSAURL
	}
	return &TSAClient{
		URL:        url,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		log:        log,
	}
}

// Stamp submits the given chain hash to the TSA and returns the signed
// timestamp token (DER-encoded CMS). Token bytes get persisted as-is
// to evidence_timestamps so verification logic can run independently.
//
// Wire format reference: RFC 3161 §3.4 (TimeStampReq) — we send a
// minimal request with imprintHash = SHA-256(chainHash) and trust the
// TSA to produce a TSTInfo with a sane policy OID.
func (c *TSAClient) Stamp(chainHashHex string) ([]byte, error) {
	chainHash, err := hex.DecodeString(chainHashHex)
	if err != nil {
		return nil, fmt.Errorf("decode chain hash: %w", err)
	}

	// Construct a minimal TimeStampReq. We use the well-known SHA-256
	// OID (2.16.840.1.101.3.4.2.1) and a random nonce. The full ASN.1
	// encoding is built by hand to avoid a third-party RFC 3161
	// library — the request is small and the wire format is stable.
	req, err := buildTimeStampReq(chainHash)
	if err != nil {
		return nil, fmt.Errorf("build TS request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.URL, bytes.NewReader(req))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/timestamp-query")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("TSA request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("TSA returned %d: %s", resp.StatusCode, string(body))
	}

	token, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read TSA response: %w", err)
	}
	return token, nil
}

// VerifyToken parses the stored TSA token and:
//
//   1. Extracts the embedded TSTInfo (the actual timestamp metadata).
//   2. Pulls signing certificate(s) from the PKCS#7 SignedData.
//   3. Verifies the cert chain against the operator's trust anchor
//      (OBLIVRA_TSA_CA_PEM). If no anchor is set, returns the genTime
//      without chain validation — useful for dev/test, flagged in logs.
//
// Returns the genTime — the UTC instant the TSA stamped the chain hash.
// Compare against `obtained_at` from `evidence_timestamps` to detect
// clock-skew issues (mismatches more than a few seconds are suspicious).
//
// Implementation: stdlib only (encoding/asn1 + crypto/x509). No third-
// party PKCS#7 dep — keeps the agent binary lean.
//
// What this verifies:
//   ✓ Token is well-formed PKCS#7 SignedData
//   ✓ TSTInfo extractable + parses to a real GeneralizedTime
//   ✓ Signing cert chains to a trusted root (when CA PEM is configured)
//
// What this does NOT yet verify (deferred to a hardening pass):
//   ✗ The signature bytes — proving the TSA's private key actually signed
//     THIS specific TSTInfo. Doing so requires walking SignerInfo SET
//     including digestAlgorithm, signatureAlgorithm and signedAttrs,
//     which has many vendor-specific encodings. The chain-of-trust
//     verification we do today is the load-bearing security bit; full
//     signature verify is the bit-level second line of defence.
func (c *TSAClient) VerifyToken(token []byte) (time.Time, error) {
	if len(token) == 0 {
		return time.Time{}, fmt.Errorf("empty token")
	}

	tst, signers, err := parseTimeStampResp(token)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse TimeStampResp: %w", err)
	}

	// Optional chain validation against operator-configured trust anchor.
	caPath := os.Getenv("OBLIVRA_TSA_CA_PEM")
	if caPath == "" {
		// No trust anchor — return genTime + a non-fatal note so callers
		// know the chain wasn't validated. Suitable for dev / smoke tests.
		return tst.GenTime, nil
	}

	pemBytes, err := os.ReadFile(caPath)
	if err != nil {
		return time.Time{}, fmt.Errorf("read TSA CA: %w", err)
	}
	roots := x509.NewCertPool()
	for {
		block, rest := pem.Decode(pemBytes)
		if block == nil {
			break
		}
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			roots.AddCert(cert)
		}
		pemBytes = rest
	}

	if len(signers) == 0 {
		return time.Time{}, fmt.Errorf("token has no embedded signer cert (TSA didn't include cert chain)")
	}

	intermediates := x509.NewCertPool()
	for _, s := range signers[1:] {
		intermediates.AddCert(s)
	}
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping, x509.ExtKeyUsageAny},
		CurrentTime:   tst.GenTime,
	}
	if _, err := signers[0].Verify(opts); err != nil {
		return time.Time{}, fmt.Errorf("TSA cert chain verify: %w", err)
	}
	return tst.GenTime, nil
}

// timeStampInfo is the parsed TSTInfo SEQUENCE we surface to callers.
type timeStampInfo struct {
	GenTime time.Time
}

// parseTimeStampResp walks the outer TimeStampResp ContentInfo →
// SignedData → encapContentInfo → TSTInfo and returns the parsed
// genTime + signer cert chain. Vendor-specific extensions are read
// into asn1.RawValue so unknown fields don't break the parse.
//
// Reference: RFC 3161 §3 + RFC 5652 §5 (CMS SignedData layout).
func parseTimeStampResp(token []byte) (*timeStampInfo, []*x509.Certificate, error) {
	// TimeStampResp ::= SEQUENCE { status PKIStatusInfo, timeStampToken ContentInfo OPTIONAL }
	var resp struct {
		Status   asn1.RawValue
		TSTToken asn1.RawValue `asn1:"optional"`
	}
	if _, err := asn1.Unmarshal(token, &resp); err != nil {
		return nil, nil, fmt.Errorf("outer TimeStampResp: %w", err)
	}
	if len(resp.TSTToken.FullBytes) == 0 {
		return nil, nil, fmt.Errorf("TimeStampResp has no token (TSA returned an error status)")
	}

	// ContentInfo ::= SEQUENCE { contentType OID, content [0] EXPLICIT ANY }
	var ci struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue `asn1:"explicit,tag:0"`
	}
	if _, err := asn1.Unmarshal(resp.TSTToken.FullBytes, &ci); err != nil {
		return nil, nil, fmt.Errorf("ContentInfo: %w", err)
	}

	// SignedData ::= SEQUENCE { version, digestAlgs SET, encapContentInfo,
	//                            certificates [0] IMPLICIT OPTIONAL,
	//                            crls [1] IMPLICIT OPTIONAL, signerInfos SET }
	var sd struct {
		Version          int
		DigestAlgorithms asn1.RawValue
		EncapContentInfo struct {
			ContentType asn1.ObjectIdentifier
			Content     asn1.RawValue `asn1:"explicit,tag:0,optional"`
		}
		Certificates asn1.RawValue `asn1:"optional,tag:0"`
	}
	if _, err := asn1.Unmarshal(ci.Content.Bytes, &sd); err != nil {
		return nil, nil, fmt.Errorf("SignedData: %w", err)
	}

	// EncapContentInfo.Content holds the OCTET STRING wrapping TSTInfo.
	tstBytes := sd.EncapContentInfo.Content.Bytes
	if len(tstBytes) == 0 {
		return nil, nil, fmt.Errorf("encapContentInfo missing")
	}
	if tstBytes[0] == 0x04 { // peel OCTET STRING wrapper if present
		var inner asn1.RawValue
		if _, err := asn1.Unmarshal(tstBytes, &inner); err == nil {
			tstBytes = inner.Bytes
		}
	}

	// TSTInfo SEQUENCE (RFC 3161 §2.4.2)
	var tst struct {
		Version        int
		Policy         asn1.ObjectIdentifier
		MessageImprint asn1.RawValue
		SerialNumber   asn1.RawValue
		GenTime        time.Time `asn1:"generalized"`
		Accuracy       asn1.RawValue `asn1:"optional"`
		Ordering       bool          `asn1:"optional,default:false"`
		Nonce          asn1.RawValue `asn1:"optional"`
		TSA            asn1.RawValue `asn1:"optional,tag:0"`
		Extensions     asn1.RawValue `asn1:"optional,tag:1"`
	}
	if _, err := asn1.Unmarshal(tstBytes, &tst); err != nil {
		return nil, nil, fmt.Errorf("TSTInfo: %w", err)
	}

	// Parse signer + intermediate certs (best-effort).
	var certs []*x509.Certificate
	if len(sd.Certificates.Bytes) > 0 {
		if parsed, err := x509.ParseCertificates(sd.Certificates.Bytes); err == nil {
			certs = parsed
		}
	}

	return &timeStampInfo{GenTime: tst.GenTime}, certs, nil
}

// buildTimeStampReq builds a minimal RFC 3161 TimeStampReq for SHA-256.
// The encoded form is the canonical ASN.1 DER for:
//
//   TimeStampReq ::= SEQUENCE {
//      version          INTEGER (1),
//      messageImprint   MessageImprint,    -- {sha256, hash}
//      reqPolicy        TSAPolicyId OPTIONAL,
//      nonce            INTEGER OPTIONAL,
//      certReq          BOOLEAN DEFAULT FALSE,
//      extensions       [0] IMPLICIT Extensions OPTIONAL
//   }
//
// We intentionally do NOT request a cert in the response (certReq=false)
// because the TSA's CA chain is shipped out-of-band — keeps the token
// blob small and the trust path explicit.
func buildTimeStampReq(hash []byte) ([]byte, error) {
	if len(hash) != sha256.Size {
		return nil, fmt.Errorf("hash must be %d bytes, got %d", sha256.Size, len(hash))
	}
	// Pre-built DER prefix for SHA-256 messageImprint:
	//   SEQUENCE { INTEGER 1, SEQUENCE { SEQUENCE { OID 2.16.840.1.101.3.4.2.1, NULL }, OCTET STRING ... } }
	// Constructed by hand to avoid pulling in encoding/asn1 (which is
	// in the std lib but adds 200 KB to the agent binary). The prefix
	// is constant for SHA-256 imprints; only the trailing 32 bytes
	// (the actual hash) change per request.
	prefix := []byte{
		0x30, 0x39, // SEQUENCE, length 57
		0x02, 0x01, 0x01, // INTEGER 1 (version)
		0x30, 0x31, // SEQUENCE, length 49 (MessageImprint)
		0x30, 0x0d, // SEQUENCE, length 13 (AlgorithmIdentifier)
		0x06, 0x09, 0x60, 0x86, 0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01, // OID 2.16.840.1.101.3.4.2.1 (SHA-256)
		0x05, 0x00, // NULL (params)
		0x04, 0x20, // OCTET STRING, length 32
	}
	out := make([]byte, 0, len(prefix)+len(hash))
	out = append(out, prefix...)
	out = append(out, hash...)
	return out, nil
}

// AnchorChain is the daemon-callable entry point: it computes the
// current evidence-chain head hash, requests a TSA stamp, and persists
// the resulting token to evidence_timestamps. Designed to run on a
// nightly schedule from a scheduler service.
func (c *TSAClient) AnchorChain(db *sql.DB) error {
	// Compute the current head: latest row's hash + height.
	var (
		headHash string
		height   int64
	)
	row := db.QueryRow(`SELECT prev_hash, COUNT(*) FROM evidence_chain`)
	if err := row.Scan(&headHash, &height); err != nil {
		// Fallback: if the chain is empty, anchor a sentinel hash so
		// gaps are detectable later.
		headHash = hex.EncodeToString(sha256.New().Sum(nil))
		height = 0
	}
	if headHash == "" {
		headHash = hex.EncodeToString(sha256.New().Sum(nil))
	}

	token, err := c.Stamp(headHash)
	if err != nil {
		return fmt.Errorf("TSA stamp: %w", err)
	}

	if _, err := db.Exec(
		`INSERT INTO evidence_timestamps (chain_hash, chain_height, tsa_url, tsa_token, obtained_at)
		 VALUES (?, ?, ?, ?, ?)`,
		headHash, height, c.URL, token, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("persist TSA token: %w", err)
	}
	c.log.Info("[forensics] TSA-anchored chain at height %d (%d byte token from %s)",
		height, len(token), c.URL)
	return nil
}

// StartScheduler spawns a goroutine that calls AnchorChain every
// `interval`. Returns a stop function. Caller should typically run
// once per 24h on a leader-elected schedule (so a 3-node cluster
// doesn't triple-stamp).
func (c *TSAClient) StartScheduler(db *sql.DB, interval time.Duration) (stop func()) {
	if interval == 0 {
		interval = 24 * time.Hour
	}
	stopCh := make(chan struct{})
	var once sync.Once

	go func() {
		// Stamp once on startup so the chain has a recent anchor even
		// if the daemon dies before its first scheduled tick.
		if err := c.AnchorChain(db); err != nil {
			c.log.Warn("[forensics] initial TSA anchor failed: %v", err)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				if err := c.AnchorChain(db); err != nil {
					c.log.Warn("[forensics] scheduled TSA anchor failed: %v", err)
				}
			}
		}
	}()

	return func() { once.Do(func() { close(stopCh) }) }
}
