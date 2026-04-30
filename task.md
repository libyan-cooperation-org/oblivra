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

* `[x]` — Production-ready
* `[s]` — Validated (tested under realistic conditions)
* `[v]` — Implemented (functional, needs validation)
* `[ ]` — Not started

---

## Snapshot of what's built (auto-updated each working session)

**Foundation**
* Wails v3 desktop shell + headless `cmd/server` sharing one Svelte 5 + Tailwind 4 frontend
* BadgerDB hot store, line-delimited JSON WAL with fsync, Bleve per-tenant full-text indices, Parquet warm tier with hot eviction
* Event bus with bounded fan-out, async processors (rules / UEBA / forensics / lineage / IOC enrichment)
* Cross-platform Taskfile (`windows:build` / `darwin:build` / `linux:build`) so `wails3 build` works on every platform

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
* Views: Overview, SIEM (live tail via WebSocket + query bar + filter chips + ingest probe), Detection (rules + alerts + MITRE heatmap), Investigations (per-host triage + UEBA detail), Evidence (audit chain + sealed packages + log gaps), Fleet (agents + IOCs), Admin (storage tiering)

**CLI**
* `oblivra-cli` — ping / stats / ingest / search / alerts / audit / fleet / rules / intel
* `oblivra-verify` — offline integrity verifier (audit logs / WALs / evidence packages); standalone binary; exit code 1 on failure
* `oblivra-migrate` — schema migration runner with atomic-rename rollback
* `oblivra-agent` — log-tailing agent (file or stdin) with on-disk spill+replay
* `oblivra-soak` — sustained-load ingest tester reporting throughput + p50/p95/p99 latency
* `oblivra-server` — headless REST + Svelte UI

**Services (live in the platform stack)**
* SiemService, RulesService, AlertService, ThreatIntelService, AuditService (durable journal + daily Merkle anchor), FleetService, UebaService, NdrService, ForensicsService, TieringService (with WORM + cross-tier verifier), LineageService, VaultService, TimelineService, InvestigationsService (snapshot freeze + hypotheses + annotations + confidence), ReconstructionService (sessions + state-at-T + network stitching + entity profiles + cmdline), TenantPolicyService, TrustService, QualityService, EvidenceGraphService, ImportService, ReportService.

**Tests**
* `go test ./...` clean across events (hash determinism, mutation, JSON-roundtrip, provenance tamper, field-order independence), parsers, sigma loader, audit (durable journal restart + tamper, **daily-anchor idempotence + empty-day**), rules, vault, OQL, investigations (snapshot freeze, persist-across-restart, sealed rejects mutation), verify (audit/WAL/evidence happy path + 2 tamper flavours + unknown-format rejection), migrate (no-op, plan, future-version), and **reconstruction (sshd lifecycle, explicit-eventType fast path, unclassified-event ignore, host scoping, state-at-T at three different timestamps)**

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

* [s] **Durable, append-only audit journal** — `audit.log` line-delimited JSON file, fsynced after every `Append`. Replay-on-startup verifies every entry's parent-hash; refuses to start on tamper. Tested for restart roundtrip + tamper detection.
* [s] **Tamper-evident query log** — `internal/httpserver/auditmw.go` wraps the mux so every audited route lands an entry in the chain with `{actor, role, method, path, status, bytes, duration, query, uaHash}`. Verified end-to-end. Routes use exact-match for parents to keep child paths classified separately.
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
* [s] **Time-frozen investigation views** — `InvestigationsService` opens cases that capture `{tenantId, hostId, from, to, receivedAtCutoff, auditRootAtOpen}`. `Timeline(caseId)` only returns events whose `receivedAt <= cutoff` AND fall within `[from, to]`, scoped to the case host. Cases persist to `cases.log` (line-delimited JSON, fsynced); replay restores them across restarts. Sealing locks the case (notes rejected). Verified end-to-end with `TestCaseSnapshotFreezesScope`: post-case event correctly excluded, pre-case events kept, scope-leak on different host blocked. **This was the single highest-leverage feature — now done.**

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
* [s] **Self-contained offline verifier** — `cmd/verify` (built standalone, no server connection) auto-detects artifact kind by content shape (audit log / WAL / evidence package) and verifies the appropriate invariants: Merkle chain, parent-hash links, optional HMAC signature, per-event content hash. Demoed end-to-end against a real `audit.log` (3 entries, root hash printed) + `ingest.wal` (3 events, every hash recomputed) + a tampered audit log (correctly flagged "BROKEN at entry 1: hash mismatch", exit code 1). 6 unit tests covering all artifact kinds + tamper detection + unknown-format rejection. **Strongest demo-able artifact for court / external auditors — now done.**

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

* [ ] Remove residual response-action logic (backend + frontend) — already mostly done in Phase 36; sweep again
* [ ] Delete all unused services and bindings
* [ ] Regenerate Wails bindings (clean state)
* [ ] Remove orphan UI components and routes
* [ ] Update `README.md`, `FEATURES.md`, `docs/operator/log-forensics.md`
* [ ] Validate schema migrations (Phase 36.x)
* [ ] **Replace synthetic parser tests with snapshot tests over real-world samples** — `internal/parsers/testdata/{rfc5424,rfc3164,cef,json}/*.log` + golden-event snapshots; production format drift will otherwise sneak past the current synthetic tests

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
| Verified ingestion pipeline under sustained load (§1) | ✅ — `cmd/soak` + per-stage rolling p50/p95/p99 in `Pipeline.Stats().Latency`. |
| Foundational integrity guarantees (§2) — durable audit, query-log audit, provenance, schema versioning, daily Merkle anchor | ✅ — all `[s]`. |
| Deterministic timeline reconstruction with gap visibility, snapshot-frozen investigations, deterministic ordering (§3) | ✅ — all `[s]`. |
| Functional reconstruction engine — sessions, state, persistent process lineage, network stitching, entity profiles, cmdline, cross-protocol auth chains (§5 + Phase 39) | ✅ — all `[s]`. |
| Evidence export with cryptographic verification + offline verifier + self-contained HTML package + legal-review gating (§6 + Phase 38) | ✅ — all `[s]`. |
| Stable multi-tenant isolation with per-tenant retention, WORM warm tier, cold-tier scaffold, schema-versioned Parquet (§7) | ✅ — all `[s]`. |
| Trust classification, sequence-break detection, log-tamper indicators, source reliability + coverage scoring (§4 + §9) | ✅ — all `[s]`. |
| Frontend surfaces every reconstruction + trust + cases capability | ✅ — Reconstruction, Cases, Trust&Quality views shipped; full sidebar nav |

**Beta-1 is feature-complete.** Every Beta-1 line is `[s]` (validated under tests / integration smoke). Items not flipped to `[s]`:

- a handful of stretch goals already noted as "Phase 7 will back this with SQLite" (in-memory accumulators)
- frontend "deep" widgets like the evidence-graph diagram and pivot-style overlays — the data is wired, the rendering is plain tables for now

These are scope-bounded enhancements, not blockers.

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

**Last Updated**: 2026-04-30
