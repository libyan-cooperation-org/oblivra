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
			-- Foreign keys are enabled globally in the connection DSN or init logic.
			-- PRAGMA foreign_keys = ON is a no-op inside a transaction.

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
	{
		version: 14,
		name:    "extend_tenants_for_isolation",
		sql: `
			ALTER TABLE tenants ADD COLUMN tier TEXT DEFAULT 'free';
			ALTER TABLE tenants ADD COLUMN status TEXT DEFAULT 'Active';
			ALTER TABLE tenants ADD COLUMN crypto_salt TEXT DEFAULT '';
			ALTER TABLE tenants ADD COLUMN updated_at DATETIME DEFAULT '2026-04-20 00:00:00';
		`,
	},
	{
		version: 15,
		name:    "create_cloud_assets",
		sql: `
			CREATE TABLE IF NOT EXISTS cloud_assets (
				id TEXT,
				tenant_id TEXT DEFAULT 'default_tenant',
				provider TEXT NOT NULL,
				region TEXT NOT NULL,
				account_id TEXT NOT NULL,
				type TEXT NOT NULL,
				name TEXT NOT NULL,
				status TEXT DEFAULT 'active',
				metadata TEXT DEFAULT '{}',
				tags TEXT DEFAULT '{}',
				first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
				last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (id, tenant_id)
			);
			CREATE INDEX IF NOT EXISTS idx_cloud_assets_tenant_provider ON cloud_assets(tenant_id, provider);
			CREATE INDEX IF NOT EXISTS idx_cloud_assets_account ON cloud_assets(account_id);
		`,
	},
	{
		version: 16,
		name:    "scim_normalization_extension",
		sql: `
			ALTER TABLE users ADD COLUMN external_id TEXT DEFAULT '';
			ALTER TABLE users ADD COLUMN active BOOLEAN DEFAULT 1;
			ALTER TABLE users ADD COLUMN display_name TEXT DEFAULT '';
			ALTER TABLE users ADD COLUMN user_type TEXT DEFAULT 'user';
			ALTER TABLE users ADD COLUMN title TEXT DEFAULT '';
			ALTER TABLE users ADD COLUMN department TEXT DEFAULT '';
			ALTER TABLE users ADD COLUMN organization TEXT DEFAULT '';
			ALTER TABLE users ADD COLUMN preferred_language TEXT DEFAULT 'en-US';
			ALTER TABLE users ADD COLUMN groups_json TEXT DEFAULT '[]';
			ALTER TABLE users ADD COLUMN scim_attributes_json TEXT DEFAULT '{}';

			CREATE UNIQUE INDEX IF NOT EXISTS idx_users_external_id ON users(external_id) WHERE external_id <> '';
		`,
	},
	{
		version: 17,
		name:    "identity_connector_persistence",
		sql: `
			CREATE TABLE IF NOT EXISTS identity_connectors (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				name TEXT NOT NULL,
				type TEXT NOT NULL,
				enabled BOOLEAN DEFAULT 1,
				config_json TEXT NOT NULL,
				sync_interval_mins INTEGER DEFAULT 60,
				last_sync DATETIME,
				status TEXT DEFAULT 'ok',
				error_message TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_identity_connectors_tenant ON identity_connectors(tenant_id);
		`,
	},
	{
		version: 18,
		name:    "automated_triage_extension",
		sql: `
			ALTER TABLE incidents ADD COLUMN triage_score INTEGER DEFAULT 0;
			ALTER TABLE incidents ADD COLUMN triage_reason TEXT DEFAULT '';
		`,
	},
	{
		version: 19,
		name:    "report_factory_init",
		sql: `
			CREATE TABLE IF NOT EXISTS report_templates (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				name TEXT NOT NULL,
				description TEXT,
				sections_json TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE IF NOT EXISTS report_schedules (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				template_id TEXT NOT NULL,
				name TEXT NOT NULL,
				interval_mins INTEGER DEFAULT 1440,
				next_run_at DATETIME,
				recipients_json TEXT,
				is_active BOOLEAN DEFAULT 1,
				last_run_at DATETIME,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (template_id) REFERENCES report_templates(id) ON DELETE CASCADE
			);

			CREATE TABLE IF NOT EXISTS generated_reports (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				schedule_id TEXT,
				template_id TEXT NOT NULL,
				title TEXT NOT NULL,
				period_start DATETIME,
				period_end DATETIME,
				file_path TEXT,
				status TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (template_id) REFERENCES report_templates(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_reports_tenant ON generated_reports(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_schedules_next ON report_schedules(next_run_at) WHERE is_active = 1;
		`,
	},
	{
		version: 20,
		name:    "dashboard_studio_init",
		sql: `
			CREATE TABLE IF NOT EXISTS dashboards (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				name TEXT NOT NULL,
				description TEXT,
				layout TEXT DEFAULT 'grid',
				owner_id TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE IF NOT EXISTS dashboard_widgets (
				id TEXT PRIMARY KEY,
				dashboard_id TEXT NOT NULL,
				title TEXT NOT NULL,
				viz_type TEXT NOT NULL,
				query_oql TEXT NOT NULL,
				layout_x INTEGER DEFAULT 0,
				layout_y INTEGER DEFAULT 0,
				layout_w INTEGER DEFAULT 4,
				layout_h INTEGER DEFAULT 4,
				refresh_interval_secs INTEGER DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (dashboard_id) REFERENCES dashboards(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_dash_tenant ON dashboards(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_widgets_dash ON dashboard_widgets(dashboard_id);
		`,
	},
	{
		version: 21,
		name:    "asset_intel_init",
		sql: `
			ALTER TABLE hosts ADD COLUMN criticality_score INTEGER DEFAULT 1;
			ALTER TABLE hosts ADD COLUMN criticality_reason TEXT;
			ALTER TABLE users ADD COLUMN criticality_score INTEGER DEFAULT 1;
			ALTER TABLE users ADD COLUMN criticality_reason TEXT;
		`,
	},
	{
		version: 22,
		name:    "graph_engine_init",
		sql: `
			CREATE TABLE IF NOT EXISTS graph_nodes (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				type TEXT NOT NULL,
				meta_json TEXT,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE TABLE IF NOT EXISTS graph_edges (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				from_node_id TEXT NOT NULL,
				to_node_id TEXT NOT NULL,
				type TEXT NOT NULL,
				attributes_json TEXT,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY(from_node_id) REFERENCES graph_nodes(id),
				FOREIGN KEY(to_node_id) REFERENCES graph_nodes(id)
			);
			CREATE INDEX IF NOT EXISTS idx_graph_nodes_tenant ON graph_nodes(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_graph_edges_tenant ON graph_edges(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_graph_edges_from ON graph_edges(from_node_id);
			CREATE INDEX IF NOT EXISTS idx_graph_edges_to ON graph_edges(to_node_id);
		`,
	},
	{
		version: 23,
		name:    "host_event_integrity_chain",
		sql: `
			ALTER TABLE host_events ADD COLUMN event_hash TEXT DEFAULT '';
			ALTER TABLE host_events ADD COLUMN prev_hash TEXT DEFAULT '';
			CREATE INDEX IF NOT EXISTS idx_host_events_hash ON host_events(event_hash);
		`,
	},
	{
		version: 24,
		name:    "create_rotation_policies",
		sql: `
			CREATE TABLE IF NOT EXISTS rotation_policies (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				credential_id TEXT NOT NULL,
				frequency_days INTEGER NOT NULL,
				last_rotation DATETIME,
				next_rotation DATETIME,
				notify_only BOOLEAN DEFAULT 0,
				is_active BOOLEAN DEFAULT 1,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (credential_id) REFERENCES credentials(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_rotation_tenant ON rotation_policies(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_rotation_next ON rotation_policies(next_rotation) WHERE is_active = 1;
		`,
	},
	{
		version: 25,
		name:    "create_suppression_rules",
		sql: `
			CREATE TABLE IF NOT EXISTS suppression_rules (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				label TEXT NOT NULL,
				description TEXT,
				rule_id TEXT,
				field TEXT NOT NULL,
				value TEXT NOT NULL,
				is_regex BOOLEAN DEFAULT 0,
				expires_at DATETIME,
				is_active BOOLEAN DEFAULT 1,
				last_matched_at DATETIME,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_suppression_lookup ON suppression_rules(tenant_id, rule_id, field) WHERE is_active = 1;
			CREATE INDEX IF NOT EXISTS idx_suppression_global ON suppression_rules(tenant_id, field) WHERE is_active = 1 AND (rule_id IS NULL OR rule_id = '');
		`,
	},
	{
		// Phase 27.2.2 — DHCP lease ledger for temporal entity
		// resolution. When alerts cite an IP, we need to map it back
		// to the host that held the lease at the alert's timestamp,
		// not whichever host happens to hold it now.
		// See `internal/identity/lease.go` for the full contract.
		version: 26,
		name:    "create_dhcp_lease_log",
		sql: `
			CREATE TABLE IF NOT EXISTS dhcp_lease_log (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				tenant_id   TEXT NOT NULL,
				ip          TEXT NOT NULL,
				hostname    TEXT,
				mac         TEXT,
				started_at  DATETIME NOT NULL,
				ended_at    DATETIME,
				source      TEXT
			);
			CREATE INDEX IF NOT EXISTS idx_dhcp_lease_lookup
				ON dhcp_lease_log(tenant_id, ip, started_at, ended_at);
		`,
	},
	{
		// Phase 22.2 / Phase 30 — GDPR Article 30 evidence for the
		// right-to-erasure workflow. When a tenant is wiped via
		// CryptographicWipe, this immutable log records who did it,
		// when, the reason, and a SHA-256 of the previous tenant row
		// (so an auditor can prove the deletion was complete and
		// non-tampered). Append-only by convention; no UPDATE/DELETE
		// statements are issued against this table by application code.
		version: 27,
		name:    "create_tenant_deletion_log",
		sql: `
			CREATE TABLE IF NOT EXISTS tenant_deletion_log (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				tenant_id TEXT NOT NULL,
				tenant_name TEXT,
				deleted_by_user TEXT NOT NULL,
				deleted_by_role TEXT,
				reason TEXT,
				prev_row_hash TEXT NOT NULL,
				deleted_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_tenant_deletion_log_tenant ON tenant_deletion_log(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_tenant_deletion_log_when ON tenant_deletion_log(deleted_at);
		`,
	},
	{
		// Compliance audit packages — generated SOC2/HIPAA/etc reports.
		// Until v28 these were held in a process-local map and lost on restart;
		// now durable per-tenant + searchable.
		version: 28,
		name:    "create_compliance_packages",
		sql: `
			CREATE TABLE IF NOT EXISTS compliance_packages (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				framework TEXT NOT NULL,
				generated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				records INTEGER DEFAULT 0,
				integrity_proof TEXT NOT NULL,
				download_url TEXT,
				from_ts TEXT,
				to_ts TEXT
			);
			CREATE INDEX IF NOT EXISTS idx_compliance_packages_tenant ON compliance_packages(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_compliance_packages_when ON compliance_packages(generated_at);
		`,
	},
	{
		// Agent registry — was process-local; now durable so the fleet
		// reappears across restarts. The columns mirror the in-memory
		// agent struct that handleAgentRegister populates today.
		version: 29,
		name:    "create_agent_registry",
		sql: `
			CREATE TABLE IF NOT EXISTS agent_registry (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				hostname TEXT,
				remote_address TEXT,
				os TEXT,
				arch TEXT,
				version TEXT,
				status TEXT DEFAULT 'online',
				last_seen DATETIME,
				registered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				quarantined INTEGER DEFAULT 0,
				metadata TEXT
			);
			CREATE INDEX IF NOT EXISTS idx_agent_registry_tenant ON agent_registry(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_agent_registry_status ON agent_registry(status);
		`,
	},
	{
		// Failed-login lockout state — was sync.Map, lost on restart.
		// Persisting so a malicious actor can't bounce the process to
		// reset their lockout timer (audit finding).
		version: 30,
		name:    "create_login_lockouts",
		sql: `
			CREATE TABLE IF NOT EXISTS login_lockouts (
				ip TEXT PRIMARY KEY,
				email TEXT,
				failed_count INTEGER DEFAULT 0,
				last_failed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				locked_until DATETIME
			);
			CREATE INDEX IF NOT EXISTS idx_login_lockouts_until ON login_lockouts(locked_until);
		`,
	},
	{
		// GDPR Art. 15 / 17 + CCPA §1798.105 / §1798.110 data-subject
		// request workflow. Operator-facing access + deletion requests
		// land here; the wipe primitive (DisasterService) executes the
		// state change. This table is the audit trail.
		version: 31,
		name:    "create_dsr_requests",
		sql: `
			CREATE TABLE IF NOT EXISTS dsr_requests (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				subject_id TEXT NOT NULL,
				request_type TEXT NOT NULL CHECK (request_type IN ('access', 'deletion')),
				reason TEXT,
				requester TEXT,
				verification TEXT,
				status TEXT NOT NULL DEFAULT 'pending'
				  CHECK (status IN ('pending', 'fulfilled', 'rejected')),
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				resolved_at DATETIME,
				resolved_by TEXT,
				resolution_notes TEXT
			);
			CREATE INDEX IF NOT EXISTS idx_dsr_tenant_status ON dsr_requests(tenant_id, status);
			CREATE INDEX IF NOT EXISTS idx_dsr_subject ON dsr_requests(subject_id);
		`,
	},
	{
		// RFC 3161 — external timestamping for the evidence chain. Each
		// row records one timestamp anchor returned by a Time Stamp
		// Authority. The chain hash at that anchor is sealed against
		// the TSA's signing certificate, giving the audit trail
		// non-repudiable proof that "the evidence chain held this
		// state at this UTC time" — closes the WORM-storage audit gap
		// for non-privileged-attacker threat models.
		version: 32,
		name:    "create_evidence_timestamps",
		sql: `
			CREATE TABLE IF NOT EXISTS evidence_timestamps (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				chain_hash TEXT NOT NULL,
				chain_height INTEGER NOT NULL,
				tsa_url TEXT NOT NULL,
				tsa_token BLOB NOT NULL,
				obtained_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				verified_at DATETIME
			);
			CREATE INDEX IF NOT EXISTS idx_evidence_ts_height ON evidence_timestamps(chain_height);
		`,
	},
	{
		// agent_oplog (Tamper Path 1, Layer 1) — agents ship their
		// own diagnostic log lines here so the central server has a
		// write-once copy. An attacker who pwns the host and `rm`s
		// /var/log/oblivra-agent.log can't erase events that already
		// reached the server.
		//
		// The table is append-only by convention (no UPDATE paths in
		// code; CryptoWipe is the only deletion path, and it's audit-
		// logged). The `UNIQUE(agent_id, batch_seq)` does idempotency
		// (retries are safe) AND gap-detection (a missing batch_seq is
		// suspicious — recorded as agent:sequence_gap in tamper rules).
		version: 33,
		name:    "create_agent_oplog",
		sql: `
			CREATE TABLE IF NOT EXISTS agent_oplog (
				id            INTEGER PRIMARY KEY AUTOINCREMENT,
				agent_id      TEXT NOT NULL,
				batch_seq     INTEGER NOT NULL,
				ts            TEXT NOT NULL,
				level         TEXT NOT NULL,
				source        TEXT,
				message       TEXT NOT NULL,
				received_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
				prev_hash     TEXT,
				UNIQUE(agent_id, batch_seq, ts, message)
			);
			CREATE INDEX IF NOT EXISTS idx_agent_oplog_received ON agent_oplog(received_at DESC);
			CREATE INDEX IF NOT EXISTS idx_agent_oplog_agent_seq ON agent_oplog(agent_id, batch_seq);
		`,
	},
	{
		// agent_heartbeats (Tamper Path 1, Layer 3) — most-recent-only
		// table (one row per agent, replaced on each heartbeat). Used
		// by the missed-heartbeat scanner: a sweep every 60s flags
		// agents whose latest heartbeat is older than 90s.
		//
		// Carries the size + inode of the agent's local log file plus
		// its last hash chain hash; tamper rules detect:
		//   • log_truncated  — log_file_size shrank between heartbeats
		//   • log_inode_changed — log file rotated (legitimate but logged)
		//   • time_skew      — wall_clock differs from server > 60s
		version: 34,
		name:    "create_agent_heartbeats",
		sql: `
			CREATE TABLE IF NOT EXISTS agent_heartbeats (
				agent_id          TEXT PRIMARY KEY,
				received_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
				wall_clock        TEXT NOT NULL,
				uptime_s          INTEGER NOT NULL,
				log_file_inode    INTEGER NOT NULL,
				log_file_size     INTEGER NOT NULL,
				last_hash         TEXT,
				prev_log_size     INTEGER,
				prev_log_inode    INTEGER,
				skew_seconds      REAL
			);
		`,
	},
	{
		// Phase 36.8: Schema cleanup — drop tables and columns that the
		// audit verified have zero application-layer references.
		//
		// Verified-dead (full hand-grep against internal/* on 2026-04-29):
		//   - tunnels         (Phase 32 shell-removal leak)
		//   - sso_providers   (never wired into any auth flow)
		//   - graph_nodes     (GraphService never reads/writes)
		//   - graph_edges     (same)
		//   - audit_logs.user_agent column (declared, never inserted/selected)
		//
		// NOT dropped (audit was wrong; live consumers exist):
		//   - dhcp_lease_log, login_lockouts, dsr_requests, evidence_timestamps
		//   - compliance_packages (mis-named — backs /api/v1/audit/packages)
		//   - agent_registry, agent_oplog, agent_heartbeats (tamper-detection)
		//
		// SQLite limitation: ALTER TABLE DROP COLUMN was only added in 3.35.0
		// (March 2021). The CGO-bundled SQLite is recent enough; if a deployment
		// pins an older SQLite the audit_logs.user_agent drop will fail
		// gracefully via IF NOT EXISTS-style guards in `DROP COLUMN IF EXISTS`.
		// We use the IF EXISTS variant on every statement so reapplying this
		// migration on a fresh DB (where the tables never existed) is safe.
		version: 35,
		name:    "drop_phase36_dead_tables",
		sql: `
			DROP INDEX IF EXISTS idx_tunnels_tenant;
			DROP TABLE IF EXISTS tunnels;

			DROP TABLE IF EXISTS sso_providers;

			DROP INDEX IF EXISTS idx_graph_nodes_tenant;
			DROP TABLE IF EXISTS graph_nodes;

			DROP INDEX IF EXISTS idx_graph_edges_tenant;
			DROP INDEX IF EXISTS idx_graph_edges_from;
			DROP INDEX IF EXISTS idx_graph_edges_to;
			DROP TABLE IF EXISTS graph_edges;

			ALTER TABLE audit_logs DROP COLUMN user_agent;
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
