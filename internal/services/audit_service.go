// Audit service implements a Merkle-chained, append-only audit log plus the
// evidence-package generator.
//
// Each entry stores its parent's hash so any tampering breaks the chain. The
// root is exposed via /api/v1/audit/verify and signed with HMAC-SHA256 if a
// signing key is configured.
package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"
)

type AuditEntry struct {
	Seq        int64             `json:"seq"`
	Timestamp  time.Time         `json:"timestamp"`
	Actor      string            `json:"actor"`
	Action     string            `json:"action"`
	TenantID   string            `json:"tenantId"`
	Detail     map[string]string `json:"detail,omitempty"`
	ParentHash string            `json:"parentHash"`
	Hash       string            `json:"hash"`
	Signature  string            `json:"signature,omitempty"`
}

type AuditService struct {
	log     *slog.Logger
	mu      sync.RWMutex
	entries []AuditEntry
	hmacKey []byte
}

func NewAuditService(log *slog.Logger, hmacKey []byte) *AuditService {
	return &AuditService{log: log, hmacKey: hmacKey}
}

func (a *AuditService) ServiceName() string { return "AuditService" }

// Append adds a new entry to the chain.
func (a *AuditService) Append(_ context.Context, actor, action, tenantID string, detail map[string]string) AuditEntry {
	a.mu.Lock()
	defer a.mu.Unlock()

	parent := ""
	if len(a.entries) > 0 {
		parent = a.entries[len(a.entries)-1].Hash
	}
	e := AuditEntry{
		Seq:        int64(len(a.entries) + 1),
		Timestamp:  time.Now().UTC(),
		Actor:      actor,
		Action:     action,
		TenantID:   tenantID,
		Detail:     detail,
		ParentHash: parent,
	}
	e.Hash = hashEntry(e)
	if len(a.hmacKey) > 0 {
		mac := hmac.New(sha256.New, a.hmacKey)
		mac.Write([]byte(e.Hash))
		e.Signature = hex.EncodeToString(mac.Sum(nil))
	}
	a.entries = append(a.entries, e)
	return e
}

func (a *AuditService) Recent(limit int) []AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	n := len(a.entries)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]AuditEntry, 0, limit)
	for i := n - 1; i >= n-limit; i-- {
		out = append(out, a.entries[i])
	}
	return out
}

type VerifyResult struct {
	OK        bool   `json:"ok"`
	Entries   int    `json:"entries"`
	BrokenAt  int64  `json:"brokenAt,omitempty"`
	RootHash  string `json:"rootHash"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// Verify recomputes every hash; returns false on first mismatch.
func (a *AuditService) Verify() VerifyResult {
	a.mu.RLock()
	defer a.mu.RUnlock()
	parent := ""
	for i, e := range a.entries {
		recomputed := AuditEntry{
			Seq:        e.Seq,
			Timestamp:  e.Timestamp,
			Actor:      e.Actor,
			Action:     e.Action,
			TenantID:   e.TenantID,
			Detail:     e.Detail,
			ParentHash: parent,
		}
		if hashEntry(recomputed) != e.Hash {
			return VerifyResult{OK: false, Entries: len(a.entries), BrokenAt: int64(i + 1), GeneratedAt: time.Now().UTC()}
		}
		parent = e.Hash
	}
	return VerifyResult{
		OK: true, Entries: len(a.entries), RootHash: parent, GeneratedAt: time.Now().UTC(),
	}
}

type EvidencePackage struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	Entries     []AuditEntry `json:"entries"`
	RootHash    string       `json:"rootHash"`
	Signature   string       `json:"signature,omitempty"`
	Algorithm   string       `json:"algorithm"`
}

// GeneratePackage produces a snapshot suitable for archival or external review.
func (a *AuditService) GeneratePackage(_ context.Context) (EvidencePackage, error) {
	res := a.Verify()
	if !res.OK {
		return EvidencePackage{}, errors.New("audit chain broken; refusing to seal")
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	pkg := EvidencePackage{
		GeneratedAt: time.Now().UTC(),
		Entries:     append([]AuditEntry(nil), a.entries...),
		RootHash:    res.RootHash,
		Algorithm:   "sha256+hmac",
	}
	if len(a.hmacKey) > 0 {
		mac := hmac.New(sha256.New, a.hmacKey)
		mac.Write([]byte(pkg.RootHash))
		pkg.Signature = hex.EncodeToString(mac.Sum(nil))
	}
	return pkg, nil
}

func hashEntry(e AuditEntry) string {
	// Deterministic hash over the canonicalised payload.
	canon := struct {
		Seq        int64             `json:"seq"`
		Timestamp  time.Time         `json:"timestamp"`
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
