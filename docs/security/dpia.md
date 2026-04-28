# Data Protection Impact Assessment (DPIA)

**System:** OBLIVRA Sovereign SIEM Platform
**Version covered:** 1.x
**Last reviewed:** 2026-04-28
**Reviewer:** Engineering — pending privacy-counsel sign-off
**Article 35 GDPR · ICO ASS-DP-DPIA template**

---

## 1. Need for the DPIA

OBLIVRA processes personal data at scale (security telemetry from monitored hosts, user identity records, command-line audit trails) and includes:

- Systematic profiling of users via UEBA (Article 35(3)(a))
- Large-scale processing of special-category data — credentials, MFA tokens (Article 35(3)(b))
- Systematic monitoring of accessible areas (network telemetry from agents) (Article 35(3)(c))

→ **A DPIA is mandatory under GDPR Article 35(3)**.

---

## 2. Description of Processing

### 2.1 Nature

OBLIVRA collects, correlates, and stores security telemetry from operator-installed agents on customer-owned hosts. It detects threats via Sigma rules, alerts the operator, and orchestrates response actions (host isolation, evidence sealing, playbook execution).

### 2.2 Scope

Personal data categories handled:

| Category | Examples in OBLIVRA | Article 9 special category? |
|---|---|---|
| **Identifiers** | User email, agent hostname, source IP, user ID | No (but quasi-identifying when combined) |
| **Authentication artifacts** | Password hashes, TOTP secrets, OIDC subject IDs, SAML NameIDs | Yes — security and authentication |
| **Behavioral profiles** | Command history, login times, anomaly scores, "risky user" flags | No (potentially profiling under Art. 22) |
| **Content of communications** | Captured terminal sessions (recordings), log lines | No (but may incidentally contain Art. 9 data) |
| **Location** | Source IP geolocation, geo-binned threat maps | No (city-level — not precise location) |

### 2.3 Context

- **Operator side:** SOC analysts, security admins, compliance officers
- **Subjects:** Employees of the customer (whose endpoints run agents), external attackers, occasional contractors
- **Lawful basis:** GDPR Art. 6(1)(f) — legitimate interest (security monitoring); Art. 6(1)(c) when required for compliance (SOC2, HIPAA, PCI)
- **Recipients:** Operator's SOC team only. No third-party SaaS by default. Optional integrations (PagerDuty, Slack, ServiceNow) are operator-opt-in and out-of-scope for this DPIA when disabled.

### 2.4 Purposes

1. Detect security incidents on customer endpoints
2. Provide auditable evidence for compliance frameworks (SOC2, HIPAA, ISO 27001)
3. Enable forensic analysis of past incidents
4. Maintain a tamper-evident record of operator actions

---

## 3. Data Flow

```
                ┌────────────────────────────────────────────────────────────┐
                │                  CUSTOMER NETWORK BOUNDARY                  │
                │                                                             │
   ┌─ HOSTS ─┐  │   ┌────────────────┐                  ┌──────────────────┐ │
   │ workstn │──┼──▶│  oblivra-agent │                  │   OBLIVRA SERVER │ │
   │ server  │──┼──▶│  (per-host)    │ HMAC + TLS 1.3 ─▶│  (REST + ingest) │ │
   │ laptop  │──┼──▶│                │                  │                  │ │
   └─────────┘  │   │   collects:    │                  │  receives:       │ │
                │   │   - syslog     │                  │  - event batches │ │
                │   │   - process    │                  │  - host metrics  │ │
                │   │   - FIM        │                  │  - command-line  │ │
                │   │   - command    │                  │                  │ │
                │   │     history    │                  │  forwards to ↓   │ │
                │   └────────────────┘                  └──────────────────┘ │
                │                                              │              │
                │                                              ▼              │
                │                                    ┌─────────────────────┐  │
                │                                    │  CORRELATION ENGINE │  │
                │                                    │  - Sigma rules      │  │
                │                                    │  - UEBA baselines   │  │
                │                                    │  - threat-intel     │  │
                │                                    │    enrichment       │  │
                │                                    └─────────────────────┘  │
                │                                              │              │
                │                                              ▼              │
                │                       ┌──────────────────────────────────┐  │
                │                       │            STORAGE TIERS         │  │
                │                       │                                  │  │
                │                       │ HOT       BadgerDB   <7 days     │  │
                │                       │ WARM      Parquet    7–90 days   │  │
                │                       │ COLD      S3-compat  >90 days    │  │
                │                       │ INDEX     Bleve      full-text   │  │
                │                       │ AUDIT     SQLite     Merkle chn  │  │
                │                       │ RECORDINGS  filesystem (asciinem)│  │
                │                       └──────────────────────────────────┘  │
                │                                              │              │
                │                                              ▼              │
                │                       ┌──────────────────────────────────┐  │
                │                       │    OPERATOR-FACING (TLS 1.3)     │  │
                │                       │    - Wails desktop GUI            │  │
                │                       │    - REST + WebSocket             │  │
                │                       │    - CLI replay tools             │  │
                │                       └──────────────────────────────────┘  │
                │                                                             │
                └────────────────────────────────────────────────────────────┘

       OUT-OF-BAND (operator-opt-in, off by default):
         · webhook → Slack / PagerDuty   · TI feed pulls (HTTPS)
         · OIDC/SAML IdP federated login · email alerts (SMTP)
```

### 3.1 PII residency

**All PII stays inside the customer's environment.** OBLIVRA is a self-hosted product:

- **Hot+warm tiers:** local disk on the OBLIVRA server (encrypted at rest via FDE — see [at-rest-encryption.md](at-rest-encryption.md))
- **Cold tier:** S3-compatible storage **owned by the customer** (typically MinIO on-prem, AWS S3 with customer-managed KMS, or Azure Blob with customer-managed keys)
- **Index:** local Bleve files on the OBLIVRA server (covered by FDE)
- **Audit:** local SQLite (FDE-protected, hash-chained, RFC 3161 timestamped)
- **Recordings:** local filesystem (asciinema cast format; FDE)

No telemetry leaves the customer environment unless an operator explicitly configures a webhook, SaaS integration, or threat-intel feed. Each such integration appears in the connector list and is documented in the operator runbook with its own data-flow appendix.

### 3.2 Cross-border transfers

By default: none. Cross-border applies only when the operator explicitly:
1. Configures an OIDC/SAML IdP outside their jurisdiction
2. Subscribes to a TI feed with provider outside their jurisdiction
3. Routes alerts via a webhook to an out-of-region destination

Each of these is opt-in and operator-controlled; OBLIVRA itself does not initiate cross-border transfers.

---

## 4. Risk Assessment

### 4.1 Likelihood × Impact matrix

| Risk | Likelihood | Impact | Score | Mitigation |
|---|---|---|---|---|
| Operator misuse (SOC analyst views non-incident-related user data) | Medium | High | 6 | RBAC scope tags on every search; immutable audit log of who-queried-what |
| Stored credential leak via SQL injection | Low | Critical | 4 | Parameterised queries everywhere; no string-built SQL in privileged paths (see audit doc) |
| Tampered evidence chain submitted to court | Low | Critical | 4 | RFC 3161 third-party timestamps (every 24h); operator-supplied trust anchor |
| Privileged operator silently rewrites audit | Medium | High | 6 | Same as above + recommended off-host backup rotation |
| Agent → server traffic intercepted | Low | High | 3 | TLS 1.3; mutual auth via HMAC fleet secret; cert pinning option |
| Subject sees stale data after deletion request | Low | Medium | 2 | DSR workflow crypto-wipes from hot+warm tiers; cold-tier purge runs on next compaction (≤24h) |
| Backup tapes / cold S3 leaks PII at rest | Low | High | 3 | Customer-managed KMS keys mandatory in deployment guide; key rotation documented |
| User-controlled SIEM query becomes XSS in dashboard | Low | Medium | 2 | Frontend `{@html}` audit shows zero hits (Phase 31 audit) |

### 4.2 Necessity & Proportionality

OBLIVRA collects only telemetry the operator's collectors emit. Each collector (FIM, syslog, metrics, EventLog) is opt-in via agent config. The default agent install enables ONLY syslog + metrics — neither are PII-heavy. Command-history collection (high-PII) requires an explicit operator opt-in.

---

## 5. Operator Obligations

### 5.1 Records of Processing (Art. 30)

OBLIVRA writes processing records to two tables:

- `audit_logs` — every operator action against PII (who searched, who viewed, who exported)
- `dsr_requests` — every Article 15/17/20 subject request

The operator can produce an Art. 30 record from these by combining the audit_log with their connector-config history.

### 5.2 Data Subject Rights (Art. 15–22)

| Right | Endpoint | Behavior |
|---|---|---|
| Access (Art. 15) | `POST /api/v1/dsr/requests` with `request_type=access` | Admin fulfills; export bundles all rows from users + audit_logs + hosts matching subject_id |
| Rectification (Art. 16) | `PATCH /api/v1/users/:id` | Admin-only; logged |
| Erasure (Art. 17) | `POST /api/v1/dsr/requests` with `request_type=deletion` | Admin fulfills; crypto-wipes user + host rows; pseudonymises audit_log actor (legal retention obligation) |
| Restriction (Art. 18) | Tenant-level `disable=true` flag | Halts ingest from a tenant's agents |
| Portability (Art. 20) | Export from access request (Art. 15) is JSON — meets portability requirement |
| Objection (Art. 21) | Out-of-band — no automated profiling decisions, so right doesn't bind here |
| Automated decisions (Art. 22) | UEBA produces RECOMMENDATIONS; final isolation/quarantine decision is always operator-confirmed (no fully automated significant decisions) |

### 5.3 Notification of Breach (Art. 33–34)

- Event bus emits `security.breach_detected` on detector triggers
- Operator must integrate with their incident-response process to meet the 72-hour notification window
- OBLIVRA does not auto-notify supervisory authorities (would need operator-jurisdiction context)

---

## 6. Outstanding Work

| Item | Status | Owner | Target |
|---|---|---|---|
| Privacy counsel review of this draft | Pending | Legal | Q3 2026 |
| Real PKCS#7 verify of TSA tokens | Stub-only | Engineering | Next minor |
| SAML SP-side metadata UI | Backend ready, UI pending | Engineering | Next minor |
| Encrypted Bleve index option | FDE-recommended; native encryption deferred | Engineering | Roadmap |
| Customer-supplied DPA template | Not yet drafted | Legal | Q3 2026 |

---

## 7. Sign-Off

| Role | Name | Date |
|---|---|---|
| Engineering Lead | | |
| Privacy Counsel | | |
| Data Protection Officer (if applicable) | | |
| Customer SOC Lead | | |

---

**This is an engineering-side DPIA draft.** It is structured for privacy-counsel review against the customer's specific deployment context. The data-flow diagram, RoPA mapping, and DSR endpoint table are reference material; the risk assessment is the section that needs the most legal input on each customer engagement.
