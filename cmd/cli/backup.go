package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Phase 49 — `oblivra-cli backup verify <path>`
//
// Offline integrity check on a data-dir snapshot (the operator's nightly
// tarball, restored to disk). It is intentionally self-contained: it does
// NOT call out to the running server, so it works on an air-gapped review
// box where the only thing available is the backup itself.
//
// Checks:
//
//  1. audit.log — replay the Merkle chain entry-by-entry; report the first
//     broken sequence. This is the same algorithm the server runs at startup.
//
//  2. warm/*.parquet — for every parquet file we look for a matching
//     .sha256 sidecar (one-line "<hex>  <basename>"). If sidecars are
//     present we verify them. We do NOT generate sidecars here — that's a
//     write operation and `verify` must be read-only.
//
//  3. vault.oblivra (if present) — parse the JSON envelope; do not unlock.
//     Confirms the file isn't truncated or corrupt.
//
// Exit code: 0 if everything verifies, 1 if any check fails. Output is JSON
// so the result can be ingested by another tool.

type backupReport struct {
	Path        string         `json:"path"`
	GeneratedAt time.Time      `json:"generatedAt"`
	OK          bool           `json:"ok"`
	Audit       auditReport    `json:"audit"`
	Warm        warmReport     `json:"warm"`
	Vault       vaultReport    `json:"vault"`
	Errors      []string       `json:"errors,omitempty"`
}

type auditReport struct {
	Present  bool   `json:"present"`
	OK       bool   `json:"ok"`
	Entries  int64  `json:"entries"`
	BrokenAt int64  `json:"brokenAt,omitempty"`
	RootHash string `json:"rootHash,omitempty"`
}

type warmReport struct {
	Files    int      `json:"files"`
	Verified int      `json:"verified"`
	NoSide   int      `json:"noSidecar"`
	Failed   []string `json:"failed,omitempty"`
}

type vaultReport struct {
	Present bool `json:"present"`
	OK      bool `json:"ok"`
}

func backupCmd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "backup: need verify <path>")
		os.Exit(2)
	}
	switch args[0] {
	case "verify":
		fs := flag.NewFlagSet("backup-verify", flag.ExitOnError)
		_ = fs.Parse(args[1:])
		if fs.NArg() < 1 {
			fmt.Fprintln(os.Stderr, "usage: oblivra-cli backup verify <path>")
			os.Exit(2)
		}
		root := fs.Arg(0)
		report := verifyBackup(root)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		if !report.OK {
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "backup: unknown subcommand", args[0])
		os.Exit(2)
	}
}

func verifyBackup(root string) backupReport {
	r := backupReport{Path: root, GeneratedAt: time.Now().UTC(), OK: true}

	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		r.OK = false
		r.Errors = append(r.Errors, fmt.Sprintf("not a directory: %s", root))
		return r
	}

	// 1. audit.log — Merkle chain replay.
	auditPath := filepath.Join(root, "audit.log")
	if st, err := os.Stat(auditPath); err == nil && !st.IsDir() {
		r.Audit = verifyAuditChain(auditPath)
		if !r.Audit.OK {
			r.OK = false
		}
	} else {
		r.Audit.Present = false
	}

	// 2. warm/*.parquet — sidecar verification.
	warmDir := filepath.Join(root, "warm")
	if st, err := os.Stat(warmDir); err == nil && st.IsDir() {
		r.Warm = verifyWarmFiles(warmDir)
		if len(r.Warm.Failed) > 0 {
			r.OK = false
		}
	}

	// 3. vault.
	vaultPath := filepath.Join(root, "oblivra.vault")
	if st, err := os.Stat(vaultPath); err == nil && !st.IsDir() {
		r.Vault.Present = true
		r.Vault.OK = canParseJSON(vaultPath)
		if !r.Vault.OK {
			r.OK = false
			r.Errors = append(r.Errors, "vault file is not valid JSON")
		}
	}

	return r
}

// verifyAuditChain walks the journal line-by-line and recomputes each
// entry's hash from its canonical projection + parent hash. Mirrors the
// server-side replay so results agree byte-for-byte.
func verifyAuditChain(path string) auditReport {
	out := auditReport{Present: true}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 1<<20), 16<<20)

	parent := ""
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" {
			continue
		}
		var e auditEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			out.OK = false
			out.BrokenAt = out.Entries + 1
			return out
		}
		recomputed := hashAuditEntry(e, parent)
		if recomputed != e.Hash {
			out.OK = false
			out.BrokenAt = e.Seq
			return out
		}
		parent = e.Hash
		out.Entries++
	}
	if err := scan.Err(); err != nil {
		out.OK = false
		return out
	}
	out.OK = true
	out.RootHash = parent
	return out
}

// auditEntry mirrors services.AuditEntry — duplicated here to keep this
// CLI binary independent of the server packages so it can be shipped
// alone to the air-gapped reviewer.
type auditEntry struct {
	Seq        int64             `json:"seq"`
	Timestamp  time.Time         `json:"timestamp"`
	Actor      string            `json:"actor"`
	Action     string            `json:"action"`
	TenantID   string            `json:"tenantId"`
	Detail     map[string]string `json:"detail,omitempty"`
	ParentHash string            `json:"parentHash"`
	Hash       string            `json:"hash"`
}

func hashAuditEntry(e auditEntry, parent string) string {
	canon := struct {
		Seq        int64             `json:"seq"`
		Timestamp  time.Time         `json:"timestamp"`
		Actor      string            `json:"actor"`
		Action     string            `json:"action"`
		TenantID   string            `json:"tenantId"`
		Detail     map[string]string `json:"detail"`
		ParentHash string            `json:"parentHash"`
	}{e.Seq, e.Timestamp, e.Actor, e.Action, e.TenantID, e.Detail, parent}
	b, _ := json.Marshal(canon)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func verifyWarmFiles(dir string) warmReport {
	var w warmReport
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".parquet") {
			return nil
		}
		w.Files++
		side := path + ".sha256"
		body, err := os.ReadFile(side)
		if err != nil {
			w.NoSide++
			return nil
		}
		want := strings.Fields(strings.TrimSpace(string(body)))
		if len(want) == 0 {
			w.Failed = append(w.Failed, path)
			return nil
		}
		got, err := sha256File(path)
		if err != nil {
			w.Failed = append(w.Failed, path)
			return nil
		}
		if !strings.EqualFold(got, want[0]) {
			w.Failed = append(w.Failed, path)
			return nil
		}
		w.Verified++
		return nil
	})
	return w
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func canParseJSON(path string) bool {
	body, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var any any
	return json.Unmarshal(body, &any) == nil
}
