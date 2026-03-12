package database

import (
	"context"
	"database/sql"

)

type WorkspaceRepository struct {
	db DatabaseStore
}

func NewWorkspaceRepository(db DatabaseStore) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}

type WorkspaceRow struct {
	ID              string
	Name            string
	Description     sql.NullString
	LayoutJSON      string
	ConnectionsJSON string
	SidebarOpen     bool
	SidebarWidth    int
	ActiveTab       sql.NullString
	IsDefault       bool
	Icon            sql.NullString
	CreatedAt       string
	UpdatedAt       string
}

func (r *WorkspaceRepository) Create(ws *WorkspaceRow) error {
	_, err := r.db.ReplicatedExecContext(context.Background(), `
		INSERT INTO workspaces (
			id, name, description, layout_json, connections_json,
			sidebar_open, sidebar_width, active_tab, is_default, icon,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ws.ID, ws.Name, ws.Description, ws.LayoutJSON, ws.ConnectionsJSON,
		ws.SidebarOpen, ws.SidebarWidth, ws.ActiveTab, ws.IsDefault, ws.Icon,
		ws.CreatedAt, ws.UpdatedAt,
	)
	return err
}

func (r *WorkspaceRepository) GetAll() ([]WorkspaceRow, error) {
	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(`
		SELECT id, name, description, layout_json, connections_json,
			sidebar_open, sidebar_width, active_tab, is_default, icon,
			created_at, updated_at
		FROM workspaces ORDER BY is_default DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []WorkspaceRow
	for rows.Next() {
		var ws WorkspaceRow
		err := rows.Scan(
			&ws.ID, &ws.Name, &ws.Description, &ws.LayoutJSON, &ws.ConnectionsJSON,
			&ws.SidebarOpen, &ws.SidebarWidth, &ws.ActiveTab, &ws.IsDefault, &ws.Icon,
			&ws.CreatedAt, &ws.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, ws)
	}
	return workspaces, nil
}

func (r *WorkspaceRepository) Update(ws *WorkspaceRow) error {
	_, err := r.db.ReplicatedExecContext(context.Background(), `
		UPDATE workspaces SET
			name = ?, description = ?, layout_json = ?, connections_json = ?,
			sidebar_open = ?, sidebar_width = ?, active_tab = ?, icon = ?,
			updated_at = ?
		WHERE id = ?`,
		ws.Name, ws.Description, ws.LayoutJSON, ws.ConnectionsJSON,
		ws.SidebarOpen, ws.SidebarWidth, ws.ActiveTab, ws.Icon,
		ws.UpdatedAt, ws.ID,
	)
	return err
}

func (r *WorkspaceRepository) Delete(id string) error {
	_, err := r.db.ReplicatedExecContext(context.Background(), "DELETE FROM workspaces WHERE id = ? AND is_default = 0", id)
	return err
}
