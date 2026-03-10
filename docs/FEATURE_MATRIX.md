# OblivraShell: Sovereign Terminal Feature Matrix

*Generated: 2026-03-10*

This document serves as the master inventory of all capabilities, defensive modules, and operational features built into the **OblivraShell** platform.

---

## 1. Core Platform & Terminal

| Feature | Description | Status |
|---|---|---|
| **Multi-Monitor SOC Workspace** | Golden Layout powered drag-and-drop workspace supporting multi-monitor pop-outs with native reactive styling. | Deployed |
| **Tactical Dock** | Window minimization dock for stashing non-critical alert feeds and terminals without destroying their state. | Deployed |
| **Elite SSH Client** | High-performance Go-native SSH client bypassing the OS. Supports PEM, Password, and Agent Auth. | Deployed |
| **Local PTY Emulation** | Real Unix pseudo-terminal (PTY) emulation for local OS interactions with raw mode support. | Deployed |
| **Terminal Output Batcher** | Debounces high-velocity terminal output (`cat large.log`) to prevent IPC bridge flooding. | Deployed |
| **Zstandard Payload Compression** | Compresses IPC payloads over the bridge to maintain UI reactivity under stress. | Deployed |
| **SSH Tunneling Manager** | Visual manager for Local, Remote, and Dynamic (SOCKS5) port forwarding. | Deployed |
| **Secure File Browser (SFTP)** | Visual file explorer operating over the active SSH/Local channel with path traversal protections. | Deployed |
| **Unified Settings Panel** | Brutalist tactical UI for managing global configuration, themes, and trusted hardware. | Deployed |
| **Command Palette** | Spotlight-style global search for hosts, alerts, and settings (`Ctrl+Shift+P`). | Deployed |

---

## 2. Cryptographic Vault & Identity

| Feature | Description | Status |
|---|---|---|
| **Sovereign AES-256-GCM Vault** | Completely offline credential manager resistant to quantum extraction (time/mem hardened Argon2id). | Deployed |
| **Memory Zeroization** | Deterministic RAM wiping (`VirtualLock`/`ZeroSlice`) of decrypted payloads to defeat memory scrapers. | Deployed |
| **Hardware Trust Anchoring** | TPM 2.0 PCR binding. Vault will refuse to decrypt if the OS boot sequence was altered. | Deployed |
| **TPM Identity Provider** | Abstracted hardware identity system for attaching physical trust to internal platform actions. | Deployed |
| **FIDO2 / YubiKey Support** | Hardware MFA required for privileged operations and Vault access. | Deployed |
| **Role-Based Access Control (RBAC)** | Strictly typed identity system (Admin, Analyst, Agent) enforcing least-privilege on all APIs. | Deployed |
| **OIDC / SAML 2.0 Identity** | Enterprise SSO federation with local metadata overrides for offline environments. | Deployed |

---

## 3. Threat Detection & SIEM (Phase 1-4)

| Feature | Description | Status |
|---|---|---|
| **High-Velocity Ingestion Pipeline** | Syslog, JSON, CEF, and LEEF ingestion handling 5,000+ EPS without backpressure failure. | Deployed |
| **BadgerDB + Bleve Search** | Rust-grade performance using BadgerDB for hot storage and Go-native Lucene (Bleve) for indexing. | Deployed |
| **YAML Detection Rule Engine** | Supports Threshold, Frequency, Sequence, and Correlation based detection strategies. | Deployed |
| **MITRE ATT&CK Mapper** | Maps all detections back to MITRE TTPs and powers a visual Coverage Heatmap. | Deployed |
| **Live Network Map (NDR)** | Visualizes NetFlow, Lateral Movement, and internal connection matrices. | Deployed |
| **Alert Escelation Dashboard** | Triage interface with incident grouping, deduplication buffers, and workflow states. | Deployed |
| **Threat Intel Enrichment** | Local offline STIX/TAXII bundle ingestion and MaxMind GeoIP tagging. | Deployed |

---

## 4. Endpoint Agent Framework (Phase 7)

| Feature | Description | Status |
|---|---|---|
| **Go Native Agent** | Lightweight endpoint agent with file tailing, Windows Event logging, and system metric streams. | Deployed |
| **eBPF Tracing (Linux)** | Kernel-level hooks tracking process execution (`execve`) and network connections. | Deployed |
| **Identity-Bound TLS** | All agents authenticate back to the commander using cryptographically signed client credentials. | Deployed |

---

## 5. Sovereign Security Infrastructure (Phase 9 & 10)

| Feature | Description | Status |
|---|---|---|
| **Nuclear Data Destruction** | `CryptoWipe` capability that algorithmically destroys the Vault and database files during a breach. | Deployed |
| **Air-Gap Kill Switch** | Instantly drops all inbound/outbound listeners, restricting the platform to forensic-only access. | Deployed |
| **Runtime Binary Attestation** | Platform refuses to boot if its executable hash doesn't match the signed release manifest. | Deployed |
| **Merkle Tree Audit Ledger** | Immutable, blockchain-style crypto ledger ensuring forensic data cannot be retroactively altered. | Deployed |
| **Immutable Compliance Reporting** | Generates PDF reports mapping current system state against PCI-DSS, SOC2, and ISO 27001. | Deployed |
| **System Trust Consensus Monitor** | Real-time Dashboard correlating TPM health, Attestation state, and Evidence Ledger integrity. | Deployed |
| **Temporal Integrity Service** | Detects clock drift, event time manipulation, and forensic timeline tampering. | Deployed |
| **Offline Update Bundles** | Processes USB-transferred, cryptographically signed `.oblivra-update` bundles for Air-Gap patches. | Deployed |

---

## 6. Advanced Operations & AI

| Feature | Description | Status |
|---|---|---|
| **Multi-Execution Broadcast** | Commands typed in a primary terminal broadcast securely to an N-node SSH fleet simultaneously. | Deployed |
| **Session Recording & Playback** | DVR-style capture of exact terminal output bytes for post-incident review. | Deployed |
| **User Entity Behavior Analytics (UEBA)** | Isolation Forest ML modeling building baselines of user activities to flag anomalies. | Deployed |
| **Graph Exploratory Threat Hunting** | Multi-hop inference engine mapping how users, nodes, and sessions intersect in an interactive UI. | Deployed |
| **Red Team Attack Simulator** | Platform self-tests its own detection rules by actively simulating MITRE steps. | Deployed |
| **Deterministic Response Replay** | Mathematical proofing module ensuring security rules execute identically regardless of transient state. | Deployed |
| **Local AI Companion** | In-console AI assistant providing terminal command generation and complex error translation. | Deployed |
