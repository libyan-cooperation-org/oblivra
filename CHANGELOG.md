# Changelog

All notable changes to the OblivraShell Sovereign Terminal will be documented in this file.

## [1.0.0] - 2026-03-10

### 🚀 Major Strategic Release
*This marks the first production-grade sovereign release containing all Phase 1-10 architectural requirements.*

### Core Defensive Capabilities Additions
*   **Cryptographic Vault (`v1.0.0`)**: Implemented AES-256-GCM hardware-bound vault with Argon2id Key Derivation and OS memory zeroization protections (defeats memory scrapers).
*   **Embedded SIEM (`v1.0.0`)**: Built a full-scale Go-native SIEM pipeline using BadgerDB and Bleve, capable of 5,000+ Events Per Second with local Lucene search.
*   **eBPF Agent Framework (`v1.0.0`)**: Added cross-platform telemetry agents with Linux eBPF telemetry hooks for Zero-Trust process monitoring.

### Front-End Desktop Experience Additions
*   **Golden Layout SOC Workspace**: Replaced static tabs with an interactive multi-monitor, draggable pop-out window engine for forensic dashboards.
*   **Elite SSH Client**: Go-native connection manager supporting multi-exec broadcasting, dynamic SOCKS5 tunnels, and real-time SFTP explorers without external dependencies.
*   **Sovereign UI Overhaul**: Upgraded from standard Tailwind to a high-contrast Brutalist tactical aesthetic engineered for low-light SOC environments.

### Enterprise Scale Additions
*   **Raft Clustering**: Built a Multi-Node HA consensus engine for database replication across distributed instances.
*   **Role-Based Access (RBAC)**: Added granular authorization controls linked to FIDO2 YubiKey identity verifications.
*   **SIEM Threat Engine**: Implemented offline IOC loading via STIX/TAXII and a multi-hop Security Graph Query engine.

### Forensics & Hardening Additions
*   **Cryptographic Integrity Checks**: Added runtime `/debug/attestation` binary hashing, Merkle Tree evidence ledgers, and Temporal Drift monitors.
*   **Disaster Scenarios**: Wired in emergency `Kill-Switch` and `Nuclear Wipe` functionality to instantly sever C2 and scrub memory during active breaches.
*   **Optimizations**: Repaired `Wails` IPC bridge flooding via `OutputBatcher` and Zstandard Payload Compression over websockets. Fixed massive JSON unmarshal leaks and DB contention bottlenecks.

---

*For detailed architectural mapping, view `docs/FEATURE_MATRIX.md`.*
