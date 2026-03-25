package forensics

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// EvidenceType classifies the kind of evidence collected.
type EvidenceType string

const (
	EvidenceFile       EvidenceType = "file"
	EvidenceLog        EvidenceType = "log"
	EvidenceScreenshot EvidenceType = "screenshot"
	EvidencePCAP       EvidenceType = "pcap"
	EvidenceMemory     EvidenceType = "memory_dump"
	EvidenceArtifact   EvidenceType = "artifact"
)

// CustodyAction describes operations in the chain of custody.
type CustodyAction string

const (
	ActionCollected   CustodyAction = "collected"
	ActionTransferred CustodyAction = "transferred"
	ActionAnalyzed    CustodyAction = "analyzed"
	ActionSealed      CustodyAction = "sealed"
	ActionExported    CustodyAction = "exported"
	ActionVerified    CustodyAction = "verified"
)

// ChainEntry is a single immutable record in the chain of custody.
type ChainEntry struct {
	Action       CustodyAction `json:"action"`
	Actor        string        `json:"actor"`
	Timestamp    string        `json:"timestamp"`
	Notes        string        `json:"notes,omitempty"`
	PreviousHash string        `json:"previous_hash"` // Signature of the prior entry
	EntryHash    string        `json:"entry_hash"`    // Signature of this entry
}

// EvidenceItem represents a single piece of forensic evidence.
type EvidenceItem struct {
	ID             string            `json:"id"`
	IncidentID     string            `json:"incident_id"`
	Type           EvidenceType      `json:"type"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	SHA256         string            `json:"sha256"`
	Size           int64             `json:"size"`
	Collector      string            `json:"collector"`
	CollectedAt    string            `json:"collected_at"`
	Sealed         bool              `json:"sealed"`
	SealedAt       *string           `json:"sealed_at,omitempty"`
	ChainOfCustody []ChainEntry      `json:"chain_of_custody"`
	Tags           []string          `json:"tags,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Data           []byte            `json:"data,omitempty"` // Raw evidence data
}

// ForensicSigner defines the interface for signing chain-of-custody entries.
type ForensicSigner interface {
	SignEntry(payload string) (string, error)
}

// EvidenceLocker is a secure, append-only evidence store with chain-of-custody tracking.
type EvidenceLocker struct {
	mu      sync.RWMutex
	items   map[string]*EvidenceItem // ID → item
	signer  ForensicSigner
	log     *logger.Logger
	
	// Persistence callback
	OnPersist func(item *EvidenceItem) error
}

// NewEvidenceLocker creates an evidence locker with the given signer.
func NewEvidenceLocker(signer ForensicSigner, log *logger.Logger) *EvidenceLocker {
	return &EvidenceLocker{
		items:  make(map[string]*EvidenceItem),
		signer: signer,
		log:    log.WithPrefix("evidence-locker"),
	}
}

// HMACSigner implements ForensicSigner using a traditional HMAC-SHA256.
type HMACSigner struct {
	key []byte
}

func NewHMACSigner(key []byte) *HMACSigner {
	return &HMACSigner{key: key}
}

func (s *HMACSigner) SignEntry(payload string) (string, error) {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// Collect creates a new evidence item and starts its chain of custody.
func (l *EvidenceLocker) Collect(
	incidentID string,
	evidenceType EvidenceType,
	name string,
	data []byte,
	collector string,
	notes string,
) (*EvidenceItem, error) {
	// Compute SHA-256 of the raw evidence data
	hash := sha256.Sum256(data)
	sha256Hex := hex.EncodeToString(hash[:])

	item := &EvidenceItem{
		ID:          uuid.New().String(),
		IncidentID:  incidentID,
		Type:        evidenceType,
		Name:        name,
		SHA256:      sha256Hex,
		Size:        int64(len(data)),
		Collector:   collector,
		CollectedAt: time.Now().Format(time.RFC3339),
		Metadata:    make(map[string]string),
		Data:        data,
	}

	entry := ChainEntry{
		Action:       ActionCollected,
		Actor:        collector,
		Timestamp:    time.Now().Format(time.RFC3339),
		Notes:        notes,
		PreviousHash: "",
	}
	
	sig, err := l.signer.SignEntry(l.serializeEntry(entry))
	if err != nil {
		return nil, fmt.Errorf("sign evidence collection: %w", err)
	}
	entry.EntryHash = sig
	item.ChainOfCustody = []ChainEntry{entry}

	l.mu.Lock()
	l.items[item.ID] = item
	persist := l.OnPersist
	l.mu.Unlock()

	if persist != nil {
		if err := persist(item); err != nil {
			return nil, fmt.Errorf("persist evidence %s: %w", item.ID, err)
		}
	}

	return item, nil
}

// Transfer records a custody transfer to a new actor.
func (l *EvidenceLocker) Transfer(itemID string, toActor string, notes string) error {
	return l.appendChainEntry(itemID, ActionTransferred, toActor, notes)
}

// Analyze records that evidence has been analyzed.
func (l *EvidenceLocker) Analyze(itemID string, analyst string, notes string) error {
	return l.appendChainEntry(itemID, ActionAnalyzed, analyst, notes)
}

// Seal marks evidence as sealed — no further modifications allowed.
func (l *EvidenceLocker) Seal(itemID string, sealer string, notes string) error {
	return l.appendChainEntry(itemID, ActionSealed, sealer, notes)
}

// Verify checks the integrity of the entire chain of custody.
// Returns true if all signatures and hash links are valid.
func (l *EvidenceLocker) Verify(itemID string) (bool, error) {
	l.mu.RLock()
	item, exists := l.items[itemID]
	if !exists {
		l.mu.RUnlock()
		return false, fmt.Errorf("evidence %s not found", itemID)
	}
	chain := make([]ChainEntry, len(item.ChainOfCustody))
	copy(chain, item.ChainOfCustody)
	l.mu.RUnlock()

	for i, entry := range chain {
		if i == 0 {
			if entry.PreviousHash != "" {
				return false, nil
			}
		} else {
			if entry.PreviousHash != chain[i-1].EntryHash {
				return false, nil
			}
		}

		expected, err := l.signer.SignEntry(l.serializeEntry(ChainEntry{
			Action:       entry.Action,
			Actor:        entry.Actor,
			Timestamp:    entry.Timestamp,
			Notes:        entry.Notes,
			PreviousHash: entry.PreviousHash,
		}))
		if err != nil || expected != entry.EntryHash {
			return false, nil
		}
	}

	return true, nil
}

// Get retrieves an evidence item by ID.
func (l *EvidenceLocker) Get(itemID string) (*EvidenceItem, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	item, exists := l.items[itemID]
	if !exists {
		return nil, fmt.Errorf("evidence %s not found", itemID)
	}
	return item, nil
}

// ListByIncident returns all evidence items for a given incident.
func (l *EvidenceLocker) ListByIncident(incidentID string) []*EvidenceItem {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var items []*EvidenceItem
	for _, item := range l.items {
		if item.IncidentID == incidentID {
			items = append(items, item)
		}
	}
	return items
}

// ListAll returns all evidence items in the locker.
func (l *EvidenceLocker) ListAll() []*EvidenceItem {
	l.mu.RLock()
	defer l.mu.RUnlock()

	items := make([]*EvidenceItem, 0, len(l.items))
	for _, item := range l.items {
		items = append(items, item)
	}
	return items
}

// Import restores evidence from a previous export.
func (l *EvidenceLocker) Import(data []byte) error {
	var items map[string]*EvidenceItem
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("unmarshal evidence: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = items
	return nil
}

// LoadItem manually adds an item to the locker, typically for database recovery.
func (l *EvidenceLocker) LoadItem(item *EvidenceItem) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items[item.ID] = item
}

// ──────────────────────────────────────────────
// Internal
// ──────────────────────────────────────────────

func (l *EvidenceLocker) appendChainEntry(itemID string, action CustodyAction, actor string, notes string) error {
	l.mu.Lock()
	item, exists := l.items[itemID]
	if !exists {
		l.mu.Unlock()
		return fmt.Errorf("evidence %s not found", itemID)
	}
	if item.Sealed && action != ActionVerified { // Verification is allowed even if sealed
		l.mu.Unlock()
		return fmt.Errorf("evidence %s is sealed; no modifications allowed", itemID)
	}

	now := time.Now().Format(time.RFC3339)
	if action == ActionSealed {
		item.Sealed = true
		item.SealedAt = &now
	}

	lastHash := ""
	if len(item.ChainOfCustody) > 0 {
		lastHash = item.ChainOfCustody[len(item.ChainOfCustody)-1].EntryHash
	}

	entry := ChainEntry{
		Action:       action,
		Actor:        actor,
		Timestamp:    now,
		Notes:        notes,
		PreviousHash: lastHash,
	}
	
	sig, err := l.signer.SignEntry(l.serializeEntry(entry))
	if err != nil {
		l.mu.Unlock()
		return fmt.Errorf("sign chain entry: %w", err)
	}
	entry.EntryHash = sig
	item.ChainOfCustody = append(item.ChainOfCustody, entry)

	persist := l.OnPersist
	l.mu.Unlock()

	if persist != nil {
		return persist(item)
	}
	return nil
}

func (l *EvidenceLocker) serializeEntry(entry ChainEntry) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s",
		entry.Action,
		entry.Actor,
		entry.Timestamp,
		entry.Notes,
		entry.PreviousHash,
	)
}

// Export returns a JSON-encoded snapshot of all evidence items in the locker.
func (l *EvidenceLocker) Export() ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return json.MarshalIndent(l.items, "", "  ")
}
