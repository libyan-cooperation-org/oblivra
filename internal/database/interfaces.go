package database

import (
	"context"
	"database/sql"


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
	// GetEncryptedPassword returns the raw stored password blob for SSH connect-time decryption.
	// Must never be used to populate the Host DTO sent to the frontend.
	GetEncryptedPassword(ctx context.Context, id string) (string, error)
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
	GetByDateRange(ctx context.Context, from, to string, limit int) ([]AuditLog, error)
	Count(ctx context.Context) (int64, error)
	Export(ctx context.Context, from, to string) ([]byte, error)
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

// CloudAssetStore defines the interface for managing cloud resource inventory.
type CloudAssetStore interface {
	Upsert(ctx context.Context, asset *CloudAsset) error
	GetByID(ctx context.Context, id string) (*CloudAsset, error)
	List(ctx context.Context, provider string, accountID string) ([]CloudAsset, error)
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context) (map[string]int, error)
}

// IdentityConnectorStore defines the interface for managing external identity sources.
type IdentityConnectorStore interface {
	List(ctx context.Context) ([]IdentityConnector, error)
	Create(ctx context.Context, c *IdentityConnector) error
	GetByID(ctx context.Context, id string) (*IdentityConnector, error)
	Update(ctx context.Context, c *IdentityConnector) error
	Delete(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error
	MarkSyncStart(ctx context.Context, id string) error
}

// UserStore defines the interface for user and identity management.
type UserStore interface {
	CreateUser(ctx context.Context, u *User) error
	UpdateUser(ctx context.Context, u *User) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	ListUsers(ctx context.Context) ([]User, error)
	DeleteUser(ctx context.Context, id string) error
}

// ReportStore defines the interface for managing reports and schedules.
type ReportStore interface {
	CreateTemplate(ctx context.Context, t *ReportTemplate) error
	GetTemplate(ctx context.Context, id string) (*ReportTemplate, error)
	ListTemplates(ctx context.Context) ([]ReportTemplate, error)
	
	CreateSchedule(ctx context.Context, s *ReportSchedule) error
	GetDueSchedules(ctx context.Context) ([]ReportSchedule, error)
	MarkScheduleRun(ctx context.Context, id string) error
	
	CreateReportInstance(ctx context.Context, g *GeneratedReport) error
	ListReports(ctx context.Context, limit int) ([]GeneratedReport, error)
}

// DashboardStore defines the interface for native Dashboard Studio management.
type DashboardStore interface {
	Create(ctx context.Context, d *Dashboard) error
	GetByID(ctx context.Context, id string) (*Dashboard, error)
	List(ctx context.Context) ([]Dashboard, error)
	Update(ctx context.Context, d *Dashboard) error
	Delete(ctx context.Context, id string) error

	AddWidget(ctx context.Context, w *DashboardWidget) error
	GetWidgets(ctx context.Context, dashboardID string) ([]DashboardWidget, error)
	UpdateWidget(ctx context.Context, w *DashboardWidget) error
	DeleteWidget(ctx context.Context, dashboardID, widgetID string) error
}

// AssetIntelProvider defines logic for criticality scoring and asset intelligence.
type AssetIntelProvider interface {
	CalculateHostCriticality(ctx context.Context, host *Host) (int, string)
	CalculateUserCriticality(ctx context.Context, user *User) (int, string)
	RefreshAll(ctx context.Context) error
}

// GraphStore defines the persistence layer for the Security Graph Engine.
type GraphStore interface {
	UpsertNode(ctx context.Context, node graph.Node) error
	UpsertEdge(ctx context.Context, edge graph.Edge) error
	GetSubGraph(ctx context.Context, startNodeID string, depth int) ([]graph.Node, []graph.Edge, error)
	FindPath(ctx context.Context, startID, endID string) ([]string, error)
	DeleteNode(ctx context.Context, id string) error
}
