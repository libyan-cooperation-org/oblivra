package workspace

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/database"
)

// PaneLayout defines how terminal panes are arranged
type PaneLayout struct {
	ID        string       `json:"id"`
	Type      string       `json:"type"` // "terminal", "split-h", "split-v"
	HostID    string       `json:"host_id,omitempty"`
	HostLabel string       `json:"host_label,omitempty"`
	Children  []PaneLayout `json:"children,omitempty"`
	Size      float64      `json:"size"` // 0.0 - 1.0, relative size
}

// Workspace represents a saveable session layout
type Workspace struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Description  string                `json:"description,omitempty"`
	Layout       PaneLayout            `json:"layout"`
	Connections  []WorkspaceConnection `json:"connections"`
	SidebarOpen  bool                  `json:"sidebar_open"`
	SidebarWidth int                   `json:"sidebar_width"`
	ActiveTab    string                `json:"active_tab,omitempty"`
	Theme        string                `json:"theme,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
	IsDefault    bool                  `json:"is_default"`
	Icon         string                `json:"icon,omitempty"`
	Tags         []string              `json:"tags,omitempty"`
}

// WorkspaceConnection defines a connection within a workspace
type WorkspaceConnection struct {
	HostID       string            `json:"host_id"`
	PaneID       string            `json:"pane_id"`
	AutoConnect  bool              `json:"auto_connect"`
	InitCommands []string          `json:"init_commands,omitempty"`
	Environment  map[string]string `json:"environment,omitempty"`
}

type WorkspaceManager struct {
	mu         sync.RWMutex
	repo       *database.WorkspaceRepository
	workspaces map[string]*Workspace
	activeID   string
}

func NewWorkspaceManager(repo *database.WorkspaceRepository) *WorkspaceManager {
	wm := &WorkspaceManager{
		repo:       repo,
		workspaces: make(map[string]*Workspace),
	}

	wm.load()

	// Create default workspace if none exist
	if len(wm.workspaces) == 0 {
		defaultWS := &Workspace{
			ID:          "default",
			Name:        "Default",
			Description: "Default workspace",
			Layout: PaneLayout{
				ID:   "root",
				Type: "terminal",
				Size: 1.0,
			},
			SidebarOpen:  true,
			SidebarWidth: 260,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			IsDefault:    true,
			Icon:         "🏠",
		}
		wm.workspaces[defaultWS.ID] = defaultWS
		wm.activeID = defaultWS.ID
		wm.save()
	}

	return wm
}

// GetAll returns all workspaces
func (wm *WorkspaceManager) GetAll() []Workspace {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	result := make([]Workspace, 0, len(wm.workspaces))
	for _, ws := range wm.workspaces {
		result = append(result, *ws)
	}
	return result
}

// GetByID returns a specific workspace
func (wm *WorkspaceManager) GetByID(id string) (*Workspace, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	ws, ok := wm.workspaces[id]
	if !ok {
		return nil, fmt.Errorf("workspace %s not found", id)
	}
	return ws, nil
}

// GetActive returns the active workspace
func (wm *WorkspaceManager) GetActive() *Workspace {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	if ws, ok := wm.workspaces[wm.activeID]; ok {
		return ws
	}

	// Return first workspace
	for _, ws := range wm.workspaces {
		return ws
	}
	return nil
}

// Create creates a new workspace
func (wm *WorkspaceManager) Create(name, description, icon string) (*Workspace, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	ws := &Workspace{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Icon:        icon,
		Layout: PaneLayout{
			ID:   "root",
			Type: "terminal",
			Size: 1.0,
		},
		SidebarOpen:  true,
		SidebarWidth: 260,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if ws.Icon == "" {
		ws.Icon = "📋"
	}

	wm.workspaces[ws.ID] = ws
	wm.save()

	return ws, nil
}

// SaveCurrent saves the current layout as a workspace
func (wm *WorkspaceManager) SaveCurrent(
	id string,
	layout PaneLayout,
	connections []WorkspaceConnection,
	sidebarOpen bool,
	sidebarWidth int,
	activeTab string,
) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	ws, ok := wm.workspaces[id]
	if !ok {
		return fmt.Errorf("workspace %s not found", id)
	}

	ws.Layout = layout
	ws.Connections = connections
	ws.SidebarOpen = sidebarOpen
	ws.SidebarWidth = sidebarWidth
	ws.ActiveTab = activeTab
	ws.UpdatedAt = time.Now()

	return wm.save()
}

// Activate switches to a workspace
func (wm *WorkspaceManager) Activate(id string) (*Workspace, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	ws, ok := wm.workspaces[id]
	if !ok {
		return nil, fmt.Errorf("workspace %s not found", id)
	}

	wm.activeID = id
	wm.save()

	return ws, nil
}

// Duplicate creates a copy of an existing workspace
func (wm *WorkspaceManager) Duplicate(id string) (*Workspace, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	src, ok := wm.workspaces[id]
	if !ok {
		return nil, fmt.Errorf("workspace %s not found", id)
	}

	// Deep copy via JSON serialization
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	var dup Workspace
	if err := json.Unmarshal(data, &dup); err != nil {
		return nil, err
	}

	dup.ID = uuid.New().String()
	dup.Name = src.Name + " (Copy)"
	dup.IsDefault = false
	dup.CreatedAt = time.Now()
	dup.UpdatedAt = time.Now()

	wm.workspaces[dup.ID] = &dup
	wm.save()

	return &dup, nil
}

// Rename renames a workspace
func (wm *WorkspaceManager) Rename(id, name string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	ws, ok := wm.workspaces[id]
	if !ok {
		return fmt.Errorf("workspace %s not found", id)
	}

	ws.Name = name
	ws.UpdatedAt = time.Now()
	return wm.save()
}

// Delete removes a workspace
func (wm *WorkspaceManager) Delete(id string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	ws, ok := wm.workspaces[id]
	if !ok {
		return fmt.Errorf("workspace %s not found", id)
	}

	if ws.IsDefault {
		return fmt.Errorf("cannot delete default workspace")
	}

	delete(wm.workspaces, id)

	// If active workspace was deleted switch to default
	if wm.activeID == id {
		for wid, w := range wm.workspaces {
			if w.IsDefault {
				wm.activeID = wid
				break
			}
		}
	}

	return wm.save()
}

// ExportWorkspace exports a workspace as JSON
func (wm *WorkspaceManager) ExportWorkspace(id string) ([]byte, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	ws, ok := wm.workspaces[id]
	if !ok {
		return nil, fmt.Errorf("workspace %s not found", id)
	}

	return json.MarshalIndent(ws, "", "  ")
}

// ImportWorkspace imports a workspace from JSON
func (wm *WorkspaceManager) ImportWorkspace(data []byte) (*Workspace, error) {
	var ws Workspace
	if err := json.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("invalid workspace data: %w", err)
	}

	ws.ID = uuid.New().String()
	ws.IsDefault = false
	ws.CreatedAt = time.Now()
	ws.UpdatedAt = time.Now()

	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.workspaces[ws.ID] = &ws
	wm.save()

	return &ws, nil
}

func (wm *WorkspaceManager) load() {
	rows, err := wm.repo.GetAll()
	if err != nil {
		return
	}

	for _, row := range rows {
		var layout PaneLayout
		json.Unmarshal([]byte(row.LayoutJSON), &layout)

		var connections []WorkspaceConnection
		json.Unmarshal([]byte(row.ConnectionsJSON), &connections)

		ws := &Workspace{
			ID:           row.ID,
			Name:         row.Name,
			Description:  row.Description.String,
			Layout:       layout,
			Connections:  connections,
			SidebarOpen:  row.SidebarOpen,
			SidebarWidth: row.SidebarWidth,
			ActiveTab:    row.ActiveTab.String,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
			IsDefault:    row.IsDefault,
			Icon:         row.Icon.String,
		}
		wm.workspaces[ws.ID] = ws
		if ws.IsDefault && wm.activeID == "" {
			wm.activeID = ws.ID
		}
	}
}

func (wm *WorkspaceManager) save() error {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	for _, ws := range wm.workspaces {
		layoutJSON, _ := json.Marshal(ws.Layout)
		connJSON, _ := json.Marshal(ws.Connections)

		row := &database.WorkspaceRow{
			ID:              ws.ID,
			Name:            ws.Name,
			LayoutJSON:      string(layoutJSON),
			ConnectionsJSON: string(connJSON),
			SidebarOpen:     ws.SidebarOpen,
			SidebarWidth:    ws.SidebarWidth,
			IsDefault:       ws.IsDefault,
			CreatedAt:       ws.CreatedAt,
			UpdatedAt:       ws.UpdatedAt,
		}
		if ws.Description != "" {
			row.Description = sql.NullString{String: ws.Description, Valid: true}
		}
		if ws.ActiveTab != "" {
			row.ActiveTab = sql.NullString{String: ws.ActiveTab, Valid: true}
		}
		if ws.Icon != "" {
			row.Icon = sql.NullString{String: ws.Icon, Valid: true}
		}

		// Simple Upsert logic: attempt create, then update on conflict (though repository currently splits them)
		// For now, let's just use Update which our repo has, we'd need a Get to check existence
		// Refined repo usage:
		existing, _ := wm.repo.GetAll() // Not efficient but works for now
		found := false
		for _, e := range existing {
			if e.ID == ws.ID {
				found = true
				break
			}
		}

		if found {
			wm.repo.Update(row)
		} else {
			wm.repo.Create(row)
		}
	}
	return nil
}
