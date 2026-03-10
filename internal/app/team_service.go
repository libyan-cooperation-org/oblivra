package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/team"
)

// TeamService exposes team vault operations to the frontend
type TeamService struct {
	BaseService
	ctx       context.Context
	teamVault *team.TeamVault
	bus       *eventbus.Bus
	log       *logger.Logger
	actorID   string // Current user ID for RBAC
}

func (s *TeamService) Name() string { return "TeamService" }

func NewTeamService(tv *team.TeamVault, bus *eventbus.Bus, log *logger.Logger) *TeamService {
	return &TeamService{
		teamVault: tv,
		bus:       bus,
		log:       log.WithPrefix("team_service"),
		actorID:   "local-user", // In reality, fetched from auth session
	}
}

func (s *TeamService) Startup(ctx context.Context) {
	s.ctx = ctx
}

func (s *TeamService) SetActorID(id string) {
	s.actorID = id
}

// GetTeamName gets the vault name
func (s *TeamService) GetTeamName() string {
	if s.teamVault == nil {
		return "Personal"
	}
	return s.teamVault.TeamName
}

// AddMember adds a user to the team vault
func (s *TeamService) AddMember(email string, name string, roleStr string) (*team.TeamMember, error) {
	role := team.Role(roleStr)
	member, err := s.teamVault.AddMember(s.actorID, email, name, role)
	if err == nil {
		s.log.Info("Added team member %s (%s) with role %s", name, email, roleStr)
		s.teamVault.LogActivity(s.actorID, "member_added", fmt.Sprintf("Added %s as %s", name, roleStr))
		s.bus.Publish("team.member_added", member)
	}
	return member, err
}

// ListMembers gets team users
func (s *TeamService) ListMembers() ([]team.TeamMember, error) {
	return s.teamVault.ListMembers(s.actorID)
}

// AddSecret saves an encrypted value
func (s *TeamService) AddSecret(title string, entryType string, secretData string) (*team.VaultEntry, error) {
	s.log.Info("Adding team secret: %s", title)
	entry, err := s.teamVault.AddEntry(s.actorID, title, entryType, []byte(secretData))
	if err == nil {
		s.teamVault.LogActivity(s.actorID, "secret_added", fmt.Sprintf("Created shared secret: %s", title))
		s.bus.Publish("team.secret_added", entry.ID)
	}
	return entry, err
}

// GetSecret reads a shared value
func (s *TeamService) GetSecret(entryID string) (string, error) {
	s.log.Info("Accessing team secret: %s", entryID)
	data, err := s.teamVault.GetEntry(s.actorID, entryID)
	if err != nil {
		return "", err
	}
	s.teamVault.LogActivity(s.actorID, "secret_accessed", fmt.Sprintf("Accessed secret: %s", entryID))

	// Try to format as JSON if it's a structured credential
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err == nil {
		formatted, _ := json.MarshalIndent(parsed, "", "  ")
		return string(formatted), nil
	}

	return string(data), nil
}

// ListSecrets returns metadata for all accessible items
func (s *TeamService) ListSecrets() ([]team.VaultEntry, error) {
	return s.teamVault.ListEntries(s.actorID)
}

// ListActivity returns recent team events
func (s *TeamService) ListActivity() ([]team.TeamActivity, error) {
	return s.teamVault.GetActivities(), nil
}
