package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// EvidenceRepository handles persistence for forensic evidence and chain-of-custody.
type EvidenceRepository struct {
	db DatabaseStore
}

// NewEvidenceRepository creates a new evidence repository.
func NewEvidenceRepository(db DatabaseStore) *EvidenceRepository {
	return &EvidenceRepository{db: db}
}

// Create inserts a new evidence item.
func (r *EvidenceRepository) Create(ctx context.Context, item *EvidenceItem) error {
	r.db.Lock()
	defer r.db.Unlock()

	tags, _ := json.Marshal(item.Tags)
	metadata, _ := json.Marshal(item.Metadata)

	item.TenantID = MustTenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO evidence (
			id, tenant_id, incident_id, type, name, description, sha256, size,
			collector, collected_at, sealed, sealed_at, tags, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TenantID, item.IncidentID, item.Type, item.Name, item.Description,
		item.SHA256, item.Size, item.Collector, item.CollectedAt,
		item.Sealed, item.SealedAt, string(tags), string(metadata),
	)
	if err != nil {
		return fmt.Errorf("insert evidence: %w", err)
	}
	return nil
}

// Update updates an evidence item.
func (r *EvidenceRepository) Update(ctx context.Context, item *EvidenceItem) error {
	r.db.Lock()
	defer r.db.Unlock()

	tags, _ := json.Marshal(item.Tags)
	metadata, _ := json.Marshal(item.Metadata)

	item.TenantID = MustTenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE evidence SET
			incident_id = ?, type = ?, name = ?, description = ?,
			sha256 = ?, size = ?, collector = ?, sealed = ?,
			sealed_at = ?, tags = ?, metadata = ?
		WHERE id = ? AND tenant_id = ?`,
		item.IncidentID, item.Type, item.Name, item.Description,
		item.SHA256, item.Size, item.Collector, item.Sealed,
		item.SealedAt, string(tags), string(metadata), item.ID, item.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update evidence: %w", err)
	}
	return nil
}

// GetByID retrieves a single evidence item.
func (r *EvidenceRepository) GetByID(ctx context.Context, id string) (*EvidenceItem, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)

	row := conn.QueryRow(`
		SELECT id, tenant_id, incident_id, type, name, description, sha256, size,
			collector, collected_at, sealed, sealed_at, tags, metadata
		FROM evidence WHERE id = ? AND tenant_id = ?`, id, tenantID)

	return r.scanEvidence(row)
}

// ListByIncident returns all evidence associated with an incident.
func (r *EvidenceRepository) ListByIncident(ctx context.Context, incidentID string) ([]EvidenceItem, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, incident_id, type, name, description, sha256, size,
			collector, collected_at, sealed, sealed_at, tags, metadata
		FROM evidence WHERE incident_id = ? AND tenant_id = ?
		ORDER BY collected_at DESC`, incidentID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query evidence by incident: %w", err)
	}
	defer rows.Close()

	var items []EvidenceItem
	for rows.Next() {
		item, err := r.scanEvidenceRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

// ListAll returns all evidence items.
func (r *EvidenceRepository) ListAll(ctx context.Context) ([]EvidenceItem, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, incident_id, type, name, description, sha256, size,
			collector, collected_at, sealed, sealed_at, tags, metadata
		FROM evidence WHERE tenant_id = ?
		ORDER BY collected_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query all evidence: %w", err)
	}
	defer rows.Close()

	var items []EvidenceItem
	for rows.Next() {
		item, err := r.scanEvidenceRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

// AddChainEntry appends a record to the chain of custody.
func (r *EvidenceRepository) AddChainEntry(ctx context.Context, entry *ChainEntry) error {
	r.db.Lock()
	defer r.db.Unlock()

	entry.TenantID = MustTenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO evidence_chain (
			evidence_id, tenant_id, action, actor, timestamp, notes, previous_hash, entry_hash
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.EvidenceID, entry.TenantID, entry.Action, entry.Actor, entry.Timestamp,
		entry.Notes, entry.PreviousHash, entry.EntryHash,
	)
	if err != nil {
		return fmt.Errorf("insert chain entry: %w", err)
	}
	return nil
}

// GetChain retrieves the full chain of custody for an evidence item.
func (r *EvidenceRepository) GetChain(ctx context.Context, evidenceID string) ([]ChainEntry, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, evidence_id, action, actor, timestamp, notes, previous_hash, entry_hash
		FROM evidence_chain
		WHERE evidence_id = ? AND tenant_id = ?
		ORDER BY timestamp ASC`, evidenceID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query evidence chain: %w", err)
	}
	defer rows.Close()

	var chain []ChainEntry
	for rows.Next() {
		var e ChainEntry
		err := rows.Scan(
			&e.ID, &e.TenantID, &e.EvidenceID, &e.Action, &e.Actor, &e.Timestamp,
			&e.Notes, &e.PreviousHash, &e.EntryHash,
		)
		if err != nil {
			return nil, fmt.Errorf("scan chain entry: %w", err)
		}
		chain = append(chain, e)
	}
	return chain, nil
}

func (r *EvidenceRepository) scanEvidence(row *sql.Row) (*EvidenceItem, error) {
	var item EvidenceItem
	var tagsJSON, metaJSON sql.NullString
	var sealedAt sql.NullTime

	err := row.Scan(
		&item.ID, &item.TenantID, &item.IncidentID, &item.Type, &item.Name, &item.Description,
		&item.SHA256, &item.Size, &item.Collector, &item.CollectedAt,
		&item.Sealed, &sealedAt, &tagsJSON, &metaJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("evidence not found")
		}
		return nil, fmt.Errorf("scan evidence: %w", err)
	}

	if sealedAt.Valid {
		t := sealedAt.Time.Format(time.RFC3339)
		item.SealedAt = &t
	}
	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &item.Tags)
	}
	if metaJSON.Valid {
		json.Unmarshal([]byte(metaJSON.String), &item.Metadata)
	}

	return &item, nil
}

func (r *EvidenceRepository) scanEvidenceRows(rows *sql.Rows) (*EvidenceItem, error) {
	var item EvidenceItem
	var tagsJSON, metaJSON sql.NullString
	var sealedAt sql.NullTime

	err := rows.Scan(
		&item.ID, &item.TenantID, &item.IncidentID, &item.Type, &item.Name, &item.Description,
		&item.SHA256, &item.Size, &item.Collector, &item.CollectedAt,
		&item.Sealed, &sealedAt, &tagsJSON, &metaJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("scan evidence rows: %w", err)
	}

	if sealedAt.Valid {
		t := sealedAt.Time.Format(time.RFC3339)
		item.SealedAt = &t
	}
	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &item.Tags)
	}
	if metaJSON.Valid {
		json.Unmarshal([]byte(metaJSON.String), &item.Metadata)
	}

	return &item, nil
}
