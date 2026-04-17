package database



type Host struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	Label           string     `json:"label"`
	Hostname        string     `json:"hostname"`
	Port            int        `json:"port"`
	Username        string     `json:"username"`
	AuthMethod      string     `json:"auth_method"`
	CredentialID    string     `json:"credential_id,omitempty"`
	JumpHostID      string     `json:"jump_host_id,omitempty"`
	Tags            []string   `json:"tags"`
	Category        string     `json:"category"`
	// Password is intentionally omitted from JSON serialization.
	// It is never sent to the frontend; use HasPassword to check if one is stored.
	Password        string     `json:"-"`
	HasPassword     bool       `json:"has_password"`
	Color           string     `json:"color"`
	Notes           string     `json:"notes"`
	IsFavorite      bool       `json:"is_favorite"`
	LastConnectedAt *string `json:"last_connected_at,omitempty"`
	ConnectionCount   int        `json:"connection_count"`
	CriticalityScore  int        `json:"criticality_score"`
	CriticalityReason string     `json:"criticality_reason"`
	CreatedAt         string     `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
}

// NodeType represents the kind of entity in the graph.
type NodeType string

const (
	NodeUser    NodeType = "user"
	NodeHost    NodeType = "host"
	NodeProcess NodeType = "process"
	NodeFile    NodeType = "file"
	NodeIP      NodeType = "ip"
)

// EdgeType represents the relationship between two nodes.
type EdgeType string

const (
	EdgeAuthenticatedTo EdgeType = "authenticated_to"
	EdgeExecuted        EdgeType = "executed"
	EdgeAccessed        EdgeType = "accessed"
	EdgeConnectedTo     EdgeType = "connected_to"
	EdgeSpawned         EdgeType = "spawned"
)

// Node represents a single entity in the security graph.
type Node struct {
	ID       string            `json:"id"`
	TenantID string            `json:"tenant_id"`
	Type     NodeType          `json:"type"`
	Meta     map[string]string `json:"meta,omitempty"`
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	From string   `json:"from"`
	To   string   `json:"to"`
	Type EdgeType `json:"type"`
	Meta map[string]string `json:"meta,omitempty"`
}

type Credential struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Label         string    `json:"label"`
	Type          string    `json:"type"`
	EncryptedData []byte    `json:"-"`
	Fingerprint   string    `json:"fingerprint"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type Session struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	HostID          string     `json:"host_id"`
	StartedAt       string  `json:"started_at"`
	EndedAt         *string `json:"ended_at,omitempty"`
	DurationSeconds int        `json:"duration_seconds"`
	BytesSent       int64      `json:"bytes_sent"`
	BytesReceived   int64      `json:"bytes_received"`
	Status          string     `json:"status"`
	RecordingPath   string     `json:"recording_path,omitempty"`
}

type AuditLog struct {
	ID          int64     `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Timestamp   string `json:"timestamp"`
	EventType   string    `json:"event_type"`
	HostID      string    `json:"host_id,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
	Details     string    `json:"details"`
	IPAddress   string    `json:"ip_address"`
	MerkleHash  string    `json:"merkle_hash,omitempty"`
	MerkleIndex int       `json:"merkle_index,omitempty"`
}

type Snippet struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Title       string    `json:"title"`
	Command     string    `json:"command"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Variables   []string  `json:"variables"`
	UseCount    int       `json:"use_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Tunnel struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	HostID      string    `json:"host_id"`
	Type        string    `json:"type"`
	LocalPort   int       `json:"local_port"`
	RemoteHost  string    `json:"remote_host"`
	RemotePort  int       `json:"remote_port"`
	AutoConnect bool      `json:"auto_connect"`
	CreatedAt   string `json:"created_at"`
}

type HostEvent struct {
	ID        int64     `json:"id"`
	TenantID  string    `json:"tenant_id"`
	HostID    string    `json:"host_id"`
	Timestamp string `json:"timestamp"`
	EventType string    `json:"event_type"` // e.g., "failed_login"
	SourceIP  string    `json:"source_ip"`
	Location  string    `json:"location"` // Enriched Geographic/DNS data
	User      string    `json:"user"`
	RawLog    string    `json:"raw_log"`
	EventHash string    `json:"event_hash"` // 1.3: SHA256(RawLog + Timestamp + PrevHash)
	PrevHash  string    `json:"prev_hash"`  // 1.3: Reference to previous event's hash
}


// Phase 20.10 — Report Factory Models

type ReportTemplate struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	SectionsJSON string                 `json:"sections_json"` // Serialized []GenericReportSection
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
	Sections     []GenericReportSection `json:"sections,omitempty"` // For DTO
}

type ReportSchedule struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	TemplateID   string `json:"template_id"`
	Name         string `json:"name"`
	IntervalMins int    `json:"interval_mins"`
	NextRunAt    string `json:"next_run_at"`
	Recipients   string `json:"recipients_json"`
	IsActive     bool   `json:"is_active"`
	LastRunAt    string `json:"last_run_at"`
	CreatedAt    string `json:"created_at"`
}

type GeneratedReport struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	ScheduleID  string `json:"schedule_id,omitempty"`
	TemplateID  string `json:"template_id"`
	Title       string `json:"title"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	FilePath    string `json:"file_path"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type GenericReportSection struct {
	Title string `json:"title"`
	Type  string `json:"type"` // "table", "summary", "trend"
	Query string `json:"query"` // OQL Query
}

type SavedSearch struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
	CreatedAt string `json:"created_at"`
}

type Incident struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	RuleID           string    `json:"rule_id"`
	GroupKey         string    `json:"group_key"` // e.g., IP address, user, host
	Status           string    `json:"status"`    // New, Active, Investigating, Closed
	Severity         string    `json:"severity"`
	Description      string    `json:"description"`
	Title            string    `json:"title"`
	FirstSeenAt      string `json:"first_seen_at"`
	LastSeenAt       string `json:"last_seen_at"`
	EventCount       int       `json:"event_count"`
	Owner            string    `json:"owner,omitempty"`
	MitreTactics     []string  `json:"mitre_tactics"`
	MitreTechniques  []string  `json:"mitre_techniques"`
	ResolutionReason string    `json:"resolution_reason,omitempty"`
	TriageScore      int       `json:"triage_score"`
	TriageReason     string    `json:"triage_reason"`
}

type ConfigChange struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Timestamp string `json:"timestamp"`
	Category  string    `json:"category"` // "settings", "rules", "sources"
	Key       string    `json:"key"`
	OldValue  string    `json:"old_value,omitempty"`
	NewValue  string    `json:"new_value,omitempty"`
	RiskScore int       `json:"risk_score"`
	Status    string    `json:"status"` // e.g., "applied"
}

// Phase 20.11 — Dashboard Studio Models

type Dashboard struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Layout      string            `json:"layout"` // "grid", "freeflow"
	OwnerID     string            `json:"owner_id"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	Widgets     []DashboardWidget `json:"widgets,omitempty"`
}

type DashboardWidget struct {
	ID                  string `json:"id"`
	DashboardID         string `json:"dashboard_id"`
	Title               string `json:"title"`
	VizType             string `json:"viz_type"` // "bar", "line", "summary", "table", "pie"
	QueryOQL            string `json:"query_oql"`
	LayoutX             int    `json:"layout_x"`
	LayoutY             int    `json:"layout_y"`
	LayoutW             int    `json:"layout_w"`
	LayoutH             int    `json:"layout_h"`
	RefreshIntervalSecs int    `json:"refresh_interval_secs"`
	CreatedAt           string `json:"created_at"`
}

type EvidenceItem struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	IncidentID  string            `json:"incident_id"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	SHA256      string            `json:"sha256"`
	Size        int64             `json:"size"`
	Collector   string            `json:"collector"`
	CollectedAt string         `json:"collected_at"`
	Sealed      bool              `json:"sealed"`
	SealedAt    *string        `json:"sealed_at,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ChainEntry struct {
	ID           int64     `json:"id"`
	TenantID     string    `json:"tenant_id"`
	EvidenceID   string    `json:"evidence_id"`
	Action       string    `json:"action"`
	Actor        string    `json:"actor"`
	Timestamp    string `json:"timestamp"`
	Notes        string    `json:"notes,omitempty"`
	PreviousHash string    `json:"previous_hash"`
	EntryHash    string    `json:"entry_hash"`
}

type CloudAsset struct {
	ID        string            `json:"id"`
	TenantID  string            `json:"tenant_id"`
	Provider  string            `json:"provider"` // aws, azure, gcp
	Region    string            `json:"region"`
	AccountID string            `json:"account_id"`
	Type      string            `json:"type"` // ec2, s3, lambda, etc.
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata"`
	Tags      map[string]string `json:"tags"`
	FirstSeen string            `json:"first_seen"`
	LastSeen  string            `json:"last_seen"`
}

// IdentityConnector represents an external identity provider configuration.
type IdentityConnector struct {
	ID               string `json:"id"`
	TenantID         string `json:"tenant_id"`
	Name             string `json:"name"`
	Type             string `json:"type"` // okta, azure_ad, ldap
	Enabled          bool   `json:"enabled"`
	ConfigJSON       string `json:"config_json"` // AES-encrypted configuration blob
	SyncIntervalMins int    `json:"sync_interval_mins"`
	LastSync         string `json:"last_sync"`
	Status           string `json:"status"`
	ErrorMessage     string `json:"error_message"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}
