package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"time"

	"github.com/kingknull/oblivrashell/internal/auth"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/forensics"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/vault"
)

// ForensicsService exposes forensic evidence management to the frontend via Wails.
// This is DESKTOP-ONLY — it uses local hardware (disks, memory, TPM) and must
// never be exposed to the Web layer.
type ForensicsService struct {
	BaseService
	ctx    context.Context
	locker *forensics.EvidenceLocker
	signer *forensics.TPMSigner
	store  database.EvidenceStore
	rbac   *auth.RBACEngine
	bus    *eventbus.Bus
	log    *logger.Logger
	// Phase 36: collector + analyzer removed (disk/memory imaging gone).

	// RFC 3161 TSA anchoring — daemon scheduled in Start, stopped in Stop.
	tsaClient   *forensics.TSAClient
	tsaStop     func()
	dbHandle    database.DatabaseStore // for the TSA scheduler's SQL access
}

// Name returns the service name.
func (s *ForensicsService) Name() string { return "forensics-service" }

// Dependencies returns service dependencies.
func (s *ForensicsService) Dependencies() []string {
	return []string{"vault"}
}

// NewForensicsService creates a new forensics service.
// It initialises the evidence locker using a TPM-rooted signer (falls back to
// vault-derived HMAC key when hardware TPM is unavailable).
// dbHandle is optional — when non-nil it enables RFC 3161 TSA anchoring
// of the evidence chain on a 24h schedule (see internal/forensics/rfc3161.go).
// Pass nil to skip TSA scheduling (e.g. in tests, or when external
// timestamping isn't required for compliance posture).
func NewForensicsService(store database.EvidenceStore, dbHandle database.DatabaseStore, v vault.Provider, rbac *auth.RBACEngine, bus *eventbus.Bus, log *logger.Logger) *ForensicsService {
	// Derive fallback HMAC key from vault; use sentinel ONLY if vault is
	// unavailable. Audit fix #6 (Phase 33 hardening pass) — the previous
	// code had `_ = v.AccessMasterKey(...)` which silently swallowed any
	// access error and dropped to the hardcoded weak sentinel. That meant
	// a transient vault read failure could downgrade evidence-locker
	// signing to a publicly-known key WITHOUT a single line in the logs.
	// We now (a) capture the access error, (b) log loudly at WARN level
	// when we fall back, (c) emit a destructive_action-class audit event
	// on the bus so the operator timeline shows the downgrade. Operators
	// see a permanent breadcrumb instead of a silent compromise.
	var fallbackKey []byte
	var keySource string
	var fallbackReason string
	switch {
	case v == nil:
		fallbackReason = "vault provider not configured"
	case !v.IsUnlocked():
		fallbackReason = "vault is locked"
	default:
		if err := v.AccessMasterKey(func(key []byte) error {
			fallbackKey = make([]byte, len(key))
			copy(fallbackKey, key)
			return nil
		}); err != nil {
			fallbackReason = "vault access failed: " + err.Error()
			fallbackKey = nil
		} else {
			keySource = "vault"
		}
	}
	if fallbackKey == nil {
		// HARD FAIL-LOUD: this is a known-public sentinel. Anything signed
		// with it is repudiable. We still allow startup so operators can
		// log in and unlock the vault, but we shout about it.
		h := sha256.Sum256([]byte("oblivra-evidence-default-key-change-me"))
		fallbackKey = h[:]
		keySource = "hardcoded-sentinel-INSECURE"
		if log != nil {
			log.Warn("[SECURITY] forensics: evidence-locker HMAC falling back to PUBLIC sentinel key — chain-of-custody is REPUDIABLE until vault is unlocked. Reason: %s",
				fallbackReason)
		}
		if bus != nil {
			bus.Publish("forensics:key_downgrade", map[string]any{
				"reason":     fallbackReason,
				"key_source": keySource,
				"severity":   "critical",
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
				"event_type": "destructive_action",
			})
		}
	} else if log != nil {
		log.Info("[forensics] evidence-locker HMAC keyed from %s", keySource)
	}

	// Create TPM-rooted signer (graceful HMAC fallback if no TPM)
	signer := forensics.NewTPMSigner(forensics.TPMSignerConfig{
		FallbackKey: fallbackKey,
	}, log)

	svc := &ForensicsService{
		store:     store,
		locker:    forensics.NewEvidenceLocker(signer, log),
		signer:    signer,
		rbac:      rbac,
		bus:       bus,
		log:       log.WithPrefix("forensics"),
		dbHandle:  dbHandle,
		tsaClient: forensics.NewTSAClient(log.WithPrefix("forensics-tsa")),
	}

	// Set persistence hook
	svc.locker.OnPersist = func(item *forensics.EvidenceItem) error {
		dbItem := svc.mapDomainToDB(item)

		// Create a system-level context since this might be called async, 
		// but use the tenant ID stored in the item if present.
		tenantID := item.Metadata["tenant_id"]
		if tenantID == "" {
			tenantID = "default_tenant"
		}
		ctx := database.WithTenant(svc.ctx, tenantID)
		dbItem.TenantID = tenantID

		if err := svc.store.Create(ctx, &dbItem); err != nil {
			// If create fails, it might already exist; try update
			if err := svc.store.Update(ctx, &dbItem); err != nil {
				return err
			}
		}

		// Save chain entries
		for _, entry := range item.ChainOfCustody {
			dbEntry := database.ChainEntry{
				TenantID:     tenantID,
				EvidenceID:   item.ID,
				Action:       string(entry.Action),
				Actor:        entry.Actor,
				Timestamp:    entry.Timestamp,
				Notes:        entry.Notes,
				PreviousHash: entry.PreviousHash,
				EntryHash:    entry.EntryHash,
			}
			_ = svc.store.AddChainEntry(ctx, &dbEntry)
		}
		return nil
	}

	return svc
}

// Startup initializes the service.
func (s *ForensicsService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("ForensicsService starting (waiting for vault unlock)...")

	// RFC 3161 TSA anchoring of the evidence chain. The interval is
	// 24h by default; override via OBLIVRA_TSA_INTERVAL (e.g. "6h" for
	// hourly anchors during a high-stakes audit window). Skipped when
	// dbHandle is nil (test mode, no relational DB available).
	if s.dbHandle != nil && s.tsaClient != nil {
		sqlDB := s.dbHandle.DB()
		if sqlDB != nil {
			interval := 24 * time.Hour
			if v := os.Getenv("OBLIVRA_TSA_INTERVAL"); v != "" {
				if d, err := time.ParseDuration(v); err == nil && d > 0 {
					interval = d
				}
			}
			s.tsaStop = s.tsaClient.StartScheduler(sqlDB, interval)
			s.log.Info("[forensics] TSA anchoring scheduler started (interval=%s, target=%s)",
				interval, s.tsaClient.URL)
		} else {
			s.log.Info("[forensics] TSA scheduler skipped — relational DB not yet open")
		}
	}

	// Subscribe to vault unlock to load existing evidence
	s.bus.Subscribe(eventbus.EventVaultUnlocked, func(e eventbus.Event) {
		s.log.Info("Vault unlocked detected, loading evidence items...")
		// Load existing items from store using global search during startup
		ctx := database.WithGlobalSearch(s.ctx)
		items, err := s.store.ListAll(ctx)
		if err != nil {
			s.log.Error("Failed to load evidence: %v", err)
			return
		}

		for _, dbItem := range items {
			ctxItem := database.WithTenant(s.ctx, dbItem.TenantID)
			chain, err := s.store.GetChain(ctxItem, dbItem.ID)
			if err != nil {
				s.log.Error("Failed to load chain for %s: %v", dbItem.ID, err)
				continue
			}

			domainItem := s.mapDBToDomain(dbItem, chain)
			s.locker.LoadItem(domainItem)
		}

		s.log.Info("ForensicsService ready with %d loaded items", len(items))
	})

	return nil
}

func (s *ForensicsService) Stop(ctx context.Context) error {
	if s.tsaStop != nil {
		s.tsaStop()
		s.tsaStop = nil
	}
	return nil
}

// CollectEvidence creates a new evidence item for an incident.
func (s *ForensicsService) CollectEvidence(
	ctx context.Context,
	incidentID string,
	evidenceType string,
	name string,
	dataBase64 string,
	collector string,
	notes string,
) (map[string]interface{}, error) {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermEvidenceWrite); err != nil {
		return nil, err
	}
	// Decode base64 data from frontend
	data := []byte(dataBase64) // In production, decode from base64

	tenantID := database.MustTenantFromContext(ctx)

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

	// Attach tenant identity to metadata so persistence hook can resolve it
	if item.Metadata == nil {
		item.Metadata = make(map[string]string)
	}
	item.Metadata["tenant_id"] = tenantID

	s.log.Info("Evidence collected: %s for incident %s (Tenant: %s)", item.ID, incidentID, tenantID)
	s.bus.Publish("forensics:collected", map[string]interface{}{
		"id":          item.ID,
		"incident_id": incidentID,
		"type":        evidenceType,
		"name":        name,
		"tenant_id":   tenantID,
	})

	return map[string]interface{}{
		"id":           item.ID,
		"sha256":       item.SHA256,
		"collected_at": item.CollectedAt,
		"chain_length": len(item.ChainOfCustody),
	}, nil
}

// TransferEvidence records a custody transfer.
func (s *ForensicsService) TransferEvidence(ctx context.Context, itemID string, toActor string, notes string) error {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermEvidenceWrite); err != nil {
		return err
	}
	if err := s.locker.Transfer(itemID, toActor, notes); err != nil {
		return err
	}
	s.log.Info("Evidence %s transferred to %s", itemID, toActor)
	return nil
}

// AnalyzeEvidence records an analysis action.
func (s *ForensicsService) AnalyzeEvidence(ctx context.Context, itemID string, analyst string, notes string) error {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermEvidenceWrite); err != nil {
		return err
	}
	if err := s.locker.Analyze(itemID, analyst, notes); err != nil {
		return err
	}
	s.log.Info("Evidence %s analyzed by %s", itemID, analyst)
	return nil
}

// SealEvidence marks evidence as sealed — no further modifications.
func (s *ForensicsService) SealEvidence(ctx context.Context, itemID string, sealer string, notes string) error {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermEvidenceWrite); err != nil {
		return err
	}
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
func (s *ForensicsService) VerifyEvidence(ctx context.Context, itemID string) (map[string]interface{}, error) {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermEvidenceRead); err != nil {
		return nil, err
	}
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

// AnalyzeFile — Phase 36: removed (entropy/disk/memory analysis is part
// of the disk-imaging response layer that was deleted with the broad
// scope cut). The evidence locker still preserves chain-of-custody for
// log-as-evidence; deep-binary analysis is no longer in scope.

// GetEvidence returns a single evidence item with full chain of custody.
func (s *ForensicsService) GetEvidence(ctx context.Context, itemID string) (*forensics.EvidenceItem, error) {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermEvidenceRead); err != nil {
		return nil, err
	}
	return s.locker.Get(itemID)
}

// ListEvidence returns all evidence for an incident.
func (s *ForensicsService) ListEvidence(ctx context.Context, incidentID string) []*forensics.EvidenceItem {
	if err := s.rbac.Enforce(auth.UserFromContext(ctx), auth.PermEvidenceRead); err != nil {
		return nil
	}
	if incidentID == "" {
		return s.locker.ListAll()
	}
	return s.locker.ListByIncident(incidentID)
}

// IsHardwareRooted returns true if chain-of-custody signatures are hardware-anchored.
func (s *ForensicsService) IsHardwareRooted() bool {
	if s.signer == nil {
		return false
	}
	return s.signer.IsHardwareRooted()
}

// Phase 36: AcquireDiskImage + AcquireMemoryDump removed. Raw disk
// imaging and physical memory acquisition are deep-IR features that
// went away with the broad scope cut. The evidence locker stays — it
// preserves chain-of-custody for log-derived evidence (RFC-3161
// timestamping, Merkle integrity, vault-keyed HMAC). Operators who
// need disk/memory IR should use a dedicated DFIR tool (Velociraptor,
// FTK, Volatility) and import the resulting evidence files via the
// generic Collect API.

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
