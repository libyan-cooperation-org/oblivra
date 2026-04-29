# OBLIVRA — Feature Reference

> **Single source of truth for what OBLIVRA does and does not do.**
>
> Positioning (post-Phase 36): **sovereign, log-driven security and forensic intelligence platform**. We ingest, detect, correlate, profile, and seal log evidence. Active response (containment, kill, isolate, IR playbooks) is delegated to external SOAR.
>
> See also: [`README.md`](../README.md) for marketing-level overview, [`task.md`](../task.md) for build phases, [`HARDENING.md`](../HARDENING.md) for security ledger.

**Last updated**: 2026-04-29 · matches the codebase at `internal/services/`, `internal/detection/`, `internal/forensics/`, `internal/agent/`.

---

## Status legend
- `[x]` Production-Ready (hardened, documented, soak-tested)
- `[v]` Validated (functionally correct, tested under load)
- `[s]` Scaffolded (code exists and compiles)
- `[ ]` Not started

---

## 1. Ingestion & Storage

| Status | Feature | Surface | Notes |
|---|---|---|---|
| `[x]` | Syslog ingest (RFC 5424 + 3164) | TCP/UDP 514 | TLS optional |
| `[x]` | JSON / CEF / LEEF parsers | Native | Auto-detected on stream |
| `[x]` | Windows EVTX parser | Native ingest | Top 30 event IDs deep-parsed |
| `[v]` | journald via agent backfill | Agent-side | Native ingest parser is generic Linux auth (sshd/sudo); deep journald parsing happens on the agent. |
| `[x]` | BadgerDB hot store | Internal | Real-time event lookup, sub-second |
| `[x]` | Bleve full-text index | Internal | Field-level search + aggregations |
| `[x]` | Parquet warm tier | Internal | 30-180d archival |
| `[s]` | S3-compatible cold tier | Build-tagged | Stub interface; full impl Phase 22.3 |
| `[x]` | Crash-safe Write-Ahead Log | Internal | Zero loss on restart |
| `[x]` | Storage tiering migrator | REST + UI | Hot → Warm → Cold lifecycle |
| `[s]` | Sustained 10k+ EPS | Benchmark | Has Go benchmark; no documented soak test yet |

## 2. Query & Search

| Status | Feature | Notes |
|---|---|---|
| `[x]` | OQL — Oblivra Query Language | Native, pipe syntax, ~SPL/KQL parity for common patterns |
| `[x]` | Sigma rule transpiler | 82+ built-in rules + community Sigma `.yml` hot-reload |
| `[x]` | KQL/SPL transpiler | Lossy — surfaces what's translatable |
| `[x]` | Live tail / WebSocket events | <100 ms latency |
| `[x]` | Saved searches | Per-tenant |

## 3. Detection & Analytics

| Status | Feature | Notes |
|---|---|---|
| `[x]` | Sigma engine + correlation engine | Threshold / Frequency / Sequence / Temporal / Graph |
| `[x]` | MITRE ATT&CK heatmap | Live coverage view |
| `[x]` | Multi-stage fusion engine | Cross-host campaign clustering |
| `[x]` | UEBA — entity baselines + Isolation Forest | Per-tenant |
| `[x]` | NDR — NetFlow/IPFIX, DNS tunneling, JA3/JA3S | Lateral-movement detection |
| `[x]` | Ransomware **detection** (entropy-based) | Detection only — response actions removed Phase 36. Pair with external SOAR. |
| `[x]` | Risk-based alerting | Severity + UEBA + IOC fusion |

## 4. Forensics & Evidence

| Status | Feature | Notes |
|---|---|---|
| `[x]` | Merkle-chained audit log | Tamper-evident |
| `[x]` | Evidence locker with chain-of-custody | HMAC-signed log-event captures |
| `[x]` | RFC 3161 timestamping | FreeTSA default + PKCS#7 tokens |
| `[v]` | Temporal integrity (clock-drift, timestamp guards) | DHCP-aware entity resolution **pending** |
| `[s]` | Audit-evidence pack export | Locker + Merkle exist; formal export endpoint pending Phase 38 |
| `[x]` | Centralized DLP redactor | Tenant-scoped patterns |
| `[ ]` | WORM mode for warm/cold tiers | Phase 38 |
| `[ ]` | Offline evidence-verification CLI | Phase 38 |
| `[ ]` | Expert-witness export bundle | Phase 39 |

## 5. Agent Framework

| Status | Feature | Notes |
|---|---|---|
| `[x]` | Lightweight Go agent | HTTP + mTLS + zlib transport |
| `[x]` | File tailing collector | |
| `[x]` | Windows Event Log collector | |
| `[x]` | journald collector | Agent-side, deep parse |
| `[x]` | Metrics collector | CPU / memory / load / network |
| `[x]` | File Integrity Monitoring (FIM) | |
| `[x]` | Offline buffering (local WAL) | Crash-safe |
| `[x]` | Edge filtering / local detection | Reduces upstream EPS |
| `[x]` | Agentless WMI collector | Windows event log via WMI/WinRM |
| `[x]` | Agentless SNMP collector | v2c/v3 trap listener |
| `[x]` | Agentless REST poller | Declarative config for SaaS sources |
| `[x]` | Agent oplog (tamper-detection) | Merkle hash chain over agent batches |
| `[x]` | Agent heartbeat | log-file size/inode tracking + clock-skew detection |
| `[x]` | Encrypted agent config storage | Ed25519 private key encryption |

## 6. Identity, Auth & Multi-Tenancy

| Status | Feature | Notes |
|---|---|---|
| `[x]` | Multi-tenant data isolation | Enforced at every search/audit/cloud-asset query |
| `[x]` | RBAC | Admin / Analyst / ReadOnly / Agent canonical roles |
| `[x]` | OIDC | `coreos/go-oidc` |
| `[x]` | SAML 2.0 | `crewjam/saml` |
| `[x]` | TOTP MFA | Software MFA for local accounts |
| `[s]` | FIDO2 / YubiKey hardware MFA | Desktop-only (Wails) |
| `[x]` | API-key auth + JWT | |
| `[x]` | Login lockout (persistent) | `login_lockouts` table |
| `[x]` | DSR workflow | GDPR/CCPA data-subject requests |

## 7. Vault & Crypto

| Status | Feature | Notes |
|---|---|---|
| `[x]` | AES-256-GCM vault | Argon2id KDF |
| `[x]` | OS keychain integration | DPAPI / Keychain / Secret Service |
| `[x]` | TPM PCR binding | Optional hardware seal |
| `[x]` | Nuclear wipe | Multi-pass cryptographic erasure |
| `[x]` | TLS certificate generation | Per-tenant CA |

## 8. UX & Operator Surface

| Status | Feature | Notes |
|---|---|---|
| `[x]` | Hybrid Desktop (Wails) + Web (Browser) | Context-aware feature gating |
| `[x]` | Investigation-first UI | HostDetail, InvestigationPanel, EntityLink |
| `[x]` | Multi-monitor pop-out windows | Workspace save/restore |
| `[x]` | Command palette (⌘K) | |
| `[x]` | Notification center | |
| `[x]` | Unified time-range picker | |
| `[x]` | 6-domain navigation | SIEM / INVEST / RESPOND / FLEET / GOVERN / ADMIN |

## 9. Self-Observability

| Status | Feature | Notes |
|---|---|---|
| `[x]` | Self-ingest via OBLIVRA agent | Goroutines / heap / EPS / detection rate |
| `[x]` | `/metrics` Prometheus scrape target | Optional — point your existing Prometheus at OBLIVRA if you want |
| `[x]` | `/debug/attestation` runtime hash | |
| `[x]` | `/monitoring` diagnostics page | Live service status |

> Phase 36 removed the bundled external observability stack (Prometheus / Grafana / Tempo). OBLIVRA ingests its own platform telemetry into the SIEM pipeline; pair with your own observability tooling if desired.

---

## Removed in Phase 36 (broad scope cut)

These features have been deleted from OBLIVRA. **Pair with external tooling** if you need them.

| Removed | Replacement (external) |
|---|---|
| SOAR / playbook execution | Tines, Torq, XSOAR, Shuffle |
| Incident response automation | Same |
| Ransomware response actions (isolate / kill / quarantine) | Same |
| Disk imaging / memory acquisition | Velociraptor, FTK, Volatility |
| AI assistant / LLM chat | Any LLM (OpenAI, Anthropic, local Ollama) |
| Plugin framework | — (no equivalent — re-architect if needed) |
| External observability stack (Prometheus / Grafana / Tempo) | Self-host your own; OBLIVRA ships `/metrics` |
| Compliance YAML packs (PCI-DSS / NIST / ISO / GDPR / HIPAA / SOC2) + report generator | Drata, Vanta, Tugboat Logic — feed them OBLIVRA's `/api/v1/audit/packages` for evidence |

## Removed in Phase 32

| Removed | Replacement |
|---|---|
| Interactive shell / SSH client / PTY / terminal grid | Use a real terminal (Windows Terminal, iTerm2). OBLIVRA's role is observability, not access. |
| SFTP browser / port-forwarding tunnels | Use OS-native tools |
| Session recording playback | Pair with a dedicated session-recording product |

> The Go libraries under `internal/ssh/` and `internal/services/{ssh,local,tunnel,recording,share,multiexec,broadcast,file,transfer,pty}_*.go` survive in-tree because non-terminal features still depend on them (canary deployment via SCP, scheduled SSH key rotation, evidence file uploads).

---

## Active Build Phases (see `task.md`)

- **Phase 37** — Log Forensics Core: gap detection, EVTX/journald deep parse expansion, unified forensic timeline, basic evidence-pack export, OQL forensic templates, Trust-tier (TE/VE/BE) enforcement.
- **Phase 38** — Court Admissibility: full forensic evidence package (PDF/HTML + signatures + verification instructions), offline-verifier CLI, WORM mode (Windows ReFS / Linux `chattr +i`), templated narrative builder (no LLM), expanded chain-of-custody UI, legal-review gate.
- **Phase 39** — Advanced Log Forensics: process-lineage reconstruction, authentication/session reconstruction, entity forensic profiles (Host / User / IP), tampered/deleted log indicators, expert-witness export bundle.
- **Phase 22.3** — Storage Tiering Polish (carry-over): ingest-through-HotTier-interface routing, S3 cold tier, per-tenant retention, cross-tier integrity verification.
