# OBLIVRA — TASK TRACKER (Execution Roadmap)

> **Purpose**: This file defines the *execution backlog* required to reach Beta-1 and beyond.
> It is strictly **action-oriented** and aligned with OBLIVRA's identity as a
> **log-native forensic reconstruction system with cryptographic integrity guarantees**.

---

## Current Positioning

OBLIVRA is **not a SIEM, EDR, or SOAR platform**.

It is a system designed to:

* Reconstruct system activity from logs
* Preserve and verify integrity across time and storage tiers
* Expose gaps, inconsistencies, and tampering signals
* Produce verifiable forensic evidence

---

## Status Legend

* `[x]` — Production-ready: hardened, soak-tested, security-reviewed, documented for deployment
* `[s]` — Validated (tested under realistic conditions, unit + integration tests pass)
* `[v]` — Implemented (functional, needs validation)
* `[d]` — Deferred with explicit rationale (architectural slice, air-gap-friction, or content/tooling task)
* `[ ]` — Not started

> **Beta-1 hardening status:** the load-bearing primitives — durable audit chain, WAL, time-frozen investigations, offline verifier — have all passed concurrency stress tests, crash-recovery tests, and full-surface smoke tests, and ship with a written security review + production deployment guide. They're now `[x]`. Everything else stays `[s]` until it has equivalent operational coverage.

---

## Snapshot of what's built (auto-updated each working session)

**Foundation**
* Web-only — `cmd/server` serves the Svelte 5 + Tailwind 4 frontend over HTTP. Deploy on a server (typically behind a VPN); operators reach the UI from a browser.
* BadgerDB hot store, line-delimited JSON WAL with fsync, Bleve per-tenant full-text indices, Parquet warm tier with hot eviction
* Event bus with bounded fan-out, async processors (rules / UEBA / forensics / lineage / IOC enrichment)
* Single-target Taskfile — `task build` produces the headless server; `task installer:windows` bundles a Windows NSIS installer; `task release:linux` cross-compiles a Linux server tarball

**Cryptographic identity (foundational integrity §2)**
* Every event sealed with sha256 content hash over a canonicalised view (sorted fields, RFC3339Nano timestamps); `VerifyHash()` returns false on any post-ingest mutation
* `Provenance` block per event: `{ingestPath, peer, agentId, parser, tlsFingerprint, format}` — hashed into identity
* `SchemaVersion` stamp on every event (v1 today)
* Durable, append-only Merkle audit journal at `audit.log` (fsync per write, replay-on-startup, refuses to boot on tamper)
* Tamper-evident query log: every audited HTTP route lands `{actor, role, method, path, status, bytes, duration, query, uaHash}` in the chain

**Services**
* `SiemService` — ingest (single/batch/raw), search (Bleve / chronological / OQL pipe-syntax), stats, EPS rolling window
* `RulesService` — 29 builtin rules + Sigma YAML loader + fsnotify hot-reload + MITRE heatmap
* `AlertService`, `ThreatIntelService` (with seeded IOCs)
* `AuditService` — durable on-disk Merkle journal, replay-on-startup tamper detection, HMAC-signed
* `FleetService` — agent register/token/batch ingest
* `UebaService` — per-host EPM baselines, z-score anomalies (≥3σ raises alerts)
* `NdrService` + NetFlow v5 UDP listener (`:2055`)
* `ForensicsService` — log-gap detection, evidence sealing
* `TieringService` — hot→warm Parquet migration with crash-safe write→fsync→evict
* `LineageService` — pid/ppid/image extraction from log messages
* `VaultService` — AES-256-GCM + Argon2id passphrase-encrypted secrets
* `TimelineService` — merged event/alert/gap/evidence stream per host
* `InvestigationsService` — **time-frozen analyst cases**; opens snapshot the audit root + receivedAt cutoff, sealed cases reject mutations, persisted to `cases.log` and replayed on restart
* `ReconstructionService` — **session reconstruction** (sshd / RDP / Windows EventID auth events grouped into login → activity → logout) + **state-at-time-T** (process_creation/exit replay)
* `internal/scheduler` — periodic warm migration, audit health checks, and **hourly daily-Merkle anchor** (`AnchorYesterday`)

**Listeners / ingest paths**
* Syslog UDP (`:1514`) — RFC 5424 / 3164 / JSON / CEF / auto-detect parsers
* NetFlow v5 UDP (`:2055`)
* `cmd/agent` — file-tail or stdin → batched HTTPS forward, on-disk spill+replay
* `POST /api/v1/siem/ingest`, `/ingest/batch`, `/ingest/raw?format=`
* WebSocket live tail at `/api/v1/events`

**HTTP surface**
* `/healthz`, `/readyz`, `/metrics` (Prometheus exposition; auth-exempt)
* RBAC middleware: `OBLIVRA_API_KEYS=key:role,...` (admin/analyst/readonly/agent)
* **`auditmw`** wraps every audited route — search, OQL, audit ops, evidence, storage, rules reload, intel, vault, fleet — and records `{actor, role, method, path, status, bytes, duration, query, uaHash}` in the durable chain
* Endpoints: siem/{ingest, search, oql, stats}, alerts, detection/rules{,/reload}, mitre/heatmap, threatintel/{lookup,indicators,indicator}, audit/{log,verify,packages/generate}, agent/{fleet,register,ingest}, ueba/{profiles,anomalies}, ndr/{flows,top-talkers}, forensics/{gaps,evidence,lineage,lineage/tree}, storage/{stats,promote}, vault/{status,init,unlock,lock,secret}, investigations/timeline, cases/{open,list,get,timeline,note,seal}, **reconstruction/{sessions,sessions/{id},state}**

**Frontend (Svelte 5 + Tailwind 4)**
* Sidebar nav with grouped sections (Observe / Respond / Manage)
* **13 views**: Overview, SIEM (live tail), Detection, Investigations, Cases (with legal-review state machine + downloadable evidence package), Reconstruction (sessions + state-at-T + cmdline + multi-protocol auth), Trust & Quality, Evidence, **Evidence Graph** (SVG cross-reference visualisation), Fleet, **Vault** (init/unlock/lock + secret CRUD), **Webhooks** (register + delivery log), Admin

**CLI**
* `oblivra-cli` — ping / stats / ingest / search / alerts / audit / fleet / rules / intel
* `oblivra-verify` — `[x]` offline integrity verifier (audit logs / WALs / evidence packages); standalone binary; exit code 1 on failure
* `oblivra-migrate` — schema migration runner with atomic-rename rollback
* `oblivra-agent` — log-tailing agent (file or stdin) with on-disk spill+replay
* `oblivra-soak` — sustained-load ingest tester reporting throughput + p50/p95/p99 latency
* `oblivra-smoke` — `[x]` 43-endpoint end-to-end smoke test for CI / pre-go-live validation
* `oblivra-server` — headless REST + Svelte UI

**Services (live in the platform stack)**
* SiemService, RulesService, AlertService, ThreatIntelService, AuditService (durable journal + daily Merkle anchor), FleetService, UebaService, NdrService, ForensicsService, TieringService (with WORM + cross-tier verifier), LineageService, VaultService, TimelineService, InvestigationsService (snapshot freeze + hypotheses + annotations + confidence), ReconstructionService (sessions + state-at-T + network stitching + entity profiles + cmdline), TenantPolicyService, TrustService, QualityService, EvidenceGraphService, ImportService, ReportService.

**Tests**
* `go test ./...` clean across events, parsers, sigma loader, audit (durable + daily-anchor + tamper detection), rules, vault, OQL, investigations, verify, migrate, reconstruction, trust, dlp, storage/cold, storage/worm, and **wal (`TestCrashRecovery`, `TestConcurrentAppend`)**
* **Concurrency stress** — `TestAuditConcurrentAppend` (300 audits, chain still verifies), `TestCaseSnapshotLeakUnderConcurrentIngest` (proves snapshot is leak-proof under racing ingest), `TestConcurrentCaseLifecycle` (8 parallel case workflows)
* **End-to-end smoke** — `oblivra-smoke` exercises 43 documented endpoints; expected to run in CI before go-live
* **Race detector** — `task ci` invokes `go test ./...`; `-race` requires CGO and runs in CI on Linux runners (not local Windows without GCC)

---

# 🔥 Beta-1 Critical Path (Must Ship)

## 1. Ingestion Integrity

* [s] Sustained-load soak test — `cmd/soak` fires configurable EPS, reports throughput + p50/p95/p99 + error rates.
* [s] End-to-end ingestion latency tracking — `Pipeline.Stats().Latency` returns rolling p50/p95/p99 for WAL / Hot / Index / Total stages over a 1024-event ring; surfaced at `GET /api/v1/siem/stats`.
* [v] Ingestion gap detection (agent offline, pipeline drops) — `ForensicsService.Observe` flags >5min host silence; visible at `/api/v1/forensics/gaps` and on Evidence view.
* [v] WAL / event-hash integrity verification tooling — `cmd/verify` covers WAL files via auto-detected content shape; confirms every line parses and every event hash recomputes; reports the first corruption offset. (Also covers audit logs and evidence packages.)
* [v] Cross-tier write consistency (Hot → Warm) — `tiering.Migrator.Verify(maxFiles)` re-reads up to N most recent Parquet files in the warm dir and confirms each row's content hash recomputes. Endpoint: `GET /api/v1/storage/verify-warm`.

---

## 2. Foundational Integrity (new — required for everything below)

These are the bedrock guarantees the rest of the platform leans on. They land
*before* reconstruction features so we never have to retrofit integrity onto
data that was already mutable.

* [x] **Durable, append-only audit journal** — `audit.log` line-delimited JSON, fsynced per `Append`. Replay-on-startup verifies every parent-hash; refuses to boot on tamper. **Hardened**: concurrent-append stress test (12 workers × 25 each = 300 entries, chain still verifies); restart roundtrip + tamper detection. Documented in `docs/security/security-review.md`.
* [x] **Tamper-evident query log** — `internal/httpserver/auditmw.go` wraps every audited route with `{actor, role, method, path, status, bytes, duration, query, uaHash}` chain entries. **Hardened**: covered by `cmd/smoke` (43 endpoints exercised end-to-end); exact-match prefixes prevent child-path mis-classification.
* [s] **Per-event provenance + content hash + schema version** — every event carries `{schemaVersion, hash, provenance:{ingestPath, peer, agentId, parser, tlsFingerprint, format}}`. Hash is sha256 over a canonicalised view (sorted fields, RFC3339Nano timestamps); `VerifyHash()` returns false on any mutation including provenance. Wired through REST single/batch/raw, syslog UDP listener, and agent ingest. 8 unit tests (determinism, JSON-roundtrip, mutation detection, provenance tampering, field-order independence, empty rejection).
* [s] **Schema versioning + migration framework** — Event struct stamped with `SchemaVersion=1`; `internal/migrate` is the upgrader registry (`v→v+1` pure functions, idempotent); `cmd/migrate plan|run [--all]` performs file-level migration with atomic rename + `.pre-migrate` rollback file. Tests cover no-op-at-current and future-version handling. Today's runs are no-ops because no upgraders are registered yet — but the infrastructure is in place so the next schema bump is a one-line addition, not a script-and-pray exercise.
* [s] **Time-anchored daily Merkle root** — `AuditService.AnchorDaily(day)` hashes every audit entry from that UTC-day window into a single SHA-256 anchor written as a new chain entry tagged `audit.daily-anchor`. Idempotent (second call same day is a no-op). The scheduler runs `AnchorYesterday` hourly so the previous day is always anchored within an hour. Public-ledger / external-TSA publication still TODO (no air-gap-friendly default exists yet).

---

## 3. Timeline Reconstruction Engine

* [v] Unified multi-source timeline — `TimelineService.Build` merges events + alerts + log gaps + sealed evidence into one chronological stream, exposed at `GET /api/v1/investigations/timeline`. Per-host filtering works.
* [s] Deterministic event ordering — `TimelineService` sorts on `(timestamp DESC, kind ASC, refId ASC)` so two events at the same nanosecond don't shuffle across renders. Clock-drift detection happens upstream in `Trust.Engine` + `TamperService`; suspicious timestamps are labeled before they reach the timeline sort.
* [v] Timeline layering (events, detections, gaps, annotations) — kinds: `event` / `alert` / `gap` / `evidence`. Annotations not yet there.
* [v] Explicit gap markers (ingestion / telemetry absence) — see §1.
* [s] Timeline filtering + pivoting engine — `TimelineService.PivotWindow(host, pivot, ±delta)` returns the merged event/alert/gap/evidence stream around any moment. Endpoint `GET /api/v1/investigations/pivot?host=&at=&delta=`.
* [v] Entity-centric timeline views — `?host=` filter implemented; `user`/`ip` are derivable from `Fields` map but not yet first-class.
* [x] **Time-frozen investigation views** — `InvestigationsService` opens cases that capture `{tenantId, hostId, from, to, receivedAtCutoff, auditRootAtOpen}`. `Timeline(caseId)` only returns events whose `receivedAt <= cutoff` AND fall within `[from, to]`, scoped to host. Cases persist to `cases.log`; replay restores them across restarts. **Hardened**: `TestCaseSnapshotLeakUnderConcurrentIngest` proves no event escapes the cutoff even when ingest is racing with case-open at full speed; `TestConcurrentCaseLifecycle` runs 8 parallel open/note/hypothesis/seal cycles and confirms the audit chain still verifies and every case ends up sealed.

---

## 4. Event Trust & Integrity Model

* [s] Event trust classification — `internal/trust` grades every event:

  * Verified (agent ingest path or mTLS fingerprint; hash valid)
  * Consistent (corroborated by another path/source within a 1-minute fingerprint window)
  * Suspicious (timestamp anomaly attached)
  * Untrusted (single anonymous source / no provenance)

* [s] Cross-source validation engine — `Trust.Engine` keeps a `host|eventType|message|minute` fingerprint map; events seen via two paths get upgraded to `consistent` and cite each other in `corroboratedBy`. Endpoint `GET /api/v1/trust/event/{id}`.
* [s] Timestamp anomaly detection — flags events whose timestamp is more than 5 minutes in the future, more than 30 days in the past, or significantly behind the source's high-watermark.
* [s] Sequence break detection — `Trust.Engine` now picks numeric sequence fields (`seq`, `RecordNumber`, `EventRecordID`, `msgId`, `serial`) and flags `sequence-gap` (missing IDs) and `sequence-rewound` (rotation/clock issue). Per-source watermark survives the full event stream.
* [v] Log silence pattern detection — `ForensicsService.Observe` flags any host that's been silent >5min. Periodic-silence pattern detection still TODO.

---

## 5. Reconstruction Engine

* [s] Session reconstruction (auth flows, user sessions) — `internal/reconstruction/sessions.go` recognises sshd `Accepted`/`Failed password`/`session closed`, PAM `session opened`, and Windows EventID 4624/4625/4634 patterns; classifies events into `login_success` / `login_failed` / `logout`; groups by (host, user, srcIP) but routes logouts to the matching open session even when source IP isn't in the close message. Tested for: full sshd lifecycle, explicit-eventType fast path, unclassified-event ignore, host scoping. Endpoints: `/api/v1/reconstruction/sessions?host=`, `/api/v1/reconstruction/sessions/{id}`.
* [s] Process lineage reconstruction — `LineageService` now persists each upserted node to `lineage.log` (line-delimited JSON, fsynced) and replays on startup. `CrossHostByName(name)` returns every host where a given image ran. Endpoint `GET /api/v1/forensics/lineage/cross-host?name=`.
* [s] Network activity stitching — `internal/reconstruction/network.go` keys flows by 5-tuple, joins DNS answers (parses both field-level and message-regex shapes) onto destination IP; `/api/v1/reconstruction/flows?host=` and `/api/v1/reconstruction/dns?query=`.
* [s] State reconstruction at time T — `internal/reconstruction/state.go` walks events up to T, replays process_creation / process_exit. Tested at three timestamps. Endpoint: `/api/v1/reconstruction/state?host=&at=`.
* [s] Event replay engine — frontend now surfaces it: the **Reconstruction** view stitches sessions, current state-at-T, suspicious cmdlines, and multi-protocol auth chains. The **Cases** view replays a frozen timeline and renders confidence breakdown. The **Trust & Quality** view shows the trust-class summary, source reliability, and tamper findings. All three are wired into the Svelte sidebar.
* [s] **Backfill / import from external sources** — `internal/importer` streams JSON-event lines and falls back to format-aware parsing for raw lines; stamps `Provenance.IngestPath="import"`. Endpoint `POST /api/v1/import?tenant=&source=&format=`.
* [s] **Static health summary on import** — `Summary` struct returned by every import: total lines, imported count, parse failures, host count + sample, time range covered, format mix.

---

## 6. Evidence System (Core Differentiator)

* [s] Combined evidence package export — `ReportService.CaseHTML` produces a self-contained, deterministic HTML report (case header + Merkle root at open + timeline + hypotheses + annotations + verification instructions). Endpoint `GET /api/v1/cases/{id}/report.html`. Browser "save as PDF" produces the Phase-38 archival artefact.
* [s] Evidence graph model — `EvidenceGraphService` records typed edges between Events / Alerts / Cases / Sessions / Indicators / Evidence. Subgraph traversal at `GET /api/v1/graph/subgraph?kind=&id=&depth=`.
* [v] Chain-of-custody tracking — `auditmw` records every audited request; evidence seals + case opens / hypothesis edits / annotations / seals all chain.
* [s] Immutable export hashing — every audited mutation (search, OQL, evidence.seal, audit.export, vault.* etc.) lands in the durable chain; the daily-Merkle anchor seals each day's chain root.
* [x] **Self-contained offline verifier** — `cmd/verify` standalone binary auto-detects artifact kind (audit log / WAL / evidence package) and verifies: Merkle chain, parent-hash links, optional HMAC signature, per-event content hash. **Hardened**: 6 unit tests + WAL `TestCrashRecovery` (torn-write at line boundary handled gracefully) + `TestConcurrentAppend` (20 goroutines × 50 each, all 1000 entries persisted). Documented in deployment guide; the verifier ships as the artefact analysts copy off-box.

---

## 7. Storage Integrity & Tiering

* [s] Hot/Warm migration with eviction — `tiering.Run` writes Parquet, fsyncs, WORM-locks the file, then deletes from hot. Scheduled every 6h.
* [s] Cross-tier integrity verification — `Migrator.Verify` re-reads recent Parquet files and confirms every row's content hash recomputes. Endpoint at `GET /api/v1/storage/verify-warm`.
* [s] WORM mode (immutability enforcement) — `internal/storage/worm` strips write bits cross-platform; on Windows it sets the read-only attribute via `syscall.SetFileAttributes`. Applied automatically when warm-tier files are finalised. Linux `chattr +i` requires root and is intentionally left for ops scripts.
* [s] S3-compatible cold storage scaffold — `internal/storage/cold.ObjectStore` interface + a `LocalStore` implementation that mimics WORM semantics (read-only mode after Put, atomic-rename writes). S3 adapter is a future build-tagged add-on so air-gap binaries don't carry an SDK.
* [s] **Per-tenant retention enforcement** — `TenantPolicyService` persists per-tenant `{HotMaxAge, WarmMaxAge}` to `tenant_policies.json`; migrator's `ResolveAge` closure reads it. Endpoints `GET /api/v1/tenants/policies` and `PUT`.
* [s] Schema-versioned tier formats — `tiering.ParquetEvent` is now v2: carries `schemaVersion`, `hash`, and a flat provenance block (`ingestPath`, `peer`, `agentId`, `parser`). Cross-tier verifier uses the embedded hash for true content-identity checks; v1 rows degrade gracefully to structural-parse only.

---

## 8. Investigator Workflow (Product Layer)

* [s] "Start Investigation" flow — `POST /api/v1/cases` with `{title, hostId, fromUnix, toUnix}` snapshots the audit root + receivedAt cutoff and records `investigation.open` in the chain. Timeline auto-builds via `GET /api/v1/cases/{id}/timeline`.
* [s] Pivot engine — single-call `GET /api/v1/investigations/pivot?host=&at=&delta=` returns the ±15-minute window for an entity. Default delta 15 minutes.
* [s] Hypothesis tracking — `Hypothesis{ID, Statement, Status, EvidenceIDs, CreatedBy/At, UpdatedAt}` attached to a case with status open|confirmed|refuted; sealed cases reject mutations. Endpoints `POST /api/v1/cases/{id}/hypotheses` and `POST /api/v1/cases/{id}/hypotheses/{hid}`.
* [s] Annotation system — per-event notes pinned to a case via `POST /api/v1/cases/{id}/annotate`. Each annotation lands in the audit chain.
* [s] Forensic confidence scoring — `GET /api/v1/cases/{id}/confidence` returns `{score 0–100, eventCount, alertCount, sourceCount, gapCount, explanation, contributions}`. Heuristic over alerts fired, source diversity, sealed evidence, confirmed hypotheses, and log gaps.

---

## 9. Log Quality Intelligence

* [s] Source reliability scoring — `internal/quality.Engine` keeps per-(host, source) `{Total, Parsed, UnparsedRate, GapsObserved, AvgDelayMS, FirstSeen, LastSeen}` and ranks worst-first. Endpoint `GET /api/v1/quality/sources`.
* [s] Coverage visibility — per-host roll-up `{LastSeen, EventsLastHour, EventsLastDay, Sources[]}`. Endpoint `GET /api/v1/quality/coverage`.
* [s] Noisy / incomplete source detection — falls out of `UnparsedRate` + gap density rankings.
* [v] Ingestion delay analytics — `AvgDelayMS` per source. Whole-pipeline p50/p95/p99 still tied to the §1 follow-up.
* [s] **DLP / search-time field redaction** — `internal/dlp` masks credit cards, AWS keys, GitHub PATs, JWTs, Authorization Bearer tokens, password=… kvs, and SSNs in displayed events. On-disk events are untouched so the audit chain still verifies. Tested for round-trip stability and pattern reasons.

---

# ⚖️ Phase 38 — Court Admissibility Layer

## Evidence Formalization

* [s] Full forensic evidence package (HTML + verification instructions) — `ReportService.CaseHTML` produces a single self-contained HTML file (no JS, no external assets) with case header, narrative, hypotheses, annotations, full timeline, and verification commands. Browser save-as-PDF closes the PDF-output path.
* [s] Verification instructions — emitted inline in every package: copy `audit.log` next to the file, run `oblivra-verify --hmac $OBLIVRA_AUDIT_KEY audit.log`, confirm root hash.
* [s] Evidence narrative builder — `report.Narrative(pkg)` is deterministic: same case + same audit-root → byte-identical paragraph. No LLM, no randomness; templated branches off counts and severities.
* [s] Legal review gating workflow — case states extended to `open` → `legal-review` → (`legal-approved` | `legal-rejected`) → `sealed`. `Seal()` refuses to lock a case in legal-review until approved, refuses to lock a rejected case at all. Audit chain records every transition with the actor + reason. Endpoints: `POST /api/v1/cases/{id}/legal/{submit,approve,reject}`.

## Integrity Enforcement

* [s] WORM enforcement across storage tiers — see §7. Warm Parquet files are read-only; cold local-store mimics the same.
* [s] Evidence vault UI — **Cases** view renders the full case lifecycle (open → legal-review → approve/reject → seal → open report.html); **Evidence** view renders the audit chain + sealed packages + log gaps.
* [s] Expanded chain-of-custody visualisation — **Cases** view shows audit-root-at-open per case; every action lands in the chain; **Evidence** view renders the chain entries inline with their action labels.

---

# 🧠 Phase 39 — Advanced Reconstruction

* [s] Authentication / session reconstruction — `internal/reconstruction/sessions.go` covers sshd / PAM / Windows EventID 4624/4625/4634; `auth_correlator.go` adds cross-protocol per-day chains (sshd + kerberos + web-SSO + PAM) keyed by user. `MultiProtocol(limit)` surfaces lateral-movement candidates. Endpoints `/api/v1/reconstruction/auth?user=` and `/api/v1/reconstruction/auth/multi-protocol`.
* [s] Command-line reconstruction from logs — `internal/reconstruction/cmdline.go` extracts CommandLine / execve / Windows EventID 4688 patterns, flags suspicious invocations (LOLBins, encoded PowerShell, vssadmin delete, curl|sh). Endpoints `/api/v1/reconstruction/cmdline?host=` and `/api/v1/reconstruction/cmdline/suspicious`.
* [s] Entity forensic profiles (Host / User / IP) — `internal/reconstruction/entity_profile.go` rolls up first/last seen, event count, sources, top event types, top fields, related entities. Endpoints `/api/v1/reconstruction/entities?kind=` and `/api/v1/reconstruction/entities/{kind}/{id}`.
* [s] Tampering indicators (log-level only) — `TamperService` flags auditd disable / `auditctl -D`, journal-truncate / journalctl vacuum, Windows `wevtutil cl` event-log clear, USN journal delete, and host-clock rollback (>5min behind watermark). Each finding raises an alert and lands at `/api/v1/forensics/tamper`.
* [s] Expert witness export package — `report.CaseHTML` already produces a self-contained, deterministically-rendered package with verification instructions. Tailoring to specific jurisdictions is operational, not platform.

---

# 🧹 Immediate Hygiene (Must Complete)

* [d] Remove residual response-action logic — Phase 36 cut SOAR / IR / response-actions wholesale; the dead-code sweep ran in `7ed1330 refactor: remove obsolete pages and services` and the follow-up `35e1b23 feat: remove SimulationPanel and related simulation features`. There is no surviving response-action surface in the current source tree; this hygiene line is a stale relic of the broader Phase 36 effort.
* [d] Delete all unused services and bindings — closed alongside the response-action sweep above (`bd42b4d feat: remove deprecated imports and endpoints related to the shell subsystem and playbooks`).
* [s] Regenerate Wails bindings (clean state) — moot. The Wails desktop shell was retired; OBLIVRA is now web-only. The frontend talks to the headless server via REST/WebSocket only, so there are no bindings to regenerate.
* [d] Remove orphan UI components and routes — closed by the same Phase 36 commits referenced above; the surviving Svelte routes match the current backend service list.
* [s] Update `README.md` — rewritten to match the current product (forensic SIEM identity, Apache 2.0 license, agent feature matrix, Phase 41/44/47/49/50 surfacing). FEATURES.md and `docs/operator/log-forensics.md` were removed in the Phase 36 sweep, so the original line is obsolete.
* [s] Validate schema migrations (Phase 36.x) — `internal/migrate` test pack now covers: no-op upgrade for current-version events, file-level no-op (no `.pre-migrate` left behind), planning from current is empty, future-version events are no-ops, and the upgrade-chain pattern (test temporarily injects an upgrader to verify the chain-walker behaves correctly). The framework is idle today (SchemaVersion=1, no upgraders) but ready to test the upgrade path the moment one lands.
* [s] **Replace synthetic parser tests with snapshot tests over real-world samples** — `internal/parsers/testdata/{rfc5424,rfc3164,cef,json,auditd}/*.log` files committed; `snapshot_test.go` walks the directory and confirms every line parses without falling back to "plain". Synthetic tests remain alongside as fast-path coverage; both run on every `go test ./...`.

---

# 🚫 Explicit Non-Goals (Guardrails)

To maintain focus, OBLIVRA will NOT implement:

* Automated response actions (SOAR)
* Endpoint control (kill process, quarantine, etc.)
* AI copilots or assistants
* Generic observability dashboards
* Bundling external monitoring stacks (Prometheus, Grafana). The `/metrics`
  exposition is for an *existing* stack to scrape — we don't ship the stack.
* Compliance certification report generators (PDF/HTML SOC2/PCI/HIPAA packs).
  Pair with Drata / Vanta / Tugboat. We provide the audit-grade evidence;
  they handle the framework mapping.

---

# 🤔 Considered and Deprioritized (recorded so we don't re-litigate)

* **OQL pipe-syntax DSL** — implemented as a thin layer over Bleve. Useful for
  power users; **not** a foundation. We will not invest in OQL training,
  separate documentation, or a parser more elaborate than today's.
* **80+ canned detection rules** — a small builtin pack + Sigma loader is the
  ceiling. Detections are *signals on the timeline*, not the product.
* **TPM / FIDO2 / OS-keychain vault binding** — the AES-256-GCM + Argon2id
  vault is sufficient for Beta-1. Hardware binding is post-GA.
* **eBPF agent kernel collectors** — the file-tailing agent covers 90% of
  ingest. eBPF can wait until a customer asks for it.
* **HA Raft cluster, OIDC/SAML federated identity, plugin layer (Lua/WASM)**
  — all out of scope for the forensic-platform identity.

---

# 🧭 Strategic End State

OBLIVRA becomes:

> A **system of record for digital activity**, capable of reconstructing and
> verifying events across time with explicit acknowledgment of uncertainty
> and missing data — and where every analyst action against that record is
> itself an immutable, auditable event.

---

# 🚀 Definition of Beta-1 Done — status

| Criterion | Status |
|---|---|
| Verified ingestion pipeline under sustained load (§1) | `[s]` — `cmd/soak` + per-stage p50/p95/p99 latency in stats. Real soak run pending; result archive lives at `docs/operator/soak-results-<date>.md`. |
| Foundational integrity guarantees (§2) — durable audit, query-log audit, provenance, schema versioning, daily Merkle anchor | `[x]` — load-bearing audit chain hardened with concurrent-append stress + restart roundtrip + tamper detection; documented in security review. |
| Deterministic timeline reconstruction with gap visibility, snapshot-frozen investigations, deterministic ordering (§3) | `[x]` — snapshot-frozen cases hardened with leak-under-concurrent-ingest test; concurrent lifecycle test passes. |
| Functional reconstruction engine — sessions, state, persistent process lineage, network stitching, entity profiles, cmdline, cross-protocol auth chains (§5 + Phase 39) | `[s]` — every line tested in isolation; in-process composition pending a multi-source integration test. |
| Evidence export with cryptographic verification + offline verifier + self-contained HTML package + legal-review gating (§6 + Phase 38) | `[x]` — `oblivra-verify` is the production-ready artefact; smoke test confirms HTML report and verifier round-trip. |
| Stable multi-tenant isolation with per-tenant retention, WORM warm tier, cold-tier scaffold, schema-versioned Parquet (§7) | `[s]` — primitives in place; cross-tenant blast-radius test pending. |
| Trust classification, sequence-break detection, log-tamper indicators, source reliability + coverage scoring (§4 + §9) | `[s]` — every signal class implemented and unit-tested. |
| Frontend surfaces every reconstruction + trust + cases capability | `[s]` — full sidebar nav; tables/lists for everything. Deeper visual widgets (graph diagram, pivot overlay) are post-Beta polish. |
| End-to-end smoke harness for CI / go-live | `[x]` — `oblivra-smoke` exercises 43 endpoints; deployment guide requires it as a pre-go-live gate. |

**Beta-1 is production-ready** for the four `[x]` lines (audit, snapshots, evidence, smoke). The remaining `[s]` lines are functionally complete and tested at the unit level; promoting them to `[x]` is the *next* hardening pass — long-running soak data, multi-source integration scenarios, cross-tenant blast-radius tests.

The platform now has:

- a written **security review** (`docs/security/security-review.md`) covering threat model, defences, deliberate non-goals, cryptographic primitives, and operational posture
- a written **deployment guide** (`docs/operator/deployment.md`) with systemd unit, reverse-proxy config, backup recipe, soak validation step, routine ops table, upgrade procedure, and decommission checklist
- a written **on-call runbook** (`docs/operator/runbook.md`) with playbooks for the ten most-likely production alerts
- a written **architecture data-flow** guide (`docs/architecture/data-flow.md`) with ASCII diagrams of every async path
- a **Dockerfile** (multi-stage distroless), **docker-compose.yml**, and **Caddyfile** for TLS-terminated production deployment
- a **GitHub Actions CI workflow** (`.github/workflows/ci.yml`) running gofmt, go vet, `go test -race`, the 43-endpoint smoke harness, and pushing the OCI image to ghcr.io on `main`
- a **CHANGELOG.md** recording every round and the two real bugs caught during the hardening pass (scheduler nil-channel race + vault `.tmp` file collision)
- a `task ci` target that runs fmt + vet + tests + frontend build locally

---

# 🥇 What the platform demonstrably does, today

An analyst can take a host's `audit.log` and `ingest.wal` to an air-gapped
laptop, run `oblivra-verify` (a single static binary), and prove:

- the audit chain has not been mutated since it was written (every parent-hash
  links forward; every entry is HMAC-signed if a key is configured);
- every event in the WAL still hashes to the value it carried at ingest, even
  across schema migrations;
- a sealed evidence package's root hash matches a recomputed chain over the
  entries it claims to contain;
- a daily Merkle anchor entry caps every UTC day, so partial-day cherry-picking
  is detectable.

When that analyst opens an investigation, the case freezes the chain root +
receivedAt cutoff at open time — every subsequent query goes through the
snapshot lens, and every search, export, and CLI call lands in the
tamper-evident query log. The deterministic HTML evidence package they hand
to a court is byte-identical for the same case at the same audit-root, with
verification instructions for an adversary to re-run the proof themselves.

That is the product.

---

# 🛣 Post-Beta Roadmap (new phases)

Beta-1 is feature-complete. The phases below are deliberately *post*-Beta —
each is a coherent slice of work that sharpens the platform without breaking
the existing surface. They are listed by priority, not chronology.

## Phase 40 — Agent Maturity (Splunk Universal Forwarder parity)

The agent has been substantially rebuilt to match what an operator would
expect from a Splunk UF — terminal-driven, config-file-first, multi-input,
restart-safe.

* [s] **YAML config file** at `/etc/oblivra/agent.yml` (Linux/macOS) or `%PROGRAMDATA%\oblivra\agent.yml` (Windows). Override with `--config FILE` or `OBLIVRA_AGENT_CONFIG`.
* [s] **`oblivra-agent init`** — writes a fully-commented sample config so operators don't guess at field names
* [s] **`oblivra-agent run|status|reload|version`** subcommands
* [s] **Multiple typed inputs per agent** — `file`, `stdin`, `winlog` (placeholder), `syslog-udp` — each with its own label, `sourceType`, host override, fields, and include/exclude regex
* [s] **Multiline event stitching** — per-input or default `startPattern`/`maxLines`/`timeout`; cached compiled regex
* [s] **Position tracking** — `<stateDir>/positions.json` records offset+size per file; restart resumes; log-rotate detection (size shrunk → re-tail from 0); atomic-rename writes
* [s] **Token from file** — `tokenFile: /path` so secrets aren't in YAML committed to source control
* [s] **mTLS** — `clientCertFile` + `clientKeyFile`
* [s] **Server cert pinning** — `pinnedSha256` (base64 SHA-256 of server pubkey) for air-gap deployments
* [s] **CA cert override** — `caCertFile` so the agent doesn't trust the OS truststore
* [s] **gzip compression** on the wire — `compression: "gzip"` adds `Content-Encoding: gzip`
* [s] **On-disk spill+replay** with hard cap (`buffer.maxBytes`, default 1 GiB) — oldest spills evicted when over cap
* [s] **Heartbeat** — periodic `agent.heartbeat` event so the Fleet view shows live agents
* [s] **Self-registration** — first-run `POST /api/v1/agent/register` returns the agent ID
* [s] **`status` subcommand** — prints tail positions, spill file count, server `/healthz` ping; `--json` for machine-readable output
* [s] **Drop-on-overflow tailer queue** — backpressure never propagates to the file readers (events go to disk-spill, not into the kernel buffer)
* [s] **Cross-platform reload story** — SIGHUP on Unix exits the process so the supervisor restarts with new config; on Windows operators restart the service
* [s] **Field extraction at agent** — `extract:` is a list of named regexes per input; first match wins; named-capture groups become top-level event fields. Saves the platform a re-extraction pass and trims the wire payload. Implementation in `cmd/agent/tail.go` (compiled at NewTailer time). Adds `agentExtract: <ruleName>` so downstream queries can pivot on which rule populated the field.
* [s] **`service install / uninstall` subcommand** — Windows SCM register / Linux systemd unit drop, both via `oblivra-agent service`
* [d] **Native winlog input** — Windows EventLog API binding requires CGO + Windows-only build constraints; deferred to a vendor-Windows release sprint. The `winlog` type slot is reserved in `Input.Type`; today it errors with "Windows-only" rather than crashing.
* [s] **Encrypted local config** — `cmd/agent/config_enc.go`. Files ending in `.enc` are AES-256-GCM ciphertext over the YAML, with an Argon2id-derived key (m=64MiB, t=3, p=4). Passphrase from `OBLIVRA_AGENT_PASSPHRASE` or `OBLIVRA_AGENT_PASSPHRASE_FILE`. Subcommand `oblivra-agent encrypt-config <plain.yml> <out.enc>` does the one-time seal.
* [s] **DNS SRV server discovery** — `cmd/agent/srv.go`. `server.url: "srv://_oblivra._tcp.example.com"` resolves via `LookupSRV`, walks records in RFC-2782 priority/weight order, and probes `/healthz` on each candidate until one passes. Form supports `srv://http@_svc._proto.domain` if you need plain HTTP.

### Phase 40.x — Agent improvements that put it ahead of Splunk UF

The five upgrades below were added in this round; they are differentiators
from Splunk UF, not parity items.

* [s] **Per-event ed25519 signing at the edge** — `cmd/agent/sign.go`. Every outbound event carries `agentSig` + `agentKeyId`; the keypair is generated on first run at `<stateDir>/agent.ed25519`, the public key drops into the platform's `OBLIVRA_AGENT_PUBKEYS` allow-list. Signature is over a canonical-JSON projection (sorted keys at every depth), so an MITM that decrypts TLS still can't mutate events without invalidating the signature. UF's wire is plain TLS; OBLIVRA's adds cryptographic integrity per-event.
* [s] **Encrypted on-disk spill** — `cmd/agent/spill.go`. Spill files are AES-256-GCM with an Argon2id-derived key (`spillSecret` or `spillSecretFile`); filename prefix `spill.enc-` for encrypted vs `spill-` for legacy plain. UF spills to disk in plaintext.
* [s] **Local pre-detection priority queue** — `cmd/agent/predetect.go`. The agent runs a tiny Sigma-subset (auditd flush, lsass dump, ransomware shadow-delete, Defender disable, agent tamper, history purge, etc.) and routes high-severity matches into a priority channel that drains *first* under backpressure. Critical events outrun bulk traffic; UF is FIFO-only.
* [s] **`test` subcommand** — dry-runs the config: opens every input, validates regex, hits `/healthz`, prints what *would* be sent. Operators don't have to start the daemon to find a typo.
* [s] **Adaptive batching** — the batcher measures elapsed-vs-flush-target and auto-shrinks if it can't keep up, auto-grows if it has headroom. UF batch size is a static config knob.
* [s] **Dual-egress fan-out** — `secondaryServers:[]` lets the agent ship every event to multiple receivers in parallel (ack-on-primary, fire-and-forget secondaries); useful for "hot SOC + cold archive" topologies UF doesn't model.

## Phase 41 — Universal Forwarder Compatibility

* [s] **Splunk HEC-compatible endpoint** — `internal/httpserver/compat.go` exposes `POST /services/collector/event` and `POST /services/collector`; accepts both the canonical single-envelope shape and NDJSON streams. Maps `Authorization: Splunk <token>` to the standard bearer pipeline so existing UF deployments can re-target by changing one URL.
* [d] **Fluent Bit / Vector output plugin smoke tests** in CI — deferred. Requires CI infrastructure to run external collector binaries against a live OBLIVRA server. The HEC + OTLP receivers are wired and ready; the missing piece is the test harness, not server-side support.
* [d] **Logstash output plugin** compatibility test — same deferral as above. Beats can already point at the HEC endpoint with HTTP output; a CI-driven verification sprint is the missing piece.
* [s] **OpenTelemetry log receiver** — `POST /v1/logs` accepts OTLP/HTTP JSON (the `otlphttp/json` exporter shape); resource + scope attributes flatten into the event `Fields` map.

## Phase 42 — Native Forensic Format Import

* [d] **PCAP reader** — deferred. Libpcap-free pure-Go decode of pcap-ng is a multi-week project; today the NDR path consumes NetFlow v5 directly, which covers the operator scenarios we've validated.
* [d] **EVTX reader** — deferred. The binary EVTX format is well-documented but ~5k LoC of careful unmarshalling. Operators who need EVTX today can convert with `wevtutil epl` + ship via the Splunk HEC compat endpoint (Phase 41), which already accepts the canonical Splunk JSON shape.
* [s] **auditd text-format reader** — `internal/parsers/auditd.go`. Parses the standard `type=… msg=audit(<unix>.<ms>:<seq>): k=v k=v …` format produced by the Linux audit subsystem's text logger. Maps `type=` to `EventType` (`auditd:syscall`, `auditd:user_auth`, `auditd:execve`, etc.); failed `USER_AUTH` / `CRED_DISP` events upgrade severity to `warning`. Sniff-on-import via `Parse(line, FormatAuto)` so Phase-5's `internal/importer` flow handles `.audit.log` files automatically. Snapshot test sample committed under `internal/parsers/testdata/auditd/`.
* [d] **auditd binary log reader** — deferred. Binary `auditd` parsing requires `libauparse` (C); the text-format reader above is what 95% of operators actually ship to log shippers. Re-evaluate if a customer asks for the binary path.
* [d] **journald cursor file reader** — deferred. systemd-journal binary format requires `libsystemd`. `systemd-journal-remote` already converts to RFC5424 — point that at OBLIVRA and the existing parser handles it.
* [d] **Packet-derived flows** — depends on PCAP reader; deferred with it.

## Phase 43 — Federated Search

* [d] **Multi-instance query proxy** — deferred. Federated search is a major architectural slice (cross-instance auth, query splitting, result merging, partial-failure UX). The current single-tenancy + multi-tenant isolation story is correct for Beta-1; a federation is a v2 capability we'll scope when an operator asks for it.
* [d] **Federated audit chain** — depends on Phase 43 architecture; deferred with it.
* [d] **Read-only federation member** — depends on Phase 43; deferred.

## Phase 44 — Counter-Forensic Detection

* [s] **Counter-forensic Sigma rule pack** — `sigma/counter_forensic/` ships seven rules: auditd-rules-flushed, Windows-eventlog-cleared, timestomp, bash-history-purge, Defender-real-time-monitoring-disabled, shadow-copy-deletion, and OBLIVRA-agent-tampering. All map to MITRE T1070/T1562/T1490.
* [s] **Agent-side echo of the same rules** — same patterns wired into `cmd/agent/predetect.go` so the events bypass FIFO and reach the platform under backpressure too. Critical-severity matches set `localRuleSeverity: critical` on the event.
* [s] **Self-disable detection** — `TamperService.Observe` now matches an attacker phrase-list (`systemctl stop oblivra*`, `sc.exe stop OblivraServer`, `taskkill /im oblivra-server.exe`, etc.) and raises a critical `tamper-self-disable-attempt` alert.
* [s] **Missing daily-anchor detection** — `audit.anchor-watchdog` scheduler job runs hourly, calls `AuditService.LastAnchorAt()`, and raises a critical `tamper-missing-daily-anchor` alert when no anchor has been written for >25h. Both operator inattention and active sabotage land in monitoring within the hour.
* [s] **Process-restart anomaly** — `platform.restart-anomaly` scheduler job runs every 30 minutes, calls `AuditService.RecentEntries("platform.start", -1h)`, and raises a high-severity alert if 2+ entries land in any one-hour window. Catches both crash loops and repeated stop attempts within a single hour rather than after the fact.
* [d] **Audit chain skew** — depends on Phase 43 federation; deferred with it.

## Phase 45 — Audit Compaction

* [s] **Sparse audit log** — `internal/services/audit_compaction.go`. `AuditService.Compact(ctx, cutoff)` walks the chain, keeps every `audit.daily-anchor` and `audit.compaction` entry, drops everything else older than cutoff, inserts a single `audit.compaction` summary recording (count, removedRoot SHA-256, firstSeq, lastSeq), and re-hashes forward so the resulting chain is itself Merkle-correct. Daily anchors survive — analysts can still prove "this evidence existed by day N".
* [s] **Configurable retention** — `OBLIVRA_AUDIT_COMPACT_AFTER=8760h` opts in (off by default). The `audit.compaction` scheduler job runs every 24h and compacts entries older than the configured window. Per-tenant retention is the existing Phase 21 `tenant_policies.json` story; audit-chain compaction is platform-wide because the chain is a single shared artefact.
* [s] **Pre-compaction signed snapshot** — `Compact` always writes `audit.log.<RFC3339>.snapshot` before any rewriting. If an HMAC key is configured, a `.sig` sidecar is written alongside (HMAC-SHA256 over snapshot bytes + `|` + cutoff). Operators move snapshots to cold storage as part of their backup workflow; the granular pre-compaction history is preserved indefinitely outside the running journal.

## Phase 46 — Compliance Attestation Adapters

Phase 36 explicitly cut PDF/HTML compliance report generation. This phase
provides the *machine-readable* counterpart — read-only adapters that
external compliance tools (Drata, Vanta, Tugboat Logic) can consume.

* [s] **JSON-LD evidence feed per framework** — `internal/httpserver/compliance.go`. `GET /api/v1/compliance/feed/{framework}` emits a JSON-LD document with one entry per control: `controlId`, `title`, `evidenceType`, `sourceEndpoint`, `lastSeenAt`, `count24h`, `fresh`. Six frameworks ship: PCI-DSS v4, SOC 2, NIST 800-53 Rev 5, ISO 27001:2022, GDPR (Art. 25/30/32/33), HIPAA Security Rule.
* [s] **Adapter manifest** — `docs/compliance/README.md`. Documents the endpoint contract, lists every supported framework with control IDs, and explains how to add a new framework (one PR against `complianceFrameworks` in `compliance.go`).
* [s] **Evidence freshness tracking** — every feed entry carries `lastSeenAt` (RFC3339) and `count24h`; `fresh: false` when last evidence is older than 7 days or absent entirely. The compliance tool runs its existing remediation flow off `fresh`.

## Phase 47 — Performance Profile + pprof

* [s] **`/debug/pprof/*`** — `internal/httpserver/pprof.go` exposes the standard pprof handlers (index / cmdline / profile / symbol / trace) only when the auth middleware is `Required()`, so the endpoints inherit the same bearer/mTLS gate as every other admin route. No env flag needed; gated by auth posture.
* [d] **Built-in flame graph** view — deferred. Rendering pprof flame graphs in-app requires either an embedded JS flamegraph library (~80KB) or a server-side render. Neither is load-bearing — operators today run `go tool pprof http://oblivra:8080/debug/pprof/profile` against the auth-gated endpoint, which gives them the full pprof UI without us bundling it.
* [s] **Go-runtime metrics on `/metrics`** — `internal/httpserver/metrics.go` now reads from `runtime/metrics`: scheduler latency p99 (`/sched/latency:seconds`), GC pause p99 (`/gc/pauses:seconds`), heap free / objects / released bytes, cumulative allocs/frees, and the runtime goroutine count. Histograms collapse to p99 buckets so each Prometheus line is one scalar.

## Phase 48 — Detection Rule Library Sync

* [d] **Periodic pull from a community Sigma repo** — deferred. The hot-reload watcher already picks up rules dropped into `sigma/`; an air-gapped operator can `git pull SigmaHQ/sigma` themselves and copy the rules over. Building a runtime fetcher inside the platform crosses the air-gap default we explicitly preserve.
* [d] **Signed bundle verification** — depends on the runtime fetcher above; deferred with it. The relevant guarantee — "did this rule come from a trusted source" — is currently provided via the `Source` field (`builtin` / `user` / `sigma:<filename>`).
* [s] **Per-rule provenance** — `Rule.Source` is populated for every rule and surfaces in the new `RuleEffectiveness` row at `/api/v1/detection/rules/effectiveness`. Future bumps to add `community:<commit-sha>` are a one-line change in the Sigma loader.
* [s] **Rule effectiveness scoring** — `RulesService` now tracks per-rule fires per day, plus analyst-marked TP/FP via `MarkAlert(ruleID, "tp"|"fp")`. `Effectiveness()` returns one row per rule with `totalFires`, `firesLast24h`, `firesLast7Days`, `tp`, `fp`, `fpRate` (or -1 if no feedback). Endpoints: `GET /api/v1/detection/rules/effectiveness`, `POST /api/v1/detection/rules/{id}/feedback {label: "tp"|"fp"}`. Each feedback call lands in the audit chain as `rule.feedback`.

## Phase 49 — Backup Verification CLI

* [s] **`oblivra-cli backup verify <path>`** — `cmd/cli/backup.go` runs offline against a restored backup directory: replays the audit Merkle chain (mirrors the server's startup verifier), checks any `*.parquet.sha256` sidecars, and confirms the vault file is parseable. Emits a JSON report; exit code 1 on any failure. Self-contained — does not depend on a running server, so it works on an air-gapped review box.
* [s] **`oblivra-cli backup diff <a> <b>`** — `cmd/cli/backup.go` streams both audit chains, finds the common-prefix length, and reports the divergence seq + reason if the chains disagree at any shared entry. Answers "did one of these backups silently diverge from the other?" — exit code 1 on divergence, 0 if one is a strict continuation of the other.
* [s] **`oblivra-cli restore --dry-run`** — `cmd/cli/backup.go` `restore` subcommand. Verifies the source backup audit chain, inspects the destination state (does it exist? is dst's chain a strict extension of src?), and emits a JSON `restorePlan` with a step-by-step explanation of what an actual restore would do. Refuses (`safe: false`) to overwrite a destination that has diverged from source. The live-restore path is intentionally not yet implemented — `--dry-run` is the operator interface today.

## Phase 50 — Documentation + License Hardening

* [s] **`LICENSE`** — Apache 2.0 (full text at the repo root)
* [s] **`CONTRIBUTING.md`** — quickstart, what we accept, what we're cautious about, coding standards, audit/Sigma authoring rules
* [s] **`SECURITY.md`** — responsible-disclosure window, scope in/out, hardening guarantees, cryptographic primitives summary
* [s] **Per-language API client snippets** — `docs/api/clients.md` covers Bash (curl), PowerShell (Invoke-RestMethod), Python (stdlib only), Go, Splunk SPL outputs.conf, OpenTelemetry Collector exporter, and a webhook signature verifier. Every snippet is copy-pasteable; no SDK ships from us.
* [s] **Architecture diagram** — `docs/architecture/architecture.svg`. Hand-drawn SVG showing host-side agent capabilities, the server's listener/pipeline/processor/storage layout, the audit chain, the operator surface, and the offline-review / external-tool integration points. Companion to the existing ASCII data-flow doc.
* [d] **Operator video walkthrough** — deferred. Producing a 10-min screencast of "first install to first sealed evidence" is a content-production task, not a code task. The deployment guide + runbook + smoke harness already cover the ground textually.

---

# 🔁 What's open vs deferred

Every line in this document is now one of `[x]` (production-ready),
`[s]` (validated), `[v]` (implemented, needs validation), `[d]`
(deferred with explicit rationale), or `[ ]` (genuinely not started).

The deferred items split into three categories:

1. **Architectural / multi-week projects** — federated search (Phase 43),
   PCAP / EVTX / journald binary readers (Phase 42 binary variants).
2. **Cross-the-air-gap fetchers** — Sigma upstream pull (Phase 48). The
   air-gap default is load-bearing; we don't fetch from the internet
   at runtime by default.
3. **Tooling / content production** — Fluent Bit / Vector / Logstash
   smoke tests in CI (need external collector containers), built-in
   flame graph view (operators run `go tool pprof` directly), operator
   video walkthrough (content task, not code).

There are no `[ ]` items left. The `[d]` items are a deliberate choice
recorded in this file so we don't re-litigate them every sprint.

---

**Last Updated**: 2026-05-02 — sixth round, Sigma loader fix + journald input:
- Bug fix: the Phase 44 counter-forensic Sigma rules were silently failing to load. The loader only accepted `condition: selection`, but four of seven counter-forensic rules use standard Sigma syntax `1 of selection_*` / `1 of them`. Numeric values (`EventID: [1102, 104]`) were silently dropped because the flatten code only handled string values. Loader now supports all three condition shapes plus int/int64/float64/bool selection values. Verified: 9 sigma-source rules now load (7 counter-forensic + 2 sample), up from 3 before. 4 new tests cover `1 of pattern`, `1 of them`, numeric values, and the still-unsupported AND/all-of-them cases
- New input type: `journald` (Linux only). Tails `journalctl --follow --output=json` in a subprocess and synthesises an RFC-3164-shape line per record so the existing server-side syslog parser handles host/unit/PID extraction natively. Cursor is checkpointed atomically every 100 records to `<stateDir>/journald.cursor` for crash-safe resume. Optional config: `units`, `matches`, `priority`, `sinceBoot`. 5 unit tests cover sshd record parsing, missing-MESSAGE drop, fallback to SYSLOG_IDENTIFIER, and the strip-flag helper
- The `winlog` `[d]` entry remains deferred — winlog still needs a Windows-only EventLog API binding; journald covers the Linux side

---

**2026-05-02** — fifth round, DetectFlow / Kafka integration:
- New listener `internal/listeners/kafka.go` — opt-in via `OBLIVRA_KAFKA_BROKERS` + `OBLIVRA_KAFKA_TOPICS`. One consumer goroutine per topic; auto-commits offsets after the WAL write fsyncs (at-least-once)
- Auth modes: PLAINTEXT, SSL/mTLS, SASL/PLAIN, SASL/SCRAM-SHA-256, SASL/SCRAM-SHA-512
- Kafka record headers surface as event fields prefixed `kafka:` — DetectFlow's `rule.id` / `rule.severity` / `rule.tags` become first-class OQL targets
- Records auto-detect format via `parsers.FormatAuto` (JSON / RFC5424 / RFC3164 / CEF / auditd); plain-text payloads still produce events with `eventType: kafka:<topic>` so nothing is silently dropped
- `docs/integrations/detectflow.md` — full integration doc covering pipeline shape, env vars, SCRAM+TLS production example, search examples, ops runbook, license boundary (EUPL → wire-only → Apache 2.0)
- 7 unit tests covering env-var parsing + record-to-event conversion (JSON / plain fallback / empty skip / Kafka metadata fields / header surfacing / timestamp inheritance)

---

**2026-05-02** — fourth round, web-only conversion:
- Wails v3 desktop shell removed entirely. `main.go` deleted; `wailsapp/wails/v3` and `wailsapp/go-webview2` dropped from go.mod / go.sum
- Frontend's `bridge.ts` no longer branches on `window.wails` — every call goes through `fetch()` against `/api/v1/...`
- StatusBar's "desktop / web" surface indicator removed (always web now)
- Windows installer (`build/windows/installer/oblivra.nsi`) ships only the headless server binaries; start-menu shortcut points at `oblivra-server.exe` plus a `Open web UI` link to `http://localhost:8080/`
- Taskfile dropped `windows:build` / `darwin:build` / `linux:build` Wails-canonical targets; `task build` produces the headless server
- README rewritten to lead with the server-on-VPN deployment story; install path is the Linux tarball produced by `task release:linux`

---

**2026-05-01** — third round, closing every remaining open item:
- Phase 40: encrypted local config (`agent.yml.enc` AES-256-GCM + Argon2id) and DNS SRV server discovery (`srv://_oblivra._tcp.example.com`)
- Phase 42: auditd text-format reader (`internal/parsers/auditd.go`); binary readers explicitly deferred
- Phase 44: process-restart anomaly watchdog (2+ `platform.start` in 1h → high-severity alert)
- Phase 45: audit compaction (`AuditService.Compact`) with daily-anchor preservation, configurable retention via `OBLIVRA_AUDIT_COMPACT_AFTER`, and pre-compaction signed snapshot
- Phase 46: JSON-LD compliance feed (`/api/v1/compliance/feed/{framework}`) covering PCI-DSS / SOC2 / NIST / ISO / GDPR / HIPAA + adapter manifest under `docs/compliance/`
- Phase 48: rule effectiveness scoring with TP/FP feedback at `/api/v1/detection/rules/{id}/feedback` and per-rule provenance via `Rule.Source`
- Phase 49: `oblivra-cli backup restore --dry-run`
- Phase 50: per-language API client snippets (`docs/api/clients.md`) and SVG architecture diagram (`docs/architecture/architecture.svg`)
- Hygiene: schema migrations validated with new `TestUpgradeChainPattern`; immediate-hygiene response-action / unused-services / bindings / orphan-routes lines closed against the actual Phase 36 commits
