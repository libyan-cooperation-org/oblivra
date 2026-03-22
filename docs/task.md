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
**Shared Backend ⚙️**
- [x] **Project Scaffolding**: Standardized Go workspace, Multi-stage Docker builds, Wails v2 configuration
- [x] **Secure Communication**: gRPC with Mutual TLS (mTLS), Self-signed cert rotation, Heartbeat protocol
- [x] **Cross-Platform Bridge**: Unified event types (Go/TS), Frontend binding generation, Theme synchronization

### Phase 1: Storage & Indexing Engine (The Badger-Bleve Core)
**Shared Backend ⚙️**
- [x] **BadgerDB Integration**: Key-value event storage, WAL optimization, Automatic VLog GC
- [x] **Bleve Search Index**: Schema-mapped indexing, Fuzzy search, Highlighted results, Field-scoped queries
- [x] **Partitioning & Lifecycle**: Day-based bucket rotation, Index flattening, Transparent query merging

### Phase 2: Sovereign Query Language (OQL) - Week 1-4 Core
**Shared Backend ⚙️**
- [x] **Parser & Lexer**: Pipe-based syntax (e.g., `source=firewall | stats count by src_ip`)
- [x] **AST & Logical Plan**: Query optimization, Predicate pushdown to BadgerDB
- [x] **Execution Engine**: Parallel partition scanning, Result aggregation, Table/Sort/Head commands

**Desktop 🖥️**
- [x] **OQL Dashboard**: Dedicated desktop-based analytics portal (`OQLDashboard.tsx`)

**Web 🌐**
- [x] **SIEM Search**: Web-based query interface (`SIEMSearch.tsx`)

### Phase 3: Identity & Access Management (Hybrid Auth)
**Shared Backend ⚙️**
- [x] **Fine-Grained RBAC**: Object-level permissions, Service Account tokens, API key management

**Desktop 🖥️**
- [x] **Sovereign Vault**: Desktop-native AES-256-GCM storage with FIDO2/WebAuthn support (`PasswordVault.tsx`)

**Web 🌐**
- [x] **Web Identity**: OIDC (Keycloak/Okta) and SAML 2.0 integration for enterprise deployments (`IdentityAdmin.tsx`)

### Phase 4: Threat Intelligence & Enrichment
**Shared Backend ⚙️**
- [x] **TI Provider Engine**: VirusTotal, AbuseIPDB, Shodan integration with caching
- [x] **GeoIP & ASN Lookup**: MaxMind/IP2Location integration with auto-update
- [x] **Internal Enrichment**: Asset-to-User mapping, Departmental profiling, Host criticality scores

**Desktop 🖥️**
- [x] **Threat Intelligence UI**: Threat graph and tactical intelligence views (`CredentialIntel.tsx`, `PurpleTeam.tsx`)

**Web 🌐**
- [x] **Threat Intel Dashboard**: Web-based threat overview (`ThreatIntelDashboard.tsx`)
- [x] **Enrichment Viewer**: Web-based contextual data viewer (`EnrichmentViewer.tsx`)

### Phase 5: Detection & Alerting Engine (The Brain)
**Shared Backend ⚙️**
- [x] **Rule Processor**: Real-time event matching (YAML/JSON rules), Scheduled query alerts
- [x] **Alert Lifecycle**: Deduplication, Severity scoring, Status tracking (New/Ack/Closed)

**Desktop 🖥️**
- [x] **Alert Dashboard**: Full incident lifecycle and acknowledgment (`AlertDashboard.tsx`)
- [x] **MITRE Heatmap**: Visual coverage map (`MitreHeatmap.tsx`)

**Web 🌐**
- [x] **Alert Management**: Web-based alert queue management (`AlertManagement.tsx`)
- [x] **MITRE Heatmap**: Web-based technique coverage visualization (`MitreHeatmap.tsx`)

### Phase 6: Local & Remote Forensics
**Shared Backend ⚙️**
- [x] **Endpoint Snapshot**: Process tree capture, Connection list, Open files, FIM status

**Desktop 🖥️**
- [x] **Evidence Locker**: Encrypted binary storage for forensic artifacts (pcap, mem dumps) (`EvidenceLocker.tsx`)
- [x] **Timeline Builder**: Unified visualization of interleaved endpoint events (`EvidenceLedger.tsx`)

**Web 🌐**
- [x] **Remote Forensics**: Web-based evidence access and visualization (`RemoteForensics.tsx`)

### Phase 7: Sovereign Agent Framework (Fleet Management)
**Shared Backend ⚙️**
- [x] **Agent Transport**: Encrypted gRPC with certificate pinning and proxy support
- [x] **eBPF Monitoring**: Real-time process exec, Network flow, and File access tracing (Linux)
- [x] **Command & Control**: Remote tasking (Shell exec, File pull, Isolation), Mass deployment

**Desktop 🖥️**
- [x] **Agent Console**: Node registration, health metrics, and direct execution (`AgentConsole.tsx`)

**Web 🌐**
- [x] **Fleet Management**: Centralized web-based agent configuration and status (`FleetManagement.tsx`)

### Phase 8: SOAR & Incident Response (Playbooks)
**Shared Backend ⚙️**
- [x] **Case Management**: Unified investigation workspace, Note taking, Evidence pinning
- [x] **Remediation**: Host isolation, Account lockout, Process termination via Agent

**Desktop 🖥️**
- [x] **Command Center**: Desktop incident response war-room (`CommandCenter.tsx`)
- [x] **Response Replay**: Playbook simulation and execution logs (`ResponseReplay.tsx`)

**Web 🌐**
- [x] **Escalation Center**: Web-based incident triage and workflow (`EscalationCenter.tsx`)
- [x] **Playbook Engine UI**: Visual automation builder (`PlaybookEngineUI.tsx`)

### Phase 9: Ransomware Defense & File Integrity
**Shared Backend ⚙️**
- [x] **Entropy Monitoring**: Real-time detection of high-entropy file writes (Encrypted payloads)
- [x] **Canary Files**: Automated deployment and monitoring of "tripwire" files
- [x] **Snapshot Recovery**: Automated trigger for Shadow Copy / ZFS snapshot on detection

**Desktop 🖥️**
- [x] **Ransomware Dashboard**: Real-time host integrity and lockdown interface (`RansomwareDashboard.tsx`)

**Web 🌐**
- [x] **Ransomware UI**: Web-based centralized status and remediation (`RansomwareUI.tsx`)

### Phase 10: User & Entity Behavior Analytics (UEBA)
**Shared Backend ⚙️**
- [x] **Baseline Engine**: Historical profiling of user/host behavior (Time, Location, Activity)
- [x] **Anomaly Detection**: Isolation Forest / Z-Score alerting on baseline deviations
- [x] **Risk Scoring**: Composite risk score per entity with temporal decay

**Desktop 🖥️**
- [x] **UEBA Panel**: Deep-dive analytics into temporal entity risk (`UEBAPanel.tsx`)

**Web 🌐**
- [x] **UEBA Overview**: Web-based entity analytics and scoring visualization (`UEBAOverview.tsx`)

### Phase 11: Network Detection & Response (NDR)
**Shared Backend ⚙️**
- [x] **Flow Analysis**: NetFlow/IPFIX ingestion, Lateral movement detection, Beaconing detection
- [x] **Encrypted Traffic Analysis (ETA)**: JA3/JA3S fingerprinting, Packet size/timing entropy
- [/] **Protocol Inspection**: Full DPI for DNS, HTTP, SMB, and TLS (SNI/Cert validation)

**Desktop 🖥️**
- [x] **Network Map**: Visualization of host communication and lateral movement (`NetworkMap.tsx`)

**Web 🌐**
- [x] **NDR Overview**: Web-based flow visualization and NDR alerts (`NDROverview.tsx`)

---

> [!NOTE]
> All long-term roadmaps, research, and future capabilities have been moved to:
> - [ROADMAP.md](ROADMAP.md) (Phases 12, 16-26)
> - [RESEARCH.md](RESEARCH.md) (Phase 13)
> - [BUSINESS.md](BUSINESS.md) (Phase 14)
> - [FUTURE.md](FUTURE.md) (Phase 15, Infrastructure, Cross-cutting)