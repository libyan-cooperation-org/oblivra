# Unified Trust Consensus for Sovereign Infrastructure

## Introduction
Sovereign infrastructure requires a trust model that does not depend on external authorities or persistent internet connectivity. This paper introduces the **Obli-Trust Consensus Protocol**, a multi-layered verification system that binds platform integrity to physical hardware, deterministic code execution, and immutable audit ledgers.

## The Five Pillars of Consensus
1. **Hardware Root (TPM/HSM)**: Identity is bound to physical burned-in keys.
2. **Runtime Attestation**: Memory and binary hashes are verified at execution time.
3. **Deterministic Logic**: Given the same input, the system must produce the same defense response (proven via TLA+).
4. **Immutable Ledger**: Every decision is recorded in a Merkle-tree based sign-only structure.
5. **Zero-Trust IPC**: Internal service boundaries are enforced via mTLS and process isolation.

## Global Integrity Score (GIS)
The GIS is a real-time metric computed by the `SystemTrustConsensusService`:
$GIS = \frac{W_{h}H + W_{r}R + W_{d}D + W_{l}L + W_{i}I}{\sum W}$
Where H, R, D, L, I represent the health of the five pillars.

## Air-Gap Protocol: The Resume Bundle
In event of total isolation, nodes maintain consensus via "Resume Bundles"—signed snapshots of the current state that can be physically transferred between nodes to maintain a unified defense posture.

## Conclusion
Unified Trust Consensus ensures that OBLIVRA remains a "Trusted Black Box" even in contested environments, where the observer cannot distinguish between a successful attack and a system failure without a verified consensus proof.
