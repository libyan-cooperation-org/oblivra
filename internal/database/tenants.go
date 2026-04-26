package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
// CryptographicWipe marks a tenant as Deleted and zeroes its crypto salt.
//
// Phase 22.2 / Phase 30 — GDPR Article 30 evidence: every wipe is
// recorded in `tenant_deletion_log` with the actor identity, the
// reason, and a SHA-256 hash of the previous tenant row. The
// `deletionAuditor` callback is invoked AFTER the wipe transaction
// commits so it can publish a bus event for the alerting / audit
// services without holding the database lock.
//
// Backwards-compat: the existing two-arg call site
// (`CryptographicWipe(ctx, id)`) is preserved via this signature.
// Callers that have actor identity should switch to
// `CryptographicWipeWithAudit` which carries through who did it
// and why.
func (r *TenantRepository) CryptographicWipe(ctx context.Context, id string) error {
	return r.CryptographicWipeWithAudit(ctx, id, "system", "", "no-reason-supplied", nil)
}

// CryptographicWipeWithAudit is the audit-aware deletion path.
// `actorUserID` and `actorRole` describe who initiated the wipe;
// `reason` is a free-form string captured for compliance.
// `auditor` is an optional callback fired post-commit so the caller
// can publish a `tenant:deleted` bus event without coupling this
// package to the eventbus package.
func (r *TenantRepository) CryptographicWipeWithAudit(
	ctx context.Context,
	id, actorUserID, actorRole, reason string,
	auditor func(deletionLogID int64),
) error {
	r.db.Lock()
	defer r.db.Unlock()

	// Read the row first so we can hash it for the audit log.
	conn, err := r.db.Conn()
	if err != nil {
		return err
	}
	var t Tenant
	err = conn.QueryRowContext(ctx,
		`SELECT id, name, tier, status, crypto_salt, created_at, updated_at FROM tenants WHERE id = ?`,
		id,
	).Scan(&t.ID, &t.Name, &t.Tier, &t.Status, &t.CryptoSalt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("read tenant for deletion: %w", err)
	}

	// SHA-256 of the previous row contents (excluding the ID itself
	// since that's stored in tenant_id). Provides tamper evidence:
	// an auditor can later cross-reference the hash against backups.
	rowSummary := fmt.Sprintf("name=%s|tier=%s|status=%s|salt=%s|created=%s|updated=%s",
		t.Name, t.Tier, t.Status, t.CryptoSalt, t.CreatedAt, t.UpdatedAt)
	prevHash := sha256.Sum256([]byte(rowSummary))
	prevHashHex := hex.EncodeToString(prevHash[:])

	now := time.Now().Format(time.RFC3339)

	// Wipe the live row.
	if _, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE tenants SET status = 'Deleted', crypto_salt = '', updated_at = ? WHERE id = ?
	`, now, id); err != nil {
		return fmt.Errorf("crypto wipe: %w", err)
	}

	// Append the deletion log entry. NOT inside a transaction with the
	// wipe — if the log write fails we don't want to roll back the
	// wipe (the tenant data really IS gone). Log the failure loudly
	// instead so it shows up in the audit pipeline.
	res, logErr := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO tenant_deletion_log (
			tenant_id, tenant_name, deleted_by_user, deleted_by_role,
			reason, prev_row_hash, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, t.Name, actorUserID, actorRole, reason, prevHashHex, now)
	if logErr != nil {
		// The wipe succeeded but the log write didn't. The caller
		// (typically a service-layer wrapper) is expected to log this
		// at ERROR level and surface it via the audit bus.
		return fmt.Errorf("wipe-applied-but-log-failed: %w", logErr)
	}

	if auditor != nil {
		var logID int64
		if id, ok := lastInsertID(res); ok {
			logID = id
		}
		auditor(logID)
	}
	return nil
}

// lastInsertID is a small helper that pulls LastInsertId off any
// sql.Result, swallowing the error case (some drivers return -1 +
// nil for tables without rowid). Caller treats 0 as "unknown".
func lastInsertID(res interface{ LastInsertId() (int64, error) }) (int64, bool) {
	if res == nil {
		return 0, false
	}
	id, err := res.LastInsertId()
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func (r *TenantRepository) ListAllTenants(ctx context.Context) ([]Tenant, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query("SELECT id, name, tier, status, crypto_salt, created_at, updated_at FROM tenants WHERE status != 'Deleted'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Tier, &t.Status, &t.CryptoSalt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}
