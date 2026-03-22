# OBLIVRA — Future Infrastructure
> Items with zero immediate dependencies. Real, will be built, not now.
> **Last reviewed**: 2026-03-22

---

## Chaos Engineering for Security

**Blocks on**: Stable prod deployment + monitoring baseline

- [ ] Automated certificate expiry simulation
- [ ] Random service crash injection (does detection continue?)
- [ ] Network partition simulation (do agents buffer correctly?)
- [ ] Clock skew injection (do time-based correlations survive?)
- [ ] Storage exhaustion simulation (graceful degradation?)
- [ ] Scheduled chaos runs in CI/staging

---

## Digital Forensics Toolkit (Beyond Current Evidence Locker)

### Memory Forensics
- [ ] Volatility 3 integration (headless analysis)
- [ ] Memory dump acquisition via agent (LiME for Linux, WinPmem for Windows)
- [ ] Automated IOC extraction from memory images
- [ ] Process hollowing / injection detection

### Disk Forensics
- [ ] Timeline generation from filesystem metadata (MFT, journal)
- [ ] Deleted file recovery metadata extraction
- [ ] Browser artifact extraction (history, cookies, cache)
- [ ] Registry hive analysis (Windows)

### Network Forensics
- [ ] PCAP storage and retrieval (linked to alerts)
- [ ] Session reconstruction from captured packets
- [ ] File carving from network streams

---

## Deception Technology (Beyond Current Honeypots)

### Moving Target Defense
- [ ] Randomize internal service ports on schedule
- [ ] Rotate decoy credentials in Active Directory
- [ ] Dynamic honeypot deployment based on threat intel

### Breadcrumb Deployment
- [ ] Plant fake credentials in memory, files, environment variables
- [ ] Monitor access to breadcrumbs as high-fidelity detection signal
- [ ] Auto-generate realistic-looking but detectable decoy data

### Deception Analytics
- [ ] Time-to-interact metrics (how fast do attackers find decoys?)
- [ ] Attacker behavior profiling from deception interactions
- [ ] Deception coverage map

---

## Internationalization (i18n)

**Blocks on**: First non-English-speaking paying customer asking for it

- [ ] String extraction (JSON/YAML)
- [ ] RTL layout support (Arabic/Hebrew)
- [ ] Date/Time/Number formatting per locale
- [ ] Target locales: DE, JA, FR, ES, AR

---

## Graceful Degradation Framework

- [ ] Graduated thresholds: Disk (95%), Memory (90%), CPU (80%)
- [ ] Tiered service shedding: AI Copilot → Enrichment → Ingest → Alerting
- [ ] Partial search results flag in UI
- [ ] Stale data warnings
- [ ] System health degradation banner

---

## Graph Storage Infrastructure

**Blocks on**: Phase 13.2 provenance work

- [ ] Adjacency list storage in BadgerDB
- [ ] Edge types: process→file, user→host, host→host, alert→entity
- [ ] Graph query language (shortest path, N-hop neighbors)
- [ ] TTL-based pruning; importance-based storage budget
- [ ] Automated edge creation from agent/enrichment/detections

---

## Scheduled Task Framework

**Blocks on**: Report Factory (Phase 20.10)

- [ ] Cron & Interval-based scheduling with persistence
- [ ] Concurrency control, priority levels, retry with exponential backoff
- [ ] Job history tracking
- [ ] Execution dashboard (`SchedulerManager.tsx`)

---

## Notification Routing Engine

**Blocks on**: Escalation Chains (currently partial)

- [ ] Condition-based routing (severity, source, time) to multiple channels
- [ ] Additional channels: PagerDuty, Twilio, Discord, Telegram
- [ ] Throttling & Digest modes for low-priority events
- [ ] Channel-specific templates
- [ ] Delivery status tracking + audit logs

---

## Infrastructure-as-Code Deployment

**Blocks on**: v2 stable + first enterprise customer

- [ ] Terraform Provider (manage users, roles, rules, connectors as code)
- [ ] Ansible Collection (server/agent deployment roles)
- [ ] `oblivra export-config` / `import-config` (idempotent, Git-friendly)
- [ ] Configuration versioning with diff viewer + rollback

---

## Platform Configuration Backup

- [ ] Automated daily backup of platform state (users, rules, dashboards)
- [ ] Full and selective restore with conflict resolution
- [ ] Version history for ALL config changes

---

## Mobile On-Call Experience

**Blocks on**: Escalation chains + enough customers to justify mobile dev

- [ ] Mobile-optimized alert detail (single column, large tap targets)
- [ ] One-tap actions: acknowledge, escalate, dismiss, snooze
- [ ] PWA support (web push notifications, "Add to Home Screen")
