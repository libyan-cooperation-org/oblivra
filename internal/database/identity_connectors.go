package database

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/vault"
)

// IdentityConnectorRepository handles persistence for external IdP configurations.
type IdentityConnectorRepository struct {
	db    DatabaseStore
	vault vault.Provider
}

// NewIdentityConnectorRepository creates a new identity connector repository.
func NewIdentityConnectorRepository(db DatabaseStore, v vault.Provider) *IdentityConnectorRepository {
	return &IdentityConnectorRepository{db: db, vault: v}
}

// List returns all connectors for the current tenant.
func (r *IdentityConnectorRepository) List(ctx context.Context) ([]IdentityConnector, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, name, type, enabled, config_json, sync_interval_mins,
		       COALESCE(last_sync, ''), status, COALESCE(error_message, ''), created_at, updated_at
		FROM identity_connectors
		WHERE tenant_id = ?
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query identity connectors: %w", err)
	}
	defer rows.Close()

	var connectors []IdentityConnector
	for rows.Next() {
		var c IdentityConnector
		err := rows.Scan(
			&c.ID, &c.TenantID, &c.Name, &c.Type, &c.Enabled, &c.ConfigJSON,
			&c.SyncIntervalMins, &c.LastSync, &c.Status, &c.ErrorMessage,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan identity connector: %w", err)
		}
		// Security: We do NOT decrypt ConfigJSON here for general listing.
		// Use GetByID or a specialized method for decrypted access.
		c.ConfigJSON = "" // Hide encrypted blob from frontend
		connectors = append(connectors, c)
	}

	return connectors, rows.Err()
}

// Create inserts a new identity connector.
func (r *IdentityConnectorRepository) Create(ctx context.Context, c *IdentityConnector) error {
	now := time.Now().Format(time.RFC3339)
	c.CreatedAt = now
	c.UpdatedAt = now
	c.TenantID = TenantFromContext(ctx)

	// Security: Encrypt configuration if vault is available
	encryptedConfig := c.ConfigJSON
	if r.vault != nil && r.vault.IsUnlocked() && c.ConfigJSON != "" {
		encrypted, err := r.vault.Encrypt([]byte(c.ConfigJSON))
		if err == nil {
			encryptedConfig = base64.StdEncoding.EncodeToString(encrypted)
		}
	}

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO identity_connectors (
			id, tenant_id, name, type, enabled, config_json, sync_interval_mins,
			status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.TenantID, c.Name, c.Type, c.Enabled, encryptedConfig,
		c.SyncIntervalMins, c.Status, c.CreatedAt, c.UpdatedAt,
	)

	return err
}

// GetByID retrieves a single connector.
func (r *IdentityConnectorRepository) GetByID(ctx context.Context, id string) (*IdentityConnector, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	var c IdentityConnector
	err = conn.QueryRow(`
		SELECT id, tenant_id, name, type, enabled, config_json, sync_interval_mins,
		       COALESCE(last_sync, ''), status, COALESCE(error_message, ''), created_at, updated_at
		FROM identity_connectors
		WHERE id = ? AND tenant_id = ?`, id, tenantID).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Type, &c.Enabled, &c.ConfigJSON,
		&c.SyncIntervalMins, &c.LastSync, &c.Status, &c.ErrorMessage,
		&c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("connector not found")
	}
	if err != nil {
		return nil, err
	}

	// Decrypt configuration for sync service usage
	if r.vault != nil && r.vault.IsUnlocked() && c.ConfigJSON != "" {
		decoded, err := base64.StdEncoding.DecodeString(c.ConfigJSON)
		if err == nil {
			decrypted, err := r.vault.Decrypt(decoded)
			if err == nil {
				c.ConfigJSON = string(decrypted)
			}
		}
	}

	return &c, nil
}

// Update updates a connector's configuration.
func (r *IdentityConnectorRepository) Update(ctx context.Context, c *IdentityConnector) error {
	c.UpdatedAt = time.Now().Format(time.RFC3339)
	c.TenantID = TenantFromContext(ctx)

	// Security: Encrypt configuration if vault is available
	encryptedConfig := c.ConfigJSON
	if r.vault != nil && r.vault.IsUnlocked() && c.ConfigJSON != "" {
		encrypted, err := r.vault.Encrypt([]byte(c.ConfigJSON))
		if err == nil {
			encryptedConfig = base64.StdEncoding.EncodeToString(encrypted)
		}
	}

	result, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE identity_connectors SET
			name = ?, enabled = ?, config_json = ?, sync_interval_mins = ?,
			updated_at = ?
		WHERE id = ? AND tenant_id = ?`,
		c.Name, c.Enabled, encryptedConfig, c.SyncIntervalMins,
		c.UpdatedAt, c.ID, c.TenantID,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("connector not found")
	}

	return nil
}

// Delete removes a connector.
func (r *IdentityConnectorRepository) Delete(ctx context.Context, id string) error {
	tenantID := TenantFromContext(ctx)
	_, err := r.db.ReplicatedExecContext(ctx, "DELETE FROM identity_connectors WHERE id = ? AND tenant_id = ?", id, tenantID)
	return err
}

// UpdateStatus updates the status of a sync operation.
func (r *IdentityConnectorRepository) UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error {
	tenantID := TenantFromContext(ctx)
	_, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE identity_connectors SET
			status = ?, error_message = ?, last_sync = ?, updated_at = ?
		WHERE id = ? AND tenant_id = ?`,
		status, errorMessage, time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339),
		id, tenantID,
	)
	return err
}

// MarkSyncStart marks the start of a sync operation.
func (r *IdentityConnectorRepository) MarkSyncStart(ctx context.Context, id string) error {
	tenantID := TenantFromContext(ctx)
	_, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE identity_connectors SET
			status = 'syncing', updated_at = ?
		WHERE id = ? AND tenant_id = ?`,
		time.Now().Format(time.RFC3339), id, tenantID,
	)
	return err
}
