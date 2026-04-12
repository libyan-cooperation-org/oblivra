package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/kingknull/oblivrashell/internal/events"
	"github.com/kingknull/oblivrashell/internal/storage"
)

// EventChain maintains a hash-linked chain of processed events per tenant.
// Each event hash commits to the previous one, forming a tamper-evident log:
//
//	event_hash = SHA256(prev_hash || event_id || tenant_id || raw_line)
//
// This allows forensic verification at query time: if any event in a chain
// is modified or deleted, the chain breaks deterministically.
type EventChain struct {
	mu        sync.Mutex
	prevHash  map[string]string // tenant -> last hash
	hot       *storage.HotStore
	seqNum    atomic.Int64
}

// NewEventChain creates a new hash-chain tracker.
// If hot is non-nil, chain heads are persisted across restarts.
func NewEventChain(hot *storage.HotStore) *EventChain {
	ec := &EventChain{
		prevHash: make(map[string]string),
		hot:      hot,
	}
	return ec
}

// chainHeadKey returns the Badger key for the chain head of a given tenant.
func chainHeadKey(tenantID string) []byte {
	return []byte(fmt.Sprintf("chain:head:%s", tenantID))
}

// loadHead restores the chain head for a tenant from BadgerDB.
func (ec *EventChain) loadHead(tenantID string) string {
	if ec.hot == nil {
		return ""
	}
	b, err := ec.hot.Get(chainHeadKey(tenantID))
	if err != nil || b == nil {
		return ""
	}
	return string(b)
}

// saveHead persists the current chain head for restart recovery.
func (ec *EventChain) saveHead(tenantID, hash string) {
	if ec.hot == nil {
		return
	}
	_ = ec.hot.Put(chainHeadKey(tenantID), []byte(hash), 0) // 0 = no TTL
}

// Seal computes and stamps the integrity hash onto the event.
// It is called in the hot path after WAL write and before SIEM indexing.
// Thread-safe: uses per-chain mutex to maintain ordering guarantees.
func (ec *EventChain) Seal(evt *events.SovereignEvent) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	tenantID := evt.TenantID
	if tenantID == "" {
		tenantID = "GLOBAL"
	}

	prev, exists := ec.prevHash[tenantID]
	if !exists {
		// First call for this tenant — check Badger for persisted head
		prev = ec.loadHead(tenantID)
		ec.prevHash[tenantID] = prev
	}

	seq := ec.seqNum.Add(1)

	// Hash = SHA256(prevHash | eventID | tenantID | rawLine | seqNum)
	h := sha256.New()
	h.Write([]byte(prev))
	h.Write([]byte(evt.Id))
	h.Write([]byte(tenantID))
	h.Write([]byte(evt.RawLine))
	h.Write([]byte(fmt.Sprintf("%d", seq)))
	hash := hex.EncodeToString(h.Sum(nil))

	evt.IntegrityHash = hash
	evt.IntegrityIndex = int32(seq)

	ec.prevHash[tenantID] = hash
	ec.saveHead(tenantID, hash)
}

// VerifyChain verifies a sequence of events form a valid hash chain.
// Returns the index of the first broken link, or -1 if valid.
func VerifyChain(events []*events.SovereignEvent) int {
	if len(events) == 0 {
		return -1
	}
	for i := 1; i < len(events); i++ {
		prev := events[i-1]
		curr := events[i]

		h := sha256.New()
		h.Write([]byte(prev.IntegrityHash))
		h.Write([]byte(curr.Id))
		h.Write([]byte(curr.TenantID))
		h.Write([]byte(curr.RawLine))
		h.Write([]byte(fmt.Sprintf("%d", curr.IntegrityIndex)))
		expected := hex.EncodeToString(h.Sum(nil))

		if curr.IntegrityHash != expected {
			return i
		}
	}
	return -1
}
