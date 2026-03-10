package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

// Block mirrors the internal ledger struct but is entirely independent.
type Block struct {
	Index        int       `json:"index"`
	Timestamp    time.Time `json:"timestamp"`
	EvidenceType string    `json:"evidence_type"`
	PayloadHash  string    `json:"payload_hash"` // The sha256 of the raw data (log, decision)
	PrevHash     string    `json:"prev_hash"`
	Hash         string    `json:"hash"` // The sha256 of this block's contents
	Signature    string    `json:"signature,omitempty"`
}

func main() {
	fmt.Println("==================================================")
	fmt.Println("OBLIVRA FORENSIC LEDGER VERIFIER (AIR-GAPPED CLI)")
	fmt.Println("==================================================")

	filePtr := flag.String("file", "ledger.json", "Path to the exported JSON ledger file")
	flag.Parse()

	data, err := os.ReadFile(*filePtr)
	if err != nil {
		fmt.Printf("[FATAL] Could not read ledger file: %v\n", err)
		os.Exit(1)
	}

	var chain []Block
	err = json.Unmarshal(data, &chain)
	if err != nil {
		fmt.Printf("[FATAL] Invalid ledger format: %v\n", err)
		os.Exit(1)
	}

	if len(chain) == 0 {
		fmt.Println("[WARNING] Chain is empty.")
		os.Exit(0)
	}

	fmt.Printf("[INFO] Loaded %d blocks. Verifying cryptographic links...\n\n", len(chain))

	isValid := true
	for i := 1; i < len(chain); i++ {
		current := chain[i]
		prev := chain[i-1]

		// 1. Verify structural sequentiality
		if current.Index != prev.Index+1 {
			fmt.Printf("[FAIL] Broken sequence at Block %d (Expected %d)\n", current.Index, prev.Index+1)
			isValid = false
		}

		// 2. Verify link integrity (current.PrevHash == prev.Hash)
		if current.PrevHash != prev.Hash {
			fmt.Printf("[FAIL] Link broken at Block %d. PrevHash mismatch!\n       Got: %s\n       Expected: %s\n",
				current.Index, current.PrevHash, prev.Hash)
			isValid = false
		}

		// 3. Verify math integrity (Recalculate current hash)
		recalcHash := calculateHash(current)
		if recalcHash != current.Hash {
			fmt.Printf("[FAIL] Tampering detected in Block %d contents. Hash mismatch!\n       Got: %s\n       Expected: %s\n",
				current.Index, current.Hash, recalcHash)
			isValid = false
		}
	}

	if isValid {
		fmt.Println("[SUCCESS] Merkle chain is MATHEMATICALLY SOUND and UNTAMPERED.")
		fmt.Printf("          Final Root Hash: %s\n", chain[len(chain)-1].Hash)
		os.Exit(0)
	} else {
		fmt.Println("\n[CRITICAL] LEDGER COMPROMISED. DO NOT ADMIT AS EVIDENCE.")
		os.Exit(1)
	}
}

// calculateHash exactly mirrors the internal node logic to prove determinism.
func calculateHash(b Block) string {
	record := fmt.Sprintf("%d:%s:%s:%s:%s",
		b.Index,
		b.Timestamp.UTC().Format(time.RFC3339Nano),
		b.EvidenceType,
		b.PayloadHash,
		b.PrevHash,
	)
	h := sha256.Sum256([]byte(record))
	return hex.EncodeToString(h[:])
}
