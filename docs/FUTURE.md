# OBLIVRA — FUTURE & CROSS-CUTTING CAPABILITIES

## Phase 15: Deception & Honeytokens (Active Defense)
- [ ] **Honey-Credential Engine**: Automated planting of fake credentials in `SecureVault` and `SovereignTerminal` connection lists
- [ ] **Decoy Agent**: A "silent" agent that reports access to unused files/registry keys as high-fidelity alerts
- [ ] **Dynamic DNS Deception**: Redirecting suspected lateral movement to specialized "sinkhole" services
- [ ] **Breadcrumb Generator**: Auto-planting `config.json` files with decoy API keys across the network

## Infrastructure: Missing Cross-Cutting Capabilities

### Graph Storage Infrastructure 🏗️ [Hybrid/Both]
- [ ] **Embedded Graph Engine**
    - [ ] Adjacency list storage; Ad-hoc relationship mapping
    - [ ] Edge types: process→file, user→host, host→host, alert→entity
    - [ ] Graph query language (shortest path, N-hop neighbors)
- [ ] **Graph Ingestion & Lifecycle**
    - [ ] Automated edge creation from agent/enrichment/detections
    - [ ] TTL-based pruning; Importance-based storage budget management

### Scheduled Task Framework
- [ ] **Job Scheduler Engine**
    - [ ] Cron & Interval-based scheduling with persistence
    - [ ] Concurrency control; Priority levels; Retry with exponential backoff
- [ ] **Job Management**
    - [ ] Job history tracking; Execution dashboard (`SchedulerManager.tsx`)

### Notification Routing Engine
- [ ] **Unified Routing Engine**
    - [ ] Condition-based routing (severity, source, time) to multiple channels
    - [ ] Channel support: Email, Slack, Teams, PagerDuty, Twilio, Webhook
    - [ ] Throttling & Digest modes for low-priority events
- [ ] **Templates & Delivery**
    - [ ] Channel-specific templates; Delivery status tracking; Audit logs

### Capacity Planning Framework
- [ ] **Sizing Calculator** (`docs/sizing.md` + tool)
    - [ ] EPS/Retention mapping to cores/RAM/disk (Small/Medium/Large/XL)
    - [ ] Runtime Capacity Monitoring: Alert at 80/90/95% thresholds
    - [ ] Migration Sizing: Splunk/Elastic conversion logic

### Chaos Engineering for Security
- [ ] **Security Chaos Testing Framework**
    - [ ] Automated certificate expiry simulation
    - [ ] Random service crash injection (does detection continue?)
    - [ ] Network partition simulation (do agents buffer correctly?)
    - [ ] Clock skew injection (do time-based correlations survive?)
    - [ ] Storage exhaustion simulation (graceful degradation?)
    - [ ] Scheduled chaos runs in CI/staging

### Digital Forensics Toolkit (Beyond Current Evidence Locker) 🏗️ [Hybrid/Both]
- [ ] **Memory Forensics**
    - [ ] Volatility 3 integration (headless analysis)
    - [ ] Memory dump acquisition via agent (LiME for Linux, WinPmem for Windows)
    - [ ] Automated IOC extraction from memory images
    - [ ] Process hollowing / injection detection
- [ ] **Disk Forensics**
    - [ ] Timeline generation from filesystem metadata (MFT, journal)
    - [ ] Deleted file recovery metadata extraction
    - [ ] Browser artifact extraction (history, cookies, cache)
    - [ ] Registry hive analysis (Windows)
- [ ] **Network Forensics**
    - [ ] PCAP storage and retrieval (linked to alerts)
    - [ ] Session reconstruction from captured packets
    - [ ] File carving from network streams

### Deception Technology (Beyond Honeypots)
- [ ] **Moving Target Defense**
    - [ ] Randomize internal service ports on schedule
    - [ ] Rotate decoy credentials in Active Directory
    - [ ] Dynamic honeypot deployment based on threat intel
- [ ] **Breadcrumb Deployment**
    - [ ] Plant fake credentials in memory, files, environment variables
    - [ ] Monitor access to breadcrumbs as high-fidelity detection signal
    - [ ] Auto-generate realistic-looking but detectable decoy data
- [ ] **Deception Analytics**
    - [ ] Time-to-interact metrics (how fast do attackers find decoys?)
    - [ ] Attacker behavior profiling from deception interactions
    - [ ] Deception coverage map (what % of network has active deception?)

### Internationalization & Globalization (i18n)
- [ ] **Localization Framework**
    - [ ] Extracted strings (JSON/YAML); RTL layout support (Arabic/Hebrew)
    - [ ] Date/Time/Number formatting per locale; Localized API error messages
- [ ] **Target Locales**
    - [ ] base: English; l10n: German, Japanese, French, Spanish, Arabic

### Graceful Degradation Framework
- [ ] **Resource-Aware Throttling**
    - [ ] Graduated thresholds: Disk (95%), Memory (90%), CPU (80%)
    - [ ] Tiered service shedding: Tier 4 (Copilot) → Tier 2 (Ingest) → Tier 1 (Alerting)
- [ ] **Resilience UX**
    - [ ] Partial search results flag; Stale data warnings; System health banner

---

## Final Audit: Operational, Commercial & Core Substrate

### Section 1: Product Experience & Accessibility (Cross-Cutting)
[... CONTENT FROM SECTION 1-5 AS PER SOURCE ...]
