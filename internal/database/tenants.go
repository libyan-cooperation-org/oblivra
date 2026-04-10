package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Tier       string `json:"tier"`
	Status     string `json:"status"` // Active, Suspended, Deleted
	CryptoSalt string `json:"-"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type TenantRepository struct {
	db DatabaseStore
}

func NewTenantRepository(db DatabaseStore) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) CreateTenant(ctx context.Context, t *Tenant) error {
	r.db.Lock()
	defer r.db.Unlock()

	now := time.Now().Format(time.RFC3339)
	t.CreatedAt = now
	t.UpdatedAt = now

	if t.ID == "" {
		t.ID = uuid.New().String()
	}

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO tenants (id, name, tier, status, crypto_salt, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.Name, t.Tier, t.Status, t.CryptoSalt, t.CreatedAt, t.UpdatedAt)

	return err
}

func (r *TenantRepository) GetTenant(ctx context.Context, id string) (*Tenant, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	var t Tenant
	err = conn.QueryRow(`
		SELECT id, name, tier, status, crypto_salt, created_at, updated_at
		FROM tenants WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &t.Tier, &t.Status, &t.CryptoSalt, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &t, nil
}

// CryptographicWipe deletes the salt making the tenant's data unrecoverable
func (r *TenantRepository) CryptographicWipe(ctx context.Context, id string) error {
	r.db.Lock()
	defer r.db.Unlock()

	// Update status and wipe crypto salt securely
	_, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE tenants SET status = 'Deleted', crypto_salt = '', updated_at = ? WHERE id = ?
	`, time.Now().Format(time.RFC3339), id)

	return err
}
