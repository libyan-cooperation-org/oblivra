# OBLIVRA — Long-Term Roadmap
> **Audience**: Engineering, product, investors
> **Prerequisite**: v2 platform stable, >50 enterprise customers
> **Last reviewed**: 2026-03-22
>
> Items here are real and will be built. They are not in `task.md` because
> none of their dependencies are met yet. Building them now would be
> scope creep disguised as ambition.

---

## Phase 16: Cloud Security Posture Management (CSPM)

**Gating requirement**: Cloud connector framework (Phase 20.7) must be stable first.

### Cloud Asset Inventory
- [ ] AWS: IAM, EC2, S3, Lambda, RDS, VPC enumeration via SDK
- [ ] Azure: Entra ID, VMs, Storage, AKS via SDK
- [ ] GCP: IAM, GCE, GCS, GKE via SDK
- [ ] Unified asset model: cloud resources alongside on-prem hosts

### Misconfiguration Detection
- [ ] S3 public bucket detection
- [ ] IAM policy overprivilege analysis (unused permissions)
- [ ] Security group / NSG rule audit (0.0.0.0/0 ingress)
- [ ] Encryption-at-rest verification for storage/databases
- [ ] MFA enforcement audit for all identity providers

### Cloud Threat Detection
- [ ] CloudTrail/Activity Log/Audit Log anomaly detection
- [ ] Impossible travel detection for cloud console access
- [ ] Privilege escalation path detection in IAM
- [ ] Resource hijacking detection (cryptomining, bot enrollment)

### Cloud Security Dashboard (`CloudPosture.tsx`)
- [ ] Multi-cloud posture score
- [ ] Drift detection from baseline
- [ ] CIS Benchmarks automated scoring (AWS/Azure/GCP)
- [ ] Remediation playbook integration

---

## Phase 17: Container & Kubernetes Security

### Runtime Protection
- [ ] Container image vulnerability scanning (Trivy/Grype integration)
- [ ] Kubernetes audit log ingestion + detection rules
- [ ] Runtime anomaly detection: unexpected process in container
- [ ] Container escape detection (nsenter, mount namespace breakout)

### Kubernetes-Native Deployment
- [ ] Helm chart for OBLIVRA server
- [ ] DaemonSet manifest for agent deployment
- [ ] Kubernetes RBAC integration (map K8s ServiceAccounts to OBLIVRA roles)
- [ ] CRD for detection rules (GitOps-native rule management)

### Service Mesh Observability
- [ ] Envoy/Istio access log ingestion
- [ ] East-west traffic anomaly detection
- [ ] mTLS certificate audit

---

## Phase 18: Vulnerability Management Integration

### Scanner Integration
- [ ] Ingest Nessus/Qualys/Rapid7 scan results (XML/JSON)
- [ ] Ingest OpenVAS reports
- [ ] Normalize to unified vulnerability model (CVE, CVSS, affected asset)

### Risk-Based Prioritization
- [ ] Correlate vulnerabilities with threat intel (exploited in-the-wild?)
- [ ] Correlate with network exposure (internet-facing? segmented?)
- [ ] Correlate with asset criticality (crown jewel analysis)
- [ ] Output: prioritized remediation queue, not raw CVE list

### Vulnerability Dashboard (`VulnManagement.tsx`)
- [ ] MTTR tracking per severity
- [ ] SLA compliance visualization
- [ ] Patch verification (was the vuln actually fixed?)
- [ ] Attack path: "this unpatched Apache → can reach this database → contains PII"

---

## Phase 19: Email & Phishing Security

### Email Log Ingestion
- [ ] Microsoft 365 Message Trace ingestion
- [ ] Google Workspace email log ingestion
- [ ] Generic SMTP log parsing

### Phishing Detection
- [ ] URL reputation checking against threat intel
- [ ] Domain similarity detection (homoglyph, typosquat)
- [ ] Attachment hash matching against known malware
- [ ] BEC detection (impersonation of executives, domain spoofing)

### User-Reported Phish Pipeline
- [ ] API endpoint for phishing report submission
- [ ] Auto-triage: extract IOCs, check reputation, score risk
- [ ] Auto-quarantine if confidence > threshold

---

## Phase 20: Platform Tier-0 Gaps

> These are large, well-defined capabilities that will eventually be required.
> See the full specification in `docs/phase_dependencies.md`.

- [ ] **20.1 — SovereignQL (OQL)**: Pipe-based query language compiler (3-4 months)
- [ ] **20.2 — Intelligent Data Tiering**: Hot/Warm/Cold/Frozen/Archive with automatic lifecycle
- [ ] **20.3 — Risk-Based Alerting**: Per-entity risk register with temporal decay
- [ ] **20.4 — SCIM Normalization**: OCSF-based common information model
- [ ] **20.5 — Log Source Health Engine**: Coverage matrix, silence detection, schema drift
- [ ] **20.6 — Detection-as-Code Engine**: Git-native rule repo, PR-based deployment, shadow mode
- [ ] **20.7 — Integration Hub**: 50+ connectors (VirusTotal, MISP, Okta, Jira, CrowdStrike, etc.)
- [ ] **20.8 — SOC Operations Intelligence**: MTTD/MTTR/MTTC metrics, analyst performance
- [ ] **20.9 — Automated Triage Engine**: Auto-pull related events, composite triage score
- [ ] **20.10 — Report Factory**: Drag-and-drop templates, PDF/HTML/DOCX, scheduled delivery
- [ ] **20.11 — Dashboard Studio**: Visual drag-and-drop builder, real-time OQL preview
- [ ] **20.12 — Native Investigation Workflow**: Ticket lifecycle, SLA engine, Kanban queue
- [ ] **20.13 — Competitive Migration Engine**: SPL→OQL transpiler, Splunk/Elastic import

---

## Phase 21: Tier 1 Platform Capabilities

- [ ] **21.1 — Federated Search**: Scatter-gather across N OBLIVRA instances
- [ ] **21.2 — Investigation Notebooks**: Markdown + query cells, multi-analyst CRDT editing
- [ ] **21.3 — Data Pipeline Engine**: Visual Cribl-style pipeline builder
- [ ] **21.4 — Malware Sandbox**: Static + dynamic analysis, YARA matching, URL detonation
- [ ] **21.5 — Asset Intelligence Engine**: Crown jewel identification, attack surface scoring
- [ ] **21.6 — Multi-Level Security (MLS)**: Classification labels, compartment tags, cross-domain guard
- [ ] **21.7 — Knowledge Base**: Analyst wiki, SOPs, lessons learned repository
- [ ] **21.8 — STIX/TAXII Server**: Bidirectional intel sharing, MISP integration

---

## Phases 22–26: Advanced Frontiers

- [ ] **22.x** — Protocol Analysis Engine (Zeek-level DPI), AI Copilot (NLQ→OQL), Covert Channel detection, Autonomous Hunt Engine
- [ ] **23.x** — Distributed Data Plane (1 TB/day ingestion), Real-time Streaming Architecture, App Marketplace
- [ ] **24.x** — Insider Threat, DLP, API Security, Autonomous Detection Validation, Unified Posture Score
- [ ] **25.x** — ITDR (AD attack detection, BloodHound-style path analysis), AI/LLM Security, EASM, DRP, OT/ICS Security, Certificate Lifecycle Management
- [ ] **26.x** — Endpoint Prevention (kernel-level), SaaS Security Posture Management, Automated Exposure Validation
