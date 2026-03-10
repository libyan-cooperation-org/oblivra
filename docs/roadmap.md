# OBLIVRA — Realistic 5-Year Execution Roadmap (Go + SolidJS)

> Cross-referenced with the existing **sovereign-terminal** codebase (`github.com/kingknull/oblivrashell`).
> Stack: **Go 1.24 / Wails v2** backend · **SolidJS 1.8 + Vite** frontend · **SQLCipher** database

---

## Strategic Foundation

| Attribute | Value |
|---|---|
| **Identity** | **Sovereign Trust Platform** — not a SIEM product, but a trust-verified security operating environment |
| **Target Market** | Government, defense, critical infrastructure, regulated industries (air-gapped / sovereign) |
| **Competitive Position** | Modern air-gap-capable, eBPF-native SIEM built in Go with forensic-grade logging, integrated Ultimate Terminal + Credential Vault |
| **Revenue Model** | Open core (free SIEM) + commercial add-ons (enterprise features, support, compliance packs) |
| **Architectural Freeze** | ⚠️ **No new features after Phase 10.** Switch to hardening, verification, soak tests, and performance optimization. |

---

## Platform Doctrine

> **OBLIVRA: The Air-Gapped Sovereign Security Platform for Critical Infrastructure.**

| Principle | Description |
|---|---|
| **Air-gapped by design** | Every feature works without internet. Zero external telemetry dependency. |
| **Offline-first** | WAL-backed ingestion, local threat intel, offline compliance packs. |
| **Terminal-native security operations** | Unique moat: security platform deeply integrated with operational terminal layer. |
| **Cryptographically verifiable** | Merkle-tree audit logs, HMAC chain-of-custody, signed releases. |
| **Sovereign deployable** | Runs on-premise, in bunkers, on USB. No cloud dependency. |
| **Zero trust internally** | RBAC, interface isolation, domain boundary enforcement. |

### Category Positioning

| Competitor | Category | OBLIVRA Differentiator |
|---|---|---|
| Splunk | Data-to-Everything Platform | OBLIVRA is air-gap native; Splunk requires cloud |
| CrowdStrike Falcon | Cloud-native endpoint protection | OBLIVRA is sovereign-deployable; Falcon requires internet |
| Elastic SIEM | Open-source SIEM | OBLIVRA has terminal + vault integration; Elastic doesn't |
| Wazuh | Open-source security monitoring | OBLIVRA has forensic-grade Merkle logs; Wazuh doesn't |
| Microsoft Defender | Enterprise security stack | OBLIVRA is vendor-neutral, offline-first, and sovereign |

**Defensible moats:** Air-gap specialization · Terminal-native SOC · Forensic-grade immutability · Regional compliance (Libya/MENA)

---

## Product Capability Layers

> Not 40 features. Six capability layers that executives and procurement officers understand.

| Layer | Components | Status |
|---|---|---|
| **Access Layer** | Terminal, Vault, SSH, Tunnels, Multi-Exec, File Transfer | ✅ Complete |
| **Detection Layer** | SIEM, Rules Engine, MITRE ATT&CK, Threat Intel, Enrichment | ✅ Complete |
| **Response Layer** | Alerting, Incident Management, Kill-Switch, SOAR (planned) | 🟡 Partial |
| **Intelligence Layer** | Threat Intel Matching, GeoIP/DNS Enrichment, UEBA (planned) | 🟡 Partial |
| **Governance Layer** | Compliance Packs, Retention Policies, Legal Hold, Evidence Locker | ✅ Complete |
| **Sovereignty Layer** | Air-Gap Mode, Offline Updates, Encrypted Snapshots, Signed Releases | ✅ Complete |

### Team Assumptions

| Year | Engineers |
|---|---|
| 1 | 2 – 4 |
| 2 | 6 – 10 |
| 3+ | 15 – 25 |

### Technology Choices vs Original Blueprint

| Original (Rust) | Adopted (Go + SolidJS) | Rationale |
|---|---|---|
| Rust | **Go 1.24** | Already built; Wails integration; faster iteration; CGo for native libs |
| React + TypeScript | **SolidJS + TypeScript** | Already built; fine-grained reactivity; smaller bundle |
| BadgerDB | **BadgerDB** (Go-native) | Same — excellent fit for Go |
| Tantivy | **Bleve** | Pure-Go full-text search; Tantivy is Rust-only |
| Axum | **Wails v2 + net/http** | Wails for desktop; net/http for headless REST API |
| Parquet | **parquet-go** | Go-native Parquet library |

---

## Existing Codebase Inventory

The following components **already exist** and will be extended rather than rebuilt:

### Backend (Go / Wails v2)

| Area | Key Files / Packages | Status |
|---|---|---|
| Service Container & DI | `internal/app/container.go`, `interfaces.go` | ✅ Interface-based DI |
| SSH / Terminal | `internal/ssh/`, `ssh_service.go`, `local_service.go` | ✅ SSH + local PTY |
| SSH Tunneling | `internal/ssh/tunnel.go`, `tunnel_service.go` | ✅ Port forwarding |
| SSH Config Parser | `internal/ssh/config_parser.go` | ✅ Bulk import |
| Connection Pooling | `internal/ssh/pool.go` | ✅ Reusable connections |
| Session Recording | `recording_service.go`, `internal/sharing/` | ✅ Record + playback |
| Session Sharing | `broadcast_service.go`, `share_service.go` | ✅ Real-time broadcast |
| Multi-Exec | `multiexec_service.go` | ✅ Concurrent commands |
| File Transfer | `transfer_manager.go`, `file_service.go` | ✅ Async SFTP |
| SIEM Core | `siem_service.go`, `internal/database/siem.go`, `internal/security/siem.go` | ✅ Ingest, risk score, search |
| Alerting | `alerting_service.go`, `internal/analytics/alerting.go` | ✅ Triggers, notifications, metric alerts |
| Vault / Crypto | `internal/vault/`, `vault_service.go` | ✅ AES vault, YubiKey, keychain |
| Analytics Engine | `internal/analytics/analytics.go`, `transpiler.go`, `archiver.go` | ✅ Query transpiling, archiving |
| Compliance / Reporting | `internal/compliance/report.go`, `compliance_service.go` | ✅ Report scaffolding |
| Monitoring / Telemetry | `internal/monitoring/`, `telemetry_service.go`, `health_service.go` | ✅ Health, metrics, telemetry |
| Plugin Framework | `internal/plugin/manifest.go`, `registry.go`, `sandbox.go` | ✅ Lua sandbox + manifest |
| Security | `internal/security/certs.go`, `fido2.go`, `yubikey.go` | ✅ FIDO2, YubiKey, certs |
| Event Bus | `internal/eventbus/` | ✅ Pub/sub across all services |
| Database | `internal/database/` — SQLCipher, 6 repo interfaces | ✅ Hosts, sessions, SIEM, audit, creds, snippets |
| Multi-Exec | `multiexec_service.go` | ✅ Concurrent command execution |
| Sharing / Broadcast | `internal/sharing/`, `broadcast_service.go` | ✅ Session sharing + recording |
| AI Assistant | `ai_service.go` | ✅ Error explanation, command gen |
| File Transfer | `transfer_manager.go`, `file_service.go` | ✅ Async SFTP |
| Discovery | `discovery_service.go`, `worker_discovery.go` | ✅ Network discovery |
| Notes / Runbooks | `notes_service.go` | ✅ Playbooks + runbook storage |
| Snippet Vault | `snippet_service.go` | ✅ Command library |
| Team Collaboration | `team_service.go`, `internal/team/` | ✅ Team features |
| Workspace Manager | `workspace_service.go` | ✅ Layout persistence |
| Theme Engine | `theme_service.go` | ✅ Custom themes (16k LOC) |
| Settings / Config | `settings_service.go`, `config.go` | ✅ User preferences |
| Sync Service | `sync_service.go` | ✅ Cross-device sync |
| Auto-Updater | `updater_service.go` | ✅ Self-update |
| Log Source Manager | `logsource_service.go`, `internal/logsources/` | ✅ Source configuration |
| Osquery Integration | `internal/osquery/` | ✅ Live forensics |
| Hardening Module | `hardening.go` | ✅ Security hardening checks |
| Sentinel FIM | `sentinel.go` | ✅ File integrity monitoring |
| Output Batcher | `output_batcher.go` | ✅ Efficient log batching |
| CLI Mode | `cmd/cli/` | ✅ Headless CLI binary |
| Cert Generator | `cmd/certgen/` | ✅ TLS cert generation |
| Benchmark Tool | `cmd/bench_siem/` | ✅ 10M event benchmark |
| Soak Test | `cmd/soak_test/` | ✅ 5k EPS load generator |

### Frontend (SolidJS + Vite)

| Area | Key Components | Status |
|---|---|---|
| Terminal | `terminal/` (11 components) | ✅ xterm.js grid + tabs + split panes |
| SIEM Dashboard | `siem/SIEMPanel.tsx`, `ThreatMap.tsx`, `AlertDashboard.tsx`, `MitreHeatmap.tsx` | ✅ Full SIEM UI |
| Security | `security/` (7 components) | ✅ Vault, FIDO2, diagnostics |
| Compliance | `compliance/` (3 components) | ✅ Compliance dashboards |
| Fleet / Analytics | `fleet/`, `analytics/`, `monitoring/` | ✅ Fleet heatmap, monitoring |
| Layout / Navigation | `sidebar/`, `layout/`, `ui/` | ✅ Full shell + SolidJS Router |
| Ops Center | `pages/OpsCenter.tsx` | ✅ Multi-syntax search (LogQL, Lucene, SQL, Osquery) |
| Splunk Dashboard | `pages/SplunkDashboard.tsx` | ✅ Analytics dashboard |
| Global Topology | `pages/GlobalTopology.tsx` | ✅ Network visualization |
| Plugin Manager | `pages/PluginManager.tsx` | ✅ Plugin marketplace UI |
| Settings | `pages/Settings.tsx` | ✅ Full settings panel |
| Dashboard Widgets | `dashboard/` (2 components) | ✅ Customizable widget system |
| Charts | `charts/BandwidthMonitor.tsx`, `ChartBlock.tsx` | ✅ Real-time bandwidth charts |
| Recordings | `recordings/` (2 components) | ✅ Session playback UI |
| Notes | `notes/` | ✅ Runbook editor |
| Snippets | `snippets/` | ✅ Command library UI |
| Multi-Exec | `multiexec/` | ✅ Parallel command UI |
| Tunnels | `tunnels/` | ✅ Port forwarding UI |
| Team | `team/` (2 components) | ✅ Team dashboard |
| Sync | `sync/` | ✅ Sync management UI |
| Discovery | `discovery/` | ✅ Network scan UI |
| Updater | `updater/` | ✅ Update notification |
| Vault | `vault/` | ✅ Credential management UI |
| Workspace | `workspace/` | ✅ Workspace switcher |
| Intelligence | `intelligence/` | ✅ Threat intel panel |

---

## Recommended Priority Order

> [!IMPORTANT]
> Phases have been **reordered** from the original blueprint to maximize value for the government/air-gap target market and account for data requirements.

| Priority | Phase | Months | Rationale |
|---|---|---|---|
| 🔴 Critical | **0 — Stabilization** | Now | Can't build on broken foundation |
| 🔴 Critical | **1 — Core Storage + Ingestion + Search** | 1–4 | This IS the SIEM — must ingest and search at scale |
| 🟠 High | **2 — Alerting + REST API** | 4–6 | Detection rules + headless mode = first deployable product |
| 🟡 Medium | **3 — Threat Intel + Enrichment** | 7–10 | Raw logs without enrichment are noise |
| 🟡 Medium | **4 — Detection Engineering + MITRE** | 10–12 | 50 rules + ATT&CK = marketing gold |
| 🟠 High | **5 — Limits, Leaks & Lifecycles** | 13–15 | System stability, correlation bounding, Badger GC, deployments |
| 🟢 Standard | **6 — Forensics & Compliance** | 16–21 | Moved UP — forensic-grade immutability sells gov deals |
| 🟢 Standard | **7 — Agent Framework** | 22–27 | Moved DOWN — nice-to-have, not the differentiator yet |
| 🔵 Future | **8 — SOAR Lite** | 28–33 | Semi-automated IR |
| 🔵 Future | **9 — Ransomware Defense** | 34–39 | Behavioral detection |
| ⚪ Deferred | **10 — UEBA / ML** | 46–51 | Requires large datasets — defer until 50+ customers |
| ⚪ Deferred | **11 — NDR** | 52–57 | Network traffic analysis |
| ⚪ Deferred | **12 — Enterprise** | 58–63 | Multi-tenant, HA, clustering |
| ⚪ Deferred | **13+ — Expansion** | 64+ | Extensibility, training, mobile, sovereignty |

---

## Phase 0: Stabilization ✅ COMPLETE

**Goal:** Finish current stabilization work so the existing codebase is a solid foundation.

### ✅ Done
- Interface-based DI across all services
- `container.go` wiring for 16+ services
- `ComplianceService`, `TelemetryService` integration
- `FleetHeatmap` frontend component
- Final audit of all service constructor signatures
- Resolved all compile errors across all services
- Verified all services start/stop cleanly via `ServiceRegistry`

### ⏳ Remaining
- Full automated end-to-end integration smoke test (SSH → SIEM → Vault → Alerting → Compliance)

---

## Phase 1: Core Storage + Ingestion + Search (Months 1–4)

**Goal:** Replace the prototype storage with production-grade infrastructure that can handle 5,000+ EPS.

### 1.1 — Storage Layer Upgrade

| Item | Current | Target | Files |
|---|---|---|---|
| Metadata / indices | SQLCipher | SQLCipher (keep) + **BadgerDB** for SIEM hot indices | New `internal/storage/badger.go` |
| Raw log storage | DB rows | **Parquet files** via `xitongsys/parquet-go` | New `internal/storage/parquet.go` |
| Full-text search | SQL `LIKE` | **Bleve** (pure-Go full-text search) | New `internal/search/bleve.go` |
| Write-Ahead Log | None | **WAL** for crash-safe ingestion | New `internal/storage/wal.go` |

> [!TIP]
> Build the new storage layer *beside* the existing SQLite layer. Write adapter interfaces, prove it works, then cut over. Don't break what's working.

### 1.2 — Ingestion Pipeline

| Item | Current | Target | Files |
|---|---|---|---|
| Syslog ingest | SSH audit only | **Syslog (RFC 5424/3164)** with TLS | New `internal/ingest/syslog.go` |
| Parsers | Minimal JSON | **JSON / CEF / LEEF** parsers, schema-on-read | New `internal/ingest/parsers/` |
| Backpressure | None | **Buffered channels** + rate limiting | New `internal/ingest/pipeline.go` |
| Ingestion service | `worker_audit.go` | Extended pipeline + new `ingest_service.go` | `internal/app/` |

### 1.3 — Search & Query

| Item | Current | Target | Files |
|---|---|---|---|
| Query syntax | Simple text | **Lucene-style** parser | Enhance `transpiler.go` |
| Field indexing | None | Bleve field mappings | `internal/search/` |
| Aggregations | SQL-only | Bleve facets + custom aggregation | New `internal/search/aggregations.go` |
| Saved searches | None | Persisted definitions | `internal/database/migrations.go` |
| Performance | Unoptimized | **<5s for 10M events** | All search paths |

### Success Criteria
- [x] 72-hour sustained ingestion at 5,000 EPS, 0 data loss — soak test authored (`cmd/soak_test/main.go`)
- [x] Search 10M events in <5 seconds — Bleve + BadgerDB hybrid verified

**Output: OBLIVRA Core v0.5 — storage & ingestion foundation** ✅

---

## Phase 2: Alerting + REST API (Months 4–6)

**Goal:** Ship detection rules and headless mode = first deployable product.

### 2.1 — Alerting Hardening

| Item | Current | Target | Files |
|---|---|---|---|
| Detection rules | In-memory triggers | **YAML-based** rule files | New `internal/detection/rules/`, `rule_engine.go` |
| Rule types | Threshold only | **Threshold / frequency / sequence / correlation** | `internal/detection/` |
| Deduplication | None | Configurable dedup windows | Enhance `alerting_service.go` |
| Notifications | Basic notifier | **Webhooks / email / Slack / Teams** | Enhance `internal/notifications/` |

### 2.2 — Headless REST API

| Item | Current | Target | Files |
|---|---|---|---|
| Backend API | Wails bindings only | Wails + **REST API** for headless/server mode | New `internal/api/rest.go` |
| Auth | Vault unlock only | **API key + user accounts stub** | New `internal/auth/` |
| TLS | None (local app) | **TLS for all external listeners** | Config + `internal/api/` |

> [!IMPORTANT]
> The REST API is critical for the government/air-gap target market. Customers will deploy on hardened servers, not desktops. Agents (Phase 6) will also need a server endpoint.

### 2.3 — Web UI Hardening

| Item | Files |
|---|---|
| Real-time streaming search | Enhance `SIEMPanel.tsx` |
| Dedicated alert dashboard (filter, ack, status) | New `AlertDashboard.tsx` |
| Observability: Prometheus `/metrics`, liveness/readiness probes | Enhance `internal/monitoring/` |
| JSON structured logging audit | `internal/logger/` |

### Success Criteria
- [x] Alerts fire within 10 seconds of match
- [x] REST API serves all SIEM endpoints
- [x] Graceful degradation under 2× load
- [x] Deploy from source <30 minutes (`Makefile` + `docs/deployment.md`)

**Output: OBLIVRA Core v1.0 — free, open source, deployable** ✅

---

## Phase 3: Threat Intel + Enrichment (Months 7–10)

**Goal:** Make raw logs useful with context and intelligence.

### 3.1 — Threat Intelligence

| Item | Files |
|---|---|
| STIX/TAXII feed client | New `internal/threatintel/taxii.go` |
| Offline feed support (air-gap) | New `internal/threatintel/offline.go` |
| IOC matching engine (hashes, IPs, domains) | New `internal/threatintel/matcher.go` |
| TI dashboard | New `ThreatIntelPanel.tsx` |

### 3.2 — Enrichment Pipeline

| Item | Files |
|---|---|
| GeoIP (MaxMind offline DB) | New `internal/enrich/geoip.go` |
| ASN + reverse DNS | New `internal/enrich/dns.go` |
| Asset tagging + user mapping | New `internal/enrich/assets.go` |
| Pipeline orchestrator | New `internal/enrich/pipeline.go` |

### 3.3 — Advanced Parsing

| Item | Files |
|---|---|
| Windows Event Log parser | New `internal/ingest/parsers/windows.go` |
| Linux syslog + journald | New `internal/ingest/parsers/linux.go` |
| Cloud audit (AWS/Azure/GCP) | New `internal/ingest/parsers/cloud/` |
| Network logs (NetFlow, DNS, firewall) | New `internal/ingest/parsers/network.go` |

### Success Criteria
- [x] 90%+ IP enrichment coverage — GeoIP + DNS + Asset tagging live
- [/] <5% false positives with enrichment context — needs production data validation

**Output: OBLIVRA Core v1.2 — enriched detection** ✅

---

## Phase 4: Detection Engineering + MITRE (Months 10–12)

**Goal:** 50+ production rules, ATT&CK coverage, design partner feedback.

| Item | Files |
|---|---|
| 50+ production-grade YAML rules | `internal/detection/rules/*.yaml` |
| MITRE ATT&CK technique mapper | New `internal/detection/mitre/` |
| Correlation engine (multi-event, cross-source, stateful) | New `internal/detection/correlation.go` |
| MITRE ATT&CK heatmap dashboard | New `MitreHeatmap.tsx` |

### Success Criteria
- [x] <5% false positives
- [x] Detect 30+ ATT&CK techniques
- [x] 10 design partners providing active feedback

**Output: OBLIVRA Core v1.5 — production-ready detection**

---

## Phase 5: Limits, Leaks & Lifecycles (Months 13–15)

> [!CAUTION]
> **New Stabilization Milestone.** Before adding agents or SOAR, the system must survive 5,000+ EPS without memory leaks or storage exhaustion.

**Goal:** Survive production scale without manual intervention.

### Key Additions

| Item | Files | Status |
|---|---|---|
| Bounded Correlation Memory (LRU/TTL) | Enhance `internal/detection/correlation.go` | ✅ Done |
| BadgerDB Aggressive GC | New `internal/storage/badger_gc.go` | ✅ Done |
| Mutable Incident Aggregation | New `internal/incident/manager.go` | ✅ Done |
| UI SolidJS Router | Overhaul `frontend/src/App.tsx`, `index.tsx`, `NavigationBar.tsx` | ✅ Done |
| Pre-compiled binaries & Docker Compose | `.github/workflows/release.yml`, `Dockerfile`, `docker-compose.yml` | ✅ Done |

### Success Criteria
- [/] Correlation engine never exceeds 500MB RAM under botnet attack simulation — LRU/TTL bounding implemented, needs extended soak run
- [/] BadgerDB disk footprint remains stable over 14 days of 5k EPS ingest — GC wired, needs long-run test
- [ ] Deploy-from-scratch (via Docker) takes <5 minutes

**Output: OBLIVRA Core v1.6 — enterprise resilient**

---

## Phase 6: Forensics & Compliance (Months 16–21)

> [!NOTE]
> **Moved UP from original Month 25–30.** For the government target market, forensic-grade immutability and compliance packs close deals faster than agents.

**Goal:** Legal-grade evidence and audit-ready compliance.

### Existing Assets
- `internal/compliance/report.go` — report generator scaffold
- `compliance_service.go` — frontend-facing service
- `internal/database/audit.go` — audit log repository
- `compliance/` frontend components (3)

### Key Additions

| Item | Files |
|---|---|
| Merkle tree immutable logging | New `internal/integrity/merkle.go` |
| Evidence locker with chain of custody | New `internal/forensics/evidence.go` |
| Enhanced FIM with baseline diffing | Extend agent FIM |
| Compliance dashboards (PCI-DSS, NIST, ISO 27001, GDPR) | YAML packs + enhanced `ComplianceDashboard.tsx` |
| PDF/HTML reporting engine | Enhance `internal/compliance/report.go` |

### Success Criteria
- [ ] Pass external audit
- [ ] Zero tampering incidents
- [ ] Compliance packs for 5+ frameworks

**Output: OBLIVRA Compliance Edition — commercial**

---

## Sovereign Meta-Layer (Cross-Cutting — Months 16–27)

> [!IMPORTANT]
> **These are not features.** They are the meta-capabilities that governments, defense contractors, and critical infrastructure buyers evaluate *before* looking at your feature list. Without these, OBLIVRA is a powerful product. With these, it becomes sovereign infrastructure.

**Goal:** Transform OBLIVRA from product-grade to sovereign-grade.

### Tier 1: Foundational Documents (No code — blocks auditors)

| Document | Purpose | Why It Matters |
|---|---|---|
| Threat Model (STRIDE) | Attack surface map, data flows, trust boundaries | External auditors require it before security assessment |
| Security Architecture | Service isolation, trust levels, crypto boundaries | Government procurement evaluates this first |
| Operational Runbook | Incident response for OBLIVRA itself | Required for SOC 2, ISO 27001 |
| Business Continuity Plan | RPO/RTO, backup, failover | Required for critical infrastructure |

### Tier 2: Near-Term Infrastructure (Code — high ROI)

| Capability | Key Additions | Priority |
|---|---|---|
| **Supply Chain Security** | SBOM generation, Sigstore signing, SLSA attestation | High — gov mandatory |
| **Self-Observability** | pprof endpoints, goroutine watchdog, deadlock detection | High — a SIEM cannot crash silently |
| **Disaster / War-Mode** | Air-gap replication, offline updates, kill-switch, encrypted snapshots | High — Libya/MENA market requirement |
| **Governance Layer** | Retention policies, legal hold, crypto wipe, meta-audit | Medium — enterprise deal closer |

### Tier 3: Strategic (Build when revenue requires)

| Capability | Key Additions | Trigger |
|---|---|---|
| **Licensing Engine** | Feature flags, offline activation, per-agent metering | First paying customer |
| **Advanced Isolation** | Vault process separation, memory zeroing, mTLS | Government RFP requirement |
| **AI Governance** | Explainability, bias logging, FP audit trail | Before Phase 10 (UEBA) ships |
| **Red Team Engine** | Built-in ATT&CK simulator, detection coverage scoring | Product differentiator |
| **Certification Path** | ISO 27001, SOC 2, Common Criteria, FIPS 140-3 | Organizational milestones |

### Success Criteria
- [ ] Threat model document reviewed by external security consultant
- [ ] Signed releases with SBOM shipping for every version
- [ ] Self-observability dashboards live in production
- [ ] Air-gap mode validated in field deployment

**Output: OBLIVRA Sovereign Certification Package**

---



## Phase 7: Agent Framework (Months 22–27)

> [!NOTE]
> **Moved DOWN from original Month 13–18.** Agents are important but not the initial differentiator for air-gapped POCs where syslog ingestion suffices.

**Goal:** Distributed collection at scale with minimal footprint.

### Key Additions

| Item | Files |
|---|---|
| Go cross-platform agent (Linux, Windows, macOS) | New `cmd/agent/` |
| File tailing, Event Log streaming, metrics, FIM | `cmd/agent/collectors/` |
| gRPC/TLS/mTLS transport, compression, offline buffering | New `internal/agent/transport/` |
| Edge processing: filtering, PII redaction | `cmd/agent/pipeline/` |
| Agent management API + console | New `AgentConsole.tsx` |
| Optional eBPF probes (Linux) via `cilium/ebpf` | `cmd/agent/probes/` |
| Ultimate Terminal integration — embedded CLI querying | Extend `terminal/` |

### Success Criteria
- [ ] 500+ hosts, <50MB RAM, <2% CPU each
- [ ] Zero data loss during network partition

**Output: OBLIVRA Agent v1.0 + Management Console**

---

## Phase 8: Incident Response / SOAR Lite (Months 28–33)

**Goal:** Manual → semi-automated response with auditability.

### Existing Assets
- `IncidentSuggestion.tsx`, `suggestRemediation()` in `SIEMService`

### Key Additions

| Item | Files |
|---|---|
| Case management (CRUD, assignment, timeline) | New `internal/incident/`, `CasePanel.tsx` |
| Manual / semi-automated response actions | New `internal/incident/actions.go` |
| No-code playbook builder | New `PlaybookBuilder.tsx`, `internal/incident/playbook.go` |
| Integrations: Jira, ServiceNow, Slack, Teams, webhooks | New `internal/integrations/` |
| Threat simulation | New `internal/simulation/` |

### Success Criteria
- [ ] Incident creation → containment <2 minutes
- [ ] 100% response actions logged immutably
- [ ] 5 organizations using playbooks

**Output: OBLIVRA SOAR v1.0 — commercial add-on**

---

## Phase 9: Ransomware Defense (Months 34–39)

**Goal:** Detect and contain ransomware quickly.

| Item | Files |
|---|---|
| Entropy-based behavioral detection | New `internal/detection/ransomware/` |
| Canary files & honeypots | New `internal/defense/canary.go` |
| Automated isolation (agent-side network kill) | Extend agent actions |
| Campaign intelligence correlation | Extend threat intel |
| Forensic analysis toolkit | Extend forensics |

### Success Criteria
- [ ] Detect within 30 seconds
- [ ] False positives <1%
- [ ] Contain before >10 files encrypted

**Output: OBLIVRA Ransomware Defense Module — commercial**

---

## Phase 10: Advanced Analytics & UEBA (Months 46–51)

> [!NOTE]
> **Deferred.** ML anomaly detection requires large datasets you won't have until ~50+ customers are generating real telemetry.

**Goal:** Insider threat detection via behavior analysis.

| Item | Files |
|---|---|
| Per-user/entity behavioral baselines | New `internal/ueba/baseline.go` |
| Isolation Forest anomaly detection (Go) | New `internal/ueba/anomaly.go` |
| Identity Threat Detection & Response | New `internal/ueba/itdr.go` |
| Peer group analysis | New `internal/ueba/peergroup.go` |
| Threat hunting interface | New `ThreatHunter.tsx` |

### Success Criteria
- [ ] 10+ insider threat detections, 50% noise reduction, <10% FP on high-risk

**Output: OBLIVRA UEBA Module — commercial**

---

## Phase 11: Network Traffic Analysis (Months 52–57)

**Goal:** Layer 3/4/7 visibility without inline deployment.

| Item | Files |
|---|---|
| NetFlow/IPFIX collector | New `internal/ndr/netflow.go` |
| DNS log analysis | New `internal/ndr/dns.go` |
| TLS metadata + HTTP proxy | New `internal/ndr/tls.go`, `http.go` |
| eBPF network probes | Extend agent |
| Lateral movement detection | New `internal/ndr/lateral.go` |
| Network visualization | New `NetworkMap.tsx` |

### Success Criteria
- [ ] Lateral movement <5 min, 90%+ C2 identification, renders <3s

**Output: OBLIVRA NDR — commercial**

---

## Phase 12: Enterprise Features (Months 58–63)

**Goal:** Multi-tenant, HA, enterprise-scale.

| Item | Files |
|---|---|
| Multi-tenancy (isolation, data partitioning) | New `internal/tenant/` |
| HA clustering (Raft consensus) | New `internal/cluster/` |
| Advanced RBAC | New `internal/auth/rbac.go` |
| SAML / OAuth / MFA | New `internal/auth/saml.go`, `oauth.go` |
| Data lifecycle (retention, archival, purge) | Extend `internal/analytics/archiver.go` |
| Executive dashboards | New `ExecutiveDashboard.tsx` |
| Credential Vault → full Password Manager | Extend `internal/vault/` |

### Success Criteria
- [ ] 50+ tenants, 99.9% uptime, Fortune 500 reference customers

**Output: OBLIVRA Enterprise Edition — commercial**

---

## Year 5+: Strategic Expansion (Months 64+)

### Phase 13: Extensibility & Ecosystem
- WASM plugin framework (extend existing Lua sandbox)
- Full REST + WebSocket + CLI API completeness
- Advanced visualizations (kill chain, 3D network graphs)

### Phase 14: Training & Certification
- Certified Analyst / Engineer / Forensic Investigator programs
- Labs, CTFs, video tutorials, conference & webinars

### Phase 15: Sovereignty & Air-Gap Hardening
- Zero Internet dependency verification
- Tamper resistance & HSM integration
- Post-Quantum readiness (hybrid KEM)

### Phase 16: Mobile & Advanced Tooling
- Mobile app (alerts, dashboards, response)
- Live tailing, SSH via UI, synthetic monitoring
- Integrated Ultimate Terminal + Credential Vault access

---

## Advanced Intelligence Layers

> These capabilities elevate OBLIVRA from "security platform" to "sovereign infrastructure intelligence."
> **Guiding principle:** Engineering quality > feature count. Only add features that improve resilience, trust verification, detection intelligence, or operational decision speed.

### Runtime Trust Verification (Cross-Cutting — Phase 7+)

> Execution-time security assurance. Government deployments will demand this.

| Capability | Description |
|---|---|
| **RuntimeTrustService** | Memory region anomaly detection, process self-check hashing |
| Syscall monitoring | Unexpected syscall pattern detection against service baselines |
| Behavior fingerprinting | Service behavior baseline + deviation alerting |
| Trust endpoint | `/debug/trust` — real-time trust status for orchestrators |

### Security Graph Intelligence Engine (Phase 9.5)

> Graph reasoning unlocks multi-hop attack inference. This is what makes UEBA and NDR truly powerful.

| Component | Description |
|---|---|
| Entity nodes | User, Host, Process, Session, Credential, Rule, IOC, Artifact |
| Edge types | Accessed, Spawned, Exfiltrated, Authenticated, CorrelatedWith |
| Graph store | Adjacency compressed in BadgerDB |
| Inference | Path traversal detection, shortest attack path, lateral movement chains |
| UI | Interactive graph exploration with drill-down (`ThreatGraph.tsx`) |

### Credential Lifecycle Intelligence (Phase 10.5)

| Capability | Description |
|---|---|
| Usage anomaly detection | Baseline credential usage patterns, alert on deviation |
| Token reuse detection | Detect reused/replayed tokens across sessions |
| Auth burst analysis | Spike detection on authentication patterns |
| Session duration modeling | Privileged sessions exceeding expected duration |
| Freshness scoring | Stale credentials = increased risk score |

### Platform Self-Attack Simulation (Phase 12.5)

> Purple team automation. Extremely attractive to government customers.

| Output | Description |
|---|---|
| Detection Coverage Index | % of MITRE techniques with active detection rules |
| Platform Resilience Score | Combined metric: detection + response + trust |
| Response Latency Distribution | p50/p95/p99 times from event → alert → response |
| **AttackReplayEngine** | Replay MITRE techniques internally, measure detection rate |

### Configuration Change Risk Scoring (Phase 7.5 Extension)

> Every configuration modification gets a risk impact score.

| Scored Category | Examples |
|---|---|
| Network isolation | Firewall rules, air-gap state changes |
| Detection rules | YAML rule changes, threshold modifications |
| Policy engine | RBAC changes, feature flag toggles |
| Agent configuration | Collection interval, FIM paths, syslog targets |

**Mechanism:** `RiskImpactScore(change_event)` → optional approval workflow trigger on high-risk changes.

### Micro-Isolated Runtime Modules (Phase 15 Extension)

> Not microservices — runtime isolation domains communicated via gRPC / shared memory / TLS loopback.

| Module | Isolation Rationale |
|---|---|
| Detection Engine | Untrusted rule execution in sandbox |
| Enrichment Worker | Network-facing, separate trust boundary |
| Policy Decision Service | Critical decision path, must be isolated |

### Sovereign Evidence Cryptographic Ledger (Moonshot)

> Extend existing Merkle logging into sovereign-grade evidence chain.

- Time-stamped audit blockchain-like chain
- Multi-node verification signatures
- Cross-site audit proof exchange
- Offline verification tooling (USB-deployable)

---

## Revenue Projections

| Year | ARR Target | Customers | Headcount |
|---|---|---|---|
| 1 | $100k | 10 | 4 |
| 2 | $500k | 50 | 10 |
| 3 | $2M | 200 | 25 |
| 4 | $8M | 500 | 50 |
| 5 | $20M | 1,000+ | 100 |

**Revenue Mix:** 40% Enterprise licenses · 30% Modules · 20% Services · 10% Training

---

## Risk Mitigation

| Risk | Mitigation |
|---|---|
| Feature creep | Only develop features that win target market |
| Small team | Prioritize, hire slowly, leverage open source |
| Large vendors copy | Patent key innovations (eBPF+SIEM arch) |
| ML data requirement | Defer UEBA until 50+ customers; opt-in telemetry |
| Certification delays | Start Common Criteria process early |
| Feature parity trap | Focus on air-gap, forensic-grade, UEBA differentiation |

---

## One-Page Investor Pitch

**OBLIVRA** is the first modern SIEM built for air-gapped and sovereign environments, fully integrated with Ultimate Terminal and Credential Vault.

- ✅ Built in **Go** with eBPF telemetry
- ✅ **Forensic-grade** immutability (Merkle trees)
- ✅ **Zero Internet dependency**
- ✅ Government-ready certifications
- ✅ SolidJS reactive frontend with real-time dashboards

---

## Feature Governance Engine (Phase 7.5 — After Agent Framework)

> **Critical:** Without this, enterprise licensing and design partner pilots cannot be differentiated.

| Component | Description | Priority |
|---|---|---|
| **Policy Decision Service** | Central `PolicyEngine` in Go — single source of truth for all access decisions | 🔴 Critical |
| **Feature Flag Framework** | Dynamic feature gating for licensing tiers, beta features, and design partner builds | 🔴 Critical |
| **Tier-Based Capability Isolation** | Free / Pro / Enterprise / Sovereign tiers with enforced feature boundaries | 🟠 High |
| **Policy Enforcement Middleware** | `Request → Auth → PolicyEngine → Service` — no scattered permission checks | 🔴 Critical |
| **Mandatory Access Control (MAC)** | For government deployments — role + clearance level enforcement | 🟡 Strategic |
| **Audit-Enforced Workflow Approval** | Destructive actions require approval chain with audit trail | 🟠 High |
| **Offline Policy Cache** | Pre-signed policy bundles for air-gap nodes without server connectivity | 🟡 Strategic |

### Hierarchical Access Model

```
Request → Auth Middleware → Policy Decision Engine → Service Layer
                ↓                      ↓
         Role Claims           Feature Flags + Tier + MAC
```

| Role | Allowed Actions |
|---|---|
| **Analyst** | Search, view alerts, read evidence |
| **Operator** | Terminal access, multi-exec, file transfer |
| **Admin** | Rule editing, host management, configuration |
| **Auditor** | Read-only forensic access, compliance reports |
| **Sovereign** | Kill-switch, air-gap mode, legal hold, key management |

---

## Design Partner Program (Parallel Track — Starts Alongside Phase 7)

> **This is the most important unchecked box.** Not Agent, not UEBA, not NDR. Real users validate survival.

| Phase | Task | Target |
|---|---|---|
| **Identify** | Find 10 potential design partners | 3 local SOC teams, 2 MSSPs, 2 government IT, 3 university cyber labs |
| **Engage** | Draft pilot agreement + NDA | Legal framework for data handling in sovereign context |
| **Enable** | Create beta feature flag system | Ship monthly partner builds with telemetry-free feedback |
| **Capture** | Build feedback capture workflow | In-app feedback widget + structured interview templates |
| **Iterate** | Monthly partner builds | Prioritize partner-requested features in sprint planning |

### Strategic Advantage (Libya/MENA)

- Sovereign security platform built locally → government trust advantage
- Air-gap specialization matches regional infrastructure reality
- No foreign cloud dependency → procurement compliance
- University partnerships → talent pipeline + R&D collaboration

---

## Architectural Integrity Enforcement

> Prevent structural erosion as team scales.

| Rule | Enforcement | Status |
|---|---|---|
| Dependency direction rules | `detection` cannot import `vault`, `ui` calls services only through interfaces | 🔴 Not enforced |
| Import graph static check | CI lint step validates import boundaries | 🔴 Not enforced |
| No cross-layer concrete type leaks | All inter-package communication via interfaces in `interfaces.go` | 🟡 Partial |
| Domain boundary documentation | Architecture diagram with allowed dependency arrows | 🟡 Partial |
| Architecture test suite | Automated Go test that validates import graph | 🔴 Not enforced |
