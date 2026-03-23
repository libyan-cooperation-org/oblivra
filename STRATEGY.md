# OBLIVRA — Strategic Rationale
> Written: 2026-03-23. Revised: 2026-03-23 (Identity + Positioning Update).
> **Read this before opening `task.md`. Read this before writing any code.**
> If you are about to add a feature, answer the five questions at the bottom first.

---

## What OBLIVRA Actually Is

This is not a SIEM. The word "SIEM" undersells it by a factor of ten.

As of March 2026, OBLIVRA is the only platform that combines all of the following in a single binary:

- **Terminal + SSH + Vault** — full PTY, connection pooling, FIDO2, AES-256 local credential store
- **SIEM** — BadgerDB + Bleve, 18,000+ EPS validated, OQL, Sigma transpiler, 82 detection rules, correlation engine
- **EDR** — eBPF agent (execve/tcp/file/ptrace probes), gRPC/mTLS, offline WAL buffering, PII redaction
- **SOAR** — playbook engine, case management, deterministic execution, Jira/ServiceNow
- **UEBA** — Isolation Forest, EMA behavioral baselines, entity risk scoring with temporal decay
- **NDR** — NetFlow/IPFIX, JA3/JA3S fingerprinting, lateral movement correlation, network map
- **Forensics** — Merkle-tree evidence locker, chain of custody, compliance packs (PCI/NIST/ISO/HIPAA/SOC2)
- **Air-gap capable** — offline update bundles, kill-switch safe-mode, no mandatory cloud dependency
- **Hybrid deployment** — same codebase runs as a Wails desktop app or a headless web server

No existing commercial product combines terminal + SIEM + EDR + vault in a single tool. Not Splunk. Not CrowdStrike. Not SentinelOne. Not Elastic. None of them.

**The correct category is: Sovereign Cyber Operations OS.**

---

## The Positioning Decision

There are two viable paths. Pick one. Building both simultaneously is how this dies.

### Option A — Enterprise SOC Platform
Compete with Splunk, IBM QRadar, Microsoft Sentinel.
- Multi-tenant, compliance-first, SOC analyst workflow
- Requires: multi-tenant isolation, SOC 2, enterprise procurement process
- Time to first revenue: 18–24 months minimum
- Risk: competing against $10B companies on their home turf

### Option B — Elite Operator Tool ✅ CHOSEN
Compete with nothing directly.
- Single security engineer or small red/blue team as primary user
- Local-first, air-gapped capable, terminal-native
- Requires: one killer workflow, ruthless UX polish, developer trust
- Time to first revenue: 3–6 months (individual licenses, team plans)
- Risk: niche market, but the niche has no real competitor

**The choice is Option B.** Here is why:

OBLIVRA's technical moat is the combination of terminal + intelligence + response in one tool. That moat only matters to someone who lives in a terminal. Enterprise SOC analysts use dashboards. Elite operators use terminals. Build for the person who will actually appreciate what makes OBLIVRA different.

The elite operator market is also the trust-building market. When a red team at a Fortune 500, a CISO at a defense contractor, or a security researcher at a university uses OBLIVRA and writes about it — that creates the credibility that unlocks Option A later. You cannot go the other direction.

---

## The Killer Workflow (Operator Mode)

One sentence: **"OBLIVRA lets a security engineer investigate, detect, and respond to an attack from a single terminal in under 60 seconds."**

The workflow that makes this real:

```
1. ssh prod-server          → native PTY, vault-stored credentials, session recorded
2. anomaly detected         → OQL auto-suggestion appears in status bar
3. pivot to logs            → one keystroke opens SIEM panel filtered to that host
4. enrich the source IP     → GeoIP + ASN + TI match inline in the event row
5. isolate the host         → playbook fires network isolation via agent, one confirmation
6. dump process memory      → forensic artifact captured, SHA-256 sealed, chain of custody started
7. store evidence           → Merkle-tree entry created, immutable, ready for incident report
```

Every step of this flow is already implemented. It just isn't wired as a single coherent UX. That wiring is the product. That is what Phase 22 builds.

No competitor does this. Splunk requires 4 different browser tabs. CrowdStrike requires leaving the terminal entirely. Security operations tools are built for analysts who use GUIs. OBLIVRA is built for engineers who use terminals. Those are different people with different needs and different willingness to pay for the right tool.

---

## The Trajectory Risk

The graveyard of security tools is full of technically excellent platforms that died at the same stage OBLIVRA is at now. The failure pattern is always the same:

1. Build impressive engineering system ✓
2. Keep adding features because it feels like progress ← **we are here**
3. Enter "DARPA-grade" research before product-market fit ← **RESEARCH.md is this**
4. First real user hits a rough edge and leaves
5. No trust layer → no adoption → no feedback → no improvement
6. Dies quietly with impressive commit history

The specific danger from RESEARCH.md (Phase 13):
- Post-quantum cryptography (Kyber, Dilithium) — PhD-level, 3–5 year horizon
- Adversarial ML robustness — research paper territory
- GNN-based detection — requires training data that doesn't exist yet
- Formal protocol verification (Tamarin/ProVerif) — correct eventually, wrong now
- Differential privacy for behavioral baselines — correct eventually, wrong now

None of these are wrong ideas. All of them are wrong timing. A platform with zero production deployments does not need post-quantum cryptography. It needs users. Users create feedback. Feedback creates a real product. A real product can justify post-quantum cryptography.

**Every hour spent on Phase 13 is an hour not spent on getting the first 10 users.**

---

## The 5 Gaps That Actually Matter Now

These are not feature gaps. They are product gaps. The distinction matters.

### Gap 1: The Operator Mode Workflow Is Not Wired
The individual capabilities exist. The integrated flow — ssh → detect → pivot → enrich → isolate → collect → seal — does not exist as a single UX. Without this, OBLIVRA is a collection of powerful tools in a trenchcoat, not a product.

**Fix: Wire the Operator Mode flow end-to-end. This is Phase 22.4.**

### Gap 2: First-Run Experience Is Broken
A security engineer downloads OBLIVRA, runs it, and sees a blank dashboard with a vault unlock screen that crashes in browser mode. They close it and never return. The product never gets a second chance.

**Fix: Setup Wizard + browser mode stability. This is Phase 22 Immediate Hygiene.**

### Gap 3: No Chaos Evidence
OBLIVRA has never been tested against agent death with events in-flight, BadgerDB corruption, OOM kill, or clock skew. The first production deployment will hit one of these. Without a documented recovery story, trust cannot be established.

**Fix: Chaos test harness. This is Phase 22.1.**

### Gap 4: No Trust Signals
A security engineer evaluating OBLIVRA asks: who audited the cryptography? Where is the threat model? What happens if I find a vulnerability? Currently the answers are: nobody external, it's internal, there's no process. These questions have to have answers before any serious operator will stake their reputation on using this tool.

**Fix: Publish threat model, cryptographic transparency doc, security.txt. This is Phase 22.5.**

### Gap 5: Multi-Tenant Isolation Is Policy, Not Structure
If OBLIVRA ever serves multiple customers on shared infrastructure, the current `TenantID` filter approach is not sufficient. A bug in middleware can expose one tenant's data to another. This must be structural before any multi-customer deployment.

**Fix: Tenant-prefixed keyspace, per-tenant Bleve index. This is Phase 22.2.**

---

## What To Stop Building

| Item | Why It's Wrong Right Now |
|---|---|
| Phase 13: Post-quantum crypto | Zero production deployments. PQC solves a future problem. |
| Phase 13: Adversarial ML | Requires production telemetry that doesn't exist yet. |
| Phase 13: GNN detector | Requires training data. Go get users first. |
| Phase 13: Formal verification | Correct in Year 3. Wrong in Month 1. |
| Phase 16: CSPM | Requires multi-tenant foundation + different buyer persona. |
| Phase 17: K8s Security | Same dependency. |
| Phase 18: Vuln Management | Different product. Different user. |
| More detection rules | 82 rules without versioning/testing is already too many to maintain. |
| ClickHouse backend | Premature optimization. BadgerDB handles current load. |
| DAG streaming engine | Correct architecture eventually. Phase 8 carry-over — park it. |

---

## The Correct Next 90 Days

| # | Work | Outcome |
|---|---|---|
| 0 | Browser crash fixes (VaultGuard, store.tsx) | Web mode actually loads |
| 0 | git rm -r --cached frontend/node_modules | Clone time drops from 10min to 30s |
| 1 | Setup Wizard (6-step first-run) | First-run completion rate goes from ~10% to ~70% |
| 2 | Operator Mode workflow wiring | The product has a story, not just features |
| 3 | Chaos test harness | Can claim "battle-tested" with evidence |
| 4 | Publish threat model + security.txt | Security engineers trust the cryptography |
| 5 | Multi-tenant structural isolation | Can serve multiple customers safely |
| 6 | Hot/Warm/Cold storage tiers | Cost model exists, can price it |
| 7 | Rule versioning + test framework | Detection engineering is a workflow, not a library |

---

## The Market Position

**"The Linux of Cyber Operations."**

More specifically: the first tool that lets a single security engineer — not a 20-person SOC with a $500K Splunk license — detect, investigate, and respond to an attack from one terminal without switching context.

This wins in markets that commercial platforms actively ignore:
- **Individual security engineers and consultants** — no $500K/yr platform, but willing to pay $500/yr for the right tool
- **Red teams** — need terminal-native tooling with integrated intelligence
- **Privacy-conscious enterprises** — EU healthcare, legal, financial — who cannot send telemetry to US cloud
- **Air-gapped environments** — defense, critical infrastructure, government — where cloud is impossible
- **Cost-sensitive buyers replacing Splunk** — the CISO who knows their Splunk bill is indefensible

The hybrid desktop+web architecture is the moat. A desktop app that scales to a SOC when you need it, goes air-gap when you need that, and never requires a cloud account for basic operation. No competitor can build this without a full rewrite. Most won't bother because the enterprise dashboard market is larger. That is the gap.

---

## The Test

Before writing any new code, answer these five questions:

1. Does this bring the Operator Mode workflow closer to being a single coherent UX?
2. Does this make OBLIVRA more trustworthy to a security engineer evaluating it for the first time?
3. Does this fix something that would make a first-time user give up and leave?
4. Does this validate that the platform survives a real production failure condition?
5. Is the answer to questions 1–4 definitively "no"?

If the answer to question 5 is yes — park the work in `ROADMAP.md` and move on.

The goal for the next 90 days is not a longer feature list. The goal is 10 security engineers who use OBLIVRA every day and tell other security engineers about it. Everything else follows from that.
