package database

import (
	"context"
	"database/sql"
	"fmt"
)

// CredentialRepository handles the persistence of encrypted credentials.
type CredentialRepository struct {
	db DatabaseStore
}

func NewCredentialRepository(db DatabaseStore) *CredentialRepository {
	return &CredentialRepository{db: db}
}

func (r *CredentialRepository) Create(ctx context.Context, c *Credential) error {
	c.TenantID = TenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO credentials (id, tenant_id, label, type, encrypted_data, fingerprint, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.TenantID, c.Label, c.Type, c.EncryptedData, c.Fingerprint, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert credential: %w", err)
	}
	return nil
}

func (r *CredentialRepository) List(ctx context.Context, typeFilter string) ([]Credential, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	var rows *sql.Rows
	var err error

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	if typeFilter != "" {
		rows, err = conn.Query(`
			SELECT id, tenant_id, label, type, encrypted_data, fingerprint, created_at, updated_at 
			FROM credentials WHERE type = ? AND tenant_id = ? ORDER BY label ASC`, typeFilter, tenantID)
	} else {
		rows, err = conn.Query(`
			SELECT id, tenant_id, label, type, encrypted_data, fingerprint, created_at, updated_at 
			FROM credentials WHERE tenant_id = ? ORDER BY label ASC`, tenantID)
	}

	if err != nil {
		return nil, fmt.Errorf("query credentials: %w", err)
	}
	defer rows.Close()

	var creds []Credential
	for rows.Next() {
		var c Credential
		err := rows.Scan(&c.ID, &c.TenantID, &c.Label, &c.Type, &c.EncryptedData, &c.Fingerprint, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			continue
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

func (r *CredentialRepository) GetByID(ctx context.Context, id string) (*Credential, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	var c Credential
	err = conn.QueryRow(`
		SELECT id, tenant_id, label, type, encrypted_data, fingerprint, created_at, updated_at
		FROM credentials WHERE id = ? AND tenant_id = ?`, id, tenantID).Scan(
		&c.ID, &c.TenantID, &c.Label, &c.Type, &c.EncryptedData, &c.Fingerprint, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("credential not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}
	return &c, nil
}

func (r *CredentialRepository) Delete(ctx context.Context, id string) error {
	tenantID := TenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, "DELETE FROM credentials WHERE id = ? AND tenant_id = ?", id, tenantID)
	return err
}

func (r *CredentialRepository) Update(ctx context.Context, c *Credential) error {
	tenantID := TenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx,
		`UPDATE credentials SET label = ?, type = ?, encrypted_data = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND tenant_id = ?`,
		c.Label, c.Type, c.EncryptedData, c.ID, tenantID)
	return err
}
