// Package identity — temporal entity resolution.
//
// Phase 27.2.2 — closes the "DHCP lease churn" gap from the audit.
//
// The problem: when an alert fires at time T citing IP 10.0.4.7, the
// SIEM needs to map that IP back to the LAPTOP THAT HELD THE LEASE AT
// TIME T, not whichever device happens to hold it now. Without this,
// alert attribution silently breaks every time DHCP rotates a lease.
//
// The lease ledger is an append-only SQLite-backed log of lease
// transitions:
//
//	id INTEGER PRIMARY KEY AUTOINCREMENT
//	tenant_id   TEXT NOT NULL
//	ip          TEXT NOT NULL
//	hostname    TEXT
//	mac         TEXT
//	started_at  DATETIME NOT NULL  -- when the lease BEGAN
//	ended_at    DATETIME            -- nullable; null = still active
//	source      TEXT                -- 'dhcp', 'agent_observed', etc.
//
// Records are NEVER updated except to set ended_at when a successor
// lease for the same (tenant, ip) is recorded — that's the lease
// expiration event. The history is forensically immutable.
//
// Lookup model:
//
//	LeaseLedger.LookupAtTime(ctx, tenantID, ip, ts) →
//	  the (hostname, mac) that owned `ip` at instant `ts`
//
// The lookup returns "" + "" + nil error when there's no record
// covering that time — caller decides whether to fail open or closed.

package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrLeaseNotFound is returned by LookupAtTime when no lease record
// covers the requested (ip, ts) pair. Callers may fall back to
// "current owner" semantics or surface as enrichment-missing.
var ErrLeaseNotFound = errors.New("identity: no lease record covers requested time")

// Lease represents a single (ip, hostname, mac) binding over a time
// interval. `EndedAt` is the zero-value when the lease is still
// active — callers should compare via .IsZero().
type Lease struct {
	ID        int64
	TenantID  string
	IP        string
	Hostname  string
	MAC       string
	StartedAt time.Time
	EndedAt   time.Time
	Source    string
}

// LeaseLedger persists and queries DHCP lease history. Backed by the
// platform's primary SQLite database; the migration that creates the
// `dhcp_lease_log` table lands as a separate schema-version bump
// alongside this code.
type LeaseLedger struct {
	db *sql.DB
}

// NewLeaseLedger constructs a ledger over the given DB handle.
func NewLeaseLedger(db *sql.DB) *LeaseLedger {
	return &LeaseLedger{db: db}
}

// EnsureSchema creates the lease table on first use. Idempotent.
// Called by the database migration runner — production callers
// don't invoke this directly. Defined here so tests can spin up a
// fresh in-memory database without pulling the full migrations
// graph.
func (l *LeaseLedger) EnsureSchema(ctx context.Context) error {
	_, err := l.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS dhcp_lease_log (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id   TEXT NOT NULL,
			ip          TEXT NOT NULL,
			hostname    TEXT,
			mac         TEXT,
			started_at  DATETIME NOT NULL,
			ended_at    DATETIME,
			source      TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_dhcp_lease_lookup
			ON dhcp_lease_log(tenant_id, ip, started_at, ended_at);
	`)
	return err
}

// Record inserts a new lease binding. If a prior lease for the same
// (tenant, ip) is currently open (ended_at IS NULL) AND its hostname
// or MAC differs from the new record, that prior lease is closed at
// `started`. This implements DHCP lease churn semantics: the lease
// transitions atomically, and history is preserved.
//
// `started` should be the wall-clock time the new lease became
// effective. Pass the current time when ingesting from a live
// observation; pass the original timestamp when backfilling.
func (l *LeaseLedger) Record(
	ctx context.Context,
	tenantID, ip, hostname, mac string,
	started time.Time,
	source string,
) error {
	if tenantID == "" || ip == "" {
		return fmt.Errorf("lease record: tenant_id and ip are required")
	}
	if started.IsZero() {
		started = time.Now().UTC()
	}

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("lease record: begin tx: %w", err)
	}
	// Defensive rollback — committed below on success.
	defer func() { _ = tx.Rollback() }()

	// Find the currently-open lease for this (tenant, ip), if any.
	var (
		openID       sql.NullInt64
		openHostname sql.NullString
		openMAC      sql.NullString
	)
	err = tx.QueryRowContext(ctx, `
		SELECT id, hostname, mac FROM dhcp_lease_log
		 WHERE tenant_id = ? AND ip = ? AND ended_at IS NULL
		 ORDER BY started_at DESC LIMIT 1
	`, tenantID, ip).Scan(&openID, &openHostname, &openMAC)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("lease record: lookup open lease: %w", err)
	}

	// If an open lease exists with the SAME hostname+mac, it's a refresh —
	// no-op. If it differs, close the old lease at the new `started`.
	if openID.Valid {
		if openHostname.String == hostname && openMAC.String == mac {
			return tx.Commit()
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE dhcp_lease_log SET ended_at = ? WHERE id = ?
		`, started.UTC().Format(time.RFC3339Nano), openID.Int64); err != nil {
			return fmt.Errorf("lease record: close prior lease: %w", err)
		}
	}

	// Insert the new lease.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO dhcp_lease_log
			(tenant_id, ip, hostname, mac, started_at, source)
		VALUES (?, ?, ?, ?, ?, ?)
	`, tenantID, ip, hostname, mac, started.UTC().Format(time.RFC3339Nano), source); err != nil {
		return fmt.Errorf("lease record: insert: %w", err)
	}
	return tx.Commit()
}

// LookupAtTime returns the (hostname, mac) bound to `ip` at the
// given instant `ts`, scoped to `tenantID`. Returns ErrLeaseNotFound
// when no record covers that time.
//
// Time-coverage semantics:
//   started_at <= ts AND (ended_at IS NULL OR ended_at > ts)
//
// The strict-less-than on ended_at means the lease covers the half-
// open interval [started, ended) — matching DHCP wire semantics
// where the new lease takes effect at exactly the cut-over instant.
func (l *LeaseLedger) LookupAtTime(
	ctx context.Context,
	tenantID, ip string,
	ts time.Time,
) (Lease, error) {
	if tenantID == "" || ip == "" {
		return Lease{}, fmt.Errorf("lease lookup: tenant_id and ip are required")
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	tsStr := ts.UTC().Format(time.RFC3339Nano)

	var (
		ls       Lease
		hn, mac  sql.NullString
		ended    sql.NullString
		source   sql.NullString
		started  string
	)
	err := l.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, ip, hostname, mac, started_at, ended_at, source
		  FROM dhcp_lease_log
		 WHERE tenant_id = ?
		   AND ip = ?
		   AND started_at <= ?
		   AND (ended_at IS NULL OR ended_at > ?)
		 ORDER BY started_at DESC LIMIT 1
	`, tenantID, ip, tsStr, tsStr).Scan(
		&ls.ID, &ls.TenantID, &ls.IP, &hn, &mac, &started, &ended, &source,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Lease{}, ErrLeaseNotFound
		}
		return Lease{}, fmt.Errorf("lease lookup: %w", err)
	}
	if hn.Valid {
		ls.Hostname = hn.String
	}
	if mac.Valid {
		ls.MAC = mac.String
	}
	if source.Valid {
		ls.Source = source.String
	}
	if t, perr := time.Parse(time.RFC3339Nano, started); perr == nil {
		ls.StartedAt = t
	}
	if ended.Valid {
		if t, perr := time.Parse(time.RFC3339Nano, ended.String); perr == nil {
			ls.EndedAt = t
		}
	}
	return ls, nil
}

// History returns every lease ever recorded for (tenant, ip), most
// recent first. Used by the operator-facing "show me every box that
// ever held this IP" workflow on the HostDetail page.
func (l *LeaseLedger) History(ctx context.Context, tenantID, ip string, limit int) ([]Lease, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := l.db.QueryContext(ctx, `
		SELECT id, tenant_id, ip, hostname, mac, started_at, ended_at, source
		  FROM dhcp_lease_log
		 WHERE tenant_id = ? AND ip = ?
		 ORDER BY started_at DESC LIMIT ?
	`, tenantID, ip, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Lease
	for rows.Next() {
		var (
			ls         Lease
			hn, mac    sql.NullString
			ended      sql.NullString
			source     sql.NullString
			started    string
		)
		if err := rows.Scan(&ls.ID, &ls.TenantID, &ls.IP, &hn, &mac, &started, &ended, &source); err != nil {
			return nil, err
		}
		if hn.Valid {
			ls.Hostname = hn.String
		}
		if mac.Valid {
			ls.MAC = mac.String
		}
		if source.Valid {
			ls.Source = source.String
		}
		if t, perr := time.Parse(time.RFC3339Nano, started); perr == nil {
			ls.StartedAt = t
		}
		if ended.Valid {
			if t, perr := time.Parse(time.RFC3339Nano, ended.String); perr == nil {
				ls.EndedAt = t
			}
		}
		out = append(out, ls)
	}
	return out, rows.Err()
}
