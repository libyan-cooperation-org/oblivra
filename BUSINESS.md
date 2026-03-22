# OBLIVRA — Business & Commercial Roadmap
> **Audience**: Founders, BD, marketing, certification bodies
> **Prerequisite**: First 10 paying customers before most of this is worth building
> **Last reviewed**: 2026-03-22

---

## Phase 14: Expansion & Analyst Ecosystem

> Build this when there are enough users to certify. Not before.

- [ ] **Certified Analyst Program** — OBLIVRA-CA: Detection engineering, alert triage, threat hunting
- [ ] **Certified Engineer Program** — OBLIVRA-CE: Deployment, integration, customization
- [ ] **Certified Forensic Investigator Program** — OBLIVRA-CFI: Evidence collection, chain of custody, reporting
- [ ] **Labs + CTF Platform** — Hosted attack scenarios for certification practice
- [ ] **Video Tutorial Series** — Installation through advanced detection authoring

---

## Commercial Capabilities (Build when customers require)

### Certification Readiness
- [ ] ISO 27001 organizational certification alignment (evidence collection framework)
- [ ] SOC 2 Type II automated evidence collection (control mapping to existing audit logs)
- [ ] Common Criteria evaluation preparation (long-term, government channel)
- [ ] FIPS 140-3 crypto module compliance pathway (required for US federal)

### Legal & Contract Readiness
- [ ] EULA template finalized with legal review
- [ ] Article 28 Data Processing Agreement (DPA) for GDPR compliance
- [ ] HIPAA Business Associate Agreement (BAA) template
- [ ] Standard Contractual Clauses (SCCs) for EU data transfers
- [ ] Security Addendum: encryption standards, incident notification SLA
- [ ] Product privacy policy, cookie policy, subprocessor list
- [ ] Penetration test report (external vendor, annual cadence)

### Go-To-Market Infrastructure
- [ ] **POC / Trial Automation** — One-click deployment with pre-loaded sample data and attack scenarios
- [ ] **POC Data Generator** — Synthetic log generator with realistic attack scenario injection
- [ ] **Support Bundle Generator** — `oblivra support-bundle` (sanitized logs, config, metrics, no secrets)
- [ ] **Competitive Comparison Mode** — Side-by-side Splunk SPL vs OQL query performance

### Documentation Suite
- [ ] Administrator Guide (Installation, User Mgmt, Log Sources, Rules, Maintenance)
- [ ] Analyst Guide (OQL Reference, Dashboards, Investigation, Hunting, Case Mgmt)
- [ ] API Reference (OpenAPI 3.0, Auth, Endpoints, SDKs — auto-generated from annotations)
- [ ] Integration Guide (Per-connector setup, Agent deployment, Syslog config)
- [ ] Release Notes (New features, Breaking changes, Migration guides)
- [ ] Static site generator (Docusaurus/MkDocs with versioning and full-text search)
- [ ] In-product contextual help links (`?` icon → relevant doc page)

### MSSP / Multi-Tenant Commercial Features
- [ ] Multi-tenant SOC view (single pane across all tenants)
- [ ] Per-tenant SLA tracking and reporting
- [ ] Tenant onboarding automation
- [ ] White-label UI capability
- [ ] Per-tenant data residency enforcement
- [ ] Cross-border data transfer logging and controls

### First-Run Experience
- [ ] **Setup Wizard** (`SetupWizard.tsx`) — 6-step onboarding for new deployments:
  - Step 1: Admin account + MFA enrollment
  - Step 2: Network config (ports, TLS cert)
  - Step 3: Log source setup (guided)
  - Step 4: Alert channel setup (test button)
  - Step 5: Detection pack selection
  - Step 6: First search tutorial (interactive)
- [ ] **Getting Started Dashboard** — Progress checklist, auto-dismiss after 7 days
- [ ] **Platform Health Assessment** — Weekly self-assessment with recommendations engine

### Advanced Isolation (Strategic)
- [ ] Vault process isolation (separate signing key service)
- [ ] mTLS between all internal service boundaries (if/when split to microservices)
- [ ] Service-level privilege separation design doc
