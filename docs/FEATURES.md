# OBLIVRA — Sovereign Security Platform Feature List

This document provides a comprehensive list of all features currently implemented or planned for OBLIVRA.

---

## 🖥️ Core Platform
**The foundational layer for daily operations, providing high-performance terminal access and secure asset management.**

### Terminal & SSH
- **Multi-Session Tab Bar**: Local (green) and SSH (orange) sessions with instant switching.
- **PTY Management**: Local PTY terminal support with full terminal emulation.
- **Connection Pooling**: High-performance SSH connection pooling for instant access.
- **Multi-Exec Broadcast**: Execute a single command across an entire fleet simultaneously.
- **SFTP File Browser**: Integrated file transfer management and browser.
- **Session Recording**: TTY recording and playback for audit and training.
- **Split Panes**: Terminal grid with customizable split panes for multi-tasking.
- **SSH Tunneling**: Integrated port forwarding and SSH tunnel management.

### Security & Vault
- **AES-256 Encrypted Vault**: Cryptographic storage for credentials and keys.
- **OS Keychain Integration**: Native integration with Windows/macOS/Linux keychains.
- **Multi-Factor Auth**: FIDO2 and YubiKey support for vault unlocking.
- **TLS Certificate Generator**: Built-in CA and certificate issuance for internal services.
- **Snippet Vault**: Secure command library for re-usable tactical snippets.

### Productivity & Collaboration
- **Notes & Runbooks**: Integrated markdown editor for operational documentation.
- **Workspace Manager**: Persistent workspace states and session groups.
- **AI Assistant**: Context-aware CLI assistance and error explanation.
- **Theme Engine**: Highly customizable tactical UI with pre-built dark/brutalist themes.
- **Command Palette**: Global quick-switcher for tools, hosts, and settings.
- **Team Sync**: Collaboration service for sharing host configs and snippets with teams.

---

## 🛡️ Embedded SIEM
**High-velocity log ingestion and real-time detection engine running entirely on-device.**

### Ingestion & Storage
- **18,000+ EPS Pipeline**: Validated 18,000 EPS burst / 10,000 EPS sustained with crash-safe WAL and zero data loss on restart.
- **Unified Ingest**: Support for Syslog (RFC 5424/3164), JSON, CEF, and LEEF.
- **State-of-the-Art Storage**: BadgerDB (hot store) + Bleve (full-text index) for sub-second search.
- **Columnar Archival**: Parquet-based long-term storage for forensic data.

### Search & Analytics
- **Multi-Syntax Query**: Support for LogQL, Lucene, SQL, and Osquery.
- **Field-Level Indexing**: Aggregations, facets, and histograms on all indexed fields.
- **Live Tail**: Real-time event streaming with sub-100ms latency.
- **Fleet Heatmaps**: High-density visualisations of infrastructure status and log volume.

### Alerting & Detection
- **Sigma Native Engine**: Directly execute community `.yml` Sigma rules.
- **Advanced Rule Types**: Threshold, Frequency, Sequence, and Correlation rules.
- **Incident Aggregation**: Automatic grouping of alerts into manageable incidents.
- **Notification Routing**: Multi-channel delivery (Slack, Email, Teams, Webhooks).

---

## 🛰️ Advanced Security & Forensics
**Capabilities designed for high-stakes environments requiring deep visibility and regulatory compliance.**

### Threat Intelligence & Enrichment
- **STIX/TAXII Client**: Automated ingestion of threat intelligence feeds.
- **Matching Engine**: O(1) IP and Hash lookups against known-bad indicators.
- **Enrichment Pipeline**: GeoIP, DNS PTR/ASN, and Asset mapping for every event.
- **Windows/Linux Forensics**: Native parsers for Event Logs and Journald.

### Compliance & Integrity
- **Merkle Tree Logging**: Cryptographically immutable audit trails.
- **Chain of Custody**: Evidence locker with baseline diffing and signing.
- **Compliance Packs**: Pre-configured rules for PCI-DSS, NIST, SOC2, and GDPR.
- **Policy Evaluator**: Continuous evaluation of infrastructure against compliance controls.

### Specialized Defense
- **Ransomware Defense**: Entropy-based behavioral detection and automated network isolation.
- **UEBA Engine**: Peer-group baselining and Isolation Forest anomaly scoring.
- **NDR (Network Detection)**: NetFlow/IPFIX collector and lateral movement detection.
- **eBPF Agent**: High-fidelity Linux kernel instrumentation for processes and network.

---

## 🚀 Infrastructure & Roadmap
**Hardening and enterprise-grade scaling capabilities.**

### Platform Hardening
- **Zero Internet Dependency**: Fully functional in air-gapped environments.
- **Signed Releases**: Cosign/Sigstore verification for all binary artifacts.
- **SBOM Generation**: Full transparency into the software supply chain (SPDX/CycloneDX).
- **Self-Observability**: Integrated Prometheus metrics and pprof profiling.

### Strategic Roadmap
- **SovereignQL (OQL)**: Custom pipe-based query language for tactical analytics.
- **ITDR (Identity Threat Detection)**: AD attack detection (DCSync, Kerberoasting) and path analysis.
- **AI/LLM Security**: Monitoring for prompt injection and shadow AI usage.
- **SOAR Gateway**: Automated response playbooks and case management orchestration.
- **Multi-Region Architecture**: Geo-distributed ingestion and global query correlation.
- **Autonomous Detection Validation**: Built-in adversary emulation for continuous coverage testing.

---

---

## 🏗️ Platform Excellence & Commercial Readiness
**Operational and commercial substrate required for enterprise-grade deployment and legal procurement.**

### Product Experience & Accessibility
- **Guided Onboarding**: First-run setup wizard for admin, network, and source configuration.
- **WCAG 2.1 AA Compliance**: Full accessibility support including screen readers, keyboard navigation, and colorblind-safe palettes.
- **Analyst Documentation**: Comprehensive, versioned guides for investigation, OQL, and administration.
- **OpenAPI 3.0 Standard**: Machine-readable API contracts with auto-generated SDKs (Go/Python).
- **Mobile On-Call View**: Responsive web-app for alert acknowledgement and triage on the move.

### Enterprise Deployment & Security
- **IaC Deployment**: Official Terraform Providers and Ansible Collections for "Platform-as-Code".
- **Platform Integrity**: Self-monitoring, binary hash verification, and immutable audit of admin actions.
- **Configuration Versioning**: Git-friendly export/import and full rollback for platform state.
- **Temporal Event Handling**: Advanced logic for late-arriving events and out-of-order logs.
- **Compliance Artifacts**: Pre-built legal templates (DPA, BAA, SCCs) and compatibility matrices (Chrome/Firefox ESR/RHEL).
- **Diagnostic Bundle**: One-command support bundle generation for rapid troubleshooting.

---

## 🖥️ Desktop vs. 🌐 Browser Architecture
**OBLIVRA as a unified codebase serving two distinct deployment models.**

| Feature | Desktop Power Tool (Wails) | Enterprise Browser Platform |
| :--- | :--- | :--- |
| **Primary Goal** | Individual tactical engineer tool | Collective SOC platform / fleet-scale |
| **OS Access** | Native PTY, OS Keychain, Local FS | Server-mediated / Web-proxy |
| **Authentication** | Local Vault / Biometric / YubiKey | OIDC / SAML / RBAC / Multi-tenant |
| **SIEM Scale** | Embedded BadgerDB (Local scope) | Clustered index (Enterprise scope) |
| **Hybrid Mode** | ✅ Native performance + Server data | ❌ Always server-mediated |
| **Exclusives** | Terminal Grid, SSH Agent Forwarding | Fleet Mgmt, SOC Metrics, Collaboration |
