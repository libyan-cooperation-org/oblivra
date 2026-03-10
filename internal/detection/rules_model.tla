--------------------------- MODULE rules_model ---------------------------
EXTENDS Integers, Sequences, FiniteSets, TLC

(* 
  Model of the OBLIVRA Detection Engine Sequence Logic.
  Captures the state transitions in Evaluator.evaluateRuleState for SequenceRules.
*)

CONSTANTS
    Steps,      \* Set of possible step identifiers (e.g., {"fail1", "fail2", "success"})
    WindowSize, \* Maximum time difference allowed between first and last event in a sequence
    MaxEvents   \* To keep the model finite, we limit total ingested events

VARIABLES
    history,    \* Sequence of events: << [id: Steps, time: Int], ... >>
    state,      \* Current sequence progress: << Event, ... >>
    alerts      \* Set of completed sequences (matches)

Vars == <<history, state, alerts>>

(* Helpers *)
LastTime(seq) == IF Len(seq) = 0 THEN 0 ELSE seq[Len(seq)].time

(* Check if event matches the NEXT step in the sequence logic *)
MatchesNext(evt, current_state) ==
    /\ Len(current_state) < Len(Steps)
    /\ evt.id = Steps[Len(current_state) + 1]

(* Check if event restarts the sequence *)
Restarts(evt) ==
    evt.id = Steps[1]

-----------------------------------------------------------------------------

Init ==
    /\ history = << >>
    /\ state = << >>
    /\ alerts = {}

(* 
  Ingest a new event and update the rule state.
  Mirrors Evaluator.evaluateRuleState logic.
*)
Ingest(evt) ==
    LET 
        \* Purge events older than WindowSize relative to the current event
        window_cutoff == evt.time - WindowSize
        purged_state == SelectSeq(state, LAMBDA e: e.time > window_cutoff)
    IN
        IF MatchesNext(evt, purged_state) THEN
            LET new_state == Append(purged_state, evt) IN
            /\ state' = new_state
            /\ alerts' = IF Len(new_state) = Len(Steps) 
                         THEN alerts \cup {new_state} 
                         ELSE alerts
        ELSE IF Restarts(evt) THEN
            /\ state' = <<evt>>
            /\ alerts' = alerts
        ELSE
            /\ state' = purged_state
            /\ alerts' = alerts

NextEvent ==
    /\ Len(history) < MaxEvents
    /\ \E s \in Steps, t \in 1..20:
        /\ (t >= LastTime(history)) \* Monotonic time
        /\ LET evt == [id |-> s, time |-> t] IN
           /\ history' = Append(history, evt)
           /\ Ingest(evt)

Next == NextEvent

-----------------------------------------------------------------------------

(* 
  Invariant: No Spurious Alerts.
  Every alert in the set MUST be a strictly increasing sequence of the defined Steps.
*)
NoSpuriousAlerts ==
    \A a \in alerts:
        /\ Len(a) = Len(Steps)
        /\ \A i \in 1..Len(Steps): a[i].id = Steps[i]
        /\ \A i \in 1..(Len(Steps)-1): a[i].time <= a[i+1].time

(* 
  Invariant: Window Enforcement.
  Events in the state must never span more than WindowSize.
*)
WindowStateInvariant ==
    IF Len(state) > 1 
    THEN state[Len(state)].time - state[1].time <= WindowSize
    ELSE TRUE

=============================================================================
