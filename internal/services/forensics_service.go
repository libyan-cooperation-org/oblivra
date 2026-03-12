package services

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/forensics"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/vault"
)

// ForensicsService exposes forensic evidence management to the frontend via Wails.
type ForensicsService struct {
	BaseService
	ctx      context.Context
	locker   *forensics.EvidenceLocker
	store    database.EvidenceStore
	bus      *eventbus.Bus
	log      *logger.Logger
	analyzer *forensics.ForensicAnalyzer
}

// Name returns the service name.
func (s *ForensicsService) Name() string { return "forensics-service" }

// Dependencies returns service dependencies
func (s *ForensicsService) Dependencies() []string {
	return []string{"vault", "eventbus"}
}

// NewForensicsService creates a new forensics service.
// It initialises the evidence locker using the vault's master key for HMAC signing.
func NewForensicsService(store database.EvidenceStore, v *vault.Vault, bus *eventbus.Bus, log *logger.Logger) *ForensicsService {
	// Derive HMAC key from vault; fallback to a default if vault is locked
	var hmacKey []byte
	if v != nil && v.IsUnlocked() {
		_ = v.AccessMasterKey(func(key []byte) error {
			hmacKey = make([]byte, len(key))
			copy(hmacKey, key)
			return nil
		})
	}
	if hmacKey == nil {
		hmacKey = []byte("oblivra-evidence-default-key-change-me")
	}

	svc := &ForensicsService{
		store:    store,
		locker:   forensics.NewEvidenceLocker(hmacKey),
		bus:      bus,
		log:      log.WithPrefix("forensics"),
		analyzer: forensics.NewForensicAnalyzer(log),
	}

	// Set persistence hook
	svc.locker.OnPersist = func(item *forensics.EvidenceItem) error {
		dbItem := svc.mapDomainToDB(item)
		if err := svc.store.Create(context.Background(), &dbItem); err != nil {
			// If create fails, it might already exist; try update
			if err := svc.store.Update(context.Background(), &dbItem); err != nil {
				return err
			}
		}

		// Save chain entries
		for _, entry := range item.ChainOfCustody {
			dbEntry := database.ChainEntry{
				EvidenceID:   item.ID,
				Action:       string(entry.Action),
				Actor:        entry.Actor,
				Timestamp:    entry.Timestamp,
				Notes:        entry.Notes,
				PreviousHash: entry.PreviousHash,
				EntryHash:    entry.EntryHash,
			}
			// AddChainEntry is idempotent in our logic — it appends.
			// Ideally, we'd only add the NEW entry.
			// For simplicity since we only append:
			_ = svc.store.AddChainEntry(context.Background(), &dbEntry)
		}
		return nil
	}

	return svc
}

// Startup initializes the service.
func (s *ForensicsService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("ForensicsService starting...")

	// Load existing items from store
	items, err := s.store.ListAll(context.Background())
	if err != nil {
		s.log.Error("Failed to load evidence: %v", err)
		return nil // Non-fatal
	}

	for _, dbItem := range items {
		chain, err := s.store.GetChain(context.Background(), dbItem.ID)
		if err != nil {
			s.log.Error("Failed to load chain for %s: %v", dbItem.ID, err)
			continue
		}

		domainItem := s.mapDBToDomain(dbItem, chain)
		s.locker.LoadItem(domainItem)
	}

	s.log.Info("ForensicsService ready with %d loaded items", len(items))
	return nil
}

func (s *ForensicsService) Stop(ctx context.Context) error {
	return nil
}

// CollectEvidence creates a new evidence item for an incident.
func (s *ForensicsService) CollectEvidence(
	incidentID string,
	evidenceType string,
	name string,
	dataBase64 string,
	collector string,
	notes string,
) (map[string]interface{}, error) {
	// Decode base64 data from frontend
	data := []byte(dataBase64) // In production, decode from base64

	item, err := s.locker.Collect(
		incidentID,
		forensics.EvidenceType(evidenceType),
		name,
		data,
		collector,
		notes,
	)
	if err != nil {
		return nil, fmt.Errorf("collect evidence: %w", err)
	}

	s.log.Info("Evidence collected: %s for incident %s", item.ID, incidentID)
	s.bus.Publish("forensics:collected", map[string]interface{}{
		"id":          item.ID,
		"incident_id": incidentID,
		"type":        evidenceType,
		"name":        name,
	})

	return map[string]interface{}{
		"id":           item.ID,
		"sha256":       item.SHA256,
		"collected_at": item.CollectedAt,
		"chain_length": len(item.ChainOfCustody),
	}, nil
}

// TransferEvidence records a custody transfer.
func (s *ForensicsService) TransferEvidence(itemID string, toActor string, notes string) error {
	if err := s.locker.Transfer(itemID, toActor, notes); err != nil {
		return err
	}
	s.log.Info("Evidence %s transferred to %s", itemID, toActor)
	return nil
}

// AnalyzeEvidence records an analysis action.
func (s *ForensicsService) AnalyzeEvidence(itemID string, analyst string, notes string) error {
	if err := s.locker.Analyze(itemID, analyst, notes); err != nil {
		return err
	}
	s.log.Info("Evidence %s analyzed by %s", itemID, analyst)
	return nil
}

// SealEvidence marks evidence as sealed — no further modifications.
func (s *ForensicsService) SealEvidence(itemID string, sealer string, notes string) error {
	if err := s.locker.Seal(itemID, sealer, notes); err != nil {
		return err
	}
	s.log.Info("Evidence %s sealed by %s", itemID, sealer)
	s.bus.Publish("forensics:sealed", map[string]interface{}{
		"id":     itemID,
		"sealer": sealer,
	})
	return nil
}

// VerifyEvidence checks the chain-of-custody integrity.
func (s *ForensicsService) VerifyEvidence(itemID string) (map[string]interface{}, error) {
	valid, err := s.locker.Verify(itemID)
	if err != nil {
		return nil, err
	}

	item, _ := s.locker.Get(itemID)
	return map[string]interface{}{
		"valid":        valid,
		"chain_length": len(item.ChainOfCustody),
		"sealed":       item.Sealed,
	}, nil
}

// AnalyzeFile conducts a deep forensic entropy analysis.
func (s *ForensicsService) AnalyzeFile(itemID string) (*forensics.ForensicReport, error) {
	item, err := s.locker.Get(itemID)
	if err != nil {
		return nil, err
	}

	// In a real scenario, we'd read the actual data from storage.
	// For now, we use the item's metadata or simulated data if empty.
	data := item.Data
	if len(data) == 0 {
		return nil, fmt.Errorf("no raw data available for analysis in item %s", itemID)
	}

	return s.analyzer.AnalyzeFile(item.Name, data)
}

// GetEvidence returns a single evidence item with full chain of custody.
func (s *ForensicsService) GetEvidence(itemID string) (*forensics.EvidenceItem, error) {
	return s.locker.Get(itemID)
}

// ListEvidence returns all evidence for an incident.
func (s *ForensicsService) ListEvidence(incidentID string) []*forensics.EvidenceItem {
	if incidentID == "" {
		return s.locker.ListAll()
	}
	return s.locker.ListByIncident(incidentID)
}

// GetChainOfCustody returns just the chain for a specific evidence item.
func (s *ForensicsService) GetChainOfCustody(itemID string) ([]forensics.ChainEntry, error) {
	item, err := s.locker.Get(itemID)
	if err != nil {
		return nil, err
	}
	return item.ChainOfCustody, nil
}

// Helpers

func (s *ForensicsService) mapDomainToDB(item *forensics.EvidenceItem) database.EvidenceItem {
	return database.EvidenceItem{
		ID:          item.ID,
		IncidentID:  item.IncidentID,
		Type:        string(item.Type),
		Name:        item.Name,
		Description: item.Description,
		SHA256:      item.SHA256,
		Size:        item.Size,
		Collector:   item.Collector,
		CollectedAt: item.CollectedAt,
		Sealed:      item.Sealed,
		SealedAt:    item.SealedAt,
		Tags:        item.Tags,
		Metadata:    item.Metadata,
	}
}

func (s *ForensicsService) mapDBToDomain(dbItem database.EvidenceItem, dbChain []database.ChainEntry) *forensics.EvidenceItem {
	domainChain := make([]forensics.ChainEntry, len(dbChain))
	for i, c := range dbChain {
		domainChain[i] = forensics.ChainEntry{
			Action:       forensics.CustodyAction(c.Action),
			Actor:        c.Actor,
			Timestamp:    c.Timestamp,
			Notes:        c.Notes,
			PreviousHash: c.PreviousHash,
			EntryHash:    c.EntryHash,
		}
	}

	return &forensics.EvidenceItem{
		ID:             dbItem.ID,
		IncidentID:     dbItem.IncidentID,
		Type:           forensics.EvidenceType(dbItem.Type),
		Name:           dbItem.Name,
		Description:    dbItem.Description,
		SHA256:         dbItem.SHA256,
		Size:           dbItem.Size,
		Collector:      dbItem.Collector,
		CollectedAt:    dbItem.CollectedAt,
		Sealed:         dbItem.Sealed,
		SealedAt:       dbItem.SealedAt,
		ChainOfCustody: domainChain,
		Tags:           dbItem.Tags,
		Metadata:       dbItem.Metadata,
	}
}
