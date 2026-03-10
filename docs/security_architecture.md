# OBLIVRA — Security Architecture Document

> Version 1.0 · 2026-03-01
> Classification: Internal — Architecture Reference

---

## 1. Architecture Overview

OBLIVRA is a monolithic Go application (Wails v2) comprising 16+ services running in a single OS process. This document defines the security boundaries, trust levels, cryptographic controls, and isolation guarantees.

### Deployment Model

| Mode | Description | Network Requirements |
|---|---|---|
| **Desktop (Wails)** | Full GUI, local-only or LAN | None (air-gap capable) |
| **Headless Server** | CLI binary, REST API exposed | TCP 8443, 1514, 9090 |
| **Docker** | Containerized headless, non-root | Same as headless |

---

## 2. Service Trust Levels

Every service in `container.go` is classified by its trust level and the data it can access.

| Service | Trust Level | Data Access | Crypto Access |
|---|---|---|---|
| **VaultService** | Critical | Master key, all credentials | AES-256-GCM, HMAC |
| **ComplianceService** | High | Audit logs, sessions, hosts, vault signing | HMAC via vault |
| **ForensicsService** | High | Evidence items, chain of custody | HMAC via vault |
| **SIEMService** | High | All security events, alerts, risk scores | None |
| **AlertingService** | High | Detection matches, notification channels | None |
| **SSHService** | High | Host credentials (via vault), session data | Key management |
| **SecurityService** | High | FIDO2 keys, certificates, YubiKey | Asymmetric crypto |
| **APIService** | Medium | Authenticated API surface | API key verification |
| **HostService** | Medium | Host inventory, connection metadata | None |
| **LogSourceService** | Medium | Log source configurations (credentials redacted) | None |
| **LocalService** | Medium | Local PTY sessions | None |
| **FileService** | Medium | Remote file listing, SFTP transfers | None |
| **SnippetService** | Low | Command snippets | None |
| **NotesService** | Low | Runbooks, notes | None |
| **ThemeService** | Low | UI themes, fonts | None |
| **WorkspaceService** | Low | Layout preferences | None |
| **SettingsService** | Low | User configuration | None |
| **TelemetryService** | Low | System metrics | None |

---

## 3. Cryptographic Architecture

### At-Rest Encryption

| Data Store | Encryption | Key Management |
|---|---|---|
| SQLite (hosts, sessions, credentials, audit) | SQLCipher (AES-256-CBC) | Vault master key |
| BadgerDB (events, WAL, Merkle leaves) | Badger built-in encryption | Derived key |
| AES Vault (`vault.db`) | AES-256-GCM | Master key derived via Argon2id from user passphrase |
| OS Keychain | OS-native (DPAPI/Keychain/libsecret) | OS-managed |
| Parquet archives | Unencrypted (archival tier) | ⬜ Planned |

### In-Transit Encryption

| Channel | Protocol | Certificate |
|---|---|---|
| REST API | TLS 1.3 | Self-signed or CA-issued (`cmd/certgen/`) |
| SSH connections | SSH (Curve25519, Ed25519) | Host key verification |
| Syslog (TCP) | TLS optional | Configurable |
| Notifications (webhooks) | HTTPS | System CA bundle |

### Key Lifecycle

```
User Passphrase
      │
      ▼
  Argon2id (salt, time=3, memory=64MB, threads=4)
      │
      ▼
  Master Key (32 bytes)
      │
      ├──▸ AES-256-GCM: Encrypt/Decrypt vault entries
      ├──▸ HMAC-SHA256: Sign audit logs, evidence chain
      ├──▸ SQLCipher: Database encryption key
      └──▸ Zeroed from memory after each operation
```

### Memory Zeroing Guarantees

| Operation | Zeroed After Use |
|---|---|
| `vault.AccessMasterKey()` | ✅ Key copied to callback, original stays in vault only |
| Credential decrypt for SSH | ✅ Credential zeroed after SSH handshake |
| HMAC signing (compliance) | ✅ Key accessed via callback, never stored |
| HMAC signing (evidence) | ⬜ Key stored in EvidenceLocker struct — to be improved |

---

## 4. Isolation Boundaries

### Process-Level

OBLIVRA runs as a **single Go process**. All services share the same memory space. This is a deliberate architectural choice for:
- Simplicity of deployment (single binary)
- Air-gap compatibility (no inter-process networking)
- Performance (zero-copy event bus)

**Tradeoff acknowledged:** If any service is compromised (e.g., via malicious plugin), the attacker has access to the full process memory, including vault keys.

**Mitigations:**
1. Plugin sandbox (Lua) has no access to filesystem, network, or Go runtime
2. Vault keys are zeroed after each operation
3. Docker deployment runs as non-root user with memory limits
4. Future: consider process isolation for vault signing service (Tier 3)

### Network-Level

| Service | Binding | Exposure |
|---|---|---|
| REST API | `0.0.0.0:8443` (configurable) | TLS + API key required |
| Syslog | `0.0.0.0:1514` (configurable) | IP allowlist recommended |
| Prometheus | `127.0.0.1:9090` | Localhost only |
| SSH outbound | Dynamic | Per-host authentication |

### Data-Level Isolation

| Data Type | Store | Access Control | Encryption |
|---|---|---|---|
| Credentials | SQLCipher → Vault-encrypted blobs | VaultService only | AES-256-GCM |
| Audit logs | SQLCipher + Merkle tree | Append-only (no update/delete API) | SQLCipher |
| Security events | BadgerDB | SIEMService, read via API | Badger encryption |
| Evidence items | BadgerDB | ForensicsService, HMAC-signed | ✅ |
| Host metadata | SQLCipher | HostService | SQLCipher |
| User settings | SQLCipher | SettingsService | SQLCipher |

---

## 5. Authentication & Authorization

### Authentication Methods

| Method | Used For | Strength |
|---|---|---|
| API Key (header) | REST API access | Medium — static secret |
| Vault passphrase | Unlock encrypted vault | High — Argon2id derived |
| SSH key/password | Managed host connections | Varies by host config |
| FIDO2/YubiKey | Hardware second factor | High — phishing-resistant |
| OS Keychain | Automatic vault unlock | Medium — OS-dependent |

### Authorization (RBAC)

| Role | Permissions |
|---|---|
| **admin** | Full access: create/delete hosts, purge logs, manage users |
| **analyst** | Read events, acknowledge alerts, generate reports |
| **viewer** | Read-only: dashboards, search, compliance reports |

Enforcement: `internal/auth/` middleware on REST routes. Destructive endpoints (`DELETE`, `PURGE`) require `admin` role.

---

## 6. Audit Trail Architecture

```
User Action
    │
    ▼
AuditRepository.Log()
    │
    ├──▸ SQLite: INSERT into audit_logs (id, timestamp, event_type, host_id, session_id, details)
    │
    └──▸ MerkleTree.AddLeaf(serialized_entry)
              │
              └──▸ leaf_hash stored alongside audit record
                   root recomputed on each insertion
                   tamper = root mismatch
```

**Properties:**
- Append-only (no UPDATE or DELETE on audit_logs exposed via API)
- Purge only via admin CLI with explicit `--before` date (audit-logged action)
- Merkle proofs exportable for third-party verification
- Evidence locker entries have independent HMAC chain

---

## 7. Threat Response Matrix

| Scenario | Detection | Response | Recovery |
|---|---|---|---|
| **Brute-force SSH** | Failed login threshold alert | Block IP (manual), lock account | Review audit log |
| **Syslog flood DoS** | Bounded queue backpressure alert | Drop oldest, alert operator | Queue drains automatically |
| **Compromised API key** | Unusual API pattern alert | Rotate key, revoke old | Audit log review |
| **Database corruption** | BadgerDB checksum failure | Alert, enter read-only mode | Restore from Parquet archives |
| **Vault compromise** | Physical access assumed | Kill-switch safe mode (planned) | Re-key vault, rotate all credentials |
| **Detection rule bypass** | Coverage gap report (planned) | Update rules, red team test | MITRE heatmap review |

---

## 8. Compliance Mapping

| Requirement | Standard | OBLIVRA Control |
|---|---|---|
| Encryption at rest | PCI-DSS 3.1, NIST SC-28, GDPR Art.32 | SQLCipher + AES Vault |
| Encryption in transit | PCI-DSS 1.1, NIST SC-8, HIPAA 164.312.e | TLS 1.3 on all listeners |
| Audit logging | PCI-DSS 10.1, NIST AU-2, SOC2 CC7.2 | AuditRepository + Merkle tree |
| Access control | PCI-DSS 7.1, NIST AC-2, ISO A.5.15 | RBAC + API key |
| Incident response | HIPAA 164.308.a.6, SOC2 CC7.3 | Evidence locker + forensics service |
| Data integrity | NIST AU-9, ISO A.5.28, GDPR Art.32.1.b | Merkle tree + HMAC chain |
