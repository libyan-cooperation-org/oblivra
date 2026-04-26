// In-memory queue of agent pending actions.
//
// Phase 31 — Pass E: closes the "remote control RPCs" gap from the
// agent feature audit. The original `handleAgentAction` was a stub
// that logged the request and returned OK without actually delivering
// anything to the agent. This file provides the queue + the dequeue
// path consumed by the agent's heartbeat flush.
//
// Why in-memory rather than SQLite:
//   - Pending actions are short-lived (delivered on the agent's next
//     heartbeat, ~10s). The window of "queued but undelivered" is
//     measured in seconds, not minutes — durability across restarts
//     isn't worth the schema cost.
//   - The cluster_state replicator (Phase 27.2.4) handles long-lived
//     state. Pending actions are inherently ephemeral.
//   - Server restart drops all queued actions; the operator simply
//     re-issues from the UI. Acceptable.
//
// Future hardening (tracked in task.md, not blocking):
//   - Move to SQLite-backed queue for guaranteed delivery across
//     restarts.
//   - Add per-agent rate limiting so a single misbehaving operator
//     can't flood an agent with actions.
//   - Wire into Raft state replicator so HA control planes see
//     consistent action queues.

package api

import (
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/agent"
)

// queuedAction wraps a pending action with metadata used by the
// queue's TTL sweeper.
type queuedAction struct {
	action   agent.PendingAction
	queuedAt time.Time
}

// agentActionsQueue is a minimal per-agent pending-actions queue.
// Operations are guarded by a single mutex — the volume of agent
// control actions is low (operator-initiated, not automated) so
// finer-grained locking isn't worth the complexity.
type agentActionsQueue struct {
	mu      sync.Mutex
	pending map[string][]queuedAction // agent_id → FIFO queue
	ttl     time.Duration             // actions older than this are dropped on dequeue
}

// newAgentActionsQueue constructs an empty queue with a 5-minute TTL.
// Five minutes is long enough to outlast any plausible agent
// heartbeat outage (heartbeats run every ~10s) but short enough that
// a misclicked action doesn't fire after the operator has moved on.
func newAgentActionsQueue() *agentActionsQueue {
	return &agentActionsQueue{
		pending: make(map[string][]queuedAction),
		ttl:     5 * time.Minute,
	}
}

// Enqueue appends an action to the named agent's pending list.
func (q *agentActionsQueue) Enqueue(agentID string, a agent.PendingAction) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pending[agentID] = append(q.pending[agentID], queuedAction{
		action:   a,
		queuedAt: time.Now(),
	})
}

// Dequeue returns and removes every pending action for the given
// agent. Expired entries (older than ttl) are silently dropped.
// Empty result is normal — the heartbeat handler always calls this
// regardless of whether anything's queued.
func (q *agentActionsQueue) Dequeue(agentID string) []agent.PendingAction {
	q.mu.Lock()
	defer q.mu.Unlock()
	queue, ok := q.pending[agentID]
	if !ok || len(queue) == 0 {
		return nil
	}
	cutoff := time.Now().Add(-q.ttl)
	out := make([]agent.PendingAction, 0, len(queue))
	for _, qa := range queue {
		if qa.queuedAt.After(cutoff) {
			out = append(out, qa.action)
		}
	}
	delete(q.pending, agentID)
	return out
}

// Pending returns the count of queued actions across all agents.
// Used by the dashboard's queue-depth widget; non-blocking snapshot.
func (q *agentActionsQueue) Pending() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	total := 0
	for _, queue := range q.pending {
		total += len(queue)
	}
	return total
}
