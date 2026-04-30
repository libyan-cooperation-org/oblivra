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

# 🔥 Beta-1 Critical Path (Must Ship)

## 1. Ingestion Integrity

* [ ] Sustained-load soak test (documented + reproducible) — `cmd/soak`, fires N events/sec for M hours, reports loss/latency/percentiles
* [ ] End-to-end ingestion latency tracking — per-event `receivedAt → walAt → hotAt → indexedAt`, p50/p95/p99 in `/api/v1/siem/stats`
* [v] Ingestion gap detection (agent offline, pipeline drops) — `ForensicsService.Observe` flags >5min host silence; visible at `/api/v1/forensics/gaps` and on Evidence view. Still needs /metrics gauge + UI prominence.
* [ ] WAL integrity verification tooling — `cmd/wal-verify` replays the WAL, confirms every line parses, reports first corruption offset
* [ ] Cross-tier write consistency (Hot → Warm) — sample-hash N events per migration, verify Parquet decode round-trips back

---

## 2. Foundational Integrity (new — required for everything below)

These are the bedrock guarantees the rest of the platform leans on. They land
*before* reconstruction features so we never have to retrofit integrity onto
data that was already mutable.

* [ ] **Durable, append-only audit journal** — today the Merkle chain lives in RAM; if the process dies the chain is gone. Persist to `audit.log` with periodic on-disk root checkpoints so verify-across-restarts is possible
* [ ] **Tamper-evident query log** — every search, every export, every CLI call lands in the audit chain (`actor`, `query`, `resultHash`). This is what makes "cherry-picked evidence" detectable
* [ ] **Per-event provenance block** — structured `{peer, agentId, parser, tlsFingerprint, ingestPath}` hashed into the event ID; mutation breaks identity
* [ ] **Schema versioning** — every WAL line + Parquet file stamped with `schemaVersion`; one-shot migration tool committed to `cmd/migrate`
* [ ] **Time-anchored daily Merkle root** — at end of each day, hash all evidence + audit entries into a daily root committed to next day's root. Optional public-ledger publish path for post-incident proof

---

## 3. Timeline Reconstruction Engine

* [ ] Unified multi-source timeline (host / network / identity)
* [ ] Deterministic event ordering (clock drift handling)
* [ ] Timeline layering (events, detections, gaps, annotations)
* [ ] Explicit gap markers (ingestion / telemetry absence)
* [ ] Timeline filtering + pivoting engine (collapses into the interactive view)
* [ ] Entity-centric timeline views (Host / User / IP)
* [ ] **Time-frozen investigation views** — opening an investigation at T snapshots the data; subsequent queries go through the snapshot lens. The "live" mode is monitoring, not reconstruction. **This is the single highest-leverage feature for court admissibility.**

---

## 4. Event Trust & Integrity Model

* [ ] Event trust classification:

  * Verified (cryptographic — agent-signed, Merkle-anchored)
  * Consistent (multi-source match)
  * Suspicious (conflict detected)
  * Untrusted / missing context
* [ ] Cross-source validation engine — same login event seen by sshd + auditd + agent → "verified"; only one source → "untrusted"
* [ ] Timestamp anomaly detection — events from the future, far past, or non-monotonic per-source sequence
* [ ] Sequence break detection — for numbered sources (Windows EventID, syslog seq), detect missing IDs
* [ ] Log silence pattern detection — already partial in `ForensicsService.Observe`; broaden to detect periodic silence (regular log gaps that may indicate stop/start tampering)

---

## 5. Reconstruction Engine

* [ ] Session reconstruction (auth flows, user sessions) — group sshd / RDP / kerberos events into login → activity → logout sequences
* [ ] Process lineage reconstruction (from logs only) — `LineageService` exists; needs persistence + cross-host stitching
* [ ] Network activity stitching (flows, DNS, connections) — join NetFlow + DNS lookups + connection logs by 5-tuple within a time window
* [ ] State reconstruction at time T — "what was running on host X at 14:32?" — replay process_creation/exit events up to T
* [ ] Event replay engine (step-by-step timeline playback) — frontend feature on top of the timeline view
* [ ] **Backfill / import from external sources** — `POST /api/v1/import` + `oblivra-cli import file.jsonl|.parquet|.csv`. Forensic work routinely starts with "here's a 10GB Splunk export" — this is essential, not optional
* [ ] **Static health summary on import** — time range covered, host count, format breakdown, parse-failure ratio, suspected gaps. Shown to the analyst *before* they start querying

---

## 6. Evidence System (Core Differentiator)

* [ ] Basic evidence package export (events + timeline + Merkle proof)
* [ ] Evidence graph model (event relationships)
* [ ] Chain-of-custody tracking (access + export logs) — flows from §2 (tamper-evident query log)
* [ ] Immutable export hashing (query + result set hash committed to audit chain)
* [ ] **Self-contained offline verifier** — single static binary (no Go runtime needed on target box) that ingests an evidence package and verifies Merkle proofs + HMAC signatures + (when present) public-ledger anchoring. **Strongest demo-able artifact for court / external auditors.**

---

## 7. Storage Integrity & Tiering

* [ ] Hot/Warm/Cold migration validation — `tiering.Run` already does write→fsync→evict; needs a periodic verifier that re-reads parquet and matches against deleted-hot hashes
* [ ] Cross-tier integrity verification — see also §1 / §6
* [ ] WORM mode (immutability enforcement) — Linux `chattr +i`, Windows ReFS integrity stream / NTFS read-only attribute on closed Parquet files
* [ ] S3-compatible cold storage support — build-tagged so air-gapped deployments aren't forced to link an SDK
* [ ] **Per-tenant retention enforcement** — currently `MaxAge` is process-global; move to a `tenant_policies` SQLite table; warm migrator obeys it
* [ ] Schema-versioned tier formats (carries from §2)

---

## 8. Investigator Workflow (Product Layer)

* [ ] "Start Investigation" flow:

  * select entity (host / user / IP)
  * auto-build timeline + freeze snapshot (§3)
  * record case open in audit chain (§2)
* [ ] Pivot engine (event → entity → ±15min timeline → related entities)
* [ ] Hypothesis tracking (attach + validate evidence) — SQLite `cases` + `case_hypotheses` tables
* [ ] Annotation system (per-event investigator notes; notes are themselves audited)
* [ ] Forensic confidence scoring (case completeness) — heuristic over (events seen, alerts fired, gaps in window, sources corroborating)

---

## 9. Log Quality Intelligence

* [ ] Source reliability scoring — per source (peer / agent / format): valid-parse ratio, gap density, late-arriving fraction
* [ ] Coverage visibility — per tenant: host inventory observed, silence ratios, format breakdown
* [ ] Noisy / incomplete source detection
* [ ] Ingestion delay analytics — uses §1 latency tracking
* [ ] **DLP / search-time field redaction** — credit cards, secrets, tokens masked in displayed events while staying intact (and signed) at rest

---

# ⚖️ Phase 38 — Court Admissibility Layer

## Evidence Formalization

* [ ] Full forensic evidence package (PDF/HTML + signatures + verification instructions)
* [ ] Verification instructions (human-readable + CLI invocation)
* [ ] Evidence narrative builder (deterministic, no AI; templated from timeline + annotations)
* [ ] Legal review gating workflow

## Integrity Enforcement

* [ ] WORM enforcement across storage tiers (carries from §7)
* [ ] Evidence vault UI improvements
* [ ] Expanded chain-of-custody visualization

---

# 🧠 Phase 39 — Advanced Reconstruction

* [ ] Authentication / session reconstruction (deep correlation across sshd / RDP / kerberos / web)
* [ ] Command-line reconstruction from logs
* [ ] Entity forensic profiles (Host / User / IP)
* [ ] Tampering indicators (log-level only — no host file-system probing)
* [ ] Expert witness export package

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

# 🚀 Definition of Beta-1 Done

* Verified ingestion pipeline under sustained load (§1)
* Foundational integrity guarantees in place — durable audit journal, query-log audit, provenance, schema versioning, daily Merkle anchor (§2)
* Deterministic timeline reconstruction with gap visibility *and snapshot-frozen investigations* (§3)
* Functional reconstruction engine — sessions, state, lineage (§5) with import/backfill (§5)
* Evidence export with cryptographic verification + offline verifier binary (§6)
* Stable multi-tenant isolation with per-tenant retention (§7)

---

# 🥇 Single highest-leverage next item

Pick from §3 + §2: **time-frozen investigation views + tamper-evident query
log**. Combined, they make OBLIVRA the only tool an analyst can take into a
courtroom and say:

> "Here is exactly what I looked at, here is the order I looked at it in,
> and here is the cryptographic proof none of it changed underneath me."

That sentence is the product.

---

**Last Updated**: 2026-04-30
