# OBLIVRA: Implementation Dependency Map (DAG)

This document visualizes the critical path for OBLIVRA development, identifying hard and soft dependencies between phases to prevent architectural stalls.

## Core Dependency Logic

```mermaid
graph TD
    subgraph "Phase 0-1: Foundations"
        P0[Phase 0: Stabilization] --> P05[Phase 0.5: Desktop vs Browser Split]
        P05 --> P1[Phase 1: Storage / Ingestion / Search]
        P1 --> P20_1[20.1: SovereignQL - OQL]
    end

    subgraph "Phase 2-4: The Product Substrate"
        P1 --> P2[Phase 2: Alerting / API]
        P2 --> P1_3[1.3: OpenAPI Spec]
        P1 --> P4[Phase 4: Compliance / Audit]
        P4 --> P4_3[4.3: Legal Readiness]
        P2 --> P1_7[1.7: Mobile On-Call]
    end

    subgraph "Advanced Feature Dependencies"
        P20_1 --> P20_6[20.6: Detection-as-Code]
        P20_1 --> P20_11[20.11: Dashboard Studio]
        P20_11 --> P20_10[20.10: Report Factory]
        P20_1 --> P22_4[22.4: Autonomous Hunt]
        
        P1 --> P3[Phase 3: Threat Intel]
        P2 --> P20_3[20.3: Risk-Based Alerting]
        P21_5[21.5: Asset Intelligence] --> P20_3
        P21_5 --> P20_9[20.9: Automated Triage]
        P20_3 --> P20_9
        P3 --> P20_9
        
        P1 --> P3_1[3.1: Graph Infrastructure]
        P3_1 --> P25_1[25.1: ITDR]
        P20_7[20.7: Identity Connectors] --> P25_1
        
        P4 --> P10_6[10.6: Attack Fusion]
        P20_4[20.4: SCIM Normalization] --> P10_6
    end

    subgraph "Market Expansion"
        P1 --> P7[Phase 7: Agents]
        P7 --> P26_1[26.1: Endpoint Prevention]
    end
```

## Critical Path for Productization (Sprint 1)

These items form the minimum viable sequence for a first production customer:

1.  **Phase 0.5** (Architecture Split) → Essential for defining deployment.
2.  **Phase 1.3** (OpenAPI) → Unblocks integration conversations.
3.  **Phase 1.2/1.6** (Docs & Accessibility) → Required for procurement.
4.  **Phase 20.18** (Entity Pages) → Primary investigation interface.
5.  **Phase 4.1/4.2** (POC Generator & Support bundle) → Commercial readiness.

## Known Conflicts & Resolution

| Conflict Point | Description | Resolution Plan |
| :--- | :--- | :--- |
| **RBA vs Asset Intel** | Phase 20.3 (RBA) needs Asset Criticality (Phase 21.5). | Implement "Scaffolded" Asset Intel (Phase 21.5) as a prerequisite for RBA. |
| **Triage Inputs** | Phase 20.9 (Triage) requires RBA, Threat Intel, and Asset Intel. | 20.9 cannot reach `Validated [v]` status until 20.3, 20.7, and 21.5 are `Scaffolded [s]`. |
| **OQL Everywhere** | Almost all advanced analytics (Dashboards, Hunting, Reports) depend on OQL. | **Tier 0 Priority**: OQL must reach `Production-Ready [x]` status before Phase 20.11 begins. |
