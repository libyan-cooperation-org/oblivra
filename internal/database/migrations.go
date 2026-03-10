package database

import "fmt"

type migration struct {
	version int
	name    string
	sql     string
}

var migrations = []migration{
	{
		version: 1,
		name:    "initial_schema",
		sql: `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE IF NOT EXISTS hosts (
				id TEXT PRIMARY KEY,
				label TEXT NOT NULL,
				hostname TEXT NOT NULL,
				port INTEGER DEFAULT 22,
				username TEXT DEFAULT '',
				auth_method TEXT DEFAULT 'key',
				credential_id TEXT,
				jump_host_id TEXT,
				tags TEXT DEFAULT '[]',
				color TEXT DEFAULT '',
				notes TEXT DEFAULT '',
				is_favorite BOOLEAN DEFAULT 0,
				last_connected_at DATETIME,
				connection_count INTEGER DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (jump_host_id) REFERENCES hosts(id) ON DELETE SET NULL
			);

			CREATE TABLE IF NOT EXISTS credentials (
				id TEXT PRIMARY KEY,
				label TEXT NOT NULL,
				type TEXT NOT NULL,
				encrypted_data BLOB NOT NULL,
				fingerprint TEXT DEFAULT '',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE IF NOT EXISTS sessions (
				id TEXT PRIMARY KEY,
				host_id TEXT NOT NULL,
				started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				ended_at DATETIME,
				duration_seconds INTEGER DEFAULT 0,
				bytes_sent INTEGER DEFAULT 0,
				bytes_received INTEGER DEFAULT 0,
				status TEXT DEFAULT 'active',
				recording_path TEXT DEFAULT '',
				FOREIGN KEY (host_id) REFERENCES hosts(id) ON DELETE CASCADE
			);

			CREATE TABLE IF NOT EXISTS audit_logs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
				event_type TEXT NOT NULL,
				host_id TEXT,
				session_id TEXT,
				details TEXT DEFAULT '{}',
				ip_address TEXT DEFAULT '',
				user_agent TEXT DEFAULT ''
			);

			CREATE TABLE IF NOT EXISTS settings (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE IF NOT EXISTS snippets (
				id TEXT PRIMARY KEY,
				title TEXT NOT NULL,
				command TEXT NOT NULL,
				description TEXT DEFAULT '',
				tags TEXT DEFAULT '[]',
				variables TEXT DEFAULT '[]',
				use_count INTEGER DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE IF NOT EXISTS tunnels (
				id TEXT PRIMARY KEY,
				host_id TEXT NOT NULL,
				type TEXT NOT NULL,
				local_port INTEGER NOT NULL,
				remote_host TEXT NOT NULL,
				remote_port INTEGER NOT NULL,
				auto_connect BOOLEAN DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (host_id) REFERENCES hosts(id) ON DELETE CASCADE
			);

			CREATE INDEX IF NOT EXISTS idx_hosts_label ON hosts(label);
			CREATE INDEX IF NOT EXISTS idx_hosts_hostname ON hosts(hostname);
			CREATE INDEX IF NOT EXISTS idx_hosts_favorite ON hosts(is_favorite);
			CREATE INDEX IF NOT EXISTS idx_sessions_host ON sessions(host_id);
			CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
			CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
			CREATE INDEX IF NOT EXISTS idx_audit_event ON audit_logs(event_type);
			CREATE INDEX IF NOT EXISTS idx_snippets_title ON snippets(title);
		`,
	},
	{
		version: 2,
		name:    "add_host_category",
		sql: `
			ALTER TABLE hosts ADD COLUMN category TEXT DEFAULT '';
		`,
	},
	{
		version: 3,
		name:    "add_host_password",
		sql: `
			ALTER TABLE hosts ADD COLUMN password TEXT DEFAULT '';
		`,
	},
	{
		version: 4,
		name:    "create_host_events",
		sql: `
			CREATE TABLE IF NOT EXISTS host_events (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				host_id TEXT NOT NULL,
				timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
				event_type TEXT NOT NULL,
				source_ip TEXT DEFAULT '',
				user TEXT DEFAULT '',
				raw_log TEXT DEFAULT '',
				FOREIGN KEY (host_id) REFERENCES hosts(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_host_events_host_id ON host_events(host_id);
		`,
	},
	{
		version: 5,
		name:    "create_workspaces",
		sql: `
			CREATE TABLE IF NOT EXISTS workspaces (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				description TEXT,
				layout_json TEXT NOT NULL,
				connections_json TEXT NOT NULL,
				sidebar_open BOOLEAN DEFAULT 1,
				sidebar_width INTEGER DEFAULT 260,
				active_tab TEXT,
				is_default BOOLEAN DEFAULT 0,
				icon TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_workspaces_default ON workspaces(is_default);
		`,
	},
	{
		version: 6,
		name:    "create_saved_searches",
		sql: `
			CREATE TABLE IF NOT EXISTS saved_searches (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				query TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
		`,
	},
	{
		version: 7,
		name:    "create_incidents",
		sql: `
			CREATE TABLE IF NOT EXISTS incidents (
				id TEXT PRIMARY KEY,
				rule_id TEXT NOT NULL,
				group_key TEXT NOT NULL,
				status TEXT DEFAULT 'New',
				severity TEXT NOT NULL,
				description TEXT DEFAULT '',
				title TEXT NOT NULL,
				first_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				event_count INTEGER DEFAULT 1,
				owner TEXT DEFAULT '',
				mitre_tactics TEXT DEFAULT '[]',
				mitre_techniques TEXT DEFAULT '[]',
				resolution_reason TEXT DEFAULT ''
			);
			CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
			CREATE INDEX IF NOT EXISTS idx_incidents_rule_group ON incidents(rule_id, group_key);
		`,
	},
	{
		version: 8,
		name:    "create_config_changes",
		sql: `
			CREATE TABLE IF NOT EXISTS config_changes (
				id TEXT PRIMARY KEY,
				timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
				category TEXT NOT NULL,
				key TEXT NOT NULL,
				old_value TEXT,
				new_value TEXT,
				risk_score INTEGER DEFAULT 0,
				status TEXT DEFAULT 'applied'
			);
			CREATE INDEX IF NOT EXISTS idx_config_changes_category ON config_changes(category);
			CREATE INDEX IF NOT EXISTS idx_config_changes_timestamp ON config_changes(timestamp);
		`,
	},
	{
		version: 9,
		name:    "create_forensic_evidence",
		sql: `
			CREATE TABLE IF NOT EXISTS evidence (
				id TEXT PRIMARY KEY,
				incident_id TEXT,
				type TEXT NOT NULL,
				name TEXT NOT NULL,
				description TEXT,
				sha256 TEXT NOT NULL,
				size INTEGER NOT NULL,
				collector TEXT NOT NULL,
				collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				sealed BOOLEAN DEFAULT 0,
				sealed_at DATETIME,
				tags TEXT,
				metadata TEXT
			);
			CREATE TABLE IF NOT EXISTS evidence_chain (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				evidence_id TEXT NOT NULL,
				action TEXT NOT NULL,
				actor TEXT NOT NULL,
				timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
				notes TEXT,
				previous_hash TEXT,
				entry_hash TEXT NOT NULL,
				FOREIGN KEY(evidence_id) REFERENCES evidence(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_evidence_incident ON evidence(incident_id);
			CREATE INDEX IF NOT EXISTS idx_evidence_chain_evidence ON evidence_chain(evidence_id);
		`,
	},
	{
		version: 10,
		name:    "add_audit_merkle_columns",
		sql: `
			ALTER TABLE audit_logs ADD COLUMN merkle_hash TEXT;
			ALTER TABLE audit_logs ADD COLUMN merkle_index INTEGER;
			CREATE INDEX IF NOT EXISTS idx_audit_logs_merkle_index ON audit_logs(merkle_index);
		`,
	},
	{
		version: 11,
		name:    "multi_tenancy",
		sql: `
			CREATE TABLE IF NOT EXISTS tenants (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			INSERT OR IGNORE INTO tenants (id, name) VALUES ('default_tenant', 'Default Organization');

			ALTER TABLE hosts ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE credentials ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE sessions ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE snippets ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE tunnels ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE host_events ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE saved_searches ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE incidents ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE config_changes ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE evidence ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE evidence_chain ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';
			ALTER TABLE audit_logs ADD COLUMN tenant_id TEXT DEFAULT 'default_tenant';

			CREATE INDEX IF NOT EXISTS idx_hosts_tenant ON hosts(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant ON audit_logs(tenant_id);
		`,
	},
	{
		version: 12,
		name:    "create_identity_tables",
		sql: `
			CREATE TABLE IF NOT EXISTS roles (
				id TEXT PRIMARY KEY,
				tenant_id TEXT DEFAULT 'default_tenant',
				name TEXT NOT NULL,
				description TEXT DEFAULT '',
				permissions TEXT DEFAULT '[]',
				is_system BOOLEAN DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_roles_tenant ON roles(tenant_id);

			INSERT OR IGNORE INTO roles (id, tenant_id, name, description, permissions, is_system)
			VALUES
				('role_admin',   'default_tenant', 'Administrator', 'Full access to all features',
				 '["*"]', 1),
				('role_analyst', 'default_tenant', 'Analyst', 'Investigation and read access',
				 '["hosts:read","sessions:read","siem:read","siem:write","incidents:read","incidents:write","evidence:read","snippets:read","compliance:read"]', 1),
				('role_readonly','default_tenant', 'Read-Only', 'View-only access to dashboards',
				 '["hosts:read","sessions:read","siem:read","incidents:read","compliance:read"]', 1);

			CREATE TABLE IF NOT EXISTS users (
				id TEXT PRIMARY KEY,
				tenant_id TEXT DEFAULT 'default_tenant',
				email TEXT NOT NULL,
				name TEXT NOT NULL,
				password_hash TEXT DEFAULT '',
				auth_provider TEXT DEFAULT 'local',
				is_mfa_enabled BOOLEAN DEFAULT 0,
				mfa_secret TEXT DEFAULT '',
				role_id TEXT DEFAULT 'role_admin',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				last_login_at DATETIME,
				FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE SET NULL
			);
			CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_tenant ON users(email, tenant_id);

			CREATE TABLE IF NOT EXISTS sso_providers (
				id TEXT PRIMARY KEY,
				tenant_id TEXT DEFAULT 'default_tenant',
				name TEXT NOT NULL,
				provider_type TEXT NOT NULL,
				client_id TEXT DEFAULT '',
				client_secret TEXT DEFAULT '',
				issuer_url TEXT DEFAULT '',
				metadata_url TEXT DEFAULT '',
				callback_url TEXT DEFAULT '',
				is_enabled BOOLEAN DEFAULT 0,
				auto_provision BOOLEAN DEFAULT 1,
				default_role_id TEXT DEFAULT 'role_analyst',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (default_role_id) REFERENCES roles(id) ON DELETE SET NULL
			);
			CREATE INDEX IF NOT EXISTS idx_sso_providers_tenant ON sso_providers(tenant_id);
		`,
	},
	{
		version: 13,
		name:    "fk_constraints_and_totp_encryption",
		sql: `
			PRAGMA foreign_keys = ON;

			-- Add FK indexes for tables missing them
			CREATE INDEX IF NOT EXISTS idx_credentials_tenant ON credentials(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_sessions_tenant ON sessions(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_snippets_tenant ON snippets(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_tunnels_tenant ON tunnels(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_incidents_tenant ON incidents(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_evidence_tenant ON evidence(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_evidence_chain_tenant ON evidence_chain(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_config_changes_tenant ON config_changes(tenant_id);

			-- Add encrypted_mfa_secret column for AES-wrapped TOTP secrets
			ALTER TABLE users ADD COLUMN encrypted_mfa_secret BLOB DEFAULT NULL;
		`,
	},
}

func (d *Database) Migrate() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	var currentVersion int
	row := d.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("get current version: %w", err)
	}

	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}

		tx, err := d.db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for migration %d: %w", m.version, err)
		}

		if _, err := tx.Exec(m.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %d (%s): %w", m.version, m.name, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
			m.version, m.name,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.version, err)
		}
	}

	return nil
}
