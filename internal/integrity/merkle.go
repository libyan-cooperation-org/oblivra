package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Hash is a 32-byte SHA-256 digest represented as a hex string.
type Hash = string

// ProofNode is a single step in a Merkle inclusion proof.
type ProofNode struct {
	Hash    Hash `json:"hash"`
	IsRight bool `json:"is_right"` // true = sibling is on the right
	Index   int  `json:"index"`
}

// LeafRecord stores metadata about each leaf in the tree.
type LeafRecord struct {
	Index     int       `json:"index"`
	Hash      Hash      `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	DataSize  int       `json:"data_size"`
}

// MerkleTree is an append-only, in-memory Merkle hash tree.
// It chains every entry into a tamper-evident structure.
// Thread-safe for concurrent reads and sequential writes.
type MerkleTree struct {
	mu     sync.RWMutex
	leaves []Hash       // SHA-256 hashes of raw data
	meta   []LeafRecord // metadata per leaf

	// Optional persistence callback — called after each AddLeaf.
	// Implementations can write to BadgerDB, file, etc.
	OnPersist func(index int, hash Hash, data []byte) error
}

// New creates an empty MerkleTree.
func New() *MerkleTree {
	return &MerkleTree{
		leaves: make([]Hash, 0, 1024),
		meta:   make([]LeafRecord, 0, 1024),
	}
}

// NewWithPersistence creates a MerkleTree with a persistence callback.
func NewWithPersistence(persist func(index int, hash Hash, data []byte) error) *MerkleTree {
	t := New()
	t.OnPersist = persist
	return t
}

// AddLeaf hashes the data, appends it, and returns the leaf hash + index.
func (t *MerkleTree) AddLeaf(data []byte) (Hash, int, error) {
	h := sha256Hash(data)

	t.mu.Lock()
	idx := len(t.leaves)
	t.leaves = append(t.leaves, h)
	t.meta = append(t.meta, LeafRecord{
		Index:     idx,
		Hash:      h,
		Timestamp: time.Now(),
		DataSize:  len(data),
	})
	persist := t.OnPersist
	t.mu.Unlock()

	if persist != nil {
		if err := persist(idx, h, data); err != nil {
			return "", 0, fmt.Errorf("merkle persist leaf %d: %w", idx, err)
		}
	}

	return h, idx, nil
}

// Root computes the current Merkle root hash.
// Returns empty string if the tree has no leaves.
func (t *MerkleTree) Root() Hash {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.leaves) == 0 {
		return ""
	}

	return computeRoot(t.leaves)
}

// LeafCount returns the number of leaves in the tree.
func (t *MerkleTree) LeafCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.leaves)
}

// GetLeaf returns the hash and metadata for a specific leaf index.
func (t *MerkleTree) GetLeaf(index int) (LeafRecord, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if index < 0 || index >= len(t.leaves) {
		return LeafRecord{}, fmt.Errorf("leaf index %d out of range [0, %d)", index, len(t.leaves))
	}

	return t.meta[index], nil
}

// Verify checks that data at leafIndex matches the stored hash
// and that the full proof path is valid against the current root.
func (t *MerkleTree) Verify(leafIndex int, data []byte) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if leafIndex < 0 || leafIndex >= len(t.leaves) {
		return false
	}

	h := sha256Hash(data)
	if h != t.leaves[leafIndex] {
		return false
	}

	// Verify the stored leaf is consistent with the current root
	root := computeRoot(t.leaves)
	proof := buildProof(t.leaves, leafIndex)
	return verifyProof(h, proof, root)
}

// GenerateProof returns a Merkle inclusion proof for a given leaf.
func (t *MerkleTree) GenerateProof(leafIndex int) ([]ProofNode, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if leafIndex < 0 || leafIndex >= len(t.leaves) {
		return nil, fmt.Errorf("leaf index %d out of range [0, %d)", leafIndex, len(t.leaves))
	}

	return buildProof(t.leaves, leafIndex), nil
}

// VerifyExternal allows a third party to verify data against a known root
// using only the proof path (no access to full tree required).
func VerifyExternal(data []byte, proof []ProofNode, expectedRoot Hash) bool {
	h := sha256Hash(data)
	return verifyProof(h, proof, expectedRoot)
}

// ExportState serializes the tree state for backup or transfer.
func (t *MerkleTree) ExportState() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	state := struct {
		Root     Hash         `json:"root"`
		Count    int          `json:"count"`
		Leaves   []Hash       `json:"leaves"`
		Metadata []LeafRecord `json:"metadata"`
	}{
		Root:     computeRoot(t.leaves),
		Count:    len(t.leaves),
		Leaves:   t.leaves,
		Metadata: t.meta,
	}

	return json.Marshal(state)
}

// ImportState restores the tree from a previously exported state.
func (t *MerkleTree) ImportState(data []byte) error {
	var state struct {
		Leaves   []Hash       `json:"leaves"`
		Metadata []LeafRecord `json:"metadata"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("unmarshal merkle state: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.leaves = state.Leaves
	t.meta = state.Metadata
	return nil
}

// ──────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────

func sha256Hash(data []byte) Hash {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func combineHash(left, right Hash) Hash {
	combined := left + right
	return sha256Hash([]byte(combined))
}

// computeRoot builds the Merkle root from a slice of leaf hashes.
func computeRoot(leaves []Hash) Hash {
	if len(leaves) == 0 {
		return ""
	}
	if len(leaves) == 1 {
		return leaves[0]
	}

	// Build tree bottom-up
	level := make([]Hash, len(leaves))
	copy(level, leaves)

	for len(level) > 1 {
		var next []Hash
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				next = append(next, combineHash(level[i], level[i+1]))
			} else {
				// Odd node: duplicate it
				next = append(next, combineHash(level[i], level[i]))
			}
		}
		level = next
	}

	return level[0]
}

// buildProof constructs a Merkle inclusion proof for the given leaf index.
func buildProof(leaves []Hash, leafIndex int) []ProofNode {
	if len(leaves) <= 1 {
		return nil
	}

	level := make([]Hash, len(leaves))
	copy(level, leaves)
	idx := leafIndex
	var proof []ProofNode

	for len(level) > 1 {
		// If odd number of nodes, duplicate the last
		if len(level)%2 != 0 {
			level = append(level, level[len(level)-1])
		}

		// Determine sibling
		siblingIdx := idx ^ 1 // XOR flips the last bit
		proof = append(proof, ProofNode{
			Hash:    level[siblingIdx],
			IsRight: siblingIdx > idx,
			Index:   siblingIdx,
		})

		// Build next level
		var next []Hash
		for i := 0; i < len(level); i += 2 {
			next = append(next, combineHash(level[i], level[i+1]))
		}

		idx /= 2
		level = next
	}

	return proof
}

// verifyProof walks the proof path to reconstruct the root.
func verifyProof(leafHash Hash, proof []ProofNode, expectedRoot Hash) bool {
	current := leafHash
	for _, node := range proof {
		if node.IsRight {
			current = combineHash(current, node.Hash)
		} else {
			current = combineHash(node.Hash, current)
		}
	}
	return current == expectedRoot
}

// LoadLeaf manually adds an existing hash to the tree, typically for database recovery.
func (t *MerkleTree) LoadLeaf(h Hash) {
	t.mu.Lock()
	defer t.mu.Unlock()
	idx := len(t.leaves)
	t.leaves = append(t.leaves, h)
	t.meta = append(t.meta, LeafRecord{
		Index: idx,
		Hash:  h,
	})
}
