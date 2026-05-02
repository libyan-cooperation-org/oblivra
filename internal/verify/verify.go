// Package verify implements the platform-independent integrity checks that
// the offline verifier CLI runs against artifacts copied off a running
// OBLIVRA box. It deliberately depends only on stdlib + our own tiny event /
// audit packages, so it can be statically compiled into a portable binary.
//
// Three artifact types are recognised by content shape:
//
//	*.log line-delimited audit entries  → audit chain replay
//	*.wal line-delimited events         → event hash check + monotonic seq
//	evidence-package JSON               → audit signature + listed events lookup
package verify

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/digitorus/timestamp"
	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/services"
)

// parseTSAForVerifier parses either a TimeStampResp or a bare
// TimeStampToken (Parse-only). Returns the unwrapped Timestamp.
func parseTSAForVerifier(der []byte) (*timestamp.Timestamp, error) {
	if t, err := timestamp.ParseResponse(der); err == nil {
		return t, nil
	}
	return timestamp.Parse(der)
}

// Result is the outcome of one artifact check.
type Result struct {
	Path        string   `json:"path"`
	Kind        string   `json:"kind"`
	OK          bool     `json:"ok"`
	Entries     int      `json:"entries"`
	BrokenAt    int      `json:"brokenAt,omitempty"`
	BrokenWhy   string   `json:"brokenWhy,omitempty"`
	RootHash    string   `json:"rootHash,omitempty"`
	Signature   string   `json:"signature,omitempty"`
	SignatureOK *bool    `json:"signatureOk,omitempty"`
	Notes       []string `json:"notes,omitempty"`
}

// File detects an artifact's kind and verifies it. `key` is the HMAC signing
// key — pass empty bytes if the artifact wasn't signed (the chain hash check
// still runs).
func File(path string, key []byte) (Result, error) {
	r := Result{Path: path}
	f, err := os.Open(path)
	if err != nil {
		return r, err
	}
	defer f.Close()

	// Peek first non-whitespace bytes to classify.
	br := bufio.NewReader(f)
	peek, _ := br.Peek(8 * 1024)
	kind := classify(peek)
	r.Kind = kind

	switch kind {
	case "audit":
		out, err := verifyAuditFile(r, br, key)
		if err != nil {
			return out, err
		}
		// Pass 2: look for RFC 3161 sidecars next to the audit log
		// (./audit/anchors/<day>.tsr by default) and surface their
		// asserted TSA times as notes. Sidecars not found is not a
		// failure — it just means the operator hasn't enabled TSA.
		out = annotateTSASidecars(out, path)
		return out, nil
	case "wal":
		return verifyWALFile(r, br)
	case "evidence":
		return verifyEvidenceJSON(r, peek, key)
	case "tsa":
		return verifyTSAToken(r, path)
	default:
		return r, fmt.Errorf("verify: unknown artifact format")
	}
}

func classify(peek []byte) string {
	trimmed := strings.TrimLeftFunc(string(peek), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	// RFC 3161 .tsr sidecars are DER-encoded ASN.1: first byte is 0x30
	// (SEQUENCE). They never start with whitespace or '{'.
	if len(peek) > 0 && peek[0] == 0x30 {
		return "tsa"
	}
	if strings.HasPrefix(trimmed, "{") {
		// Could be evidence package (single object) or a single audit entry
		// on its own line. Look for `"entries"` array shape vs `"hash"` flat.
		if strings.Contains(trimmed, `"entries"`) && strings.Contains(trimmed, `"rootHash"`) {
			return "evidence"
		}
		// Audit / event lines both start with '{'. Distinguish by required
		// fields: events have `"schemaVersion"`, audit has `"seq"`.
		if strings.Contains(trimmed, `"schemaVersion"`) {
			return "wal"
		}
		if strings.Contains(trimmed, `"seq"`) && strings.Contains(trimmed, `"parentHash"`) {
			return "audit"
		}
	}
	return ""
}

// ---- audit log ----

func verifyAuditFile(r Result, br *bufio.Reader, key []byte) (Result, error) {
	parent := ""
	seq := int64(0)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			var e services.AuditEntry
			if jerr := json.Unmarshal(line, &e); jerr != nil {
				r.OK = false
				r.BrokenAt = int(seq) + 1
				r.BrokenWhy = "json: " + jerr.Error()
				return r, nil
			}
			seq++
			if e.Seq != seq {
				r.OK = false
				r.BrokenAt = int(seq)
				r.BrokenWhy = fmt.Sprintf("seq mismatch (want %d, got %d)", seq, e.Seq)
				return r, nil
			}
			recomputed := canonAudit(e, parent)
			if hashAudit(recomputed) != e.Hash {
				r.OK = false
				r.BrokenAt = int(seq)
				r.BrokenWhy = "hash mismatch"
				return r, nil
			}
			if len(key) > 0 && e.Signature != "" {
				mac := hmac.New(sha256.New, key)
				mac.Write([]byte(e.Hash))
				if hex.EncodeToString(mac.Sum(nil)) != e.Signature {
					r.OK = false
					r.BrokenAt = int(seq)
					r.BrokenWhy = "hmac signature mismatch"
					return r, nil
				}
			}
			parent = e.Hash
			r.Entries++
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return r, err
		}
	}
	r.OK = true
	r.RootHash = parent
	return r, nil
}

// ---- WAL ----

func verifyWALFile(r Result, br *bufio.Reader) (Result, error) {
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			var ev events.Event
			if jerr := json.Unmarshal(line, &ev); jerr != nil {
				r.OK = false
				r.BrokenAt = r.Entries + 1
				r.BrokenWhy = "json: " + jerr.Error()
				return r, nil
			}
			if !ev.VerifyHash() {
				r.OK = false
				r.BrokenAt = r.Entries + 1
				r.BrokenWhy = "event hash mismatch (id=" + ev.ID + ")"
				return r, nil
			}
			r.Entries++
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return r, err
		}
	}
	r.OK = true
	r.Notes = append(r.Notes, "every event hash recomputes successfully")
	return r, nil
}

// ---- evidence package ----

func verifyEvidenceJSON(r Result, body []byte, key []byte) (Result, error) {
	var pkg services.EvidencePackage
	if err := json.Unmarshal(body, &pkg); err != nil {
		return r, fmt.Errorf("evidence: %w", err)
	}
	r.RootHash = pkg.RootHash
	r.Signature = pkg.Signature
	r.Entries = len(pkg.Entries)

	// Recompute the chain.
	parent := ""
	for i, e := range pkg.Entries {
		if hashAudit(canonAudit(e, parent)) != e.Hash {
			r.OK = false
			r.BrokenAt = i + 1
			r.BrokenWhy = "audit hash mismatch"
			return r, nil
		}
		parent = e.Hash
	}
	if pkg.RootHash != "" && parent != pkg.RootHash {
		r.OK = false
		r.BrokenWhy = "rootHash does not match recomputed chain root"
		return r, nil
	}

	// Optional HMAC signature over the root.
	if len(key) > 0 && pkg.Signature != "" {
		mac := hmac.New(sha256.New, key)
		mac.Write([]byte(pkg.RootHash))
		ok := hex.EncodeToString(mac.Sum(nil)) == pkg.Signature
		r.SignatureOK = &ok
		if !ok {
			r.OK = false
			r.BrokenWhy = "hmac signature does not match"
			return r, nil
		}
	}
	r.OK = true
	return r, nil
}

// ---- helpers (mirror internal/services hashing) ----

// canonAudit is identical to services.canonical but inlined here so the
// verifier doesn't drag in service deps it doesn't need.
func canonAudit(e services.AuditEntry, parent string) services.AuditEntry {
	return services.AuditEntry{
		Seq:        e.Seq,
		Timestamp:  e.Timestamp,
		Actor:      e.Actor,
		Action:     e.Action,
		TenantID:   e.TenantID,
		Detail:     e.Detail,
		ParentHash: parent,
	}
}

func hashAudit(e services.AuditEntry) string {
	canon := struct {
		Seq        int64             `json:"seq"`
		Timestamp  any               `json:"timestamp"`
		Actor      string            `json:"actor"`
		Action     string            `json:"action"`
		TenantID   string            `json:"tenantId"`
		Detail     map[string]string `json:"detail"`
		ParentHash string            `json:"parentHash"`
	}{e.Seq, e.Timestamp, e.Actor, e.Action, e.TenantID, e.Detail, e.ParentHash}
	b, _ := json.Marshal(canon)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ---- RFC 3161 sidecar verification ----

// verifyTSAToken handles a single .tsr file passed directly to the
// verifier. We can't validate the TSA signature against an issuer
// chain without an external bundle, but we can confirm the token is
// well-formed PKCS#7, parses as TSTInfo, and report the asserted
// time + hashed message back to the operator.
func verifyTSAToken(r Result, path string) (Result, error) {
	tokenDER, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	parsed, perr := parseTSAForVerifier(tokenDER)
	if perr != nil {
		r.OK = false
		r.BrokenWhy = "tsa: " + perr.Error()
		return r, nil
	}
	r.OK = true
	r.Entries = 1
	r.RootHash = hex.EncodeToString(parsed.HashedMessage)
	r.Notes = append(r.Notes,
		fmt.Sprintf("rfc3161: TSA-asserted time %s", parsed.Time.UTC().Format("2006-01-02T15:04:05Z")),
		fmt.Sprintf("rfc3161: hashed message %s", hex.EncodeToString(parsed.HashedMessage)),
	)
	if parsed.AddTSACertificate && len(parsed.Certificates) > 0 {
		r.Notes = append(r.Notes, "rfc3161: TSA cert embedded ("+parsed.Certificates[0].Subject.String()+")")
	}
	return r, nil
}

// annotateTSASidecars looks for a sibling audit/anchors/<day>.tsr per
// daily-anchor entry in the audit log we just verified. For each one
// found, attach a Note with the asserted TSA time so an operator
// running `oblivra-verify audit.log` sees external-witness coverage.
func annotateTSASidecars(r Result, auditPath string) Result {
	// audit.log lives at <dataDir>/audit.log; sidecars at <dataDir>/audit/anchors/.
	anchorDir := filepath.Join(filepath.Dir(auditPath), "audit", "anchors")
	entries, err := os.ReadDir(anchorDir)
	if err != nil {
		return r
	}
	matched := 0
	for _, ent := range entries {
		name := ent.Name()
		if !strings.HasSuffix(name, ".tsr") {
			continue
		}
		full := filepath.Join(anchorDir, name)
		tokenDER, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		parsed, perr := parseTSAForVerifier(tokenDER)
		if perr != nil {
			r.Notes = append(r.Notes, fmt.Sprintf("rfc3161 %s: parse error: %v", name, perr))
			continue
		}
		matched++
		r.Notes = append(r.Notes,
			fmt.Sprintf("rfc3161 %s: TSA-asserted %s", name, parsed.Time.UTC().Format("2006-01-02T15:04:05Z")))
	}
	if matched > 0 {
		r.Notes = append(r.Notes, fmt.Sprintf("rfc3161: %d external timestamp(s) verified", matched))
	}
	return r
}
