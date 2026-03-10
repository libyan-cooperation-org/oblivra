# Graph-Based Multi-Hop Intrusion Inference in Sovereign Environments

## Abstract
Traditional SIEM solutions often fail to detect sophisticated lateral movement that traverses multiple trust boundaries in isolation. This paper proposes a deterministic graph-based reasoning engine for OBLIVRA that correlates disparate entity behaviors—User, Host, Process, and Credential—into a unified causal graph to infer multi-hop intrusions.

## Core Doctrine: Entity Attribution
Every event ingested into OBLIVRA is mapped to a set of entities:
- **E_h**: Host Entity (ID, Trusted Status)
- **E_u**: User Entity (Identity, Role, MFA Status)
- **E_p**: Process Entity (Hash, Signed Status, Parent)

## Inference Engine Architecture
The inference engine utilizes a compressed adjacency store (BadgerDB-backed) to track edges:
- `Accessed(E_u, E_h)`
- `Spawned(E_h, E_p)`
- `Authenticated(E_u, E_h, E_credential)`

### Path Analysis
Intrusion is inferred by finding the shortest path between a "Compromised Node" (e.g., failed login burst) and a "Critical Asset" (e.g., Vault). 

## Mathematical Foundation
We define the **Intrusion Probability P(I)** as a function of edge density and node anomaly scores:
$P(I) = \sum_{e \in Path} \text{Anomaly}(Node(e)) \times \text{Weight}(Edge(e))$

## Conclusion
By transitioning from row-based log analysis to graph-based inference, OBLIVRA can detect lateral movement even when individual steps appear benign or are heavily deduped.
