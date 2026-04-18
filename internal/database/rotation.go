package database

import (
	"context"
	"database/sql"
	"time"
)

type rotationRepository struct {
	db DatabaseStore
}

func NewRotationRepository(db DatabaseStore) RotationStore {
	return &rotationRepository{db: db}
}

func (r *rotationRepository) List(ctx context.Context) ([]RotationPolicy, error) {
	tenantID := MustTenantFromContext(ctx)
	rows, err := r.db.QueryContext(ctx, 
		"SELECT id, tenant_id, credential_id, frequency_days, last_rotation, next_rotation, notify_only, is_active, created_at, updated_at FROM rotation_policies WHERE tenant_id = ?", 
		tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []RotationPolicy
	for rows.Next() {
		var p RotationPolicy
		err := rows.Scan(&p.ID, &p.TenantID, &p.CredentialID, &p.FrequencyDays, &p.LastRotation, &p.NextRotation, &p.NotifyOnly, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, nil
}

func (r *rotationRepository) GetByCredentialID(ctx context.Context, credID string) (*RotationPolicy, error) {
	tenantID := MustTenantFromContext(ctx)
	row := r.db.QueryRowContext(ctx, 
		"SELECT id, tenant_id, credential_id, frequency_days, last_rotation, next_rotation, notify_only, is_active, created_at, updated_at FROM rotation_policies WHERE tenant_id = ? AND credential_id = ?", 
		tenantID, credID)
	
	var p RotationPolicy
	err := row.Scan(&p.ID, &p.TenantID, &p.CredentialID, &p.FrequencyDays, &p.LastRotation, &p.NextRotation, &p.NotifyOnly, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *rotationRepository) Upsert(ctx context.Context, p *RotationPolicy) error {
	p.TenantID = MustTenantFromContext(ctx)
	now := time.Now().Format(time.RFC3339)
	if p.CreatedAt == "" {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	_, err := r.db.ReplicatedExecContext(ctx, 
		`INSERT INTO rotation_policies (id, tenant_id, credential_id, frequency_days, last_rotation, next_rotation, notify_only, is_active, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET 
		 credential_id=excluded.credential_id, 
		 frequency_days=excluded.frequency_days, 
		 last_rotation=excluded.last_rotation, 
		 next_rotation=excluded.next_rotation, 
		 notify_only=excluded.notify_only, 
		 is_active=excluded.is_active, 
		 updated_at=excluded.updated_at`,
		p.ID, p.TenantID, p.CredentialID, p.FrequencyDays, p.LastRotation, p.NextRotation, p.NotifyOnly, p.IsActive, p.CreatedAt, p.UpdatedAt)
	
	return err
}

func (r *rotationRepository) Delete(ctx context.Context, id string) error {
	tenantID := MustTenantFromContext(ctx)
	_, err := r.db.ReplicatedExecContext(ctx, "DELETE FROM rotation_policies WHERE id = ? AND tenant_id = ?", id, tenantID)
	return err
}

func (r *rotationRepository) GetDue(ctx context.Context) ([]RotationPolicy, error) {
	now := time.Now().Format(time.RFC3339)
	rows, err := r.db.QueryContext(ctx, 
		"SELECT id, tenant_id, credential_id, frequency_days, last_rotation, next_rotation, notify_only, is_active, created_at, updated_at FROM rotation_policies WHERE is_active = 1 AND next_rotation <= ?", 
		now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []RotationPolicy
	for rows.Next() {
		var p RotationPolicy
		err := rows.Scan(&p.ID, &p.TenantID, &p.CredentialID, &p.FrequencyDays, &p.LastRotation, &p.NextRotation, &p.NotifyOnly, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, nil
}
