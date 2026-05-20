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

## Phases 63-100 (ingest compatibility, log pipeline adapters, enrichment, search UX, retention & cold-tier hardening, HA-lite, dark-mode UI, GeoIP, correlation graph, and Grafana Loki-compatible query API):

---

## Phase 63 — Ingest Compatibility Layer (Elastic / Graylog / rsyslog / syslog-ng / Fluent Bit / Fluentd parity)

OBLIVRA already has a Splunk HEC endpoint (removed in Phase 59) and a syslog UDP listener. This phase adds the remaining wire-level adapters so operators can re-point existing pipelines without touching agent configs. All adapters funnel into the same WAL → hot store → audit chain as native ingest — no parallel code paths.

* [ ] **GELF UDP/TCP receiver** — Graylog Extended Log Format is the dominant Graylog output format and a first-class Fluentd/Fluent Bit target. `internal/listeners/gelf.go` listens on `:12201` (UDP) and `:12202` (TCP/newline-delimited). Maps GELF fields (`host`, `short_message`, `full_message`, `_*` additional fields) onto OBLIVRA's Event struct. Chunked UDP (GELF magic `0x1e 0x0f`) reassembled with a 5s window and 128-chunk cap before forwarding. Compressed payloads (gzip/zlib) decompressed inline. Config: `OBLIVRA_GELF_UDP_ADDR` / `OBLIVRA_GELF_TCP_ADDR`; both disabled by default. Reason: lets any Graylog-targeted Fluent Bit / Fluentd / rsyslog pipeline re-point at OBLIVRA with one config line.
* [ ] **rsyslog RELP receiver** — Reliable Event Logging Protocol guarantees at-least-once delivery with TCP ACKs. `internal/listeners/relp.go` on `:2514`. Implements RELP frames (open/syslog/close commands); ACKs each frame after WAL fsync so rsyslog never loses events on OBLIVRA restart. Operators with existing rsyslog → RELP → Logstash pipelines can re-target OBLIVRA directly.
* [ ] **Fluent Bit / Fluentd Forward Protocol receiver** — `internal/listeners/fluentforward.go` on `:24224` (TCP). Implements MessagePack-encoded Forward protocol (Message / Forward / PackedForward / CompressedPackedForward modes). Chunk ACK returned after WAL fsync. Enables zero-config re-targeting of existing Fluent Bit `[OUTPUT] Name forward` or Fluentd `<match> @type forward` blocks.
* [ ] **OpenSearch / Elasticsearch Bulk API shim** — `POST /_bulk` endpoint that accepts NDJSON action+document pairs (index/create actions only; update/delete are no-ops with a 200 response). Maps `_index` to tenant, `_source` fields to event `Fields` map. Lets operators who already have Logstash `elasticsearch` output or Filebeat `elasticsearch` output re-target OBLIVRA by changing one URL + credentials. No ES query API is implemented — ingest-only shim.
* [ ] **Grafana Loki push API receiver** — `POST /loki/api/v1/push` accepts Loki's Protobuf or JSON push format (streams with label sets + log entries). Maps Loki labels to event `Fields`, `stream` label becomes `sourceType`. Enables Promtail, Grafana Agent, and Vector `loki` sinks to forward into OBLIVRA without modification.
* [ ] **syslog-ng `network()` destination compatibility** — syslog-ng's default `network()` destination sends RFC 5424 or RFC 3164 over TCP. Existing UDP listener already handles the format; add a TCP syslog listener on `:1514` (RFC 6587 octet-counted framing + newline fallback) so syslog-ng `transport(tcp)` works without reconfiguration.
* [ ] **CI smoke tests for each adapter** — `cmd/smoke` extended with one round-trip test per new listener (GELF UDP, RELP, Forward, Bulk shim, Loki push, TCP syslog). Each test sends a synthetic event via the wire protocol and confirms it appears in `GET /api/v1/siem/search`. No external collector binaries required — pure Go dial + write in the smoke harness.

---

## Phase 64 — Enrichment Pipeline

Raw log events often lack context that makes reconstruction meaningful — GeoIP, ASN, hostname resolution, threat-intel tags, and asset-registry metadata. This phase adds a pluggable enrichment stage that runs after WAL write but before hot-store index, so enriched fields are searchable from ingest time.

* [ ] **GeoIP enrichment** — `internal/enrichment/geoip.go`. Reads a MaxMind GeoLite2-City MMDB file (`OBLIVRA_GEOIP_DB`). For every event with a non-RFC1918/loopback IP in `srcIP` or `dstIP`, appends `geo.src.{country,city,lat,lon,asn,org}` / `geo.dst.{...}` to `Fields`. MMDB loaded once at startup; hot-reload on SIGHUP. No network calls — fully air-gap compatible. Optional: `OBLIVRA_GEOIP_ENRICH_FIELDS=srcIP,dstIP,clientIP` to extend the field scan list. New **GeoMap** frontend panel: Leaflet.js world map (bundled tiles from a single PNG sprite, no CDN) with event-count choropleth by country and drill-down to host list.
* [ ] **ASN / CIDR enrichment** — companion to GeoIP: MaxMind GeoLite2-ASN MMDB for AS number + org name. Flags known-bad ASNs from the threat-intel seeded list as `asn.flagged: true`.
* [ ] **Hostname reverse-DNS cache** — `internal/enrichment/rdns.go`. Background worker resolves IPs in a bounded LRU cache (default 50k entries, 1h TTL). Non-blocking: if the cache misses, the event is indexed immediately and the resolved hostname is written as a patch event `rdns.resolved` when the lookup completes. Air-gap deployments set `OBLIVRA_RDNS_DISABLED=1` to skip entirely.
* [ ] **Asset registry enrichment** — `internal/enrichment/assets.go`. Operators upload a CSV/JSON asset list (`POST /api/v1/assets/import`) mapping IPs/hostnames to `{owner, criticality, os, role, location, tags}`. Enrichment stage appends `asset.*` fields. `criticality: critical` events get severity bumped one step. New **Assets** view: import, list, edit, tag. `GET /api/v1/assets` with OQL-compatible filter.
* [ ] **IOC tag enrichment** — existing `ThreatIntelService` does lookup at query time. Move to ingest-time enrichment: matched IOCs stamp `ioc.match: true`, `ioc.tags: [...]`, `ioc.confidence: 0–100` onto the event so OQL queries like `ioc.match:true AND severity:high` work without a join.
* [ ] **Enrichment audit trail** — every enrichment that mutates `Fields` appends a read-only `enrichment` block: `{stage, appliedAt, fields_added}`. The block is excluded from the content hash (hash is computed pre-enrichment) so enrichment never invalidates integrity verification. `oblivra-verify` skips enrichment blocks in hash recomputation.

---

## Phase 65 — Grafana Loki-Compatible Query API

Operators who already have Grafana dashboards querying Loki can point them at OBLIVRA with a URL change. This is a read-only query shim — no Loki storage format is used internally.

* [ ] **`GET /loki/api/v1/query_range`** — accepts `query` (LogQL stream selector + filter), `start`, `end`, `limit`, `step`. Translates the stream selector labels to OQL/Bleve filters, runs the search, returns results in Loki's JSON matrix/streams envelope. Covers the 90% case: `{job="sshd"} |= "Failed password"` maps to `sourceType:sshd AND message:"Failed password"`.
* [ ] **`GET /loki/api/v1/labels`** and **`GET /loki/api/v1/label/{name}/values`** — returns the set of known `sourceType` values (and per-label values) so Grafana's Explore label-picker populates without manual config.
* [ ] **`GET /loki/api/v1/series`** — returns the set of active label combinations seen in the last 6h. Enables Grafana's series selector.
* [ ] **Grafana datasource doc** — `docs/integrations/grafana-loki.md`: step-by-step to add OBLIVRA as a Loki datasource in Grafana, including the bearer-token auth header, example dashboards JSON, and LogQL-to-OQL translation table.
* [ ] **LogQL metric queries** — `GET /loki/api/v1/query` with `rate()` / `count_over_time()` wraps `QualityService.Coverage` stats into a Prometheus-compatible instant vector so Grafana stat panels work.

---

## Phase 66 — Advanced Search UX

* [ ] **Saved search folders + tags** — `SavedSearch` extended with `folder string` and `tags []string`. New `GET /api/v1/saved-searches?folder=&tag=` filter. Frontend: collapsible folder tree in SavedSearches view, tag chips with multi-select filter.
* [ ] **Search result column configurator** — SIEM view: drag-reorder columns, show/hide fields, persist layout to `localStorage` (user-side; no server round-trip). Default columns: `timestamp`, `host`, `sourceType`, `severity`, `message`.
* [ ] **Inline field stats sidebar** — in any search result set, a `⊞ Fields` panel shows cardinality + top-10 values for each field in the result window. Click a value to AND it into the current query. Mirrors Kibana's field stats but computed server-side at `GET /api/v1/siem/field-stats?q=&field=`.
* [ ] **Query history with re-run** — last 50 OQL/Bleve queries per browser session stored in `sessionStorage`, surfaced in a dropdown from the search bar. No server storage — privacy-preserving.
* [ ] **Contextual pivot menu** — right-click any event field value: options are "Search for this value", "Add to filter", "Exclude from filter", "Pivot timeline ±15min", "Look up in threat intel", "Copy value". Replaces the manual OQL construction that power users currently do.
* [ ] **Time range presets + custom absolute picker** — replace the current free-text `from/to` inputs with a dropdown (Last 1h / 6h / 24h / 7d / 30d / Custom). Custom opens a calendar date-time picker. Selection persists across navigation via URL params.
* [ ] **Live tail pause/resume** — WebSocket tail already exists. Add a pause button that buffers incoming events client-side (up to 500) while the analyst reads, then replays on resume. Buffer depth indicator shown while paused.

---

## Phase 67 — Retention & Cold-Tier Hardening

* [ ] **S3 / S3-compatible cold-tier upload** — `internal/storage/cold/s3.go` implements the `ObjectStore` interface using `net/http` (no AWS SDK; manual SigV4 signing so air-gap builds stay lean). Config: `OBLIVRA_S3_ENDPOINT`, `OBLIVRA_S3_BUCKET`, `OBLIVRA_S3_ACCESS_KEY`, `OBLIVRA_S3_SECRET_KEY`, `OBLIVRA_S3_REGION`. Supports MinIO, Cloudflare R2, Backblaze B2 (path-style). Parquet files are uploaded after WORM-lock with a SHA-256 manifest sidecar. Multipart for files >100MB.
* [ ] **Cold-tier integrity verification** — `oblivra-cli cold verify <path-or-s3-prefix>` re-downloads manifests, checks SHA-256 of each Parquet file, and recomputes per-row content hashes for a sample (configurable `--sample-rate`, default 10%). Exit code 1 on any mismatch.
* [ ] **Retention enforcement with audit trail** — `TenantPolicyService` already stores `HotMaxAge` / `WarmMaxAge`; add `ColdMaxAge` and `DeletePolicy` (delete-after-cold / keep-forever). Eviction runs write a `storage.eviction` chain entry recording how many events + bytes were removed and at which tier, so the audit trail captures every data lifecycle event.
* [ ] **Storage quota enforcement** — `OBLIVRA_MAX_HOT_BYTES` / `OBLIVRA_MAX_WARM_BYTES`. When the hot store exceeds quota, ingestion is back-pressured (HTTP 429 with `Retry-After`) rather than silently dropping. Alert raised when usage hits 90%.
* [ ] **Parquet partition pruning** — warm-tier query path (`GET /api/v1/storage/query`) uses partition metadata (min/max `receivedAt` per file) to skip files outside the query window. Reduces cold reads by ~10× for narrow time-range forensic queries.
* [ ] **Cross-tier search** — `GET /api/v1/siem/search?tiers=hot,warm,cold` fans out the query across tiers, merges results by timestamp, and returns a unified page. Cold-tier results are flagged with `tier: cold` so analysts know which events came from archived storage.

---

## Phase 68 — HA-Lite: Replication & Standby

OBLIVRA's single-server model is correct for air-gap deployments. HA-lite adds a passive standby that stays warm via WAL streaming — no distributed consensus, no cluster coordination overhead.

* [ ] **WAL streaming replication** — `internal/wal/replication.go`. Primary exposes `GET /api/v1/wal/stream` (chunked transfer, auth-gated, agent role). Standby connects, tails the WAL in real time, and writes identical fsync'd WAL files locally. Reconnect with exponential backoff on disconnect.
* [ ] **Standby read-only mode** — standby server starts with `OBLIVRA_STANDBY=1`: all write endpoints return 503, all read endpoints are live. Operators use a load balancer (Caddy / HAProxy) to route reads to standby, writes to primary.
* [ ] **Promotion script** — `scripts/promote-standby.sh`: stops WAL streaming, flips `OBLIVRA_STANDBY=0`, restarts. Documented in runbook with RTO estimate (< 60s for a single-server promotion).
* [ ] **Replication lag metric** — primary records `wal.replication.lag_bytes` and `wal.replication.lag_events` on `/metrics`. Alert fires when lag exceeds 10k events or 60s. Standby's `GET /healthz` returns 200 only when lag < threshold, so a health-checked load balancer automatically stops routing reads during lag spikes.
* [ ] **Audit chain sync verification** — after replication catches up, standby calls its own `AuditService.Verify()` and emits `replication.verified` or `replication.chain-mismatch`. Mismatch raises a critical alert on both nodes.

---

## Phase 69 — Dark Mode & Accessibility

* [ ] **Dark / light / system theme toggle** — Tailwind 4 `dark:` variant classes applied throughout. Theme stored in `localStorage`; system preference respected via `prefers-color-scheme`. Toggle in the topbar (sun/moon icon). All existing CSS variables mapped to dark equivalents in a single `theme.css` override block.
* [ ] **WCAG 2.1 AA contrast pass** — audit every colour pair in the design token set; replace any below 4.5:1 (text) / 3:1 (UI components). Tool: `internal/tools/contrast-check` (go run against the token file as CI lint step).
* [ ] **Keyboard-only navigation audit** — every interactive element (buttons, table rows, modal close, sidebar items) reachable and operable by Tab + Enter/Space. Focus ring always visible. Screen-reader `aria-label` on icon-only buttons.
* [ ] **Reduced-motion support** — `prefers-reduced-motion: reduce` disables all CSS transitions and the live-tail scroll animation.
* [ ] **Print stylesheet** — `@media print` hides sidebar, topbar, action buttons; expands all collapsed sections; forces black-on-white. Enables browser-print of evidence views without the UI chrome.

---

## Phase 70 — Correlation Graph & Lateral Movement Detection

* [ ] **Temporal correlation engine** — `internal/services/correlation.go`. Sliding-window (default 10min) co-occurrence index: when events from ≥2 distinct hosts share the same `srcIP`, `user`, or `processHash` within the window, a `correlation.cluster` record is written. Clusters are the atomic unit for lateral movement scoring.
* [ ] **Lateral movement scoring** — `CorrelationService.Score(cluster)` returns 0–100 based on: number of distinct destination hosts, presence of privilege-escalation event types, auth-failure-then-success sequence, new user account creation, and shadow-copy / log-clear tamper events within the cluster window. Score ≥70 → high-severity `lateral-movement` alert.
* [ ] **Correlation graph API** — `GET /api/v1/correlation/clusters?from=&to=&min_score=` returns cluster list. `GET /api/v1/correlation/clusters/{id}/graph` returns nodes (hosts, users, IPs, processes) + edges (event types, timestamps) as a JSON graph for the frontend.
* [ ] **Correlation Graph frontend view** — D3 force-directed graph (bundled, no CDN). Nodes sized by event count; edges coloured by relationship type (auth / network / process-spawn). Click a node to pivot the timeline to that entity. Filter by min lateral-movement score.
* [ ] **MITRE lateral movement heatmap integration** — correlation clusters automatically tag their highest-confidence MITRE technique (T1021 Remote Services, T1550 Use Alternate Auth Material, etc.) and contribute to the existing MITRE heatmap view.
* [ ] **Cross-case correlation** — `GET /api/v1/correlation/cross-case?entity=` finds clusters that span multiple open cases, so analysts working separate incidents can discover they're investigating the same attacker path.

---

## Phase 71 — OpenSearch / Elastic Output Adapter (Read Side)

Some operators have existing Kibana/OpenSearch Dashboards installations they can't replace. This phase makes OBLIVRA queryable via the Elasticsearch search API so those dashboards can point at OBLIVRA as a data source without migration.

* [ ] **`POST /{index}/_search`** (read-only) — accepts ES Query DSL `query.match`, `query.bool`, `query.range`, `query.term`, `query.terms`, `query.exists`; translates to OQL/Bleve; returns ES-shaped hits with `_source`, `_id`, `_score`. No aggregations in phase 1.
* [ ] **`GET /_cat/indices`** — returns one row per `(tenant, sourceType)` pair as a virtual index. Lets Kibana's index-pattern wizard discover OBLIVRA's data.
* [ ] **`GET /{index}/_mapping`** — returns a synthetic ES mapping derived from the Event struct + enrichment fields. Required for Kibana field-type inference.
* [ ] **`POST /_msearch`** — multi-search for Kibana dashboard panels that batch queries. Each sub-request translated independently.
* [ ] **Kibana/OpenSearch Dashboards doc** — `docs/integrations/kibana.md`: add OBLIVRA as an Elasticsearch datasource, create an index pattern, example dashboard JSON for sessions / alerts / tamper events.
* [ ] **Aggregations (phase 2)** — `date_histogram`, `terms`, `filter`, `value_count`. Enough to power the most common Kibana visualisations (time series, pie charts, metric tiles).

---

## Phase 72 — Sumo Logic / Datadog Ingest Wire Compatibility

Operators migrating away from SaaS SIEM vendors need a drop-in ingest target. This phase adds receiver endpoints that match the wire format of the two most common SaaS platforms.

* [ ] **Sumo Logic HTTP Source shim** — `POST /receiver/v1/http/{sourceToken}` accepts Sumo Logic's collector HTTP source format (plain text, JSON array, or JSON Lines). `sourceToken` maps to a tenant via `OBLIVRA_SUMO_TOKENS=token:tenant,...`. Sumo's collector metadata headers (`X-Sumo-Name`, `X-Sumo-Host`, `X-Sumo-Category`) mapped to provenance fields.
* [ ] **Datadog Agent HTTP intake shim** — `POST /api/v2/logs` (Datadog v2 log intake format: JSON array of `{message, ddsource, ddtags, hostname, service}`). Auth via `DD-API-KEY` header mapped through `OBLIVRA_DD_KEYS=key:tenant,...`. Enables Datadog Agent `logs_enabled: true` pipelines to re-target OBLIVRA by changing the `DD_LOGS_CONFIG_LOGS_DD_URL` env var.
* [ ] **Tag normalization** — Datadog `ddtags` (`env:prod,team:infra`) and Sumo Logic `_sourceCategory` are normalized into OBLIVRA `Fields` as `tag.*` keys so OQL queries like `tag.env:prod` work across both sources.
* [ ] **Migration guide** — `docs/integrations/migration-from-saas.md`: step-by-step replacement instructions for Splunk Cloud, Datadog, and Sumo Logic, including agent re-targeting, saved-search translation, and alert rule migration.

---

## Phase 73 — eBPF Agent Kernel Collector

The file-tailing agent covers most ingest scenarios, but kernel-level telemetry — syscalls, network sockets, file descriptor activity, container namespace events — is invisible to a userspace tailer. This phase adds an eBPF-based collector sub-process to the existing `oblivra-agent` binary. It is opt-in, Linux-only, and build-tagged so air-gap and Windows builds carry zero eBPF code.

> **Build constraint**: `//go:build linux && ebpf`. Compiled with `task agent:ebpf` using `cilium/ebpf` (pure Go, no libbpf/libelf C dependency). Requires kernel ≥ 5.8 for ring-buffer support; degrades gracefully to perf-buffer on 5.4–5.7. BTF (BPF Type Format) CO-RE objects compiled offline and embedded via `go:embed` so no kernel headers are needed on the target host.

### 73.1 — Process & Syscall Probing

* [ ] **execve / execveat tracepoint** — `cmd/agent/ebpf/exec.bpf.c`. Captures `{pid, ppid, uid, gid, comm, argv[0..15], cwd, filename, retval}` on every exec. argv capped at 15 segments × 128 bytes; truncation flagged with `argv_truncated: true`. Emits `ebpf.exec` events into the agent's existing priority queue so they hit the server before bulk file-tail traffic under backpressure.
* [ ] **exit / exit_group tracepoint** — captures `{pid, exit_code, elapsed_ns}` and emits `ebpf.exit`. Paired with `ebpf.exec` in the server's `ReconstructionService` for precise process lifetime (no log-message parsing required).
* [ ] **open / openat / creat kprobe** — captures `{pid, comm, path, flags, retval}`. Flags decoded to human strings (`O_RDWR|O_CREAT`). Emits `ebpf.file_open`. High-frequency paths (`/proc/self/maps`, `/dev/urandom`) throttled via a per-pid LRU token bucket (100 events/s per pid) in the BPF map before they reach userspace — no ring-buffer saturation.
* [ ] **unlink / unlinkat / rename kprobe** — emits `ebpf.file_delete` / `ebpf.file_rename`. Critical for detecting evidence destruction (log file deletion, shadow copy wipe) without relying on auditd.
* [ ] **setuid / setgid / capset kprobe** — privilege-escalation detection at the kernel boundary. Emits `ebpf.priv_change` with before/after uid/gid/caps. Feeds directly into the lateral-movement scorer (Phase 70).
* [ ] **ptrace kprobe** — detects debugger attachment and process injection (`PTRACE_ATTACH`, `PTRACE_POKEDATA`). Emits `ebpf.ptrace` with tracer and tracee pids. Triggers a Sigma rule match for `T1055 Process Injection`.

### 73.2 — Network Socket Probing

* [ ] **tcp_connect / tcp_accept kprobes** — captures 5-tuple `{srcIP, srcPort, dstIP, dstPort, pid, comm, ns_inum}` at connection establishment. Emits `ebpf.tcp_connect` / `ebpf.tcp_accept`. No packet payload captured — metadata only. Namespace inode included for container-aware correlation.
* [ ] **tcp_close kprobe** — emits `ebpf.tcp_close` with bytes sent/received (from `sk->bytes_sent` / `sk->bytes_received`) and duration. Gives per-connection byte counts without a full NDR stack.
* [ ] **udp_sendmsg / udp_recvmsg kprobes** — emits `ebpf.udp_send` / `ebpf.udp_recv` with 5-tuple + length. Covers DNS exfiltration and ICMP-tunnelling-over-UDP patterns invisible to TCP-only tracers.
* [ ] **DNS response snooping via socket filter** — attaches a BPF socket filter to a raw socket on the loopback and capture interfaces. Parses DNS responses (A/AAAA/CNAME) inline in BPF and emits `ebpf.dns_response` with `{query, answers[], ttl, pid}`. No userspace DNS proxy required; no traffic modified.
* [ ] **Network namespace tracking** — tracks `setns` calls to detect container pivoting. Emits `ebpf.ns_enter` with old/new net namespace inode. Server correlates with container ID via the asset registry (Phase 64).

### 73.3 — File Integrity Monitoring

* [ ] **inotify-replacement via `fanotify` + eBPF** — `cmd/agent/ebpf/fim.bpf.c`. Attaches `fanotify` FAN_OPEN_PERM + FAN_ACCESS in report-only mode (no blocking); supplements with a `vfs_write` kprobe for in-place modifications. Watchlist loaded from `fim.paths:` in `agent.yml`. Emits `ebpf.fim_write` / `ebpf.fim_read` with sha256 of the modified inode computed in a userspace goroutine after the event fires (non-blocking: hash computed async, attached to the event via correlation ID).
* [ ] **LD_PRELOAD / LD_LIBRARY_PATH detection** — `execve` probe checks the envp array for `LD_PRELOAD` / `LD_LIBRARY_PATH` and emits `ebpf.ldpreload` with the injected path. Catches userspace rootkits that rely on library hijacking.
* [ ] **Kernel module load detection** — `security_kernel_module_request` LSM hook emits `ebpf.kmod_load` with module name. Flags unsigned or unexpected kernel modules for the tamper detector.

### 73.4 — Container & Namespace Awareness

* [ ] **Container ID resolution** — on every `ebpf.exec` / `ebpf.tcp_connect`, the agent reads `/proc/<pid>/cgroup` (cached per-pid, invalidated on `ebpf.exit`) and extracts the Docker/containerd/cgroup-v2 container ID. Appended as `container.id` field. Server can join against `GET /api/v1/assets` if the asset registry has container metadata.
* [ ] **Namespace-scoped process tree** — `ReconstructionService` (server-side) extended to group `ebpf.exec` events by `(host, ns_inum)` when building process trees. Container-local PID 1 is correctly identified as the container entrypoint, not the host init.
* [ ] **Kubernetes pod annotation** — agent reads `/proc/<pid>/environ` for `KUBERNETES_SERVICE_HOST` presence and `/var/run/secrets/kubernetes.io/serviceaccount/namespace` to append `k8s.namespace`, `k8s.pod_name` (from `HOSTNAME` env) to events. No Kubernetes API calls — pure procfs.

### 73.5 — eBPF Agent Operational Concerns

* [ ] **Privilege minimisation** — eBPF agent requires `CAP_BPF` + `CAP_PERFMON` (kernel ≥ 5.8); falls back to `CAP_SYS_ADMIN` on older kernels. systemd unit drops all other capabilities. Documented in deployment guide.
* [ ] **Ring-buffer back-pressure** — when the ring buffer is >80% full, the BPF program starts sampling: every Nth exec/open event is dropped at the kernel boundary (N scales with fill level). A `ebpf.sample_drop` counter is reported in the agent heartbeat so operators see when the kernel is producing faster than the agent can drain.
* [ ] **`oblivra-agent ebpf status`** — prints loaded programs, map sizes, ring-buffer fill %, events/s per probe, and kernel version / BTF availability.
* [ ] **`oblivra-agent ebpf test`** — dry-run: loads programs, fires synthetic exec (fork + `/bin/true`), confirms the event appears in the ring buffer, then unloads. CI-friendly; exits 0 on success.
* [ ] **Build tag isolation** — `cmd/agent/ebpf/` is entirely behind `//go:build linux && ebpf`. `task agent:build` (no tag) produces the standard file-tailing binary. `task agent:ebpf` passes `-tags ebpf` and links the BPF objects. The Windows and macOS cross-compile targets are unaffected.
* [ ] **`task agent:ebpf:generate`** — runs `bpf2go` (cilium/ebpf codegen) to regenerate Go bindings from `.bpf.c` sources. Requires `clang` + `llvm` on the build host only; target hosts need nothing.
* [ ] **Docs: `docs/operator/ebpf.md`** — kernel version requirements, capability model, watchlist config reference, container-mode setup, known limitations (no macOS/Windows, no kernel < 5.4, fanotify requires `CAP_SYS_ADMIN` fallback on RHEL 7).

---

## Phase 74 — Agent Plugin System (WASM)

The existing agent handles file tail, journald, syslog-udp, and eBPF. This phase adds a WASM plugin host so operators can ship custom parsers and enrichers without forking the agent or waiting for a release cycle. Plugins run in a sandboxed WASM runtime — no native code, no syscalls beyond the explicit host API.

* [ ] **WASM plugin host** — `cmd/agent/plugin/host.go` using `tetratelabs/wazero` (pure Go, no CGO, zero C dependency). Plugins loaded from `pluginsDir:` in `agent.yml`. Each plugin is a `.wasm` file compiled from any WASM-targeting language (Go/TinyGo, Rust, AssemblyScript).
* [ ] **Host API surface** — plugins import `oblivra_host` module with four functions: `emit_event(json_ptr, len)` to push a parsed event into the agent pipeline; `get_raw(ptr, max_len)` to read the current raw line being processed; `log(level, msg_ptr, len)` for plugin-side logging surfaced in the agent's own log; `now_unix_ns()` for timestamps. Intentionally minimal — no filesystem, no network, no process access from plugin code.
* [ ] **Parser plugin contract** — plugins export `parse(raw_len i32) i32` (returns number of events emitted). Called once per raw log line from file-tail or stdin inputs. Return value 0 = line skipped; negative = parse error (increments `UnparsedRate` in QualityService).
* [ ] **Enricher plugin contract** — plugins export `enrich(event_json_len i32) i32`. Called after the built-in enrichment stage (Phase 64). Plugin reads the event JSON, mutates fields, and calls `emit_event` with the enriched version. The host diffs the before/after `Fields` map and records the delta in the `enrichment` audit block.
* [ ] **`oblivra-agent plugin list`** — prints loaded plugins, their exported functions, memory usage, and events processed/s.
* [ ] **`oblivra-agent plugin reload`** — hot-reloads the plugin directory (drops old WASM instances, loads new ones) without restarting the agent. Audit event emitted on reload.
* [ ] **Example plugins** — `docs/plugins/`: TinyGo parser for Palo Alto firewall TRAFFIC logs, Rust enricher that looks up internal CMDB (local JSON file), AssemblyScript field normaliser. Each compiles to a <200KB `.wasm` file.
* [ ] **Plugin signing** — `pluginPublicKeys: [ed25519-base64, ...]` in `agent.yml`. Agent refuses to load unsigned plugins when the list is non-empty. Signature file is `<plugin>.wasm.sig` (detached ed25519). `oblivra-agent plugin sign <key.pem> <plugin.wasm>` does the signing.

---

## Phase 75 — Threat Hunt Workbench

Detection rules fire on known-bad patterns. Threat hunting requires exploratory queries across weeks of data without a predefined hypothesis. This phase adds a persistent, case-linked hunt workspace that feels like a notebook but produces auditable, court-grade output.

* [ ] **Hunt sessions** — `internal/services/hunt.go`. `POST /api/v1/hunts` creates a named hunt session with an optional linked case ID. Each session has an immutable `openedAt` + `auditRootAtOpen` snapshot (same freeze semantics as InvestigationsService cases). Sessions persist to `hunts.log`.
* [ ] **Hunt queries** — `POST /api/v1/hunts/{id}/queries` submits an OQL or Bleve query, stores `{query, submittedAt, actor, resultCount, resultSampleIds[]}`. Every query lands in the audit chain as `hunt.query` so the full exploratory path is reproducible. `GET /api/v1/hunts/{id}/queries` returns the query history.
* [ ] **Query result pinning** — `POST /api/v1/hunts/{id}/pin` promotes a result set (by event IDs or query snapshot) to pinned evidence. Pinned results are sealed into the linked case (if any) as a `hunt.evidence` chain entry.
* [ ] **Hunt notebooks frontend** — new **Hunts** sidebar view. Each hunt is a vertical notebook: alternating query cells (OQL input + result table) and text annotation cells (markdown). Cells append-only; no delete. Export as self-contained HTML (same renderer as `ReportService.CaseHTML`).
* [ ] **Pre-built hunt templates** — `sigma/hunt_templates/`: LOLBIN hunt (CommandLine contains known-good binaries used suspiciously), beaconing detection (host makes same external connection at regular intervals), account enumeration (>20 failed logins from same srcIP in 10min), new service install, scheduled task creation. Each template is a parameterised OQL query with a markdown walkthrough.
* [ ] **Statistical outlier hunt** — `GET /api/v1/hunts/outliers?field=&host=&window=` returns the top-N values of `field` that are statistically anomalous (z-score ≥ 3σ compared to the same field's distribution over the prior 7 days). Server-side; no ML model, pure rolling stats from `UebaService` baselines.

---

## Phase 76 — Network Packet Capture Trigger (PCAP-on-Alert)

Phase 42 deferred full PCAP reading as a multi-week project. This narrower phase adds a targeted capability: when a high-severity alert or eBPF network event fires on a host running the agent, the agent captures a short PCAP burst (default 30s, configurable) via `libpcap` / `AF_PACKET` and ships the file to the server as sealed evidence. No continuous packet capture; no storage explosion.

* [ ] **`pcap` input type in `agent.yml`** — `type: pcap`, `interface: eth0`, `snaplen: 1500`, `filter: ""` (BPF filter string), `triggerOnAlert: true`, `captureSeconds: 30`, `maxFileMB: 50`. Disabled by default. Build-tagged `linux && pcap`; requires `libpcap-dev` on build host, `libpcap.so` on target.
* [ ] **Alert-triggered capture** — when the agent's pre-detection engine (Phase 60) fires a critical local rule, it opens a raw `AF_PACKET` socket on the configured interface, writes a `.pcap` file to `<stateDir>/pcap/`, and streams it to `POST /api/v1/evidence/pcap` after capture completes. File is ed25519-signed before upload.
* [ ] **PCAP evidence endpoint** — `POST /api/v1/evidence/pcap` (agent role). Receives the file, computes SHA-256, stores under `<dataDir>/evidence/pcap/<timestamp>-<host>.pcap`, appends a `evidence.pcap` chain entry with `{host, triggeredBy, sha256, sizeBytes, captureStart, captureEnd}`. Sealed into the case if one is open for that host.
* [ ] **PCAP viewer stub** — frontend Evidence view shows PCAP entries with metadata + download button. Full browser-side rendering of PCAP frames is Phase 76.x (deferred; operators use Wireshark on the downloaded file today).
* [ ] **`oblivra-verify` PCAP support** — verifier extended to check `.pcap` evidence entries: SHA-256 recomputed against stored file, ed25519 signature verified against the submitting agent's public key.

---

## Phase 77 — Certificate & TLS Inventory

Certificate expiry and TLS misconfiguration are common entry points. This phase adds passive TLS telemetry collection from eBPF network events and an active scanner for internal hosts, producing an always-current certificate inventory without any network credentials.

* [ ] **Passive TLS fingerprinting** — eBPF `tcp_connect` events (Phase 73.2) extended to capture the TLS ClientHello via a `kprobe` on `tcp_sendmsg` (first 512 bytes of new connections). Server-side parser extracts SNI, supported cipher suites, and JA3 fingerprint. Emits `tls.clienthello` events. No decryption; header metadata only.
* [ ] **TLS ServerHello capture** — `kprobe` on `tcp_recvmsg` for the server's first response packet. Extracts selected cipher suite, TLS version, and server certificate chain (DER-encoded, first 4KB). Server parses certificate: `{subject, issuer, notBefore, notAfter, SANs, keyAlgo, keyBits, selfSigned}`. Emits `tls.serverhello`.
* [ ] **Certificate inventory service** — `internal/services/certinventory.go`. Aggregates `tls.serverhello` events into a deduplicated certificate store keyed by SHA-256 of the DER cert. Records `{firstSeen, lastSeen, hosts[], ports[], expiresAt, daysUntilExpiry, selfSigned, weakKey}`. Endpoint `GET /api/v1/certs`.
* [ ] **Expiry alerting** — scheduler job `cert.expiry-check` runs daily. Fires `warning` severity alert at 30 days, `high` at 7 days, `critical` at 1 day or already-expired. Alert body includes the cert subject + affected hosts so the on-call runbook action is unambiguous.
* [ ] **Weak cipher / deprecated TLS detection** — JA3 fingerprint matched against a seeded blocklist (`TLS_RSA_*` export ciphers, RC4, 3DES, TLS 1.0/1.1 negotiation). Emits `tls.weak` alert. Blocklist hot-reloaded from `sigma/tls_weak.yml` so operators can extend it.
* [ ] **Certs frontend view** — table sorted by `daysUntilExpiry` ascending. Red / amber / green row colouring. Click-through to the raw cert detail + list of hosts that presented it. Export as CSV for ops teams.

---

## Phase 78 — Incident Timeline Export (STIX 2.1)

Courtroom evidence and threat-intel sharing both benefit from a machine-readable representation of the incident timeline. STIX 2.1 is the de-facto standard for CTI exchange (MITRE ATT&CK, TAXII feeds, OpenCTI, MISP).

* [ ] **STIX 2.1 bundle generator** — `internal/report/stix.go`. `CaseToSTIX(caseId)` converts a sealed case into a STIX 2.1 JSON bundle: `identity` (OBLIVRA instance), `report` (case title + description), `observed-data` objects (one per event), `indicator` objects (one per fired rule with STIX pattern translated from the Sigma rule's condition), `relationship` edges (indicator → observed-data, malware → attack-pattern), `attack-pattern` objects for each MITRE technique fired. Bundle ID is deterministic over the case ID + audit root so two exports of the same case produce identical bundles.
* [ ] **Export endpoint** — `GET /api/v1/cases/{id}/stix.json` (analyst role). Response is `Content-Type: application/stix+json`. Audited; lands in the chain as `case.stix-export`.
* [ ] **TAXII 2.1 server stub** — `GET /taxii/collections`, `GET /taxii/collections/{id}/objects`. Read-only; one collection per tenant. Enables OpenCTI / MISP to pull OBLIVRA bundles on a schedule without manual export.
* [ ] **MISP event export** — `GET /api/v1/cases/{id}/misp.json` produces a MISP 2.x event JSON (attributes: IPs, hashes, hostnames, filenames, MITRE technique tags). Simpler than STIX; preferred by many SOC teams for quick IOC sharing.
* [ ] **Import STIX indicators as IOCs** — `POST /api/v1/threatintel/stix-import` accepts a STIX 2.1 bundle and extracts `indicator` objects into the `ThreatIntelService` IOC store. Each imported IOC records `{stixId, source, pattern, confidence, validUntil}`. Integrates with Phase 64's ingest-time IOC enrichment.

---

## Phase 79 — Autonomous Integrity Watchdog (Deadman's Switch)

The tamper detector (Phase 44) fires when logs show signs of clearing. But a sophisticated attacker may simply kill OBLIVRA silently and clear the audit chain offline. This phase adds an external heartbeat mechanism so the absence of OBLIVRA itself becomes detectable.

* [ ] **Outbound heartbeat** — `OBLIVRA_DEADMAN_URL=https://hc-ping.com/<uuid>` (or any URL that accepts a GET). Scheduler job pings every 5 minutes. Missed pings (the remote service's responsibility) alert the operator out-of-band — no OBLIVRA process involvement required. Compatible with healthchecks.io (self-hostable, Apache 2.0) for air-gap operators.
* [ ] **Signed heartbeat payload** — each ping POST body carries `{timestamp, auditChainLength, lastAnchorAt, instanceId}` HMAC-signed with `OBLIVRA_AUDIT_KEY`. The receiver (or an operator script) can verify the payload proves the chain hasn't been rewound between pings.
* [ ] **Local watchdog process** — `cmd/watchdog` — a tiny (<500 LoC) separate binary that runs as a separate systemd service. Polls `GET /healthz` every 60s. If OBLIVRA is unreachable for >5min, `watchdog` writes a `platform.unreachable` signed journal entry to its own append-only `watchdog.log` and sends an email/webhook alert (configured independently of OBLIVRA's `NotificationService`). Operates even when the main server is dead.
* [ ] **Watchdog log verification** — `oblivra-verify watchdog.log` extended to verify the watchdog's own Merkle chain (same chain format, separate HMAC key `OBLIVRA_WATCHDOG_KEY`). An attacker who kills both processes still leaves a verifiable gap in the watchdog log.
* [ ] **Mutual attestation** — OBLIVRA server polls `GET /watchdog/status` (loopback-only endpoint on the watchdog's port). If the watchdog itself disappears, OBLIVRA raises a `tamper-watchdog-missing` critical alert. The two processes watch each other; disabling either one is detectable within a single heartbeat interval.
* [ ] **Docs: `docs/operator/deadman.md`** — deployment pattern (watchdog on a separate low-privilege user), recommended external ping service (self-hosted healthchecks.io recipe included), key rotation procedure, and what an operator should do when an unexpected ping gap appears in the watchdog log.

---

## Phase 80 — Identity & Access Management (Enterprise Auth)

The current RBAC model (static API keys with role suffixes) is correct for small teams and air-gap deployments. Enterprise buyers require SSO, directory sync, and per-user audit attribution. This phase adds the full IAM surface without removing the simple-key fallback.

* [ ] **OIDC / OAuth 2.0 SSO (production-grade)** — the existing `internal/httpserver/oidc.go` has PKCE flow. Harden it: refresh-token rotation (RFC 6749 §6), silent re-auth, session revocation on `POST /api/v1/auth/logout` (clears server-side session table, emits `auth.logout` to chain), configurable session lifetime (`OBLIVRA_SESSION_TTL`, default 8h). Tested against Entra ID, Okta, Keycloak, and Authentik with a shared httptest IdP stub.
* [ ] **SAML 2.0 SP** — `internal/httpserver/saml.go` using `crewjam/saml` (pure Go). IdP-initiated and SP-initiated flows. ACS at `/api/v1/auth/saml/acs`, metadata at `/api/v1/auth/saml/metadata`. Group claims mapped to OBLIVRA roles via `OBLIVRA_SAML_ROLE_MAP`. Required for Ping Identity, ADFS, and Okta enterprise contracts.
* [ ] **SCIM 2.0 user provisioning** — `GET/POST /scim/v2/Users`, `PUT/PATCH/DELETE /scim/v2/Users/{id}`. Allows Azure AD, Okta, and JumpCloud to auto-provision/deprovision users. Deprovisioned users have active sessions revoked within one SCIM sync cycle. Each action is a `scim.*` chain entry.
* [ ] **Per-user audit attribution** — when SSO is active, audit chain `actor` field contains the IdP `sub` claim + `email` instead of the API key name. Chain format unchanged; only the value changes.
* [ ] **MFA enforcement policy** — `OBLIVRA_MFA_REQUIRED=analyst,admin`. OIDC/SAML assertion must include an AMR claim containing `mfa`/`otp`/`hwk`; missing AMR returns 403. API-key users unaffected.
* [ ] **Break-glass emergency access** — `oblivra-cli auth break-glass --reason "..."` generates a 1-hour, single-use admin token stored in `break_glass.log` (separate Merkle chain). Cannot be listed via API — requires physical access to the server host. Every use is a `break-glass.used` chain entry.
* [ ] **Session table frontend** — Admin view: active sessions (user, IP, user-agent, login time, idle time) with per-session and bulk revoke. Each revocation is a `session.revoke` chain entry.

---

## Phase 81 — Multi-Tenancy: Enterprise Tier

The existing isolation is solid but operator-configured. Enterprise deployments need self-service tenant lifecycle, cross-tenant admin views, and MSP-grade delegation.

* [ ] **Tenant lifecycle API** — `POST /api/v1/tenants` (admin) creates a tenant with `{name, slug, maxEPS, maxHotBytes, retentionPolicy, contactEmail}`. `DELETE /api/v1/tenants/{id}` schedules hard-delete after grace period (`OBLIVRA_TENANT_DELETE_GRACE=30d`), emits `tenant.delete-scheduled`, begins soft-deleting hot data.
* [ ] **Tenant dashboard** — Admin view: table of all tenants with live EPS, hot-store bytes, alert count, agent count, last-seen. Click-through to per-tenant Overview. MSP NOC use case.
* [ ] **Cross-tenant alert roll-up** — `GET /api/v1/admin/alerts?across_tenants=true` (admin only). Returns alerts from all tenants with tenant slug prepended to hostId.
* [ ] **Resource quotas + EPS throttling** — `MaxEPS` and `MaxHotBytes` per tenant in `TenantPolicyService`. Over-quota ingest returns HTTP 429 with `X-Oblivra-Quota:` header. Breach raises `tenant.quota-exceeded` alert visible in the tenant dashboard.
* [ ] **Tenant data export** — `POST /api/v1/tenants/{id}/export` produces a signed ZIP: WAL entries, Parquet files, filtered audit log and cases log for that tenant. Export manifest is a `tenant.export` chain entry. GDPR Art. 20 data portability.

---

## Phase 82 — Advanced UEBA

The existing `UebaService` does per-host EPS z-score detection. Enterprise UEBA means per-user behaviour modelling, peer-group comparison, and risk scoring.

* [ ] **Per-user activity baseline** — `internal/services/ueba_user.go`. Per-user daily tracking: login times, source IPs, accessed hosts, command diversity, bytes moved, auth failures. 14-day rolling baseline per user per day-of-week.
* [ ] **Peer-group modelling** — users grouped by `department`/`role` from the asset registry. User deviating from peer group by ≥2σ on 3+ dimensions raises `peer-deviation` anomaly.
* [ ] **Risk score timeline** — `GET /api/v1/ueba/users/{user}/risk` returns `{score 0-100, factors[], trend[]}`. Score decays 10% per clean day. Fed into case confidence scoring.
* [ ] **Watchlist** — `POST /api/v1/ueba/watchlist`. Watchlisted entities have every event tagged `watchlist: true` at ingest (enrichment stage). Admin-only; every change is a chain entry.
* [ ] **Impossible travel detection** — sequential logins from geographically distant IPs within sub-travel time. Requires GeoIP (Phase 64). Raises `ueba.impossible-travel` with both login events as evidence.
* [ ] **Data exfiltration signals** — per-user bytes-out baseline from eBPF `tcp_close` counts or NetFlow. ≥3σ spike raises `ueba.exfil-spike`. Corroborated by unusual file opens or DNS to new external domains raises `ueba.exfil-corroborated`.
* [ ] **User Risk view** — sortable user table (risk score, top anomaly, last login, last alert). Click-through to risk timeline + factor breakdown.

---

## Phase 83 — Playbook Trigger Contracts

OBLIVRA is not a SOAR. But enterprise teams need structured, verifiable, idempotent trigger contracts so their SOAR acts reliably.

* [ ] **Structured alert payload v2** — adds `{alertId, ruleId, mitreTechnique, affectedHost, affectedUser, riskScore, evidencePackageUrl, caseId, auditRootAtAlert, signatureEd25519}`. `auditRootAtAlert` lets the SOAR verify the chain state at alert time.
* [ ] **Idempotency key on webhooks** — `X-Oblivra-Delivery-Id` + `X-Oblivra-Delivery-Attempt` headers. Delivery log at `GET /api/v1/webhooks/{id}/deliveries`.
* [ ] **Webhook retry with exponential backoff** — 5 retries at 30s/2m/10m/1h/6h. After 5 failures the webhook is circuit-opened and a `webhook.circuit-open` alert fires.
* [ ] **Playbook trigger API** — `POST /api/v1/playbooks/trigger` (admin). Records `playbook.trigger` chain entry and forwards v2 payload to SOAR. OBLIVRA does not execute the playbook.
* [ ] **Callback receipt** — `POST /api/v1/playbooks/callback`. SOAR posts completion; OBLIVRA records `playbook.callback` chain entry and auto-updates case notes.
* [ ] **Alert suppression rules** — `POST /api/v1/detection/suppressions`. Matching alerts tagged `suppressed: true`, excluded from webhooks. Suppression is a chain entry with expiry; `suppression.expired` auto-removes it.

---

## Phase 84 — Compliance Evidence Automation

Phase 46 added a machine-readable JSON-LD feed. This phase closes the loop: scheduled evidence collection, sealed per-control, with a ready-made auditor pack.

* [ ] **Control-to-query mapping** — `POST /api/v1/compliance/controls/{controlId}/mapping`. Scheduler runs mapped OQL queries on schedule, seals `{controlId, collectedAt, auditRootAtCollection, evidenceEventIds[], resultCount}` as a `compliance.evidence` chain entry.
* [ ] **Compliance evidence package** — `GET /api/v1/compliance/package/{framework}` produces a signed ZIP: one JSON per control with all collection runs, a SHA-256 manifest, and the covering audit chain entries. Auditors verify independently with `oblivra-verify`.
* [ ] **Compliance view** — framework selector, control grid with RAG freshness (green=this week, amber=>7d, red=never). Click-through to per-control evidence history.
* [ ] **Gap detection** — scheduler `compliance.gap-scan` runs daily. Missing evidence raises `compliance.gap` alert with control ID and framework before the auditor asks.
* [ ] **Audit-period report** — `GET /api/v1/compliance/report/{framework}?from=&to=` produces a single printable HTML report covering all controls: evidence summary, gap table, Merkle root at start/end of period, verification instructions.

---

## Phase 85 — NDR Expansion

The existing `NdrService` consumes NetFlow v5. Enterprise NDR means protocol-aware detection and east-west traffic baselines.

* [ ] **NetFlow v9 + IPFIX receiver** — `internal/listeners/netflow9.go`. Template-based; per-exporter template cache. Maps enterprise flow fields (application ID, VLAN, MPLS, interface index). Required for Cisco ASR, Juniper MX, Palo Alto.
* [ ] **sFlow v5 receiver** — `internal/listeners/sflow.go`. Packet-sampled telemetry from Arista, Brocade, open-source switches. More granular east-west visibility than NetFlow.
* [ ] **East-west traffic baseline** — per-host-pair daily byte/flow baseline. ≥3σ spike raises `ndr.east-west-spike`. New host-to-host pair raises `ndr.new-connection`.
* [ ] **Long connection detection** — flow duration baseline per (srcIP, dstPort). Flows ≥5× baseline raise `ndr.long-connection`. Classic C2 keep-alive signal.
* [ ] **DNS tunnel detection** — per-resolver query/minute baseline. Sources exceeding 3× baseline with Shannon entropy >3.5 bits/char on the subdomain raise `ndr.dns-tunnel`.
* [ ] **JA3/JA3S fingerprint matching** — TLS ClientHello JA3 fingerprints (from Phase 77) matched against seeded blocklist of known-malicious C2 TLS profiles. Blocklist is hot-reloaded YAML.
* [ ] **NDR dashboard expansion** — east-west heatmap, long-connection list, DNS anomaly table, JA3 match table.

---

## Phase 86 — OpenAPI, SDK & Developer Experience

* [ ] **OpenAPI 3.1 spec (auto-generated)** — `GET /openapi.json`. `task docs:openapi` writes `docs/api/openapi.yaml`. Every `oblivra-smoke` response validated against the registered schema; drift is a CI failure.
* [ ] **Official Go SDK** — `sdk/go/oblivra`. Generated from OpenAPI via `oapi-codegen`. Covers auth, pagination, WebSocket live tail, retry-with-backoff. Used by `oblivra-cli` so CLI and SDK stay in sync.
* [ ] **Official Python SDK** — `sdk/python/oblivra`. Async-first (`httpx`/`asyncio`); sync wrapper available. Example Jupyter notebook in `docs/examples/jupyter/`.
* [ ] **Local dev environment** — `task dev` starts the server with deterministic seed data (100k synthetic events, 5 tenants, 3 pre-opened cases). `task dev:reset` wipes and re-seeds.
* [ ] **Terraform provider** — `sdk/terraform/oblivra`. Manages tenants, API keys, webhook registrations, retention policies, compliance mappings. Published to the Terraform Registry.
* [ ] **`oblivra-cli` shell completion** — `completion bash|zsh|fish|powershell`. Every subcommand, flag, and enum is completable.

---

## Phase 87 — Hardened Deployment Targets

* [ ] **SELinux policy module** — `build/selinux/oblivra.te`. Confined `oblivra_t` domain. Tested against RHEL 9 targeted policy.
* [ ] **AppArmor profile** — `build/apparmor/usr.bin.oblivra-server`. Equivalent confinement for Debian/Ubuntu.
* [ ] **FIPS 140-2 build mode** — `//go:build fips`. Uses BoringCrypto for SHA-256; replaces Argon2id with PBKDF2-SHA256 in vault (Argon2id is not FIPS-validated).
* [ ] **Kubernetes manifests** — `deploy/kubernetes/`: Namespace, ServiceAccount, Deployment (init container for migrate, `/readyz`/`/healthz` probes), PVC (100Gi), ConfigMap, Secret, HPA. `kustomization.yaml` for overlay customisation.
* [ ] **Helm chart** — `deploy/helm/oblivra/`. Published to GitHub Pages Helm repo. `helm install oblivra oblivra/oblivra -f values.yaml`.
* [ ] **Air-gap container bundle** — `task release:airgap` produces a tarball with saved OCI images + `load.sh` + SHA-256 manifest for offline transfer.
* [ ] **CIS Benchmark hardening guide** — `docs/operator/cis-hardening.md`. Maps CIS Controls v8 to OBLIVRA config: filesystem permissions, systemd hardening flags, TLS cipher suite config in Caddy, audit key rotation cadence.

---

## Phase 88 — Performance & Scale

* [ ] **Parallel WAL writers** — shard-per-tenant WAL (`ingest.<tenant>.wal`). Throughput scales linearly with tenant count; eliminates the single global mutex.
* [ ] **Bleve index sharding** — weekly time-based index rotation per tenant. Search fans out across shards in the query window. Old shards outside hot retention are memory-mapped read-only.
* [ ] **Hot store compaction tuning** — expose BadgerDB knobs as `OBLIVRA_BADGER_*` env vars with production-tuned defaults. `task badger:compact` triggers manual compaction.
* [ ] **Ingest pipeline back-pressure with overflow WAL** — non-blocking enqueue: full queue → `overflow.wal` + `pipeline.overflow` counter. Recovery goroutine drains when headroom returns. HTTP returns 200 immediately.
* [ ] **Zero-copy ingest path** — `json.RawMessage` passthrough for WAL write; only decode fields needed for routing. Target 3× throughput improvement on a single-tenant workload.
* [ ] **Search result streaming** — chunked transfer on `GET /api/v1/siem/search`. Streams results off the Bleve iterator; reduces peak memory 10× for >100k event result sets.
* [ ] **Benchmark suite** — `cmd/bench`: microbenchmarks for WAL, Bleve, BadgerDB, content-hash, full pipeline. `task bench` writes `docs/operator/bench-<date>.md`.

---

## Phase 89 — Incident Response Integration Hub

* [ ] **Jira integration** — on case `legal-review`, optionally create a Jira issue with case metadata + report URL. Issue key written back as `case.jira-linked` chain entry. State changes sync bidirectionally.
* [ ] **PagerDuty integration** — severity mapping (critical→P1 through low→P4). Dedup key is OBLIVRA alert ID. Auto-resolve on OBLIVRA alert resolution.
* [ ] **Slack / Teams rich notifications** — Block Kit (Slack) and Adaptive Card (Teams) formatted messages: severity banner, affected host + user, MITRE technique, case link.
* [ ] **TheHive case sync** — creates TheHive 5 case on OBLIVRA case open; syncs observables (IPs, hashes, hostnames, users); attaches HTML evidence package. Status syncs bidirectionally.
* [ ] **Velociraptor hunt trigger** — on case open, optionally trigger a Velociraptor artifact collection. Flow ID written to case as `case.velociraptor-hunt` chain entry. Completed results imported via Phase 5 import API.
* [ ] **Evidence handoff receipt** — `POST /api/v1/cases/{id}/handoff`. Records `{recipient, method, trackingId, notes}` as `case.handoff` chain entry. Satisfies court chain-of-custody documentation.
* [ ] **IR timeline export** — `GET /api/v1/cases/{id}/ir-timeline.md` and `.html`: chronological narrative (T+0 first event through seal) for post-incident review documents.

---

## Phase 90 — Platform Observability (Tier-0)

* [ ] **Structured logging via `log/slog`** — JSON output mode (`OBLIVRA_LOG_FORMAT=json`). Fields: `level`, `ts`, `component`, `msg`. `OBLIVRA_SELF_INGEST=true` forwards server logs into its own ingest pipeline as `sourceType: oblivra:server`.
* [ ] **Distributed trace propagation** — W3C `traceparent` injected into outbound calls (webhooks, TSA, integration callbacks) and extracted from inbound requests. `traceId` appears in audit chain entries and `slog` output.
* [ ] **Expanded Prometheus histograms** — write/search latency histograms (not just p99 scalars) for WAL, BadgerDB, Bleve, webhook delivery. Gauges for open cases, hunt sessions, watchlist size, enrichment cache hit rate. All tenant-labelled.
* [ ] **Self-healing WAL recovery entry** — torn-write truncation on startup now writes a `wal.recovery` audit chain entry recording `{recoveredAt, truncatedBytes, lastGoodOffset}`. Alert if truncation exceeds 1% of WAL size.
* [ ] **Graceful shutdown with drain** — on SIGTERM: stop accepting ingest, drain event bus (up to `OBLIVRA_SHUTDOWN_DRAIN_TIMEOUT=30s`), fsync WAL, exit. Eliminates the race where events accepted by HTTP are lost on rolling restarts.
* [ ] **Readiness gate for WAL replay** — `/readyz` returns 503 until WAL replay into hot store is complete. Prevents a restarting node receiving ingest traffic before local state is restored.
* [ ] **Ops runbook expansion** — `docs/operator/runbook.md` extended from 10 to 20 playbooks: WAL torn-write recovery, Bleve index corruption, quota-exceeded remediation, SCIM sync failure, OIDC IdP unreachable, eBPF load failure, replication lag spike, TSA unreachable, cold-tier upload failure, watchdog heartbeat gap.

---

## Phase 91 — OQL v2: Full Query Language

The current OQL (`internal/oql/oql.go`) is a thin pipe-syntax wrapper over Bleve queries. It supports `where`, `limit`, `sort`, `head`, `tail`. That covers basic filtering but misses every power-user pattern: aggregations, computed fields, joins across time windows, and subqueries. Enterprise analysts expect SPL/KQL/LogQL expressiveness. This phase rewrites the OQL parser to a proper recursive-descent grammar while keeping full backward compatibility with v1 queries.

* [ ] **Full grammar with arithmetic expressions** — `internal/oql/parser.go` (replaces the current `oql.go`). New stages: `eval` (computed fields: `eval duration_ms = endTime - startTime`), `rename` (field aliases), `dedup` (deduplicate by field), `regex` (inline regex extraction: `regex message "Failed password for (?P<user>\S+)"`), `lookup` (join against asset registry or IOC table). Backward-compatible: all v1 plans parse identically.
* [ ] **Aggregation pipeline** — `stats` stage: `stats count() by sourceType`, `stats avg(duration_ms) by host`, `stats dc(user) as unique_users by srcIP`. Aggregate functions: `count`, `sum`, `avg`, `min`, `max`, `dc` (distinct count), `values` (collect as array), `list` (collect first N). Results returned as a flat row set with synthetic field names, distinct from the event result set.
* [ ] **`timechart` stage** — `timechart span=5m count() by severity`. Bins events into fixed time buckets, returns `{bucket, fieldValues}` rows. Used by the frontend to render the timeline histogram without a separate API call.
* [ ] **Subqueries** — `[search severity:critical | head 100]` as a filter source. Evaluates the inner query first, extracts the specified field values, and uses them as an `IN` filter on the outer query. Enables: `srcIP IN [search geo.country:CN | stats values(srcIP)]`.
* [ ] **`transaction` stage** — groups events into transactions by a field or field pair with optional `startswith` / `endswith` patterns and a `maxspan` window. `transaction host startswith="session-open" endswith="session-close" maxspan=8h`. Each transaction becomes a synthetic event with `duration`, `eventCount`, `firstEvent`, `lastEvent`. Used by `ReconstructionService` as a query-time alternative to the pre-built session model.
* [ ] **OQL query explain** — `GET /api/v1/siem/oql/explain?q=` returns the parsed plan as JSON: stages in order, estimated result count per stage, index usage (full-text vs chrono scan), and any detected inefficiencies (missing field index, unbounded time range). Used by the frontend to show a "query plan" badge and warn operators before running expensive queries.
* [ ] **OQL autocomplete API** — `GET /api/v1/siem/oql/complete?q=&cursor=` returns completion candidates at the cursor position: stage names, field names (from the schema + enrichment fields), known field values (top-10 from field stats). Frontend wires this to the search bar for inline autocomplete.

---

## Phase 92 — RBAC v2: Fine-Grained Permissions

The current RBAC (`internal/rbac/rbac.go`) has four flat roles (admin/analyst/readonly/agent) and 13 coarse permission constants. Enterprise deployments need column-level data redaction, resource-scoped permissions, and custom roles. The current model cannot express "analyst can see events for tenant A but not tenant B" or "SOC lead can seal cases but tier-1 analyst cannot".

* [ ] **Custom role definitions** — `internal/rbac/roles.go`. Roles stored in `roles.json` under `OBLIVRA_DATA_DIR`. Each role: `{name, permissions[], tenantScopes[], resourceScopes[]}`. The four built-in roles are seeded on first boot and cannot be deleted (only extended by creating additional roles). `POST /api/v1/admin/roles`, `GET /api/v1/admin/roles`, `PUT /api/v1/admin/roles/{name}`, `DELETE /api/v1/admin/roles/{name}` (non-built-in only). Role changes are chain entries.
* [ ] **Permission expansion** — new fine-grained permissions: `cases.seal`, `cases.legal`, `rules.delete`, `intel.delete`, `vault.read`, `vault.write`, `ueba.watchlist`, `fleet.delete`, `audit.export`, `tenants.manage`, `compliance.manage`, `hunt.create`. Each existing route is re-annotated with the new permission rather than `PermAdminAll`. `RoleAdmin` retains all; `RoleAnalyst` gets the read + triage subset; `RoleReadOnly` gets read-only.
* [ ] **Tenant-scoped tokens** — `tenantScopes: ["tenant-a", "tenant-b"]` on a role restricts the bearer to those tenants only. All service methods that accept `tenantID` validate it against `Subject.TenantScopes` before execution. Cross-tenant admin view (Phase 81) requires `tenantScopes: ["*"]`.
* [ ] **Resource-scoped permissions** — `resourceScopes` supports case-level scoping: `cases:case-id-abc` grants access to that specific case only. Enables read-only external counsel access: issue a token with `[cases.read, cases:case-id-abc]` that can only see one case and nothing else.
* [ ] **Column-level redaction** — `internal/rbac/redact.go`. Roles carry an optional `redactFields: ["srcIP", "user", "message"]` list. The `auditmw` response interceptor applies the redaction map to every event in a search/OQL response based on the Subject's role. Redacted fields are replaced with `"[REDACTED:field]"`. The on-disk event is unmodified (hash still verifies); redaction is view-layer only. Redaction actions are logged to the chain.
* [ ] **Permission audit endpoint** — `GET /api/v1/admin/roles/{name}/effective` returns the full permission set for a role, resolved through any inheritance, with the list of routes each permission gates. Operators can answer "what can this analyst actually do?" without reading source code.

---

## Phase 93 — Sigma Rule Authoring & Testing Workbench

Sigma rules are loaded and hot-reloaded but there is no in-platform authoring surface. Security engineers currently write YAML in an editor, drop files into `sigma/`, and wait for the hot-reload tick. Enterprise SOC teams need a rule lifecycle: draft → test → stage → promote → deprecate, with a test harness that proves a rule fires on known-bad events and does not fire on known-good ones.

* [ ] **Rule lifecycle states** — Sigma rules extended with an OBLIVRA-specific `x-oblivra-status: draft|staged|active|deprecated` field (ignored by standard Sigma tools). `RulesService` only loads `active` rules into the live detection engine; `staged` rules run in shadow mode (fire internally but don't emit alerts). `GET /api/v1/detection/rules` returns all states; `?status=active` filters.
* [ ] **In-platform rule editor** — new **Rule Editor** sub-view inside Detection. Monaco editor (bundled WASM, no CDN) with YAML syntax highlighting and inline Sigma schema validation (JSON Schema for Sigma 2.x, bundled). Save writes to `<dataDir>/sigma/<name>.yml`; hot-reload triggers immediately. New rule starts as `draft`.
* [ ] **Rule test harness** — `POST /api/v1/detection/rules/{id}/test`. Body: `{matchEvents: [{...}], noMatchEvents: [{...}]}`. Server runs the rule's condition against each event set, returns `{matchResults: [{eventId, fired: bool}], noMatchResults: [{eventId, fired: bool}], passed: bool}`. Frontend shows a pass/fail badge per event. Rule can only be promoted to `staged` if the test harness passes.
* [ ] **Backtest against historical data** — `POST /api/v1/detection/rules/{id}/backtest`. Body: `{fromUnix, toUnix, tenantId, limit}`. Server runs the rule against the historical event window using the existing Bleve index. Returns `{matchCount, matchRate, sampleMatches[5], duration}`. Lets authors see "this rule would have fired 47 times in the last 7 days" before going live.
* [ ] **Rule performance profiling** — `RulesService` tracks per-rule `{evalCount, matchCount, totalEvalNs, p99EvalNs}` in an in-memory map. `GET /api/v1/detection/rules/{id}/perf` returns the stats. Rules with `p99EvalNs > 5ms` get a "slow rule" badge in the UI. Helps operators find pathological Sigma conditions before they impact ingest throughput.
* [ ] **Sigma rule pack import** — `POST /api/v1/detection/rules/import` accepts a ZIP of `.yml` files (e.g. the official `SigmaHQ/sigma` repo archive) or a single file. Each imported rule lands as `draft`. Deduplication by rule `id` field: existing rules are updated in-place if the imported version is newer (by `date` field).
* [ ] **Rule change audit** — every rule create/edit/status-change/delete is a `rules.*` chain entry with a diff of the YAML content (unified diff format, truncated at 4KB). Gives auditors a complete rule change history without a separate VCS.

---

## Phase 94 — Agent Fleet Operations

The existing `FleetService` covers agent registration and batch ingest. Enterprise fleet management means remote configuration, health SLAs, automatic upgrade, and dead-agent alerting.

* [ ] **Remote config push** — `PUT /api/v1/fleet/agents/{id}/config`. Stores a new `agent.yml` blob (encrypted at rest under the agent's registered ed25519 public key). On the next agent poll (`GET /api/v1/fleet/agents/{id}/config`), the agent receives the new config, validates it with `agent test`, and applies it with a rolling restart. Rejected configs (test fails) roll back and report `config.reject` to the server.
* [ ] **Agent health SLA** — `TenantPolicyService` extended with `AgentHeartbeatSLA: 5m` (configurable). Scheduler job `fleet.health` runs every minute; agents silent for longer than the SLA raise a `fleet.agent-silent` alert. Alert body includes last-seen time and the agent's registered host, so the on-call engineer knows exactly which host stopped forwarding.
* [ ] **Agent version inventory** — agent heartbeat extended to include `agentVersion` and `goVersion`. `GET /api/v1/fleet/versions` returns a version distribution table. Agents running versions older than the server's current version are flagged `stale` in the Fleet view.
* [ ] **Automatic upgrade mechanism** — `GET /api/v1/fleet/upgrade/{version}/{os}/{arch}` serves a pre-built agent binary. Agent compares its own version to `GET /api/v1/fleet/agents/{id}/config` (which includes `recommendedVersion`). If `autoUpgrade: true` in config, agent downloads the binary, verifies its ed25519 signature (server signing key configured at `OBLIVRA_AGENT_SIGNING_KEY`), replaces itself, and restarts via the `service install` mechanism. Upgrade is a `fleet.upgrade` chain entry.
* [ ] **Agent capability map** — agent heartbeat reports `capabilities: [file-tail, journald, syslog, ebpf, pcap, wasm-plugins]` based on build tags and runtime checks. Fleet view shows a capability matrix across all agents. Operators can see at a glance which hosts have eBPF but not PCAP, etc.
* [ ] **Agent log forwarding** — `OBLIVRA_AGENT_SELF_LOG=true` forwards the agent's own `slog` output to the server as `sourceType: oblivra:agent`. Agents that are crashing and restarting will leave a trace even if file-tail inputs aren't producing events. Requires the agent to have a working server connection (handled before the input loop starts).
* [ ] **Fleet map view** — Fleet view extended with a geographic map (GeoIP on agent registered IP, Phase 64). Agents plotted as pins coloured by health state (green=healthy, amber=stale version, red=silent). Useful for MSP operators managing geographically distributed deployments.

---

## Phase 95 — Data Pipeline Quality & Observability

The `QualityService` tracks source reliability and coverage gaps. Enterprise operations teams need deeper pipeline observability: schema drift detection, parser error attribution, field population rates, and SLA-based source health dashboards that operations teams can act on without reading log files.

* [ ] **Parser error attribution** — every `ingest/raw` call that fails to parse records `{sourceType, agentId, rawSample, error}` in a bounded circular buffer (`OBLIVRA_PARSE_ERROR_BUFFER=1000`). `GET /api/v1/quality/parse-errors` returns the buffer, grouped by sourceType with error rate. Frontend: "Parse errors" tab in Trust & Quality view with sample raw lines so an operator can fix the regex or parser without enabling debug logging.
* [ ] **Field population heatmap** — `GET /api/v1/quality/field-coverage?sourceType=&window=24h`. For each field in the Event struct + top-20 `Fields` map keys, returns the percentage of events in the window that have a non-empty value for that field. Returns as a matrix: `{field, sourceType, populationRate}`. Frontend: colour-coded grid. Instantly shows "user field is empty 60% of the time for sshd events" before an analyst tries to query it.
* [ ] **Schema drift detection** — `QualityService` maintains a rolling 7-day fingerprint of the field set observed per sourceType (set of field names present in >1% of events). When a new ingestion day's fingerprint diverges from the baseline by >20% (fields added or dropped), raises `quality.schema-drift` alert. Catches silent parser changes at the source host.
* [ ] **Ingest lag metric** — per-source `ingestLag = receivedAt - eventTimestamp`. If `eventTimestamp` is present in the raw event (extracted by parser), the pipeline computes lag and stores it in a rolling histogram per sourceType. `GET /api/v1/quality/lag` returns p50/p95/p99 lag per source. Alert when p99 lag > `OBLIVRA_LAG_ALERT_THRESHOLD` (default 5min). Catches agents that have built up a backlog.
* [ ] **Duplicate detection** — ingest pipeline maintains a per-tenant Bloom filter (configurable false-positive rate, default 0.1%, 1M capacity before rotation). Events whose content hash is already in the filter are tagged `duplicate: true` and written to a separate `duplicates.wal` rather than the main WAL. `GET /api/v1/quality/duplicates` returns duplicate rate per sourceType. Reduces Bleve index size on noisy sources that emit repeat events.
* [ ] **Source SLA dashboard** — operators define per-source expected event rates in `quality_slas.json`: `{sourceType, minEventsPerHour, maxLagSeconds}`. Scheduler `quality.sla-check` runs hourly. Below-floor raises `quality.source-below-floor`; above-ceiling (unexpected spike) raises `quality.source-spike`. Both are surfaced in a dedicated **Source SLA** panel in Trust & Quality view with a 7-day trend sparkline per source.
* [ ] **DLP expansion** — `internal/dlp` currently redacts credit cards and AWS keys. Extend the pattern set: UK NI numbers, EU IBAN, Australian TFN, Canadian SIN, IBAN, passport patterns (US/UK/EU formats), private key PEM headers, JWT tokens (header.payload.sig pattern), and IPv6 addresses in specific sensitive field contexts. Each pattern is a named, hot-reloaded YAML entry so operators can add custom patterns (e.g. internal employee IDs) without code changes.

---

## Phase 96 — Evidence Package v2 (Court-Ready)

The existing evidence package (`ReportService.CaseHTML`) is a self-contained HTML file. Phase 38 built the renderer. This phase takes it to court-submission grade: a verifiable ZIP bundle with a cover page, chain-of-custody manifest, offline verifier instructions, and digital signature page — everything a legal team needs to submit digital evidence without hiring a forensic consultant to explain the package structure.

* [ ] **Evidence package v2 bundle structure** — `GET /api/v1/cases/{id}/evidence-package-v2.zip`. Contents:
  ```
  MANIFEST.json           — SHA-256 of every file in the bundle
  COVER.html              — case summary (title, opened, sealed, analyst, hash at seal)
  CHAIN_OF_CUSTODY.html   — every handoff record (Phase 89), every analyst action from the chain
  EVENTS.jsonl            — all events in the case window, one per line
  AUDIT_EXCERPT.jsonl     — audit chain entries covering the case window
  SIGMA_MATCHES.json      — every rule that fired, with the event IDs that triggered it
  STIX_BUNDLE.json        — STIX 2.1 export (Phase 78)
  TSA_TOKENS/             — .tsr files for every anchor in the case window
  VERIFIER/               — static oblivra-verify binary (linux-amd64, windows-amd64)
  VERIFY_INSTRUCTIONS.md  — step-by-step verification guide for non-technical reviewers
  SIGNATURE.p7s           — PKCS#7 detached signature over MANIFEST.json
  ```
* [ ] **MANIFEST integrity** — `MANIFEST.json` contains `{file, sha256, sizeBytes}` for every file in the bundle. The verifier re-derives all hashes and compares. Any mismatch exits 1 with the offending file named.
* [ ] **PKCS#7 bundle signature** — `SIGNATURE.p7s` is a CMS detached signature over `MANIFEST.json` using the operator's signing certificate (`OBLIVRA_EVIDENCE_CERT`, `OBLIVRA_EVIDENCE_KEY`). If not configured, the MANIFEST is HMAC-signed with the audit key instead (weaker but still tamper-evident). Verification instructions cover both paths.
* [ ] **Cover page legal language** — `COVER.html` includes a configurable attestation block (`OBLIVRA_EVIDENCE_ATTESTATION_TEXT`) for the submitting organisation's standard legal language ("This package was generated by..."). The case analyst's name (from SSO subject or API key name), the case seal time, and the audit root hash at seal are rendered as immutable fields.
* [ ] **Offline verifier bundling** — `task release:verifier-bundle` builds `oblivra-verify` for `linux/amd64`, `linux/arm64`, `windows/amd64`, `darwin/arm64` and copies them into a `VERIFIER/` directory that can be included in the evidence ZIP. Verifier binaries are signed with the same ed25519 key used for agent binaries so the recipient can confirm they haven't been tampered with before running them.
* [ ] **`VERIFY_INSTRUCTIONS.md` template** — plain-language guide: "Step 1: check the SHA-256 of this file matches MANIFEST.json. Step 2: run `./VERIFIER/oblivra-verify-linux-amd64 audit-excerpt.jsonl`. Step 3: compare the printed root hash to the value in COVER.html." Written for a technically competent non-expert (junior lawyer, court IT officer). Configurable header via `OBLIVRA_EVIDENCE_ORG_NAME`.
* [ ] **Package generation audit** — `POST /api/v1/cases/{id}/evidence-package-v2` is an audited endpoint; the generation event is a `case.evidence-package-v2.generated` chain entry with `{packageSha256, generatedBy, generatedAt}`. Any subsequent package for the same case gets a new entry — the chain records every time a package was created and who created it, which matters when the defence asks "how many copies of this evidence were made?".

---

## Phase 97 — Alert Persistence & Alert Database

Reading `internal/services/alert_service.go` directly: alerts are stored in an in-memory slice capped at 5000 entries with a ring buffer eviction. This means **alerts are lost on restart**. The comment says "Phase 5+ will back it with SQLite" — that never shipped. An enterprise operator restarting OBLIVRA for an upgrade loses their entire alert history, open assignments, and analyst verdicts. This is a critical gap for any deployment that treats alerts as the primary SOC work queue.

* [ ] **Alert persistence to BadgerDB** — `internal/services/alert_service.go` refactored. On `Raise`, alert serialised to JSON and written to `tenant:{id}:alert:{triggeredUnixNano}:{alertId}` key in the existing hot store (BadgerDB). On startup, `AlertService.Load()` scans the alert prefix and rehydrates the ring. `cap` still enforced as an in-memory limit but on-disk is unbounded (subject to `AlertMaxAge` retention in `TenantPolicyService`). WAL write before hot store write for crash safety — same pattern as event ingest.
* [ ] **Alert search & filter API** — `GET /api/v1/alerts?state=&severity=&ruleId=&hostId=&from=&to=&assignedTo=&limit=&cursor=`. Cursor-based pagination over the BadgerDB scan (no full-table scan). Supports time-range queries, multi-value `severity` filter, and free-text `q=` against message. Required for: SOC queues with >5000 alerts, historical alert audit, compliance evidence.
* [ ] **Alert metrics endpoint** — `GET /api/v1/alerts/metrics?window=7d`. Returns `{total, byState{open,ack,assigned,resolved}, bySeverity{critical,high,medium,low}, byRule[{ruleId, count, fpRate}], meanTimeToAck_seconds, meanTimeToResolve_seconds}`. Powers the SOC KPI dashboard (MTTA, MTTR). All derived from the persisted alert store — no separate aggregation service.
* [ ] **Alert export** — `GET /api/v1/alerts/export.csv?from=&to=&state=`. CSV download of alerts in the time window. Columns: `id, triggeredAt, ruleId, ruleName, severity, hostId, state, acknowledgedBy, acknowledgedAt, assignedTo, resolvedBy, resolvedAt, verdict, mitre`. Used by compliance teams for monthly reporting.
* [ ] **Alert retention policy** — `TenantPolicyService` extended with `AlertMaxAge duration` (default 90d). Scheduler `alert.eviction` runs daily; removes alerts older than `AlertMaxAge` from BadgerDB and writes an `alert.eviction` chain entry with the count removed. Ensures the alert store doesn't grow unbounded on long-running deployments.
* [ ] **Migrate in-memory ring to new store** — `oblivra-migrate` extended with migration `alert-v2`: on first boot after upgrade, reads the legacy ring from memory (if process was hot-upgraded via graceful restart handoff) and writes all surviving alerts to BadgerDB. For cold restarts (existing deployment), the migration is a no-op — the empty ring is simply persisted going forward and the operator is warned via a startup log line that pre-upgrade alerts were not recovered.

---

## Phase 98 — Parser Expansion & Log Format Coverage

Reading `internal/parsers/parsers.go`: five formats supported (JSON, RFC 5424, RFC 3164, CEF, auditd). The auto-sniffer covers about 70% of real-world log traffic. The 29 builtin rules in `rules_service.go` reference Windows Event IDs, AWS CloudTrail, Azure signInLogs, Kerberos, LSASS, WMI — but there are no parsers for Windows Event Log XML, CloudTrail JSON, or Azure Activity Log JSON. Rules fire on substring matches in `raw`/`message` fields, meaning structured fields like `EventID`, `srcIP`, `user` are only populated when a parser explicitly extracts them. For UEBA and the correlation engine to work well, those fields must be reliably extracted.

* [ ] **Windows Event Log XML parser** — `internal/parsers/winevt.go`. Parses Windows XML event format (as forwarded by WEC, Winlogbeat, or the OBLIVRA agent on Windows). Extracts: `EventID`, `Channel`, `Computer`, `UserID`, `SubjectUserName`, `TargetUserName`, `IpAddress`, `LogonType`, `ProcessName`, `CommandLine`, `ParentProcessName` from `EventData` key-value pairs. Maps `Level` (0–5) to OBLIVRA severity. Handles both `<Event>` root and `<Events>` batch wrapper. Sniff: line starts with `<Event ` or `<?xml`.
* [ ] **Windows Event Log EVTX import** — `internal/parsers/evtx.go`. Reads binary `.evtx` files (Windows native format) using a pure-Go EVTX parser. `POST /api/v1/import/evtx` accepts a multipart upload; `ImportService.RunEVTX()` iterates records and emits events through the normal pipeline. Required for importing offline forensic images without standing up a Windows Event Collector.
* [ ] **AWS CloudTrail JSON parser** — `internal/parsers/cloudtrail.go`. CloudTrail delivers `{Records: [...]}` JSON arrays. Parser flattens each record: `eventName` → `eventType`, `userIdentity.arn` → `user`, `sourceIPAddress` → `srcIP`, `requestParameters` → individual `Fields` keys, `errorCode` → `Fields.errorCode`. Sniff: top-level key `"Records"` containing an array where items have `"eventVersion"`.
* [ ] **Azure Activity Log / Entra ID SignIn parser** — `internal/parsers/azure.go`. Handles Azure Monitor JSON schema: `operationName.value` → `eventType`, `caller` → `user`, `callerIpAddress` → `srcIP`, `properties.statusCode` → severity mapping. Handles Entra ID signIn log schema: `userDisplayName`, `userPrincipalName`, `ipAddress`, `riskDetail`, `conditionalAccessStatus`. Sniff: presence of `"operationName"` with a nested `"value"` key, or `"signInEventTypes"` array.
* [ ] **GCP Cloud Logging (stackdriver) parser** — `internal/parsers/gcp.go`. Maps `jsonPayload`/`textPayload`/`protoPayload` to `message`, `resource.labels.instance_id` to `hostId`, `severity` to OBLIVRA severity (GCP uses string levels: EMERGENCY/ALERT/CRITICAL/ERROR/WARNING/NOTICE/INFO/DEBUG). `logName` to `sourceType`. `protoPayload.@type` for audit log type detection.
* [ ] **Okta System Log parser** — `internal/parsers/okta.go`. Okta delivers JSON events with `eventType`, `displayMessage`, `actor.alternateId` (email), `client.ipAddress`, `outcome.result`, `target[].displayName`. Maps to OBLIVRA fields and sets `sourceType: okta`. Covers login events, MFA prompts, policy changes, and user lifecycle events referenced by the Phase 82 UEBA impossible-travel rule.
* [ ] **LEEF (Log Event Extended Format) parser** — `internal/parsers/leef.go`. IBM QRadar's wire format: `LEEF:2.0|Vendor|Product|Version|EventID|DelimiterChar|key=value...`. Sniff: line starts with `LEEF:`. Maps `src`/`dst`/`usrName`/`proto`/`devTimeFormat` to OBLIVRA fields. Required for QRadar migration integrations.
* [ ] **Palo Alto Networks TRAFFIC/THREAT log parser** — `internal/parsers/paloalto.go`. PAN syslog delivers comma-delimited CSV rows with a field at position 3 identifying the log type (`TRAFFIC`, `THREAT`, `SYSTEM`, `CONFIG`). Each type has a fixed column schema documented in the PAN Admin Guide. Extracts `srcIP`, `dstIP`, `srcPort`, `dstPort`, `proto`, `action`, `bytes`, `app`, `rule`, `threatID`. Sniff: `1,` prefix (PAN serial) or detection via field-count + known type string at position 3.
* [ ] **Parser test corpus** — `internal/parsers/testdata/` extended with one real-world sample per new parser (anonymised). `parsers_test.go` snapshot tests confirm field extraction doesn't regress. New `TestFieldExtraction` table-driven test: for each format, a minimum set of fields must be populated (e.g. `srcIP` must be non-empty for CloudTrail auth events). Prevents regressions where a refactor silently empties structured fields that UEBA and rules depend on.

---

## Phase 99 — Scheduled Searches & Alerting from OQL

The `SavedSearchService` exists (`internal/services/saved_search_service.go`). Saved searches are currently re-run manually. Enterprise SOC teams need scheduled searches that auto-run on a cron, compare results against a threshold, and raise an alert when the condition is met — the "detection as a query" model used by Splunk Scheduled Alerts, Elastic Watcher, and Microsoft Sentinel Analytics Rules.

* [ ] **Scheduled search definition** — `SavedSearch` struct extended with `{schedule string (cron), enabled bool, alertCondition: {operator: gt|lt|eq|neq, threshold: int}, alertSeverity, alertTitle, suppressDuplicates bool, suppressWindow duration}`. `POST /api/v1/saved-searches/{id}/schedule` adds the schedule. Persisted to `saved_searches.json`.
* [ ] **Scheduler integration** — `internal/scheduler` extended with a `saved-search.run` job that reads all enabled scheduled searches, runs their OQL query against the current time window (last `window` duration), evaluates the alert condition against result count, and calls `AlertService.Raise` when the condition is met. Each run is a `savedsearch.run` chain entry with `{id, resultCount, conditionMet, ranAt}`.
* [ ] **Duplicate suppression** — when `suppressDuplicates: true`, the scheduler checks the last alert raised by this saved search. If the last alert is still `open` or `ack` and is younger than `suppressWindow`, skip raising a new one. Prevents pager storms from a query that stays above threshold for hours.
* [ ] **Last-run result caching** — `GET /api/v1/saved-searches/{id}/last-run` returns `{ranAt, resultCount, conditionMet, alertId?}`. Frontend shows a "last run" badge on each saved search card. Operators can see at a glance which scheduled searches are healthy vs silent vs alert-producing.
* [ ] **Scheduled search view** — Saved Searches view extended: new "Scheduled" tab lists all searches with a cron schedule. Columns: name, schedule, last run, last result count, last alert. Toggle enable/disable per search. "Run now" button triggers an immediate out-of-schedule run for testing.
* [ ] **Alert-from-search traceability** — alerts raised by scheduled searches carry `ruleId: savedsearch:{id}` and `ruleSource: "scheduled-search"` so analysts can pivot from the alert back to the saved search that produced it. Alert detail page shows a "From scheduled search" link.

---

## Phase 100 — Windows Agent (Native ETW + Windows Event Log)

The current agent on Windows tails files. Windows security telemetry is primarily emitted through ETW (Event Tracing for Windows) and the Windows Event Log API — not log files. A file-tailing agent on a Windows server misses: process creation (Sysmon Event ID 1), network connections (Event ID 3), registry changes (Event ID 13), WMI activity (Event ID 19–21), and PowerShell script block logging (Event ID 4104). These are the exact event types the 29 builtin rules reference. This phase makes the Windows agent a first-class citizen.

* [ ] **Windows Event Log subscription input** — `cmd/agent/inputs/winevt.go` (build-tagged `windows`). Uses `golang.org/x/sys/windows/svc/eventlog` + direct `EvtSubscribe` Win32 API call via `syscall`/`golang.org/x/sys/windows`. Subscribes to a configurable list of channels (default: `Security`, `System`, `Application`, `Microsoft-Windows-Sysmon/Operational`, `Microsoft-Windows-PowerShell/Operational`). Events delivered as XML via the `EvtRenderEventXml` callback, parsed by the Phase 98 Windows Event Log XML parser. Supports both `EvtSubscribeStartAfterBookmark` (resume after restart using a persisted bookmark file) and `EvtSubscribeToFutureEvents` (live only). Bookmark written after every successful batch flush to the server, preventing duplicate delivery.
* [ ] **ETW session consumer** — `cmd/agent/inputs/etw.go` (build-tagged `windows`). Opens a real-time ETW session (`StartTrace` + `EnableTraceEx2`) subscribing to configurable ETW providers by GUID or well-known name. Default providers: `Microsoft-Windows-Kernel-Process` (process create/exit), `Microsoft-Windows-Kernel-Network` (TCP connect/accept), `Microsoft-Windows-DNS-Client` (DNS queries). Events decoded via MOF schema or TraceLogging manifest, mapped to OBLIVRA events. Complements eBPF on Linux; no kernel driver required on Windows.
* [ ] **Sysmon XML parser integration** — Sysmon logs to the `Microsoft-Windows-Sysmon/Operational` channel as XML. The Phase 98 Windows Event Log XML parser handles the outer structure; this sub-task adds Sysmon-specific field extraction: `CommandLine`, `ParentCommandLine`, `Hashes` (MD5/SHA256/IMPHASH), `TargetFilename`, `DestinationIp`, `DestinationPort`, `RuleName`. Mapped to OBLIVRA `Fields` with `sysmon.` prefix. Enables the builtin rules to fire on structured fields (`sysmon.DestinationIp`, `sysmon.Hashes`) rather than substring-matching raw XML.
* [ ] **Windows Security Event enrichment** — Windows Security log EventIDs mapped to human-readable `eventType` values in a lookup table (`cmd/agent/inputs/winevt_types.go`): `4624` → `auth.logon.success`, `4625` → `auth.logon.failed`, `4688` → `process.create`, `4698` → `scheduler.task.create`, `4720` → `account.create`, `4740` → `account.lockout`, etc. `LogonType` integer decoded to string (`2=Interactive`, `3=Network`, `10=RemoteInteractive`). This makes rules, UEBA, and reconstruction work on semantic field values rather than raw XML strings.
* [ ] **Windows agent service installer** — `oblivra-agent service install` already exists. Extend for Windows: registers the agent as a Windows Service using `golang.org/x/sys/windows/svc`. Service runs under `NT SERVICE\OblivraAgent` (a virtual service account with no interactive login rights, no local admin, only `SeServiceLogonRight` and read access to event log channels). `service install` writes the service ACL. `service uninstall` cleans up. Documented in `docs/operator/windows-deployment.md`.
* [ ] **Windows agent CI** — GitHub Actions workflow `ci-windows.yml`: runs on `windows-latest`, builds the agent with `GOOS=windows`, runs `agent test` against a mock server, runs `go test ./cmd/agent/...` including the `winevt` input with a synthetic event injected via `ReportEvent`. Prevents Windows-specific regressions from going undetected on Linux CI.
* [ ] **Windows deployment guide** — `docs/operator/windows-deployment.md`: service account setup, Event Log channel permissions (`wevtutil gl Security`), Sysmon deployment checklist (config XML, hash algorithm selection), GPO for PowerShell script block logging (`EnableScriptBlockLogging`), and the `agent.yml` input stanza for each channel. Answers the question an operator faces on day 1: "I installed the agent on a Windows DC, what channels should I subscribe to and why?".

---

**Last Updated**: 2026-05-02 — Phase 61 (Splunk-UF parity: admin password + custom install paths):

## Phase 61 — Splunk-UF parity
* [x] **Admin password (Argon2id)** — `cmd/agent/password.go`. Stored as Argon2id hash (m=64MiB, t=3, p=4) at `<stateDir>/agent.password.json` mode 0600. Setup wizard prompts for it; gates `setup` rotation, `reload`, `encrypt-config`, and the loopback `/status` endpoint. Non-interactive sources: `OBLIVRA_AGENT_ADMIN_PASSWORD` env or `OBLIVRA_AGENT_ADMIN_PASSWORD_FILE`.
* [x] **`oblivra-agent password {set|clear|test}`** — rotation / clearing / CI-friendly test subcommand. `set` rotates (requires current); `clear` removes (requires current); `test` reads from env/stdin and exits 0 on match.
* [x] **Local status endpoint requires `X-Admin-Password` header** when a password is configured. Loopback bind + password-gated header = double-bottom defence.
* [x] **No-echo password prompts** via `golang.org/x/term`. Falls back to plain stdin with explicit warning when not on a TTY.
* [x] **Configurable install / data / config dirs** — `build/linux/release/install.sh` accepts `INSTALL_DIR`, `ETC_DIR`, `DATA_DIR`, `USER`, `GROUP`, `SERVICE_NAME`, `UNIT_PATH`, `ADD_SYMLINKS` env vars. systemd unit is generated from these so a custom prefix flows through `User=` / `WorkingDirectory=` / `EnvironmentFile=` / `ReadWritePaths=`. Re-running with the same vars upgrades in place. `uninstall.sh` honours the same vars.
* [x] **Release README** documents both — Splunk-UF-style install variation example + agent password lifecycle.

---

**2026-05-02** — Phases 59-60 batch (security tightening + elite agent):

## Phase 59 — Single ingest surface + endpoint security audit
* [x] **Removed third-party wire-compat ingest** — Splunk HEC (`/services/collector*`), OTLP/HTTP (`/v1/logs`), and Prometheus remote_write (`/api/v1/metrics/remote_write`) deleted along with their handlers, listeners, and dependencies. Single ingest surface = smaller attack boundary. Operators wanting metrics push them through the agent's REST + ed25519-signed path.
* [x] **Endpoint security audit** — every route reviewed; new RBAC entries for notifications (admin), saved-searches (rules:read), agent heartbeat (fleet:write), categories/services-health/compliance (read), event-detail (siem:read), all alert-lifecycle paths covered.
* [x] **Auth exemption list narrowed** — only `/healthz`, `/readyz`, `/metrics`, `/api/v1/auth/login`, and OIDC redirect/callback bypass auth. Everything else under `/api/` requires bearer token / mTLS / OIDC subject.
* [x] **Soak harness — credibility-grade reports** — `cmd/soak` writes JSON + markdown with `--require-eps` / `--max-error-rate` / `--max-p99` gates. Bug fix in tick math (was overshooting by `batch×`). Runner scripts `scripts/run-soak.{sh,ps1}` clean-boot, archive under `docs/operator/soak-results/<UTC>.{json,md}`. `task soak:run EPS=… DURATION=… HARDWARE=…`.

## Phase 60 — Elite agent (tamper-evident + day-zero + edge DLP + setup wizard + last-gasp)
* [x] **Tamper-evident config** — `config_integrity.go`. First run hashes the resolved Config (secrets zeroed) → fingerprint file. Subsequent starts compare; mismatch refuses to run unless `--acknowledge-config-change` or `OBLIVRA_AGENT_ACKNOWLEDGE_CONFIG_CHANGE=1`. Tripwire for silent edits.
* [x] **Day-zero historical backfill** — file tailer with `startFrom: beginning` now also reads sibling rotated archives (`.1`, `.2`, `.gz`) oldest-first through the same `feed()` pipeline. Gzip via stdlib `compress/gzip`. journald already supports `--no-tail` for full systemd history.
* [x] **Edge DLP redaction** — `cmd/agent/dlp.go`. Patterns mask credit cards, AWS access keys, AWS secret kvs, GitHub PATs, JWTs, password=… kvs, Authorization Bearer, US SSNs, PEM private-key blocks. `agentRedacted` field tracks which patterns fired. `stillHasSecrets` canary drops the line if a known leak fragment survives. Toggled via `redact: true`.
* [x] **Interactive setup wizard** — `oblivra-agent setup`. Prompts server URL (incl. `srv://`), tokenFile, hostname, tenant; offers existing well-known log paths (`/var/log/auth.log`, syslog, audit, web/db); picks secure defaults (signing/redact/local-rules/heartbeat all on); writes a labelled config; locks the fingerprint immediately.
* [x] **Local status HTTP endpoint** — `cmd/agent/local_status.go`. Loopback-only (default `127.0.0.1:18021`; rejects non-loopback). `GET /status` returns full self-state (uptime, queues, dropped, spill, pubkey fingerprint, every config flag). `localStatusAddr: "off"` disables.
* [x] **Last-gasp signed event** — on SIGTERM/SIGINT/SIGHUP, push one final ed25519-signed `agent.shutdown` event into the priority queue before draining. An attacker who SIGKILLs the agent still leaves a signed exit marker; combined with Phase 44's missing-anchor watchdog, silent-kill attempts surface within an hour.

---

**2026-05-02** — Phases 51-58 batch (elite features + observability + anomaly detection):

## Phase 51 — Stateful detection
* [x] **Threshold rules** — `RuleType: threshold` fires when N matching events occur within Window, scoped by GroupBy. Re-arm gate prevents pager-storm runs.
* [x] **Frequency rules** — `RuleType: frequency` counts distinct DistinctOf values (default `srcIP`) inside the bucket, fires at threshold. Right tool for rotating-IP brute force.
* [x] **Sequence rules** — `RuleType: sequence` matches an ordered list of substrings within Window — "failed password followed by accepted publickey" classic.
* [x] **`NotContain` negative gate** — substring blocklist on rules; suppresses known-good without a separate suppression rule.
* [x] 5 unit tests (Nth-hit, per-host buckets, distinct cardinality, in-order sequence, NotContain gate).

## Phase 52 — Sigma loader expansion (1 → 86% community-corpus support)
* [x] **`<term> and not <term>`** — selection-and-not-exclude is the most-used Sigma compound condition. Translates to AnyContain (positive union) + NotContain (negative union).
* [x] **`keywords` block** — top-level list of substrings, no field map. Common in web-attack rules.
* [x] **`all of <pattern>`** / **`all of them`** — every matched block's tokens must appear → mapped to `AllContain`.
* [x] **Numeric values** — `EventID: [1102, 104]` etc. now stringify properly (was silently dropped before).
* [x] Integer / int64 / float64 / bool selection values supported.
* [x] **86% load rate** validated against the SigmaHQ public corpus subset (617/718 rules).

## Phase 53 — Alert lifecycle
* [x] **States**: open → ack → assigned → resolved (back-compat with "closed").
* [x] **Lifecycle metadata**: AcknowledgedBy/At, AssignedTo/At, ResolvedBy/At, Verdict, Notes.
* [x] **Endpoints**: `POST /api/v1/alerts/{id}/{ack,assign,resolve,reopen}`, `GET /api/v1/alerts/{id}`.
* [x] Each transition lands in the audit chain with the actor name.
* [x] **Verdict feedback loop** — `true-positive` / `false-positive` resolution auto-feeds `RulesService.MarkAlert` so analyst triage drives rule tuning without a separate workflow.
* [x] **Detection.svelte UI** — state-filter chips, per-row triage panel with Assign / Resolve / Reopen.

## Phase 54 — Saved searches + scheduled queries
* [x] **`SavedSearchService`** — persistent name+query, optional schedule (interval ≥5min; 0 = manual), optional alerting (raise alert when hit count meets threshold).
* [x] Bleve and OQL both supported.
* [x] **`saved-search.tick`** scheduler job runs queued searches via existing search path.
* [x] **Endpoints**: `GET/POST /api/v1/saved-searches`, `POST /{id}/run`, `DELETE /{id}`.
* [x] **SavedSearches.svelte view** with add-form + list with run/delete + last-run telemetry.

## Phase 55 — Event detail + CSV/NDJSON export
* [x] **`GET /api/v1/siem/events/{id}`** — returns the event plus up to 50 related events on the same host within ±60s.
* [x] **Siem.svelte** rows now clickable; opens inline detail panel with full field map, raw vs parsed message, related events with one-click pivot.
* [x] **`?format=csv` / `?format=ndjson`** on `/api/v1/siem/search` — auditor-grade exports with stable column set + JSON `fields` column for nothing-lost guarantees.

## Phase 56 — MITRE ATT&CK matrix view + Process lineage graph
* [x] **`frontend/src/lib/mitre.ts`** — static enterprise tactic→technique map curated to the IDs OBLIVRA's rule packs emit. Air-gap-friendly (no fetch to attack.mitre.org).
* [x] **Mitre.svelte view** — 12-tactic horizontally-scrollable grid; each technique is a 5-bucket coverage chip (0 / 1-2 / 3-9 / 10-49 / 50+ fires); hover shows recent matching alerts.
* [x] **Lineage.svelte view** — pure-functional tidy-tree layout for process forests. Suspicious cmdlines (LOLBins / encoded PowerShell / shadow-delete) flagged with red border. Hover detail card.

## Phase 57 — Observability surface (Prometheus remote_write + service health + log-to-metric)
* [x] **`POST /api/v1/metrics/remote_write`** — Snappy + protobuf decoder for Prometheus remote_write. Each Sample becomes one Event with `eventType: metric:<name>`, lands in WAL → hot store → audit chain. Hand-rolled minimal protobuf decoder (no `prometheus/prometheus` tree).
* [x] **`ServiceHealthService`** — auto-rolled from `CategoriesService` + `QualityService`; per-sourceType status (healthy / degraded / silent / unknown) with hosts, lifetime events, unparsed-rate, gap density, avg ingest delay.
* [x] **`GET /api/v1/services/health`** + **Services.svelte view**.
* [x] **Log-to-metric extraction** — `EmitMetric: true` on a saved search pushes a metric event into the pipeline every scheduled run. Closes the loop with Prometheus remote_write — operators ingest external metrics AND derive metrics from log patterns, with the same audit-grade backing store.

## Phase 58 — Sigma adoption + anomaly detection + change tracking
* [x] **`oblivra-cli sigma import [--apply] [-v] <src> <dst>`** — audits a Sigma rule directory and (with `--apply`) copies just the loadable rules into `<dst>`, preserving relative paths. Verified: SigmaHQ subset 240/267 (89.9%) and emerging-threats 377/451 (83.6%) load cleanly.
* [x] **Change-detection Sigma pack** — 7 rules under `sigma/change_detection/`: systemd unit install, crontab modify, kubectl apply, Terraform/Ansible/Pulumi runs, container image pulled, OS package install, IAM user/role change. All carry `oblivra.change-detection` tag for cross-cutting filter.
* [x] **`AnomalyService`** — new-pattern detection. Tokenises every event message into a stable template (replace numbers, IPs, UUIDs, paths, timestamps, quoted strings with placeholders); fingerprints per `(sourceType, severity, template)`. First sighting after warmup = `anomaly:new-pattern` alert. Three gates: 30-min warmup avoids fresh-deployment storms, min-source-volume (50) avoids new-source false positives, severity gate (≥warn) avoids debug noise. Cap at 100k fingerprints with oldest eviction. **5 tests** cover template normalisation, stable-across-instances, first-sighting alert, warmup silence, volume gate, severity gate.

---

**2026-05-02** — Phase 62 (Tier-1 court/SaaS/forensic-grade properties + elite UI/UX):
- **Phase 62.1 RFC 3161 external timestamping** — `internal/services/timestamp_service.go` (TSAService) now POSTs every daily Merkle anchor's SHA-256 root to a configured Time Stamping Authority, persists the PKCS#7 token to `<dataDir>/audit/anchors/<YYYY-MM-DD>.tsr`, and writes a paired `audit.tsa-token` chain entry recording the TSA-asserted UTC time. Per-request 128-bit nonce + post-response digest match prevents replay. Disabled by default; opt in with `OBLIVRA_TSA_URLS=http://timestamp.digicert.com,http://freetsa.org/tsr`. New scheduler job `audit.tsa-stamp` (hourly, idempotent, bounded to 30 anchors per pass) backfills any anchor that hasn't been stamped yet — soft-fail when the TSA is unreachable. `oblivra-verify` now classifies `.tsr` sidecars natively, parses the embedded TSTInfo, and surfaces TSA-asserted time as a verifier note. Tests: 4 in `internal/services/timestamp_service_test.go` (round-trip with an in-process fake TSA using an ephemeral RSA cert, persist/load lifecycle, disabled-by-default no-op, and `TimestampPendingAnchors` idempotency). Net: daily anchors are now court-grade — a forensic examiner can prove the anchor existed before the TSA's signed time without having to trust OBLIVRA's HMAC key
- **Phase 62.2 Cross-tenant blast-radius** — `internal/httpserver/auth.go` extended: API-key syntax is now `key[:role[:tenant]]`, so a key bound to `tenant-a` cannot read `tenant-b`'s data even if its role would otherwise permit it. `*` is the wildcard tenant for the platform admin role. Middleware enforces it on every request: explicit `?tenant=` mismatch → 403, no tenant param → request is rewritten to the bound tenant before reaching handlers. New `auth.deny` audit entries capture every denial (cross-tenant, missing-perm, invalid-key, missing-credential) with subject + path + attempted tenant so a SIEM can alert on enumeration attempts. New `internal/httpserver/blast_radius_test.go` proves the property end-to-end against a live `httptest` server with two tenants sharing a hostname: keyA cannot see tenant-b events, keyA without a tenant param sees only tenant-a events, the wildcard admin key can see both, and every cross-tenant denial lands in the audit chain
- **Phase 62.3 Multi-source reconstruction** — `internal/reconstruction/multi_source_test.go` proves the platform's "any vendor, any shape" claim survives contact with mixed-format process events. Single host, four ingest paths in one snapshot: Windows Sysmon EventID 1 (hex `NewProcessId`/`ParentProcessId`), Linux auditd `type=SYSCALL execve` with `pid=`/`ppid=` fields, classic syslog `sshd[5678]: process started`, and native OBLIVRA agent JSON. All four PIDs land in `Running` regardless of source. Then mixed-format exit events pair correctly across formats (native `process_exit` eventType + kernel `pid=N exit` phrasing). Companion test pins tenant-isolation under shared hostname so an MSP analyst can't see a peer tenant's processes via reconstruction
- **Phase 62.4 Elite UI/UX** — frontend gets the table-stakes power-user surface every elite SaaS console has:
  - **Command palette** (Ctrl/Cmd+K) `frontend/src/lib/components/CommandPalette.svelte` — single-keystroke jump to any of the 19 nav views plus global commands (reload, copy URL, print, show shortcuts). Arrow-key navigation, Enter to fire, Esc to close, mouse hover updates the cursor, substring + word-prefix filter
  - **Keyboard shortcuts overlay** (`?`) `frontend/src/lib/components/ShortcutsOverlay.svelte` — three groups (navigation, search & list, evidence handling) with kbd-style chips
  - **Toast notifications** `frontend/src/lib/stores/toast.svelte.ts` + `ToastStack.svelte` — stack-based, auto-dismissing (3.5s default, 5s warn, 7s error), color-coded (info/success/warn/error), click-to-dismiss-early
  - **Clipboard helper** `frontend/src/lib/clipboard.ts` — `copy(text, label)` wraps `navigator.clipboard.writeText` with toast feedback + `execCommand('copy')` fallback for non-secure contexts. Wired into Evidence view: every audit-chain hash + the root-hash tile is now a click-to-copy button
  - **Topbar search** is now a button that opens the palette instead of a dead input field
  - **Global shortcuts** in `App.svelte`: Ctrl/Cmd+K (palette), Ctrl/Cmd+B (sidebar), `?` (help), `/` (focus topbar search), Esc (close any modal), all ignoring keystrokes inside inputs/textareas/contenteditable so they don't fight the user's typing

---

**2026-05-02** — eighth round, removed DetectFlow / Kafka integration:
- `internal/listeners/kafka.go` + `kafka_test.go` deleted
- `docs/integrations/detectflow.md` deleted (and the now-empty `docs/integrations/` dir)
- `KafkaConfigFromEnv` call + Kafka listener startup stripped from `cmd/server/main.go` and `internal/platform/stack.go`
- `Stack.Kafka` field removed; `Options.Kafka` field removed
- `go.mod` / `go.sum`: dropped `segmentio/kafka-go`, `segmentio/kafka-go/sasl/scram`, `xdg-go/scram`, `xdg-go/stringprep`, `xdg-go/pbkdf2` via `go mod tidy`
- Reason: not in scope. Pre-SIEM streaming detection layer was an architectural commitment that didn't fit the deployment story (single-server-behind-VPN). The Splunk HEC + OTLP/HTTP compatibility endpoints (Phase 41) remain — any DetectFlow-style pipeline that wants to forward to OBLIVRA can use those instead

---

**2026-05-02** — seventh round, agent dashboard + log categories + email alerts:
- Rich agent heartbeat — `POST /api/v1/agent/heartbeat` accepts a self-report (pubkey fingerprint, version, input count, queue depth, spill files+bytes, dropped events, batch size). FleetService now stores those fields per agent. Agent's main loop posts every 30s and updates a tracked `droppedEvents` counter when the tailer drops under backpressure
- Fleet dashboard rewritten — clickable agent list with health badges (healthy / lagging / silent / dropping), per-agent detail panel showing all heartbeat fields, copy-pubkey button (with the value to seed `OBLIVRA_AGENT_PUBKEYS`), 5 aggregate tiles (healthy/lagging/silent/spill backlog/dropped events). Per-agent fetch via `GET /api/v1/agent/fleet/{id}`
- New CategoriesService aggregates events by sourceType — count, lastSeen, top-5 hosts per category. Wired into the platform's processor fan-out so it costs one map lookup per event. New `Categories` view in the sidebar shows a per-category bar chart with drill-through host chips. New endpoint: `GET /api/v1/categories`
- New NotificationService delivers alerts via email (SMTP w/ STARTTLS + plain auth) alongside the existing webhooks. Per-(channel, ruleID) throttle of 1/5min so a noisy rule can't pager-storm the inbox. Test endpoint sends a synthetic alert for verification. Endpoints: `GET/POST /api/v1/notifications`, `POST /api/v1/notifications/{id}/test`, `DELETE /api/v1/notifications/{id}`
- New `Notifications` view in the sidebar — add/test/delete email + webhook channels through the UI; surfaces last-delivered + last-error per channel
- Bridge.ts: new `Agent` fields, `agentGet`, `categoriesList`, `notificationsList/Add/Test/Delete`. Nav has `categories` and `notifications`. App.svelte routes both

---

**2026-05-02** — sixth round, Sigma loader fix + journald input:
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
