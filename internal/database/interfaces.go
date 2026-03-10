package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/kingknull/oblivrashell/internal/cluster"
)

// DatabaseStore defines the interface for the central application database.
type DatabaseStore interface {
	IsLocked() bool
	Conn() (*sql.DB, error)
	Close() error
	DB() *sql.DB
	Open(dbPath string, encryptionKey []byte) error
	Migrate() error
	Lock()
	Unlock()
	RLock()
	RUnlock()

	SetClusterManager(cm cluster.Manager)
	ClusterManager() cluster.Manager
	ReplicatedExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// CredentialStore defines the interface for managing encrypted credentials.
type CredentialStore interface {
	List(ctx context.Context, typeFilter string) ([]Credential, error)
	Create(ctx context.Context, c *Credential) error
	GetByID(ctx context.Context, id string) (*Credential, error)
	Update(ctx context.Context, c *Credential) error
	Delete(ctx context.Context, id string) error
}

// HostStore defines the interface for host management.
type HostStore interface {
	Create(ctx context.Context, host *Host) error
	GetByID(ctx context.Context, id string) (*Host, error)
	GetAll(ctx context.Context) ([]Host, error)
	GetFavorites(ctx context.Context) ([]Host, error)
	Search(ctx context.Context, query string) ([]Host, error)
	GetByTag(ctx context.Context, tag string) ([]Host, error)
	Update(ctx context.Context, host *Host) error
	Delete(ctx context.Context, id string) error
	ToggleFavorite(ctx context.Context, id string) (bool, error)
	RecordConnection(ctx context.Context, id string) error
	GetAllTags(ctx context.Context) ([]string, error)
	Count(ctx context.Context) (int, error)
}

// SessionStore defines the interface for session management and tracking.
type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	End(ctx context.Context, id string, status string, bytesSent, bytesReceived int64) error
	GetRecent(ctx context.Context, limit int) ([]Session, error)
	GetByHostID(ctx context.Context, hostID string, limit int) ([]Session, error)
}

// AuditStore defines the interface for audit logging and archival.
type AuditStore interface {
	Log(ctx context.Context, eventType string, hostID string, sessionID string, details map[string]interface{}) error
	GetRecent(ctx context.Context, limit int) ([]AuditLog, error)
	GetByDateRange(ctx context.Context, from, to time.Time, limit int) ([]AuditLog, error)
	Count(ctx context.Context) (int64, error)
	Export(ctx context.Context, from, to time.Time) ([]byte, error)
	InitIntegrity(ctx context.Context) error
	ValidateIntegrity(ctx context.Context) bool
}

// SIEMStore defines the interface for security event monitoring and analytics.
type SIEMStore interface {
	InsertHostEvent(ctx context.Context, event *HostEvent) error
	GetHostEvents(ctx context.Context, hostID string, limit int) ([]HostEvent, error)
	SearchHostEvents(ctx context.Context, query string, limit int) ([]HostEvent, error)
	GetFailedLoginsByHost(ctx context.Context, hostID string) ([]map[string]interface{}, error)
	CalculateRiskScore(ctx context.Context, hostID string) (int, error)
	GetGlobalThreatStats(ctx context.Context) (map[string]interface{}, error)
	GetEventTrend(ctx context.Context, days int) ([]map[string]interface{}, error)
	AggregateHostEvents(ctx context.Context, query string, facetField string) (map[string]int, error)
	CreateSavedSearch(ctx context.Context, search *SavedSearch) error
	GetSavedSearches(ctx context.Context) ([]SavedSearch, error)
}

// SnippetStore defines the interface for managing command snippets.
type SnippetStore interface {
	List(ctx context.Context) ([]Snippet, error)
	Get(ctx context.Context, id string) (Snippet, error)
	Create(ctx context.Context, s *Snippet) error
	Update(ctx context.Context, s *Snippet) error
	Delete(ctx context.Context, id string) error
	IncrementUseCount(ctx context.Context, id string) error
}

// IncidentStore defines the interface for managing security incidents.
type IncidentStore interface {
	Upsert(ctx context.Context, incident *Incident) error
	GetByID(ctx context.Context, id string) (*Incident, error)
	GetByRuleAndGroup(ctx context.Context, ruleID string, groupKey string) (*Incident, error)
	Search(ctx context.Context, status string, owner string, limit int) ([]Incident, error)
	UpdateStatus(ctx context.Context, id string, status string, reason string) error
}

// ConfigChangeStore defines the interface for tracking configuration changes and risk scores.
type ConfigChangeStore interface {
	RecordChange(ctx context.Context, change *ConfigChange) error
	GetChanges(ctx context.Context, category string, limit int) ([]ConfigChange, error)
}

// EvidenceStore defines the interface for forensic evidence management and chain-of-custody.
type EvidenceStore interface {
	Create(ctx context.Context, item *EvidenceItem) error
	Update(ctx context.Context, item *EvidenceItem) error
	GetByID(ctx context.Context, id string) (*EvidenceItem, error)
	ListByIncident(ctx context.Context, incidentID string) ([]EvidenceItem, error)
	ListAll(ctx context.Context) ([]EvidenceItem, error)
	AddChainEntry(ctx context.Context, entry *ChainEntry) error
	GetChain(ctx context.Context, evidenceID string) ([]ChainEntry, error)
}
