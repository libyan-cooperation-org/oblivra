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
	ConnectionCount int        `json:"connection_count"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
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
