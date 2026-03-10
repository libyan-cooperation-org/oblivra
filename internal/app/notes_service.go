package app

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/notes"
)

// NotesService handles host notes and runbooks operations
type NotesService struct {
	BaseService
	ctx     context.Context
	manager *notes.NotesManager
	bus     *eventbus.Bus
	log     *logger.Logger
}

func (s *NotesService) Name() string { return "NotesService" }

func NewNotesService(manager *notes.NotesManager, bus *eventbus.Bus, log *logger.Logger) *NotesService {
	return &NotesService{
		manager: manager,
		bus:     bus,
		log:     log.WithPrefix("notes-svc"),
	}
}

func (s *NotesService) Startup(ctx context.Context) {
	s.ctx = ctx
	s.log.Info("Notes service started")
}

// GetNotes returns all notes for a host
func (s *NotesService) GetNotes(hostID string) []notes.Note {
	if s.manager == nil {
		return nil
	}
	return s.manager.GetNotes(hostID)
}

// GetNote returns a specific note
func (s *NotesService) GetNote(id string) (*notes.Note, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("notes manager not initialized")
	}
	return s.manager.GetNote(id)
}

// SaveNote saves a note
func (s *NotesService) SaveNote(note notes.Note) (*notes.Note, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("notes manager not initialized")
	}
	n, err := s.manager.SaveNote(note)
	if err == nil {
		s.bus.Publish(eventbus.EventType("note:updated"), n)
	}
	return n, err
}

// DeleteNote deleting a note
func (s *NotesService) DeleteNote(id string) error {
	if s.manager == nil {
		return fmt.Errorf("notes manager not initialized")
	}
	err := s.manager.DeleteNote(id)
	if err == nil {
		s.bus.Publish(eventbus.EventType("note:deleted"), id)
	}
	return err
}

// GetRunbooks returns runbooks for a host
func (s *NotesService) GetRunbooks(hostID string) []notes.Runbook {
	if s.manager == nil {
		return nil
	}
	return s.manager.GetRunbooks(hostID)
}

// GetRunbook returns a specific runbook
func (s *NotesService) GetRunbook(id string) (*notes.Runbook, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("notes manager not initialized")
	}
	return s.manager.GetRunbook(id)
}

// SaveRunbook saves a runbook
func (s *NotesService) SaveRunbook(rb notes.Runbook) (*notes.Runbook, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("notes manager not initialized")
	}
	r, err := s.manager.SaveRunbook(rb)
	if err == nil {
		s.bus.Publish(eventbus.EventType("runbook:updated"), r)
	}
	return r, err
}

// DeleteRunbook deletes a runbook
func (s *NotesService) DeleteRunbook(id string) error {
	if s.manager == nil {
		return fmt.Errorf("notes manager not initialized")
	}
	err := s.manager.DeleteRunbook(id)
	if err == nil {
		s.bus.Publish(eventbus.EventType("runbook:deleted"), id)
	}
	return err
}

// Search searches across notes and runbooks
func (s *NotesService) Search(query string) map[string]interface{} {
	if s.manager == nil {
		return map[string]interface{}{"notes": nil, "runbooks": nil}
	}
	notesList, runbookList := s.manager.Search(query)
	return map[string]interface{}{
		"notes":    notesList,
		"runbooks": runbookList,
	}
}
