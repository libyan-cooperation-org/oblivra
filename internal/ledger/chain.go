package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Block represents a single piece of evidence in the immutable chain.
type Block struct {
	Index        uint64    `json:"index"`
	Timestamp    time.Time `json:"timestamp"`
	Data         []byte    `json:"data"`          // The actual evidence payload (e.g. JSON string of the decision)
	DataType     string    `json:"data_type"`     // E.g. "decision", "alert", "file_change"
	PreviousHash string    `json:"previous_hash"` // The SHA-256 hash of the previous block
	Hash         string    `json:"hash"`          // The SHA-256 hash of this block
}

// Chain manages the sequence of Blocks.
// In a full production system, this would be backed by BadgerDB/Disk.
// For this implementation, we will keep it in memory for the session but
// provide serialization methods.
type Chain struct {
	mu     sync.RWMutex
	blocks []*Block
}

func NewChain() *Chain {
	c := &Chain{
		blocks: make([]*Block, 0),
	}
	// Create genesis block
	c.AddBlock([]byte("GENESIS"), "system")
	return c
}

// CalculateHash generates the SHA256 string for a Block.
func CalculateHash(b *Block) string {
	record := fmt.Sprintf("%d%s%s%s%s", b.Index, b.Timestamp.String(), string(b.Data), b.DataType, b.PreviousHash)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// AddBlock creates a new block securely linked to the previous one.
func (c *Chain) AddBlock(data []byte, dataType string) *Block {
	c.mu.Lock()
	defer c.mu.Unlock()

	var prevHash string
	var newIndex uint64

	if len(c.blocks) > 0 {
		prevBlock := c.blocks[len(c.blocks)-1]
		prevHash = prevBlock.Hash
		newIndex = prevBlock.Index + 1
	} else {
		prevHash = "" // Genesis block
		newIndex = 0
	}

	newBlock := &Block{
		Index:        newIndex,
		Timestamp:    time.Now().UTC(),
		Data:         data,
		DataType:     dataType,
		PreviousHash: prevHash,
	}
	newBlock.Hash = CalculateHash(newBlock)

	c.blocks = append(c.blocks, newBlock)
	return newBlock
}

// Verify validates the cryptographic integrity of the entire chain.
// It checks that each block's hash is correct, and that the PreviousHash
// pointer aligns perfectly with the predecessor.
func (c *Chain) Verify() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := 1; i < len(c.blocks); i++ {
		currentBlock := c.blocks[i]
		previousBlock := c.blocks[i-1]

		if currentBlock.Hash != CalculateHash(currentBlock) {
			return fmt.Errorf("block hash is invalid at index %d", currentBlock.Index)
		}

		if currentBlock.PreviousHash != previousBlock.Hash {
			return fmt.Errorf("chain is broken: previous hash mismatch at index %d", currentBlock.Index)
		}
	}

	return nil
}

// GetBlocks returns a snapshot of the current blocks (for UI/API).
func (c *Chain) GetBlocks() []Block {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return copies to prevent mutation
	snap := make([]Block, len(c.blocks))
	for i, b := range c.blocks {
		snap[i] = *b
	}
	return snap
}

// Export returns the chain as a JSON array.
func (c *Chain) Export() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.Marshal(c.blocks)
}
