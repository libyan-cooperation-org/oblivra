package database

import (
	"context"
	"database/sql"
	"time"
)

// SessionRepository handles session CRUD
type SessionRepository struct {
	db DatabaseStore
}

func NewSessionRepository(db DatabaseStore) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *Session) error {
	session.TenantID = TenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO sessions (id, tenant_id, host_id, started_at, status)
		VALUES (?, ?, ?, ?, ?)`,
		session.ID, session.TenantID, session.HostID, session.StartedAt, session.Status,
	)
	return err
}

func (r *SessionRepository) End(ctx context.Context, id string, status string, bytesSent, bytesReceived int64) error {
	tenantID := TenantFromContext(ctx)

	now := time.Now()
	_, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE sessions SET
			ended_at = ?,
			status = ?,
			bytes_sent = ?,
			bytes_received = ?,
			duration_seconds = CAST((julianday(?) - julianday(started_at)) * 86400 AS INTEGER)
		WHERE id = ? AND tenant_id = ?`,
		now, status, bytesSent, bytesReceived, now, id, tenantID,
	)
	return err
}

func (r *SessionRepository) GetRecent(ctx context.Context, limit int) ([]Session, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, host_id, started_at, ended_at, duration_seconds,
			bytes_sent, bytes_received, status, recording_path
		FROM sessions
		WHERE tenant_id = ?
		ORDER BY started_at DESC
		LIMIT ?`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var endedAt sql.NullTime

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.HostID, &s.StartedAt, &endedAt,
			&s.DurationSeconds, &s.BytesSent, &s.BytesReceived,
			&s.Status, &s.RecordingPath,
		)
		if err != nil {
			return nil, err
		}

		if endedAt.Valid {
			s.EndedAt = &endedAt.Time
		}

		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}

func (r *SessionRepository) GetByHostID(ctx context.Context, hostID string, limit int) ([]Session, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, host_id, started_at, ended_at, duration_seconds,
			bytes_sent, bytes_received, status, recording_path
		FROM sessions WHERE host_id = ? AND tenant_id = ?
		ORDER BY started_at DESC
		LIMIT ?`, hostID, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var endedAt sql.NullTime

		err := rows.Scan(
			&s.ID, &s.TenantID, &s.HostID, &s.StartedAt, &endedAt,
			&s.DurationSeconds, &s.BytesSent, &s.BytesReceived,
			&s.Status, &s.RecordingPath,
		)
		if err != nil {
			return nil, err
		}

		if endedAt.Valid {
			s.EndedAt = &endedAt.Time
		}

		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}
