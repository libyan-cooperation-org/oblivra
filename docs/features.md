# OBLIVRA — Complete Feature Manifest

> **Sovereign Terminal** — Air-gapped, offline-first security operations platform
> 48 backend services · 46 internal packages · 227 Go files · 109 TSX components · 32 routes
> **Generated:** 2026-03-04

---

## 1. Terminal & SSH

| Feature | Backend | Frontend |
|---|---|---|
| SSH client (key/password/agent auth) | `internal/ssh/client.go`, `auth.go` | — |
| Local PTY terminal | `local_service.go` | `TerminalLayout.tsx` |
| SSH connection pooling | `internal/ssh/pool.go` | — |
| SSH config parser + bulk import | `internal/ssh/config_parser.go` | — |
| SSH tunneling / port forwarding | `internal/ssh/tunnel.go`, `tunnel_service.go` | Tunnels panel |
| Session recording & playback | `recording_service.go` | `RecordingPanel.tsx` |
| Session sharing & broadcast | `broadcast_service.go`, `share_service.go` | — |
| Multi-exec concurrent commands | `multiexec_service.go` | Multi-exec panel |
| Terminal grid with split panes | — | `terminal/` components |
| File browser & SFTP transfers | `file_service.go`, `transfer_manager.go` | `FileBrowser.tsx` |

---

## 2. Security & Vault

| Feature | Backend | Frontend |
|---|---|---|
| AES-256-GCM encrypted Vault | `internal/vault/vault.go`, `crypto.go` | — |
| OS keychain integration | `internal/vault/keychain.go` | — |
| FIDO2 / YubiKey support | `internal/security/fido2.go`, `yubikey.go` | `SecurityKey.tsx` |
| TLS certificate generation | `internal/ssh/certificate.go` | — |
| Snippet vault / command library | `snippet_service.go` | Snippets panel |
| Ransomware detection (entropy-based) | `internal/security/ransomware.go` | `RansomwareDashboard.tsx` |
| Canary file deployment | `internal/security/canary.go` | — |
| Honeypot infrastructure | `internal/security/honeypot.go` | — |
| File integrity monitoring (FIM) | `sentinel.go` | — |
| Hardening module | `hardening.go` | — |

---

## 3. Identity & Access Management

| Feature | Backend | Frontend |
|---|---|---|
| Unified User model with bcrypt auth | `internal/database/users.go` | `UsersPanel.tsx` |
| Role-based access control (RBAC) | `internal/auth/rbac.go` | Role cards |
| OIDC/OAuth2 SSO | `internal/auth/oidc.go` | — |
| SAML 2.0 Service Provider | `internal/auth/saml.go` | — |
| TOTP MFA (QR code generation) | `internal/auth/mfa.go` | — |
| API key authentication | `internal/auth/apikey.go` | — |
| Identity Service (login, MFA, CRUD) | `identity_service.go` | `/identity` route |

---

## 4. SIEM & Detection

| Feature | Backend | Frontend |
|---|---|---|
| Syslog ingestion (RFC 5424/3164) | `internal/ingest/syslog.go` | — |
| JSON / CEF / LEEF parsers | `internal/ingest/parsers/` | — |
| Windows Event Log parser | `parsers/windows.go` | — |
| Linux syslog + journald parser | `parsers/linux.go` | — |
| Cloud audit (AWS/Azure/GCP) | `parsers/cloud_aws.go`, etc. | — |
| Network logs (NetFlow, DNS, FW) | `parsers/network.go` | — |
| Schema-on-read normalization | `internal/ingest/pipeline.go` | — |
| Backpressure + rate limiting | `internal/ingest/pipeline.go` | — |
| Write-Ahead Log (WAL) | `internal/ingest/wal.go` | — |
| BadgerDB hot storage | `internal/storage/badger.go` | — |
| Bleve full-text search indexing | `internal/search/bleve.go` | — |
| Parquet cold archival (ZSTD) | `internal/analytics/archiver.go` | — |
| Lucene-style query parser | `internal/search/transpiler.go` | — |
| Field-level indexing | Bleve field mappings | — |
| Aggregations (facets, histograms) | `internal/search/` | — |
| Saved searches | DB model + API | SIEM panel |
| YAML detection rule engine | `internal/detection/` | — |
| Threshold / frequency / sequence / correlation rules | `internal/detection/rules/` | — |
| MITRE ATT&CK technique mapper | `internal/detection/mitre/` | `MitreHeatmap.tsx` |
| Alert deduplication | Configurable windows | — |
| Regex timeout / ReDoS prevention | Safe parsing | — |
| Real-time streaming search | — | `SIEMPanel.tsx` |

---

## 5. Alerting & Notifications

| Feature | Backend | Frontend |
|---|---|---|
| Webhook notifications | `internal/notifications/` | — |
| Email notifications | `internal/notifications/` | — |
| Slack notifications | `internal/notifications/` | — |
| Microsoft Teams notifications | `internal/notifications/` | — |
| Alert dashboard (filter, ack, status) | — | `AlertDashboard.tsx` |

---

## 6. Incident Response & SOAR

| Feature | Backend | Frontend |
|---|---|---|
| Case management (CRUD, assignment, timeline) | `incident_service.go` | `CommandCenter.tsx` |
| Semi-automated response actions | `internal/incident/actions.go` | — |
| No-code playbook builder | `playbook_service.go` | — |
| Jira integration | `internal/incident/jira.go` | — |
| ServiceNow integration | `internal/incident/servicenow.go` | — |
| Automated network isolation | Agent-side | — |
| Campaign intelligence correlation | `internal/incident/` | — |

---

## 7. Threat Intelligence & Enrichment

| Feature | Backend | Frontend |
|---|---|---|
| STIX/TAXII Client | `internal/threatintel/taxii.go` | — |
| Offline rule ingestion (JSON, OpenIOC) | `internal/threatintel/` | — |
| IOC Match Engine (O(1) lookups) | `internal/threatintel/` | `ThreatIntelPanel.tsx` |
| GeoIP enrichment (MaxMind) | `internal/enrich/geoip.go` | — |
| DNS enrichment (ASN, PTR) | `internal/enrich/dns.go` | — |
| Asset/User mapping | `internal/enrich/` | — |
| Enrichment pipeline orchestrator | `internal/enrich/pipeline.go` | — |

---

## 8. UEBA & Machine Learning

| Feature | Backend | Frontend |
|---|---|---|
| Per-user/entity behavioral baselines | `internal/ueba/baseline.go` | — |
| Isolation Forest anomaly detection | `internal/ueba/anomaly.go` | — |
| Identity Threat Detection (ITDR) | `internal/ueba/itdr.go` | — |
| Peer group model | `internal/ueba/peergroup.go` | — |
| Threat hunting interface | — | `ThreatHunter.tsx` |
| UEBA risk scoring panel | — | `UEBAPanel.tsx` |

---

## 9. Network Detection & Response (NDR)

| Feature | Backend | Frontend |
|---|---|---|
| NetFlow/IPFIX collector | `internal/ndr/` | — |
| DNS log analysis (DGA, tunneling) | `internal/ndr/` | — |
| TLS metadata (JA3/JA3S) | `internal/ndr/` | — |
| HTTP proxy log parser | `internal/ndr/` | — |
| Lateral movement detection | `internal/ndr/lateral.go` | — |
| Network map visualization | — | `NetworkMap.tsx` |

---

## 10. Forensics & Compliance

| Feature | Backend | Frontend |
|---|---|---|
| Merkle tree immutable audit logging | `internal/integrity/merkle.go` | — |
| Evidence locker (chain of custody) | `internal/forensics/evidence.go` | `EvidenceLocker.tsx` |
| Enhanced FIM with baseline diffing | `sentinel.go` | — |
| PCI-DSS compliance pack | YAML rules | — |
| NIST 800-53 compliance pack | YAML rules | — |
| ISO 27001 compliance pack | YAML rules | — |
| GDPR compliance pack | YAML rules | — |
| HIPAA compliance pack | YAML rules | — |
| SOC 2 Type II compliance pack | YAML rules | — |
| PDF/HTML report generator | `internal/compliance/report.go` | — |
| Compliance evaluator engine | `internal/compliance/evaluator.go` | `ComplianceCenter.tsx` |
| Governance dashboard | — | `GovernanceDashboard.tsx` |

---

## 11. Agent Framework

| Feature | Backend | Frontend |
|---|---|---|
| Agent binary | `cmd/agent/main.go` | — |
| File tailing collector | Agent module | — |
| Windows Event Log collector | Agent module | — |
| System metrics collector | Agent module | — |
| FIM collector | Agent module | — |
| gRPC/TLS/mTLS transport | `internal/agent/` | — |
| Zstd compression | Transport layer | — |
| Offline buffering (local WAL) | Agent-side | — |
| Edge filtering + PII redaction | Agent-side | — |
| Agent registration + heartbeat | API endpoint | `AgentConsole.tsx` |
| Fleet-wide config push | `agent_service.go` | — |
| eBPF probes (Linux: exec, net, file) | `internal/agent/ebpf/` | — |

---

## 12. Enterprise Features

| Feature | Backend | Frontend |
|---|---|---|
| Multi-tenancy (data partitioning) | All repos + migration v11 | — |
| HA clustering (Raft consensus) | `internal/cluster/` | — |
| RBAC + SAML/OIDC + MFA | `internal/auth/` | `UsersPanel.tsx` |
| Data lifecycle management | `lifecycle_service.go` | — |
| Executive dashboard | — | `ExecutiveDashboard.tsx` |

---

## 13. Ops & Monitoring

| Feature | Backend | Frontend |
|---|---|---|
| Unified Ops Center (LogQL, Lucene, SQL, Osquery) | — | `OpsCenter.tsx` |
| Splunk-style analytics dashboard | — | `SplunkDashboard.tsx` |
| Customizable widget dashboard | — | `Dashboard.tsx` |
| Network discovery | `discovery_service.go` | — |
| Global topology visualization | — | `GlobalTopology.tsx` |
| Bandwidth monitor | — | `BandwidthMonitor.tsx` |
| Fleet heatmap | — | `FleetHeatmap.tsx` |
| Osquery integration | `internal/osquery/` | — |
| Log source manager | `logsource_service.go` | — |
| Health & metrics endpoints | `health_service.go`, `metrics_service.go` | — |
| Telemetry worker | `telemetry_service.go` | — |
| Self-monitoring dashboard | — | `SelfMonitor.tsx` |
| Prometheus `/metrics` endpoint | `internal/api/rest.go` | — |
| Liveness + readiness probes | `/healthz`, `/readyz` | — |

---

## 14. Governance & Policy

| Feature | Backend | Frontend |
|---|---|---|
| Policy Decision Service | `internal/policy/engine.go` | — |
| Feature flag framework | `internal/policy/` | — |
| Tier-based capability isolation | Free/Pro/Enterprise/Sovereign | — |
| Mandatory Access Control (MAC) | Government mode | — |
| Audit-enforced approval chains | `internal/policy/` | — |
| Offline policy cache | Pre-signed bundles | — |
| Feature governance UI | — | `FeatureGovernance.tsx` |
| Config change risk scoring | `risk_service.go` | `ConfigRisk.tsx` |
| Formal verification engine | `internal/policy/verifier.go` | `PolicyVerifier.tsx` |

---

## 15. Security Graph & Intelligence

| Feature | Backend | Frontend |
|---|---|---|
| Entity graph model | `internal/graph/` | — |
| Adjacency compressed store (BadgerDB) | `internal/graph/` | — |
| Path traversal detection | `internal/graph/` | — |
| Multi-hop attack inference | `internal/graph/` | — |
| Lateral movement chain detection | `internal/graph/` | — |
| Graph exploration UI | — | `ThreatGraph.tsx` |
| Credential usage anomaly detection | `credential_intel_service.go` | `CredentialIntel.tsx` |

---

## 16. Simulation & Red Team

| Feature | Backend | Frontend |
|---|---|---|
| Threat simulation module | `internal/simulation/` | `SimulationPanel.tsx` |
| AttackReplayEngine (MITRE replay) | `internal/simulation/` | `AttackSimulation.tsx` |
| Detection coverage scoring | `internal/simulation/` | — |
| Platform resilience score | `internal/simulation/` | — |

---

## 17. Runtime Trust & Attestation

| Feature | Backend | Frontend |
|---|---|---|
| RuntimeTrustService (memory anomaly) | `trust_service.go` | `RuntimeTrust.tsx` |
| Syscall pattern detection | `internal/attestation/` | — |
| Service behavior fingerprinting | `internal/attestation/` | — |
| Process self-check hashing | `internal/attestation/` | — |
| Trust status endpoint | `/debug/trust` | — |
| System Trust Consensus Monitor | `governance_service.go` | `GlobalIntegrityDashboard.tsx` |

---

## 18. Disaster Recovery & War Mode

| Feature | Backend | Frontend |
|---|---|---|
| Air-gap replication mode | `disaster_service.go` | — |
| Kill-switch safe-mode | `disaster_service.go` | `WarMode.tsx` |
| Encrypted snapshot export/import | `disaster_service.go` | — |
| Cold backup restore automation | `disaster_service.go` | — |
| Data destruction workflow | — | `DataDestruction.tsx` |
| Temporal integrity service | `internal/temporal/` | `TemporalIntegrity.tsx` |

---

## 19. Productivity & Collaboration

| Feature | Backend | Frontend |
|---|---|---|
| Notes & runbook service | `notes_service.go` | Notes panel |
| Workspace manager | `workspace_service.go` | — |
| AI assistant (error explain, cmd gen) | `ai_service.go` | — |
| Theme engine (custom themes) | `theme_service.go` | — |
| Settings & configuration | `settings_service.go` | `Settings.tsx` |
| Command palette & quick switcher | — | `ui/` components |
| Auto-updater | `updater_service.go` | Updater panel |
| Team collaboration | `team_service.go` | `TeamDashboard.tsx` |
| Sync service | `sync_service.go` | — |

---

## 20. Infrastructure

| Feature | Backend | Frontend |
|---|---|---|
| Plugin framework (Lua sandbox) | `internal/plugin/` | `PluginManager.tsx` |
| Event bus pub/sub (bounded queue) | `internal/eventbus/` | — |
| Output batcher | `output_batcher.go` | — |
| REST API (chi router, TLS) | `internal/api/rest.go` | — |
| CLI mode binary | `cmd/cli/` | — |
| SIEM benchmark tool | `cmd/bench_siem/` | — |
| Soak test generator | `cmd/soak_test/` | — |

---

