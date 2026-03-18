# OBLIVRA — Master Task Tracker (90-Day Plan)

## Current Inventory (Platform Coverage)

| Component | Desktop 🖥️ | Web 🌐 |
| :--- | :--- | :--- |
| **Authentication** | Vault (Local-first) | OIDC / SAML / Fleet |
| **Search Engine** | OQL (Local) | OQL (Federated) |
| **Ingestion** | Syslog/Agent/Files | Cloud / SaaS / Multi-region |
| **Forensics** | Local Disk / Memory | Remote Snapshot / Timeline |
| **Terminal** | Full PTY / SSH / Vault | Web Terminal (Restricted) |

---

### Phase 0: Core Foundation & Hybrid Substrate
- [x] **Project Scaffolding**: Standardized Go workspace, Multi-stage Docker builds, Wails v2 configuration
- [x] **Secure Communication**: gRPC with Mutual TLS (mTLS), Self-signed cert rotation, Heartbeat protocol
- [x] **Cross-Platform Bridge**: Unified event types (Go/TS), Frontend binding generation, Theme synchronization

### Phase 1: Storage & Indexing Engine (The Badger-Bleve Core)
- [x] **BadgerDB Integration**: Key-value event storage, WAL optimization, Automatic VLog GC
- [x] **Bleve Search Index**: Schema-mapped indexing, Fuzzy search, Highlighted results, Field-scoped queries
- [x] **Partitioning & Lifecycle**: Day-based bucket rotation, Index flattening, Transparent query merging

### Phase 2: Sovereign Query Language (OQL) - Week 1-4 Core
- [x] **Parser & Lexer**: Pipe-based syntax (e.g., `source=firewall | stats count by src_ip`)
- [x] **AST & Logical Plan**: Query optimization, Predicate pushdown to BadgerDB
- [x] **Execution Engine**: Parallel partition scanning, Result aggregation, Table/Sort/Head commands
- [/] **UI Integration**: `QueryEditor.tsx` with syntax highlighting and Intellisense

### Phase 3: Identity & Access Management (Hybrid Auth)
- [x] **Sovereign Vault**: Desktop-native AES-256-GCM storage with FIDO2/WebAuthn support
- [x] **Web Identity**: OIDC (Keycloak/Okta) and SAML 2.0 integration for enterprise deployments
- [x] **Fine-Grained RBAC**: Object-level permissions, Service Account tokens, API key management

### Phase 4: Threat Intelligence & Enrichment
- [x] **TI Provider Engine**: VirusTotal, AbuseIPDB, Shodan integration with caching
- [x] **GeoIP & ASN Lookup**: MaxMind/IP2Location integration with auto-update
- [x] **Internal Enrichment**: Asset-to-User mapping, Departmental profiling, Host criticality scores

### Phase 5: Detection & Alerting Engine (The Brain)
- [x] **Rule Processor**: Real-time event matching (YAML/JSON rules), Scheduled query alerts
- [x] **MITRE ATT&CK Mapping**: Tactic/Technique tagging, Coverage heatmap generation
- [x] **Alert Lifecycle**: Deduplication, Severity scoring, Status tracking (New/Ack/Closed)

### Phase 6: Local & Remote Forensics
- [x] **Endpoint Snapshot**: Process tree capture, Connection list, Open files, FIM status
- [x] **Evidence Locker**: Encrypted binary storage for forensic artifacts (pcap, mem dumps)
- [x] **Timeline Builder**: Unified visualization of interleaved endpoint events

### Phase 7: Sovereign Agent Framework (Fleet Management)
- [x] **Agent Transport**: Encrypted gRPC with certificate pinning and proxy support
- [x] **eBPF Monitoring**: Real-time process exec, Network flow, and File access tracing (Linux)
- [x] **Command & Control**: Remote tasking (Shell exec, File pull, Isolation), Mass deployment

### Phase 8: SOAR & Incident Response (Playbooks)
- [x] **Case Management**: Unified investigation workspace, Note taking, Evidence pinning
- [x] **Playbook Engine**: Visual automation builder, Integration with Jira/ServiceNow/Slack
- [x] **Remediation**: Host isolation, Account lockout, Process termination via Agent

### Phase 9: Ransomware Defense & File Integrity
- [x] **Entropy Monitoring**: Real-time detection of high-entropy file writes (Encrypted payloads)
- [x] **Canary Files**: Automated deployment and monitoring of "tripwire" files
- [x] **Snapshot Recovery**: Automated trigger for Shadow Copy / ZFS snapshot on detection

### Phase 10: User & Entity Behavior Analytics (UEBA)
- [x] **Baseline Engine**: Historical profiling of user/host behavior (Time, Location, Activity)
- [x] **Anomaly Detection**: Isolation Forest / Z-Score alerting on baseline deviations
- [x] **Risk Scoring**: Composite risk score per entity with temporal decay

### Phase 11: Network Detection & Response (NDR)
- [x] **Flow Analysis**: NetFlow/IPFIX ingestion, Lateral movement detection, Beaconing detection
- [x] **Encrypted Traffic Analysis (ETA)**: JA3/JA3S fingerprinting, Packet size/timing entropy
- [/] **Protocol Inspection**: Full DPI for DNS, HTTP, SMB, and TLS (SNI/Cert validation)

---

> [!NOTE]
> All long-term roadmaps, research, and future capabilities have been moved to:
> - [ROADMAP.md](ROADMAP.md) (Phases 12, 16-26)
> - [RESEARCH.md](RESEARCH.md) (Phase 13)
> - [BUSINESS.md](BUSINESS.md) (Phase 14)
> - [FUTURE.md](FUTURE.md) (Phase 15, Infrastructure, Cross-cutting)
