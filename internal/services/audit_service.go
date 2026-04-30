// Audit service implements a Merkle-chained, append-only audit log plus the
// evidence-package generator.
//
// Every entry stores its parent's hash so any tampering breaks the chain. The
// chain is persisted to a line-delimited JSON file (`audit.log`); on startup
// we replay every line, verify it against its parent hash, and refuse to
// start if the chain is broken. The in-memory entries slice is a write-
// through cache of the on-disk journal.
package services

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const auditFile = "audit.log"

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

	path string   // on-disk journal path; "" means in-memory only (tests)
	file *os.File // append-mode handle, fsynced after every write
}

// NewAuditService returns an in-memory-only audit service. Use NewDurable
// for production where the chain must survive restarts.
func NewAuditService(log *slog.Logger, hmacKey []byte) *AuditService {
	return &AuditService{log: log, hmacKey: hmacKey}
}

// NewDurable opens (or creates) an append-only on-disk journal at
// {dir}/audit.log, replays every entry, verifies the chain, and returns a
// service that fsyncs every subsequent Append.
//
// Returns an error (with the broken seq) if the persisted chain has been
// tampered — callers should refuse to start in that case.
func NewDurable(log *slog.Logger, dir string, hmacKey []byte) (*AuditService, error) {
	if dir == "" {
		return nil, errors.New("audit: dir required for durable journal")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("audit mkdir: %w", err)
	}
	path := filepath.Join(dir, auditFile)

	a := &AuditService{log: log, hmacKey: hmacKey, path: path}

	// Replay (creates the file if missing).
	if err := a.replay(path); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("audit open: %w", err)
	}
	a.file = f
	log.Info("audit journal opened", "path", path, "entries", len(a.entries))
	return a, nil
}

// replay reads the on-disk journal, validates each entry's hash against its
// declared parent, and populates a.entries. Returns an error citing the
// first bad sequence number on tamper.
func (a *AuditService) replay(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	br := bufio.NewReader(f)
	parent := ""
	seq := int64(0)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			var e AuditEntry
			if jerr := json.Unmarshal(line, &e); jerr != nil {
				return fmt.Errorf("audit replay: bad json at offset %d: %w", seq+1, jerr)
			}
			seq++
			if e.Seq != seq {
				return fmt.Errorf("audit replay: seq mismatch at line %d (got %d)", seq, e.Seq)
			}
			recomputed := canonical(e, parent)
			if hashEntry(recomputed) != e.Hash {
				return fmt.Errorf("audit replay: chain broken at seq %d", seq)
			}
			a.entries = append(a.entries, e)
			parent = e.Hash
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func (a *AuditService) ServiceName() string { return "AuditService" }

// Append adds a new entry to the chain. Persists synchronously when a journal
// is attached.
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
	e.Hash = hashEntry(canonical(e, parent))
	if len(a.hmacKey) > 0 {
		mac := hmac.New(sha256.New, a.hmacKey)
		mac.Write([]byte(e.Hash))
		e.Signature = hex.EncodeToString(mac.Sum(nil))
	}

	if a.file != nil {
		line, err := json.Marshal(e)
		if err == nil {
			line = append(line, '\n')
			if _, werr := a.file.Write(line); werr != nil {
				a.log.Error("audit journal write failed", "err", werr)
			} else if serr := a.file.Sync(); serr != nil {
				a.log.Error("audit journal fsync failed", "err", serr)
			}
		}
	}

	a.entries = append(a.entries, e)
	return e
}

// Recent returns the newest N entries (newest first).
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
	OK          bool      `json:"ok"`
	Entries     int       `json:"entries"`
	BrokenAt    int64     `json:"brokenAt,omitempty"`
	RootHash    string    `json:"rootHash"`
	GeneratedAt time.Time `json:"generatedAt"`
	Path        string    `json:"path,omitempty"`
}

// Verify recomputes every hash; returns false on first mismatch.
func (a *AuditService) Verify() VerifyResult {
	a.mu.RLock()
	defer a.mu.RUnlock()
	parent := ""
	for i, e := range a.entries {
		if hashEntry(canonical(e, parent)) != e.Hash {
			return VerifyResult{OK: false, Entries: len(a.entries), BrokenAt: int64(i + 1), GeneratedAt: time.Now().UTC(), Path: a.path}
		}
		parent = e.Hash
	}
	return VerifyResult{
		OK: true, Entries: len(a.entries), RootHash: parent,
		GeneratedAt: time.Now().UTC(), Path: a.path,
	}
}

type EvidencePackage struct {
	GeneratedAt time.Time    `json:"generatedAt"`
	Entries     []AuditEntry `json:"entries"`
	RootHash    string       `json:"rootHash"`
	Signature   string       `json:"signature,omitempty"`
	Algorithm   string       `json:"algorithm"`
}

// GeneratePackage produces a snapshot suitable for archival or external
// review. Pure read — callers (HTTP handler / Wails service) are responsible
// for auditing the export action, since they know the actor identity.
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

// AnchorDaily computes the chain root over every entry from `day` (00:00 UTC
// inclusive) until just before `day+24h`, then writes it as a new audit
// entry tagged `audit.daily-anchor`. The next day's chain links forward
// through this anchor, so a later "we cannot find evidence E from yesterday"
// dispute is settled by re-running the verifier.
//
// Returns (anchorEntry, isNewAnchor, error). If an anchor already exists for
// the day we report it without writing a duplicate.
func (a *AuditService) AnchorDaily(ctx context.Context, day time.Time) (AuditEntry, bool, error) {
	day = day.UTC().Truncate(24 * time.Hour)
	endOfDay := day.Add(24 * time.Hour)
	tag := day.Format("2006-01-02")

	a.mu.RLock()
	for _, e := range a.entries {
		if e.Action == "audit.daily-anchor" && e.Detail["day"] == tag {
			a.mu.RUnlock()
			return e, false, nil // already anchored
		}
	}
	// Hash everything in [day, endOfDay).
	h := sha256.New()
	count := 0
	for _, e := range a.entries {
		if e.Timestamp.Before(day) {
			continue
		}
		if !e.Timestamp.Before(endOfDay) {
			continue
		}
		h.Write([]byte(e.Hash))
		count++
	}
	dailyRoot := hex.EncodeToString(h.Sum(nil))
	a.mu.RUnlock()

	if count == 0 {
		return AuditEntry{}, false, nil // no entries that day; nothing to anchor
	}

	e := a.Append(ctx, "system", "audit.daily-anchor", "default", map[string]string{
		"day":     tag,
		"entries": strconv.Itoa(count),
		"root":    dailyRoot,
	})
	a.log.Info("audit daily anchor", "day", tag, "entries", count, "dailyRoot", dailyRoot)
	return e, true, nil
}

// AnchorYesterday is the scheduled-job variant — anchors whatever day
// yesterday was relative to the local clock.
func (a *AuditService) AnchorYesterday(ctx context.Context) error {
	yesterday := time.Now().UTC().Add(-24 * time.Hour)
	_, _, err := a.AnchorDaily(ctx, yesterday)
	return err
}

// Close flushes and closes the journal file.
func (a *AuditService) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.file != nil {
		if err := a.file.Sync(); err != nil {
			return err
		}
		err := a.file.Close()
		a.file = nil
		return err
	}
	return nil
}

// canonical returns the deterministic shape we hash. We exclude Hash + Signature
// so re-derivation matches.
func canonical(e AuditEntry, parent string) AuditEntry {
	return AuditEntry{
		Seq:        e.Seq,
		Timestamp:  e.Timestamp,
		Actor:      e.Actor,
		Action:     e.Action,
		TenantID:   e.TenantID,
		Detail:     e.Detail,
		ParentHash: parent,
	}
}

func hashEntry(e AuditEntry) string {
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
