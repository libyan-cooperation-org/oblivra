package lineage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// StageType represents a step in the data processing pipeline.
type StageType string

const (
	StageIngested  StageType = "ingested"
	StageParsed    StageType = "parsed"
	StageEnriched  StageType = "enriched"
	StageDetected  StageType = "detected"
	StageAlerted   StageType = "alerted"
	StageResponded StageType = "responded"
	StageEvidenced StageType = "evidenced"
)

// LineageRecord represents one step in the provenance chain.
type LineageRecord struct {
	ID        string                 `json:"id"`
	EntityID  string                 `json:"entity_id"`
	Stage     StageType              `json:"stage"`
	Timestamp string                 `json:"timestamp"`
	ParentID  string                 `json:"parent_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	ProofHash string                 `json:"proof_hash"`
}

// LineageChain is an ordered provenance trail for an entity.
type LineageChain struct {
	EntityID string          `json:"entity_id"`
	Records  []LineageRecord `json:"records"`
}

// LineageEngine tracks data provenance across pipeline stages.
type LineageEngine struct {
	mu       sync.RWMutex
	records  map[string]*LineageRecord // id -> record
	byEntity map[string][]string       // entityID -> []recordID
	bus      *eventbus.Bus
	log      *logger.Logger
}

// NewLineageEngine creates a new provenance tracking engine.
func NewLineageEngine(bus *eventbus.Bus, log *logger.Logger) *LineageEngine {
	e := &LineageEngine{
		records:  make(map[string]*LineageRecord),
		byEntity: make(map[string][]string),
		bus:      bus,
		log:      log.WithPrefix("lineage"),
	}

	// Subscribe to pipeline events for auto-capture
	if bus != nil {
		bus.Subscribe("siem.event_indexed", func(event eventbus.Event) {
			defer func() {
				if r := recover(); r != nil {
					e.log.Debug("Recovered from panic in Lineage: %v", r)
				}
			}()
			if m, ok := event.Data.(map[string]interface{}); ok {
				entityID, _ := m["event_id"].(string)
				if entityID != "" {
					e.AddRecord(entityID, StageIngested, "", m)
				}
			}
		})

		bus.Subscribe("detection.match", func(event eventbus.Event) {
			if m, ok := event.Data.(map[string]interface{}); ok {
				entityID, _ := m["event_id"].(string)
				if entityID != "" {
					e.AddRecord(entityID, StageDetected, "", m)
				}
			}
		})

		bus.Subscribe("alert.created", func(event eventbus.Event) {
			if m, ok := event.Data.(map[string]interface{}); ok {
				entityID, _ := m["event_id"].(string)
				if entityID != "" {
					e.AddRecord(entityID, StageAlerted, "", m)
				}
			}
		})
	}

	return e
}

// AddRecord inserts a new provenance record into the lineage DAG.
func (e *LineageEngine) AddRecord(entityID string, stage StageType, parentID string, metadata map[string]interface{}) string {
	e.mu.Lock()
	defer e.mu.Unlock()

	id := e.generateID(entityID, stage)
	record := &LineageRecord{
		ID:        id,
		EntityID:  entityID,
		Stage:     stage,
		Timestamp: time.Now().Format(time.RFC3339),
		ParentID:  parentID,
		Metadata:  metadata,
		ProofHash: e.computeProof(entityID, stage, parentID),
	}

	e.records[id] = record
	e.byEntity[entityID] = append(e.byEntity[entityID], id)

	// Cap to 50K records to prevent unbounded growth
	if len(e.records) > 50000 {
		e.evictOldest()
	}

	return id
}

// GetChain returns the full provenance chain for an entity.
func (e *LineageEngine) GetChain(entityID string) *LineageChain {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ids, ok := e.byEntity[entityID]
	if !ok {
		return &LineageChain{EntityID: entityID}
	}

	chain := &LineageChain{
		EntityID: entityID,
		Records:  make([]LineageRecord, 0, len(ids)),
	}

	for _, id := range ids {
		if rec, exists := e.records[id]; exists {
			chain.Records = append(chain.Records, *rec)
		}
	}

	return chain
}

// GetProvenance returns a single lineage record by ID.
func (e *LineageEngine) GetProvenance(recordID string) *LineageRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rec, ok := e.records[recordID]
	if !ok {
		return nil
	}
	copy := *rec
	return &copy
}

// GetRecentLineage returns the N most recent lineage records.
func (e *LineageEngine) GetRecentLineage(limit int) []LineageRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()

	all := make([]LineageRecord, 0, len(e.records))
	for _, rec := range e.records {
		all = append(all, *rec)
	}

	// Sort by timestamp descending
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if parseTime(all[j].Timestamp).After(parseTime(all[i].Timestamp)) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	if limit > 0 && limit < len(all) {
		all = all[:limit]
	}

	return all
}

// Stats returns summary statistics for the lineage store.
func (e *LineageEngine) Stats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stageCounts := make(map[string]int)
	for _, rec := range e.records {
		stageCounts[string(rec.Stage)]++
	}

	return map[string]interface{}{
		"total_records":  len(e.records),
		"total_entities": len(e.byEntity),
		"by_stage":       stageCounts,
	}
}

func (e *LineageEngine) generateID(entityID string, stage StageType) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", entityID, stage, time.Now().UnixNano())))
	return hex.EncodeToString(h[:8])
}

func (e *LineageEngine) computeProof(entityID string, stage StageType, parentID string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%d", entityID, stage, parentID, time.Now().UnixNano())))
	return hex.EncodeToString(h[:])
}

func (e *LineageEngine) evictOldest() {
	// Find and remove the oldest 10K records
	type aged struct {
		id string
		ts time.Time
	}
	var all []aged
	for id, rec := range e.records {
		all = append(all, aged{id: id, ts: parseTime(rec.Timestamp)})
	}
	// Sort ascending
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].ts.Before(all[i].ts) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	evictCount := 10000
	if evictCount > len(all) {
		evictCount = len(all) / 2
	}
	for i := 0; i < evictCount; i++ {
		rec := e.records[all[i].id]
		if rec != nil {
			// Remove from entity index
			ids := e.byEntity[rec.EntityID]
			for j, rid := range ids {
				if rid == all[i].id {
					e.byEntity[rec.EntityID] = append(ids[:j], ids[j+1:]...)
					break
				}
			}
			if len(e.byEntity[rec.EntityID]) == 0 {
				delete(e.byEntity, rec.EntityID)
			}
		}
		delete(e.records, all[i].id)
	}
}
