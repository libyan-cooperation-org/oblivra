package notes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/platform"
)

// Note represents a text note attached to a host or standalone
type Note struct {
	ID        string    `json:"id"`
	HostID    string    `json:"host_id,omitempty"` // empty = standalone
	Title     string    `json:"title"`
	Content   string    `json:"content"`  // Markdown
	Category  string    `json:"category"` // "runbook", "note", "incident", "config"
	Tags      []string  `json:"tags"`
	IsPinned  bool      `json:"is_pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RunbookStep is a single step in a runbook
type RunbookStep struct {
	Order       int    `json:"order"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`  // executable command
	Expected    string `json:"expected,omitempty"` // expected output
	Warning     string `json:"warning,omitempty"`
	IsOptional  bool   `json:"is_optional"`
}

// Runbook is a structured guide
type Runbook struct {
	ID          string        `json:"id"`
	HostID      string        `json:"host_id,omitempty"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Steps       []RunbookStep `json:"steps"`
	Tags        []string      `json:"tags"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type Storage struct {
	Notes    map[string]*Note    `json:"notes"`
	Runbooks map[string]*Runbook `json:"runbooks"`
}

// NotesManager handles host notes and runbooks
type NotesManager struct {
	mu       sync.RWMutex
	storage  Storage
	savePath string
}

func NewNotesManager() *NotesManager {
	nm := &NotesManager{
		storage: Storage{
			Notes:    make(map[string]*Note),
			Runbooks: make(map[string]*Runbook),
		},
		savePath: filepath.Join(platform.ConfigDir(), "notes.json"),
	}

	nm.load()
	return nm
}

// ---- Notes API ----

func (nm *NotesManager) GetNotes(hostID string) []Note {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	var result []Note
	for _, n := range nm.storage.Notes {
		if hostID == "" || n.HostID == hostID {
			result = append(result, *n)
		}
	}
	return result
}

func (nm *NotesManager) GetNote(id string) (*Note, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if n, ok := nm.storage.Notes[id]; ok {
		return n, nil
	}
	return nil, fmt.Errorf("note %s not found", id)
}

func (nm *NotesManager) SaveNote(note Note) (*Note, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if note.ID == "" {
		note.ID = uuid.New().String()
		note.CreatedAt = time.Now()
	}

	note.UpdatedAt = time.Now()

	n := &note
	nm.storage.Notes[n.ID] = n

	if err := nm.save(); err != nil {
		return nil, err
	}

	return n, nil
}

func (nm *NotesManager) DeleteNote(id string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, ok := nm.storage.Notes[id]; !ok {
		return fmt.Errorf("note %s not found", id)
	}

	delete(nm.storage.Notes, id)
	return nm.save()
}

// ---- Runbooks API ----

func (nm *NotesManager) GetRunbooks(hostID string) []Runbook {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	var result []Runbook
	for _, rb := range nm.storage.Runbooks {
		if hostID == "" || rb.HostID == hostID {
			result = append(result, *rb)
		}
	}
	return result
}

func (nm *NotesManager) GetRunbook(id string) (*Runbook, error) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if rb, ok := nm.storage.Runbooks[id]; ok {
		return rb, nil
	}
	return nil, fmt.Errorf("runbook %s not found", id)
}

func (nm *NotesManager) SaveRunbook(rb Runbook) (*Runbook, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if rb.ID == "" {
		rb.ID = uuid.New().String()
		rb.CreatedAt = time.Now()
	}

	rb.UpdatedAt = time.Now()

	r := &rb
	nm.storage.Runbooks[r.ID] = r

	if err := nm.save(); err != nil {
		return nil, err
	}

	return r, nil
}

func (nm *NotesManager) DeleteRunbook(id string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if _, ok := nm.storage.Runbooks[id]; !ok {
		return fmt.Errorf("runbook %s not found", id)
	}

	delete(nm.storage.Runbooks, id)
	return nm.save()
}

// ---- Search ----

func (nm *NotesManager) Search(query string) ([]Note, []Runbook) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	q := strings.ToLower(query)
	var notes []Note
	var runbooks []Runbook

	for _, n := range nm.storage.Notes {
		if strings.Contains(strings.ToLower(n.Title), q) ||
			strings.Contains(strings.ToLower(n.Content), q) {
			notes = append(notes, *n)
			continue
		}
		for _, tag := range n.Tags {
			if strings.Contains(strings.ToLower(tag), q) {
				notes = append(notes, *n)
				break
			}
		}
	}

	for _, rb := range nm.storage.Runbooks {
		if strings.Contains(strings.ToLower(rb.Title), q) ||
			strings.Contains(strings.ToLower(rb.Description), q) {
			runbooks = append(runbooks, *rb)
			continue
		}
		for _, tag := range rb.Tags {
			if strings.Contains(strings.ToLower(tag), q) {
				runbooks = append(runbooks, *rb)
				break
			}
		}
	}

	return notes, runbooks
}

// ---- Storage ----

func (nm *NotesManager) load() {
	data, err := os.ReadFile(nm.savePath)
	if err != nil {
		return
	}

	var saved Storage
	if err := json.Unmarshal(data, &saved); err != nil {
		return
	}

	nm.storage = saved

	if nm.storage.Notes == nil {
		nm.storage.Notes = make(map[string]*Note)
	}
	if nm.storage.Runbooks == nil {
		nm.storage.Runbooks = make(map[string]*Runbook)
	}
}

func (nm *NotesManager) save() error {
	data, err := json.MarshalIndent(nm.storage, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(nm.savePath), 0700); err != nil {
		return err
	}

	return os.WriteFile(nm.savePath, data, 0600)
}
