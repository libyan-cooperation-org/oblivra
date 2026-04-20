package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type SnippetRepository struct {
	db DatabaseStore
}

func NewSnippetRepository(db DatabaseStore) *SnippetRepository {
	return &SnippetRepository{db: db}
}

// List returns all snippets
func (r *SnippetRepository) List(ctx context.Context) ([]Snippet, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)

	rows, err := conn.Query(`
		SELECT id, tenant_id, title, command, description, tags, variables, use_count, created_at, updated_at
		FROM snippets WHERE tenant_id = ? ORDER BY use_count DESC, title ASC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query snippets: %w", err)
	}
	defer rows.Close()

	var snippets []Snippet
	for rows.Next() {
		var s Snippet
		var tagsJSON, variablesJSON string
		if err := rows.Scan(
			&s.ID, &s.TenantID, &s.Title, &s.Command, &s.Description,
			&tagsJSON, &variablesJSON, &s.UseCount, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan snippet: %w", err)
		}

		if err := json.Unmarshal([]byte(tagsJSON), &s.Tags); err != nil {
			s.Tags = []string{}
		}
		if err := json.Unmarshal([]byte(variablesJSON), &s.Variables); err != nil {
			s.Variables = []string{}
		}

		snippets = append(snippets, s)
	}
	return snippets, rows.Err()
}

func (r *SnippetRepository) Get(ctx context.Context, id string) (Snippet, error) {
	var s Snippet
	conn, err := r.db.Conn()
	if err != nil {
		return s, err
	}

	tenantID := MustTenantFromContext(ctx)

	var tagsJSON, variablesJSON string
	err = conn.QueryRow(`
		SELECT id, tenant_id, title, command, description, tags, variables, use_count, created_at, updated_at
		FROM snippets WHERE id = ? AND tenant_id = ?
	`, id, tenantID).Scan(
		&s.ID, &s.TenantID, &s.Title, &s.Command, &s.Description,
		&tagsJSON, &variablesJSON, &s.UseCount, &s.CreatedAt, &s.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return s, fmt.Errorf("snippet not found: %s", id)
		}
		return s, fmt.Errorf("get snippet: %w", err)
	}

	if err := json.Unmarshal([]byte(tagsJSON), &s.Tags); err != nil {
		s.Tags = []string{}
	}
	if err := json.Unmarshal([]byte(variablesJSON), &s.Variables); err != nil {
		s.Variables = []string{}
	}
	return s, nil
}

// Create inserts a new snippet
func (r *SnippetRepository) Create(ctx context.Context, s *Snippet) error {
	tagsJSON, err := json.Marshal(s.Tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}
	variablesJSON, err := json.Marshal(s.Variables)
	if err != nil {
		variablesJSON = []byte("[]")
	}
	now := time.Now().Format(time.RFC3339)
	s.CreatedAt = now
	s.UpdatedAt = now
	s.TenantID = MustTenantFromContext(ctx)

	_, err = r.db.ReplicatedExecContext(ctx, `
		INSERT INTO snippets (id, tenant_id, title, command, description, tags, variables, use_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.TenantID, s.Title, s.Command, s.Description, string(tagsJSON), string(variablesJSON), s.UseCount, now, now)

	if err != nil {
		return fmt.Errorf("create snippet: %w", err)
	}
	return nil
}

// Update saves snippet changes
func (r *SnippetRepository) Update(ctx context.Context, s *Snippet) error {

	tagsJSON, err := json.Marshal(s.Tags)
	if err != nil {
		tagsJSON = []byte("[]")
	}
	variablesJSON, err := json.Marshal(s.Variables)
	if err != nil {
		variablesJSON = []byte("[]")
	}
	now := time.Now().Format(time.RFC3339)
	s.UpdatedAt = now
	tenantID := MustTenantFromContext(ctx)

	res, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE snippets SET title = ?, command = ?, description = ?, tags = ?, variables = ?, updated_at = ?
		WHERE id = ? AND tenant_id = ?
	`, s.Title, s.Command, s.Description, string(tagsJSON), string(variablesJSON), now, s.ID, tenantID)

	if err != nil {
		return fmt.Errorf("update snippet: %w", err)
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return fmt.Errorf("snippet not found: %s", s.ID)
	}
	return nil
}

// Delete removes a snippet
func (r *SnippetRepository) Delete(ctx context.Context, id string) error {
	tenantID := MustTenantFromContext(ctx)
	_, err := r.db.ReplicatedExecContext(ctx, `DELETE FROM snippets WHERE id = ? AND tenant_id = ?`, id, tenantID)
	return err
}

// IncrementUseCount adds 1 to the use count
func (r *SnippetRepository) IncrementUseCount(ctx context.Context, id string) error {
	tenantID := MustTenantFromContext(ctx)
	_, err := r.db.ReplicatedExecContext(ctx, `UPDATE snippets SET use_count = use_count + 1 WHERE id = ? AND tenant_id = ?`, id, tenantID)
	return err
}
