package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/vault"
)

// BookmarkService is a Wails-bound service for SSH host bookmark management.
// Wraps HostStore with Vault-encrypted credential handling.
// This gives the frontend a Termius-like host management experience.
type BookmarkService struct {
	BaseService
	hosts database.HostStore
	creds database.CredentialStore
	vault vault.Provider
	bus   *eventbus.Bus
	log   *logger.Logger
}

// BookmarkEntry is the frontend-friendly view of an SSH host bookmark.
type BookmarkEntry struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Hostname        string   `json:"hostname"`
	Port            int      `json:"port"`
	Username        string   `json:"username"`
	AuthMethod      string   `json:"auth_method"`
	CredentialID    string   `json:"credential_id,omitempty"`
	JumpHostID      string   `json:"jump_host_id,omitempty"`
	Tags            []string `json:"tags"`
	Category        string   `json:"category"`
	Color           string   `json:"color"`
	Notes           string   `json:"notes"`
	IsFavorite      bool     `json:"is_favorite"`
	HasPassword     bool     `json:"has_password"`
	LastConnectedAt *string  `json:"last_connected_at,omitempty"`
	ConnectionCount int      `json:"connection_count"`
}

// BookmarkCreateInput is the payload for creating a new bookmark.
type BookmarkCreateInput struct {
	Label      string   `json:"label"`
	Hostname   string   `json:"hostname"`
	Port       int      `json:"port"`
	Username   string   `json:"username"`
	AuthMethod string   `json:"auth_method"`
	Password   string   `json:"password,omitempty"`
	Tags       []string `json:"tags"`
	Category   string   `json:"category"`
	Color      string   `json:"color"`
	Notes      string   `json:"notes"`
	JumpHostID string   `json:"jump_host_id,omitempty"`
}

func (s *BookmarkService) Name() string        { return "bookmark-service" }
func (s *BookmarkService) Dependencies() []string { return []string{"vault"} }
func (s *BookmarkService) Start(ctx context.Context) error { return nil }
func (s *BookmarkService) Stop(ctx context.Context) error  { return nil }

// NewBookmarkService creates the bookmark management service.
func NewBookmarkService(
	hosts database.HostStore,
	creds database.CredentialStore,
	vlt vault.Provider,
	bus *eventbus.Bus,
	log *logger.Logger,
) *BookmarkService {
	return &BookmarkService{
		hosts: hosts,
		creds: creds,
		vault: vlt,
		bus:   bus,
		log:   log.WithPrefix("bookmarks"),
	}
}

// ListAll returns all SSH host bookmarks.
func (s *BookmarkService) ListAll(ctx context.Context) ([]BookmarkEntry, error) {
	hosts, err := s.hosts.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list hosts: %w", err)
	}
	return s.toEntries(hosts), nil
}

// GetFavorites returns only bookmarks marked as favorites.
func (s *BookmarkService) GetFavorites(ctx context.Context) ([]BookmarkEntry, error) {
	hosts, err := s.hosts.GetFavorites(ctx)
	if err != nil {
		return nil, fmt.Errorf("get favorites: %w", err)
	}
	return s.toEntries(hosts), nil
}

// Search finds bookmarks matching a query (hostname, label, tags).
func (s *BookmarkService) Search(ctx context.Context, query string) ([]BookmarkEntry, error) {
	hosts, err := s.hosts.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	return s.toEntries(hosts), nil
}

// GetByTag returns bookmarks with a specific tag.
func (s *BookmarkService) GetByTag(ctx context.Context, tag string) ([]BookmarkEntry, error) {
	hosts, err := s.hosts.GetByTag(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("get by tag: %w", err)
	}
	return s.toEntries(hosts), nil
}

// GetAllTags returns all unique tags used across bookmarks.
func (s *BookmarkService) GetAllTags(ctx context.Context) ([]string, error) {
	return s.hosts.GetAllTags(ctx)
}

// Create adds a new SSH host bookmark. Password is encrypted via Vault.
func (s *BookmarkService) Create(ctx context.Context, input BookmarkCreateInput) (*BookmarkEntry, error) {
	if input.Hostname == "" {
		return nil, fmt.Errorf("hostname is required")
	}
	if input.Label == "" {
		input.Label = input.Hostname
	}
	if input.Port == 0 {
		input.Port = 22
	}
	if input.Username == "" {
		input.Username = "root"
	}
	if input.AuthMethod == "" {
		input.AuthMethod = "password"
	}

	host := &database.Host{
		ID:         uuid.New().String(),
		TenantID:   database.MustTenantFromContext(ctx),
		Label:      input.Label,
		Hostname:   input.Hostname,
		Port:       input.Port,
		Username:   input.Username,
		AuthMethod: input.AuthMethod,
		Tags:       input.Tags,
		Category:   input.Category,
		Color:      input.Color,
		Notes:      input.Notes,
		JumpHostID: input.JumpHostID,
	}

	// Encrypt and store password via Vault if provided
	if input.Password != "" && s.vault != nil && s.vault.IsUnlocked() {
		encrypted, err := s.vault.Encrypt([]byte(input.Password))
		if err != nil {
			return nil, fmt.Errorf("encrypt password: %w", err)
		}
		host.Password = string(encrypted)
		host.HasPassword = true
	}

	if err := s.hosts.Create(ctx, host); err != nil {
		return nil, fmt.Errorf("create host: %w", err)
	}

	s.log.Info("[BOOKMARKS] Created: %s (%s@%s:%d)", host.Label, host.Username, host.Hostname, host.Port)
	s.bus.Publish("bookmark:created", map[string]string{
		"id": host.ID, "label": host.Label, "hostname": host.Hostname,
	})

	entry := s.toEntry(*host)
	return &entry, nil
}

// Update modifies an existing bookmark.
func (s *BookmarkService) Update(ctx context.Context, id string, input BookmarkCreateInput) (*BookmarkEntry, error) {
	host, err := s.hosts.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("host not found: %w", err)
	}

	if input.Label != "" {
		host.Label = input.Label
	}
	if input.Hostname != "" {
		host.Hostname = input.Hostname
	}
	if input.Port > 0 {
		host.Port = input.Port
	}
	if input.Username != "" {
		host.Username = input.Username
	}
	if input.AuthMethod != "" {
		host.AuthMethod = input.AuthMethod
	}
	if input.Tags != nil {
		host.Tags = input.Tags
	}
	if input.Category != "" {
		host.Category = input.Category
	}
	host.Color = input.Color
	host.Notes = input.Notes
	host.JumpHostID = input.JumpHostID

	// Update password if provided
	if input.Password != "" && s.vault != nil && s.vault.IsUnlocked() {
		encrypted, err := s.vault.Encrypt([]byte(input.Password))
		if err != nil {
			return nil, fmt.Errorf("encrypt password: %w", err)
		}
		host.Password = string(encrypted)
		host.HasPassword = true
	}

	if err := s.hosts.Update(ctx, host); err != nil {
		return nil, fmt.Errorf("update host: %w", err)
	}

	s.log.Info("[BOOKMARKS] Updated: %s (%s)", host.Label, host.ID)
	entry := s.toEntry(*host)
	return &entry, nil
}

// Delete removes a bookmark permanently.
func (s *BookmarkService) Delete(ctx context.Context, id string) error {
	if err := s.hosts.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete host: %w", err)
	}
	s.log.Info("[BOOKMARKS] Deleted: %s", id)
	s.bus.Publish("bookmark:deleted", map[string]string{"id": id})
	return nil
}

// ToggleFavorite toggles the favorite status of a bookmark.
func (s *BookmarkService) ToggleFavorite(ctx context.Context, id string) (bool, error) {
	return s.hosts.ToggleFavorite(ctx, id)
}

// GetCount returns the total number of bookmarks.
func (s *BookmarkService) GetCount(ctx context.Context) (int, error) {
	return s.hosts.Count(ctx)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func (s *BookmarkService) toEntries(hosts []database.Host) []BookmarkEntry {
	entries := make([]BookmarkEntry, 0, len(hosts))
	for _, h := range hosts {
		entries = append(entries, s.toEntry(h))
	}
	return entries
}

func (s *BookmarkService) toEntry(h database.Host) BookmarkEntry {
	return BookmarkEntry{
		ID:              h.ID,
		Label:           h.Label,
		Hostname:        h.Hostname,
		Port:            h.Port,
		Username:        h.Username,
		AuthMethod:      h.AuthMethod,
		CredentialID:    h.CredentialID,
		JumpHostID:      h.JumpHostID,
		Tags:            h.Tags,
		Category:        h.Category,
		Color:           h.Color,
		Notes:           h.Notes,
		IsFavorite:      h.IsFavorite,
		HasPassword:     h.HasPassword,
		LastConnectedAt: h.LastConnectedAt,
		ConnectionCount: h.ConnectionCount,
	}
}

// QuickSearch performs a fuzzy search across labels, hostnames, and tags.
func (s *BookmarkService) QuickSearch(ctx context.Context, query string) ([]BookmarkEntry, error) {
	if strings.TrimSpace(query) == "" {
		return s.ListAll(ctx)
	}
	return s.Search(ctx, query)
}
