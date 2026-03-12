package database

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/vault"
)

// HostRepository handles host CRUD operations.
// It integrates with the vault to ensure credentials are encrypted at rest.
type HostRepository struct {
	db    DatabaseStore
	vault vault.Provider
}

// NewHostRepository creates a new host repository
func NewHostRepository(db DatabaseStore, v vault.Provider) *HostRepository {
	return &HostRepository{db: db, vault: v}
}

// Create inserts a new host
func (r *HostRepository) Create(ctx context.Context, host *Host) error {

	tags, err := json.Marshal(host.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	host.CreatedAt = now
	host.UpdatedAt = now
	host.TenantID = TenantFromContext(ctx)

	// Security: Encrypt password if vault is available and unlocked
	encryptedPassword := host.Password
	if r.vault != nil && r.vault.IsUnlocked() && host.Password != "" {
		encrypted, err := r.vault.Encrypt([]byte(host.Password))
		if err == nil {
			// We store the encrypted blob as a base64 string in the TEXT column
			encryptedPassword = base64.StdEncoding.EncodeToString(encrypted)
		}
	}

	_, err = r.db.ReplicatedExecContext(ctx, `
		INSERT INTO hosts (
			id, tenant_id, label, hostname, port, username, password, auth_method,
			credential_id, jump_host_id, tags, category, color, notes,
			is_favorite, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		host.ID, host.TenantID, host.Label, host.Hostname, host.Port,
		host.Username, encryptedPassword, host.AuthMethod, host.CredentialID,
		host.JumpHostID, string(tags), host.Category, host.Color, host.Notes,
		host.IsFavorite, host.CreatedAt, host.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert host: %w", err)
	}

	return nil
}

// GetByID retrieves a single host
func (r *HostRepository) GetByID(ctx context.Context, id string) (*Host, error) {

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	row := conn.QueryRow(`
		SELECT id, tenant_id, label, hostname, port, username, COALESCE(password, ''), auth_method,
			COALESCE(credential_id, ''), COALESCE(jump_host_id, ''),
			tags, COALESCE(category, ''), color, notes, is_favorite,
			last_connected_at, connection_count, created_at, updated_at
		FROM hosts WHERE id = ? AND tenant_id = ?`, id, tenantID)

	host, err := r.scanHost(row)
	if err != nil {
		return nil, fmt.Errorf("get host %s: %w (tenant: %s)", id, err, tenantID)
	}
	return host, nil
}

// GetAll returns all hosts
func (r *HostRepository) GetAll(ctx context.Context) ([]Host, error) {

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, label, hostname, port, username, COALESCE(password, ''), auth_method,
			COALESCE(credential_id, ''), COALESCE(jump_host_id, ''),
			tags, COALESCE(category, ''), color, notes, is_favorite,
			last_connected_at, connection_count, created_at, updated_at
		FROM hosts
		WHERE tenant_id = ?
		ORDER BY is_favorite DESC, label ASC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query hosts: %w", err)
	}
	defer rows.Close()

	var hosts []Host
	for rows.Next() {
		host, err := r.scanHostRows(rows)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, *host)
	}

	return hosts, rows.Err()
}

// GetFavorites returns favorite hosts
func (r *HostRepository) GetFavorites(ctx context.Context) ([]Host, error) {

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, label, hostname, port, username, COALESCE(password, ''), auth_method,
			COALESCE(credential_id, ''), COALESCE(jump_host_id, ''),
			tags, COALESCE(category, ''), color, notes, is_favorite,
			last_connected_at, connection_count, created_at, updated_at
		FROM hosts WHERE is_favorite = 1 AND tenant_id = ?
		ORDER BY label ASC`, tenantID) // conn via Conn() to respect vault-lock guard
	if err != nil {
		return nil, fmt.Errorf("query favorites: %w", err)
	}
	defer rows.Close()

	var hosts []Host
	for rows.Next() {
		host, err := r.scanHostRows(rows)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, *host)
	}

	return hosts, rows.Err()
}

// Search finds hosts matching a query
func (r *HostRepository) Search(ctx context.Context, query string) ([]Host, error) {

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)
	searchTerm := "%" + strings.ToLower(query) + "%"

	rows, err := conn.Query(`
		SELECT id, tenant_id, label, hostname, port, username, COALESCE(password, ''), auth_method,
			COALESCE(credential_id, ''), COALESCE(jump_host_id, ''),
			tags, COALESCE(category, ''), color, notes, is_favorite,
			last_connected_at, connection_count, created_at, updated_at
		FROM hosts
		WHERE tenant_id = ? AND (LOWER(label) LIKE ? OR LOWER(hostname) LIKE ?
			OR LOWER(tags) LIKE ? OR LOWER(category) LIKE ? OR LOWER(notes) LIKE ?)
		ORDER BY connection_count DESC, label ASC`,
		tenantID, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("search hosts: %w", err)
	}
	defer rows.Close()

	var hosts []Host
	for rows.Next() {
		host, err := r.scanHostRows(rows)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, *host)
	}

	return hosts, rows.Err()
}

// GetByTag returns hosts with a specific tag
func (r *HostRepository) GetByTag(ctx context.Context, tag string) ([]Host, error) {

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)
	searchTag := fmt.Sprintf("%%%q%%", tag)

	rows, err := conn.Query(`
		SELECT id, tenant_id, label, hostname, port, username, COALESCE(password, ''), auth_method,
			COALESCE(credential_id, ''), COALESCE(jump_host_id, ''),
			tags, COALESCE(category, ''), color, notes, is_favorite,
			last_connected_at, connection_count, created_at, updated_at
		FROM hosts WHERE tags LIKE ? AND tenant_id = ?
		ORDER BY label ASC`, searchTag, tenantID) // conn via Conn() to respect vault-lock guard
	if err != nil {
		return nil, fmt.Errorf("query by tag: %w", err)
	}
	defer rows.Close()

	var hosts []Host
	for rows.Next() {
		host, err := r.scanHostRows(rows)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, *host)
	}

	return hosts, rows.Err()
}

// Update updates a host
func (r *HostRepository) Update(ctx context.Context, host *Host) error {

	tags, err := json.Marshal(host.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	host.UpdatedAt = time.Now().Format(time.RFC3339)
	host.TenantID = TenantFromContext(ctx)

	// Security: Encrypt password if vault is available and unlocked
	encryptedPassword := host.Password
	if r.vault != nil && r.vault.IsUnlocked() && host.Password != "" {
		encrypted, err := r.vault.Encrypt([]byte(host.Password))
		if err == nil {
			encryptedPassword = base64.StdEncoding.EncodeToString(encrypted)
		}
	}

	result, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE hosts SET
			label = ?, hostname = ?, port = ?, username = ?, password = ?,
			auth_method = ?, credential_id = ?, jump_host_id = ?,
			tags = ?, category = ?, color = ?, notes = ?, is_favorite = ?,
			updated_at = ?
		WHERE id = ? AND tenant_id = ?`,
		host.Label, host.Hostname, host.Port, host.Username, encryptedPassword,
		host.AuthMethod, host.CredentialID, host.JumpHostID,
		string(tags), host.Category, host.Color, host.Notes, host.IsFavorite,
		host.UpdatedAt, host.ID, host.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update host: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("host %s not found", host.ID)
	}

	return nil
}

// Delete removes a host
func (r *HostRepository) Delete(ctx context.Context, id string) error {

	tenantID := TenantFromContext(ctx)

	result, err := r.db.ReplicatedExecContext(ctx, "DELETE FROM hosts WHERE id = ? AND tenant_id = ?", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete host: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("host %s not found (tenant: %s)", id, tenantID)
	}

	return nil
}

// ToggleFavorite toggles the favorite status
func (r *HostRepository) ToggleFavorite(ctx context.Context, id string) (bool, error) {

	conn, err := r.db.Conn()
	if err != nil {
		return false, err
	}

	tenantID := TenantFromContext(ctx)

	var current bool
	err = conn.QueryRow("SELECT is_favorite FROM hosts WHERE id = ? AND tenant_id = ?", id, tenantID).Scan(&current)
	if err != nil {
		return false, fmt.Errorf("get favorite status: %w", err)
	}

	newStatus := !current
	_, err = r.db.ReplicatedExecContext(ctx,
		"UPDATE hosts SET is_favorite = ?, updated_at = ? WHERE id = ? AND tenant_id = ?",
		newStatus, time.Now().Format(time.RFC3339), id, tenantID,
	)
	if err != nil {
		return false, fmt.Errorf("toggle favorite: %w", err)
	}

	return newStatus, nil
}

// RecordConnection updates connection stats
func (r *HostRepository) RecordConnection(ctx context.Context, id string) error {

	tenantID := TenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE hosts SET
			last_connected_at = ?,
			connection_count = connection_count + 1,
			updated_at = ?
		WHERE id = ? AND tenant_id = ?`,
		time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339), id, tenantID,
	)
	return err
}

// GetAllTags returns all unique tags
func (r *HostRepository) GetAllTags(ctx context.Context) ([]string, error) {

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)

	rows, err := conn.Query("SELECT tags FROM hosts WHERE tags != '[]' AND tenant_id = ?", tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tagSet := make(map[string]bool)
	for rows.Next() {
		var tagsJSON string
		if err := rows.Scan(&tagsJSON); err != nil {
			continue
		}
		var tags []string
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
			continue
		}
		for _, tag := range tags {
			tagSet[tag] = true
		}
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	return tags, nil
}

// GetEncryptedPassword retrieves the raw stored password blob for a host.
// Used only at SSH connection time to decrypt in-place — never returned to frontend.
func (r *HostRepository) GetEncryptedPassword(ctx context.Context, id string) (string, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return "", err
	}

	tenantID := TenantFromContext(ctx)
	var pw string
	err = conn.QueryRow(
		"SELECT COALESCE(password, '') FROM hosts WHERE id = ? AND tenant_id = ?",
		id, tenantID,
	).Scan(&pw)
	if err != nil {
		return "", fmt.Errorf("get encrypted password: %w", err)
	}
	return pw, nil
}

// Count returns total host count
func (r *HostRepository) Count(ctx context.Context) (int, error) {

	conn, err := r.db.Conn()
	if err != nil {
		return 0, err
	}

	tenantID := TenantFromContext(ctx)

	var count int
	err = conn.QueryRow("SELECT COUNT(*) FROM hosts WHERE tenant_id = ?", tenantID).Scan(&count)
	return count, err
}

// scanHost scans a single row
// SECURITY: password column is scanned but immediately cleared before returning.
// Plaintext passwords must NEVER be sent to the frontend via the Host DTO.
// Decryption happens only inside prepareSSHConfig at connection time.
func (r *HostRepository) scanHost(row *sql.Row) (*Host, error) {
	var host Host
	var tagsJSON string
	var lastConnected sql.NullTime
	var rawPassword string // scanned only to verify column exists; never exposed

	err := row.Scan(
		&host.ID, &host.TenantID, &host.Label, &host.Hostname, &host.Port,
		&host.Username, &rawPassword, &host.AuthMethod, &host.CredentialID,
		&host.JumpHostID, &tagsJSON, &host.Category, &host.Color, &host.Notes,
		&host.IsFavorite, &lastConnected, &host.ConnectionCount,
		&host.CreatedAt, &host.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("host not found")
		}
		return nil, fmt.Errorf("scan host: %w", err)
	}

	// Indicate whether a password is stored without exposing its value.
	// The frontend uses this boolean to show/hide the password auth option.
	host.HasPassword = rawPassword != ""
	host.Password = "" // never send plaintext or ciphertext to frontend

	if lastConnected.Valid {
		t := lastConnected.Time.Format(time.RFC3339)
		host.LastConnectedAt = &t
	}

	json.Unmarshal([]byte(tagsJSON), &host.Tags)
	if host.Tags == nil {
		host.Tags = []string{}
	}

	return &host, nil
}

// scanHostRows scans from multiple rows.
// SECURITY: same contract as scanHost — password is never returned in the DTO.
func (r *HostRepository) scanHostRows(rows *sql.Rows) (*Host, error) {
	var host Host
	var tagsJSON string
	var lastConnected sql.NullTime
	var rawPassword string

	err := rows.Scan(
		&host.ID, &host.TenantID, &host.Label, &host.Hostname, &host.Port,
		&host.Username, &rawPassword, &host.AuthMethod, &host.CredentialID,
		&host.JumpHostID, &tagsJSON, &host.Category, &host.Color, &host.Notes,
		&host.IsFavorite, &lastConnected, &host.ConnectionCount,
		&host.CreatedAt, &host.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan host: %w", err)
	}

	host.HasPassword = rawPassword != ""
	host.Password = ""

	if lastConnected.Valid {
		t := lastConnected.Time.Format(time.RFC3339)
		host.LastConnectedAt = &t
	}

	json.Unmarshal([]byte(tagsJSON), &host.Tags)
	if host.Tags == nil {
		host.Tags = []string{}
	}

	return &host, nil
}
