package team

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Role defines the access level for a team member
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// Permission defines specific actions
type Permission string

const (
	PermManageUsers   Permission = "manage_users"
	PermManageRoles   Permission = "manage_roles"
	PermManageEntries Permission = "manage_entries"
	PermViewEntries   Permission = "view_entries"
	PermShareEntries  Permission = "share_entries"
)

// TeamMember represents a user in the team
type TeamMember struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	JoinedAt  string    `json:"joined_at"`
	IsActive  bool      `json:"is_active"`
	PublicKey string    `json:"public_key,omitempty"` // For E2E encrypted sharing
}

// VaultEntry represents a shared secret or credential
type VaultEntry struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	EntryType     string        `json:"entry_type"`     // "password", "ssh_key", "api_token"
	EncryptedData string        `json:"encrypted_data"` // Encrypted with the team/folder key
	Tags          []string      `json:"tags"`
	CreatedAt     string        `json:"created_at"`
	UpdatedAt     string        `json:"updated_at"`
	CreatedBy     string        `json:"created_by"`
	AccessLog     []AccessEvent `json:"access_log"`
}

// AccessEvent records who accessed what and when
type AccessEvent struct {
	Timestamp string    `json:"timestamp"`
	MemberID  string    `json:"member_id"`
	Action    string    `json:"action"` // "view", "edit", "share"
	IPAddress string    `json:"ip_address,omitempty"`
}

// TeamActivity represents a high-level team event
type TeamActivity struct {
	ID        string    `json:"id"`
	Timestamp string    `json:"timestamp"`
	ActorID   string    `json:"actor_id"`
	ActorName string    `json:"actor_name"`
	Action    string    `json:"action"` // "member_added", "secret_added", "secret_accessed"
	Details   string    `json:"details"`
}

// TeamVault manages shared credentials with RBAC
type TeamVault struct {
	mu         sync.RWMutex
	TeamID     string
	TeamName   string
	Members    map[string]*TeamMember
	Entries    map[string]*VaultEntry
	Activities []TeamActivity
	rolePerms  map[Role][]Permission
	masterKey  []byte // The team's symmetric key
}

// NewTeamVault initializes a new team vault
func NewTeamVault(teamName string) *TeamVault {
	key := make([]byte, 32)
	rand.Read(key)

	tv := &TeamVault{
		TeamID:     uuid.New().String(),
		TeamName:   teamName,
		Members:    make(map[string]*TeamMember),
		Entries:    make(map[string]*VaultEntry),
		Activities: make([]TeamActivity, 0),
		rolePerms:  make(map[Role][]Permission),
		masterKey:  key,
	}

	// Initialize default RBAC matrix
	tv.rolePerms[RoleOwner] = []Permission{PermManageUsers, PermManageRoles, PermManageEntries, PermViewEntries, PermShareEntries}
	tv.rolePerms[RoleAdmin] = []Permission{PermManageUsers, PermManageEntries, PermViewEntries, PermShareEntries}
	tv.rolePerms[RoleMember] = []Permission{PermManageEntries, PermViewEntries, PermShareEntries}
	tv.rolePerms[RoleViewer] = []Permission{PermViewEntries}

	return tv
}

// HasPermission checks if a role has a specific permission
func (tv *TeamVault) HasPermission(role Role, perm Permission) bool {
	tv.mu.RLock()
	defer tv.mu.RUnlock()

	perms, ok := tv.rolePerms[role]
	if !ok {
		return false
	}

	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// LogActivity records a team event
func (tv *TeamVault) LogActivity(actorID string, action string, details string) {
	actorName := "Unknown"
	if actor, ok := tv.Members[actorID]; ok {
		actorName = actor.Name
	}

	activity := TeamActivity{
		ID:        uuid.New().String(),
		Timestamp: time.Now().Format(time.RFC3339),
		ActorID:   actorID,
		ActorName: actorName,
		Action:    action,
		Details:   details,
	}

	tv.Activities = append(tv.Activities, activity)
	// Keep last 100 activities
	if len(tv.Activities) > 100 {
		tv.Activities = tv.Activities[1:]
	}
}

// GetActivities returns a copy of the activity log
func (tv *TeamVault) GetActivities() []TeamActivity {
	tv.mu.RLock()
	defer tv.mu.RUnlock()

	activities := make([]TeamActivity, len(tv.Activities))
	copy(activities, tv.Activities)
	return activities
}

// AddMember adds a new member to the team
func (tv *TeamVault) AddMember(actorID string, email, name string, role Role) (*TeamMember, error) {
	tv.mu.Lock()
	defer tv.mu.Unlock()

	// RBAC Check
	actor, ok := tv.Members[actorID]
	// Allow if vault is empty (first user) or if actor has permission
	if len(tv.Members) > 0 {
		if !ok || !tv.hasPermissionUnsafe(actor.Role, PermManageUsers) {
			return nil, fmt.Errorf("permission denied")
		}
	} else {
		// First user is automatically Owner
		role = RoleOwner
	}

	member := &TeamMember{
		ID:       uuid.New().String(),
		Email:    email,
		Name:     name,
		Role:     role,
		JoinedAt: time.Now().Format(time.RFC3339),
		IsActive: true,
	}

	tv.Members[member.ID] = member
	return member, nil
}

// AddEntry creates a new shared entry
func (tv *TeamVault) AddEntry(actorID string, title, entryType string, rawData []byte) (*VaultEntry, error) {
	tv.mu.Lock()
	defer tv.mu.Unlock()

	// RBAC Check
	actor, ok := tv.Members[actorID]
	if !ok || !tv.hasPermissionUnsafe(actor.Role, PermManageEntries) {
		return nil, fmt.Errorf("permission denied")
	}

	// In a real app, encrypt rawData with tv.masterKey using AES-GCM here
	encryptedData := base64.StdEncoding.EncodeToString(rawData)

	entry := &VaultEntry{
		ID:            uuid.New().String(),
		Title:         title,
		EntryType:     entryType,
		EncryptedData: encryptedData,
		CreatedAt:     time.Now().Format(time.RFC3339),
		UpdatedAt:     time.Now().Format(time.RFC3339),
		CreatedBy:     actorID,
		AccessLog:     make([]AccessEvent, 0),
	}

	tv.Entries[entry.ID] = entry

	return entry, nil
}

// GetEntry retrieves and logs access to an entry
func (tv *TeamVault) GetEntry(actorID string, entryID string) ([]byte, error) {
	tv.mu.Lock()
	defer tv.mu.Unlock()

	actor, ok := tv.Members[actorID]
	if !ok || !tv.hasPermissionUnsafe(actor.Role, PermViewEntries) {
		return nil, fmt.Errorf("permission denied")
	}

	entry, ok := tv.Entries[entryID]
	if !ok {
		return nil, fmt.Errorf("entry not found")
	}

	// Log access
	entry.AccessLog = append(entry.AccessLog, AccessEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		MemberID:  actorID,
		Action:    "view",
	})

	// Decrypt
	rawData, err := base64.StdEncoding.DecodeString(entry.EncryptedData)
	if err != nil {
		return nil, err
	}

	return rawData, nil
}

// ListMembers returns all active team members
func (tv *TeamVault) ListMembers(actorID string) ([]TeamMember, error) {
	tv.mu.RLock()
	defer tv.mu.RUnlock()

	_, ok := tv.Members[actorID]
	if !ok {
		return nil, fmt.Errorf("not a member")
	}

	var members []TeamMember
	for _, m := range tv.Members {
		if m.IsActive {
			members = append(members, *m)
		}
	}
	return members, nil
}

// ListEntries returns metadata for all entries
func (tv *TeamVault) ListEntries(actorID string) ([]VaultEntry, error) {
	tv.mu.RLock()
	defer tv.mu.RUnlock()

	actor, ok := tv.Members[actorID]
	if !ok || !tv.hasPermissionUnsafe(actor.Role, PermViewEntries) {
		return nil, fmt.Errorf("permission denied")
	}

	var entries []VaultEntry
	for _, e := range tv.Entries {
		// Make a copy without the sensitive data
		copyEntry := *e
		copyEntry.EncryptedData = ""
		copyEntry.AccessLog = nil
		entries = append(entries, copyEntry)
	}

	return entries, nil
}

// Internal helper must be called with lock held
func (tv *TeamVault) hasPermissionUnsafe(role Role, perm Permission) bool {
	perms, ok := tv.rolePerms[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}
