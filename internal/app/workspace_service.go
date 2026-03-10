package app

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/workspace"
)

// WorkspaceService handles workspace related operations
type WorkspaceService struct {
	BaseService
	ctx     context.Context
	manager *workspace.WorkspaceManager
	bus     *eventbus.Bus
	log     *logger.Logger
}

func (s *WorkspaceService) Name() string { return "WorkspaceService" }

func NewWorkspaceService(manager *workspace.WorkspaceManager, bus *eventbus.Bus, log *logger.Logger) *WorkspaceService {
	return &WorkspaceService{
		manager: manager,
		bus:     bus,
		log:     log.WithPrefix("workspace-svc"),
	}
}

func (s *WorkspaceService) Startup(ctx context.Context) {
	s.ctx = ctx
	s.log.Info("Workspace service started")
}

// GetAll returns all workspaces
func (s *WorkspaceService) GetAll() []workspace.Workspace {
	if s.manager == nil {
		return nil
	}
	return s.manager.GetAll()
}

// GetActive returns the active workspace
func (s *WorkspaceService) GetActive() *workspace.Workspace {
	if s.manager == nil {
		return nil
	}
	return s.manager.GetActive()
}

// Create creates a new workspace
func (s *WorkspaceService) Create(name, description, icon string) (*workspace.Workspace, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("workspace manager not initialized")
	}
	ws, err := s.manager.Create(name, description, icon)
	if err == nil {
		s.bus.Publish(eventbus.EventType("workspace:created"), ws)
	}
	return ws, err
}

// SaveCurrent saves the given layout for the specified workspace
func (s *WorkspaceService) SaveCurrent(id string, layout workspace.PaneLayout, connections []workspace.WorkspaceConnection, sidebarOpen bool, sidebarWidth int, activeTab string) error {
	if s.manager == nil {
		return fmt.Errorf("workspace manager not initialized")
	}
	err := s.manager.SaveCurrent(id, layout, connections, sidebarOpen, sidebarWidth, activeTab)
	if err == nil {
		s.bus.Publish(eventbus.EventType("workspace:updated"), map[string]string{"id": id})
	}
	return err
}

// Activate switches to the specified workspace
func (s *WorkspaceService) Activate(id string) (*workspace.Workspace, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("workspace manager not initialized")
	}
	ws, err := s.manager.Activate(id)
	if err == nil {
		s.bus.Publish(eventbus.EventType("workspace:activated"), ws)
	}
	return ws, err
}

// Duplicate duplicates an existing workspace
func (s *WorkspaceService) Duplicate(id string) (*workspace.Workspace, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("workspace manager not initialized")
	}
	ws, err := s.manager.Duplicate(id)
	if err == nil {
		s.bus.Publish(eventbus.EventType("workspace:created"), ws)
	}
	return ws, err
}

// Rename renames an existing workspace
func (s *WorkspaceService) Rename(id, name string) error {
	if s.manager == nil {
		return fmt.Errorf("workspace manager not initialized")
	}
	err := s.manager.Rename(id, name)
	if err == nil {
		s.bus.Publish(eventbus.EventType("workspace:updated"), map[string]string{"id": id})
	}
	return err
}

// Delete deletes the specified workspace
func (s *WorkspaceService) Delete(id string) error {
	if s.manager == nil {
		return fmt.Errorf("workspace manager not initialized")
	}
	err := s.manager.Delete(id)
	if err == nil {
		s.bus.Publish(eventbus.EventType("workspace:deleted"), map[string]string{"id": id})
	}
	return err
}

// ExportWorkspace exports a workspace as JSON bytes
func (s *WorkspaceService) ExportWorkspace(id string) ([]byte, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("workspace manager not initialized")
	}
	return s.manager.ExportWorkspace(id)
}

// ImportWorkspace imports a workspace from JSON bytes
func (s *WorkspaceService) ImportWorkspace(data []byte) (*workspace.Workspace, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("workspace manager not initialized")
	}
	ws, err := s.manager.ImportWorkspace(data)
	if err == nil {
		s.bus.Publish(eventbus.EventType("workspace:created"), ws)
	}
	return ws, err
}
