# Changelog

All notable changes to Oblivra Sovereign Terminal are documented here.

## [1.1.4] - 2026-04-25

### 🛡️ Supply chain — 5 reachable CVEs closed
`govulncheck` flagged 5 reachable vulnerabilities. All closed via patch/minor upgrades — no breaking changes:

- `github.com/go-git/go-git/v5` v5.16.4 → v5.17.1 (covers GO-2026-4910, GO-2026-4909, GO-2026-4473)
- `github.com/russellhaering/goxmldsig` v1.4.0 → v1.6.0 (GO-2026-4753)
- `github.com/golang/glog` v1.0.0 → v1.2.4 (GO-2025-3372)

`govulncheck ./...` now reports "No vulnerabilities found." Dependabot's count of 8 included unreachable advisories — those are tracked but lower priority.

### 🚦 Phase 22.1 — Node failure simulation closes
`cmd/chaos/main.go` Scenario 6 builds a 3-node Raft cluster over `hashicorp/raft`'s in-memory transport with a no-op FSM, kills the elected leader, and asserts a different node wins re-election within 5s. CGO-free so it runs in any build environment (the existing `internal/cluster/leader_failure_simulation_test.go` requires gcc to link the SQLite driver and is blocked from CGO_ENABLED=0 lanes).

Together with the existing tests, this closes Phase 22.1's "node failure simulation" item:
- Election under partition (existing `TestRaftSplitBrain`)
- Idempotent retry after leader failure (existing `TestLeaderFailureIdempotency`)
- Election convergence after leader kill (new chaos Scenario 6)

### 🔒 Phase 26.5 / 25.10 — Hardware-rooted FIDO2 quorum verification
Closes the gap that `quorum.go:111`'s comment flagged: "we assume the caller has already verified the FIDO2 auth." Now `QuorumManager.Approve` takes a challengeID + WebAuthn assertion outputs and drives `FIDO2Manager.CompleteAuthentication` to verify the ECDSA signature against the registered public key BEFORE counting the vote. Failed verification rejects with a WARN log naming user + request ID. Development-mode fallback (FIDO2Manager == nil) emits a clearly-marked WARN so operators can see when hardware-trust is bypassed in dev.

`internal/services/security_service.go:QuorumApprove` plumbs the new parameters through to the bound API.

### 🔕 Phase 26.9 — Alert suppression feedback loop
`MarkFalsePositive` now publishes a `suppression:suggested` event on the bus with the evidence so a UI listener can present a one-click "create suppression rule" prompt. New helper `SuppressionService.SuggestFromEvidence(evidence)` extracts a draft rule by finding the most consistent field/value across evidence rows (host_id, user, src_ip, event_type, rule_id). Rules are returned with `IsActive: false` — operators must explicitly enable.

In-memory `MatchCount(ruleID)` and `MatchCounts()` expose per-rule hit counts so operators can see which suppression rules are pulling weight (or which have gone stale). Counts reset on restart by design — durable counts need a schema column that's a Phase 26.9 follow-up.

### 🧙 Phase 22.5 — Setup Wizard MVP
`frontend-web/src/pages/SetupWizard.svelte` ships a 4-step first-run flow:

1. **Administrator account** — email + 12+ char passphrase with confirm
2. **Alert channel** — none / email / generic webhook / Slack incoming-webhook
3. **Detection pack** — essential (~25 high-confidence rules) / extended (recommended, all built-ins) / paranoid (built-ins + SigmaHQ community)
4. **Orientation tutorial** — quick lap of search, alerts, fleet, and the degraded banner

Wired into `App.svelte` at `/setup` (public route). POSTs to the existing `/api/v1/setup/initialize` route. Steps deferred from the original 6-step claim (TLS cert, first log source) are operator-infra concerns covered by `cmd/certgen` and `Onboarding.svelte` respectively.

### 📝 Documentation
- `task.md` — Phase 22.1 node failure simulation now `[x]`; Phase 25.10 multi-party enforcement upgraded `[v]→[x]`; Phase 26.5 cryptographic quorum upgraded `[v]→[x]`; Phase 26.9 alert suppression now has full feedback-loop description (still `[v]` pending maintenance-window schema); Phase 22.5 Setup Wizard now `[v]` with MVP scope explained.

---

## [1.1.3] - 2026-04-25

### 🚦 Phase 22.1 — Reliability Engineering (S2 close-out)

Four more S2 reliability items shipped, finishing the bulk of the open 22.1 backlog.

#### BadgerDB corruption recovery
`internal/storage/badger.go:NewHotStore` now ladders through three recovery levels on open failure:

1. Normal open
2. Truncate-mode open (drops a torn vlog tail — typical after a power-loss mid-write)
3. Read-only fallback with CRITICAL-level log so the operator can extract data via `HotStore.ExportSnapshot(dst)` and reinitialise from `HotStore.ImportSnapshot(src)`

Previously a torn last write would refuse the SIEM's startup. Now the service either heals itself or surfaces a recoverable read-only handle. The new `ExportSnapshot`/`ImportSnapshot` pair uses Badger's native protobuf backup stream so it round-trips cleanly across Badger versions.

#### Time Synchronization Enforcement
`internal/events/events.go` adds a `TimeConfidence` enum (`normal` / `late` / `skewed` / `unknown`) and a pure `ClassifyTime(timestamp, now)` function. `pipeline.processEvent` tags every event with `EventTimeConfidence` + signed `SkewSeconds` **before** WAL and index writes — so the tag is durable and queryable. Skewed events log a single info line per occurrence for NTP correlation.

Thresholds: ±60s = normal, 60s to 5min in the past = late, >60s in the future or >5min in the past = skewed, unparseable = unknown (skew=0).

#### Deterministic Replay (MVP)
New `cmd/replay` CLI exposes the deterministic-replay foundation:

- `replay --mode=capture --wal <path> --out manifest.ndjson` — walks every record in a WAL via the existing `storage.WAL.Replay` primitive and emits a per-record SHA-256 manifest (NDJSON, order-preserving).
- `replay --mode=verify --wal <path> --against manifest.ndjson` — re-walks the WAL and asserts every record matches the manifest by index, length, and SHA. Exact diff on drift.

This locks down **input determinism** (the WAL is byte-identical run-over-run). The full alert-equivalence layer (replay through the detection engine + diff alerts) is the follow-up; the MVP is enough to prove a WAL hasn't been tampered with between a baseline run and a regression run.

#### 50-tenant isolation test scaled to 1000 events/tenant
`tests/tenant_isolation_test.go` now runs 50 × 1000 = 50k events (up from 50 × 10) so the test actually exercises the index size sweet spot the task.md claim was made about. `-short` mode skips the larger run for fast CI loops.

### 🧹 Cleanup

- `internal/isolation/manager.go` — six `m.log.Info(fmt.Sprintf(...))` and `m.log.Error(fmt.Sprintf(...))` sites converted to direct format-arg passing (`m.log.Info("[Worker-%s] %s", wType, text)`). `go vet` is now clean on the package; one of the lingering Phase 28 audit warnings closed.
- 6 prior commits (audit verification + license gates + GDPR audit + agent reconnect + degraded banner) pushed to `origin/main`.

### 📝 Documentation
- `task.md` — Phase 22.1: BadgerDB corruption recovery now `[x]`, Time Synchronization Enforcement now `[x]`, Deterministic Replay marked `[v]` with MVP scope explained, 50-tenant test bumped to match the claim.

---

## [1.1.2] - 2026-04-25

### 🚦 Phase 22.1 — Reliability Engineering (S2)

#### Agent reconnect guarantee
End-to-end durability across server restarts with >1000 events in flight, no data loss and no duplication. Closed the long-standing race where `flushOnce`'s blanket WAL truncate could destroy events written between `ReadAll` and `Truncate`.

- **Per-event sequence numbers** — `Event.Seq` is monotonically assigned by the agent's WAL on Write. Persisted via `internal/agent/cursor.go` (atomic temp-file + rename) so a crash between cursor reserve and WAL encode at worst burns a sequence number, never reuses one.
- **Server ack watermark** — `/api/v1/agent/ingest` response now returns `acked_seq` (highest Seq durably accepted). Per-agent state stored on `AgentInfo.LastAckedSeq`. Replays with `Seq <= LastAckedSeq` are silently skipped, so a retry after a partial-batch failure cannot double-ingest.
- **`WAL.TruncateUpTo(ackedSeq)`** — rewrites the WAL keeping only events with Seq above the watermark. Atomic temp-file + rename. Preserves writes that arrived during the flush race; the deprecated `Truncate()` (which blasted the whole file) is kept for compatibility but no longer called by the transport.
- **Chaos scenario 5** — `cmd/chaos/main.go --scenario=reconnect` drives 1500 events through the cursor + WAL + simulated server-restart, asserts the agent emits exactly Seq 751..1500 on cycle 2 (no reissue of 1..750, no missed events). Passes in ~40 ms.

#### Graceful degradation banner
Pipeline already self-classified at >3× rated EPS or >95% buffer fill — operators just had no surface to see it.

- `LoadStatus.String()` exposes a stable wire format (`healthy` / `degraded` / `critical` / `unknown`).
- `GET /api/v1/health/load` — lightweight endpoint returning `{status, queue_fill_pct, events_per_second, dropped_events, collected_at}`. Designed for 10s-cadence polling without the cost of `/ingest/status`.
- `pipeline:load_status_changed` bus event published on every transition for WebSocket-based consumers.
- `DegradedBanner.svelte` (`frontend-web/src/components/`) — top-of-page amber/red banner with dismiss-once-per-state semantics, wired into `App.svelte`. Hides on `healthy` / `unknown` so it never flashes during cold boot.

### 🔒 Security Hardening

#### GDPR-grade tenant deletion audit trail (Phase 22.2)
Tenant cryptographic-wipe path now writes an immutable Merkle-chained audit record before/after the wipe so GDPR Art. 17 erasures leave regulator-grade evidence.

- `handleAdminTenantWipe` reads tenant name/tier pre-wipe, accepts optional `reason` in the request body, captures actor user_id/email/IP, and emits a `tenant.deleted` (or `tenant.delete_failed` on error) audit entry via `AuditRepository.Log`. The audit table's Merkle chain is rebuilt from disk on every boot via `InitIntegrity` — any tampering with the deletion record is detectable.
- `tenant:deleted` bus event published for live consumers (UI tenant-list refresh, etc.).

#### Plaintext settings logging fix (Phase 25.12 #1)
`internal/services/settings_service.go` no longer logs setting values at any level. The DEBUG line now emits only `setting key=%s value_bytes=%d` — length-only telemetry that's still useful for "did it change size?" diagnostics. The `isSensitiveKey()` allowlist was hardened with a substring fallback (`password`, `passphrase`, `secret`, `token`, `webhook`, `credential`, `private_key`, `auth_key`, `client_secret`) so newly-added sensitive keys fail closed (encrypted at rest + redacted) without requiring an explicit allowlist edit.

### 📝 Documentation
- `task.md` — Phase 22.1 chaos test harness, automated soak regression, **agent reconnect guarantee**, and **graceful degradation under overload** all now `[x]` with file:line evidence and chaos-scenario validation.
- `task.md` — Phase 22.2 tenant deletion audit trail upgraded from `[v]` to `[x]`.
- `task.md` — Phase 25.12 #1 plaintext settings logging now `[x]`.

---

## [1.1.1] - 2026-04-25

### 🔍 Verification Audit Pass — Phase 28
A re-audit of every `[x]` item in Phases 22, 23, 25, and 26 against the actual code paths. Several claims were overstated or shipped only as backend stubs without UI; corrections applied in `task.md` Phase 28.

**Confirmed already-shipped (status was open):**
- 22.1 Chaos test harness — `cmd/chaos/main.go` (520 LOC) covers WAL CRC replay, BadgerDB VLog corruption + truncate-mode reopen, OOM/burst load-shed probe, clock skew ±5 min
- 22.1 Automated soak regression — `.github/workflows/soak.yml` runs 30 min × 5,000 EPS on every release tag; fails on >10% EPS drop, >0.1% event loss, or min-window <50% of target

**Reset to open (claim was overstated):**
- 23.5 Clipboard OSC 52 — backend missing entirely
- 23.6 AI Autocomplete UI — `CommandHistoryService.GetSuggestions` exists; floating suggestion box does not
- 26.10 Hot/Warm/Cold Tiering — contradicted open `[ ]` in Phase 22.3; only Hot (Badger) + Parquet archive exist

**Downgraded to partial:**
- 22.2 50-tenant isolation test runs 10 events/tenant (not 1000); structural isolation valid but throughput claim overstated
- 22.2 Tenant deletion — flips status + wipes salt but does not write an immutable deletion record (GDPR right-to-erasure evidence)
- 23.4 OperatorBanner.svelte — backend service exists; component file does not
- 26.4 System-wide backpressure — worker pool + bus rate limit + NATS priorities exist; no explicit circuit breaker pattern
- 26.5 / 26.9 — voting structure / `MarkFalsePositive` exist; FIDO2 hardware signature verification on quorum approvals + rule-based suppression with feedback loop do not

### 🔒 Security Hardening
- **License gates closed** on previously-ungated premium endpoints, including the destructive `POST /api/v1/ransomware/isolate` (network isolation action with no licensing check at all). Coverage extended to `playbooks/run`, `playbooks/metrics`, `ueba/stats`, `ndr/protocols`, `ransomware/{events,stats}`.
- **Honeypot credential leak** — `RegisterTrigger` was logging `decoy.Value` (the plaintext honeypot username) at WARN level. Now logs only `id` + `type` so audit log readers cannot exfiltrate trap credentials.
- **Tenant query string concat removed** — `internal/api/rest.go` `handleSearch` and `handleAlertsList` were prepending `TenantID:%s AND (%s)` to user queries. Reclassified after re-audit: this was *dead code, not a leak vector* (storage layer at `siem_badger.go:175-185` already dispatches via `MustTenantFromContext` to a per-tenant Bleve index; auth middleware plumbs tenant from authenticated session). Removed the redundant predicate so future auditors don't keep flagging it.

### 🛡️ CI Security Tooling
- **`gosec`** — static analysis for InsecureSkipVerify, math/rand-as-security-source, fmt.Sprintf-into-SQL. SARIF uploads to GitHub Security tab.
- **`gitleaks`** — full-history secret scanning. Would have caught the `sync.oblivrashell.dev` dark-site URL pre-merge.
- **`govulncheck`** — CVE reachability analysis on every PR; fails on advisories with reachable vulnerable symbols.

### 🐛 Build Stability
- Fixed three pre-existing `parseTime()` 2-return-value misuse sites (`internal/security/fido2.go:95,167`, `siem.go:316`, `honeypot_service_test.go:38`) that were blocking `go vet` on `internal/security`. Unparseable challenge timestamps now fail closed (treated as expired); SIEM forwarder falls back to epoch on malformed timestamps so the event still ingests.

### 📝 Documentation
- `task.md` — Phase 28 verification audit summary appended; current sprint marker advanced from S1 to S2; 50+ items reclassified with file:line evidence and verdict (VERIFIED / PARTIAL / FAILED).

---

## [1.1.0] - 2026-03-16

### 🔒 Security Hardening (31 findings resolved)
- Removed hardcoded `S@nad2026!` credential from `ImportGPayStaging()`
- `Password` field tagged `json:"-"` across all host models — never serialised to frontend
- `crypto/rand` used exclusively for vault password generation with rejection-sampling via `math/big` (eliminates modulo bias)
- `NuclearDestruction` uses `crypto/rand` first pass before overwrite
- `defer ZeroSlice` added to every credential decrypt IIFE — secrets cleared on function return
- `SubscribeWithID` / `Unsubscribe` added to event bus — all subscriptions now cleanly removable
- WebSocket `CheckOrigin` allowlist enforced, TLS 1.3 minimum on REST API
- Plugin sandbox `cancelCtx` stored and called on `Stop()` — no goroutine leak
- Multi-exec job pruner caps at 100 jobs; fails hard instead of plaintext fallback
- Wails CSP tightened in `wails.json`
- Vault unlock clears passphrase field in UI immediately after call

### 🔍 Sigma Detection Engine
- Full Sigma transpiler: 35+ field modifiers, keyword lists, MITRE tag extraction
- 15+ logsource → EventType mappings (Sysmon, AWS CloudTrail, Linux audit, Windows Security, etc.)
- Sigma community rules load from `sigma/` directory at startup
- `FuzzSigmaTranspile` fuzz test added to CI

### 📡 OpenTelemetry + Observability
- `InitTracing()` wired into `ObservabilityService` — non-fatal, recovers from any OTel panic
- `RegisterDetectionMetrics()` pre-registers all Prometheus counters at startup
- `docker-compose.yml` extended with Prometheus (9090), Grafana Tempo (3200), Grafana (3000)
- `ops/prometheus.yml`, `ops/tempo.yml` — full scrape and OTLP configs
- `ops/grafana/provisioning/` — auto-provisions datasources and a pre-built Oblivra dashboard

### 🖥 Terminal
- LOCAL and SSH sessions now fully isolated — rendered simultaneously, switching is instant
- LOCAL tabs shown with green badge, SSH tabs with orange badge
- Active tab has coloured top border matching session type
- xterm.js re-initialisation on tab switch eliminated (`visibility` instead of `display:none`)
- `fitAddon.fit()` fires on `active` prop change and after layout settles
- Auto-opens a local shell on first navigation to `/terminal`
- Empty state shows "New Local Shell" button + SSH sidebar hint

### 🎨 UI/UX — Splunk Enterprise Design System
- Complete colour palette overhaul: `#1a1c20` / `#212327` / `#2b2d31` surfaces, `#0099e0` blue, `#f58b00` orange CTA, `#5cc05c` green, `#e04040` red
- All glassmorphism (`backdrop-filter: blur`) removed from toasts, modals, command palette
- All `transform: translateY(-2px)` hover lifts removed from cards
- Left navigation expanded from 64px icon-only to 200px text-based Splunk-style rail
- Top bar: `#0d0e10` with orange brand block
- Vault gate: flat enterprise login — orange CTA button, no animations
- Every CSS file audited — undefined `--tactical-*`, `--bg-*`, `--splunk-*` variables resolved
- `variables.css` exports every alias including `--glass-bg-subtle`, `--primary-color`, `--error-color`, `--success-color`, `--warning-color`, `--bg-danger-subtle`, `--bg-success-subtle`
- Files fully rewritten: `siem.css`, `compliance.css`, `vault.css`, `incident.css`, `settings.css`, `purple-team.css`, `ops_center.css`, `executive.css`, `sidebar.css`, `dashboard.css`, `heatmap.css`, `modal.css`, `toast.css`, `command-palette.css`
- ECharts theme updated to Splunk palette throughout Dashboard

### 🔧 Bug Fixes
- `synthetic-service` nil pointer panic on startup fixed — `NewSyntheticManager` now correctly passed
- `vault.Unlock` normalises empty `[]byte{}` hardware key to `nil` — eliminates spurious "incorrect password" on first attempt
- `VaultService.SetContext()` propagates Wails runtime context after `Startup()` — fixes `EventsEmit: invalid context` warning
- Import order in `vault_service.go` corrected (`crypto/rand` before `math/big`)
- OTel `go.mod` entries cleaned — removed non-existent modules (`otel/codes`, `otel/sdk/trace`, `otel/semconv/v1.26.0` as separate modules)
- `otel.go` rewritten to use only API packages — no SDK required in default build

### 📦 Supply Chain
- CI: multi-OS test matrix, fuzz runs, architecture boundary tests, SBOM + Grype on every PR
- Release: cross-platform builds, syft SBOM (SPDX + CycloneDX), cosign keyless signing, SLSA attestation
- `SHA256SUMS.txt` covers all binaries and SBOMs
- Changelog extraction wired into release body

### 🗂 Models
- `monitoring.DiagnosticsSnapshot` exported from `models.ts`
- `services.AIResponse` and `services.Message` exported from `models.ts`
- AI Assistant page wired end-to-end (route `/ai-assistant`, live Ollama status badge, Chat / Explain Error / Generate Command modes)
- MITRE Heatmap page wired end-to-end (route `/mitre-heatmap`, tactic coverage vs. gap visualisation)

---

## [1.0.0] - 2026-03-10

### 🚀 Major Strategic Release
*First production-grade sovereign release containing all Phase 1-10 architectural requirements.*

### Core Defensive Capabilities
- **Cryptographic Vault**: AES-256-GCM hardware-bound vault with Argon2id KDF and OS memory zeroization
- **Embedded SIEM**: Go-native pipeline using BadgerDB and Bleve, capable of 5,000+ EPS with local Lucene search
- **eBPF Agent Framework**: Cross-platform telemetry agents with Linux eBPF hooks for Zero-Trust process monitoring

### Frontend
- **SOC Workspace**: Multi-monitor, draggable pop-out window engine for forensic dashboards
- **SSH Client**: Go-native connection manager — multi-exec broadcasting, SOCKS5 tunnels, SFTP explorer
- **Sovereign UI**: High-contrast tactical aesthetic for low-light SOC environments

### Enterprise Scale
- **Raft Clustering**: Multi-node HA consensus engine for database replication
- **RBAC**: Granular authorization controls with FIDO2 YubiKey identity verification
- **SIEM Threat Engine**: Offline IOC loading via STIX/TAXII, multi-hop Security Graph Query engine

### Forensics & Hardening
- **Runtime Attestation**: Binary hashing at `/debug/attestation`, Merkle Tree evidence ledgers, temporal drift monitors
- **Disaster Response**: Emergency kill-switch and nuclear-wipe functionality
- **Performance**: `OutputBatcher` IPC bridge flood protection, Zstandard payload compression, DB contention fixes

---

*Full architectural mapping: `docs/FEATURE_MATRIX.md`*
