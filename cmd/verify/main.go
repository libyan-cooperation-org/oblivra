// oblivra-verify — offline integrity checker for OBLIVRA artifacts.
//
// Detects the artifact type (audit log / WAL / evidence package) by content
// shape and verifies the appropriate cryptographic invariants. Runs without
// any network access — point it at a file copied off the system.
//
// Usage:
//
//	oblivra-verify path/to/audit.log
//	oblivra-verify path/to/audit.log --hmac "$OBLIVRA_AUDIT_KEY"
//	oblivra-verify path/to/ingest.wal
//	oblivra-verify path/to/evidence.json
//	oblivra-verify path/*.{log,wal,json}
//
// Exit codes:
//
//	0  — every artifact verified
//	1  — at least one artifact failed verification
//	2  — bad usage / unreadable file
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/kingknull/oblivra/internal/verify"
)

func main() {
	hmac := flag.String("hmac", os.Getenv("OBLIVRA_AUDIT_KEY"), "HMAC signing key (hex or raw); leave empty to skip signature checks")
	jsonOut := flag.Bool("json", false, "emit machine-readable JSON")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: oblivra-verify [--hmac KEY] [--json] PATH [PATH...]")
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	key := []byte(*hmac)
	results := make([]verify.Result, 0, flag.NArg())
	failed := 0
	for _, p := range flag.Args() {
		r, err := verify.File(p, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", p, err)
			failed++
			continue
		}
		results = append(results, r)
		if !r.OK {
			failed++
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
	} else {
		printText(results)
	}

	if failed > 0 {
		os.Exit(1)
	}
}

func printText(rs []verify.Result) {
	for _, r := range rs {
		mark := "✓"
		if !r.OK {
			mark = "✗"
		}
		fmt.Printf("%s  %-7s  %s  (%d entries)\n", mark, r.Kind, r.Path, r.Entries)
		if r.RootHash != "" {
			fmt.Printf("    root: %s\n", r.RootHash)
		}
		if r.SignatureOK != nil {
			if *r.SignatureOK {
				fmt.Println("    hmac signature: ok")
			} else {
				fmt.Println("    hmac signature: BAD")
			}
		}
		for _, n := range r.Notes {
			fmt.Println("    note:", n)
		}
		if !r.OK {
			fmt.Printf("    BROKEN at entry %d: %s\n", r.BrokenAt, r.BrokenWhy)
		}
	}
}
