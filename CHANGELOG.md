# Changelog

All notable changes to OBLIVRA. The numbering follows the internal "round" sequence — the round that delivered each batch is preserved so the audit trail is reconstructible.

## Unreleased

### Hardening (round 16)

- Webhook delivery test (HMAC roundtrip + severity filter + concurrent fan-out)
- OIDC PKCE state-nonce test against an httptest mock IdP — confirms code-verifier roundtrips and replay attempts are rejected
- S3 SigV4 adapter test against an httptest mock — verifies header markers, list/put/get/delete roundtrip
- Tiering migrator: nanosecond timestamp filenames so back-to-back migrations within the same second can't collide on the WORM-locked previous file *(real bug caught by the new stress test)*
- Trust engine: O(n²) fingerprint-corroboration loop replaced with a cap-at-5 strategy *(real bug caught by the new stress test — was timing out at 600s on a 2000-event run; now finishes in 10ms)*
- Containerisation: `Dockerfile` (multi-stage distroless build), `docker-compose.yml` (with Caddy TLS termination), `Caddyfile`
- GitHub Actions CI workflow at `.github/workflows/ci.yml` — runs `gofmt`, `go vet`, `go test -race`, frontend build, end-to-end smoke harness, and ghcr.io image push on `main`

### Hardening (round 15)

- Three new frontend views: **Vault** (init/unlock/lock + secret CRUD), **Evidence Graph** (SVG cross-reference visualisation), **Webhooks** (register + delivery log)
- Sidebar nav reorganised into 13 views across 3 groups

### Hardening (round 14)

- `TestCrossTenantBlastRadius` — proves tenant-A search cannot return tenant-B events even when both share a hostId
- `TestCaseScopeRespectsTenantBoundary` — case scoped to tenant-A excludes tenant-B's data
- `TestPlatformIntegration` — full in-process composition (ingest → rules → alerts → audit → trust → tamper → case open → frozen timeline → HTML report). **Caught a real race in `internal/scheduler`** — `close(s.done)` panicked when `Stop()` ran first; fixed by capturing `done` locally before goroutine start
- Vault `TestConcurrentReadWrite` — 8 writers + 8 readers × 25 ops. **Caught a real bug** — concurrent `os.Rename` on `.tmp` files races on Windows. Fixed with a `saveMu` that serialises the snapshot+write+rename critical section
- `oblivra-soak` baseline captured: 100% success rate, 1013 EPS sustained, p50=643ms, p99=3.1s. Archived as `docs/operator/soak-results-baseline.md`
- `internal/storage/cold/s3.go` — build-tagged S3 adapter (HTTP + AWS SigV4 only, no SDK)
- `internal/httpserver/oidc.go` — OIDC Authorization Code flow with PKCE; stdlib only
- `internal/services/webhook_service.go` — outbound alert delivery with HMAC-SHA256 body signing

### Hardening (round 13)

- WAL stress: `TestRoundtripAndReplay`, `TestCrashRecovery` (torn-write at line boundary), `TestConcurrentAppend` (1000 events from 20 goroutines)
- Concurrent audit chain stress test — 300 entries, chain still verifies under contention
- `TestCaseSnapshotLeakUnderConcurrentIngest` — proves the snapshot is leak-proof under racing ingest
- `TestConcurrentCaseLifecycle` — 8 parallel case workflows
- `cmd/smoke` — 43-endpoint exerciser; `✓` per check, exit 1 on first surprise
- `task ci` — fmt + vet + tests + frontend build
- `docs/security/security-review.md` — written threat model + ops posture
- `docs/operator/deployment.md` — systemd unit + reverse-proxy config + backup recipe + soak gate + decommission

### Foundations (rounds 9–12)

- §2 foundational integrity — durable audit journal (`audit.log`), tamper-evident query log (`auditmw`), per-event provenance + content hash + schema version, time-anchored daily Merkle anchor
- §3 time-frozen investigation snapshots — case opens capture audit-root + receivedAt cutoff
- §5 reconstruction engine — sessions / state-at-T / cmdline / multi-protocol auth chains / process lineage / network stitching / entity profiles
- §6 self-contained offline verifier (`oblivra-verify`)
- §7 storage tiering — Parquet warm tier with WORM lock, schema-versioned to v2, S3 cold-tier scaffold
- §8 investigations — hypothesis tracking, annotations, confidence scoring, legal-review state machine
- §9 log quality intelligence — source reliability, coverage, DLP redaction
- Phase 38 evidence package — deterministic HTML render
- Phase 39 advanced reconstruction — entity forensic profiles, command-line reconstruction, tamper signals

## 0.1.0 — initial Beta-1 cut

Initial public-repo cut. Wails v3 desktop shell + headless `cmd/server`, BadgerDB hot store, Bleve indices, Sigma rule loader, CLI tools, frontend Svelte UI.

---

**Two real bugs caught by the hardening test passes** — the kind of issue that quietly corrupts state in production and never produces a stack trace:

1. Scheduler nil-channel race on shutdown
2. Concurrent vault `.tmp` file collision on rename

Both fixes are in `main` along with the tests that caught them, so the regression won't return.
