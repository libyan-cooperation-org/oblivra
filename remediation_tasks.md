# Audit Remediation Tasks

## [x] Phase 1: Critical Pipeline & Concurrency Fixes
- [x] Fix C1: Track worker restarts in `ingest/pipeline.go` WaitGroup
- [x] Fix C2: Implement `sync.Once` for idempotent pipeline shutdown
- [x] Fix C5: Resolve WebSocket goroutine leak in `rest.go` (Read Loop + Select)

## [x] Phase 2: Auth & Security Hardening
- [x] Fix C4: Unify context keys for user identity in `auth/rbac.go` and `auth/apikey.go`
- [x] Fix 3.6: Implement random 32-byte vault canary in `vault/vault.go`
- [x] Fix C3: Remove hardcoded policy verification key in `container.go`
- [x] Fix C6: Remove hardcoded internal IPs in `app/host_service.go`

## [x] Phase 3: Lifecycle & Error Handling
- [x] Fix C7: Update `container.Init()` to fail hard on critical subsystem failure
- [x] Propagate service context to all background workers in Ingest Pipeline
- [x] Verify fix for C5 with manual testing (Build Verification)

## [x] Phase 5: Architectural Hardening (Kernel & Registry)
- [x] Refactor `container.go` into domain-specific clusters (Initial Decoupling)
- [x] Implement `platform` package with `Service` interface and `Registry`
- [x] Implement Topological Sort for dependency-aware startup in `platform/kernel.go`
- [x] Move clustered initialization into specialized bootstrap modules (Refactored `internal/app` to `internal/services`)

## [x] Phase 6: Observability & Integrity
- [x] Implement Unified Metrics & Tracing across all ingestion pipelines
- [x] Standardize Health Checks for all 60+ embedded services
- [x] Implement Structured Logging (zap/zerolog)
- [x] Enforce Rigid Schema Validation at the agent ingress boundary

## [x] Phase 7: Enterprise Readiness (PRR Follow-up)
- [x] Implement Backpressure in Ingest Pipeline (Bounded Channels)
- [x] Add Panic Recovery to all Background Workers
- [x] Implement Agent Rate Limiting & Connection Limits
- [x] Add Structured Event Versioning (v1, v2)
- [x] Harden WebSocket Security (Origin validation, connection limits)
- [x] Implement Memory Pressure Protection (runtime health)

## [x] Phase 8: FAANG-Grade Architecture Upgrades
- [x] Dedicated Runtime Layer (Platform Kernel)
- [x] Typed Event Fabric
- [ ] DAG-based Streaming Processing Engine
- [x] Plugin Sandboxing (Process isolation or WASM)

## Phase 9 — Supply Chain & Hardening
- [ ] H1 — SLSA Build Integrity & Binary Signing
- [x] H2 — Secure Memory Zeroing for secrets
