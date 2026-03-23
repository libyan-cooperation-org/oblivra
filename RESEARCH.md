# OBLIVRA — Research Roadmap
> **Audience**: Research team, academic partners, NSA/DARPA grant applications
> **Status**: PARKED. None of this work starts until OBLIVRA has 10 active production users.
> **Why**: See `STRATEGY.md`. Building post-quantum cryptography before product-market fit
> is the canonical way to spend years on a platform nobody uses.
> **Prerequisite**: 10 active daily users → 72h soak pass → first paying customer → then this.
> **Last reviewed**: 2026-03-23 (Strategic Pivot — Operator Mode First)

---

## Phase 13: Elite Research & Academic Rigor (DARPA/NSA Grade)

### 13.1 — Formal Verification Extension (Beyond TLA+)

**Already done** (in `task.md`):
- `deterministic_model.tla` — 5 safety invariants + liveness
- `rules_model.tla` — NoSpuriousAlerts + WindowStateInvariant

**Remaining research work:**

#### Protocol Verification (Tamarin/ProVerif)
- [ ] Model gRPC agent↔server mutual auth protocol
- [ ] Model vault unlock/seal ceremony
- [ ] Model cluster Raft leader election trust boundaries
- [ ] Prove: no key material leaks across trust boundary transitions

#### Runtime Verification (Temporal Logic Monitoring)
- [ ] LTL monitors on detection pipeline (`□(event_ingested → ◇ rule_evaluated)`)
- [ ] Safety monitors: "no alert fires without corresponding raw event"
- [ ] Liveness monitors: "every ingested event eventually reaches storage"
- [ ] Implement as lightweight in-process monitors, not external tools

#### Property-Based Testing (go-rapid / gopter)
- [ ] All parsers: arbitrary byte sequences never panic
- [ ] All API endpoints: random payloads never produce 5xx
- [ ] Correlation engine: event ordering invariants hold under permutation
- [ ] Crypto operations: round-trip encrypt/decrypt identity for all key sizes

#### Symbolic Execution of Critical Paths
- [ ] Identify top 10 security-critical functions (auth, crypto, rule eval)
- [ ] Run KLEE or go-z3 bindings on auth bypass paths
- [ ] Prove: no input to API auth middleware produces unauthorized access

#### Information Flow Analysis (Taint Tracking)
- [ ] Static taint analysis: PII fields never reach unencrypted log sinks
- [ ] Static taint analysis: API keys never serialized to debug logs
- [ ] Enforce via CI — forbidden data flow paths as test assertions

---

### 13.2 — Provenance & Causal Reasoning (DARPA Transparent Computing)

#### Whole-System Provenance Graph
- [ ] Extend eBPF agent to emit process→file→network causal edges
- [ ] Build provenance DAG storage (BadgerDB adjacency lists or embedded graph)
- [ ] Implement backward tracing: "this alert → what caused it → full kill chain"
- [ ] Implement forward tracing: "this IOC → what did it touch → blast radius"
- [ ] Graph pruning: dependency reduction to keep storage bounded

#### Automated Root Cause Analysis
- [ ] Causal inference engine: given alert, walk provenance graph backward
- [ ] Identify initial access vector automatically from kill chain reconstruction
- [ ] Generate human-readable incident narrative from graph path
- [ ] Benchmark: mean time to root cause < 60s for simulated attacks

#### Attack Graph Generation
- [ ] Given network topology + vulnerability data, generate possible attack paths
- [ ] Score paths by exploitability × impact
- [ ] Visualize in `AttackGraph.tsx` with interactive path highlighting
- [ ] Integrate with compliance: "which unpatched path violates PCI requirement X?"

---

### 13.3 — Adversarial ML Robustness

#### Evasion Testing Framework
- [ ] Implement gradient-free adversarial perturbation for Isolation Forest
- [ ] Test: can attacker slowly shift baseline to make malicious behavior "normal"?
- [ ] Test: can attacker poison training data via controlled benign events?
- [ ] Document attack surface of ML pipeline in threat model

#### Concept Drift Detection
- [ ] Monitor feature distribution statistics per entity over time
- [ ] Alert when baseline drift exceeds statistical threshold (KL divergence)
- [ ] Auto-trigger retraining or baseline reset on drift detection
- [ ] Distinguish: legitimate behavior change vs. adversarial drift

#### Model Integrity Verification
- [ ] Hash all model parameters at training time, verify at inference time
- [ ] Signed model artifacts (extend Sigstore to ML models)
- [ ] Tamper detection: alert if model weights modified on disk

#### Differential Privacy for Behavioral Baselines
- [ ] Add calibrated noise to per-user baselines
- [ ] Prove: individual user behavior not reconstructable from stored baseline
- [ ] Formal ε-δ privacy guarantee documentation

---

### 13.4 — Post-Quantum Cryptography Readiness

#### PQC Algorithm Integration
- [ ] ML-KEM (Kyber) for key encapsulation in vault operations
- [ ] ML-DSA (Dilithium) for release signing (alongside Ed25519)
- [ ] SLH-DSA (SPHINCS+) as backup stateless signature scheme
- [ ] Use Go 1.23+ `crypto/mlkem` or CIRCL library

#### Hybrid Cryptography Mode
- [ ] Vault: X25519 + ML-KEM hybrid key agreement
- [ ] Agent transport: TLS 1.3 with hybrid key exchange
- [ ] Signed releases: dual Ed25519 + ML-DSA signatures
- [ ] Configurable: operators choose classical-only, hybrid, or PQC-only

#### Crypto Agility Framework
- [ ] Abstract all crypto operations behind algorithm-negotiation layer
- [ ] Runtime algorithm selection without recompilation
- [ ] Migration tooling: re-encrypt vault with new algorithm suite
- [ ] Document crypto inventory: every algorithm, every use site

---

### 13.5 — Reproducible Research & Academic Contribution

**Already done**: CIC-IDS-2017 + Zeek benchmark datasets, harness bug fixes, internal whitepapers

#### Remaining
- [ ] Publish standardized detection benchmark (precision, recall, F1)
- [ ] Containerized benchmark runner anyone can reproduce
- [ ] Publish results with confidence intervals, not single-run numbers
- [ ] Map detection coverage to MITRE Engenuity evaluation format
- [ ] Self-score against published APT29/Turla/Wizard Spider scenarios
- [ ] Gap analysis report auto-generation
- [ ] Peer-reviewed paper pipeline: Provenance, Formal Verification, Adversarial Robustness
- [ ] Sanitized dataset export for academic partners
- [ ] Plugin API for researchers to deploy experimental detectors

---

### 13.6 — Novel Detection Paradigms

#### Graph Neural Network (GNN) Detector
- [ ] Model network communications as temporal graph
- [ ] Train GNN on normal graph structure, detect structural anomalies
- [ ] Target: lateral movement, C2 beaconing, data staging
- [ ] Inference in Go via ONNX runtime (no Python dependency)

#### Program Synthesis for Rule Generation
- [ ] Given attack description (NL or STIX), generate YAML rule
- [ ] Constraint-based synthesis: rule must match positive examples, reject negatives
- [ ] Integrate with AI assistant for analyst-guided rule authoring

#### Information-Theoretic Detection
- [ ] Kolmogorov complexity estimation for command sequences
- [ ] Entropy rate monitoring on per-user command distributions
- [ ] Detect: encoded/obfuscated commands, steganographic exfiltration

#### Temporal Pattern Mining
- [ ] Sequential pattern mining on event streams (PrefixSpan/GSP adapted)
- [ ] Discover recurring attack subsequences automatically
- [ ] Cluster similar attack campaigns without predefined rules
