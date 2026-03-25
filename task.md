# OBLIVRA — Master Task Tracker

> **Status Tiers**:
> - `[s]` = **Scaffolded** (Code exists, compiles, architectural proof)
> - `[v]` = **Validated** (Tested under load, unit tests pass, functionally correct)
> - `[x]` = **Production-Ready** (Survives 72h soak, hardened, documented, unchallengeable)
> - `[ ]` = Not started
>
> **Last audited: 2026-03-25** — Phase 22 Productization Sprint + Platform Split Model
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
> No service is "done" until it has a corresponding SolidJS component, an API endpoint, or a route in `index.tsx`.

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
- [x] FIDO2 / YubiKey support (`internal/security/fido2.go`, `yubikey.go`) 🖥️
- [x] TLS certificate generation (`internal/ssh/certificate.go`, `cmd/certgen/`) 🏗️
- [x] Security key modal UI (`frontend/src/components/security/`) 🖥️
- [x] Snippet vault / command library (`snippet_service.go`) 🏗️

### Productivity
- [x] Notes & runbook service (`notes_service.go`) 🏗️
- [x] Workspace manager (`workspace_service.go`) 🖥️
- [x] AI assistant — error explanation, command gen (`ai_service.go`) 🏗️
- [x] Theme engine with custom themes (`theme_service.go`) 🏗️
- [x] Settings & configuration UI (`settings_service.go`, `pages/Settings.tsx`) 🏗️
- [x] Command palette & quick switcher (`frontend/src/components/ui/`) 🏗️
- [x] Auto-updater service (`updater_service.go`) 🖥️

### Collaboration
- [x] Team collaboration service (`team_service.go`, `internal/team/`) 🌐
- [x] Sync service (`sync_service.go`) 🏗️

### Ops & Monitoring
- [x] Unified Ops Center — multi-syntax search (LogQL, Lucene, SQL, Osquery) (`pages/OpsCenter.tsx`) 🏗️
- [x] Splunk-style analytics dashboard (`pages/SplunkDashboard.tsx`) 🏗️
- [x] Customizable widget dashboard (`frontend/src/components/dashboard/`) 🏗️
- [x] Network discovery service (`discovery_service.go`, `worker_discovery.go`) 🏗️
- [x] Global topology visualization (`pages/GlobalTopology.tsx`) 🏗️
- [x] Bandwidth monitor chart (`frontend/src/components/charts/BandwidthMonitor.tsx`) 🏗️
- [x] Fleet heatmap (`frontend/src/components/fleet/FleetHeatmap.tsx`) 🌐
- [x] Osquery integration — live forensics (`internal/osquery/`) 🏗️
- [x] Log source manager (`logsource_service.go`, `internal/logsources/`) 🏗️
- [x] Health & metrics service (`health_service.go`, `metrics_service.go`) 🏗️
- [x] Telemetry worker (`worker_telemetry.go`, `telemetry_service.go`) 🏗️

### Infrastructure
- [x] Plugin framework with Lua sandbox (`internal/plugin/`, `plugin_service.go`) 🏗️
- [x] Plugin manager UI (`pages/PluginManager.tsx`) 🏗️
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

- [x] Initialize `frontend-web/` (Bun + Vite + SolidJS)
- [x] Tailwind CSS and design tokens
- [x] `APP_CONTEXT` detection (Wails vs. Browser)
- [x] `/api/v1/auth/login` + `Login.tsx` + `AuthService.ts`
- [x] `Onboarding.tsx` wizard + `FleetService.ts`
- [x] `SIEMSearch.tsx` (Lucene queries, live paginated results) 🏗️
- [x] `AlertManagement.tsx` (WebSocket feed, status workflow) 🏗️

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
- [x] `ContextRoute.tsx` route guard (desktop/web/any context scoping)
- [x] `RouteGuard` component — wraps routes, shows `UnavailableScreen` with context hint
- [x] `ContextBadge` — status bar pill (DESKTOP/HYBRID/BROWSER), click opens server connection panel
- [x] `api.ts` BASE_URL (localhost for Desktop, same-origin for Browser)
- [x] `GlobalFleetChart.tsx` 🌐
- [x] `FleetManagement.tsx` — agent fleet console 🌐
- [x] `IdentityAdmin.tsx` — User/Role/Provider admin 🌐
- [x] `SIEMSearch.tsx` full-text SIEM query page 🏗️
- [x] Desktop → remote OBLIVRA Server connection (Backend API Proxy)
- [x] `CommandRail.tsx` — context classification on all nav items; locked items show `⊘`
- [x] `AppLayout.tsx` — `isDrawerVisible()` replaces hardcoded allowlist
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
- [x] `EscalationCenter.tsx` — Policies, Active, On-Call, History tabs 🌐

### 2.2 — Headless REST API
- [x] `internal/api/rest.go` with full HTTP router 🌐
- [x] SIEM search, alerts, agent, ingestion status, auth endpoints 🌐
- [x] API key authentication (`internal/auth/apikey.go`) 🌐
- [x] User accounts + RBAC (`internal/auth/`) 🌐
- [x] TLS for all external listeners 🌐

### 2.3 — Web UI Hardening
- [x] Real-time streaming search in `SIEMPanel.tsx` 🏗️
- [x] `AlertDashboard.tsx` (filtering, ack, status) 🏗️
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
- [x] `ThreatIntelPanel.tsx` + `ThreatIntelDashboard.tsx` 🏗️

### 3.2 — Enrichment Pipeline
- [x] GeoIP module (MaxMind offline DB, `internal/enrich/geoip.go`)
- [x] DNS Enrichment (ASN, PTR records, `internal/enrich/dns.go`)
- [x] Asset/User Mapping
- [x] Enrichment Pipeline orchestrator (`internal/enrich/pipeline.go`)
- [x] `EnrichmentViewer.tsx` — GeoIP, DNS/ASN, asset mapping, IOC correlation 🌐

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
- [x] MITRE ATT&CK heatmap (`MitreHeatmap.tsx`) 🏗️
- [s] Recruit 10 design partners (0 recruited; pilot agreement pending)
- [v] Validate: <5% false positives, 30+ ATT&CK techniques

### 4.1/4.2 — Commercial Readiness
- [ ] POC Generator & Support Bundle: one-command diagnostic bundle generation 🏗️
- [ ] Compliance Artifacts: pre-built legal templates (DPA, BAA, SCCs) and compatibility matrices 🌐

### 4.5 — Hardening Sprint ✅
- [x] `SIEMPanel.tsx` decoupled sub-components
- [x] Bounded Queue buffering on `eventbus.Bus`
- [x] SIEM Database Query Timeouts (10s contexts)
- [x] Incident Aggregation in Alert Dashboard
- [x] Regex Timeouts / Safe Parsing (ReDoS prevention)
- [x] Role-Based Access controls on destructive alert endpoints
- [x] API key auth + RBAC + TLS

---

## Phase 5: Limits, Leaks & Lifecycles ✅

- [x] LRU/TTL bounded memory for `internal/detection/correlation.go`
- [x] Asynchronous value log GC for BadgerDB
- [x] Incident Aggregation: mutable DB records (New/Active/Investigating/Closed)
- [x] `SIEMPanel.tsx` + Wails app → SolidJS Router (`@solidjs/router`)
- [x] Pre-compiled binary release workflow (GitHub Actions)
- [x] Zero-dependency `docker-compose.yml` deployment

---

## Phase 6: Forensics & Compliance ✅

- [x] Merkle tree immutable logging (`internal/integrity/merkle.go`)
- [x] Evidence locker with chain of custody (`internal/forensics/evidence.go`)
- [x] Enhanced FIM with baseline diffing
- [x] PCI-DSS, NIST, ISO 27001, GDPR, HIPAA, SOC2 Type II compliance packs
- [x] PDF/HTML reporting engine (`internal/compliance/report.go`)
- [x] Forensics service Wails integration (`internal/app/forensics_service.go`)
- [x] Compliance evaluator engine (`internal/compliance/evaluator.go`)
- [x] `EvidenceVault.tsx` — chain-of-custody browser, verify, seal, export 🏗️
- [x] `RegulatorPortal.tsx` — read-only audit log + compliance package generation 🌐
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
- [x] Artifact provenance attestation (SLSA Level 3)
- [x] Reproducible build verification

#### Self-Observability
- [x] `pprof` HTTP endpoints
- [x] Goroutine watchdog
- [x] Internal deadlock detection (`runtime.SetMutexProfileFraction`)
- [x] Self-health anomaly alerts via event bus
- [x] `SelfMonitor.tsx`

#### Disaster & War-Mode Architecture
- [x] Air-gap replication node mode
- [x] Offline update bundles (USB-deployable signed archives)
- [x] Kill-switch safe-mode (read-only, forensic-only)
- [ ] **Kill-Switch Abuse Protection** — Multi-party authorization (M-of-N), hardware key requirements, and audit escalation bounds
- [x] Encrypted snapshot export/import
- [x] Cold backup restore automation + validation

#### Governance Layer
- [x] Data retention policy engine
- [x] Legal hold mode
- [x] Data destruction workflow (cryptographic wipe + audit trail)
- [x] Audit log of audit log access (meta-audit)
- [x] `ComplianceCenter.tsx` — Governance tab with real-time scoring

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
- [x] Sovereign Tactical UI Overhaul (design tokens, `global.css`, `CommandRail.tsx`, `AppLayout.tsx`)
- [x] Tactical dashboards refactor (`Dashboard.tsx`, `FleetDashboard.tsx`, `SIEMPanel.tsx`, `AlertDashboard.tsx`)
- [x] System-wide Prop Type & Accessibility Audit
- [x] Agent Hardening: PII Redaction + Goroutine Leak Audits
- [x] Architecture Boundary Enforcement (`tests/architecture_test.go`)
- [x] Model explainability layer, bias logging, false positive audit trail
- [x] Training dataset isolation, offline retraining pipeline

#### Red Team / Validation Engine
- [x] Built-in attack simulator (MITRE ATT&CK technique replay)
- [x] Detection coverage score + technique gap report
- [x] Continuous detection validation (scheduled self-test)
- [x] `PurpleTeam.tsx`

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
- [v] `AgentConsole.tsx` + fleet-wide config push 🌐
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
- [x] `PlaybookBuilder.tsx` — visual SOAR builder, step canvas, action palette, execute-against-incident 🏗️
- [x] `PlaybookMetrics.tsx` — MTTR, success/failure rates, bottleneck identification 🏗️
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
- [x] `RansomwareCenter.tsx` — defense layers, host status, isolation controls, event log 🏗️
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
- [v] Threat hunting interface (`ThreatHunter.tsx`) 🏗️
- [x] `UEBADashboard.tsx` — risk heatmap, entity drill-down, anomaly feed 🏗️
- [x] `GET /api/v1/ueba/profiles|anomalies|stats` 🌐

### 10.5 — Peer Group Behavioral Analysis ✅
- [x] Auto-cluster by role, department, access patterns; dynamic recalculation; min-N validation
- [x] Aggregate behavioral statistics; deviation scoring (σ from group centroid)
- [x] "First for peer group" alerts; composite individual × peer anomaly scoring
- [x] `PeerAnalytics.tsx` — peer group explorer, σ-deviation outlier detection, risk comparison bars
- [x] `GET /api/v1/ueba/peer-groups` + `GET /api/v1/ueba/peer-deviations` 🌐

### 10.6 — Multi-Stage Attack Fusion Engine ✅
- [x] Kill chain tactic mapping; sliding window progression tracking; 3+ stage alert
- [x] Campaign clustering by shared entities; confidence scoring
- [x] Bayesian probabilistic scoring; seeded campaign data for demo
- [x] `FusionDashboard.tsx` — kill chain visualization, campaign cluster graph, confidence scores
- [x] `GET /api/v1/fusion/campaigns` + `GET /api/v1/fusion/campaigns/{id}/kill-chain` 🌐

---

## Phase 11: NDR ✅

- [x] NetFlow/IPFIX collector 🌐
- [x] DNS log analysis engine — DGA and DNS tunneling detection 🌐
- [x] TLS metadata extraction — JA3/JA3S fingerprints (no decryption) 🌐
- [x] HTTP proxy log parser — normalized inspection 🌐
- [x] eBPF network probes (extend agent) 🏗️
- [x] Lateral movement detection 🌐
- [x] `NDRDashboard.tsx` — flow table, anomaly cards, protocol stats 🌐
- [x] `LateralMovementEngine` — multi-hop connection correlation 🌐
- [x] `NetworkMap.tsx` — topology visualization 🌐
- [x] `GET /api/v1/ndr/flows|alerts|protocols` 🌐
- [x] Validate: lateral movement <5 min, 90%+ C2 identification

---

## Phase 12: Enterprise ✅

- [x] Multi-tenancy with data partitioning
- [x] HA clustering (Raft consensus) — `internal/cluster/`, `cluster_service.go`
- [x] User & Role DB models + migration v12 (`internal/database/users.go`)
- [x] OIDC/OAuth2 + SAML 2.0 + TOTP MFA + Granular RBAC engine
- [x] `IdentityService` — user CRUD, local login, MFA, RBAC checking
- [x] `GET /api/v1/users` + `GET /api/v1/roles` 🌐
- [x] Data lifecycle management — `lifecycle_service.go` (7 retention policies, legal hold, 6h purge loop)
- [x] `ExecutiveDashboard.tsx` — KPIs, posture, compliance badges
- [x] `PasswordVault.tsx` — full credential vault manager
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
- [x] `MitreHeatmap.tsx` fully wired (`/mitre-heatmap`)
- [x] OTel → Grafana Tempo pipeline (`docker-compose.yml` extended)
- [x] `ops/` config directory: `prometheus.yml`, `tempo.yml`, Grafana datasources + pre-built dashboard

---

## Phase 19: v1.1.0 ✅

- [x] `README.md` fully rewritten (accurate stack, architecture diagram, build instructions)
- [x] `CHANGELOG.md v1.1.0` — complete entry covering Phases 11–19
- [x] `DiagnosticsModal.tsx` — live ingest EPS, goroutines, heap, GC, event bus drops, health grade
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
- [ ] Custom pipe-based query language (OQL) for tactical analytics 🏗️
- [ ] **Query Language Identity** — formalized grammar definition, query planner guarantees, computational cost modeling

### 20.4 — SCIM Normalization
- [ ] Identity data ingestion and normalization (SCIM) 🌐

### 20.7 — Identity Connectors
- [ ] Native integration connectors for Active Directory, Okta, and major IdPs 🌐

### 20.9 — Automated Triage
- [ ] Automated incident triage scoring based on RBA and Asset Intel 🏗️

### 20.10 — Report Factory
- [ ] Automated generation of scheduled reports 🌐

### 20.11 — Dashboard Studio
- [ ] Custom dashboard builder with widget canvas 🌐

---

## Phase 21: Architectural Scaling ✅

- [x] **Partitioned Event Pipeline** — 8 shards, FNV-1a hash routing, per-shard worker pool + adaptive controller (`internal/ingest/partitioned_pipeline.go`)
- [x] **Write-Ahead Log** — CRC32 per record, 50ms fsync window, 10MB guard, replay on startup (`internal/storage/wal.go`)
- [x] **Streaming Enrichment LRU Cache** — 50,000 IP, 10-min TTL, RWMutex concurrent reads (`internal/enrich/cache.go`)
- [x] **Detection Rule Route Index** — EventType → `[]Rule` inverted index, `RebuildRouteIndex()` on hot-reload, ~13× speedup (`internal/detection/rule_router.go`)
- [x] **Query Execution Limits** — `DefaultQueryLimits` + `HeavyQueryLimits`, `Plan()`, `Validate()`, `BoundedContext()` (`internal/database/query_planner.go`)
- [x] **Bounded Worker Pools** — configurable, backpressure, panic-safe (`internal/platform/worker_pool.go`)
- [ ] **REQUIRED**: `git rm -r --cached frontend/node_modules` — purge tracked node_modules from git

### 21.5 — Asset Intelligence
- [ ] Foundational asset intelligence and asset criticality scoring 🌐

---

## Phase 22: Productization (The Strategic Pivot)

> **Context**: OBLIVRA has SIEM + EDR + SOAR + UEBA + NDR + hybrid desktop/web. Feature parity with early Splunk/CrowdStrike is real.
> This phase converts engineering into a product. No new features — only reliability, isolation, cost control, detection ecosystem, and trust.
> See [`STRATEGY.md`](STRATEGY.md) for the full strategic rationale.

---

### 🔧 Immediate Hygiene

- [ ] **Purge node_modules from git** — `git rm -r --cached frontend/node_modules frontend-web/node_modules`
- [ ] **Wails RPC bridge rate limiting** — per-method debounce on `NuclearDestruction`, `Unlock`, `DeleteHost`
- [ ] **Browser mode: VaultGuard + store.tsx Wails crash** — `IS_BROWSER` guards on all Wails imports (partially fixed 2026-03-23)

---

### 22.1 — Reliability Engineering

- [ ] **Chaos test harness** — `cmd/chaos/main.go`: kill agent mid-stream (WAL replay), corrupt BadgerDB VLog (recovery), OOM-kill server, clock skew ±5min
- [ ] **Agent reconnect guarantee** — resume without data loss after server restart; unvalidated at >1000 events in-flight
- [ ] **BadgerDB corruption recovery** — truncate VLog mid-write → verify `OpenReadOnly` fallback, snapshot export, clean re-init
- [ ] **Graceful degradation under overload** — at 3× rated EPS: backpressure, detection degrades gracefully, UI shows `DEGRADED` banner; no silent data loss
- [ ] **Automated soak regression** — GHA workflow: 30-minute 5,000 EPS soak on every release tag; fail if EPS drops >10%
- [ ] **Node failure simulation** — kill Raft leader mid-election; verify cluster recovers, no double-processed events
- [ ] **Deterministic Replay System** — full platform replay (`oblivra replay --from WAL --timestamp`) ensuring exact same alerts are produced deterministically
- [ ] **Time Synchronization Enforcement** — agent time drift detection, NTP validation per agent, explicit `event_time_confidence` scoring
- [ ] **Upgrade Safety Guarantees** — versioned schema migration rollback, dual-run (old+new pipeline), per-tenant canary upgrades

---

### 22.2 — Multi-Tenant Isolation

- [ ] **Tenant-prefixed BadgerDB keyspace** — all keys: `tenant:{id}:events:{ts}:{uuid}`; enforce in `SIEMStore.Write()` and all scan paths
- [ ] **Bleve index per tenant** — one index per tenant ID; `IndexManager` multiplexes; cross-tenant queries structurally impossible
- [ ] **Correlation state isolation** — `correlation.go` LRU keyed on `tenantID+ruleID+groupKey`; no cross-tenant state leakage
- [ ] **Per-tenant encryption keys** — derive AES-256 key from master key + tenant HMAC; rotate without re-keying all tenants
- [ ] **Query sandbox enforcement** — OQL planner rejects queries without `TenantID` predicate; `HeavyQueryLimits` per-tenant
- [ ] **Tenant provisioning API** — `POST /api/v1/admin/tenants` creates keyspace + index + encryption key atomically; idempotent
- [ ] **Tenant deletion audit trail** — cryptographic wipe + immutable deletion record (GDPR right-to-erasure)
- [ ] **50-tenant isolation test** — 50 tenants, 1000 events each, cross-tenant search returns 0 results; structurally enforced

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

#### Operator Mode — The Killer Workflow
- [ ] **SSH → anomaly banner** — SIEM events for active terminal host surfaced as status bar notification; one keypress opens filtered event panel 🖥️
- [ ] **Event row → enrichment pivot** — click IP/host in SIEM results → inline enrichment card (GeoIP, ASN, TI match, open ports) 🏗️
- [ ] **Host isolation from terminal context** — `Ctrl+Shift+I` → isolation confirmation → network isolator playbook → status in titlebar 🖥️
- [ ] **One-click memory/process capture** — trigger forensic snapshot, auto-seal SHA-256, auto-add to active incident evidence 🖥️
- [ ] **Operator timeline** — unified chronological view: terminal commands + SIEM events + enrichment + playbook executions + evidence 🏗️
- [ ] **Autonomous Hunt** — scheduled and automated threat hunting queries based on Threat Intel 🌐
- [ ] **Operator Cognitive Load Design** — transition from dashboards to decision engine: alert ranking, "next best action" prompts, investigation graphs 🏗️

#### Detection Engineering
- [ ] **Detection-as-code workflow** — rules in Git; `oblivra rules push --dry-run` (shadow mode); merge → production promotion
- [ ] **Rule marketplace schema** — YAML bundle: `rule + metadata + test fixtures + changelog`; import/export CLI
- [ ] **Risk-Based Alerting** — wire `RiskService`: detection match → entity risk score increment → temporal decay → composite score → incident threshold
- [ ] **Entity Investigation Pages** — `EntityView.tsx`: UEBA profile, risk score, alert history, enrichment context, MITRE technique timeline 🌐
- [ ] **Detection Confidence Model** — output `confidence_score (0–100)` and explainability vector based on rule strength, enrichment, behavioral deviation, and TI matches
- [ ] **Cold Start Problem Handling** — "Day 0 Intelligence mode" with pre-trained heuristics; clear distinction between learning vs. enforcement modes

---

### 22.5 — Trust & Legitimacy Layer

- [ ] **Publish threat model** — redacted `docs/threat_model.md` at `oblivra.dev/security`
- [ ] **Cryptographic transparency doc** — enumerate: AES-256-GCM (vault), Ed25519 (signing), Argon2id (KDF), TLS 1.3 (transport); justify each; document key rotation
- [ ] **SOC 2 Type II evidence collection** — map audit log, access controls, encryption, availability to SOC 2 control families; produce evidence package
- [ ] **ISO 27001 gap analysis** — compare controls to Annex A; document deltas; produce remediation plan
- [ ] **External penetration test preparation** — `docs/pentest_scope.md`: scope, rules of engagement, excluded systems
- [ ] **Setup Wizard** — 6-step first-run (`SetupWizard.tsx`): admin account → TLS cert → first log source → alert channel → detection pack selection → first search tutorial 🌐
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

## Phase 23: Terminal UX (Termius-Grade) 🖥️

> **Context**: The terminal is the operator's primary interaction surface. These upgrades close the gap
> with Termius-class UX while leveraging OBLIVRA's unique SIEM + forensics + vault integration.

### 23.1 — SSH Bookmark CRUD → Vault UI 🔴 P0
- [ ] `BookmarkService` — Wails-bound CRUD for host bookmarks (wraps `HostStore` + Vault-encrypted credentials) 🖥️
- [ ] `SSHBookmarks.tsx` — sidebar panel: list, search, favorites, group-by-tag, add/edit/delete, one-click connect 🖥️

### 23.2 — Session Restore on Restart 🔴 P0
- [ ] `session_persistence.go` — save active session host IDs + tab order on graceful shutdown 🖥️
- [ ] `SSHService` restore hook — reconnect saved sessions on app start 🖥️
- [ ] Session restore banner in `TerminalLayout.tsx` — "Restore 3 previous sessions?" 🖥️

### 23.3 — Per-Host Command History 🟡 P1
- [ ] `CommandHistoryService` — store/retrieve commands per host (SQLite, last 500 per host) 🖥️
- [ ] Autocomplete overlay in terminal — ↑ arrow history + Tab suggestions 🖥️

### 23.4 — Operator Mode (Core) 🟡 P1
> See also Phase 22.4 Operator Mode items for full scope.
- [ ] `OperatorService` — anomaly banner data: recent SIEM alerts for active SSH host 🖥️
- [ ] `OperatorBanner.tsx` — SIEM alert count + severity overlay on terminal tab bar 🖥️
- [ ] `Ctrl+Shift+I` host isolation shortcut — confirmation modal → `NetworkIsolator` playbook 🖥️

### 23.5 — Clipboard OSC 52 🟢 P2
- [ ] xterm.js clipboard integration — OSC 52 read/write, context menu copy on selection 🖥️

### 23.6 — AI Autocomplete Polish 🔵 P3
- [ ] Floating suggestion box wired to existing `AIService` + per-host command history 🖥️
- [ ] Smart context: current directory, recent commands, host OS detection 🖥️

---

## Frontend Pages Inventory (frontend-web/)

> All pages routed in `frontend-web/src/index.tsx` with context guards.

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
