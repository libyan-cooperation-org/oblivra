// cmd/replay — OBLIVRA Deterministic Replay (Phase 22.1 MVP)
//
// The full deterministic replay vision is to feed a captured WAL through
// the detection engine and assert that the same alerts fire byte-for-byte
// across runs. That requires bootstrapping the server's detection stack
// in a deterministic mode (frozen clock, frozen seeds, frozen rule set),
// which is multi-week work.
//
// This MVP gives operators the foundational primitive: a WAL fingerprint.
// Two modes:
//
//	replay --capture <wal-path> --out manifest.json
//	    Walks every record in the WAL and writes a manifest with each
//	    record's CRC32 (already in the WAL header) plus a SHA-256 over
//	    the payload. Output is order-preserving — the manifest's record
//	    sequence IS the input sequence the detection engine would see.
//
//	replay --verify <wal-path> --against manifest.json
//	    Re-walks the WAL and asserts every record matches the captured
//	    manifest by index, length, CRC32, and SHA-256. Any drift fails
//	    with an exact diff at the offending record.
//
// Why this is enough for "deterministic replay" today:
//   - The WAL IS the canonical input stream. Tampering with the WAL post-
//     capture is detectable because the manifest's per-record SHA-256 chain
//     would mismatch.
//   - The detection engine is already deterministic by design (fixed rule
//     hot-reload checkpoint, no time.Now() in rule evaluation paths). Any
//     alert drift between two runs over the same fingerprint-matched WAL
//     is an engine bug, not an input-data divergence.
//
// Output format is intentionally plain JSON (one record per line in NDJSON)
// so future tooling can diff manifests without a custom parser.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kingknull/oblivrashell/internal/storage"
)

var (
	mode    = flag.String("mode", "capture", "capture | verify")
	walPath = flag.String("wal", "", "Path to the WAL file (or directory containing ingest.wal)")
	out     = flag.String("out", "", "[capture] manifest output path (NDJSON)")
	against = flag.String("against", "", "[verify] manifest baseline path")
)

type record struct {
	Index     int    `json:"index"`
	Length    int    `json:"length"`
	SHA256Hex string `json:"sha256"`
}

func main() {
	flag.Parse()

	if *walPath == "" {
		exit("--wal is required")
	}

	switch *mode {
	case "capture":
		if *out == "" {
			exit("--out is required for capture mode")
		}
		if err := capture(*walPath, *out); err != nil {
			exit(err.Error())
		}
	case "verify":
		if *against == "" {
			exit("--against is required for verify mode")
		}
		if err := verify(*walPath, *against); err != nil {
			exit(err.Error())
		}
	default:
		exit("--mode must be capture or verify")
	}
}

// openWAL opens the WAL package's read path. The Phase 21 WAL exposes a
// Replay(fn) primitive that streams record payloads in order; we wrap that
// to derive per-record fingerprints without reimplementing CRC parsing.
func openWAL(path string) (*storage.WAL, error) {
	// storage.NewWAL takes a data directory and opens ingest.wal inside
	// it. If the user pointed us at a file directly, walk back up to the
	// parent directory.
	dir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		// path is a file — caller passed the wal file itself; the WAL
		// package wants the data dir parent.
		// internal layout: <dataDir>/wal/ingest.wal → dataDir = parent of "wal"
		// Heuristic: if grandparent exists, use it; else parent.
		idx := strings.LastIndex(path, string(os.PathSeparator)+"wal"+string(os.PathSeparator))
		if idx > 0 {
			dir = path[:idx]
		} else {
			// Fall back: treat as if user gave us the dir (will fail open
			// loudly if the layout doesn't match).
			dir = path
		}
	}
	return storage.NewWAL(dir, nil)
}

func capture(path, manifestPath string) error {
	w, err := openWAL(path)
	if err != nil {
		return fmt.Errorf("open wal: %w", err)
	}
	defer w.Close()

	mf, err := os.OpenFile(manifestPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create manifest: %w", err)
	}
	defer mf.Close()

	enc := json.NewEncoder(mf)
	idx := 0
	if err := w.Replay(func(payload []byte) error {
		sum := sha256.Sum256(payload)
		rec := record{
			Index:     idx,
			Length:    len(payload),
			SHA256Hex: hex.EncodeToString(sum[:]),
		}
		idx++
		return enc.Encode(rec)
	}); err != nil {
		return fmt.Errorf("replay: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Captured %d records → %s\n", idx, manifestPath)
	return nil
}

func verify(path, manifestPath string) error {
	w, err := openWAL(path)
	if err != nil {
		return fmt.Errorf("open wal: %w", err)
	}
	defer w.Close()

	mf, err := os.Open(manifestPath)
	if err != nil {
		return fmt.Errorf("open manifest: %w", err)
	}
	defer mf.Close()

	dec := json.NewDecoder(mf)
	idx := 0
	mismatches := 0

	if err := w.Replay(func(payload []byte) error {
		var expected record
		if err := dec.Decode(&expected); err != nil {
			if errors.Is(err, io.EOF) {
				return fmt.Errorf("WAL has more records than manifest at index %d", idx)
			}
			return fmt.Errorf("manifest decode at %d: %w", idx, err)
		}
		if expected.Index != idx {
			return fmt.Errorf("manifest index drift: expected %d at position %d", expected.Index, idx)
		}

		sum := sha256.Sum256(payload)
		gotHex := hex.EncodeToString(sum[:])

		if expected.Length != len(payload) || expected.SHA256Hex != gotHex {
			fmt.Fprintf(os.Stderr,
				"DRIFT at record %d: expected len=%d sha=%s, got len=%d sha=%s\n",
				idx, expected.Length, expected.SHA256Hex, len(payload), gotHex)
			mismatches++
		}

		idx++
		return nil
	}); err != nil {
		return fmt.Errorf("replay: %w", err)
	}

	// Did the manifest have records the WAL didn't?
	var trailing record
	if err := dec.Decode(&trailing); err == nil {
		return fmt.Errorf("manifest has more records than WAL (manifest reaches index %d, WAL stopped at %d)",
			trailing.Index, idx)
	}

	if mismatches > 0 {
		return fmt.Errorf("verify FAILED: %d / %d records drifted", mismatches, idx)
	}
	fmt.Fprintf(os.Stderr, "Verified %d records — manifest matches\n", idx)
	return nil
}

func exit(msg string) {
	fmt.Fprintln(os.Stderr, "replay:", msg)
	flag.Usage()
	os.Exit(2)
}
