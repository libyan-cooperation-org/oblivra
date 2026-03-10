package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/integrity"
)

// AuditRepository handles audit log operations with cryptographic integrity.
type AuditRepository struct {
	db   DatabaseStore
	tree *integrity.MerkleTree
}

// NewAuditRepository creates a new audit repository and initializes its Merkle tree.
func NewAuditRepository(db DatabaseStore) *AuditRepository {
	return &AuditRepository{
		db:   db,
		tree: integrity.New(),
	}
}

// InitIntegrity rebuilds the Merkle tree from existing persistent logs.
func (r *AuditRepository) InitIntegrity(ctx context.Context) error {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return err
	}

	rows, err := conn.Query("SELECT merkle_hash FROM audit_logs WHERE merkle_hash IS NOT NULL ORDER BY merkle_index ASC")
	if err != nil {
		return fmt.Errorf("query audit hashes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err == nil {
			r.tree.LoadLeaf(h)
		}
	}
	return rows.Err()
}

// Log records an audit event and secures it with a Merkle hash.
func (r *AuditRepository) Log(ctx context.Context, eventType string, hostID string, sessionID string, details map[string]interface{}) error {
	r.db.Lock()
	defer r.db.Unlock()

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	// Add to Merkle Tree for tamper-evidence
	payload := fmt.Sprintf("%s|%s|%s|%s", eventType, hostID, sessionID, string(detailsJSON))
	hash, index, err := r.tree.AddLeaf([]byte(payload))
	if err != nil {
		return fmt.Errorf("merkle add: %w", err)
	}

	tenantID := TenantFromContext(ctx)

	_, err = r.db.ReplicatedExecContext(ctx, `
		INSERT INTO audit_logs (tenant_id, timestamp, event_type, host_id, session_id, details, merkle_hash, merkle_index)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tenantID, time.Now(), eventType, hostID, sessionID, string(detailsJSON), hash, index,
	)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}

// GetRecent returns the most recent audit logs.
func (r *AuditRepository) GetRecent(ctx context.Context, limit int) ([]AuditLog, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, timestamp, event_type,
			COALESCE(host_id, ''), COALESCE(session_id, ''),
			details, COALESCE(ip_address, ''),
			COALESCE(merkle_hash, ''), COALESCE(merkle_index, 0)
		FROM audit_logs
		WHERE tenant_id = ?
		ORDER BY timestamp DESC
		LIMIT ?`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		log, err := r.scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, *log)
	}

	return logs, rows.Err()
}

// GetByDateRange returns logs within a specific time window.
func (r *AuditRepository) GetByDateRange(ctx context.Context, from, to time.Time, limit int) ([]AuditLog, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, timestamp, event_type,
			COALESCE(host_id, ''), COALESCE(session_id, ''),
			details, COALESCE(ip_address, ''),
			COALESCE(merkle_hash, ''), COALESCE(merkle_index, 0)
		FROM audit_logs
		WHERE tenant_id = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
		LIMIT ?`, tenantID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("query audit logs by date: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		log, err := r.scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, *log)
	}

	return logs, rows.Err()
}

// Count returns the total number of logs.
func (r *AuditRepository) Count(ctx context.Context) (int64, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return 0, err
	}

	tenantID := TenantFromContext(ctx)
	var count int64
	err = conn.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE tenant_id = ?", tenantID).Scan(&count)
	return count, err
}

// ValidateIntegrity checks if the Merkle tree is valid (has at least one leaf if logs exist).
func (r *AuditRepository) ValidateIntegrity(ctx context.Context) bool {
	r.db.RLock()
	defer r.db.RUnlock()

	count, _ := r.Count(ctx)
	if count == 0 {
		return true // Empty is valid
	}
	return r.tree.LeafCount() > 0 && r.tree.Root() != ""
}

// Export returns audit logs as a JSON blob.
func (r *AuditRepository) Export(ctx context.Context, from, to time.Time) ([]byte, error) {
	logs, err := r.GetByDateRange(ctx, from, to, 100000)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(logs, "", "  ")
}

func (r *AuditRepository) scanAuditLog(rows *sql.Rows) (*AuditLog, error) {
	var log AuditLog
	err := rows.Scan(
		&log.ID, &log.TenantID, &log.Timestamp, &log.EventType,
		&log.HostID, &log.SessionID, &log.Details, &log.IPAddress,
		&log.MerkleHash, &log.MerkleIndex,
	)
	if err != nil {
		return nil, fmt.Errorf("scan audit log: %w", err)
	}
	return &log, nil
}
