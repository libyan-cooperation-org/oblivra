# OBLIVRA — Data Flow Architecture

This document is the map an operator or a security reviewer needs to reason
about what's happening inside the platform. Every arrow is annotated with
the durability/integrity guarantee that the receiving stage adds.

## The big picture

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                                                              │
│   syslog UDP   ─┐                                                            │
│   NetFlow v5   ─┤                                                            │
│   REST API     ─┼─►  ingest.Pipeline   ─►  WAL    ─►  Hot store  ─► Bleve   │
│   agent batch  ─┤    (events.Validate                  (BadgerDB)  (per-    │
│   import bulk  ─┘     + content hash)                              tenant)  │
│                            │                                                 │
│                            ▼                                                 │
│                       events.Bus  ─►  Rules  ─►  Alerts  ─►  Webhooks       │
│                            │      ─►  UEBA   ──┐                             │
│                            │      ─►  Forensics ─►  Audit chain ─► Daily    │
│                            │      ─►  Lineage  ─┘   (durable           Merkle│
│                            │      ─►  IOC enrich    audit.log)        anchor│
│                            │      ─►  Trust   ─┐                             │
│                            │      ─►  Quality  ┘                             │
│                            │      ─►  Tamper                                 │
│                            │      ─►  Reconstruction (sessions / state /     │
│                            │           cmdline / network / entity / auth)   │
│                            │                                                 │
│                            ▼                                                 │
│                       WebSocket /api/v1/events  ─►  browser live tail        │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘

                ┌──────────────────────────┐
                │ Tiering migrator (every  │
                │ 6h, idempotent + crash-  │  ─►  Parquet warm (WORM-locked)
                │ safe write→fsync→evict)  │       │
                └──────────────────────────┘       ▼
                                              S3 cold (build-tagged)
```

## The integrity pipeline

A single ingested event passes through these stages, each adding a stronger
guarantee:

```
              ┌─────────────────────────────────────────────────────────────┐
              │                  events.Event arrives                        │
              │  - Source-typed (rest / syslog-udp / agent / import / raw)  │
              │  - Provenance recorded (peer / agentId / parser / TLS fp)   │
              └─────────────────────────────────────────────────────────────┘
                                       │
                       Validate() — fills defaults, computes:
                                       │
                                       ▼
              ┌─────────────────────────────────────────────────────────────┐
              │  Hash = sha256(canonical(event))                            │
              │  → mutating any field, including provenance, breaks hash    │
              └─────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
              ┌─────────────────────────────────────────────────────────────┐
              │  WAL append (line-delimited JSON, fsynced)                  │
              │  → recoverable across crash, replayable through migrator   │
              └─────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
              ┌─────────────────────────────────────────────────────────────┐
              │  Hot store (BadgerDB, per-tenant key prefix)                │
              │  → tenant isolation is structural — no shared keyspace     │
              └─────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
              ┌─────────────────────────────────────────────────────────────┐
              │  Bleve index (per-tenant)                                   │
              │  → search-time queries; failure here logs but doesn't drop │
              └─────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
              ┌─────────────────────────────────────────────────────────────┐
              │  events.Bus broadcast (drop-on-slow per subscriber)         │
              │  → ingest must never wait on a downstream processor        │
              └─────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
              ┌─────────────────────────────────────────────────────────────┐
              │  Async processors:                                          │
              │   - rules.Evaluate  → maybe raise Alert                     │
              │   - ueba.Observe    → baseline + z-score                    │
              │   - forensics       → log gaps, evidence sealing            │
              │   - lineage         → process tree (persistent journal)    │
              │   - reconstruction  → sessions, state, cmdline, auth, etc. │
              │   - trust           → verified/consistent/suspicious/untr. │
              │   - quality         → source reliability + coverage        │
              │   - tamper          → auditd / journalctl / clock signals  │
              │   - intel           → IOC match (raises ioc-match alert)   │
              └─────────────────────────────────────────────────────────────┘
                                       │
                       Alert raised? ──► AuditService.Append → audit.log
                                       │       (Merkle-chained, fsynced)
                                       │
                       Hourly: AuditService.AnchorYesterday
                                       │
                                       ▼
              ┌─────────────────────────────────────────────────────────────┐
              │  Daily Merkle root = sha256(every entry of yesterday's      │
              │  chain) → committed as a new chain entry tagged             │
              │  audit.daily-anchor. Future cross-day forgery now           │
              │  detectable by the verifier.                                │
              └─────────────────────────────────────────────────────────────┘
```

## Read-side: HTTP request to response

```
client request
   │
   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  security middleware    — sets X-Content-Type-Options, X-Frame-Options │
│  auth middleware (RBAC) — extracts Bearer key, attaches Subject to ctx │
│  audit middleware       — captures method/path/status/bytes/duration   │
│  ServeMux                                                              │
└─────────────────────────────────────────────────────────────────────────┘
   │
   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Handler (e.g. siemSearch)                                             │
│   - Parses URL params                                                  │
│   - Calls Service method (SiemService.Search)                          │
│   - Service may dispatch to Bleve (full-text) or hot-store (chrono)   │
│   - Service applies tenant scope from query                           │
└─────────────────────────────────────────────────────────────────────────┘
   │
   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  Response written → audit middleware records the entry in audit.log   │
│  with {actor, role, method, path, status, bytes, duration, query}     │
└─────────────────────────────────────────────────────────────────────────┘
```

## Investigation: opening a case

```
analyst: POST /api/v1/cases {title, hostId, fromUnix, toUnix}
   │
   ▼
InvestigationsService.Open
   │
   ├── snapshot the audit chain root (RootHash) → AuditRootAtOpen
   ├── snapshot time.Now() as ReceivedAtCutoff
   ├── persist case to cases.log (line-delimited JSON, fsynced)
   └── audit.Append(actor, "investigation.open", scope, root)

later: GET /api/v1/cases/{id}/timeline
   │
   ▼
InvestigationsService.Timeline(caseId)
   │
   ├── read events from hot store within [from, to]
   ├── FILTER OUT events whose receivedAt > ReceivedAtCutoff
   │   (this is what makes the snapshot tamper-proof — newer events
   │    arriving in the live store are invisible to the case)
   ├── merge in alerts, log gaps, sealed evidence packages
   └── sort by (timestamp DESC, kind, refId) for deterministic order

GET /api/v1/cases/{id}/report.html
   │
   ▼
ReportService.CaseHTML
   │
   ├── compose CaseInfo + Timeline + Confidence + AuditRoot into Package
   ├── render via deterministic html/template (no JS, no external assets)
   └── include verification instructions:
       "oblivra-verify --hmac $OBLIVRA_AUDIT_KEY audit.log"
```

## Storage tiering: hot → warm → cold

```
                    ┌───────────────────────────────┐
       (live ingest)│         Hot (BadgerDB)         │
       ───────────► │   per-tenant key prefix       │
                    │   ordered by nanosecond ts    │
                    └──────────────┬────────────────┘
                                   │
                  scheduled every 6h, plus on-demand:
                                   │
                                   ▼
                   ┌───────────────────────────────┐
                   │  TieringService.Promote       │
                   │   1. Range-scan hot for      │
                   │      events older than       │
                   │      tenant's HotMaxAge      │
                   │   2. Write Parquet file      │
                   │      (schema v2 with hash)   │
                   │   3. fsync                   │
                   │   4. WORM-lock the file      │
                   │   5. Delete from hot         │
                   │   ↑                          │
                   │   crash-safe: write→sync→    │
                   │   evict ordering means a     │
                   │   crash mid-pass leaves     │
                   │   a duplicate in warm,      │
                   │   never a missing event     │
                   └──────────────┬────────────────┘
                                   │
                                   ▼
                   ┌───────────────────────────────┐
                   │       Warm (Parquet)          │
                   │   WORM-locked files           │
                   │   schema v2: schemaVersion,   │
                   │   hash, full provenance       │
                   └──────────────┬────────────────┘
                                   │
              periodic verification (cross-tier):
                                   │
                   for the most recent N parquet
                   files, recompute every row's
                   content hash → must match
                                   │
                                   ▼
                   ┌───────────────────────────────┐
                   │     Cold (post-Beta)          │
                   │   - LocalStore (default,      │
                   │     mimics WORM)              │
                   │   - S3Store (build-tagged,    │
                   │     SigV4, no SDK)           │
                   └───────────────────────────────┘
```

## Cross-cutting: what the audit chain captures

Every meaningful action lands in `audit.log`. The chain is:

```
entry[N].hash = sha256(canonical(
    seq, timestamp, actor, action, tenantId, detail,
    parentHash = entry[N-1].hash
))
entry[N].signature = HMAC-SHA256(audit-key, entry[N].hash)
```

What lands:

| Action                         | Source                                    |
|--------------------------------|-------------------------------------------|
| `platform.start`               | server boot                               |
| `siem.search` / `siem.oql`     | auditmw on every analyst query            |
| `siem.ingest.raw`              | auditmw                                   |
| `audit.read` / `audit.verify`  | auditmw                                   |
| `audit.export`                 | evidence package generation               |
| `audit.daily-anchor`           | hourly scheduler job                      |
| `evidence.seal`                | ForensicsService.CollectByHost            |
| `investigation.open`           | InvestigationsService.Open                |
| `investigation.note`           | annotation added to a case                |
| `investigation.hypothesis.*`   | hypothesis lifecycle                      |
| `investigation.legal.*`        | legal-review state machine                |
| `investigation.seal`           | case sealed                               |
| `storage.promote`              | warm-tier migration triggered             |
| `rules.reload`                 | Sigma rule reload                         |
| `intel.add`                    | new threat-intel indicator                |
| `vault.{init,unlock,lock,...}` | vault operations                          |
| `fleet.register`               | new agent registration                    |
| `webhook.register`             | webhook registration                      |
| `tamper-*`                     | tamper findings (auditctl, journal, ...)  |

The platform itself is the system of record for its own operation —
re-reading `audit.log` reconstructs everything any operator did, in order.
That's what `oblivra-verify` proves for a third party.
