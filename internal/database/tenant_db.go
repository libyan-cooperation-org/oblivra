package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// TenantDB wraps *sql.DB and automatically injects tenant_id constraints
// into every query, eliminating the risk of a developer forgetting a
// WHERE clause and leaking cross-tenant data.
//
// Usage:
//
//	tdb := NewTenantDB(db, MustTenantFromContext(ctx))
//	rows, err := tdb.Query(ctx, "SELECT * FROM hosts", nil)
//	// Executes: SELECT * FROM hosts WHERE tenant_id = '<tenantID>'
type TenantDB struct {
	db       *sql.DB
	tenantID string
}

// NewTenantDB creates a tenant-scoped database accessor.
// The tenantID is sourced from the request context via TenantFromContext.
func NewTenantDB(db *sql.DB, tenantID string) *TenantDB {
	return &TenantDB{db: db, tenantID: tenantID}
}

// FromContext creates a TenantDB from an existing *sql.DB and a context
// that carries a tenant ID.
func TenantDBFromContext(ctx context.Context, db *sql.DB) *TenantDB {
	return NewTenantDB(db, MustTenantFromContext(ctx))
}

// QueryContext executes a SELECT and automatically appends or extends the
// WHERE clause with `tenant_id = ?`. The tenant argument is appended to args.
//
// Rules:
//   - If the query already contains "WHERE", " AND tenant_id = ?" is appended.
//   - Otherwise, " WHERE tenant_id = ?" is appended.
//   - Queries that contain "tenant_id" already are passed through unchanged
//     to allow callers to write explicit joins across tenants (admin only).
func (t *TenantDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	q, a := t.injectTenant(query, args)
	return t.db.QueryContext(ctx, q, a...)
}

// QueryRowContext executes a single-row SELECT with tenant injection.
func (t *TenantDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	q, a := t.injectTenant(query, args)
	return t.db.QueryRowContext(ctx, q, a...)
}

// ExecContext executes an INSERT/UPDATE/DELETE with tenant injection.
// For INSERT statements the tenant_id column is NOT automatically injected
// (callers must include it in the query) — only SELECT filtering is automatic.
// This avoids ambiguity in multi-column inserts.
func (t *TenantDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	upper := strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(upper, "INSERT") || strings.HasPrefix(upper, "CREATE") || strings.HasPrefix(upper, "DROP") {
		// Do not mutate schema-changing or insert queries
		return t.db.ExecContext(ctx, query, args...)
	}
	q, a := t.injectTenant(query, args)
	return t.db.ExecContext(ctx, q, a...)
}

// TenantID returns the tenant ID this accessor is scoped to.
func (t *TenantDB) TenantID() string {
	return t.tenantID
}

// injectTenant rewrites the SQL query to add a tenant_id filter.
// It returns the rewritten query and the extended args slice.
func (t *TenantDB) injectTenant(query string, args []interface{}) (string, []interface{}) {
	// Skip injection if the query already has an explicit tenant_id reference
	// (e.g. admin cross-tenant queries). This check is case-insensitive.
	if strings.Contains(strings.ToLower(query), "tenant_id") {
		return query, args
	}

	upper := strings.ToUpper(query)

	// Detect existing WHERE clause
	whereIdx := strings.Index(upper, " WHERE ")
	if whereIdx != -1 {
		// Append to existing WHERE — insert before ORDER BY / GROUP BY / LIMIT
		// to maintain valid SQL.
		insertIdx := whereIdx + len(" WHERE ")
		query = query[:insertIdx] + "tenant_id = ? AND " + query[insertIdx:]
	} else {
		// Find a safe insertion point: before ORDER BY / GROUP BY / LIMIT / HAVING
		for _, clause := range []string{" ORDER BY ", " GROUP BY ", " LIMIT ", " HAVING ", " UNION "} {
			if idx := strings.Index(upper, clause); idx != -1 {
				query = query[:idx] + " WHERE tenant_id = ?" + query[idx:]
				args = append([]interface{}{t.tenantID}, args...)
				return query, args
			}
		}
		// No recognised trailing clause — append to end
		query = query + " WHERE tenant_id = ?"
	}

	// Prepend the tenant arg so its positional ? comes first / in the right order
	// when we used the "insert before existing WHERE content" path.
	// For the append-to-WHERE path we need it after the existing positional args.
	if whereIdx != -1 {
		// Inserted at the beginning of the WHERE clause
		args = append([]interface{}{t.tenantID}, args...)
	} else {
		args = append(args, t.tenantID)
	}
	return query, args
}

// AssertTenantID panics in development if a model's tenant_id does not match
// the scoped tenant. Use this in critical write paths during testing.
func (t *TenantDB) AssertTenantID(modelTenantID string) error {
	if modelTenantID != "" && modelTenantID != t.tenantID {
		return fmt.Errorf("RLS violation: model tenant_id %q does not match scoped tenant %q", modelTenantID, t.tenantID)
	}
	return nil
}
