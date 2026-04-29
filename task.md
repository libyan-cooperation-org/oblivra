# OBLIVRA — Phase Tracker (Build Roadmap)

> **What this file is**: the single source of truth for the chronological build narrative — every phase, its goal, and current status.
>
> **What lives elsewhere**:
> - [`HARDENING.md`](HARDENING.md) — security findings, fixes, postmortems, and hardening gates
> - [`ROADMAP.md`](ROADMAP.md) — long-term vision (CSPM, K8s, vuln mgmt)
> - [`RESEARCH.md`](RESEARCH.md) — DARPA/NSA-grade research items
> - [`BUSINESS.md`](BUSINESS.md) — certifications, legal, GTM
> - [`FUTURE.md`](FUTURE.md) — cross-cutting (chaos engineering, deception, i18n)
> - [`STRATEGY.md`](STRATEGY.md) — Phase 22 strategic rationale

**Last audited**: 2026-04-29
**Current positioning**: Sovereign, log-driven security and forensics platform.

---

## Status Tiers
- `[x]` = **Production-Ready** (hardened, documented, soak-tested)
- `[v]` = **Validated** (functionally correct, tested under load)
- `[s]` = **Scaffolded** (code exists and compiles)
- `[ ]` = Not started

---

## Platform Architecture — Golden Rule

> **Desktop = Sensitive + Local + Operator Actions**
> **Web = Shared + Scalable + Multi-user**

### Desktop (Wails App) — Must live here
- Vault & secrets management (AES-256, OS keychain, FIDO2/YubiKey)
- Local/offline operations and air-gap mode
- Agent build/signing and certificate generation
- Local forensics evidence handling

### Web (Browser UI) — Must live here
- Fleet-wide SIEM search and observability
- Alerting, escalation, and notifications
- Central detection rule management
- Threat hunting and investigation tools
- Fleet management and agent oversight
- Multi-tenancy and RBAC

### Hybrid (Both)
- Search, detection rules, dashboards, and alerts (scoped differently per surface)

**Never on Web**: vault master key, agent signing keys, local filesystem access.

---

## Core Platform Features (Current State)

### Security & Vault
- [x] AES-256 encrypted Vault with OS keychain integration
- [s] FIDO2 / YubiKey support
- [x] TLS certificate generation
- [x] Security key management UI

### Storage, Ingestion & Search
- [x] BadgerDB (hot) + Bleve (search) + Parquet (warm) + JSONL (cold)
- [x] Crash-safe Write-Ahead Log (WAL)
- [x] Syslog, JSON, CEF, Windows EVTX, Linux journald parsers
- [s] High-throughput ingestion pipeline (benchmarked; sustained-load soak test pending)
- [x] OQL (Oblivra Query Language) with pipe syntax
- [x] Storage tiering migrator (Hot → Warm → Cold)

### Detection & Analytics
- [x] Sigma rule engine + transpiler (82+ rules)
- [x] MITRE ATT&CK mapping and heatmap
- [x] Correlation and multi-stage fusion engine
- [x] UEBA (behavioral baselines + peer group analysis)
- [x] NDR (NetFlow, DNS tunneling, JA3)
- [x] Ransomware behavioral detection (entropy-based — detection only; response actions removed Phase 36)
- [x] Risk-based alerting

### Forensics & Integrity
- [x] Merkle-tree chained audit logging
- [x] RFC 3161 timestamping
- [x] Evidence locker with chain-of-custody
- [x] Temporal entity resolution (DHCP lease tracking)
- [x] Centralized DLP redactor

### Agent Framework
- [x] Lightweight Go agent with gRPC + mTLS + zstd
- [x] File tailing, Windows Event Log, journald, metrics, FIM
- [x] Offline buffering (local WAL) + edge filtering
- [x] Agentless collectors (WMI, SNMP, REST polling)
- [x] Encrypted config storage, multi-output routing, watchdog

### Enterprise
- [x] Multi-tenancy with strong data isolation
- [x] OIDC/SAML + MFA + granular RBAC
- [x] Audit log evidence pack (pair with external compliance tooling — Drata/Vanta/Tugboat — for attestation)

### UX & Productivity
- [x] Hybrid desktop/web architecture with context guards
- [x] Investigation-first UI (HostDetail, InvestigationPanel, EntityLink)
- [x] Multi-monitor pop-out windows + workspace save/restore
- [x] Command palette, notification center, unified time range picker

---

## Phase History (Condensed)

### Phase 0–0.5: Foundation & Stabilization ✅
Core service registry, desktop/web context separation, web MVP, accessibility, and architectural hardening.

### Phase 1: Core Storage + Ingestion + Search ✅
BadgerDB, Bleve, Parquet archival, WAL, high-performance ingestion pipeline, and OQL foundation.

### Phase 2: Alerting + REST API ✅
Detection rules, alerting with escalation, REST API with RBAC, real-time streaming.

### Phase 3: Threat Intel + Enrichment ✅
STIX/TAXII, IOC matching, GeoIP, DNS, asset mapping, advanced log parsers.

### Phase 4: Detection Engineering + MITRE ✅
Sigma rules, correlation engine, MITRE heatmap, rule testing framework.

### Phase 5–6: Stability & Audit Evidence ✅
Memory bounds, incident lifecycle, Merkle audit logging, evidence locker, RFC 3161 timestamping.

### Phase 7: Agent Framework ✅
Full agent + agentless collection, encrypted config, multi-output, local detection rules.

### Phase 10: UEBA & Behavioral Analytics ✅
User/entity baselines, Isolation Forest, peer group analysis, multi-stage fusion.

### Phase 11: Network Detection & Response (NDR) ✅
NetFlow/IPFIX, DNS analysis, JA3, lateral movement detection.

### Phase 12: Enterprise Capabilities ✅
Multi-tenancy isolation, HA foundations, identity & RBAC, data lifecycle management.

### Phase 15: Sovereignty ✅
Offline updates, signature verification, zero-internet mode.

### Phase 17–21: Detection Quality & Scaling ✅
Full Sigma support, OQL enhancements, partitioned pipeline, query limits, rule routing.

### Phase 22: Productization & Reliability ✅
Multi-tenant isolation, reliability engineering (chaos testing, reconnect, degradation handling), storage tiering foundation.

### Phase 23: Desktop UX & Windowing ✅
Frameless window chrome, multi-monitor pop-outs, workspace save/restore, notification center.

### Phase 27: Advanced Platform Mechanics ✅
OQL `parse` commands, temporal entity resolution, centralized DLP, Raft control plane improvements.

### Phase 30–31: Investigation-First UI ✅
HostDetail page, global InvestigationPanel, EntityLink, 7-domain navigation, ActivityFeed, forensic backfill.

### Phase 32: Shell Subsystem Removal ✅
Interactive terminal/SSH/SFTP subsystem removed from UI (backend libraries retained for non-terminal use).

### Phase 35: Storage Tiering Wiring ✅
Hot/Warm/Cold migrator fully wired, REST API, dashboard, and observability.

### Phase 36: Scope Cut — Log-Driven Forensics Platform ✅
**Major repositioning**. Removed: SOAR, incident response playbooks, ransomware response actions, disk/memory imaging, AI assistant, plugin framework, external observability stack (Prometheus/Grafana/Tempo), and compliance YAML packs + evaluator + PDF/HTML report generator (36.x).
Focus narrowed to high-integrity log collection, detection, UEBA, NDR, and forensic evidence handling.

---

## Current Open Work (Post-Phase 36)

### Phase 37: Log Forensics Core (In Progress)
- [ ] Log gap and anti-forensic activity detection (within OBLIVRA's own evidence chain)
- [ ] Enhanced EVTX and journald deep parsing (top 30 EVTX event IDs + top 20 journald units; expanded per-quarter)
- [ ] Unified forensic timeline view with severity rails
- [ ] Basic Evidence Package export (events + timeline + Merkle proof)
- [ ] Forensic search templates in OQL
- [ ] Trust-tier (TE/VE/BE) enforcement in search/export pipelines

### Phase 38: Court Admissibility Layer
- [ ] Full Forensic Evidence Package (PDF/HTML + signatures + verification instructions)
- [ ] Evidence Verification Portal (offline CLI verifier)
- [ ] WORM mode for warm/cold tiers (Windows ReFS integrity streams + Linux `chattr +i`)
- [ ] Templated narrative builder for investigation reports (no LLM)
- [ ] Expanded chain-of-custody UI in EvidenceVault
- [ ] Legal review gate before claiming admissibility

### Phase 39: Advanced Log Forensics
- [ ] Process lineage and command-line reconstruction from logs
- [ ] Authentication/session reconstruction
- [ ] Entity Forensic Profile tab (Host/User/IP)
- [ ] Tampered/deleted log indicators (within OBLIVRA's own evidence chain — not host-filesystem scanning)
- [ ] Expert witness export package

### Phase 22.3: Storage Tiering Polish (Carry-over)
- [ ] Ingest pipeline writes through HotTier interface
- [ ] Cold tier S3 support (build-tagged)
- [ ] Per-tenant retention overrides
- [ ] Cross-tier integrity verification

### Immediate Hygiene
- [ ] Final Phase 36 cleanup (dead FSM paths, Wails bindings regeneration, docs refresh)
- [ ] Update `README.md` and `FEATURES.md` with new log forensics positioning
- [ ] Create `docs/operator/log-forensics.md`

---

## Strategic End State

OBLIVRA is not a traditional SIEM.

It is a **cryptographically verifiable forensic log system** capable of reconstructing logged system activity across time, storage tiers, and organizational boundaries — with explicit gap markers where telemetry was unavailable or tampered.

---

## Next Milestone — Beta-1

- Stable high-integrity ingestion pipeline (with documented sustained-load soak test)
- Verified detection and correlation engine
- Forensic timeline reconstruction with explicit gap markers
- Functional evidence export with cryptographic proofs and offline verifier
- Stable multi-tenant isolation

---

**Last updated**: 2026-04-29
