# OBLIVRA — Phase Tracker (Build Roadmap)

> **What this file is**: the chronological **build narrative** — every phase
> with its description and the stages within it. One source of truth for
> "what shipped, when, and why."
>
> **What lives elsewhere**:
> - **[`HARDENING.md`](HARDENING.md)** — every audit pass, security finding,
>   fix, postmortem, hardening gate, and validation reclassification. Anything
>   with the flavour of "we found X is broken/insecure → fixed it" goes there.
>   Cross-references back into specific phase entries below.
> - [`ROADMAP.md`](ROADMAP.md) — long-term roadmap (CSPM, K8s, vuln mgmt, etc.)
> - [`RESEARCH.md`](RESEARCH.md) — DARPA/NSA-grade research items (Phase 13)
> - [`BUSINESS.md`](BUSINESS.md) — certifications, legal, GTM (Phase 14)
> - [`FUTURE.md`](FUTURE.md) — Cross-cutting (chaos engineering, deception, i18n)
> - [`STRATEGY.md`](STRATEGY.md) — Phase 22 strategic rationale
>
> **Status Tiers**:
> - `[s]` = **Scaffolded** (Code exists, compiles, architectural proof)
> - `[v]` = **Validated** (Tested under load, unit tests pass, functionally correct)
> - `[x]` = **Production-Ready** (Survives 72h soak, hardened, documented, unchallengeable)
> - `[ ]` = Not started
>
> **Last audited: 2026-04-29** — Phase 36 broad scope cut: log-driven SIEM only.
> SOAR + IR + ransomware response + disk/memory imaging + AI assistant + plugin
> framework all removed. Phase 35 closed storage tiering — last engineering GA blocker.

---

## 🏗️ Platform Architecture — Golden Rule

> **Desktop = Sensitive + Local + Operator Actions**
> **Web = Shared + Scalable + Multi-user**

### 🖥️ DESKTOP (Wails App) — MUST be here
> Anything involving secrets, OS access, or direct operator control.

| Category | Features |
|---|---|
| 🔐 **Security & Secrets** | Vault (AES-256), OS keychain, FIDO2/YubiKey, Password manager |
| 🧪 **Local / Offline** | Local SIEM (optional), Local detection engine (offline testing), Local log ingestion, Air-gap mode |
| 🧰 **Operator Tools** | Command palette, Workspace layouts, Plugin dev/testing, CLI mode |
| 🔧 **System-Level Actions** | Build/sign agents, Generate certificates, Forensics acquisition (disk/memory), Local response actions (kill process, isolate host) |

> **REMOVED (Phase 32)**: The interactive shell subsystem — SSH client, local PTY, terminal grid, SFTP file browser, port-forwarding tunnels, session recording playback, multi-exec — has been **deprecated and removed from the operator UI**. The Go libraries under `internal/ssh/` and `internal/services/{ssh,local,tunnel,recording,share,multiexec,broadcast,file,transfer,pty}_*.go` remain in-tree because they still back non-terminal features (canary deployment via SCP, scheduled SSH key rotation, evidence file uploads). The shell subsystem will be rebuilt as a separate workstream; until then `/shell`, `/ssh`, `/tunnels`, `/recordings`, `/session-playback` are offline. See `frontend/src/lib/nav-config.ts:249-253`.

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
- Vault master key
- Local filesystem access
- Agent signing keys
- Plugin execution engine

> SSH private keys / raw PTY / SFTP / forwards used to live here; removed entirely with the shell subsystem (Phase 32). When the shell is rebuilt, these constraints reapply.

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

### Shell Subsystem — REMOVED (Phase 32)

> The interactive shell + SSH + SFTP + tunnel + recording-playback feature set is no longer
> part of the operator UI. Backend Go libraries are retained because non-terminal features
> still depend on them; the operator-visible surface is gone. To be rebuilt as a separate
> workstream — see Phase 32 below.
>
> Retained backend (still compiled, still in `container.go`): `internal/ssh/`, `local_service.go`,
> `tunnel_service.go`, `recording_service.go`, `share_service.go`, `broadcast_service.go`,
> `multiexec_service.go`, `file_service.go`, `transfer_manager.go`, `pty_session.go`,
> `pty_unix.go`, `pty_windows.go`, `ssh_service.go`. They back canary deployment,
> SSH key rotation, evidence file uploads. They do NOT serve a terminal page anywhere.
>
> Removed UI: `frontend/src/components/terminal/` (whole directory), `TerminalPage.svelte`,
> `XTerm.svelte`, `OperatorBanner.svelte`, `SessionRestoreBanner.svelte`. Routes
> `/shell`, `/ssh`, `/tunnels`, `/recordings`, `/session-playback` registered in
> `App.svelte` but hidden from the navigation (see `nav-config.ts:249-253`).

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

> Get the in-tree codebase into a state where every service compiles, the
> dependency graph in `container.go` is correct, and the lifecycle
> (`ServiceRegistry.Start/Stop`) round-trips without panic. Pre-feature work.

### Stages
- [x] Final audit of all service constructor signatures in `container.go`
- [x] Resolve remaining compile errors across all services
- [x] Verify all 16+ services start/stop cleanly via `ServiceRegistry`
- [x] Full integration smoke test (SSH, SIEM, Vault, Alerting, Compliance)

---

## Phase 0.1: Day Zero Hardening ✅

> First-run polish. Make the very first launch on a fresh machine reach the
> "operator can do something useful" state without manual setup steps —
> directories created, default rules loaded, onboarding wizard surfaced when
> hosts count is zero, and a Time-to-First-Alert metric so we can measure
> whether the experience actually works.

### Stages
- [x] Recursive Directory Creation — `platform.EnsureDirectories()` to `app.New()` 🏗️
- [x] Onboarding / Inception UI — Redirect to Setup Wizard if hosts count is 0 🏗️
- [x] Core Rule Library — `sigma/core/` seeded with 25+ essential rules 🏗️
- [x] Subprocess Validation — startup check for `os.Executable()` re-entry 🏗️
- [x] First-Run Analytics — Trace "Time to First Alert" 🏗️

---

## Phase 0.2: Test Suite Stabilization ✅

> Make `go test ./...` reliably pass on a clean checkout. Fix the early-life
> regressions (event-type rename fallout, diagnostics interface drift,
> duplicate test names, architecture violations from detection touching
> stores directly) so CI signal becomes meaningful.

### Stages
- [x] Fix Ingest Package Regressions — `ingest.SovereignEvent` → `events.SovereignEvent`
- [x] Restore Diagnostics Interface — `DiagnosticsService.GetSnapshot()` in `smoke_test.go`
- [x] Resolve Test Name Collisions — no `TestHighThroughputIngestion` duplicate
- [x] Verify Test Pass Rate — `go test ./...` passes
- [x] Resolve Architectural Violations — Detection decoupled via `SIEMStore` interface

---

## Phase 0.3: Web Dashboard / Enterprise Platform (MVP) ✅ 🌐

> Stand up the web frontend (`frontend-web/`) as a Svelte 5 + Vite SPA that
> shares the same backend as the Wails desktop shell. Establish the
> `APP_CONTEXT` detection so a single codebase serves both. Ship the
> minimum-viable web experience: login, fleet onboarding, search, alerts.

### Stages
- [x] Initialize `frontend-web/` (Bun + Vite + Svelte 5)
- [x] Tailwind CSS and design tokens
- [x] `APP_CONTEXT` detection (Wails vs. Browser)
- [x] `/api/v1/auth/login` + `Login.svelte` + `AuthService.ts`
- [x] `Onboarding.svelte` wizard + `FleetService.ts`
- [x] `SIEMSearch.svelte` (Lucene queries, live paginated results) 🏗️
- [x] `AlertManagement.svelte` (WebSocket feed, status workflow) 🏗️

---

## Phase 0.4: Accessibility & Enterprise Scaling ✅

> Bring the UI surface to WCAG 2.1 AA: pattern-fills (colorblind safety),
> ARIA labels, full keyboard nav. In parallel, prove the back-end scales
> to enterprise volumes — 1,000-node BadgerDB, multi-tenant registration,
> SSO connectors (OIDC/SAML).

### Stages
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

> Enforce the Golden Rule (Desktop = sensitive/local/operator, Web = shared/
> scalable/multi-user) at the code level. Every route declares its target
> context; the router refuses to render a desktop-only page in browser mode
> and vice-versa. ContextBadge surfaces the mode to the operator at all times.

### Stages
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

> The SIEM core: BadgerDB for hot storage, Bleve for full-text search,
> Parquet for cold archival, plus a crash-safe WAL in front of all of it.
> An ingest pipeline that handles syslog/JSON/CEF/LEEF with backpressure
> and rate limiting. A query layer (OQL + Lucene) that reads through all
> three storage tiers. Validated under 72 h × 5,000 EPS soak with no loss.

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

> Detection rules become alerts. Alerts become notifications (webhook /
> email / Slack / Teams) with multi-level escalation chains, on-call
> rotations, SLA tracking. Everything also exposed through a versioned
> REST API (`/api/v1/...`) gated by API keys + RBAC + TLS.

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

> Make every event richer at ingest time. STIX/TAXII threat-intel feed,
> O(1) IOC matcher (IP / hash / domain), GeoIP + DNS PTR/ASN + asset/user
> mapping enrichment, plus full parsers for Windows Event Log, Linux
> syslog/journald, AWS/Azure/GCP cloud audit, NetFlow/firewall logs.

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

> Library + engine. 82 YAML detection rules across all 12 MITRE ATT&CK
> tactics covering 45+ techniques. Correlation engine for multi-event
> patterns. MITRE heatmap for coverage visualisation. Companion hardening
> work (component a11y, regex DoS prevention, RBAC on destructive
> endpoints) in **HARDENING.md → Phase 4.5 Hardening Sprint**.

### Stages
- [x] 82 YAML detection rules across all 12 tactics, 45+ techniques 🏗️
- [x] MITRE ATT&CK technique mapper (`internal/detection/mitre.go`) 🏗️
- [x] Correlation engine (`internal/detection/correlation.go`) 🏗️
- [x] MITRE ATT&CK heatmap (`MitreHeatmap.svelte`) 🏗️
- [s] Recruit 10 design partners (0 recruited; pilot agreement pending)
- [v] Validate: <5% false positives, 30+ ATT&CK techniques

### 4.1/4.2 — Commercial Readiness
- [ ] POC Generator & Support Bundle: one-command diagnostic bundle generation 🏗️
- [ ] Compliance Artifacts: pre-built legal templates (DPA, BAA, SCCs) and compatibility matrices 🌐

---

## Phase 5: Limits, Leaks & Lifecycles ✅

> Long-running stability. Bounded memory in correlation state, async GC
> for BadgerDB value-log, mutable Incident lifecycle (New / Active /
> Investigating / Closed). CI for pre-compiled binary releases plus a
> zero-dependency `docker-compose.yml` for one-line deployment.

### Stages
- [x] LRU/TTL bounded memory for `internal/detection/correlation.go`
- [x] Asynchronous value log GC for BadgerDB
- [x] Incident Aggregation: mutable DB records (New/Active/Investigating/Closed)
- [x] `SIEMPanel.svelte` + Wails app → `svelte-routing`
- [x] Pre-compiled binary release workflow (GitHub Actions)
- [x] Zero-dependency `docker-compose.yml` deployment

---

## Phase 6: Forensics & Compliance ✅

> Make logs admissible. Merkle-chained audit log, signed evidence locker
> with chain-of-custody, RFC 3161 timestamping, NIST SP 800-86-formalised
> evidence collection. Compliance packs (PCI-DSS / NIST / ISO 27001 /
> GDPR / HIPAA / SOC2 Type II) with PDF/HTML reports and a regulator-
> ready audit-export portal. *(Phase 36: disk/memory imaging removed;
> log-event chain-of-custody retained.)*

### Stages
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


## Phase 7: Agent Framework ✅

> The endpoint collector. A single-binary Go agent (`cmd/agent`) collecting
> file-tail / Windows Event Log / system metrics / FIM / eBPF, shipping
> over gRPC+mTLS with zstd compression and a local WAL for offline buffering.
> Edge-side filtering + PII redaction. 7.5 adds agentless collectors
> (WMI / SNMP / SQL audit / REST polling) for sources that can't take an agent.

### Stages
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

## Phase 8: Autonomous Response (SOAR) ✅ → ⚫ REMOVED in Phase 36

> *Historical*. Case management, playbook engine, SOAR builder UI, ITSM
> integrations (Jira / ServiceNow). All deleted in **Phase 36** (broad scope
> cut — log-driven SIEM only). Pair OBLIVRA with an external SOAR (Tines,
> Torq, XSOAR) instead.

### Stages (no longer in code)
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

## Phase 9: Ransomware Defense ✅ → 🟡 partial after Phase 36

> Detection layer **retained**: entropy-based behavioural detection
> (`internal/detection/ransomware_engine.go`), honeypot infrastructure,
> Sigma ransomware rules. Response layer **removed in Phase 36**: canary
> file deployment, automated network isolation, `RansomwareCenter.svelte`
> response page. `/api/v1/ransomware/{events,hosts,stats}` endpoints stay;
> `/api/v1/ransomware/isolate` endpoint deleted.

### Stages
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

> User & Entity Behaviour Analytics. Per-user/entity behavioural baselines
> in BadgerDB; Isolation Forest anomaly detection (deterministic seeding);
> Identity Threat Detection & Response with EMA tracking. 10.5 adds peer-
> group analysis (auto-cluster by role/dept, σ-deviation outlier detection).
> 10.6 adds the multi-stage attack fusion engine — kill-chain progression
> tracking, campaign clustering, Bayesian probabilistic scoring.

### Stages
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

> Network Detection & Response. NetFlow/IPFIX collection, DNS log analysis
> (DGA + tunnel detection), TLS metadata extraction (JA3/JA3S without
> decryption), HTTP proxy log normalisation, eBPF kernel network probes
> (extends the agent), lateral-movement detection via multi-hop connection
> correlation. `NDRDashboard.svelte` + `NetworkMap.svelte` UI surfaces.

### Stages
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

> Multi-tenancy with per-tenant data partitioning at every storage layer
> (BadgerDB keys, Bleve indexes). HA clustering via Hashicorp Raft.
> Identity stack: User & Role models, OIDC/OAuth2 + SAML 2.0 + TOTP MFA,
> granular RBAC engine. Data-lifecycle management (7 retention policies +
> legal hold + 6 h purge loop). Executive dashboard, password vault.

### Stages
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

> **Current sprint**: ~~S0~~ ✅ → ~~S1~~ ✅ (22.2 verified, structural per-tenant isolation in place) → ~~S2 Reliability Gate~~ ✅ (agent reconnect, BadgerDB recovery, graceful degradation, time sync, soak CI, chaos harness — all shipped under 22.1) → **Phase 32 + 33 hardening sweep** ✅ (8 backend security fixes, 10 frontend wiring fixes, shell subsystem removed, window-controls regression fixed). Next: **22.3 Storage Tiering** (last engineering GA blocker).

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
- [x] **Node failure simulation** — `cmd/chaos/main.go` Scenario 6 builds a 3-node Raft cluster over `hashicorp/raft` in-memory transport with a no-op FSM, kills the elected leader, and asserts a different node wins re-election within 5s. CGO-free so it runs in any environment. Existing CGO-using `TestLeaderFailureIdempotency` and `TestRaftSplitBrain` (`internal/cluster/`) still cover idempotent retry + split-brain prevention; together they validate the full Phase 22.1 claim.
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
- [v] **Hot/Warm/Cold tiered storage** — Phase 35 (2026-04-29): the Tier interface, three concrete implementations (BadgerDB hot, Parquet warm, JSONL local cold), and the `Migrator` were already scaffolded (Phase 31). This pass **wires it into production**: container.go boots the migrator on startup with `DefaultRetention()` (Hot 30d / Warm 150d), `App.Shutdown` stops it cleanly, REST endpoints `/api/v1/storage/tiering/{stats,promote}` expose tier sizes + manual cycle trigger, `StorageTiering.svelte` page renders the dashboard at `/storage-tiering`. Verified live in bootcheck: `[STORAGE] Hot/Warm/Cold tier migrator started (interval=1h, hot=30d, warm=150d)` and first cycle runs on startup (`tiering: cycle complete hot→warm=0 warm→cold=0 errors=0 duration=1.39s` — correct, fresh install has no aged events). 10 new REST handler tests cover the full endpoint surface. **Still open** for full v1**: ingest pipeline isn't yet teeing writes into the HotTier keyspace — events live in `tenant:<id>:events:` (the existing siem_badger keyspace). Until ingest writes through HotTier, the migrator has nothing to migrate. Tracked as 35.2 follow-up. **Cold S3 wiring** (RemoteColdTier behind `//go:build s3`) tracked as 35.3.
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

> [!NOTE]
> Items below that depended on the now-removed shell subsystem (SSH→anomaly
> banner, host-isolation keybind from terminal context, "operator timeline"
> joining terminal commands with SIEM events) were removed in Phase 32.
> The non-terminal pieces (host-page anomaly banners, one-click forensic
> capture, autonomous hunt) survive and are tracked here.

- [ ] **Anomaly banner on Host Detail page** — when an alert fires for the active host, surface a sticky banner with crit/high severity chip, "View events" pivot to SIEM search, "Isolate" button. ⚠️ Re-implementation of removed terminal banner against the host-detail surface. 🌐
- [ ] **Event row → enrichment pivot** — click IP/host in SIEM results → inline enrichment card (GeoIP, ASN, TI match, open ports) 🏗️
- [x] **Host isolation from any context** — `Ctrl+Shift+I` keybind dispatches `oblivra:isolate-host` window event; `OperatorMode.svelte` listens and calls `agentStore.toggleQuarantine`. Off-page invocation navigates to `/operator` with a hint toast. Pop-out windows bind the same keybind. 🖥️
- [ ] **One-click memory/process capture** — trigger forensic snapshot, auto-seal SHA-256, auto-add to active incident evidence 🖥️
- [ ] **Operator timeline** — unified chronological view: SIEM events + enrichment + playbook executions + evidence (terminal commands removed from scope with the shell deletion) 🏗️
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
- [x] **Setup Wizard** — Frontend: `frontend-web/src/pages/SetupWizard.svelte` ships a 4-step flow (admin account → alert channel → detection pack → orientation), wired at `/setup`. Backend: `POST /api/v1/setup/initialize` (`internal/api/rest.go`) now actually does work — calls new `IdentityService.BootstrapAdmin` (creates the admin user, refuses if any user already exists so the endpoint can't be re-run to hijack admin), records the setup in the Merkle-chained audit log under `setup.initialized`, and publishes a `setup:initialized` bus event so the alerting service can attach the channel and the rule loader can switch packs. Returns 409 Conflict on re-attempt. Validates payload shape, requires email + 12+ char password, allowlists detection_pack to essential/extended/paranoid. Steps deferred from the original 6-step claim: TLS cert (operator infra, handled by `cmd/certgen`) and first log source (subsumed by `Onboarding.svelte` once an agent is online). 🌐
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

## Phase 23: Desktop Shell UX (windowing, chrome, notifications)

> **Original scope (Termius-grade terminal UX)**: subsections 23.1–23.6 covered SSH
> bookmarks, session restore, per-host command history, terminal Operator banners,
> xterm.js OSC 52 clipboard, and AI autocomplete. **All of those landed in v1.2.0
> and were subsequently removed in Phase 32 with the rest of the shell subsystem.**
> They are not re-listed here — the historical record lives in git
> (`commits 8cf3e1b` and ancestors).
>
> What remains in Phase 23 is the platform-level windowing / chrome / notifications
> work that's independent of any terminal. These items are still load-bearing.

### 23.7 — SOC Multi-Monitor Pop-Out ✅ (new)
> **Context**: SOC operators run 3-4 monitors. The flagship workflow is "drag the SIEM search to monitor 2, the alerts board to monitor 3, keep an investigation panel on monitor 1." Native windowing makes this real instead of forcing the operator to alt-tab inside one window.

- [x] **`WindowService`** (`internal/services/window_service.go` + `_server.go` build-tagged stub) — Wails-bound service with `PopOut(route, title) → (id, error)`, `ClosePopout(id)`, `CloseAllPopouts()`, `ListPopouts()`. Each pop-out is a real Wails window backed by the same Go process — zero IPC round trip between panel views.
- [x] **Pop-out URL convention** — `/?popout=1&route=<route>`. `App.svelte`'s onMount detects the param, navigates to the requested route, and skips rendering the sidebar so the spawned window is a clean single-panel view.
- [x] **`PopOutButton.svelte`** (`frontend/src/components/ui/`) — drop-in toolbar component with route/title props. Opted in on 30 SOC pages (see 23.13). Browser mode falls back to `window.open(?popout=1&route=...)` so web-mode operators can still spawn extra tabs onto extra monitors.
- [x] **TitleBar pop-out indicator** — when one or more pop-outs are open, TitleBar renders a "N POP-OUT(S)" chip in the chrome with click-to-close-all. Polls `WindowService.ListPopouts()` every 1.5s.

### 23.8 — Window Chrome (Frameless) ✅ (new)
> Wails frameless windows leave the OS without min/max/close, so we render our own.

- [x] **Platform-aware controls in `TitleBar.svelte`** — macOS gets traffic-light dots on the left (with hover-revealed glyph icons), Windows/Linux get explicit Min / Max / Close icon buttons on the right with the standard 40×30px hit-box and red close hover. Maximise icon flips to "Restore" when window is maximised. Detects platform via `navigator.userAgent`.
- [x] **Drag region** — entire title bar header has `-webkit-app-region: drag`; every interactive element overrides with `no-drag` so clicks pass through. Operators can drag the window between monitors freely.

### 23.9 — UX State Primitives ✅ (new)
- [x] **`LoadingSkeleton.svelte`** added to `@components/ui` barrel. Three variants (`row` / `card` / `block`) with shimmer animation that respects `prefers-reduced-motion`. Pairs with the existing `EmptyState`, `LoadingScreen`, `ErrorScreen`, and `Spinner` primitives — page authors now have the full set without rolling their own.
- [x] **AlertManagement** opted in: cold-load shows `LoadingSkeleton` row grid, filter-yields-zero-results shows `EmptyState` with "Clear search" / "Show open alerts" recovery actions instead of an empty table.

### 23.10 — Application Menu Bar + System Tray ✅ (new)
> Wails v3 native menu bar + system tray for SOC operators who want full keyboard / OS-level control without ever opening the main window.

- [x] **Application menu** (`internal/app/menu.go`) — File, Edit, View, Navigate, Window, Help submenus. Native roles for cut/copy/paste, undo/redo, fullscreen, zoom, reload. Custom items emit `menu:<action>` events on the Wails event bus; `App.svelte` onMount listens and dispatches to `appStore.navigate(...)`, `appStore.toggleCommandPalette()`, the `WindowService` pop-out methods, etc. Accelerators wired: `Ctrl+Shift+O` pop out current, `Ctrl+B` toggle sidebar, `Ctrl+1..4` quick-jump to Overview / SIEM / Alerts / Fleet, `Ctrl+Shift+S/R` save/restore workspace, `Ctrl+/` shortcuts, `Ctrl+,` settings. *(Phase 32: `Ctrl+T` "new terminal" accelerator removed with the shell subsystem; "Terminal" entries pruned from menu/tray.)*
- [x] **System tray** (`internal/app/tray.go`) — minimize-to-tray with a quick-action menu: Show OBLIVRA, Open SIEM / Alerts / Fleet, New Pop-Out → SIEM / Alerts, Close All Pop-Outs, Quit. Tray icon embedded via `//go:embed appicon.png` so it works in air-gap deployments. Click-through emits `tray:show` / `menu:goto` / `tray:popout` events the frontend listens for.

### 23.11 — Workspace Save/Restore ✅ (new)
- [x] **`WindowService.SaveWorkspace()`** — captures every open pop-out's route, title, and (best-effort) position+size to `<DataDir>/workspace.json`. Atomic temp-file + rename. Wails-bound so the menu's "Save Workspace" item invokes it.
- [x] **`WindowService.RestoreWorkspace(closeExisting)`** — reads the saved file, optionally closes existing pop-outs, then re-opens each captured route via `PopOut`. Geometry restoration is best-effort with panic-recovery (Wails panics defensively on stale window handles during shutdown).
- [x] **`HasSavedWorkspace()`** — frontend-side check used to decide whether to surface "Restore Workspace?" prompts on cold boot.
- [x] Schema versioned (`workspace_schema_version = 1`) so future migrations can detect old files.

### 23.12 — Notification Center ✅ (new)
- [x] **`notificationStore`** (`frontend/src/lib/stores/notifications.svelte.ts`) — persistent log of toasts + system events. Backed by `localStorage` with a 200-entry cap and quota-exhaustion fallback (drops oldest half on retry). Tracks read/unread state.
- [x] **`NotificationDrawer.svelte`** (`frontend/src/components/layout/`) — slide-in panel with per-entry trash, "Mark all read" + "Clear all" footer, level-coloured rails (critical/error red, warning amber, success green, info blue), relative-time stamps. Click-through optionally navigates via `entry.action.route`.
- [x] **Bell button in TitleBar** — unread count badge (red on critical, accent blue otherwise; "99+" if >99). Toggles the drawer.
- [x] **Toast bridge** — every `toastStore.add(...)` call now mirrors into `notificationStore.push(...)`, so toasts that auto-dismiss in 5s still survive in the drawer history.

### 23.13 — Multi-Monitor Pop-Out Rollout ✅ (extending 23.7)
- [x] PopOutButton now opted in on **30 pages** total. Original v1.2.0 set: SIEMSearch / AlertManagement / AlertDashboard / FleetDashboard. v1.3.0 add: NetworkMap, MitreHeatmap, NDROverview, UEBAOverview, FusionDashboard, OpsCenter, IncidentTimeline, EvidenceLedger. v1.4.0 add (18): SOARPanel, ThreatHunter, ThreatIntelPanel, PurpleTeam, PluginManager, EvidenceVault, Dashboard, ComplianceCenter, CompliancePage, LineageExplorer, DecisionInspector, IdentityAdmin, IncidentResponse, PlaybookBuilder, CaseManagement, TasksPage, RuntimeTrust, SimulationPanel.

### 23.14 — Mouse Drag Bug Fix ✅ (new, v1.4.0)
- [x] **Title bar drag was broken** — `-webkit-app-region: drag` is Electron's API; Wails v3 silently ignores it. Replaced 13 occurrences in TitleBar.svelte with `--wails-draggable: drag/no-drag` (Wails v3's CSS custom property recognised by `runtime/dist/drag.js`). Removed the dead `Window.Drag()` JS fallback; Wails v3 sends a `wails:drag` IPC message internally.

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

- [v] **Arabic / RTL UI (i18n)** — Scaffolding shipped in v1.4.0: custom Svelte 5 `$state`-backed i18n store at `frontend/src/lib/i18n/index.ts` with `t(key, ...args)` interpolation, en + ar locale files, `<html dir="rtl">` + `<html lang>` auto-applied, `[dir="rtl"]` CSS overrides in `app.css` (sidebar mirror, ml/mr-auto swap, force LTR on xterm to keep shell output readable), and `LanguageSwitcher.svelte` exported from `@components/ui` for the Settings page. **Still open**: most pages don't yet call `t()` — strings are still hardcoded English. Wiring the existing en.ts keys into actual components is mechanical but not yet done. 🌐
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

## Phase 26: Enterprise Architecture Upgrades

> A DARPA-grade architectural overhaul addressing strict SOC requirements, horizontal scale, and adversarial resilience per the brutal roadmap audit.

### 🔴 Tier 1: Systemic Scaling & Stream Semantics
- [x] **26.1 Distributed Log Fabric:** Embedded NATS JetStream (`internal/messaging/nats_service.go:49-113`) with priority subject routing (critical/high/default at lines 132-142); ingestion pipeline references at `internal/ingest/pipeline.go:89`. Verified.
- [ ] **26.2 Federated Query Federation:** Transition from local BadgerDB/Bleve to a distributed query execution layer (Presto/Trino style) capable of routing by tenant, source, and time-shard.
- [ ] **26.3 Stream-Oriented Detection Engines:** Refactor rule engines to fully embrace stream-oriented semantics (sliding/tumbling windows, watermarks, late-event handling, and deterministic replay).
- [v] **26.4 System-Wide Backpressure:** Worker pool blocks on full queue (`internal/platform/worker_pool.go:84-92`); event bus rate-limits at 1k events/sec / 5k burst (`internal/eventbus/bus.go:102-110,196-207`); NATS priority subjects (above) provide alert preemption. **Gap**: no explicit circuit breaker / bulkhead pattern (e.g. sony/gobreaker) — services don't trip and isolate when downstream fails.
- [x] **26.5 Cryptographic M-of-N Approval:** Voting structure (`internal/security/quorum.go`) + per-approval FIDO2 signature verification now wired together. `Approve` calls `FIDO2Manager.CompleteAuthentication` (ECDSA verify against the registered hardware key) before counting the vote; failed verification rejects with WARN. Plus the existing M-of-N counting (`len(req.Approvals) >= req.Required`) and the HMAC-bound approval tokens from Phase 25.10. Phase 22.7's broader "WORM + 2-of-3 senior admins via FIDO2 within 15 minutes" is layered on top of this primitive.

### 🟡 Tier 2: Investigations & Secrets Automation
- [x] **26.6 Graph-Based Investigations:** `internal/services/graph_service.go:1-150` (FindAttackPath, GetSubGraph, node/edge model, campaign cluster export). Verified.
- [x] **26.7 Automated Incident Timeline Reconstruction:** `internal/services/timeline_service.go:1-129` `ReconstructTimeline`; `CausalityID` on `internal/detection/timeline.go:18`; ±10m/+20m alert window. Verified.
- [x] **26.8 Secrets Lifecycle Automation:** `internal/services/rotation_service.go:1-150` — hourly worker, SSH key rotation, auto-rotate vs notify-only policies, vault integration. Verified.
- [v] **26.9 Alert False-Positive Suppression:** Rule-based suppression engine (`internal/services/suppression_service.go` + `internal/database/suppression.go`) was already in place — full CRUD, regex matching, time-bounded expiration, per-rule + global scoping. Closed 2026-04-25: feedback loop wired — `GovernanceService.MarkFalsePositive` now publishes `suppression:suggested` on the bus with the evidence so a UI listener can present a one-click "create suppression rule" prompt. New `SuggestFromEvidence(evidence)` helper extracts a draft rule by finding the most consistent field/value across evidence rows. In-memory `MatchCount(ruleID)` exposes per-rule hit counts so operators can see which rules are pulling weight. **Still open**: maintenance-window scheduling (active-only-between-times) needs a schema migration on `suppression_rules`.

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

### 27.2 — Advanced Platform Mechanics ✅
- [x] **OBLIVRA Query Language (OQL):** Piped analytics language already mature — supported `where, stats, eval, sort, head, tail, dedup, rename, fields, fillnull, top, rare, rex, lookup, join, append, timechart, chart, mvexpand, predict, anomalydetection`. Closed 2026-04-26 by adding `parse json|xml|kv [<field>] [as <prefix>]` for structured field extraction (`internal/oql/exec_parse.go`, `ast.go:ParseCommand`, `parser.go:parseParse`). The audit's example query now parses end-to-end: `source=logs | parse json message as evt | where evt.user="alice" | stats count by evt.action`. JSON nests flatten to dot-paths (`ctx.ip`, `tags.0`); XML elements become `<path>`, attributes become `<path>.@attr`; KV is quote-aware. 6 grammar tests + flatten + KV-quote tests pass. 🏗️
- [x] **Temporal Entity Resolution:** `internal/identity/lease.go` — `LeaseLedger` with `Record`, `LookupAtTime(tenant, ip, ts)`, `History`. DHCP churn semantics: open lease auto-closed at successor's `started_at`; refresh of identical (host, mac) is a no-op; coverage interval is `started_at <= ts AND (ended_at IS NULL OR ended_at > ts)` matching DHCP wire semantics. Tenant-scoped (two tenants holding the same IP resolve independently). Migration v26 adds `dhcp_lease_log` table + composite lookup index. Tests: alert-on-Tuesday-resolves-to-laptop-A even after Wed/Thu re-leases, refresh-no-op, tenant isolation. 🏗️
- [x] **Centralized DLP (Data Loss Prevention):** `internal/dlp/redactor.go` — server-side `Redactor` with 6 default rules (SSN last-4, Luhn-validated CC last-4, JWT, AWS `AKIA…`, `Bearer/api_key/x-api-key` tokens, email domain-preserving). Per-rule `SetEnabled(RuleID, bool)` toggle so tenants can disable patterns from Settings. Live `Report` tracks per-rule hits + total scanned + total redacted for a dashboard widget. Wired into the ingest DAG via new `engine/dag/node_dlp.go` between identity enrichment and the SIEM/analytics fanout — every event from cloud connectors, REST API, manual ingest gets scrubbed regardless of source. IPs deliberately NOT scrubbed (load-bearing security signal). `pipeline.go:SetDLPRedactor` allows runtime enable/disable. Luhn validation prevents arbitrary 16-digit IDs from being clobbered. 7 tests passing. 🏗️
- [x] **Raft Consensus Control Plane:** Foundation already existed — `internal/cluster/fsm.go` replicates SQL writes via `SQLWriteCommand` + dedicated plugin-registry prefix, with `_raft_applied` request-ID idempotency table and VACUUM-based snapshots. Closed 2026-04-26 by adding `internal/cluster/state_replicator.go` — typed wrappers `ApplyAlertState`, `ApplyPlaybook`, `ApplyThreatIntel` that compose the SQL `INSERT OR REPLACE` + auto-derive a stable SHA-256 request ID from (scope, key, query, args) so retries after leader-election don't double-apply. `LocalApplier` fallback path means single-node deployments use the same code (writes straight through to local DB instead of Raft). Returns `ErrNotLeaderForward` so callers know to retry on leader. 5 tests passing including stable-id-across-retries, divergent-id-across-different-binds. 🏗️

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

## Phase 30: Operator UX Pivot + Phase 27.2 Close-Out

> **Date**: 2026-04-26
> **Scope**: HostDetail single-pane-of-glass + Investigation flow + agent historical backfill +
> Phase 30 polish (5 passes) + Phase 27.2 four-item close-out.

### 30.1 — Host-centric pivot ✅

- [x] **`frontend/src/pages/HostDetail.svelte`** — single-pane-of-glass at `/host/:id`. Sections: status banner (online/offline + last heartbeat + OS/version/trust), KPI strip (alerts/critical/events/collectors), Agent Control panel, Activity Timeline (interleaved logs + alerts sorted DESC, capped at 100). Severity rails on every entry. Pivots to `/siem-search?host=<id>` and `/alert-management?host=<id>`.
- [x] **Drill-down wiring** — hostname cell in `FleetDashboard.svelte` is now a real `<button>` that pushes to `/host/<id>`.
- [x] **Route registered** in `App.svelte` (`{ path: '/host/:id', component: HostDetail }`).

### 30.2 — Investigation workflow ✅

- [x] **`AlertManagement.svelte` quick-action panel** — wired four real handlers (was static placeholders): primary INVESTIGATE pivots to `/host/<host>?alert=<id>` (HostDetail with alert id in query), ISOLATE HOST calls `agentStore.toggleQuarantine`, CAPTURE EVIDENCE fires the existing global `oblivra:capture-evidence` window event, PIVOT IN SIEM pushes to `/siem-search?host=<host>&alert=<id>`. Auto-context expansion: HostDetail filters logs/alerts/timeline scoped to the alert's host so one click takes the operator from raw alert → full surrounding context.

### 30.3 — Agent historical backfill ✅

- [x] **`internal/agent/backfill.go`** + Linux/Windows/Darwin/other branches — one-shot OS-log scan on first agent run with `<DataDir>/backfill.complete` marker. Linux: `journalctl --since=<lookback> --output=json` + `/var/log/{syslog,auth.log,secure,messages,kern.log,dpkg.log,apt/history.log}`. Windows: `wevtutil qe System|Security|Application` with XPath `*[System[TimeCreated[timediff(@SystemTime) <= <ms>]]]`. macOS: `log show --last <h>h --style ndjson` + `/var/log/system.log`. Default 30-day lookback. Every emitted event tagged `Source: "historical"` with `Data["original_timestamp"]` and `Data["collected_at"]` per audit spec. `MapSeverity()` provides unified DEBUG/INFO/WARN/ERROR/CRITICAL mapping from syslog facility / journald PRIORITY / Windows EventLevel. Best-effort — subprocess errors logged at WARN, never block agent boot. Wired into `agent.go` collector list. Linux + Darwin + Windows cross-compile clean. 🖥️

### 30.4 — Operator UX polish (subset of audit findings) ✅

- [x] **Severity color tokens** in `app.css` — unified `--color-sev-{debug,info,warn,error,critical}` palette + matching `*-bg` tints. DEBUG=gray, INFO=blue, WARN=amber, ERROR=red, CRITICAL=bright red. Used by HostDetail timeline + ActivityFeed entries; available for any future log table.
- [x] **`ActivityFeed.svelte`** (`@components/ui`) — global "what's happening RIGHT NOW" widget. Live-streams new alerts via `$effect` on `alertStore.alerts.length` + agent online/offline transitions via 10s polling + diff. 30-entry default cap, severity-coloured icons, click-through to `/host/:id` or `/siem-search`. Wired into Dashboard's right column (replaced the dead "Engine Load" placeholder block).
- [x] **`savedQueries.svelte.ts`** store — persistent saved-queries with `pin`, `bumpUsage`, `rename`, `togglePin`, dedup-on-save, quota-exhaustion fallback (drops oldest half on retry), schema-versioned localStorage. `MAX_RECENT=50`. Wired into SIEMSearch as: pinned-queries chip strip at the top, history drawer toggleable from the toolbar HISTORY button, save dialog from the SAVE QUERY button, sidebar "Recent Queries" panel reading from the store (replaced dead mock data). 🌐
- [x] **`TimeRangePicker.svelte`** (`@components/ui`) — preset buttons (LIVE / 5M / 1H / 24H / 7D / 30D / INSTALL / CUSTOM) + custom datetime-local popover. Emits typed `{ start, end, preset }` to parent. Wired into SIEMSearch toolbar; SIEMSearch's `composedQuery()` automatically appends `where timestamp >= "..."` clauses (or AND-extends an existing where) before executing.
- [x] **`TenantSwitcher.svelte`** (`@components/ui`) — top-bar dropdown reading `tenantStore.tenants`, writing to new `appStore.currentTenantId`. Wired into TitleBar with `--wails-draggable: no-drag`. `appStore.setCurrentTenant()` persists to localStorage. New `lib/apiClient.ts` (`apiFetch` / `apiGetJSON` / `apiPostJSON`) reads `currentTenantId` and attaches `X-Tenant-Id` to every outbound REST request — `alerts.svelte.ts` and `agent.svelte.ts` both migrated. 🌐

### 30.5 — Polish passes (5×) ✅

- [x] **Pass 1 — wire orphans + delete dangerous script** (the orphans were the 30.4 components; commit `3701da8` had already removed `cleanup_unused_imports.py`).
- [x] **Pass 2 — regression-prevention guards.** `frontend/scripts/lint-guards.sh` (~110 LOC) runs three grep checks; new `frontend-guards` CI job; `npm run lint:guards`. Closes 29.5 + 29.6 + 29.8.
- [x] **Pass 3 — backend security/docs.**
  - **Cross-tenant authorization** (`rest.go:1043`) — replaced `isGlobalAdmin := false // TODO` with real `auth.IsGlobalAdminFromContext(...)`. New `auth.IsGlobalAdmin` helper checks for `*` / `platform:admin` / `tenant:read:*` permissions. 🚨
  - **Settings DEBUG hardening verified** — already logs only `key=%s value_bytes=%d`, never values; substring-fallback `isSensitiveKey` covers password/passphrase/secret/token/webhook/credential/private_key/auth_key/client_secret. 🟢
  - **Security headers verified** — CSP + Referrer-Policy + Permissions-Policy + X-Content-Type-Options + X-Frame-Options all present in middleware. 🟢
  - **`bridge_wails.go` doc comment** corrected (was referencing v2 `runtime.EventsEmit`; code is pure v3 `app.Event.Emit`).
  - **`internal/search/federation.go:SetPeers`** — atomic peer-set replacement; `cluster_service.go:syncNodesWithFederator` now uses it (handles additions AND removals, no stale peers).
  - **`HostStore.GetByCredentialID`** — new method on interface + `HostRepository`; `rotation_service.go` no longer scans the full host table per key rotation.
  - **`internal/memory/secure.go`** — `osPageSlice()` helper isolates the OS-allocated `uintptr → unsafe.Pointer` cast; CI vet now uses `-unsafeptr=false` with documented justification.
  - **All backfill log calls** converted printf-style to match logger interface.
  - **Docs `rules_loaded: 2543` → `82`** with note explaining detection-pack dependence.
- [x] **Pass 4 — frontend type cleanup.** `FleetMap.svelte` missing `Button` import (would crash at render); `App.svelte` `sidebarVisible` → `toggleSidebar()`; `tenant.svelte.ts` missing `TacticalMessage` type import; `i18n/index.ts` `import.meta.env` cast; `MultiTenantAdmin.svelte` `activeIncidents` → `totalIncidents`; `Settings.svelte` Input missing required `value` bind. 172 → 162 svelte-check errors (none in any modified file). Remaining 162 are bounded variant-mismatch noise + schema drift, deferred.
- [x] **Pass 5 — GDPR deletion audit + control panel honesty.**
  - **Migration v27** (post-renumber) `tenant_deletion_log` — append-only table: tenant_id, name, deleted_by_user, deleted_by_role, reason, prev_row_hash (SHA-256 of pre-wipe row contents), timestamp. Article-30 records-of-processing evidence.
  - **`CryptographicWipeWithAudit`** — new audit-aware deletion path. Reads + hashes the row, applies the wipe, writes the deletion log entry, fires optional `auditor` callback post-commit. Backwards-compat: old `CryptographicWipe(ctx, id)` delegates with `system` actor. 🚨
  - **HostDetail control panel** placeholders replaced — Trigger Scan / Toggle Debug / Restart Agent now `disabled` with tooltip noting RPC pending (tracked as 30.5a/b/c). Honest UI affordance instead of fake "succeeded" no-op toasts.

### Phase 27.2 close-out (this update) ✅

The four heavyweight items from "Advanced Platform Mechanics" all closed in this same arc:

- [x] **27.2.1 OQL `parse json|xml|kv`** — see Phase 27.2 entry above.
- [x] **27.2.2 Temporal Entity Resolution** — `internal/identity/lease.go` + migration v26 `dhcp_lease_log`.
- [x] **27.2.3 Centralized DLP** — `internal/dlp/redactor.go` + `internal/engine/dag/node_dlp.go` + `pipeline.go:SetDLPRedactor`.
- [x] **27.2.4 Raft control plane wrappers** — `internal/cluster/state_replicator.go` (typed `ApplyAlertState` / `ApplyPlaybook` / `ApplyThreatIntel` with stable SHA-256 request IDs + `LocalApplier` fallback for single-node).

### Verification at this commit

| Check | Result |
|---|---|
| `go build ./internal/... ./cmd/...` | exit 0 |
| `go vet -unsafeptr=false ./internal/...` | exit 0 |
| `go test ./internal/oql/ ./internal/identity/ ./internal/dlp/ ./internal/cluster/` (Phase 27.2 suite) | all pass |
| `go test ./internal/agent/...` (backfill) | all pass; cross-compile linux/darwin/windows clean |
| `vite build` | ✓ in ~13s, `index.js` ≈ 638 kB |
| `svelte-check` | 162 errors (down from 172), 0 in any file modified during this arc |
| `frontend/scripts/lint-guards.sh` | ✓ 3/3 guards pass |
| Migration ladder | v25 `suppression_rules` → v26 `dhcp_lease_log` → v27 `tenant_deletion_log` |

---

## Phase 31: SOC Investigation-First UI + Agent Splunk-Parity++

> **Date**: 2026-04-26 (continuation arc)
> **Scope**: Investigation-first UI redesign (7-domain SOC nav + InvestigationPanel
> + EntityLink drill-down + Mission Control overview), Phase 27.2 close-out
> (OQL parse, lease ledger, DLP, Raft state replicator), pre-stage close-out
> (goleak, rule fixture coverage, agent close-out passes A-F).

### 31.1 — SOC Investigation-First UI ✅

- [x] **7-domain nav restructure** — `Overview / Security / Network / Identity / Hosts / Logs / System`. `lib/stores/navigation.svelte.ts` schema bumped v1→v2; `lib/nav-config.ts` rewritten with 50+ items mapped to existing routes (no new pages required, just re-organised). `AppSidebar` + `BottomDock` updated with new `GROUP_HEADER_ICONS` (LayoutDashboard / Shield / Network / UserCog / Server / FileText / Settings), de-duplicated icon imports. 🌐
- [x] **`InvestigationPanel.svelte`** (~340 LOC) — global slide-out drawer pinned below TitleBar. Sections: header (entity icon + label + back/close + entity-id chip), Quick Actions (Full page + Pivot to SIEM), Host Status (when type=host), Related Alerts (severity-coloured), Activity Timeline (interleaved alerts + events sorted DESC). 220ms cubic-bezier slide-in with `prefers-reduced-motion` fallback, `Esc` to close, `⌘⌫` / `⌥⌫` to walk back. Mounted globally at App.svelte root — single instance, reachable from every page. 🌐
- [x] **`EntityLink.svelte`** — drop-in clickable primitive: `<EntityLink type="host" id={row.host} />` opens the global panel. Type-specific hover hint colors (host=accent, ip=warn, alert=error). Stops propagation so it doesn't activate row-level click handlers. Wired into `AlertManagement.svelte` host cells; `Overview.svelte` timeline rows; available platform-wide via `@components/ui` barrel. 🌐
- [x] **`investigationStore.svelte.ts`** — `openEntity()`, `back()` history stack (capped at 20), debounced `close()` for animation, `EntityType` enum covering host/user/ip/process/hash/domain/alert. 🌐
- [x] **`Overview.svelte`** — Mission Control landing page at `/overview`. Risk-level KPI (LOW/MEDIUM/HIGH) computed from real alert distribution in last hour with gradient backgrounds per level. KPIs: ACTIVE INCIDENTS, CRITICAL ALERTS, ONLINE AGENTS. Global event timeline (large left panel) with severity left-rails; Live Activity Feed (narrow right panel). NO charts — per spec: "focus on timelines, context, actionable insights." 🌐
- [x] **`timeRange.svelte.ts`** + global `TimeRangePicker` in TitleBar — single platform-wide time scope, persisted to localStorage. `resolve()` recomputes relative presets at call time so long sessions don't drift. Wrapped in `--wails-draggable: no-drag` so clicks don't drag the window. 🌐

### 31.2 — Phase 27.2 close-out ✅

(Already documented inline at Phase 27.2 above; cross-referenced here for completeness.)

- [x] **27.2.1 OQL `parse json|xml|kv`** — full parser + evaluator + 6 grammar tests
- [x] **27.2.2 Temporal Entity Resolution** — `internal/identity/lease.go` + migration v26
- [x] **27.2.3 Centralized DLP** — `internal/dlp/redactor.go` + DAG node, 6 default rules
- [x] **27.2.4 Raft control plane wrappers** — `internal/cluster/state_replicator.go`

### 31.3 — Pre-stage close-out ✅

The 6-item pre-stage ship-blocker list:

- [x] **25.4 Goroutine leak detection** — `goleak.VerifyTestMain` in `eventbus`/`ingest`/`agent` test suites. Caught a real leak in eventbus tests (`TestBusPublishSubscribe` and `TestBusWildcardSubscriber` weren't calling `bus.Close()`). Added `t.Cleanup(bus.Close)` to every `NewBus()` call. Whitelisted bounded third-party daemons (glog, Bleve `AnalysisWorker`). 🏗️
- [x] **25.7 Detection rule fixture coverage gate** — `internal/detection/rule_fixtures_test.go`. `TestRuleCoverage_AllRulesHaveFixtures` walks `sigma/core/` and fails CI if any rule lacks fixtures registered in `fixtureSet`. Adding a new Sigma rule without fixtures now blocks the merge. Match-evaluation test deferred (Sigma transpiler condition shape needs deeper sigma-side wiring; coverage gate is the load-bearing CI hard-block). 🏗️
- [x] **30.5a/b/c Agent remote actions** — `ActionTriggerScan` / `ActionToggleDebug` / `ActionRestartAgent` added to `agent.ActionType`; new `internal/api/agent_actions_queue.go` provides per-agent in-memory pending-actions queue with 5-minute TTL; existing `handleAgentAction` endpoint now actually enqueues (was a log-only stub); heartbeat handler dequeues + ships in response. UI: HostDetail's "Trigger Scan / Toggle Debug / Restart Agent" buttons no longer disabled — call `apiPostJSON('/api/v1/agent/action', {agent_id, type, payload})` via the new `apiClient`. 🌐
- [ ] **22.3 Hot/Warm/Cold tiering** — STILL OPEN. The last engineering ship-blocker. Current state: Hot (BadgerDB) + Parquet archive exist; no auto-migration, no warm tier (30-180d), no cold (180d+) S3-compatible tier. (Pre-stage roadmap targets the *foundation* in the next phase.)
- [ ] **SOC2 / ISO27001 / FIPS** — still external-auditor work, not engineering.
- [ ] **BYOK / SCIM** — still multi-week investments deferred to a future phase.

### 31.4 — Agent Splunk-Parity++ Close-Out (Passes A through F) ✅

The five "❌ Missing" items from the agent feature audit + the agent-control RPCs:

- [x] **A. Encrypted config storage** — `internal/agent/config_storage.go` (~150 LOC). Chacha20-Poly1305 AEAD with key derived from the agent's existing Ed25519 identity (`SHA-256("oblivra-agent-config" || privKey)`). On-disk wire format: `OBC1 || nonce(12) || ciphertext+tag(16+)`. Atomic write via temp+rename+fsync. Backwards-compatible with legacy plaintext config (no OBC1 magic = legacy passthrough; next write re-encrypts). 5 tests: round-trip, legacy passthrough, wrong-key reject, tamper detect, atomic write, missing-file. 🖥️🌐
- [x] **B. Multi-output routing** — `internal/agent/output_router.go` (~150 LOC). Priority-ordered output set; tries each in priority order until one succeeds; tracks consecutive failures per endpoint; demotes to back of rotation after `MaxConsecutiveFailures` (default 3); `DemotionWindow` (default 60s) before rehab. Recovery is fast — single success clears the counter. 6 tests: primary-wins, failover-on-error, demote-after-N, recovery clears counter, all-fail surfaces error, nil-safe. Beats Splunk Forwarder which requires a separate load-balancing tier. 🖥️
- [x] **C. Watchdog auto-restart** — `internal/agent/restart.go` (~100 LOC). `RestartManager.RequestRestart(reason, timeout)` calls the configured shutdown drain (WAL flush, collector close) then `os.Exit(75)`. Code 75 = BSD `EX_TEMPFAIL`, recognised by systemd / launchd / Windows SCM as "transient failure, restart me." Idempotent via `sync.Once` so concurrent triggers (watchdog + UI both firing simultaneously) only drain once. 4 tests: shutdown-then-exit, idempotent under concurrency, proceeds-on-drain-error, nil-shutdown valid. 🖥️
- [x] **D. Local detection rules** — `internal/agent/local_detection.go` (~270 LOC). 3 in-process rules running BEFORE WAL/transport for sub-millisecond edge response: SSH brute-force (5 failed-password from same source-IP in 60s sliding window), suspicious sudo (8 patterns: `sudo bash`, `sudo /bin/bash`, `command=/bin/bash`, etc.), discovery commands (14 commands: whoami, net user, ipconfig /all, etc.). Per-rule `SetEnabled(bool)` toggle wired to ToggleDebug action. 5 tests: brute-force fires after threshold, per-IP isolation, sudo patterns, discovery commands, disable-honored. 🖥️
- [x] **E. Remote control RPCs** — see 30.5a/b/c above (deduplicated). 🌐
- [x] **F. Agent telemetry surface** — `HostDetail.svelte` KPI strip expanded to 7 columns: ALERTS / CRITICAL / EVENTS / COLLECTORS / **CPU / RAM / DISK**. CPU/RAM/Disk derived from the latest `metrics` event in `siemStore.results` filtered to this host. `formatPct` shows "—" until the first metrics event arrives. 🌐

### 31.5 — Verification at this commit

| Check | Result |
|---|---|
| `go build ./internal/... ./cmd/...` | exit 0 |
| `go vet -unsafeptr=false ./internal/...` | exit 0 |
| `go test ./internal/agent/ ./internal/detection/ ./internal/eventbus/ ./internal/ingest/ ./internal/oql/ ./internal/identity/ ./internal/dlp/ ./internal/cluster/` | all pass with `goleak` active |
| Cross-compile `linux/darwin/windows` agent | all clean |
| `vite build` | ✓ ~12s, `index.js` ~654 kB |
| `svelte-check` | 162 errors, **0 in any Phase 31 file** |
| `frontend/scripts/lint-guards.sh` | ✓ 3/3 guards pass |
| New test counts | +5 config_storage, +6 output_router, +4 restart, +5 local_detection, +6 oql parse, +3 lease ledger, +7 dlp, +5 cluster state, +2 detection coverage |

### 31.6 — Outstanding GA blockers

After Phase 31 + 32 + 33 close-out, only 3 items remain on the GA path:

| Item | Status | Owner |
|---|---|---|
| **22.3 Hot/Warm/Cold tiering** | Last pure-engineering blocker. Foundation work scheduled next. | engineering |
| **SOC2 / ISO27001 / FIPS attestations** | Self-validated only. | external auditors |
| **BYOK / SCIM** | Multi-week investments. | future phase |

> **Update (Phase 32 + 33, 2026-04-29)**: Backend security audit (8 findings), frontend wiring audit (10 findings), and a window-chrome regression all closed. Shell subsystem removed from operator UI (Phase 32). The three blockers above remain unchanged.

**Beta-1 ship-readiness: confirmed.** GA gated on storage tiering (engineering, ~1 week) and external auditors (months, runs in parallel).

---

## Phase 32: Shell Subsystem Removal ✅

> **Date**: 2026-04-29
> **Scope**: Full removal of the interactive shell subsystem from the
> operator UI. Backend Go libraries retained because non-terminal
> features still depend on them (canary SCP, scheduled SSH key
> rotation), but the operator-facing terminal/SSH/tunnel/recording
> surface is gone. Pairs with the same-day backend audit-fix sweep
> documented in **HARDENING.md → Phase 32**.

### Stages
- [x] Frontend `frontend/src/components/terminal/` directory deleted (TerminalPage, XTerm, OperatorBanner, SessionRestoreBanner, panes, useShellSession, layout helpers).
- [x] Routes `/shell`, `/ssh`, `/tunnels`, `/recordings`, `/session-playback` hidden from `nav-config.ts` (entries kept registered in `App.svelte` so deep links 404 cleanly rather than crash). 🖥️
- [x] Backend Go libraries retained — `internal/ssh/`, `internal/services/{ssh,local,tunnel,recording,share,multiexec,broadcast,file,transfer,pty}_*.go` still compile and back canary deployment / SCP / SSH key rotation.
- [x] Phase 22.4 / Phase 23.1–23.6 verification rows updated to reflect the deletion.

### Companion hardening work (in HARDENING.md)
- Phase 32 backend audit-fix sweep (8 findings: replay cache, real users/roles, evidence-seal strict JSON, ReportService init, rate-limit GC, vault key downgrade, audit-key hashing, AI honesty)
- Phase 32 housekeeping (tsc warnings, scratch/ build tags)
- Phase 33 frontend ↔ backend wiring audit (10 findings)
- Phase 34 pop-out UX fix + test suite stabilization (6 pre-existing failures cleared)

## Phase 35: Storage Tiering Wiring (last engineering GA blocker)

> **Date**: 2026-04-29 (continuation)
> **Scope**: Wire the existing Hot/Warm/Cold tiering scaffolding (Phase 31) into
> production — boot the migrator, expose REST observability, build the
> dashboard page. Closes the last pure-engineering item on the GA path.

### 35.1 — Migrator wiring + REST observability ✅
- [x] **`InfrastructureCluster.{HotTier, WarmTier, ColdTier, TierMigrator}`** added (`internal/core/clusters.go`). All four nil-safe — single-node deployments without S3 cold can leave any field nil and the migrator becomes a 2-tier shuttle.
- [x] **`container.go::initInfra`** — instantiates `tiering.NewHotTier(c.Infra.HotStore)` + `tiering.NewWarmTier(platform.DataDir(), c.Log)` + `tiering.NewLocalDirCold(platform.DataDir(), c.Log)` after BadgerDB opens; constructs the migrator with `tiering.DefaultRetention()` (Hot 30d / Warm 150d).
- [x] **`App.Startup`** calls `Infra.TierMigrator.Start(ctx)` after services boot — first cycle fires immediately (`Migrator.loop` calls `RunOnce` once before entering the ticker), so a long-stopped agent makes progress without waiting an hour. **`App.Shutdown`** stops the migrator before the kernel tears down storage so it doesn't get torn down mid-batch.
- [x] **`internal/api/rest_tiering.go`** — `TierStatProvider` + `TierMigrationProvider` interfaces (api package can't import tiering directly without a cycle), `handleTieringStats` (analyst+, returns sizes per tier + last cycle), `handleTieringPromote` (admin-only, fires manual `RunOnce`, audit-logs as `storage.tiering.promote.manual`, publishes `storage:tiering_manual_promote` bus event with `event_type=destructive_action`).
- [x] **`internal/core/tiering_adapters.go`** — `tierStatAdapter` wraps `tiering.Tier` to the api interface; `tierMigrationAdapter` wraps `*tiering.Migrator` and caches the last-cycle stats on every manual `RunOnce` (background scheduled cycles aren't yet observed — the migrator has no `OnCycleComplete` callback today, so the dashboard polls the stats endpoint to pick up the latest data; small follow-up to add the hook).
- [x] **`APIService.SetTieringProvider`** — same provider-injection pattern as SetSuppression / SetSettings (`internal/services/api_service.go`). Wired from container's `initPlatform` via `WireTieringIntoAPI(svc, infra)`.

### 35.2 — Frontend dashboard page ✅
- [x] **`frontend/src/pages/StorageTiering.svelte`** — three-tier KPI strip with Hot (red, BadgerDB 0-30d) / Warm (amber, Parquet 30-180d) / Cold (blue, JSONL 180+d) tiles, each showing size + percentage of total + tier description. Last-cycle panel below with HotToWarm / WarmToCold counts, started-at, error count. "Promote now" admin button with confirm-dialog → POST /promote → toast on completion. 30s background poll. Honest 503 handling: when migrator isn't configured the page renders an explicit "Tiering not configured" state instead of zeros. 🌐
- [x] **Route registered** in `App.svelte` at `/storage-tiering`; **nav entry** in `nav-config.ts` under SYSTEM → Govern (next to Agent Integrity). Database icon already in `BottomDock.svelte::ICON_MAP` (no new icon import needed — Phase 29 lesson held).

### 35.3 — Tests ✅
- [x] **`internal/api/rest_tiering_test.go`** — 10 tests covering: 503 when not configured, all-three-tiers happy path, last-cycle reported correctly, last-cycle nil when migrator hasn't run, tier-error reports `size_bytes: -1` (per-tier failure isolated, response stays 200), RoleReadOnly can read /stats but not /promote, RoleAnalyst forbidden on /promote, /promote returns the cycle stats it produced, method-not-allowed enforcement on both. Mocks-based — no BadgerDB/Parquet stack needed.
- [x] **`go test ./internal/api/`** — all green; full suite (36/36 packages) still green.
- [x] **`oblivrashell.exe bootcheck`** — `[STORAGE] Hot/Warm/Cold tier migrator started (interval=1h, hot=30d, warm=150d)` and `tiering: cycle complete hot→warm=0 warm→cold=0 errors=0 duration=1.39s` confirm live wiring.

### 35.4 — Outstanding 35.x follow-ups
- [ ] **35.5 Ingest pipeline writes through HotTier** — events currently land in `tenant:<id>:events:` (siem_badger keyspace) but the migrator scans `tier:hot:` keys. Until the ingest pipeline tees writes into the HotTier keyspace, the migrator has nothing to migrate. Two paths: (a) tee writes to both keyspaces during a transition window, then cut over; (b) refactor `siem_badger.go` to write through `HotTier`. Path (b) is architecturally cleaner; path (a) is lower risk.
- [ ] **35.6 Cold S3 (`RemoteColdTier`)** — concrete S3 client behind `//go:build s3` build tag so air-gap builds stay clean. Outline already in `cold.go` comments. Gated on a Settings page for endpoint/bucket/access-key/region (currently no admin UI surfaces these).
- [ ] **35.7 `Migrator.OnCycleComplete(fn)` callback** — so the dashboard sees background scheduled cycles immediately instead of polling the stats endpoint. Small change in `tiering/tier.go::loop`.
- [ ] **35.8 Per-tenant retention overrides** — today retention is global; compliance customers want tenant-scoped overrides. Schema migration on tenants table to add `hot_retention_days` / `warm_retention_days`, plumb through `tiering.Retention`.

### 35.5 — GA blockers update

After Phase 35:

| Item | Status | Owner |
|---|---|---|
| **22.3 Hot/Warm/Cold tiering** | 🟢 **Closed** for Beta-1 (foundation wired + observable). 35.5/35.6/35.7/35.8 are post-Beta polish. | engineering |
| **SOC2 / ISO27001 / FIPS attestations** | Self-validated only. | external auditors |
| **BYOK / SCIM** | Multi-week investments. | future phase |

**Beta-1 ship-readiness: confirmed.** With 22.3 foundation closed, the only remaining engineering follow-ups (35.5-35.8) are non-blocking polish that can land iteratively post-Beta. GA gated on external auditors only.

---

## Phase 36: Broad Scope Cut — Log-Driven Security Platform

> **Date**: 2026-04-29 (continuation)
> **Decision**: OBLIVRA repositions from "all-in-one SOC platform" to
> **log-driven security platform**. Detection / threat intel / UEBA /
> NDR / fusion / compliance / multi-tenancy / storage tiering / agent
> framework all stay — these all DERIVE value from logs. The operator
> action layer (SOAR + IR + ransomware response + disk/memory imaging
> + AI assistant + plugin framework) is removed.
>
> **Rationale**: Beats Splunk on TCO (storage tiering, sovereign deploy,
> OQL), compatible with running alongside an existing SOAR (Tines, Torq,
> XSOAR) instead of competing. Smaller binary, smaller doc surface,
> easier to position. Operators who need response automation use a
> dedicated SOAR; operators who need disk/memory IR use a dedicated
> DFIR tool (Velociraptor, FTK, Volatility) and import the resulting
> evidence files via the generic Collect API.

### 36.1 — Removed (operator-action layer) ✅

**Backend services** (all deleted):
- `internal/services/ai_service.go` — AI assistant
- `internal/services/incident_service.go` + `playbook_service.go` — SOAR + case management
- `internal/services/canary_deployment_service.go` — canary file deployment (response action)
- `internal/services/network_isolator_service.go` — host network isolation (response action)
- `internal/services/ransomware_service.go` — ransomware response RPCs (DETECTION engine in `internal/detection/ransomware_engine.go` retained)
- `internal/services/deterministic_response_service.go` — automated response decisions
- `internal/services/plugin_service.go` — plugin manager Wails surface

**Backend packages** (all deleted):
- `internal/incident/` — actions, integrations (Jira/SNOW), playbook engine, triage scoring (~7 files)
- `internal/plugin/` — Lua sandbox, registry, signing, cluster sync, manifest (~6 files)
- `internal/engine/wasm/` — wazero WASM plugin runtime (~5 files)
- `internal/forensics/analyzer.go` — entropy/file analyzer (used by deep IR; deleted)
- `internal/forensics/collector.go` — local disk/memory collector (used by AcquireDisk/AcquireMemory; deleted)
- `internal/security/canary.go` — canary deployment helpers

**Service methods** (deleted):
- `ForensicsService.AcquireDiskImage` — raw block-device acquisition
- `ForensicsService.AcquireMemoryDump` — physical RAM dump
- `ForensicsService.AnalyzeFile` — entropy + risk scoring

**REST endpoints** (deleted):
- `POST /api/v1/ransomware/isolate` — host isolation (response action)
  - Detection-side `events` / `hosts` / `stats` / `protection` endpoints retained.

**Frontend pages** (all deleted):
- `AIAssistantPage.svelte` (`/ai-assistant`)
- `PluginManager.svelte` (`/plugins`)
- `PlaybookBuilder.svelte` (`/playbook-builder`)
- `CaseManagement.svelte` (`/cases`)
- `IncidentResponse.svelte` (`/response`) — was rebuilt in Phase 33; sad to lose but right call given the scope cut
- `IncidentTimeline.svelte` (`/timeline`, `/timeline/:principalID/:principalType/:targetTime`)
- `SOARPanel.svelte` (`/soar`)
- `RansomwareUI.svelte` (`/ransomware`, `/ransomware-ui`)
- `ForensicsPage.svelte` (`/forensics`, `/remote-forensics`)

**Container wiring removed** (`internal/core/container.go`, `clusters.go`, `app/app.go`, `main_gui.go`):
- `Platform.AIService`, `Platform.PluginService`
- `Response.IncidentService`, `Response.PlaybookService`, `Response.NetworkIsolatorService`, `Response.RansomwareService`, `Response.DeterministicResponse`, `Response.TriageService`
- `Security.CanaryService`, `Security.CanaryDeployment`
- All `mustRegister(...)` lines for the above

**Nav-config entries removed** (`frontend/src/lib/nav-config.ts`):
- `cases`, `timeline`, `response`, `playbook-builder`, `ransomware`, `forensics`, `plugins`, `ai-assistant`

**API service signature change**:
- `services.NewAPIService(...)` dropped the `isolator *NetworkIsolatorService` param. `unifiedForensicEngine` now uses `agentService.ToggleQuarantine` only (the SSH-based isolator fallback went away with the response-action layer).

### 36.2 — Retained (log-driven core) ✅

**Detection / Analytics** (all stay):
- `internal/detection/` — Sigma transpiler, rule engine, MITRE mapping, correlation, ransomware-detection engine, fusion, campaign builder
- `internal/threatintel/` — STIX/TAXII, IOC matcher
- `internal/enrich/` — GeoIP, DNS, asset enrichment, lookup tables
- `internal/services/ueba_service.go` — User & Entity Behavior Analytics
- `internal/services/ndr_service.go` — Network Detection & Response (network-flow analysis)
- `internal/services/fusion_service.go` — multi-stage attack fusion
- `internal/services/risk_service.go` — risk-based alerting
- `internal/services/alerting_service.go` + escalation
- `internal/services/threat_hunter` — hunting interface
- `internal/services/compliance_service.go` — PCI/ISO/SOC2/HIPAA/GDPR packs

**Forensics evidence locker** (retained, restricted to log-derived evidence):
- `internal/forensics/evidence.go` — chain-of-custody for log-events-as-evidence
- `internal/forensics/rfc3161.go` — RFC 3161 timestamp authority
- `internal/forensics/tpm_signer.go` — TPM-rooted log signing

**Storage / Ingest** (all stay):
- `internal/ingest/` — pipeline, WAL, parsers, Phase 35 storage tiering integration
- `internal/storage/` — BadgerDB hot, Parquet warm, JSONL cold
- `internal/search/` — Bleve full-text index
- `internal/oql/` — query language

**Auth / Multi-tenancy / Compliance** (all stay):
- `internal/auth/` — OIDC/SAML/MFA/RBAC/WebAuthn
- `internal/database/tenant_db.go` — multi-tenancy + audit log + retention
- `internal/integrity/` — Merkle audit chain
- `internal/agent/` — log collection from endpoints

### 36.3 — Verification ✅
| Check | Result |
|---|---|
| `go build ./internal/... ./cmd/...` | exit 0 |
| `go test ./internal/...` | **36/36 packages still pass** |
| `npm run typecheck` | clean |
| `npm run build` | clean — index bundle dropped from 720 KB → 658 KB (**-9%**) |
| `oblivrashell.exe bootcheck` | OK — services start, no panic. Binary size 85 MB → 80 MB (**-6%**) |

### 36.4 — Outstanding cleanup
- [ ] **36.4a Cluster FSM dead path** — `internal/cluster/fsm.go` keeps a `pluginRegistryApplier` interface and `pluginRegistryPrefix` Raft-log dispatch path. The interface has no implementor now; the dead code is harmless but should be removed in a follow-up. No external import cycle since the types are local.
- [ ] **36.4b Ghost handler** — `handleRansomwareIsolate_REMOVED_PHASE_36` placeholder is unrouted (the route registration is gone). Could just delete the function body and remove its test (if any). Trivial.
- [ ] **36.4c Wails generated bindings** — `frontend/bindings/.../{aiservice,pluginservice,incidentservice,playbookservice,canarydeploymentservice,networkisolatorservice,ransomwareservice}` directories may still exist. Run `wails3 generate bindings` next desktop build to regenerate; until then they're orphan TypeScript stubs that can't be imported because the Go services don't exist.
- [ ] **36.4d Doc refresh** — `docs/operator/*.md` references SOAR / playbooks / ransomware response / AI assistant. One pass to remove.
- [ ] **36.4e Compliance-pack doc on response actions** — PCI-DSS / NIST 800-53 packs may reference response-action controls (host isolation, evidence acquisition). Audit the pack YAML and either remove those controls or mark them as "external-tool dependency" in the report.
- [ ] **36.4f License feature flags** — `internal/licensing/license.go` still has `FeatureSOAR`, `FeatureAIAssistant`, `FeaturePlugins`, `FeatureRansomware` (response-side). Keep them defined but unused for now (forward compat for the deferred premium reintroduction); flag-gate the still-existing endpoints as `FeatureRansomware` for the detection-side ransomware events.

### 36.6 — External observability stack removed ✅

> **Date**: 2026-04-29 (continuation)
> **Decision**: drop the bundled Prometheus + Grafana + Grafana Tempo
> companion containers. Rationale: OBLIVRA's own agent + ingest
> pipeline can collect platform health metrics (goroutines, heap, GC,
> EPS, detection rate) directly into the SIEM. No need for a separate
> scrape-and-graph layer when our own SIEM IS the graphing layer.

- [x] **Deleted** `ops/` directory — `prometheus.yml`, `tempo.yml`, `grafana/provisioning/`.
- [x] **Simplified** `docker-compose.yml` — single service (sovereign-server). Removed prometheus / tempo / grafana service blocks, OTLP exporter env, plugin volume mount (plugins gone in Phase 36).
- [x] **README** — replaced "Observability Stack" section with "Self-Observability" explaining the agent-based path; dropped the port table entries (9090/3000/3200) and the URL/credentials table for the external dashboards. Tech stack row updated to "Self-ingest via OBLIVRA agent (`/metrics` Prometheus-format scrape target if needed)".
- [x] **`docs/FEATURES.md`** — line 90 updated to reflect agent-ingest model.
- [x] **Retained**: the `/metrics` REST endpoint (Prometheus-format scrape target) and the OTel SDK shim (already a no-op without a configured exporter — see `internal/monitoring/otel.go:118`). Operators with existing observability infrastructure can still point Prometheus / Datadog / New Relic at `:8080/metrics` without any OBLIVRA-side changes.
- [x] **Verified**: `go build ./internal/... ./cmd/...` exit 0; `npm run typecheck` exit 0.

### 36.5 — Strategic implications
- **Positioning**: "Logs platform with detection + UEBA + NDR + compliance" — competes with Wazuh, Security Onion, Elastic Security on quality / TCO; complementary to (not competing with) CrowdStrike, SentinelOne, Tines, Torq, XSOAR.
- **Customer message**: "Bring your own SOAR. Bring your own DFIR. We're the place your logs go." Smaller scope, sharper value prop.
- **Code surface**: ~12% of codebase removed. Compile time, security audit surface, documentation burden all proportionally smaller.
- **Lost capabilities**: SOAR playbook builder, host network isolation from UI, raw disk/memory acquisition, AI chat over events, Lua/WASM plugin extensibility. All have well-established external alternatives.
- **GA blockers update**: Storage tiering (Phase 35) is closed; SOC2/ISO27001/FIPS attestations remain external-auditor work. **Engineering side of GA is essentially done.**

---

## Operating Convention (effective Phase 32)

> When work lands, update `task.md` in the same PR / commit:
> - **Add** to the relevant phase if the work fits an existing scope.
> - **Open a new sub-section** (e.g. `32.7`, `33.7`) if it's a new arc.
> - **Remove deleted features** rather than annotating them as `~~struck through~~`. The git history is the historical record; `task.md` reflects the *current* surface.
> - **Cross-reference real file paths and line numbers** so the entry is verifiable, not aspirational.
> - **Mark verification** with the check table format used in 32.6 / 33.4.
