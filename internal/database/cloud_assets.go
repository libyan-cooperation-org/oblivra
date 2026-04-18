package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// CloudAssetRepository handles cloud resource discovery and inventory tracking.
type CloudAssetRepository struct {
	db DatabaseStore
}

// NewCloudAssetRepository creates a new cloud asset repository.
func NewCloudAssetRepository(db DatabaseStore) *CloudAssetRepository {
	return &CloudAssetRepository{db: db}
}

// Upsert inserts a new cloud asset or updates an existing one if ID matches for the tenant.
func (r *CloudAssetRepository) Upsert(ctx context.Context, asset *CloudAsset) error {
	tenantID := MustTenantFromContext(ctx)
	asset.TenantID = tenantID

	now := time.Now().Format(time.RFC3339)
	if asset.FirstSeen == "" {
		asset.FirstSeen = now
	}
	asset.LastSeen = now

	metadataJSON, _ := json.Marshal(asset.Metadata)
	tagsJSON, _ := json.Marshal(asset.Tags)

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO cloud_assets (
			id, tenant_id, provider, region, account_id, type, name, status, 
			metadata, tags, first_seen, last_seen
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id, tenant_id) DO UPDATE SET
			status = excluded.status,
			metadata = excluded.metadata,
			tags = excluded.tags,
			last_seen = excluded.last_seen
	`, asset.ID, asset.TenantID, asset.Provider, asset.Region, asset.AccountID,
		asset.Type, asset.Name, asset.Status, string(metadataJSON), string(tagsJSON),
		asset.FirstSeen, asset.LastSeen)

	if err != nil {
		return fmt.Errorf("upsert cloud asset: %w", err)
	}

	return nil
}

// GetByID retrieves a single cloud asset by its ID with tenant isolation.
func (r *CloudAssetRepository) GetByID(ctx context.Context, id string) (*CloudAsset, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)

	row := conn.QueryRow(`
		SELECT id, tenant_id, provider, region, account_id, type, name, status, 
			metadata, tags, first_seen, last_seen
		FROM cloud_assets WHERE id = ? AND tenant_id = ?`, id, tenantID)

	return r.scanAsset(row)
}

// List returns a slice of cloud assets based on optional filters with tenant isolation.
func (r *CloudAssetRepository) List(ctx context.Context, provider string, accountID string) ([]CloudAsset, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)
	query := "SELECT id, tenant_id, provider, region, account_id, type, name, status, metadata, tags, first_seen, last_seen FROM cloud_assets WHERE tenant_id = ?"
	args := []interface{}{tenantID}

	if provider != "" {
		query += " AND provider = ?"
		args = append(args, provider)
	}
	if accountID != "" {
		query += " AND account_id = ?"
		args = append(args, accountID)
	}

	rows, err := conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query cloud assets: %w", err)
	}
	defer rows.Close()

	var assets []CloudAsset
	for rows.Next() {
		asset, err := r.scanAssetRows(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, *asset)
	}

	return assets, rows.Err()
}

// Delete removes a cloud asset record.
func (r *CloudAssetRepository) Delete(ctx context.Context, id string) error {
	tenantID := MustTenantFromContext(ctx)
	_, err := r.db.ReplicatedExecContext(ctx, "DELETE FROM cloud_assets WHERE id = ? AND tenant_id = ?", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete cloud asset: %w", err)
	}
	return nil
}

// GetStats returns a map of asset counts per provider for the tenant.
func (r *CloudAssetRepository) GetStats(ctx context.Context) (map[string]int, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)
	rows, err := conn.Query("SELECT provider, COUNT(*) FROM cloud_assets WHERE tenant_id = ? GROUP BY provider", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var provider string
		var count int
		if err := rows.Scan(&provider, &count); err != nil {
			continue
		}
		stats[provider] = count
	}
	return stats, nil
}

func (r *CloudAssetRepository) scanAsset(row *sql.Row) (*CloudAsset, error) {
	var a CloudAsset
	var metaJSON, tagsJSON string
	err := row.Scan(&a.ID, &a.TenantID, &a.Provider, &a.Region, &a.AccountID, &a.Type, &a.Name, &a.Status, &metaJSON, &tagsJSON, &a.FirstSeen, &a.LastSeen)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	json.Unmarshal([]byte(metaJSON), &a.Metadata)
	json.Unmarshal([]byte(tagsJSON), &a.Tags)
	return &a, nil
}

func (r *CloudAssetRepository) scanAssetRows(rows *sql.Rows) (*CloudAsset, error) {
	var a CloudAsset
	var metaJSON, tagsJSON string
	err := rows.Scan(&a.ID, &a.TenantID, &a.Provider, &a.Region, &a.AccountID, &a.Type, &a.Name, &a.Status, &metaJSON, &tagsJSON, &a.FirstSeen, &a.LastSeen)
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(metaJSON), &a.Metadata)
	json.Unmarshal([]byte(tagsJSON), &a.Tags)
	return &a, nil
}
