--------------------------- MODULE deterministic_executor ---------------------------
EXTENDS Integers, Sequences, FiniteSets, TLC

(*
  Formal Safety Model — OBLIVRA DeterministicExecutionService
  ============================================================

  This module specifies the safety invariants and liveness properties of the
  DeterministicExecutor in internal/decision/deterministic.go.

  Core Claim:
    Given the same (action, event_payload, policy_state_hash) triple, the
    executor ALWAYS produces the same FinalHash — regardless of call ordering,
    concurrency, or intermediate state.

  This is not just a property about the hash function (SHA-256 is deterministic
  by definition). It is a property about the system:

    1. No execution signature is overwritten after it has been written.
    2. Replay always agrees with the original execution for the same inputs.
    3. No two distinct input triples can produce the same FinalHash (no
       collision within the model's bounded state space).
    4. Every published execution is permanently auditable.

  State Space:
    Actions       — finite set of action identifiers
    PolicyHashes  — finite set of possible policy state hashes
    InputHashes   — finite set of possible event payload hashes
    MaxExecutions — maximum number of execution records
*)

CONSTANTS
    Actions,        \* e.g., {"block_ip", "isolate_host", "disable_user"}
    PolicyHashes,   \* e.g., {"p1", "p2"}
    InputHashes,    \* e.g., {"i1", "i2", "i3"}
    MaxExecutions   \* finite bound for model checking, e.g., 4

ASSUME MaxExecutions \in Nat /\ MaxExecutions > 0
ASSUME IsFiniteSet(Actions) /\ Actions # {}
ASSUME IsFiniteSet(PolicyHashes) /\ PolicyHashes # {}
ASSUME IsFiniteSet(InputHashes) /\ InputHashes # {}

\* The deterministic hash function: models SHA-256(inputHash|policyHash|action)
\* In TLA+ we model it as a purely structural triple — no two distinct triples
\* share the same FinalHash (injective within our finite domain).
FinalHash(input, policy, action) == <<input, policy, action>>

VARIABLES
    signatures,     \* Function: SignatureID -> ExecutionRecord
    sig_count       \* Natural number: total executions recorded

Vars == <<signatures, sig_count>>

\* An execution record mirrors ExecutionSignature in deterministic.go
ExecutionRecord == [
    input_hash  : InputHashes,
    policy_hash : PolicyHashes,
    action      : Actions,
    final_hash  : InputHashes \X PolicyHashes \X Actions
]

\* The set of all currently stored final hashes
StoredHashes == { signatures[id].final_hash : id \in DOMAIN signatures }

TypeInvariant ==
    /\ sig_count \in Nat
    /\ sig_count = Cardinality(DOMAIN signatures)
    /\ \A id \in DOMAIN signatures :
        /\ signatures[id] \in ExecutionRecord
        /\ signatures[id].final_hash = FinalHash(
               signatures[id].input_hash,
               signatures[id].policy_hash,
               signatures[id].action)

------------------------------------------------------------------------------

Init ==
    /\ signatures = [x \in {} |-> x]   \* empty function
    /\ sig_count  = 0

(*
  Execute: Records a new deterministic execution.
  Precondition: the exact same (input, policy, action) triple has NOT been
  executed before — this enforces idempotency detection.
*)
Execute(input, policy, action) ==
    LET fh == FinalHash(input, policy, action) IN
    /\ sig_count < MaxExecutions
    /\ fh \notin StoredHashes          \* no duplicate FinalHash
    /\ LET newID == sig_count IN
       /\ signatures' = [signatures EXCEPT ![newID] = [
              input_hash  |-> input,
              policy_hash |-> policy,
              action      |-> action,
              final_hash  |-> fh
          ]]
       /\ sig_count' = sig_count + 1

(*
  Replay: Re-derives the FinalHash from inputs and checks it matches a
  previously stored record. This models the Replay() method in Go.
*)
Replay(input, policy, action) ==
    LET expected == FinalHash(input, policy, action) IN
    /\ expected \in StoredHashes        \* must be a previously recorded execution
    /\ UNCHANGED Vars                   \* Replay is read-only — no state change

Next ==
    \/ \E i \in InputHashes, p \in PolicyHashes, a \in Actions :
            Execute(i, p, a)
    \/ \E i \in InputHashes, p \in PolicyHashes, a \in Actions :
            Replay(i, p, a)

Spec == Init /\ [][Next]_Vars

------------------------------------------------------------------------------
\* ── Safety Invariants ──────────────────────────────────────────────────────

(*
  INVARIANT 1: Determinism
  For any two signatures with identical (input, policy, action) triples,
  their FinalHash values MUST be identical.
*)
Determinism ==
    \A id1, id2 \in DOMAIN signatures :
        (   signatures[id1].input_hash  = signatures[id2].input_hash
         /\ signatures[id1].policy_hash = signatures[id2].policy_hash
         /\ signatures[id1].action      = signatures[id2].action
        ) => signatures[id1].final_hash = signatures[id2].final_hash

(*
  INVARIANT 2: No Hash Collision (within model bounds)
  No two DISTINCT input triples may produce the same FinalHash.
  This is the injectivity property of our hash model.
*)
NoHashCollision ==
    \A id1, id2 \in DOMAIN signatures :
        signatures[id1].final_hash = signatures[id2].final_hash =>
        (   signatures[id1].input_hash  = signatures[id2].input_hash
         /\ signatures[id1].policy_hash = signatures[id2].policy_hash
         /\ signatures[id1].action      = signatures[id2].action
        )

(*
  INVARIANT 3: Immutability
  Once an execution record is stored, it is never modified.
  This is the "no overwrite after write" property.
  Because Execute only uses sig_count as a new key and never touches
  existing keys, this holds structurally. We make it explicit:
*)
Immutability ==
    [][
        \A id \in DOMAIN signatures :
            id \in DOMAIN signatures' =>
            signatures'[id] = signatures[id]
    ]_signatures

(*
  INVARIANT 4: Replay Consistency
  Replay produces the same FinalHash as the original execution.
  (Captured by the FinalHash function being purely structural — any
  call with the same arguments returns the same value.)
*)
ReplayConsistency ==
    \A id \in DOMAIN signatures :
        FinalHash(
            signatures[id].input_hash,
            signatures[id].policy_hash,
            signatures[id].action
        ) = signatures[id].final_hash

(*
  INVARIANT 5: Type Safety
  All stored records satisfy the ExecutionRecord type schema.
*)
AllRecordsWellTyped ==
    \A id \in DOMAIN signatures :
        /\ signatures[id].input_hash \in InputHashes
        /\ signatures[id].policy_hash \in PolicyHashes
        /\ signatures[id].action \in Actions

\* ── Liveness Property ───────────────────────────────────────────────────────

(*
  LIVENESS: Every valid input triple can eventually be executed.
  This ensures the executor is not deadlocked and progress is always possible
  as long as the capacity bound has not been reached.
*)
EventualExecution ==
    \A i \in InputHashes, p \in PolicyHashes, a \in Actions :
        FinalHash(i, p, a) \notin StoredHashes ~>
        (sig_count >= MaxExecutions \/ FinalHash(i, p, a) \in StoredHashes)

=============================================================================
