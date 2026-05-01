package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

// Phase 45 — audit compaction.
//
// On long-running deployments the audit journal grows linearly. Most
// older detail (analyst hover events, search queries, etc.) is no longer
// load-bearing — what matters is that the daily Merkle anchor for that
// period was written, signed, and not tampered. Compaction prunes the
// detail while preserving every `audit.daily-anchor` entry plus a single
// `audit.compaction` summary that records how many entries were removed
// and the chain root over the removed range.
//
// Three guarantees:
//
//  1. Every `audit.daily-anchor` survives. An analyst can still prove
//     "this evidence existed by day N" because the anchor for N is still
//     there, and the analyst still has the signed pre-compaction snapshot
//     if they want the granular history.
//
//  2. The compacted chain is itself Merkle-chained — every entry's
//     ParentHash links to the entry before it. Verification against the
//     compacted log uses the same `replay` path as before.
//
//  3. A signed snapshot of the FULL pre-compaction journal is written
//     before any rewriting happens. The snapshot path is returned so
//     operators can move it to cold storage as part of their backup
//     workflow.

// CompactionResult describes what changed.
type CompactionResult struct {
	StartedAt    time.Time `json:"startedAt"`
	FinishedAt   time.Time `json:"finishedAt"`
	Cutoff       time.Time `json:"cutoff"`
	BeforeCount  int       `json:"beforeCount"`
	AfterCount   int       `json:"afterCount"`
	Removed      int       `json:"removed"`
	SnapshotPath string    `json:"snapshotPath"`
	NewRootHash  string    `json:"newRootHash"`
}

// Compact removes every entry strictly older than `cutoff` that is NOT
// an `audit.daily-anchor`. A single `audit.compaction` entry is inserted
// at the position of the first removed run, recording (count, originalRoot).
//
// The journal is rewritten atomically: a snapshot of the current journal
// is saved first; the in-memory entries slice is rebuilt with fresh hashes
// linking forward from a clean genesis; the new journal is written to a
// temp file and renamed over the old one. On any error, the snapshot
// remains and the original journal is untouched.
func (a *AuditService) Compact(ctx context.Context, cutoff time.Time) (CompactionResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.path == "" {
		return CompactionResult{}, errors.New("compact: in-memory audit service cannot be compacted")
	}
	res := CompactionResult{StartedAt: time.Now().UTC(), Cutoff: cutoff.UTC(), BeforeCount: len(a.entries)}

	// 1. Pre-compaction snapshot.
	snap, err := a.writeSnapshot(cutoff)
	if err != nil {
		return res, fmt.Errorf("compact: snapshot: %w", err)
	}
	res.SnapshotPath = snap

	// 2. Decide what's kept. We must keep every audit.daily-anchor (the
	// proof points), every audit.compaction (so successive compactions
	// don't drop history of prior compactions), and everything newer
	// than cutoff. We also remember the chain root of the removed range
	// for the summary entry.
	var (
		kept    []AuditEntry
		removed []AuditEntry
	)
	for _, e := range a.entries {
		switch {
		case !e.Timestamp.Before(cutoff):
			kept = append(kept, e)
		case e.Action == "audit.daily-anchor", e.Action == "audit.compaction":
			kept = append(kept, e)
		default:
			removed = append(removed, e)
		}
	}
	res.Removed = len(removed)
	if res.Removed == 0 {
		res.AfterCount = res.BeforeCount
		res.FinishedAt = time.Now().UTC()
		if len(a.entries) > 0 {
			res.NewRootHash = a.entries[len(a.entries)-1].Hash
		}
		return res, nil
	}

	// Stable Merkle root of removed entries — straight SHA-256 over the
	// concatenated hex hashes, same shape as AnchorDaily. Lets a verifier
	// re-derive this from the snapshot to confirm the summary is honest.
	h := sha256.New()
	for _, e := range removed {
		h.Write([]byte(e.Hash))
	}
	removedRoot := hex.EncodeToString(h.Sum(nil))

	// 3. Build the new chain. We insert a single audit.compaction summary
	// at the timestamp of the most recent removed entry — that keeps the
	// new chain's timestamps monotonic.
	summary := AuditEntry{
		Timestamp: removed[len(removed)-1].Timestamp,
		Actor:     "system",
		Action:    "audit.compaction",
		TenantID:  "default",
		Detail: map[string]string{
			"removed":     strconv.Itoa(len(removed)),
			"removedRoot": removedRoot,
			"firstSeq":    strconv.FormatInt(removed[0].Seq, 10),
			"lastSeq":     strconv.FormatInt(removed[len(removed)-1].Seq, 10),
			"cutoff":      cutoff.UTC().Format(time.RFC3339),
		},
	}
	// Slot the summary into `kept` at the right chronological place.
	insertIdx := 0
	for i, e := range kept {
		if e.Timestamp.After(summary.Timestamp) {
			insertIdx = i
			break
		}
		insertIdx = i + 1
	}
	withSummary := make([]AuditEntry, 0, len(kept)+1)
	withSummary = append(withSummary, kept[:insertIdx]...)
	withSummary = append(withSummary, summary)
	withSummary = append(withSummary, kept[insertIdx:]...)

	// 4. Re-hash forward.
	parent := ""
	for i := range withSummary {
		withSummary[i].Seq = int64(i + 1)
		withSummary[i].ParentHash = parent
		withSummary[i].Hash = hashEntry(canonical(withSummary[i], parent))
		if len(a.hmacKey) > 0 {
			mac := hmac.New(sha256.New, a.hmacKey)
			mac.Write([]byte(withSummary[i].Hash))
			withSummary[i].Signature = hex.EncodeToString(mac.Sum(nil))
		}
		parent = withSummary[i].Hash
	}

	// 5. Atomic rewrite of the journal.
	if err := a.rewriteJournal(withSummary); err != nil {
		return res, fmt.Errorf("compact: rewrite: %w", err)
	}

	a.entries = withSummary
	res.AfterCount = len(withSummary)
	res.FinishedAt = time.Now().UTC()
	res.NewRootHash = parent
	a.log.Info("audit compaction complete",
		"before", res.BeforeCount, "after", res.AfterCount,
		"removed", res.Removed, "snapshot", res.SnapshotPath, "newRoot", parent)
	_ = ctx
	return res, nil
}

func (a *AuditService) writeSnapshot(cutoff time.Time) (string, error) {
	src, err := os.Open(a.path)
	if err != nil {
		return "", err
	}
	defer src.Close()
	stamp := time.Now().UTC().Format("20060102T150405Z")
	out := a.path + "." + stamp + ".snapshot"
	dst, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	// Sidecar: HMAC over the full snapshot if a key is configured. Lets
	// reviewers prove the snapshot is the one our service produced.
	if len(a.hmacKey) > 0 {
		body, err := os.ReadFile(out)
		if err != nil {
			return "", err
		}
		mac := hmac.New(sha256.New, a.hmacKey)
		mac.Write(body)
		mac.Write([]byte("|"))
		mac.Write([]byte(cutoff.UTC().Format(time.RFC3339)))
		sig := hex.EncodeToString(mac.Sum(nil))
		if err := os.WriteFile(out+".sig", []byte(sig+"\n"), 0o600); err != nil {
			return "", err
		}
	}
	return out, nil
}

func (a *AuditService) rewriteJournal(entries []AuditEntry) error {
	tmp := a.path + ".compact.tmp"
	f, err := os.OpenFile(tmp, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	for _, e := range entries {
		line, err := json.Marshal(e)
		if err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
		line = append(line, '\n')
		if _, err := f.Write(line); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	f.Close()

	// Close the live journal handle, swap in the new file, reopen append.
	if a.file != nil {
		_ = a.file.Sync()
		_ = a.file.Close()
		a.file = nil
	}
	if err := os.Rename(tmp, a.path); err != nil {
		return err
	}
	reopened, err := os.OpenFile(a.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	a.file = reopened
	return nil
}
