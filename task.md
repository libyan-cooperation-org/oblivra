# OBLIVRA — Master Task Tracker

> **Status Tiers**:
> - `[s]` = **Scaffolded** (Code exists, compiles, architectural proof)
> - `[v]` = **Validated** (Tested under load, unit tests pass, functionally correct)
> - `[x]` = **Production-Ready** (Survives 72h soak, hardened, documented, unchallengeable)
> - `[ ]` = Not started
>
> **Last audited: 2026-04-25** — Phase 22 Productization Sprint + Platform Split Model
> **Verification pass 2026-04-25** — every `[x]` item in Phases 22, 23, 25, 26 was re-checked against actual code paths; corrections applied in place. See `## Phase 28: 2026-04-25 Verification Audit` at the bottom of this file for the full delta.
>
> **Companion files** (not this file's concern):
> - [`ROADMAP.md`](ROADMAP.md) — Phases 16–26 (CSPM, K8s, vuln mgmt, etc.)
> - [`RESEARCH.md`](RESEARCH.md) — Phase 13 (DARPA/NSA-grade research)
> - [`BUSINESS.md`](BUSINESS.md) — Phase 14 (certifications, legal, GTM)
> - [`FUTURE.md`](FUTURE.md) — Cross-cutting (chaos engineering, deception, i18n)
> - [`STRATEGY.md`](STRATEGY.md) — Phase 22 strategic rationale

---

## 🏗️ Platform Architecture — Golden Rule

> **Desktop = Sensitive + Local + Operator Actions**
> **Web = Shared + Scalable + Multi-user**

### 🖥️ DESKTOP (Wails App) — MUST be here
> Anything involving secrets, OS access, or direct operator control.

| Category | Features |
|---|---|
| 🔐 **Security & Secrets** | Vault (AES-256), OS keychain, FIDO2/YubiKey, Password manager |
| 💻 **Terminal & SSH** | SSH client (keys/agent), Local PTY, Multi-session grid, Port forwarding/tunneling, Session recording, Multi-exec |
| 📁 **File & System Access** | SFTP file browser, Local file operations, Upload/download, `~/.ssh/config` import |
| 🧪 **Local / Offline** | Local SIEM (optional), Local detection engine (offline testing), Local log ingestion, Air-gap mode |
| 🧰 **Operator Tools** | Command palette (local hosts), Workspace layouts, Plugin dev/testing, CLI mode |
| 🔧 **System-Level Actions** | Build/sign agents, Generate certificates, Forensics acquisition (disk/memory), Local response actions (kill process, isolate host) |

### 🌐 WEB (Browser UI) — MUST be here
> Anything involving teams, scale, or central control.

| Category | Features |
|---|---|
| 📊 **SIEM & Observability** | Log search (fleet-wide), Dashboards, Real-time streaming, Aggregations |
| 🚨 **Alerting** | Alert dashboard, Acknowledge/assign, Escalation workflows, Notifications (Slack/email/Teams) |
| 🧠 **Detection (Production)** | Central rule engine, Rule management, Correlation engine, Alert deduplication |
| 🕵️ **Threat Hunting** | Query interface, Saved searches, MITRE heatmap, Investigation tools |
| 🖥️ **Fleet Management** | Agent list & status, Health monitoring, Config push, Upgrades |
| 🔁 **SOAR** | Playbooks, Case management, Incident timelines, Jira/ServiceNow integration |
| 🏢 **Enterprise** | Users & roles (RBAC), Multi-tenancy, SAML/OIDC/MFA, API keys |
| 📜 **Compliance** | Reports (PCI/ISO/SOC2), Audit logs, Legal hold, Retention policies |
| 🌍 **Threat Intelligence** | TAXII feeds, IOC database, Enrichment pipeline |

### ⚖️ HYBRID (Both Desktop + Web)
> Same feature, different scope.

| Feature | Desktop Scope | Web Scope |
|---|---|---|
| 🔍 Search | Local logs | Fleet logs |
| 🧠 Detection Rules | Testing rules | Production rules |
| 🔎 Threat Hunting | Local investigation | Organization-wide |
| 📊 Dashboards | Personal | Shared |
| 🧾 Alerts | Local alerts | Global alerts |
| 🧬 Forensics | Collect evidence | View/analyze evidence |

### ❌ NEVER on Web (Desktop ONLY — always)
- SSH private keys
- Vault master key
- Raw terminal access (PTY)
- Local filesystem access
- Agent signing keys
- Plugin execution engine

---

## Development Rules ⚠️

> [!IMPORTANT]
> **Every production-exposed capability MUST have a frontend UI OR an API workflow.**
> Internal engines (e.g. enrichment pipeline, policy logic) do not require immediate UI.
> No service is "done" until it has a corresponding Svelte 5 component, an API endpoint, or a route in `App.svelte`.

> [!CAUTION]
> **ARCHITECTURAL GRADUATION POLICY after Phase 10.**
> - Phases 0-10: Core platform (Feature-complete v1). No further additions to core pipeline.
> - Phases 11-15: Extension modules (Independently hardened before next begins).
> - Phases 16+: Market expansion (Requires v1 soak test pass as prerequisite).
> - Every phase beyond 10 requires documented justification and independent hardening gates.

---

## Core Platform Features (Pre-existing) ✅

> All exist in code, compile, and are wired into `container.go`.

### Terminal & SSH
- [x] SSH client with key/password/agent auth (`internal/ssh/client.go`, `auth.go`) 🖥️
- [x] Local PTY terminal (`local_service.go`) 🖥️
- [x] SSH connection pooling (`internal/ssh/pool.go`) 🖥️
- [x] SSH config parser + bulk import (`internal/ssh/config_parser.go`) 🖥️
- [x] SSH tunneling / port forwarding (`internal/ssh/tunnel.go`, `tunnel_service.go`) 🖥️
- [x] Session recording & playback (`recording_service.go`, `internal/sharing/`) 🖥️
- [x] Session sharing & broadcast (`broadcast_service.go`, `share_service.go`) 🏗️
- [x] Multi-exec concurrent commands (`multiexec_service.go`) 🖥️
- [x] Terminal grid with split panes (`frontend/src/components/terminal/`) 🖥️
- [x] File browser & SFTP transfers (`file_service.go`, `transfer_manager.go`) 🖥️

### Security & Vault
- [x] AES-256 encrypted Vault (`internal/vault/vault.go`, `crypto.go`) 🖥️
- [x] OS keychain integration (`internal/vault/keychain.go`) 🖥️
- [s] FIDO2 / YubiKey support (`internal/security/fido2.go`, `yubikey.go`) 🖥️
- [x] TLS certificate generation (`internal/ssh/certificate.go`, `cmd/certgen/`) 🏗️
- [x] Security key modal UI (`frontend/src/components/security/`) 🖥️
- [x] Snippet vault / command library (`snippet_service.go`) 🏗️

### Productivity
- [x] Notes & runbook service (`notes_service.go`) 🏗️
- [x] Workspace manager (`workspace_service.go`) 🖥️
- [x] AI assistant — error explanation, command gen (`ai_service.go`) 🏗️
- [x] Theme engine with custom themes (`theme_service.go`) 🏗️
- [x] Settings & configuration UI (`settings_service.go`, `pages/Settings.svelte`) 🏗️
- [x] Command palette & quick switcher (`frontend/src/components/ui/`) 🏗️
- [x] Auto-updater service (`updater_service.go`) 🖥️

### Collaboration
- [x] Team collaboration service (`team_service.go`, `internal/team/`) 🌐
- [x] Sync service (`sync_service.go`) 🏗️

### Ops & Monitoring
- [x] Unified Ops Center — multi-syntax search (LogQL, Lucene, SQL, Osquery) (`pages/OpsCenter.svelte`) 🏗️
- [x] Splunk-style analytics dashboard (`pages/SplunkDashboard.svelte`) 🏗️
- [x] Customizable widget dashboard (`frontend/src/components/dashboard/`) 🏗️
- [x] Network discovery service (`discovery_service.go`, `worker_discovery.go`) 🏗️
- [x] Global topology visualization (`pages/GlobalTopology.svelte`) 🏗️
- [x] Bandwidth monitor chart (`frontend/src/components/charts/BandwidthMonitor.svelte`) 🏗️
- [x] Fleet heatmap (`frontend/src/components/fleet/FleetHeatmap.svelte`) 🌐
- [x] Osquery integration — live forensics (`internal/osquery/`) 🏗️
- [x] Log source manager (`logsource_service.go`, `internal/logsources/`) 🏗️
- [x] Health & metrics service (`health_service.go`, `metrics_service.go`) 🏗️
- [x] Telemetry worker (`worker_telemetry.go`, `telemetry_service.go`) 🏗️

### Infrastructure
- [x] Plugin framework with Lua sandbox (`internal/plugin/`, `plugin_service.go`) 🏗️
- [x] Plugin manager UI (`pages/PluginManager.svelte`) 🏗️
- [x] Event bus pub/sub (`internal/eventbus/`) 🏗️
- [x] Output batcher (`output_batcher.go`) 🏗️
- [x] Hardening module (`hardening.go`) 🏗️
- [x] Sentinel file integrity monitor (`sentinel.go`) 🏗️
- [x] CLI mode binary (`cmd/cli/`) 🖥️
- [x] SIEM benchmark tool (`cmd/bench_siem/`) 🏗️
- [x] Soak test generator (`cmd/soak_test/`) 🏗️

---

## Phase 0: Stabilization ✅

- [x] Final audit of all service constructor signatures in `container.go`
- [x] Resolve remaining compile errors across all services
- [x] Verify all 16+ services start/stop cleanly via `ServiceRegistry`
- [x] Full integration smoke test (SSH, SIEM, Vault, Alerting, Compliance)

---

## Phase 0.1: Day Zero Hardening ✅

- [x] Recursive Directory Creation — `platform.EnsureDirectories()` to `app.New()` 🏗️
- [x] Onboarding / Inception UI — Redirect to Setup Wizard if hosts count is 0 🏗️
- [x] Core Rule Library — `sigma/core/` seeded with 25+ essential rules 🏗️
- [x] Subprocess Validation — startup check for `os.Executable()` re-entry 🏗️
- [x] First-Run Analytics — Trace "Time to First Alert" 🏗️

---

## Phase 0.2: Test Suite Stabilization ✅

- [x] Fix Ingest Package Regressions — `ingest.SovereignEvent` → `events.SovereignEvent`
- [x] Restore Diagnostics Interface — `DiagnosticsService.GetSnapshot()` in `smoke_test.go`
- [x] Resolve Test Name Collisions — no `TestHighThroughputIngestion` duplicate
- [x] Verify Test Pass Rate — `go test ./...` passes
- [x] Resolve Architectural Violations — Detection decoupled via `SIEMStore` interface

---

## Phase 0.3: Web Dashboard / Enterprise Platform (MVP) ✅ 🌐

- [x] Initialize `frontend-web/` (Bun + Vite + Svelte 5)
- [x] Tailwind CSS and design tokens
- [x] `APP_CONTEXT` detection (Wails vs. Browser)
- [x] `/api/v1/auth/login` + `Login.svelte` + `AuthService.ts`
- [x] `Onboarding.svelte` wizard + `FleetService.ts`
- [x] `SIEMSearch.svelte` (Lucene queries, live paginated results) 🏗️
- [x] `AlertManagement.svelte` (WebSocket feed, status workflow) 🏗️

---

## Phase 0.4: Accessibility & Enterprise Scaling ✅

- [x] WCAG 2.1 AA Compliance Audit (pattern-fills, ARIA labels, keyboard nav)
- [x] Real-time SIEM heatmaps with pattern-fills
- [x] High-density "War Room" grid view
- [x] Fleet status overview with drill-down
- [x] OIDC provider redirects (Google/Okta)
- [x] SAML 2.0 metadata exchange flow
- [x] Multi-tenant registration & isolation
- [x] BadgerDB optimized for 1,000+ nodes

---

## Phase 0.5: Architectural Hardening (Desktop vs. Browser) ✅

- [x] `context.ts` — `APP_CONTEXT` detection, `IS_DESKTOP`, `IS_BROWSER`, `IS_HYBRID` exports
- [x] `isRouteAvailable()`, `getServiceCapabilities()`, `configureHybridMode()` / `disconnectHybridMode()`
- [x] `ContextRoute.svelte` route guard (desktop/web/any context scoping)
- [x] `RouteGuard` component — wraps routes, shows `UnavailableScreen` with context hint
- [x] `ContextBadge` — status bar pill (DESKTOP/HYBRID/BROWSER), click opens server connection panel
- [x] `api.ts` BASE_URL (localhost for Desktop, same-origin for Browser)
- [x] `GlobalFleetChart.svelte` 🌐
- [x] `FleetManagement.svelte` — agent fleet console 🌐
- [x] `IdentityAdmin.svelte` — User/Role/Provider admin 🌐
- [x] `SIEMSearch.svelte` full-text SIEM query page 🏗️
- [x] Desktop → remote OBLIVRA Server connection (Backend API Proxy)
- [x] `CommandRail.svelte` — context classification on all nav items; locked items show `⊘`
- [x] `AppLayout.svelte` — `isDrawerVisible()` replaces hardcoded allowlist
- [x] Route availability matrix: 60+ routes classified (desktop-only, browser-only, both)
- [x] `docs/architecture/desktop_vs_browser.md` — context detection spec, route matrix

---

## Phase 1: Core Storage + Ingestion + Search ✅

### 1.1 — Storage Layer
- [v] Integrate BadgerDB 🏗️
- [s] Integrate Bleve (pure-Go Lucene alternative) 🏗️
- [s] Integrate Parquet Archival 🏗️
- [v] Syslog (RFC 5424/3164) ingestion pipeline 🌐
- [v] Crash-safe Write-Ahead Log (WAL) 🏗️
- [s] Storage adapter interfaces (SQLite → Bleve/BadgerDB) 🏗️
- [s] Migrate SIEM queries to Bleve + BadgerDB 🏗️
- [x] Benchmark: 10M event search <5s

### 1.2 — Ingestion Pipeline
- [s] Syslog listener with TLS (`internal/ingest/syslog.go`)
- [s] JSON, CEF, LEEF parsers (`internal/ingest/parsers.go`)
- [s] Schema-on-read normalization
- [s] Backpressure + rate limiting (`internal/ingest/pipeline.go`)
- [s] `IngestService` wired in `internal/app/`
- [v] 72h sustained soak test at 5,000 EPS
- [v] 180k event burst (18,000+ EPS peak); 10,000 EPS sustained

### 1.3 — Search & Query
- [s] Lucene-style query parser (extend `transpiler.go`/Bleve) 🏗️
- [s] Field-level indexing via Bleve field mappings 🏗️
- [s] Aggregation support (facets, group-by, histograms) 🏗️
- [s] Saved searches (DB model + API + UI) 🏗️
- [x] Refactor `EmitEvent` signature in `internal/services/interfaces.go`
- [x] Update `main.go` with `RootPath`
- [x] Fix `Taskfile.yml` includes
- [x] Refactor `internal/services/analytics_service.go`
- [x] Refactor `internal/services/identity_service.go` (Signatures)
- [x] Refactor `internal/services/bookmark_service.go` (`QuickSearch`)
- [x] Refactor `internal/app/app.go` (Analytics calls)
- [x] Refactor `internal/services/interfaces.go` (Events fix)
- [x] Refactor `internal/services/logsource_service.go` (Context args)
- [x] Refactor `internal/services/siem_service.go` (Context args)
- [x] Refactor `internal/services/compliance_service.go` (Context args)
- [x] Refactor `internal/services/snippet_service.go` (Secret args)
- [x] Refactor `internal/services/interfaces.go` (SessionOperations context)
- [x] Refactor `internal/services/ssh_service.go` (Implement Service & Executor)
- [x] Refactor `internal/services/local_service.go` (Context args)
- [x] Verify build (`go build ./...`)
- [x] Transition `wails.json` to NPM
- [x] Verify full Wails bundle (`wails3 build`)
- [x] Performance validation: <5s for 10M events
- [x] OpenAPI 3.0 spec: machine-readable API contracts with auto-generated SDKs 🌐

### 1.7 — Mobile On-Call View
- [ ] Responsive web-app for alert acknowledgement and triage on mobile 🌐

### 20.4.5 — Lookup Tables
- [s] CSV/JSON lookup file upload and API-based updates 🏗️
- [s] Exact, CIDR, Wildcard, Regex match support 🏗️
- [s] `GET /api/v1/lookups/query` — OQL-ready single-key lookup 🏗️
- [s] Pre-built lookups: RFC 1918, Port-to-Service, MITRE technique-to-name 🏗️

---

## Phase 2: Alerting + REST API ✅

### 2.1 — Alerting Hardening
- [x] YAML detection rule loader (`internal/detection/rules/`) 🏗️
- [x] Rule engine: threshold, frequency, sequence, correlation rules 🏗️
- [x] Alert deduplication with configurable windows 🏗️
- [x] Notifications: webhook, email, Slack, Teams channels 🌐
- [x] Test: alerts fire within 10s

### 2.1.5 — Notification Escalation
- [x] Multi-level escalation chains (Analyst → Lead → Manager → CISO) 🌐
- [x] Time-based escalation + SLA breach detection 🌐
- [x] On-call rotation schedules + acknowledgment API 🌐
- [x] `EscalationCenter.svelte` — Policies, Active, On-Call, History tabs 🌐

### 2.2 — Headless REST API
- [x] `internal/api/rest.go` with full HTTP router 🌐
- [x] SIEM search, alerts, agent, ingestion status, auth endpoints 🌐
- [x] API key authentication (`internal/auth/apikey.go`) 🌐
- [x] User accounts + RBAC (`internal/auth/`) 🌐
- [x] TLS for all external listeners 🌐

#### 🔍 OQL & Engine
- [x] `oql.ts` centralized evaluator
- [x] Real-time telemetry filtering logich in `SIEMPanel.svelte` 🏗️
- [x] `AlertDashboard.svelte` (filtering, ack, status) 🏗️
- [x] Prometheus-compatible `/metrics` endpoint 🌐
- [x] Liveness + readiness probes 🌐
- [x] All services: JSON structured logging

### 2.3 — Web UI Hardening
- [x] Real-time streaming search in `SIEMPanel.svelte` 🏗️
- [x] `AlertDashboard.svelte` (filtering, ack, status) 🏗️
- [x] Prometheus-compatible `/metrics` endpoint 🌐
- [x] Liveness + readiness probes 🌐
- [x] All services: JSON structured logging

---

## Phase 3: Threat Intel + Enrichment ✅

### 3.1 — Threat Intelligence
- [x] STIX/TAXII Client (`internal/threatintel/taxii.go`) 🏗️
- [x] Offline rule ingestion (JSON, OpenIOC) 🏗️
- [x] `MatchEngine` O(1) IP/Hash lookups 🏗️
- [x] IOC Matcher integrated into `IngestionService` 🏗️
- [x] `ThreatIntelPanel.svelte` + `ThreatIntelDashboard.svelte` 🏗️

### 3.2 — Enrichment Pipeline
- [x] GeoIP module (MaxMind offline DB, `internal/enrich/geoip.go`)
- [x] DNS Enrichment (ASN, PTR records, `internal/enrich/dns.go`)
- [x] Asset/User Mapping
- [x] Enrichment Pipeline orchestrator (`internal/enrich/pipeline.go`)
- [x] `EnrichmentViewer.svelte` — GeoIP, DNS/ASN, asset mapping, IOC correlation 🌐

### 3.3 — Advanced Parsing
- [x] Windows Event Log parser (`internal/ingest/parsers/windows.go`) 🏗️
- [x] Linux syslog + journald parser (`internal/ingest/parsers/linux.go`) 🏗️
- [x] Cloud audit parsers (AWS/Azure/GCP) 🌐
- [x] Network logs (NetFlow, DNS, firewall) 🌐
- [x] Unified parser registry (`internal/ingest/parsers/registry.go`) 🏗️

### 3.4 — Graph Infrastructure
- [ ] Foundational graph database layer for entity relationship tracking 🏗️

---

## Phase 4: Detection Engineering + MITRE ✅

- [x] 82 YAML detection rules across all 12 tactics, 45+ techniques 🏗️
- [x] MITRE ATT&CK technique mapper (`internal/detection/mitre.go`) 🏗️
- [x] Correlation engine (`internal/detection/correlation.go`) 🏗️
- [x] MITRE ATT&CK heatmap (`MitreHeatmap.svelte`) 🏗️
- [s] Recruit 10 design partners (0 recruited; pilot agreement pending)
- [v] Validate: <5% false positives, 30+ ATT&CK techniques

### 4.1/4.2 — Commercial Readiness
- [ ] POC Generator & Support Bundle: one-command diagnostic bundle generation 🏗️
- [ ] Compliance Artifacts: pre-built legal templates (DPA, BAA, SCCs) and compatibility matrices 🌐

### 4.5 — Hardening Sprint ✅
- [x] `SIEMPanel.svelte` decoupled sub-components
- [x] Bounded Queue buffering on `eventbus.Bus`
- [x] SIEM Database Query Timeouts (10s contexts)
- [x] Incident Aggregation in Alert Dashboard
- [x] Regex Timeouts / Safe Parsing (ReDoS prevention)
- [x] Role-Based Access controls on destructive alert endpoints
- [x] API key auth + RBAC + TLS
- [x] Built-in attack simulator (MITRE ATT&CK technique replay)
- [x] Detection coverage score + technique gap report
- [x] Continuous detection validation (scheduled self-test)
- [x] `PurpleTeam.svelte`

#### 🛠️ Component & Page Hardening
- [x] `CommandPalette` (hostname fix, ARIA roles, tabindex)
- [x] `SIEMSearch` (OQL Parser integration, layout stabilization)
- [x] `Settings` (Form binding resolution, a11y warnings)
- [x] `DataTable` (ARIA roles, keyboard sorting, aria-sort)
- [x] `Badge` & `KPI` (Standardized a11y roles)
- [x] `SearchBar` (Keyboard navigation, a11y roles)
- [x] `VaultLocked` & `Login` (Barrier UI, MFA bridge, browser auth)

---

## Phase 5: Limits, Leaks & Lifecycles ✅

- [x] LRU/TTL bounded memory for `internal/detection/correlation.go`
- [x] Asynchronous value log GC for BadgerDB
- [x] Incident Aggregation: mutable DB records (New/Active/Investigating/Closed)
- [x] `SIEMPanel.svelte` + Wails app → `svelte-routing`
- [x] Pre-compiled binary release workflow (GitHub Actions)
- [x] Zero-dependency `docker-compose.yml` deployment

---

## Phase 6: Forensics & Compliance ✅

- [s] Merkle tree immutable logging (`internal/integrity/merkle.go`)
- [s] Evidence locker with chain of custody (`internal/forensics/evidence.go`)
- [x] Enhanced FIM with baseline diffing
- [s] PCI-DSS, NIST, ISO 27001, GDPR, HIPAA, SOC2 Type II compliance packs
- [x] PDF/HTML reporting engine (`internal/compliance/report.go`)
- [x] Forensics service Wails integration (`internal/app/forensics_service.go`)
- [x] Compliance evaluator engine (`internal/compliance/evaluator.go`)
- [x] `EvidenceVault.svelte` — chain-of-custody browser, verify, seal, export 🏗️
- [x] `RegulatorPortal.svelte` — read-only audit log + compliance package generation 🌐
- [s] Validate: external audit pass (self-audited only)

### 6.5 — Legal-Grade Digital Evidence 🏗️
- [x] RFC 3161 Timestamping + batch submission
- [x] NIST SP 800-86 chain-of-custody formalization
- [x] E01/AFF4 forensic export with integrity proofs
- [x] Expert Witness Package: provenance reports + tool validation
- [ ] **End-to-End Event Integrity Proof** — agent-side `event_hash`, continuous pipeline hash chaining, query-time verification mode

### 6.6 — Regulator-Ready Audit Export 🌐
- [x] JSON Lines with cryptographic chaining (RFC 3161/Merkle)
- [x] Regulator Portal: scoped, read-only audit viewer
- [x] One-click compliance package generation (SOC2, ISO27001, PCI-DSS, HIPAA, GDPR)

---

## Sovereign Meta-Layer ✅

### 🔴 Tier 1: Documents
- [x] Formal Threat Model (STRIDE) — `docs/threat_model.md`
- [x] Security Architecture Document — `docs/security_architecture.md`
- [x] Operational Runbook — `docs/ops_runbook.md`
- [x] Business Continuity Plan — `docs/bcp.md`

### 🟡 Tier 2: Near-Term Code

#### Supply Chain Security
- [x] SBOM auto-generation (`syft`/`cyclonedx-gomod` in GHA)
- [x] Signed releases (Cosign / Sigstore)
- [s] Artifact provenance attestation (SLSA Level 3)
- [x] Reproducible build verification

#### Self-Observability
- [x] `pprof` HTTP endpoints
- [x] Goroutine watchdog
- [x] Internal deadlock detection (`runtime.SetMutexProfileFraction`)
- [x] Self-health anomaly alerts via event bus
- [x] `SelfMonitor.svelte`

#### Disaster & War-Mode Architecture
- [x] Air-gap replication node mode
- [x] Offline update bundles (USB-deployable signed archives)
- [x] Kill-switch safe-mode (read-only, forensic-only)
- [ ] **Kill-Switch Abuse Protection** — Multi-party authorization (M-of-N), hardware key requirements, audit escalation bounds
- [x] Encrypted snapshot export/import
- [x] Cold backup restore automation + validation

#### Governance Layer
- [x] Data retention policy engine
- [x] Legal hold mode
- [x] Data destruction workflow (cryptographic wipe + audit trail)
- [x] Audit log of audit log access (meta-audit)
- [x] Sovereign-grade vault resilience (30s heartbeat + auto-recovery)
- [x] Vault daemon crash-loop backoff (exponential retry)
- [x] Synthetic anti-tamper self-test (`-trigger-tamper` flag)
- [x] **Phase 26: Sovereign Intelligence & Stability**
    - [x] Detection circuit breaker (`MAX_COST` throttling)
    - [x] Dynamize `FusionDashboard.svelte` with live clusters
    - [x] Cluster-aware node highlighting in `ThreatGraph.svelte`
    - [x] Sigma rule cost-based rejection in `Verifier`
- [x] `ComplianceCenter.svelte` — Governance tab with real-time scoring

### 🔵 Tier 3: Strategic

#### Licensing & Monetization
- [x] Feature flag framework — 48 features, 4 tiers (`internal/licensing/license.go`)
- [x] Offline license activation — Ed25519 signed tokens, hardware-bound, no network call
- [x] Per-agent metering + usage tracking (`internal/services/licensing_service.go`)
- [x] License enforcement middleware + `LicensingService` Wails binding + `/license` UI page

#### Advanced Isolation & Zero-Trust Architecture
- [ ] Vault process isolation (separate signing key service)
- [x] Memory zeroing guarantees on all crypto operations
- [ ] Service-level privilege separation design doc

#### AI Governance
- [x] Sovereign Tactical UI Overhaul (design tokens, `global.css`, `CommandRail.svelte`, `AppLayout.svelte`)
- [x] Tactical dashboards refactor (`Dashboard.svelte`, `FleetDashboard.svelte`, `SIEMPanel.svelte`, `AlertDashboard.svelte`)
- [x] System-wide Prop Type & Accessibility Audit
- [x] Agent Hardening: PII Redaction + Goroutine Leak Audits
- [x] Architecture Boundary Enforcement (`tests/architecture_test.go`)
- [x] Model explainability layer, bias logging, false positive audit trail
- [x] Training dataset isolation, offline retraining pipeline

#### Red Team / Validation Engine
- [x] Built-in attack simulator (MITRE ATT&CK technique replay)
- [x] Detection coverage score + technique gap report
- [x] Continuous detection validation (scheduled self-test)
- [x] `PurpleTeam.svelte`

#### Certification Readiness
- [ ] ISO 27001 organizational certification alignment
- [ ] SOC 2 Type II evidence collection automation
- [ ] Common Criteria evaluation preparation
- [ ] FIPS 140-3 crypto module compliance pathway

---

## Tier 1-4 Hardening Gates ✅

### 🔴 Tier 1: Foundational Security
- [x] SAST: `golangci-lint` with `gosec`, `errcheck`, `staticcheck`
- [x] SCA: `syft` + `grype` in CI
- [x] Unit Test Coverage: ≥80% for new/modified packages
- [x] Architecture Boundary Enforcement: `go vet` + custom linter
- [x] Frontend Linting: `eslint` + `prettier` + `tsc --noEmit`
- [x] Secret Detection: `gitleaks` in pre-commit + CI

### 🟡 Tier 2: Runtime & Integration
- [x] Integration Tests: end-to-end for ingestion, detection, alerting
- [x] Fuzz Testing: `go-fuzz` for parsers, network handlers, deserialization
- [x] Performance Benchmarking: regression checks on EPS, query latency
- [x] Memory Leak Detection: `go test -memprofile` + `pprof` in CI
- [x] Race Condition Detection: `go test -race` for all packages
- [x] Container Image Hardening: distroless base, non-root user, minimal packages

### 🟠 Tier 3: Operational & Resilience
- [x] Threat Modeling Review (STRIDE for new features)
- [x] Security Architecture Review (peer review)
- [x] Penetration Testing: external vendor engagement (annual)
- [x] Disaster Recovery Testing: quarterly failover drills
- [x] Configuration Hardening Audit: CIS Benchmarks
- [x] Supply Chain Integrity: SBOM verification, signed artifacts

### 🟣 Tier 4: Compliance & Assurance
- [x] Compliance Audit: ISO 27001, SOC 2 Type II, PCI-DSS evidence collection
- [x] Code Audit: independent security code review
- [x] Incident Response Playbook Review: annual tabletop exercises
- [x] Privacy Impact Assessment (PIA): GDPR, CCPA
- [x] Legal Review: EULA, data processing agreements, open-source licensing

---

## Phase 7: Agent Framework ✅

- [v] Agent binary scaffold (`cmd/agent/main.go`) 🏗️
- [v] File tailing, Windows Event Log streaming, system metrics, FIM collectors 🏗️
- [v] gRPC/TLS/mTLS transport layer 🏗️
- [v] Zstd compression + offline buffering (local WAL) 🏗️
- [v] Edge filtering + PII redaction 🏗️
- [v] Agent registration + heartbeat API 🌐
- [v] `AgentConsole.svelte` + fleet-wide config push 🌐
- [x] eBPF collector (`internal/agent/ebpf_collector_linux.go` — kprobe/tracepoint, epoll ring-buffer, 4 probes, /proc fallback) 🏗️
- [x] Agent mutex-guarded fleet map (`agentsMu sync.RWMutex` in `RESTServer`)
- [x] `GET /api/v1/agents` — full fleet list with status 🌐

### 7.5 — Agentless Collection Methods ✅
- [x] `WMICollector` — Windows Event Log via WMI/WinRM; poll interval, multi-channel (`internal/agentless/collectors.go`) 🌐
- [x] `SNMPCollector` — SNMPv2c/v3 trap listener; MIB-based event translation 🌐
- [x] `RemoteDBCollector` — SQL audit log polling (Oracle, SQL Server, Postgres, MySQL); cursor-based HWM 🌐
- [x] `RESTPoller` — Declarative REST API polling for SaaS sources; JSON path extraction 🌐
- [x] `CollectorManager` — registry, `StartAll()`, `StopAll()`, `Statuses()` 🌐
- [x] `GET /api/v1/agentless/status` + `GET /api/v1/agentless/collectors` 🌐

---

## Phase 8: Autonomous Response (SOAR) ✅

- [v] Case management (CRUD, assignment, timeline) 🌐
- [v] Playbook Engine: selective response & approval gating 🏗️
- [v] Rollback Integrity: state-aware recovery 🏗️
- [x] Jira/ServiceNow integration (`internal/incident/integrations.go`) 🌐
- [v] Deterministic Execution Service 🏗️
- [x] `PlaybookBuilder.svelte` — visual SOAR builder, step canvas, action palette, execute-against-incident 🏗️
- [x] `PlaybookMetrics.svelte` — MTTR, success/failure rates, bottleneck identification 🏗️
- [x] `GET/POST /api/v1/playbooks` — CRUD; `POST /api/v1/playbooks/run`; `GET /api/v1/playbooks/metrics` 🌐

### Playbook Marketplace / Community Library
- [x] Import/export playbooks as YAML bundles (rule marketplace schema: `rule + metadata + test fixtures + changelog`)
- [ ] Version-controlled playbook repository
- [ ] Community-contributed playbook catalog

---

## Phase 9: Ransomware Defense ✅

- [x] Entropy-based behavioral detection (`internal/detection/ransomware_engine.go`) 🏗️
- [x] Canary file deployment (`canary_deployment_service.go`) 🏗️
- [v] Honeypot infrastructure 🏗️
- [x] Automated network isolation (`network_isolator_service.go`) 🏗️
- [x] `RansomwareCenter.svelte` — defense layers, host status, isolation controls, event log 🏗️
- [x] `GET /api/v1/ransomware/events|hosts|stats` + `POST /api/v1/ransomware/isolate` 🌐

### Immutable Backup Verification
- [ ] Verify backup integrity hashes on schedule
- [ ] Alert if backup has not completed in policy window
- [ ] Test restore automation (validate backups are actually recoverable)

### Ransomware Negotiation Intelligence
- [ ] Threat actor TTP database (known ransomware groups)
- [ ] Decryptor availability checking (NoMoreRansom integration)
- [ ] Payment risk scoring (OFAC sanctions list checking)

---

## Phase 10: UEBA / ML ✅

- [v] Per-user/entity behavioral baselines (persistence in BadgerDB) 🏗️
- [v] Isolation Forest anomaly detection (deterministic seeding) 🏗️
- [v] Identity Threat Detection & Response (EMA behavior tracking) 🏗️
- [v] Threat hunting interface (`ThreatHunter.svelte`) 🏗️
- [x] `UEBADashboard.svelte` — risk heatmap, entity drill-down, anomaly feed 🏗️
- [x] `GET /api/v1/ueba/profiles|anomalies|stats` 🌐

### 10.5 — Peer Group Behavioral Analysis ✅
- [x] Auto-cluster by role, department, access patterns; dynamic recalculation; min-N validation
- [x] Aggregate behavioral statistics; deviation scoring (σ from group centroid)
- [x] "First for peer group" alerts; composite individual × peer anomaly scoring
- [x] `PeerAnalytics.svelte` — peer group explorer, σ-deviation outlier detection, risk comparison bars
- [x] `GET /api/v1/ueba/peer-groups` + `GET /api/v1/ueba/peer-deviations` 🌐

### 10.6 — Multi-Stage Attack Fusion Engine ✅
- [x] Kill chain tactic mapping; sliding window progression tracking; 3+ stage alert
- [x] Campaign clustering by shared entities; confidence scoring
- [x] Bayesian probabilistic scoring; seeded campaign data for demo
- [x] `FusionDashboard.svelte` — kill chain visualization, campaign cluster graph, confidence scores
- [x] `GET /api/v1/fusion/campaigns` + `GET /api/v1/fusion/campaigns/{id}/kill-chain` 🌐

---

## Phase 11: NDR ✅

- [x] NetFlow/IPFIX collector 🌐
- [x] DNS log analysis engine — DGA and DNS tunneling detection 🌐
- [x] TLS metadata extraction — JA3/JA3S fingerprints (no decryption) 🌐
- [x] HTTP proxy log parser — normalized inspection 🌐
- [x] eBPF network probes (extend agent) 🏗️
- [x] Lateral movement detection 🌐
- [x] `NDRDashboard.svelte` — flow table, anomaly cards, protocol stats 🌐
- [x] `LateralMovementEngine` — multi-hop connection correlation 🌐
- [x] `NetworkMap.svelte` — topology visualization 🌐
- [x] `GET /api/v1/ndr/flows|alerts|protocols` 🌐
- [x] Validate: lateral movement <5 min, 90%+ C2 identification

---

## Phase 12: Enterprise ✅

- [x] Multi-tenancy with data partitioning
- [s] HA clustering (Raft consensus) — `internal/cluster/`, `cluster_service.go`
- [x] User & Role DB models + migration v12 (`internal/database/users.go`)
- [x] OIDC/OAuth2 + SAML 2.0 + TOTP MFA + Granular RBAC engine
- [x] `IdentityService` — user CRUD, local login, MFA, RBAC checking
- [x] `GET /api/v1/users` + `GET /api/v1/roles` 🌐
- [x] Data lifecycle management — `lifecycle_service.go` (7 retention policies, legal hold, 6h purge loop)
- [x] `ExecutiveDashboard.svelte` — KPIs, posture, compliance badges
- [x] `PasswordVault.svelte` — full credential vault manager
- [x] Validate: 50+ tenants, 99.9% uptime

---

## Phase 13: Research Milestones ✅ (Partial)

- [x] TLA+ model: `DeterministicExecutionService` (5 invariants, liveness: `EventualExecution`)
- [x] TLA+ model: detection rule engine execution paths (`NoSpuriousAlerts` + `WindowStateInvariant`)
- [x] Benchmark datasets expanded (`test/datasets/` — CIC-IDS-2017, Zeek traces)
- [x] `contains()` helper bug fixed in `harness.go`
- [x] Benchmark runner wired (`cmd/benchmark_ids_zeek/`)
- [v] Strategic Research Publications (internal whitepapers drafted)

---

## Phase 15: Sovereignty ✅

- [x] Zero Internet dependency audit (`zero_internet_audit.md`)
- [x] Offline Update Bundle support (`ApplyOfflineUpdate` in `updater.go`)
- [x] Signature verification enforcement (`internal/updater/signature.go` — Ed25519, ldflags key injection)
- [x] Offline update bundle integrity validation + downgrade protection (`DowngradeProtector`, semver-aware)

---

## Phase 16: Full Security Audit — 31 Findings ✅

> All 31 findings from the 2026-03-12/16 senior-engineer security audit resolved.

- [x] All 🔴 Critical findings resolved (plaintext passwords, hardcoded credentials, sanitizer bugs, plugin goroutine leak)
- [x] All 🟡 High findings resolved (TLS enforcement, WebSocket allowlist, timing side-channels, Argon2 adaptive memory, CSP)
- [x] All 🟠 Medium findings resolved (crypto rand, DeployKey injection, multiexec cap, search limit, RBAC context key)
- [x] All 🔵 Low findings resolved (CDN leak, vault bypass, acceptable timing risk, bridge try/catch fallback)
- [x] EventBus: `SubscribeWithID` + `Unsubscribe` with atomic per-Bus counter

---

## Phase 17: Commercial-Grade Capabilities ✅

- [x] Full Sigma → Oblivra transpiler with all field modifiers (`|contains`, `|startswith`, `|endswith`, `|re:`, `|all`)
- [x] MITRE ATT&CK tag extraction (14 tactics mapped; `T####`/`T####.###` techniques)
- [x] `logsource` → `EventType` mapping for 15+ source types; timeframe parsing
- [x] `LoadSigmaFile()` + `LoadSigmaDirectory()` + auto-load from `sigma/` on start
- [x] `sigma_test.go` (6 test cases) + `sigma_fuzz_test.go` (7-entry seed corpus)
- [x] OpenTelemetry Tracing: `InitTracing()`, adaptive sampler, `RecordDetectionMatch` etc.
- [x] Supply chain: multi-OS CI matrix, SBOM (SPDX + CycloneDX), Cosign signing, SLSA provenance

---

## Phase 18: Loose Ends Closed ✅

- [x] AI Assistant wired (`/ai-assistant`, Ollama status badge, 3 modes)
- [x] `MitreHeatmap.svelte` fully wired (`/mitre-heatmap`)
- [x] OTel → Grafana Tempo pipeline (`docker-compose.yml` extended)
- [x] `ops/` config directory: `prometheus.yml`, `tempo.yml`, Grafana datasources + pre-built dashboard

---

## Phase 19: v1.1.0 ✅

- [x] `README.md` fully rewritten (accurate stack, architecture diagram, build instructions)
- [x] `CHANGELOG.md v1.1.0` — complete entry covering Phases 11–19
- [x] `DiagnosticsModal.svelte` — live ingest EPS, goroutines, heap, GC, event bus drops, health grade
- [x] Sigma hot-reload — `fsnotify v1.8.0` watcher, 500ms debounce, `ReloadSigmaRules()` Wails method
- [x] Unlock bug — all 3 root causes fixed (HasKeychainEntry, VaultUnlock path, polling loop → event subscription)

---

## Phase 20: Detection & Docs Expansion ✅

- [x] **82 total detection rules** (30 new): Windows LOLBin/PowerShell/shadow copy/LSASS/WMI/registry/Defender/PTH/DCSync/Golden Ticket; Linux rootkit/LD_PRELOAD/Docker escape/unsigned kernel module; Cloud AWS root/IAM/S3/Azure impossible travel; Network DNS tunneling/SMB lateral/C2 beaconing; Supply chain; Insider threat; OT/ICS Modbus
- [x] `detection_engine_test.go` — 18 tests
- [x] `vault_service_test.go` — 12 tests
- [x] `ingest/pipeline_unit_test.go` — queue/process, buffer drop, metrics, stop cleanly, benchmarks
- [x] `tests/smoke_test.go` — expanded with alerting, Sigma, diagnostics, observability subtests
- [x] **5 operator docs** in `docs/operator/`: `quickstart.md`, `detection-authoring.md`, `sigma-rules.md`, `alerting-config.md`, `api-reference.md`

### 20.1 — SovereignQL (OQL)
- [x] Custom pipe-based query language (OQL) for tactical analytics 🏗️
- [x] **Query Language Identity** — formalized grammar definition, query planner guarantees, computational cost modeling

### 20.4 — SCIM Normalization
- [x] Identity data ingestion and normalization (SCIM) 🌐

### 20.7 — Identity Connectors
- [x] Native integration connectors for Active Directory, Okta, and major IdPs 🌐

### 20.9 — Automated Triage
- [x] Automated incident triage scoring based on RBA and Asset Intel 🏗️

### 20.10 — Report Factory
- [x] Automated generation of scheduled reports 🌐

### 20.11 — Dashboard Studio
- [x] Custom dashboard builder with widget canvas 🌐

---

## Phase 21: Architectural Scaling ✅

- [x] **Partitioned Event Pipeline** — 8 shards, FNV-1a hash routing, per-shard worker pool + adaptive controller (`internal/ingest/partitioned_pipeline.go`)
- [x] **Write-Ahead Log** — CRC32 per record, 50ms fsync window, 10MB guard, replay on startup (`internal/storage/wal.go`)
- [x] **Streaming Enrichment LRU Cache** — 50,000 IP, 10-min TTL, RWMutex concurrent reads (`internal/enrich/cache.go`)
- [x] **Detection Rule Route Index** — EventType → `[]Rule` inverted index, `RebuildRouteIndex()` on hot-reload, ~13× speedup (`internal/detection/rule_router.go`)
- [x] **Query Execution Limits** — `DefaultQueryLimits` + `HeavyQueryLimits`, `Plan()`, `Validate()`, `BoundedContext()` (`internal/database/query_planner.go`)
- [x] **Bounded Worker Pools** — configurable, backpressure, panic-safe (`internal/platform/worker_pool.go`)
- [x] `git rm -r --cached frontend/node_modules` — node_modules purged from git tracking

### 21.5 — Asset Intelligence
- [ ] Foundational asset intelligence and asset criticality scoring 🌐

---

## Phase 22: Productization (The Strategic Pivot)

> **Context**: OBLIVRA has SIEM + EDR + SOAR + UEBA + NDR + hybrid desktop/web. Feature parity with early Splunk/CrowdStrike is real.
> This phase converts engineering into a product. No new features — only reliability, isolation, cost control, detection ecosystem, and trust.
> See [`STRATEGY.md`](STRATEGY.md) for the full strategic rationale.

---

### 🗺️ Execution Sequence — Open Work Build Order
> Sub-phases are documented in their original numbering (22.1–22.7) but must be **executed in the priority order below**.
> Older open items from phases 3, 6, 9, 20, 21, 24 are slotted into the correct sprint.

| Sprint | Theme | Sub-Phases / Items | ~Effort |
|---|---|---|---|
| **S0 🚨** | Emergency: dark-site URLs + marketing copy | 22.6 (URLs only), 24.4 | < 1 day |
| **S1 🔴** | Multi-Tenant Isolation | **22.2** (all 8 items) | 2 wks |
| **S2 🔴** | Reliability Gate (4 of 9) | **22.1** ★ (reconnect, degradation, soak CI, BadgerDB recovery) | 2 wks |
| **S3 🟡** | Setup Wizard + Trust Signals | **22.5** ★ (wizard, security.txt, threat model, crypto doc) | 1.5 wks |
| **S4 🟡** | Storage Economics | **22.3** (Hot/Warm/Cold, rate limits, cost dashboard) | 1.5 wks |
| **S5 🟡** | Detection Quality | **22.4** remaining + **22.1** deferred items | 2 wks |
| **S6 🟢** | Feature Gap Closure | **24.2** (Arabic i18n, backup integrity, VT) + **24.3** (partials) | 2 wks |
| **S7 🟢** | Platform & Analytics | **Phase 20** (OQL, reports, studio) + **21.5** + **3.4** | 2 wks |
| **S8 🟢** | Commercial Readiness | **4.1/4.2**, **22.5** deferred, **1.7** (mobile) | 1 wk |
| **S9 🔵** | Architecture Hardening | **22.6** remaining + **6.5** + **Phase 9** open | 2 wks |
| **S10 🔵** | Sovereign / Nation-State | **22.7** (all 6) + Sovereign Meta-Layer remaining | 3 wks |
| **Defer ⚫** | v2+ Features | Cloud connectors, ClickHouse, ITDR, AI/LLM Sec, Endpoint Prevention | — |

> **Current sprint**: ~~S0~~ ✅ → ~~S1~~ ✅ (22.2 verified, structural per-tenant isolation in place) → **S2** (Reliability Gate — chaos harness + soak regression already shipped; agent reconnect, BadgerDB recovery, graceful degradation, time sync remain)

---

---

### 🔧 Immediate Hygiene

- [x] **Purge node_modules from git** — `git rm -r --cached frontend/node_modules frontend-web/node_modules`
- [x] **Wails RPC bridge rate limiting** — per-method debounce on `NuclearDestruction`, `Unlock`, `DeleteHost`
- [x] **Browser mode: VaultGuard + store.svelte Wails crash** — `IS_BROWSER` guards on all Wails imports
- [x] **S0: Dark-site URL eradication** — `internal/sync/engine.go`: removed hardcoded `https://sync.oblivrashell.dev`; `NewSyncEngine()` now accepts `syncEndpoint` param; empty string = offline mode; guards added to `pushToCloud`/`fetchFromCloud`. `internal/updater/updater.go`: `CheckUpdate()`/`DownloadAndApply()` return clean disabled signal when `repoURL == ""` (already the default in `container.go`). Compiled ✅

---

### 22.1 — Reliability Engineering

- [x] **Chaos test harness** — `cmd/chaos/main.go`: WAL CRC replay, BadgerDB VLog corruption + truncate-mode reopen, OOM/burst load-shed probe, clock skew ±5 min, **and (2026-04-25 add) Scenario 5: agent reconnect with 1000+ events in flight**. `cmd/chaos-fuzzer/` and `cmd/chaos-harness/` extend this.
- [x] **Agent reconnect guarantee** — Per-event sequence numbers (`Event.Seq`), persistent cursor at `<dataDir>/wal/cursor.json` (`internal/agent/cursor.go`), `WAL.TruncateUpTo(ackedSeq)` partial-truncate (`internal/agent/wal.go`), server response now includes `acked_seq`, server tracks `AgentInfo.LastAckedSeq` and dedupes replays with `Seq <= LastAckedSeq`. Validated end-to-end by chaos scenario 5: 1500 events, cycle 1 acks 1..750 → WAL keeps 751..1500; cycle 2 (post-"restart") sends only 751..1500, never reissuing 1..750. **Open**: legacy agent-side `Truncate()` is now marked deprecated; remove call sites in a follow-up. *(Phase 22.1)*
- [x] **BadgerDB corruption recovery** — `internal/storage/badger.go:NewHotStore` now ladders through 3 recovery levels: normal open → truncate-mode open (drops torn vlog tail) → read-only fallback. Read-only opens log a CRITICAL line so operators know to extract via the new `HotStore.ExportSnapshot(dst)` (Badger native protobuf backup stream) and reinitialise from `HotStore.ImportSnapshot(src)`. Service no longer goes dark on a routine power-loss tear.
- [x] **Graceful degradation under overload** — Pipeline already classified DEGRADED/CRITICAL state at >3× rated EPS or >95% buffer fill (`internal/ingest/adaptive.go:101`). New: `LoadStatus.String()` for stable wire format, lightweight `GET /api/v1/health/load` endpoint (returns just status + queue/EPS/dropped — safe for 10s-cadence polling), `pipeline:load_status_changed` bus event published on every transition, and `DegradedBanner.svelte` (`frontend-web/src/components/`) wired into `App.svelte` to render an amber/red top-of-page banner with dismiss button. *(Phase 22.1)*
- [x] **Automated soak regression** — `.github/workflows/soak.yml`: 30-min 5,000 EPS soak on every release tag, fails if EPS drops >10%, event loss >0.1%, or min-window EPS <50% of target. Captures heap pprof. Verified 2026-04-25.
- [ ] **Node failure simulation** — kill Raft leader mid-election; verify cluster recovers, no double-processed events
- [v] **Deterministic Replay System** — MVP shipped: `cmd/replay` provides `--mode=capture` (writes per-record SHA-256 manifest from a WAL) and `--mode=verify` (re-walks the WAL and asserts every record matches the manifest by index/length/SHA). This locks down *input determinism*; the alert-equivalence layer (replay through detection engine + diff alerts) is the follow-up. The MVP is enough to detect WAL tampering or drift between two captures of the same source.
- [x] **Time Synchronization Enforcement** — `internal/events/events.go` adds `TimeConfidence` enum (`normal`/`late`/`skewed`/`unknown`) and `ClassifyTime(ts, now)` pure function. `pipeline.processEvent` tags every event with `EventTimeConfidence` + signed `SkewSeconds` *before* WAL/index writes (durable on disk). Skewed events log a single info line per occurrence so operators can correlate with NTP failures. Thresholds: ±60s → normal, >60s past ≤5min → late, >60s future or >5min past → skewed.
- [ ] **Upgrade Safety Guarantees** — versioned schema migration rollback, dual-run (old+new pipeline), per-tenant canary upgrades

---

### 22.2 — Multi-Tenant Isolation

- [x] **Tenant-prefixed BadgerDB keyspace** — `formatEventKey()` writes `tenant:{id}:events:{ts}:{uuid}` (`internal/storage/siem_badger.go:72`); ALL scan paths use the prefix (`siem_badger.go:109,134`, `badger_source.go:29,37`). Verified.
- [x] **Bleve index per tenant** — `SearchEngine.getIndex(tenantID)` returns a tenant-scoped index from `s.indexes[tenantID]` map; separate filesystem paths `bleve_{tenantID}.idx` (`internal/search/bleve.go:45-85`). Cross-tenant queries are structurally impossible. Verified.
- [v] **Correlation state isolation** — `correlation_store.go:20` keys on `tenant:{tenantID}:correlation:{ruleID}:{window}:{groupKey}` ✓, but in-memory LRU at `correlation.go:138` is keyed on `tenant+ruleID` only (groupKey isolation enforced *within* the LRU at `correlation.go:153-162`, not in the LRU key itself). Functionally correct; partial against the literal "tenantID+ruleID+groupKey" claim. Not a security issue.
- [x] **Per-tenant encryption keys** — `tenant_crypto.go:25-33` `DeriveTenantKey` = `HMAC-SHA256(masterKey, tenantID || salt)` → AES-256. Per-tenant rotation supported. Verified.
- [x] **Query sandbox enforcement** — `internal/database/query_planner.go:62` returns `"sandbox violation: query must contain TenantID predicate"`. Verified.
- [x] **Tenant provisioning API** — `POST /api/v1/admin/tenants` (`rest_tenants.go:14-57`) generates salt + creates tenant; Badger/Bleve indexes auto-create on first write. Idempotent. Verified.
- [x] **Tenant deletion audit trail** — `rest_tenants.go:handleAdminTenantWipe` now: (1) reads tenant name/tier before wipe so the audit record can identify what was deleted, (2) accepts an optional `reason` in the request body (e.g. `"GDPR Art. 17 request from data subject ABC123"`), (3) calls `s.audit.Log("tenant.deleted", ...)` which is Merkle-chained and tamper-evident via `InitIntegrity` replay, (4) emits a `tenant:deleted` bus event for downstream consumers, (5) records a `tenant.delete_failed` audit entry on error so attempted-but-failed erasures still leave evidence. Captures actor user_id, email, IP, basis. *(Phase 22.2 + GDPR Art. 30)*
- [v] **50-tenant isolation test** — `tests/tenant_isolation_test.go` runs 50 tenants × **10 events each** (claim said 1000) and verifies cross-tenant search returns 0 results. Structural isolation confirmed; throughput claim overstated by 100×. Worth bumping to 1k events/tenant to validate at scale.

> **Note (2026-04-25)**: The redundant `fmt.Sprintf("TenantID:%s AND ...")` in `rest.go:803,846` flagged in Phase 25.9 as a leak vector has been removed. Storage-layer `MustTenantFromContext` + per-tenant Bleve index dispatch is the source of truth (auth middleware plumbs tenant via `database.WithTenant` in `apikey.go:143,164`). Removing the string concat removed dead code that *looked* like an injection vulnerability without actually being one.

---

### 22.3 — Cost & Performance Layer

- [x] **Sigma `count by` aggregate functions** — `parseCountByCondition()` with full regex; `| count() > N`, `| count by FIELD > N`, `| count(FIELD) by GROUPBY > N`; rules auto-promoted to `FrequencyRule` with correct `Threshold` and `GroupBy` (`internal/detection/sigma.go`); 2 new test cases added
- [ ] **Ingestion rate limiting per tenant** — configurable EPS ceiling; excess events dropped with counter; UI shows utilization bar
- [ ] **Hot/Warm/Cold tiered storage** — complete `QueryPlanner` hot/cold split: Hot (BadgerDB 0–30d), Warm (Parquet 30–180d), Cold (S3-compatible 180d+)
- [ ] **Query cost estimation** — estimate rows × field complexity × time range; reject if cost > tenant limit; expose estimate in UI
- [ ] **Enrichment budget** — GeoIP + DNS capped at N lookups/sec/tenant; excess tagged `enrichment:skipped`; visible in diagnostics
- [ ] **Storage usage dashboard** — per-tenant: events stored, index size, archive size, projected 30/90/365 day cost
- [ ] **Economic Model Enforcement** — CPU/RAM/IO caps per tenant, query cost billing hooks, strict storage quota enforcement

---

### 22.4 — Detection Engineering Platform + Operator Mode

#### Rule Versioning & Management ✅
- [x] **Rule versioning** — `Version string` field on `Rule` struct; `RuleEngine.previousRules` map; `UpsertRule()` archives previous; `RollbackRule()` restores; `GetPreviousVersion()` accessor (`internal/detection/rules.go`)
- [x] **MITRE coverage gap report** — `GenerateMITREGapReport()` per-technique scoring (covered/partial/none); MITRE Navigator JSON layer export with colour coding (`internal/detection/rules.go`)
- [x] **Rule test framework** — `RuleTestFixture`, `RuleTestResult`, `RuleTestSuiteResult`; `TestRule()` runs fixtures against conditions; `matchRuleConditions()` with `regex:` prefix support (`internal/detection/rules.go`)

#### Operator Mode — The Killer Workflow (Partial)
- [v] **SSH → anomaly banner** — `OperatorService.GetContext()` (`internal/services/operator_service.go:65-150`) retrieves SIEM host alerts and exposes them through `OperatorContext`. **UI gap**: status-bar surfacing + one-keypress event panel keybind not yet implemented in frontend. 🖥️
- [ ] **Event row → enrichment pivot** — click IP/host in SIEM results → inline enrichment card (GeoIP, ASN, TI match, open ports) 🏗️
- [v] **Host isolation from terminal context** — `OperatorMode.svelte:44-52` → `agentStore.toggleQuarantine(agentID, true)` → `ToggleQuarantine()` (`agent.svelte.ts:117-127`). **UI gap**: no `Ctrl+Shift+I` keybind handler found, no confirmation modal, titlebar status indicator unverified. 🖥️
- [ ] **One-click memory/process capture** — trigger forensic snapshot, auto-seal SHA-256, auto-add to active incident evidence 🖥️
- [ ] **Operator timeline** — unified chronological view: terminal commands + SIEM events + enrichment + playbook executions + evidence 🏗️
- [ ] **Autonomous Hunt** — scheduled and automated threat hunting queries based on Threat Intel 🌐
- [ ] **Operator Cognitive Load Design** — transition from dashboards to decision engine: alert ranking, "next best action" prompts, investigation graphs 🏗️

#### Detection Engineering
- [ ] **Detection-as-code workflow** — rules in Git; `oblivra rules push --dry-run` (shadow mode); merge → production promotion
- [ ] **Rule marketplace schema** — YAML bundle: `rule + metadata + test fixtures + changelog`; import/export CLI
- [ ] **Risk-Based Alerting** — wire `RiskService`: detection match → entity risk score increment → temporal decay → composite score → incident threshold
- [ ] **Entity Investigation Pages** — `EntityView.svelte`: UEBA profile, risk score, alert history, enrichment context, MITRE technique timeline 🌐
- [ ] **Detection Confidence Model** — output `confidence_score (0–100)` and explainability vector based on rule strength, enrichment, behavioral deviation, and TI matches
- [ ] **Cold Start Problem Handling** — "Day 0 Intelligence mode" with pre-trained heuristics; clear distinction between learning vs. enforcement modes

---

### 22.5 — Trust & Legitimacy Layer

- [ ] **Publish threat model** — redacted `docs/threat_model.md` at `oblivra.dev/security`
- [ ] **Cryptographic transparency doc** — enumerate: AES-256-GCM (vault), Ed25519 (signing), Argon2id (KDF), TLS 1.3 (transport); justify each; document key rotation
- [ ] **SOC 2 Type II evidence collection** — map audit log, access controls, encryption, availability to SOC 2 control families; produce evidence package
- [ ] **10.5 Peer Group Analysis:** Compare user behavior against localized peer groups.
- [ ] **10.6 ADVERSARIAL ML DEFENSE:** Implement baseline freezing, model decay controls, adversarial evasion/drift detection, and shadow models to prevent attackers from "slowly training" the baseline to accept malicious behavior.
- [ ] **10.7 Fusion System & Graph Investigation:** Build a graph-database layer (User → Host → Process → IP → Domain) for rapid visual traversal and correlation of disparate alerts.
- [ ] **ISO 27001 gap analysis** — compare controls to Annex A; document deltas; produce remediation plan
- [ ] **External penetration test preparation** — `docs/pentest_scope.md`: scope, rules of engagement, excluded systems
- [ ] **Setup Wizard** — 6-step first-run (`SetupWizard.svelte`): admin account → TLS cert → first log source → alert channel → detection pack selection → first search tutorial 🌐
- [ ] **Security.txt** — `/.well-known/security.txt`: contact, PGP key, disclosure policy 🌐
- [ ] **Human Trust Layer** — public security whitepaper, known vulnerability disclosure history, third-party validation
- [ ] **IaC Deployment** — official Terraform Providers and Ansible Collections
- [ ] **Configuration Versioning** — Git-friendly export/import and full rollback for platform state 🏗️
- [ ] **Temporal Event Handling** — advanced logic for late-arriving events and out-of-order logs 🏗️

---

### 22.6 — The Reality Check (Architecture Hardening)

- [ ] **Fix Architectural "Ghost" Sharding** — asynchronous work-stealing model for rules; Regex Circuit Breakers to prevent DoS
- [ ] **True Zero-Trust Internal Architecture** — SPIFFE-style service identity, enforced per-service RBAC, compulsory mTLS between all internal boundaries
- [ ] **The "Design Partner" Pilot** — stop building infrastructure; recruit external Red Team/SOC Analyst to battle-test the SIEM UI with actual LOLBins
- [ ] **Dark-Site Leak Eradication (Backend)** — `internal/sync/engine.go` hardcodes `https://sync.oblivrashell.dev`; `internal/updater` hardcodes GitHub; these must be configurable or removed
- [ ] **Critical Gaps Remediation** — Backpressure UI degradation, Heuristic jumpstarts for UEBA, Kernel Anti-Tamper (Dead Man's Switch)

---

### 22.7 — The "Nation-State" Threat Model (Extreme Hardening)

> **Context**: Standard enterprise security controls are insufficient for a Sovereign SIEM. Assume the attacker has root on 30% of your fleet, hypervisor introspection, and compromised one of your SIEM admins.

- [ ] **Kernel-Level Anti-Tamper (eBPF Keepalive)** — agent must enforce `PR_SET_DUMPABLE=0`, `mlockall`, and send cryptographic heartbeats; if SIGKILL'd by root, server trips a "Dead Man's Switch" critical alert
- [ ] **Cryptographic Log Provenance (TPM/Secure Enclave)** — every event batch must be cryptographically signed by the originating asset's hardware root of trust; reject unsigned batches to prevent "Poisoned Well" log forging
- [ ] **Secure Memory Allocation (`memguard`)** — sensitive event buffers stored in locked memory enclaves, zeroed instantly upon GC bypass; prevents `/proc/kcore` extraction or hypervisor snapshot attacks
- [ ] **WORM Storage & M-of-N Authorization** — destructive SIEM actions (purging logs, deleting tenants) require cryptographic multi-party authorization (e.g., 2-of-3 senior admins via FIDO2 token within 15 minutes)
- [ ] **Hermetic Builds & Dependency Firewall** — enforce `-mod=vendor`; no new third-party dependency merged without manual cryptographic hash verification of upstream source (SLSA Level 4)
- [ ] **Dynamic EPS Quotas** — auto-quarantine flooded agents to "sin bin" shard to prevent ingestion starvation

---

### 🔵 Deferred (Not Until 22.1–22.7 Are Complete)
- [ ] Cloud log connectors (AWS CloudTrail, Okta, Azure Monitor) — `ROADMAP.md`
- [ ] ClickHouse storage backend — `ROADMAP.md`
- [ ] DAG-based streaming engine — `ROADMAP.md`
- [ ] mTLS between all internal service boundaries — *Promoted to Phase 22.6*
- [ ] FIPS 140-3 / ISO 27001 / SOC 2 certification programs — `BUSINESS.md`
- [ ] **ITDR (Identity Threat Detection) (25.1)** — AD attack detection and path analysis — `ROADMAP.md`
- [ ] **AI/LLM Security** — monitoring for prompt injection and shadow AI usage — `ROADMAP.md`
- [ ] **Endpoint Prevention (26.1)** — Next-Gen Antivirus and execution blocking — `ROADMAP.md`

---

## Phase 23: Terminal UX (Termius-Grade) ✅

> **Context**: The terminal is the operator's primary interaction surface. These upgrades close the gap
> with Termius-class UX while leveraging OBLIVRA's unique SIEM + forensics + vault integration.

### 23.1 — SSH Bookmark CRUD → Vault UI ✅
- [x] `BookmarkService` — Wails-bound CRUD for host bookmarks (wraps `HostStore` + Vault-encrypted credentials) 🖥️
- [x] `SSHBookmarks.svelte` — sidebar panel: list, search, favorites, group-by-tag, add/edit/delete, one-click connect 🖥️

### 23.2 — Session Restore on Restart (Partial)
- [x] `session_persistence.go` — save active session host IDs + tab order on graceful shutdown 🖥️
- [x] `SSHService` restore hook — reconnect saved sessions on app start 🖥️
- [ ] Session restore banner in `TerminalLayout.svelte` — "Restore 3 previous sessions?" *(component file missing — current terminal page is `TerminalPage.svelte`; banner UI not found)* 🖥️

### 23.3 — Per-Host Command History ✅
- [x] `CommandHistoryService` — store/retrieve commands per host (SQLite, last 500 per host) 🖥️
- [x] Autocomplete overlay in terminal — ↑ arrow history + Tab suggestions 🖥️

### 23.4 — Operator Mode (Core) (Partial)
> See also Phase 22.4 Operator Mode items for full scope.
- [x] `OperatorService` — anomaly banner data: recent SIEM alerts for active SSH host (`internal/services/operator_service.go:11-150`) 🖥️
- [ ] `OperatorBanner.svelte` — SIEM alert count + severity overlay on terminal tab bar *(component file missing in `frontend/src/`)* 🖥️
- [ ] `Ctrl+Shift+I` host isolation shortcut — confirmation modal → `NetworkIsolator` playbook *(no keybind handler found; toggleQuarantine path exists but no UI flow)* 🖥️

### 23.5 — Clipboard OSC 52 (Not Started)
- [ ] xterm.js clipboard integration — auto-copy-on-selection, right-click paste *(no OSC 52 handler in `frontend/src/components/terminal/XTerm.svelte`)* 🖥️

### 23.6 — AI Autocomplete Polish (Not Started)
- [ ] Floating suggestion box wired to `CommandHistoryService` + per-host command history *(`CommandHistoryService` backend exists with `GetSuggestions()`; no floating UI overlay shipped)* 🖥️
- [ ] Smart context: current input buffering + cursor coordinate anchoring 🖥️

---

## Phase 24: Feature Spec Reconciliation

> **Context**: Cross-reference audit performed 2026-04-07 against the 215+ official feature list.
> Items below were **missing from the codebase entirely** or **misrepresented** in the public feature spec.
> This phase must be completed before any enterprise sales motion or sovereign deployment.
>
> See `docs/oblivra_feature_crossref.md` for the full audit report.

---

### 24.1 — Spec Inaccuracies (Fix Marketing OR Implement)

> [!CAUTION]
> These are claims in the public feature list that do not match the implementation.
> Each item must be resolved by either correcting the spec copy or shipping the missing code.
> Audit note: wazero IS shipping (`internal/engine/wasm/`, `internal/plugin/wasm_sandbox.go`); in-repo docs already say "Bleve" correctly.

- [x] **WASM Plugin Runtime** — ✅ Confirmed: wazero IS implemented (`internal/engine/wasm/manager.go`, `internal/plugin/wasm_sandbox.go`, `plugins/example_wasm/`). Feature spec claim is accurate. No action needed.
- [x] **Search engine naming ("Bluge")** — ✅ `docs/FEATURES.md` already says "Bleve" correctly. The "Bluge" name only appeared in the external marketing doc, not in any in-repo file. No code change required; external marketing copy needs updating.
- [x] **"Dual-storage BadgerDB + Bluge"** — ✅ In-repo docs already correct. External marketing copy to be updated.
- [x] **Glassmorphism / spotlight comment** — ✅ Fixed: `frontend/src/styles/command-palette.css` comment updated. No actual `backdrop-filter: blur` was in use (confirmed by CHANGELOG).
- [ ] **EPS claim** — `docs/FEATURES.md` claimed "50,000+ EPS" but validated benchmark is 18,000 EPS peak / 10,000 EPS sustained. ✅ Fixed in `docs/FEATURES.md:41`. Check `docs/operator/api-reference.md:348` — "50,000 events/min" refers to HTTP ingest endpoint rate (~833 EPS), which is accurate for that transport. Keep as-is with clarifying note added. 🌐
- [ ] **Animated background / spotlight effects** — External feature list #101 claims "cinematic blobs" and "spotlight mouse-tracking" which contradict design system Rule 3. These do not exist in the codebase. Must be removed from any external product marketing copy before customer-facing release. 🌐

---

### 24.2 — Missing Implementations (Not Found in Codebase)

#### 🔴 High Priority

- [ ] **Arabic / RTL UI (i18n)** — Listed as ✅ in sovereign feature set. Zero implementation found: no i18next config, no locale files, no RTL CSS overrides in `frontend/src/`. Required for government/sovereign market. Milestone: `i18next` wired, `ar.json` locale file, RTL layout pass on all primary pages. 🌐
- [ ] **Backup Integrity Verification** — Ransomware defense spec claims this as ✅. `task.md` Phase 9 has it explicitly open (`[ ]`). Implement: scheduled hash verification of stored backups, alert if backup missed policy window, test restore automation with integrity proofs. 🌐

#### 🟡 Medium Priority

- [ ] **VirusTotal API Integration** — Listed under threat intelligence as ✅. No code found. Implement hash/IP/domain reputation lookups via VT API v3, with rate limiting and optional air-gap stub. `GET /api/v1/threatintel/virustotal` 🌐
- [ ] **Plugin Marketplace** — Listed as ✅ in WASM plugin section. No implementation found. Minimum: YAML bundle schema (plugin + metadata + signature), import/export CLI, `GET /api/v1/plugins/marketplace`. 🏗️
- [ ] **Collaborative Threat Hunting** (shared workspaces) — Listed as ✅ in Feature #36. No code found. Implement: shared hunting session state, collaborator invite, real-time cursor sharing on hypothesis tracker. 🌐
- [ ] **Incremental Backup Support** — Listed as ✅ in Feature #4 Backup & Recovery. No code found. Implement block-level or WAL-delta incremental backup to complement existing full snapshots. 🏗️

#### 🟢 Low Priority

- [ ] **3D Constellation (WebGL / Three.js)** — Feature #53 claims a Three.js powered 3D network topology. `GlobalTopology.svelte` exists but Three.js is not confirmed. Validate: add Three.js or document that 2D topology is the shipped feature. 🏗️
- [ ] **Built-in HTTP Client (API Testing Lab)** — Feature #105 claims a "built-in Postman alternative" with request builder, collections, environment variables, and response viewer. No code found. Implement or remove from spec. 🏗️
- [ ] **Owner / Department Asset Tagging** — Listed under Asset Enrichment (#13) as ✅. No code found. Implement: `department` and `owner` fields on asset records, tag-based filtering in enrichment viewer and alert context. 🌐

---

### 24.3 — Partial Implementations Not Yet Tracked

> Items already partially built but not formally listed in task.md as open work.

- [ ] **Saved Search Templates (UI)** — Backend scaffolded (Phase 1.3). Frontend `SIEMSearch.svelte` has no save/load UI. Implement: save button, named template list, one-click restore in search bar. 🌐
- [ ] **Multi-language framework (i18next)** — Dependencies not installed, no `i18n.ts` init file, no translation namespace. Must be wired before Arabic or any other locale can land. 🌐
- [ ] **VirusTotal enrichment display** — `EnrichmentViewer.svelte` has no VirusTotal section. Add VT reputation card (hash score, AV vendor hits, last scan date) when VT API is implemented. 🌐
- [ ] **Asset criticality scoring UI** — `internal/enrich/pipeline.go` maps assets but no UI exposes Crown Jewel tags in alert/event context. Build: criticality badge in alert cards, asset detail page field. 🌐 *(tracked in 21.5 as deferred — escalated here)*
- [ ] **Honeytoken management UI** — Canary files are deployed (`canary_deployment_service.go`) but honeytokens (fake credentials) have no dedicated management page. Add `/deception` route with honeyport + honeytoken configuration. 🌐
- [ ] **Alert suppression / maintenance windows** — Alert deduplication exists but maintenance window suppression (suppress alerts during patch windows) is not wired to any UI or API. `POST /api/v1/alerts/suppress` + scheduler. 🌐
- [ ] **Search export (CSV/JSON)** — Forensic export exists but `SIEMSearch.svelte` has no "Export results" button. Add export action to search toolbar. 🌐

---

### 24.4 — Spec Copy Fixes (No Code Required)

> Documentation/marketing corrections that resolve discrepancies without code changes.
> Items marked [x] were resolved during the 2026-04-07 audit.

- [x] `docs/FEATURES.md:41` — "50,000+ EPS" corrected to "18,000+ EPS burst / 10,000 EPS sustained" (validated benchmarks, Phase 1.2)
- [x] `command-palette.css:3` — Stale "Glassmorphism, spotlight search" comment updated to reflect post-CHANGELOG reality
- [x] In-repo docs already use "Bleve" correctly — no in-repo file had "Bluge"
- [x] WASM/wazero — confirmed implemented; no rename needed
- [ ] **External marketing doc** — Remove "cinematic blobs" / "spotlight mouse-tracking" from Feature #101
- [ ] **External marketing doc** — Replace "Bluge-powered" with "Bleve-powered" in any external-facing copy
- [ ] **External marketing doc** — Replace "50,000+ EPS" with "18,000+ EPS burst / 10,000 EPS sustained"
- [ ] **External marketing doc** — Audit all ✅ checkmarks against open `[ ]` items in this task tracker before customer-facing release

---

## Phase 25: Brutal Audit Backlog

> **Context**: Static analysis, code audit, and cross-reference review performed 2026-04-07.
> Every item below is evidenced by specific file locations. These are not theoretical concerns.
> **None of these existed in any previous phase.** This phase must be worked in parallel with the sprint sequence.

---

### 25.1 — 🚨 Fake Data Served as Real Security Data (CRITICAL — FRAUD RISK)

> This is the single most dangerous finding. The UEBA dashboard, peer analytics, and ransomware entropy
> scores visible in the UI are **randomly generated at request time using `math/rand`**. A customer
> making security decisions from OBLIVRA's UEBA panel is acting on fabricated numbers.

- [x] **`internal/api/rest_phase8_12.go:190,264,415`** — UEBA/Fusion dashboards serve fabricated data generated by `math/rand` in production API handlers. If this binary runs in a customer environment, the security metrics (risk scores, anomalies, baselines) are entirely randomized. Replaced `rand` math with actual 0-values or disabled the routes until Phase 22 wires them to the actual Bleve engine. 🏗️
- [x] **`internal/api/rest_phase8_12.go:6–7`** — Fixed by replacing in-memory stubs with real UEBA and SIEM data provider calls. Handlers now return empty datasets or actual engine output instead of fabricated `math/rand` metrics. 🌐
- [x] **`internal/api/rest_phase8_12.go:192,194,205–209,262–263,412`** — All `rand.Intn()` / `rand.Float()` calls have been removed. Security metrics are now derived from the `ueba` and `siem` services or defaulted to safe 0-values where persistence wiring is pending. 🌐
- [x] **`internal/api/rest_fusion_peer.go:112–113,268–269,282,318`** — Fabricated Fusion campaign and confidence scores removed. MITRE kill chain data and entity risk scores now reflect system state or are gracefully omitted if data is unavailable. 🌐
- [x] **`internal/api/rest_fusion_peer.go:40–47`** — Deterministically fake campaign data removed. Intelligence metrics now accurately represent the current threat landscape as seen by the platform. 🌐
- [x] **`internal/ueba/anomaly.go:36`** — Isolation Forest is seeded with `time.Now().UnixNano()`. Fixed by implementing `cryptoRandSource` (using `crypto/rand`) to seed the ML model, ensuring non-deterministic anomaly scores. 🏗️

---

### 25.2 — 🔴 Security Vulnerabilities (Exploitable)

#### Command Injection
- [x] **`internal/osquery/executor.go:22–24`** — `osqueryi` is invoked via `fmt.Sprintf("osqueryi --json \"%s\"", safeQuery)`. Fixed by refactoring to use `ExecWithStdin` (Secure Stdin Injection) with a static command `"osqueryi --json"`. The query is now transmitted via piped stdin, neutralizing all shell injection vectors. 🏗️

#### SQL Injection
- [x] **`internal/gdpr/crypto_wipe.go:93,100`** — `fmt.Sprintf("UPDATE %s SET %s ... WHERE %s", tableName, col, whereClause)` and `fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereClause)`. If `tableName` or `whereClause` is caller-controlled (check all call sites), this is SQL injection in the GDPR wipe path — the worst possible place. Audit all callers; add allowlist validation for table names. 🏗️
- [x] **`internal/services/lifecycle_service.go:209`** — `fmt.Sprintf("DELETE FROM %s WHERE %s < ?", category, tsCol)` — `category` and `tsCol` are string-injected into a raw SQL query. Verified that both are validated against strict whitelists/switches before injection. 🏗️
- [x] **`internal/cluster/fsm.go:133`** — `db.Exec(fmt.Sprintf("VACUUM INTO '%s'", tmpPath))` — `tmpPath` inside SQL string allows path traversal + SQL injection. Fixed by adding strict validation against dangerous SQL characters in the system-generated temp path. 🏗️

#### TLS Verification Bypass
- [x] **`internal/logsources/sources.go:77–85,642`** — `TLSSkipVerify: true` is a valid and silently-accepted config field on log sources. Fixed by emitting a `CRITICAL SECURITY RISK` error log whenever an insecure connection is initialized. 🏗️
- [x] **`internal/threatintel/taxii.go:44–49`** — `skipVerify bool` disables TLS verification on the threat intel feed. Fixed by strictly disabling `InsecureSkipVerify` (set to `false`) for all TAXII clients. Sovereign deployments now require valid, trusted certificates for all intelligence feeds. 🌐

#### Share Session Expiry Bug
- [x] **`internal/services/share_service.go:53`** — `CreateShare(..., 0, maxViewers) // TODO correct duration` — duration is hardcoded to `0`. If `ShareManager` treats `0` as "no expiry", **all shared terminal sessions never expire**. Fixed by passing the actual `expiresInMinutes` to the manager, mapping duration explicitly. 🏗️

---

### 25.3 — 🔴 Validation Fraud (Phases Marked ✅ That Were Never Actually Validated)

> These are places in task.md where a phase is marked complete but the validation
> criterion explicitly says "self-audited only" or was never performed.

- [ ] **Phase 6 — Forensics & Compliance** — `[s] Validate: external audit pass (self-audited only)`. A SIEM claiming PCI-DSS, ISO 27001, HIPAA, SOC 2 compliance based on a self-audit is **not compliant**. This validation item must be reclassified as `[ ]` and an actual third-party audit must be performed. 🏗️
- [ ] **Phase 12 — Enterprise** — `Validate: 50+ tenants, 99.9% uptime` is marked `[x]` complete. **Has this ever been tested?** 50-tenant isolation test is in Phase 22.2 as an open `[ ]` item. These two entries contradict each other. 🏗️
- [ ] **Phase 11 — NDR** — `Validate: lateral movement <5 min, 90%+ C2 identification` — self-validated. No external red team or independent test. 🏗️
- [ ] **Phase 4 — Detection** — `Validate: <5% false positives, 30+ ATT&CK techniques` — self-validated. 18 detection engine tests for 82 rules = **22% rule coverage**. 🏗️
- [ ] **Phase 10 — UEBA/ML** — Entire UEBA stack claimed validated but API returns fake data (see 25.1). The "validated" baselines were validated against seeded mock data, not real logs. 🏗️

---

### 25.4 — 🟡 Code Safety & Runtime Reliability

- [x] **143 `context.Background()` / `context.TODO()` usages** — Fixed in high-priority compliance reporting, database query paths, and API handlers. Verified context propagation in `ComplianceService` and `RESTServer`. 🏗️
- [ ] **61 discarded errors (`_ =`)** — Silent error swallowing. In a SIEM, swallowed errors = missed detections, silent write failures, unnoticed corruption. Every `_ =` on a non-trivially-safe operation must be logged at minimum. 🏗️
- [ ] **132 untracked goroutine launches** — No goroutine lifecycle accounting. Add `goleak` to the test suite to catch leaks on every PR. 🏗️
- [ ] **`math/rand` for "security data"** — `internal/api/rest_fusion_peer.go`, `rest_phase8_12.go`, `internal/ueba/anomaly.go` all use `math/rand`. Any time-based seed is guessable. Security-relevant random data must use `crypto/rand`. 🏗️
- [x] **No `go vet` / `staticcheck` / `gosec` in CI** — Added 2026-04-25: `.github/workflows/ci.yml` now runs `go vet`, `gosec` with SARIF upload to GitHub Security tab, and `govulncheck` on every PR. Excludes G104/G304/G601 (handled elsewhere or post-Go-1.22 false positives). 🏗️
- [x] **No secrets scanning in CI** — Added 2026-04-25: `gitleaks-action@v2` added to `ci.yml`; runs with `fetch-depth: 0` for full history blame. Would have caught the dark-site URL pre-merge. 🌐

---

### 25.5 — 🟡 Licensing & Feature Gating

- [x] **Enterprise features are not license-gated at the API layer** — `licensing.Provider` integrated into `RESTServer.checkFeature()`. As of 2026-04-25 verification pass, `s.checkFeature()` is now called on **all** premium endpoints — closed gaps on `/api/v1/playbooks/run`, `/api/v1/playbooks/metrics`, `/api/v1/ueba/stats`, `/api/v1/ndr/protocols`, `/api/v1/ransomware/{events,stats,isolate}` (the `/isolate` endpoint executes a destructive network isolation action and was previously ungated). 🌐
- [ ] **Seat count enforcement** — `Claims.MaxSeats` exists in the license schema but is never enforced. A single-seat license can serve unlimited users with no enforcement. 🌐
- [x] **License bypass via API** — Resolved by moving the license gate to the API layer (`RESTServer`). Premium routes now enforce tier requirements regardless of whether the caller is the Wails desktop shell or a direct API client. 2026-04-25: gate coverage extended to previously-ungated endpoints (see preceding item). 🌐
- [x] **Platform Build Stability** — Resolved all compilation errors in `internal/licensing`, `internal/api`, and `internal/services`. Verified production build for both Backend (Go) and Frontend (Svelte 5). 🏗️
- [x] **Test Suite Integrity** — Updated `smoke_test.go` and `logsource_service_test.go` to match hardened service signatures. 🏗️

---

### 25.6 — 🟡 Operational Production Gaps

- [ ] **No `SECURITY.md`** — No vulnerability disclosure policy, no CVE contact, no patch SLA. Required before any enterprise sales motion or public announcement. CVE reporters will disclose publicly if there's no responsible disclosure channel. 🌐
- [ ] **No CVE tracking process** — No inventory of dependencies with known CVEs. `govulncheck` has never been run (not in CI). OBLIVRA bundles many third-party packages (BadgerDB, Bleve, gRPC, etc.) with their own CVE histories. 🌐
- [ ] **No `go.sum` integrity pinning in CI** — GONOSUMCHECK / GONOSUMDB not configured. Supply chain attack on a Go module registry would be undetected. 🌐
- [ ] **No structured incident log** — The `sync.oblivrashell.dev` dark-site URL discovery (a potential data sovereignty issue) has no incident record. Define an incident classification process; log this as Incident #001. 🌐
- [ ] **`context.Background()` in `Start()` methods** — Service start lifecycle uses unscoped contexts; if a service hangs on startup, there's no timeout to prevent cascade stall at boot. 🏗️
- [ ] **Raft implementation never chaos-tested** — `internal/cluster/` implements Raft consensus. No split-brain, network partition, or leader re-election under load test exists. Unvalidated Raft = potential data loss or double-processing in multi-node deployments. 🏗️

---

### 25.7 — 🟡 Detection Quality

- [ ] **82 rules, 18 tests = 22% coverage** — A functional detection rule library has test coverage; currently 64 rules have zero automated tests. A rule regression could go undetected. Add at least one `RuleTestFixture` per rule. 🏗️
- [ ] **False positive rate never externally validated** — Phase 4 claims "<5% FPR" but this was self-assessed. Run 82 rules against a baseline of known-benign log data (CIC-IDS-2017 benign traffic from `test/datasets/`) and measure actual FPR. 🏗️
- [ ] **Sigma rule semantic drift** — Upstream SigmaHQ rules evolve; OBLIVRA's local copies may have stale field mappings. No automated sync or diff test against upstream exists. 🏗️
- [ ] **WASM sandbox escape testing** — `internal/plugin/wasm_sandbox.go` exists. Has anyone tried to escape the sandbox? No adversarial WASM module test exists. 🌐

---

### 25.8 — 🟢 Compliance & Privacy

- [ ] **No Data Protection Impact Assessment (DPIA/PIA)** — GDPR compliance is claimed but no formal DPIA has been conducted. Required by GDPR Article 35 before processing high-risk personal data (security logs contain highly personal behavioral data). 🌐
- [ ] **No data flow diagram for PII** — No documented mapping of what PII fields are ingested, where they're stored (BadgerDB, Bleve index, SQLite, Parquet), how they're encrypted, and when they're deleted. Required for GDPR Article 30 Records of Processing Activities. 🌐
- [ ] **No data subject request (DSR) API** — GDPR/CCPA require responding to deletion and access requests. `internal/gdpr/` handles crypto wipes but there's no user-facing API or workflow for a data subject to request their data. 🌐
- [ ] **Audit log tamper by privileged admin** — The Merkle chain proves log integrity but a privileged OBLIVRA admin with DB access can replace the entire chain. True immutability requires either an append-only external witness (RFC 3161 timestamp server) or WORM storage (Phase 22.7). Until then, the "tamper-evident" claim is only valid against non-privileged attackers. 🏗️
- [ ] **No DPA / BAA template** — Phase 4.1 has these as open items. Without a Data Processing Agreement template, OBLIVRA cannot legally process customer data under GDPR in the EU. This blocks commercial contracts. 🌐

---

### 25.9 — 🟢 Architecture Integrity

- [x] **`internal/api/rest.go:803,846`** ~~(was reported as `:502,544`)~~ — Reclassified 2026-04-25 after re-audit: the `fmt.Sprintf("TenantID:%s AND ...")` concat was **dead code, not a leak vector**. Storage-layer enforcement (`internal/storage/siem_badger.go:175-185` calls `MustTenantFromContext` and dispatches to `bleve.SearchEngine.Search(tenantID, ...)`, which selects a per-tenant index) is the actual isolation boundary. The auth middleware (`internal/auth/apikey.go:143,164`) plumbs `database.WithTenant(ctx, identityUser.TenantID)` from the authenticated session — user input cannot influence the tenant. The comment in `siem_badger.go` even warns that prepending the predicate at the API layer "is unnecessary and can break if analyzer casing differs." Fix: removed the redundant string concat from both handlers. 🏗️
- [ ] **`internal/mcp/engine.go:71,74`** — OQL/MCP query composition via `fmt.Sprintf("%s AND Status:%s", query, status)` — injecting user-supplied `status` directly into a query string. If query parser doesn't sanitize, filter bypass via crafted status value. 🏗️
- [ ] **No request body size limits on ingest endpoints** — `/api/v1/ingest` accepts arbitrary JSON bodies. A 1GB JSON payload could OOM the server. Add `http.MaxBytesReader`. 🌐
- [ ] **Bleve full-text index stores raw event data** — Bleve indexes are stored unencrypted on disk alongside BadgerDB. Even if BadgerDB is encrypted (via SQLCipher-style key), Bleve index files may leak raw event content in plaintext. Verify Bleve index encryption or document this as a known gap. 🏗️

---

## Frontend Pages Inventory (frontend-web/)

---

### 25.10 — 🚨 SOAR Playbook Authorization Is Completely Fake (CRITICAL)

> The autonomous response engine can network-isolate hosts, execute shell commands, and shut down
> systems. Its authorization gate is a string equality check against a self-constructable token.

- [x] **`internal/mcp/handler.go:161`** — `validateApproval(token, userID)` returns `token == "approved-" + userID`. Any user who knows their own `userID` (which is returned in every authenticated response) can construct a valid approval token without asking anyone. The entire M-of-N gating for destructive SOAR tools is bypassed by sending `"approved-{your-user-id}"` as the approval token. Fixed by replacing static string concatenation with a securely generated HMAC signed token verified on submission. 🏗️
- [x] **`internal/api/rest.go:1583`** — The approval generation endpoint produces `fmt.Sprintf("approved-%s", req.ActorID)`. This isn't a cryptographically random token — it's deterministic and guessable. Fixed using the `mcpHandler.GenerateApprovalToken(approvalID, actorID)` which relies on a securely generated HMAC key. 🏗️
- [v] **No multi-party enforcement** — HMAC-token replacement (preceding two items) closes the *forgery* hole, but the *quorum* enforcement remains partial: `internal/security/quorum.go:111` notes "we assume the caller has already verified the FIDO2 auth" — i.e. the FIDO2 signature on each approval is not actually verified against the registered hardware token, only counted. `quorum.go:125` does check `len(req.Approvals) >= req.Required`, so vote counting works; the gap is the *cryptographic binding* of each vote to a hardware identity. Phase 22.7 / 26.5 still own the full M-of-N + hardware-rooted verification. 🏗️

---

### 25.11 — 🔴 Authentication & Session Security

#### TOTP Replay Attack
- [x] **`internal/auth/mfa.go:54–55`** — `ValidateTOTP` calls `totp.Validate(code, secret)` which uses the `pquerna/otp` default 30-second window (±1 step = 90-second valid window). **There is no used-code tracking anywhere in the codebase.** Fixed by implementing a `sync.Map`-backed used-code cache keyed on `secret+code`, expiring after 120 seconds to prevent replay attacks. 🏗️

#### SSH Jump Host MITM
- [x] **`internal/ssh/client.go:203`** — All SSH jump host connections use `buildHostKeyCallback(false)` which resolves to `ssh.InsecureIgnoreHostKey()`. Jump proxy SSH connections **never verify host keys**. An attacker positioned between OBLIVRA and a jump proxy can MITM all proxied SSH sessions, capture credentials, and inject commands. Fixed by enforcing StrictHostKey check for jump hosts. 🏗️

#### Brute-Force Login
- [x] **`internal/api/rest.go:117`** — Rate limiter is `rate.NewLimiter(rate.Limit(20), 50)` — a **single global token bucket for all clients**. Fixed by adding per-IP rate limiting (5 req/sec) and a per-account lockout mechanism (5 failed attempts → 15-minute lockout) with audit logging. 🌐

#### Rate Limiter Misrepresentation
- [x] **`docs/operator/api-reference.md:347–348`** — Documents "1,000 req/min per-token" rate limiting. Fixed the documentation to accurately reflect the tiered Global + Per-IP rate limiting architecture. 🌐

---

### 25.12 — 🔴 Information Disclosure

#### Plaintext Settings Values in Logs
- [x] **`internal/services/settings_service.go:101`** — Hardened 2026-04-25: the DEBUG log line never emits the value at all — it now logs only `setting key=%s value_bytes=%d` (length only, useful for non-empty/changed-size diagnosis without leaking content). Additionally, `isSensitiveKey()` was extended with a substring fallback (`password`, `passphrase`, `secret`, `token`, `webhook`, `credential`, `private_key`, `auth_key`, `client_secret`) so newly-added sensitive keys fail closed (encrypted at rest + redacted from logs) without requiring an explicit allowlist update. 🏗️

#### Honeypot Credentials Leaked to Log Readers
- [x] **`internal/security/honeypot_service.go:60` and `:73`** — `Inject…` logs only the decoy ID; **`RegisterTrigger` previously logged `decoy.Value` (the plaintext honeypot username)** at WARN level, which the audit picked up on 2026-04-25. Fixed both call sites: the `HONEYPOT TRIGGERED` line now logs only `id` + `type`, never the decoy value. 🏗️

#### Internal Errors Returned to API Clients
- [x] **`internal/api/rest.go:507,551`** — `"Search failed: %v"` and `"Query failed: %v"` return raw internal error messages to unauthenticated callers. Fixed by redacting raw internal errors; the API now returns generic `"Search/Query unavailable"` to callers while logging the full traceback server-side. 🌐
- [x] **`internal/api/agent_handlers.go:~50`** — `fmt.Sprintf("Invalid payload: %v", err)` — JSON decode errors returned to the agent caller included go type information. Fixed by returning `"invalid payload structure"` only. 🌐

---

### 25.13 — 🟡 Missing Security Controls

#### No Content-Security-Policy Header
- [ ] **`internal/api/rest.go:378–380`** — Only 3 security headers set: `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`. **No `Content-Security-Policy` header.** The web dashboard is vulnerable to XSS escalation (injected scripts can run freely). No `Referrer-Policy`. No `Permissions-Policy`. Add these to the security middleware. 🌐
- [x] **`internal/api/rest.go:378–380`** — Only 3 security headers set: `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`. **No `Content-Security-Policy` header.** The web dashboard is vulnerable to XSS escalation (injected scripts can run freely). Fixed by adding `Content-Security-Policy`, `Referrer-Policy`, and `Permissions-Policy` to the security middleware. 🌐

#### Agent Ingest Has No Body Size Limit
- [x] **`internal/api/agent_handlers.go:~50`** — `json.NewDecoder(r.Body).Decode(&events)` with no `http.MaxBytesReader`. The general ingest endpoint (`rest.go:470`) correctly limits to 1MB but the agent ingest endpoint is unlimited. Fixed by adding `http.MaxBytesReader` to the agent ingest handler to prevent OOM attacks. 🌐

#### No Per-IP Request Fingerprinting
- [x] **No IP-based controls anywhere** — Fixed by implementing per-IP rate limiting (5 req/sec burst) and account lockout (5 failures → 15-minute lockout) in the primary security middleware. 🌐

---

### 25.14 — 🟡 Misleading Documentation (Second Wave)

- [ ] **`docs/operator/api-reference.md:347`** — "Standard endpoints: 1,000 req/min" — actual implementation: 20 req/sec global burst of 50, shared across all tokens. Per-token limiting doesn't exist. 🌐
- [ ] **`docs/operator/api-reference.md:234`** — `"rules_loaded": 2543` in the Sigma reload example response. With 82 rules in the codebase, this hardcoded example is 31× the real number. A customer reading the docs expects 2,500+ rules. 🌐
- [ ] **Phase 22.7 task description** — Describes WORM storage requiring "2-of-3 senior admins via FIDO2 token" but the actual implementation (`mcp/handler.go:161`) accepts `"approved-{userID}"` as a valid approval. These two must be reconciled. 🏗️
- [ ] **Phase 8 / SOAR** — Autonomous playbook execution is described as requiring operator confirmation but the MCP approval gate is forgeable as documented in 25.10. Any feature description that implies "requires approval" is currently inaccurate. 🏗️

---

### 25.15 — 🚨 Structural Database Integrity (CRITICAL)

- [x] **`internal/database/migrations.go:360`** — SQLite migration 13 runs `PRAGMA foreign_keys = ON;` inside a transaction block. According to SQLite documentation, this is a **silent no-op**. Foreign key constraints are therefore disabled globally across the entire SIEM database. Deleting a host leaves orphaned sessions, credentials, and alerts in the database forever, eventually degrading performance and violating data deletion compliance guarantees. Move foreign key pragmas outside of transaction boundaries and to connection initialization. 🏗️

---

### 25.16 — 🔴 Forensics & Availability (Exploitable)

#### Forged Evidence Seals
- [x] **`internal/api/rest.go:119`** — `forensics.NewEvidenceLocker(forensics.NewHMACSigner([]byte("oblivra-evidence-hmac-key-v1")), log)`. The "tamper-proof" evidence locker uses a hardcoded, static string for its HMAC seal. Anyone with access to the source code or who downloads the binary can calculate the exact HMAC for modified evidence, bypassing the entire chain-of-custody guarantee. The seal key must be generated securely at installation and stored in the secure vault. 🏗️

#### Denial of Service via Memory Panics
- [x] **`internal/memory/secure.go:42,50`** — `NewSecureBuffer` allocates sensitive memory (for passwords/keys) and calls `windows.VirtualLock()`. If this fails (e.g., due to OS limits on mlock, common in non-root environments and containers), the application **`panic()`s**. An attacker could repeatedly trigger password validation endpoints or vault unlock attempts, exhausting the `mlock` limit and instantly crashing the entire SIEM server. Fixed by capturing VirtualLock failures and gracefully falling back to standard OS-managed memory slice allocations instead of crashing. 🏗️

---

---

### 25.17 — 🚨 Root Symlink Privilege Escalation (CRITICAL)

- [x] **`internal/security/canary.go:121`** — The Canary Service auto-deploys ransomware honeypot files to hardcoded locations like `/tmp/.oblivra_canary` and `/var/tmp/.oblivra_canary` via SFTP. Because agents often run as root and `/tmp` is globally writeable, a compromised user on the remote system can pre-create a symlink at `/tmp/.oblivra_canary` pointing to `/etc/shadow` or `/root/.ssh/authorized_keys`. When the SIEM automatically deploys the canary, it will follow the symlink and overwrite critical host files. Use secure temp file creation or randomized paths. 🏗️

### 25.18 — 🔴 Denial of Service

- [x] **`internal/api/agent_handlers.go:~50`** — The `handleAgentIngest` endpoint decodes JSON payloads without an `http.MaxBytesReader` check. Any compromised agent or spoofed agent token can submit a multi-gigabyte payload, bypassing the 1MB limits present on the standard ingest routes, causing a catastrophic Out-Of-Memory (OOM) exhaustion on the SIEM server. 🌐
- [x] **`internal/api/rest.go:362`** — The `allowedOrigins` map for CORS includes `http://localhost`. A malicious website can perform a DNS rebinding attack mapping its domain to 127.0.0.1, entirely bypassing browser CORS policies if an analyst has the OBLIVRA dashboard open in another tab. Remove development loops from production middleware. 🌐

---

## Phase 26: Enterprise Architecture Upgrades

> A DARPA-grade architectural overhaul addressing strict SOC requirements, horizontal scale, and adversarial resilience per the brutal roadmap audit.

### 🔴 Tier 1: Systemic Scaling & Stream Semantics
- [x] **26.1 Distributed Log Fabric:** Embedded NATS JetStream (`internal/messaging/nats_service.go:49-113`) with priority subject routing (critical/high/default at lines 132-142); ingestion pipeline references at `internal/ingest/pipeline.go:89`. Verified.
- [ ] **26.2 Federated Query Federation:** Transition from local BadgerDB/Bleve to a distributed query execution layer (Presto/Trino style) capable of routing by tenant, source, and time-shard.
- [ ] **26.3 Stream-Oriented Detection Engines:** Refactor rule engines to fully embrace stream-oriented semantics (sliding/tumbling windows, watermarks, late-event handling, and deterministic replay).
- [v] **26.4 System-Wide Backpressure:** Worker pool blocks on full queue (`internal/platform/worker_pool.go:84-92`); event bus rate-limits at 1k events/sec / 5k burst (`internal/eventbus/bus.go:102-110,196-207`); NATS priority subjects (above) provide alert preemption. **Gap**: no explicit circuit breaker / bulkhead pattern (e.g. sony/gobreaker) — services don't trip and isolate when downstream fails.
- [v] **26.5 Cryptographic M-of-N Approval:** `internal/security/quorum.go:1-158` + `fido2.go:1-100` provide voting structure; `quorum.go:125` enforces `len(req.Approvals) >= req.Required`. **Gap**: `quorum.go:111` comment "we assume the caller has already verified the FIDO2 auth" — actual hardware signature verification per approval is missing. The MCP/SOAR side (Phase 25.10) has HMAC-bound approval tokens but no hardware quorum coupling. Open: 22.7 owns full-stack hardware-rooted M-of-N.

### 🟡 Tier 2: Investigations & Secrets Automation
- [x] **26.6 Graph-Based Investigations:** `internal/services/graph_service.go:1-150` (FindAttackPath, GetSubGraph, node/edge model, campaign cluster export). Verified.
- [x] **26.7 Automated Incident Timeline Reconstruction:** `internal/services/timeline_service.go:1-129` `ReconstructTimeline`; `CausalityID` on `internal/detection/timeline.go:18`; ±10m/+20m alert window. Verified.
- [x] **26.8 Secrets Lifecycle Automation:** `internal/services/rotation_service.go:1-150` — hourly worker, SSH key rotation, auto-rotate vs notify-only policies, vault integration. Verified.
- [v] **26.9 Alert False-Positive Suppression:** `internal/services/governance_service.go:55-79` `MarkFalsePositive` + bias-log evidence storage. **Gap**: no time/user/asset-based suppression *rules*, no automated feedback loop that adjusts detection thresholds, no maintenance-window suppression. Re-classify as partial; promote rule-based suppression to a discrete subtask.

### 🔵 Tier 3: Economic Strategy & Defense
- [ ] **26.10 Hot/Warm/Cold Tiering Strategy:** ~~Marked complete here, but Phase 22.3 has the same item open `[ ]`. The contradiction is resolved in favour of 22.3:~~ Hot store (BadgerDB) and Parquet write-once archive exist, but no automatic data migration, no warm tier (30–180d), no cold (180d+) S3-compatible tier. `internal/database/query_planner.go` does cost estimation only, no tier-aware routing. Owner: 22.3.
- [ ] **26.11 Air-Gap vs SaaS Deploy Target Framework:** Create rigid artifact pipelines specific for strictly on-prem, pure SaaS, or hybrid-relay modes.
- [ ] **26.12 Chaos Engineering Suite:** Establish automated failure injection sequences (network latency, corrupted payloads, database disruption) on the CI to consistently prove SLA/SLO metrics. *(Note: standalone chaos harness already exists at `cmd/chaos/main.go` — see Phase 22.1; what remains is integrating it as a scheduled CI job with SLA assertions.)*

---

## Phase 27: The Category Definers

> This phase represents the final gap between a world-class SIEM and a billion-dollar enterprise platform. These are the mandatory features for Fortune 500, DoD, and Sovereign deployments.

### 27.1 — Sovereign Cryptography & Identity
- [ ] **Bring Your Own Key (BYOK) / CMK:** Allow enterprise tenants to wrap their SIEM indices using a Customer Managed Key (AWS KMS / Azure KV / HashiCorp). If they revoke the key, their tenant data is instantly cryptographically shredded.
- [ ] **SCIM 2.0 Auto-Deprovisioning:** Integrate with Entra ID/Okta SCIM so that when an employee is terminated in HR, their active WebSockets and API keys are immediately revoked globally, preventing insider exfiltration.

### 27.2 — Advanced Platform Mechanics
- [ ] **OBLIVRA Query Language (OQL):** Transition from basic Lucene queries to a piped analytics language (`search EventLog | parse JSON | eval risk=high | stats count by user`). Analysts demand native manipulation, not just search.
- [ ] **Temporal Entity Resolution:** Implement deterministic IP-to-Hostname tracing that accounts for DHCP lease churn. If an IP triggers an alert on Tuesday, ensure it maps to the laptop that held the lease at that specific millisecond, not today's leaseholder.
- [ ] **Centralized DLP (Data Loss Prevention):** We have agent-based PII redaction, but cloud logs (AWS CloudTrail, Google Workspace) bypass the agent. Implement a central ingest-layer RegExp engine that dynamically masks SSNs, credit cards, and tokens before they touch indexing.
- [ ] **Raft Consensus Control Plane:** Move playbook storage, threat intel ingestion, and alert state management to a raft-backed consensus model to allow active-active control planes without split-brain corruption.

---

> All pages routed in `frontend-web/src/index.svelte` with context guards.

| Route | Component | Context | Phase |
|---|---|---|---|
| `/` | `Dashboard` | any | 0.3 |
| `/login` | `Login` | public | 0.3 |
| `/onboarding` | `Onboarding` | public | 0.3 |
| `/siem/search` | `SIEMSearch` | any | 0.3 |
| `/alerts` | `AlertManagement` | any | 0.3 |
| `/lookups` | `LookupManager` | any | 1.3 |
| `/threatintel` | `ThreatIntelDashboard` | any | 3.1 |
| `/enrich` | `EnrichmentViewer` | any | 3.2 |
| `/mitre-heatmap` | `MitreHeatmap` | any | 4 |
| `/fleet` | `FleetManagement` | web | 7 |
| `/identity` | `IdentityAdmin` | web | 12 |
| `/escalation` | `EscalationCenter` | web | 2.1.5 |
| `/playbooks` | `PlaybookBuilder` | any | 8 |
| `/playbook-metrics` | `PlaybookMetrics` | any | 8 |
| `/ueba` | `UEBADashboard` | any | 10 |
| `/peer-analytics` | `PeerAnalytics` | any | 10.5 |
| `/fusion` | `FusionDashboard` | any | 10.6 |
| `/ndr` | `NDRDashboard` | any | 11 |
| `/ransomware` | `RansomwareCenter` | any | 9 |
| `/regulator` | `RegulatorPortal` | web | 6.6 |
| `/evidence` | `EvidenceVault` | any | 6.5 |

---

## REST API Inventory (internal/api/)

> Registered in `rest.go` `NewRESTServer()`. All behind `secureMiddleware` + optional auth middleware.

| Endpoint | Method | Phase |
|---|---|---|
| `/api/v1/auth/login\|logout\|me\|oidc/*\|saml/*` | GET/POST | 0.3 |
| `/api/v1/siem/search` | GET/POST | 1.3 |
| `/api/v1/alerts` | GET | 2.1 |
| `/api/v1/events` | WS | 2.1 |
| `/api/v1/ingest/status` | GET | 1.2 |
| `/api/v1/agent/ingest\|register\|fleet` | GET/POST | 7 |
| `/api/v1/agents` | GET | 7 |
| `/api/v1/agentless/status\|collectors` | GET | 7.5 |
| `/api/v1/lookups/*` | GET/POST/DELETE | 1.3 |
| `/api/v1/escalation/*` | GET/POST/DELETE | 2.1.5 |
| `/api/v1/threatintel/stats\|indicators\|lookup\|campaigns` | GET | 3.1 |
| `/api/v1/enrich` + `/api/v1/enrich/recent` | GET | 3.2 |
| `/api/v1/mitre/heatmap` | GET | 4 |
| `/api/v1/forensics/evidence/*` + `/api/v1/forensics/export` | GET/POST | 6.5 |
| `/api/v1/audit/log\|packages\|packages/generate` | GET/POST | 6.6 |
| `/api/v1/users` + `/api/v1/roles` | GET | 12 |
| `/api/v1/ueba/profiles\|anomalies\|stats` | GET | 10 |
| `/api/v1/ueba/peer-groups\|peer-deviations` | GET | 10.5 |
| `/api/v1/ndr/flows\|alerts\|protocols` | GET | 11 |
| `/api/v1/ransomware/events\|hosts\|stats` | GET | 9 |
| `/api/v1/ransomware/isolate` | POST | 9 |
| `/api/v1/playbooks` | GET/POST | 8 |
| `/api/v1/playbooks/actions\|run\|metrics` | GET/POST | 8 |
| `/api/v1/fusion/campaigns` | GET | 10.6 |
| `/api/v1/fusion/campaigns/{id}/kill-chain` | GET | 10.6 |
| `/healthz` + `/readyz` + `/metrics` | GET | 2.3 |
| `/debug/attestation` | GET | 17 |
| `/api/v1/openapi.yaml` | GET | 2.2 |

---

## Phase 28: 2026-04-25 Verification Audit

> **Scope**: Re-checked every `[x]` claim in Phases 22, 23, 25, 26 against the actual code paths.
> Used four parallel Explore agents + targeted reads. The deltas below are reflected in-place above.

### ✅ Items confirmed as already-complete (status was `[ ]` but code exists)

| Item | Evidence |
|---|---|
| **22.1 Chaos test harness** | `cmd/chaos/main.go` (520 LOC) ships all four scenarios: WAL CRC replay, BadgerDB VLog corruption + truncate-mode reopen, OOM/burst load-shed probe, clock skew ±5 min. Plus `cmd/chaos-fuzzer/`, `cmd/chaos-harness/`. |
| **22.1 Automated soak regression** | `.github/workflows/soak.yml` triggers on every release tag + manual dispatch; runs 30 min × 5,000 EPS; fails on >10% EPS drop, >0.1% event loss, or min-window <50% of target. Captures heap pprof. |

### ⚠️ Items that were `[x]` but are actually partial/wrong (downgraded)

| Item | What's actually true |
|---|---|
| **22.2 Correlation state isolation** | LRU at `correlation.go:138` keys on `tenant+ruleID`, not `tenant+ruleID+groupKey`. groupKey isolation enforced *within* the LRU at lines 153-162. Functionally correct, claim wording overstates. |
| **22.2 Tenant deletion audit trail** | Status flip + salt wipe done; no immutable deletion record (no `deletion_log`, no audit-bus publish). GDPR right-to-erasure evidence missing. |
| **22.2 50-tenant isolation test** | Test runs **10 events/tenant**, not 1000 as claimed. Structural isolation valid; throughput claim overstated. |
| **22.4 SSH → anomaly banner** | `OperatorService.GetContext()` exists. UI status-bar surfacing + one-keypress event panel keybind missing. |
| **22.4 Host isolation from terminal** | `agentStore.toggleQuarantine` wired. No `Ctrl+Shift+I` keybind handler, no confirmation modal, titlebar status indicator unverified. |
| **23.2 Session restore banner** | Backend `session_persistence.go` save/restore present. `TerminalLayout.svelte` doesn't exist (current page is `TerminalPage.svelte`); banner UI missing. |
| **23.4 OperatorBanner.svelte** | `OperatorService` backend exists; component file `OperatorBanner.svelte` does not. |
| **23.5 Clipboard OSC 52** | XTerm imported. No OSC 52 handler, no auto-copy-on-selection, no right-click paste. Reset to `[ ]`. |
| **23.6 AI Autocomplete UI** | `CommandHistoryService.GetSuggestions` exists. No floating suggestion box, no cursor anchoring. Reset to `[ ]`. |
| **25.10 No multi-party enforcement** | HMAC-token replacement closes the *forgery* hole; FIDO2 hardware-signature verification of each approval is still missing (`quorum.go:111` skips it). |
| **26.4 System-Wide Backpressure** | Worker pool + bus rate limit + NATS priorities exist; explicit circuit breaker / bulkhead pattern absent. |
| **26.5 Cryptographic M-of-N Approval** | Voting structure exists; per-approval FIDO2 signature verification missing. |
| **26.9 Alert False-Positive Suppression** | `MarkFalsePositive` exists; rule-based suppression + automated feedback loop + maintenance windows do not. |
| **26.10 Hot/Warm/Cold Tiering** | Contradicted open `[ ]` in 22.3 — only Hot (Badger) + Parquet archive exist; no warm/cold migration. Reset to `[ ]`; owner is 22.3. |

### 🛠️ Fixes shipped during this audit pass

| Change | Files |
|---|---|
| Removed redundant `TenantID` string concat (Phase 25.9 reclassified — was dead code, not a leak vector). | `internal/api/rest.go` (handleSearch ~803, handleAlertsList ~846) |
| Added missing license gates on premium endpoints. The `/api/v1/ransomware/isolate` endpoint executes a destructive network isolation — previously **no license gate at all**. Now gated. Also fixed `playbooks/run`, `playbooks/metrics`, `ueba/stats`, `ndr/protocols`, `ransomware/events`, `ransomware/stats`. | `internal/api/rest_phase8_12.go` |
| Honeypot `RegisterTrigger` previously logged plaintext decoy username at WARN. Now logs only `id` + `type`. | `internal/security/honeypot_service.go:73` |
| Added `gosec` (SARIF → GitHub Security tab), `gitleaks` (full-history secret scanning), and `govulncheck` (CVE scan with reachability analysis) to `ci.yml`. Phase 25.4 #5 + #6 now resolved. | `.github/workflows/ci.yml` |

### ✅ Items confirmed correct (audit verdict: VERIFIED, no change)

A non-exhaustive list of `[x]` claims that survived the re-audit unchanged:
- 22.2 tenant-prefixed keyspace, per-tenant Bleve index, per-tenant encryption keys, query sandbox, provisioning API
- 25.2 osquery stdin-piped, GDPR table allowlist, lifecycle whitelist, FSM tmpPath validation, logsource TLS warning, TAXII InsecureSkipVerify=false, share-service expiresInMinutes
- 25.10 HMAC approval tokens (mcp/handler.go:161-182, rest.go:GenerateApprovalToken)
- 25.11 TOTP replay cache (sync.Map, 120s expiry), per-IP rate limiting + 5-failure account lockout
- 25.12 generic search/query error responses, agent ingest "invalid payload structure"
- 25.13 CSP / Referrer-Policy / Permissions-Policy headers; agent ingest 10MB MaxBytesReader
- 25.16 Evidence locker uses `DynamicHMACSigner{provider: keyProvider, purpose: "forensic_hmac"}` — key loaded from vault, not hardcoded (audit agent's "FAILED" verdict was incorrect on this one)
- 25.17 Canary path randomized via `time.Now().UnixNano()` (still in `/tmp/.oblivra_canary_<rand>` — randomization mitigates symlink pre-creation; could be hardened further with `O_CREAT|O_EXCL` semantics over SFTP)
- 26.1, 26.6, 26.7, 26.8 (verified above)

### 🚨 Open critical items not yet addressed by this pass

1. **`internal/services/settings_service.go:60`** still logs `Setting setting: %s=%s` at DEBUG with raw value — sensitive setting values (SMTP passwords, webhook secrets) leak to log files when DEBUG logging is on. (Phase 25.12)
2. **Phase 6 / Phase 12 self-validated compliance claims** still need reclassification to `[ ]` until externally audited.
3. **GDPR right-to-erasure** — tenant deletion path needs an immutable deletion record (Phase 22.2 partial item).
4. **`internal/isolation/manager.go`** non-constant format strings on logger calls (5 instances) and **`internal/memory/secure.go:71`** unsafe.Pointer warning — pre-existing `go vet` flags found during the audit; not security-critical but pollute the vet output and should be cleaned up before `go vet` becomes an enforcing CI gate.

### 🛠️ Fix shipped post-audit-summary

**`internal/security/fido2.go` and `internal/security/siem.go` parseTime misuse** — pre-existing build errors where callers treated `parseTime()` (returns `(time.Time, error)`) as a single value passed into `time.Now().After(...)` or `.Unix()`. Fixed in the same commit as the audit corrections (`fido2.go:95,167`, `siem.go:316`, plus `honeypot_service_test.go:38`). Unparseable challenge timestamps now fail closed (treated as expired); SIEM forwarder falls back to epoch on a malformed timestamp so the event still ingests instead of dropping. `go vet ./internal/api/... ./internal/security/...` exits clean.
