# OBLIVRA — LONG-TERM ROADMAP

## Phase 12: Enterprise Depth (Scale & Cluster)
- [ ] **Clustered Ingestion (Load Balancing)**:
    - [ ] `IngestProxy` (Nginx/Envoy based) for incoming syslog/agent/HTTP traffic
    - [ ] Shared state via `etcd` or `consul` for configuration syncing
- [ ] **Distributed Query Header**:
    - [ ] Federated search across multiple BadgerDB instances
- [ ] **Storage Lifecycle Policy Manager**:
    - [ ] Auto-move buckets from Hot (SSD) to Cold (HDD/S3) storage after N days
- [ ] **Multi-Tenant Authorization (RBAC Hardening)**:
    - [ ] Scoped query visibility (User A can only see logs from Source X)
- [ ] **High Availability (HA) Control Plane**:
    - [ ] Database replication (streaming) for non-event data (cases, users, vault)

## Phase 16: Cloud Security Posture Management (CSPM)
- [ ] Cloud Asset Inventory: AWS (IAM, EC2, S3, Lambda, RDS, VPC), Azure (Entra ID, VMs, Storage, AKS), GCP (IAM, GCE, GCS, GKE)
- [ ] Misconfiguration Detection: S3 public buckets, IAM permissive roles, unencrypted RDS, open Security Groups
- [ ] Compliance Benchmarks: CIS Foundations for AWS/Azure/GCP
- [ ] Map findings to existing compliance packs (PCI, NIST, ISO)
- [ ] Cloud-specific compliance reports
- [ ] **Cloud Threat Detection**
    - [ ] CloudTrail/Activity Log/Audit Log anomaly detection
    - [ ] Impossible travel detection for cloud console access
    - [ ] Privilege escalation path detection in IAM
    - [ ] Resource hijacking detection (cryptomining, bot enrollment)
- [ ] **Cloud Security Dashboard** (`CloudPosture.tsx`)
    - [ ] Multi-cloud posture score
    - [ ] Drift detection from baseline
    - [ ] Remediation playbook integration (auto-fix misconfigs)

### Phase 17: Container & Kubernetes Security
- [ ] **Runtime Protection**
    - [ ] Container image vulnerability scanning (Trivy/Grype integration)
    - [ ] Kubernetes audit log ingestion + detection rules
    - [ ] Pod security policy / admission controller violations
    - [ ] Runtime anomaly detection: unexpected process in container
    - [ ] Container escape detection (nsenter, mount namespace breakout)
- [ ] **Kubernetes-Native Deployment**
    - [ ] Helm chart for OBLIVRA server
    - [ ] DaemonSet manifest for agent deployment
    - [ ] Kubernetes RBAC integration (map K8s ServiceAccounts to OBLIVRA roles)
    - [ ] CRD for detection rules (GitOps-native rule management)
- [ ] **Service Mesh Observability**
    - [ ] Envoy/Istio access log ingestion
    - [ ] East-west traffic anomaly detection
    - [ ] mTLS certificate audit

### Phase 18: Vulnerability Management Integration
- [ ] **Scanner Integration**
    - [ ] Ingest Nessus/Qualys/Rapid7 scan results (XML/JSON)
    - [ ] Ingest OpenVAS reports
    - [ ] Normalize to unified vulnerability model (CVE, CVSS, affected asset)
- [ ] **Risk-Based Prioritization**
    - [ ] Correlate vulnerabilities with threat intel (exploited in-the-wild?)
    - [ ] Correlate with network exposure (internet-facing? segmented?)
    - [ ] Correlate with asset criticality (crown jewel analysis)
    - [ ] Output: prioritized remediation queue, not raw CVE list
- [ ] **Vulnerability Dashboard** (`VulnManagement.tsx`)
    - [ ] MTTR tracking per severity
    - [ ] SLA compliance visualization
    - [ ] Patch verification (was the vuln actually fixed?)
- [ ] **Attack Path Correlation**
    - [ ] Combine vulnerability data + network topology + provenance
    - [ ] Show: "this unpatched Apache → can reach this database → contains PII"
    - [ ] Quantified risk score per attack path

### Phase 19: Email & Phishing Security
- [ ] **Email Log Ingestion**
    - [ ] Microsoft 365 Message Trace ingestion
    - [ ] Google Workspace email log ingestion
    - [ ] Generic SMTP log parsing
- [ ] **Phishing Detection**
    - [ ] URL reputation checking against threat intel
    - [ ] Domain similarity detection (homoglyph, typosquat)
    - [ ] Attachment hash matching against known malware
    - [ ] BEC detection (impersonation of executives, domain spoofing)
- [ ] **User-Reported Phish Pipeline**
    - [ ] API endpoint for phishing report submission
    - [ ] Auto-triage: extract IOCs, check reputation, score risk
    - [ ] Auto-quarantine if confidence > threshold
    - [ ] Analyst review queue for borderline cases

### Phase 20: Sovereign Query Language (SovereignQL / OQL) - Advanced
- [ ] Query cost estimator (reject queries that would scan >N GB without index)
- [ ] Query result caching (LRU, TTL-aware)
- [ ] Saved queries → scheduled queries → alerts (full lifecycle)

### Phase 21: Tier 1 Platform Capabilities
- [ ] Federated Search (Multi-Instance)
- [ ] Investigation Notebooks (Analyst Workbench)
- [ ] Data Pipeline Engine (Cribl-Style)
- [ ] Automated Analysis Engine (Malware Sandbox)
- [ ] Asset Intelligence Engine
- [ ] Multi-Level Security (MLS) Framework (Government)
- [ ] Knowledge Base (Analyst Wiki)
- [ ] Intelligence Sharing Platform (STIX/TAXII Server)

### Phase 22: Tier 2 Depth Capabilities (NSA/Research Grade)
- [ ] Protocol Analysis Engine (Zeek-Level DPI)
- [ ] Natural Language Security Analyst (AI Copilot)
- [ ] Covert Channel & Steganography Detection
- [ ] Autonomous Hunt Engine

### Phase 23: Tier 3 Scale & Architecture
- [ ] Distributed Data Plane (Indexer Clustering)
- [ ] Real-Time Streaming Architecture (CEP)
- [ ] App / Extension Marketplace
- [ ] Security Data Lakehouse Mode (BYOS)
- [ ] Cloud Log Collection Framework (Multi-Cloud)
- [ ] Multi-Region Architecture (Geo-Distributed)

### Phase 24: Advanced Frontiers (Specialized Programs)
- [ ] Insider Threat Detection
- [ ] Data Loss Prevention (DLP)
- [ ] API Security Monitoring
- [ ] Autonomous Detection Validation (Adversary Emulation)
- [ ] Unified Security Posture Score
- [ ] Data Flow Mapping (Privacy Compliance)
- [ ] Third-Party / Vendor Risk Management
- [ ] Secrets Sprawl Detection

### Phase 25: Advanced Specialized Domains
- [ ] Identity Threat Detection & Response (ITDR)
- [ ] AI/LLM Security (Shadow AI, Prompt Injection)
- [ ] External Attack Surface Management (EASM)
- [ ] Digital Risk Protection (DRP)
- [ ] OT/ICS Security
- [ ] Certificate Lifecycle Management (CLM)

### Phase 26: Market Expansion
- [ ] Endpoint Prevention Agent (EPP/EDR Convergence)
- [ ] SaaS Security Posture Management (SSPM)
- [ ] Automated Exposure Validation

---

## Summary: Priority Ranking

### Seven-Pass Consolidated Gap Table (Updated 2026-03-18)

| Platform | Phases | Status | Key Capabilities |
| :--- | :--- | :--- | :--- |
| **Hybrid** 🏗️ | Phase 0-5 | ✅ **Validated** | Storage (BadgerDB/Bleve), Ingest (5k EPS), Alerts, Intel, MITRE, Search |
| **Desktop** 🖥️ | Core + Phase 6 | ✅ **Validated** | SSH/PTY, Vault (AES-256), Terminal Grid, SFTP, Offline Updates, FIDO2 |
| **Web** 🌐 | Phase 0.3-0.5 | ✅ **Validated** | Login (OIDC/SAML), Fleet, Identity Admin, Escalation, Regulator Portal |
| **Hybrid** 🏗️ | Phase 6 | ✅ **Scaffolded** | Forensics, Compliance (PCI/NIST/ISO/GDPR/HIPAA/SOC2) |
| **Web** 🌐 | Phase 7 | ✅ **Validated** | Agent Framework (gRPC, eBPF, FIM, WAL) |
| **Hybrid** 🏗️ | Phase 8 | [v] **Validated** | SOAR (Case Mgmt, Playbook Engine, Jira/SNOW) |
| **Hybrid** 🏗️ | Phase 9-10 | [v] **Validated** | Ransomware (Entropy, Canary, Isolation), UEBA (IF, baselines) |
| **Web** 🌐 | Phase 11 | ✅ **Scaffolded** | NDR (NetFlow, DNS, TLS, Lateral Movement) |
| **Web** 🌐 | Phase 12 | ✅ **Scaffolded** | Enterprise (Multi-tenant, HA Cluster, RBAC, Lifecycle) |
| **All** | Phase 13+ | [v] **Validated** | Formal Verification, PQC, CSPM, K8s, Vuln Mgmt, OQL |
| **Web** 🌐 | Phase 16-26 | [ ] **Planned** | CSPM, Container Security, Email/Phishing, EASM, DRP, OT/ICS |
