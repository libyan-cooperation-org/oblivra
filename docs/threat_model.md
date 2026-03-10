# OBLIVRA — Formal Threat Model (STRIDE)

> Version 1.0 · 2026-03-01
> Methodology: Microsoft STRIDE + OWASP Threat Modeling

---

## 1. System Overview

OBLIVRA is a sovereign SIEM platform built as a single Go binary (Wails v2) with an embedded SolidJS frontend. It operates in air-gapped, government, and critical infrastructure environments.

### Process Boundary

Single OS process containing:
- SSH client + terminal multiplexer
- SIEM ingestion pipeline (Syslog → Parser → Enrichment → Detection → Storage)
- REST API server (TLS)
- Encrypted SQLite database (SQLCipher)
- BadgerDB event store
- Bleve search index
- AES-256 vault (in-process memory)
- Event bus (in-process pub/sub)

### External Interfaces

| Interface | Protocol | Port | Auth | Direction |
|---|---|---|---|---|
| REST API | HTTPS (TLS 1.3) | 8443 | API Key + RBAC | Inbound |
| Syslog Ingestion | TCP/UDP | 1514 | IP allowlist | Inbound |
| Prometheus Metrics | HTTP | 9090 | None (bind localhost) | Inbound |
| SSH to managed hosts | SSH | 22 | Key/Password/Agent | Outbound |
| SFTP file transfer | SSH/SFTP | 22 | Key/Password | Outbound |
| Threat Intel feeds | HTTPS | 443 | API Key | Outbound (optional) |
| Notification channels | HTTPS | 443 | Webhook/API Key | Outbound (optional) |

---

## 2. Data Flow Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    OBLIVRA PROCESS                       │
│                                                         │
│  ┌──────────┐   ┌──────────┐   ┌──────────────────┐    │
│  │ Syslog   │──▸│ Parser   │──▸│ Enrichment       │    │
│  │ Listener │   │ Pipeline │   │ (GeoIP,DNS,Asset) │    │
│  └──────────┘   └──────────┘   └────────┬─────────┘    │
│                                          │              │
│  ┌──────────┐   ┌──────────┐   ┌────────▼─────────┐    │
│  │ REST API │◂──│ Event Bus│◂──│ Detection Engine  │    │
│  │ (TLS)    │   │ (pub/sub)│   │ (YAML Rules,      │    │
│  └──────────┘   └──────────┘   │  Correlation,     │    │
│       │              │         │  MITRE Mapping)    │    │
│       │              │         └────────┬─────────┘    │
│  ┌────▼─────┐   ┌────▼─────┐   ┌───────▼──────────┐   │
│  │ SQLCipher│   │ BadgerDB │   │ Bleve Index      │   │
│  │ (hosts,  │   │ (events, │   │ (full-text       │   │
│  │  creds,  │   │  WAL,    │   │  search)         │   │
│  │  audit)  │   │  Merkle) │   └──────────────────┘   │
│  └──────────┘   └──────────┘                           │
│       │                                                 │
│  ┌────▼──────────────────────────────┐                  │
│  │ AES-256 Vault (in-memory keys)    │                  │
│  │ OS Keychain integration           │                  │
│  │ FIDO2/YubiKey                     │                  │
│  └───────────────────────────────────┘                  │
└─────────────────────────────────────────────────────────┘
         │                    ▲
         ▼                    │
  ┌──────────────┐    ┌───────────────┐
  │ Managed Hosts│    │ Syslog Sources│
  │ (SSH/SFTP)   │    │ (firewalls,   │
  └──────────────┘    │  servers,     │
                      │  endpoints)   │
                      └───────────────┘
```

---

## 3. Trust Boundaries

| Boundary | What Crosses | Trust Level |
|---|---|---|
| **TB-1: Network → OBLIVRA** | Syslog packets, REST requests | Untrusted → Authenticated |
| **TB-2: OBLIVRA → Managed Hosts** | SSH commands, SFTP transfers | Authenticated → Remote Trusted |
| **TB-3: OBLIVRA → External APIs** | Threat intel queries, notifications | Authenticated → External |
| **TB-4: User → REST API** | API calls with key + RBAC claims | Untrusted → Role-Verified |
| **TB-5: Wails Frontend → Go Backend** | IPC function calls | Trusted (same process) |
| **TB-6: In-Process Memory** | Vault keys, session credentials | Trusted (process boundary) |

---

## 4. STRIDE Analysis

### S — Spoofing

| Threat | Target | Severity | Mitigation | Status |
|---|---|---|---|---|
| Spoofed syslog source | Syslog listener | Medium | IP allowlist on ingestion | ✅ Implemented |
| API key theft | REST API | High | TLS-only transport, API key rotation | ✅ Implemented |
| SSH credential theft from vault | Vault | Critical | AES-256-GCM encryption, OS keychain, FIDO2 | ✅ Implemented |
| Man-in-the-middle on SSH | Managed hosts | High | Host key verification, known_hosts | ✅ Implemented |
| Forged audit log entries | Audit system | High | Merkle tree integrity chain | ✅ Implemented |

### T — Tampering

| Threat | Target | Severity | Mitigation | Status |
|---|---|---|---|---|
| Modify audit logs post-write | SQLite audit_logs | Critical | Merkle tree + HMAC chain verification | ✅ Implemented |
| Tamper with detection rules | YAML rule files | High | File integrity monitoring (Sentinel) | ✅ Implemented |
| Modify evidence after collection | Evidence locker | Critical | HMAC-signed chain of custody + seal | ✅ Implemented |
| Database corruption | SQLCipher/BadgerDB | Medium | BadgerDB checksums, encrypted at rest | ✅ Implemented |
| Binary replacement attack | OBLIVRA binary | High | Signed releases (Sigstore) | ⬜ Planned (Tier 2) |

### R — Repudiation

| Threat | Target | Severity | Mitigation | Status |
|---|---|---|---|---|
| User denies executing command | SSH sessions | Medium | Full session recording + audit log | ✅ Implemented |
| Admin denies deleting alerts | Alert dashboard | Medium | Audit log on all destructive actions | ✅ Implemented |
| Analyst denies accessing evidence | Evidence locker | High | Chain-of-custody with actor tracking | ✅ Implemented |
| Deny generating compliance report | Compliance engine | Low | Report generation logged in audit | ✅ Implemented |

### I — Information Disclosure

| Threat | Target | Severity | Mitigation | Status |
|---|---|---|---|---|
| Credential leak from memory dump | Vault in-process | Critical | Memory zeroing after use via `AccessMasterKey()` | ✅ Implemented |
| Log data exposed via metrics | Prometheus endpoint | Medium | Bind to localhost only, no PII in metrics | ✅ Implemented |
| Unencrypted database on disk | SQLite | High | SQLCipher with encryption key | ✅ Implemented |
| Source IP leak in error messages | REST API responses | Low | Sanitized error messages | ✅ Implemented |
| Secrets in git repository | Source code | Critical | `.gitignore` + Secrets-in-Code audit | ✅ Audited |

### D — Denial of Service

| Threat | Target | Severity | Mitigation | Status |
|---|---|---|---|---|
| Syslog flood (>10k EPS) | Ingestion pipeline | High | Bounded queue + backpressure + drop policy | ✅ Implemented |
| Regex-based rule causing ReDoS | Detection engine | High | Regex timeout + safe parsing | ✅ Implemented |
| BadgerDB disk exhaustion | Storage | Medium | Async value log GC | ✅ Implemented |
| Correlation state memory bomb | Detection engine | High | LRU/TTL bounded memory (500MB cap) | ✅ Implemented |
| Goroutine leak under load | Go runtime | Medium | Context cancellation on all workers | ✅ Implemented |
| Search query abuse | Bleve index | Medium | Query timeout (10s context) | ✅ Implemented |

### E — Elevation of Privilege

| Threat | Target | Severity | Mitigation | Status |
|---|---|---|---|---|
| Unauthenticated API access | REST endpoints | Critical | API key middleware on all routes | ✅ Implemented |
| Non-admin performing destructive action | Alert/host deletion | High | RBAC enforcement on destructive endpoints | ✅ Implemented |
| SQL injection via search | SQLite/Bleve | High | Parameterized queries, Bleve query parser | ✅ Implemented |
| Plugin sandbox escape | Lua runtime | Medium | Sandbox with no filesystem/network access | ✅ Implemented |
| SSRF via threat intel feeds | Outbound HTTP | Medium | URL validation + allowlist | ⬜ Planned |

---

## 5. Insider Threat Assumptions

| Assumption | Rationale |
|---|---|
| **Operator with shell access can read process memory** | Single-process Go binary; all secrets in process memory. Mitigation: vault zeroes keys after use. |
| **Admin with API key can delete all data** | By design (RBAC enforced). Mitigation: audit log is append-only + Merkle-chained. |
| **Analyst can view all events** | Role-based filtering planned but not enforced per-event. Mitigation: field-level redaction for PII. |
| **Physical access = full compromise** | Standard for on-premise. Mitigation: full-disk encryption (OS-level), SQLCipher. |

---

## 6. Supply-Chain Threats

| Threat | Severity | Mitigation | Status |
|---|---|---|---|
| Compromised Go dependency | High | `govulncheck`, module pinning, SBOM | ✅ govulncheck clean, ⬜ SBOM planned |
| Compromised npm dependency | Medium | `bun audit`, lockfile pinning | ✅ Lockfile exists |
| Tampered GitHub Actions runner | Medium | Reproducible builds, Sigstore signing | ⬜ Planned |
| Malicious Docker base image | Medium | Pin Alpine digest, multi-stage build | ✅ Multi-stage build |

---

## 7. Risk Summary

| Risk Level | Count | Category |
|---|---|---|
| ✅ Mitigated | 28 | Active controls in code |
| ⬜ Planned | 4 | In roadmap (Tier 2 meta-layer) |
| 🔴 Accepted | 2 | Physical access, insider with shell |

**Overall Assessment:** The application implements defense-in-depth across all STRIDE categories. The primary residual risks are physical access compromise (standard for on-premise) and the single-process trust boundary (mitigated by memory zeroing and audit trails).
