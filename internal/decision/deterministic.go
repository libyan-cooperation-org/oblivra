package decision

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// ExecutionSignature represents an immutable cryptographic fingerprint
// of a specific automated response action given exact inputs.
type ExecutionSignature struct {
	ID           string    `json:"id"`
	Timestamp    string    `json:"timestamp"`
	EventBatchID string    `json:"event_batch_id"`
	PolicyHash   string    `json:"policy_hash"` // State of rules at time of execution
	ActionTaken  string    `json:"action_taken"`
	InputHash    string    `json:"input_hash"`
	FinalHash    string    `json:"final_hash"` // SHA256(InputHash + PolicyHash + ActionTaken)
}

// DeterministicExecutor guarantees reproducible response mappings.
type DeterministicExecutor struct {
	mu         sync.RWMutex
	signatures map[string]ExecutionSignature
	bus        *eventbus.Bus
	log        *logger.Logger
}

func NewDeterministicExecutor(bus *eventbus.Bus, log *logger.Logger) *DeterministicExecutor {
	return &DeterministicExecutor{
		signatures: make(map[string]ExecutionSignature),
		bus:        bus,
		log:        log.WithPrefix("deterministic_engine"),
	}
}

// ExecuteAndSign takes inputs, simulates deterministic mapping, and outputs a reproducible signature.
func (e *DeterministicExecutor) ExecuteAndSign(action string, eventPayload string, policyStateHash string) ExecutionSignature {
	// 1. Canonicalize the input JSON string
	canPayload := e.CanonicalizeJSON(eventPayload)

	// Hash the raw inputs
	hInput := sha256.Sum256([]byte(canPayload))
	inputHash := hex.EncodeToString(hInput[:])

	// Compute final deterministic hash
	finalStr := fmt.Sprintf("%s|%s|%s", inputHash, policyStateHash, action)
	hFinal := sha256.Sum256([]byte(finalStr))
	finalHash := hex.EncodeToString(hFinal[:])

	// Create record
	sig := ExecutionSignature{
		ID:           finalHash[:16],
		Timestamp:    time.Now().Format(time.RFC3339),
		EventBatchID: inputHash[:12],
		PolicyHash:   policyStateHash,
		ActionTaken:  action,
		InputHash:    inputHash,
		FinalHash:    finalHash,
	}

	e.mu.Lock()
	e.signatures[sig.ID] = sig
	e.mu.Unlock()

	e.log.Info("Computed Deterministic Execution Signature: %s (Action: %s)", sig.ID, action)

	// In a real system we would dispatch the action here reliably via gRPC to the Edge Module
	if e.bus != nil {
		e.bus.Publish("response_engine:executed", sig)
	}

	return sig
}

// GetSignatures returns all computed execution signatures
func (e *DeterministicExecutor) GetSignatures() []ExecutionSignature {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var list []ExecutionSignature
	for _, sig := range e.signatures {
		list = append(list, sig)
	}
	return list
}

// Replay simulates what an action *would* result in based on past inputs to prove determinism
func (e *DeterministicExecutor) Replay(inputHash, policyHash, action string) (string, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Recalculate deterministic hash
	finalStr := fmt.Sprintf("%s|%s|%s", inputHash, policyHash, action)
	hFinal := sha256.Sum256([]byte(finalStr))
	finalHash := hex.EncodeToString(hFinal[:])

	// Check if this matches a real past execution
	for _, sig := range e.signatures {
		if sig.FinalHash == finalHash {
			return finalHash, true
		}
	}
	return finalHash, false
}

// Export builds a JSON representation of all verifiable execution proofs.
func (e *DeterministicExecutor) Export() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	b, _ := json.Marshal(e.signatures)
	return string(b)
}

// CanonicalizeJSON parses the raw string into a map/array, and re-marshals it
// using stable, minified sorting to ensure byte-for-byte identical representation
// regardless of key order during generation.
func (e *DeterministicExecutor) CanonicalizeJSON(raw string) string {
	var generic interface{}
	err := json.Unmarshal([]byte(raw), &generic)
	if err != nil {
		// If it's not JSON, return the raw string (perhaps it's plain text)
		return raw
	}

	// json.Marshal automatically sorts map keys in Go
	canonical, err := json.Marshal(generic)
	if err != nil {
		return raw
	}
	return string(canonical)
}
