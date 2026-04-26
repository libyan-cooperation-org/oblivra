package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// StateReplicator is the typed wrapper around (*Node).ApplyWrite for
// the three control-plane state classes called out in Phase 27.2.4:
//
//   1. Alert state         (open/ack/closed/suppressed transitions)
//   2. Playbook definitions (yaml-shaped DSL the SOAR engine consumes)
//   3. Threat intel indicators (TI feed snapshot rows)
//
// Why this layer exists:
//
// `Node.ApplyWrite` already replicates raw SQL writes through the
// Raft log. Without a typed wrapper, every call site had to:
//   (a) hand-roll SQL strings,
//   (b) generate a deterministic request-ID for idempotency,
//   (c) decide what to do when the local node isn't the leader.
//
// StateReplicator centralizes (a)-(c) so:
//   - Adding a new replicated type is one method on this struct.
//   - Single-node deployments where there's no Raft fall back to a
//     `LocalApplier` (typically a *sql.DB-backed shim) — same SQL,
//     no consensus round trip.
//   - Followers that receive a write request return ErrNotLeader,
//     which the caller can handle by forwarding to the leader.
//
// The wrapper does NOT carry domain logic (state-transition rules,
// validation, etc.) — those live in `internal/services`. This is
// strictly the "how to durably persist + replicate" plumbing.

// LocalApplier is the fallback execution path for single-node mode.
// It must accept the same SQL+args shape that the Raft FSM applies
// on followers, so the on-disk state is identical regardless of
// which path was taken.
type LocalApplier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (LocalExecResult, error)
}

// LocalExecResult mirrors sql.Result so callers can read affected
// row counts. Keeping the interface minimal makes single-node
// deployments easy to satisfy.
type LocalExecResult interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// ErrNoBackend is returned when a replicator has neither a Raft node
// nor a LocalApplier configured. Callers should fail loudly — silent
// data loss is worse than a hard error.
var ErrNoBackend = errors.New("cluster.StateReplicator: no Raft node and no LocalApplier configured")

// ErrNotLeaderForward signals that the caller should retry the
// request against the current Raft leader. Surfaces only in clustered
// mode; single-node never returns this.
var ErrNotLeaderForward = errors.New("cluster.StateReplicator: not leader, retry on leader node")

// StateReplicator persists alert/playbook/TI state through Raft when
// available, and through a local DB shim otherwise.
type StateReplicator struct {
	node    *Node
	local   LocalApplier
}

// NewStateReplicator returns a replicator that prefers Raft when
// `node` is non-nil. `local` is required so single-node deployments
// have a backing store. Either argument may be nil — passing both nil
// turns every Apply* into ErrNoBackend.
func NewStateReplicator(node *Node, local LocalApplier) *StateReplicator {
	return &StateReplicator{node: node, local: local}
}

// applyOrLocal is the shared path used by every typed Apply*
// helper. Generates an idempotent request ID derived from the SQL
// and the bind values so retries from the caller don't double-apply.
func (r *StateReplicator) applyOrLocal(
	ctx context.Context,
	scope, key, query string,
	args ...interface{},
) error {
	requestID := r.deriveRequestID(scope, key, query, args)

	// Cluster path: replicate via Raft when we have a leader handle.
	if r.node != nil {
		if !r.node.IsLeader() {
			return ErrNotLeaderForward
		}
		_, _, err := r.node.ApplyWrite(ctx, requestID, query, args...)
		return err
	}

	// Single-node path: write straight to the local DB.
	if r.local != nil {
		_, err := r.local.ExecContext(ctx, query, args...)
		return err
	}
	return ErrNoBackend
}

// deriveRequestID hashes (scope, key, query, args) so the same
// logical write produces the same request ID across retries. The
// FSM's `_raft_applied` table dedupes on this, so callers retrying
// after a transient leader-election won't double-apply.
func (r *StateReplicator) deriveRequestID(scope, key, query string, args []interface{}) string {
	h := sha256.New()
	h.Write([]byte(scope))
	h.Write([]byte{0x1f})
	h.Write([]byte(key))
	h.Write([]byte{0x1f})
	h.Write([]byte(query))
	for _, a := range args {
		h.Write([]byte{0x1f})
		// JSON-encode args for stable hashing across types.
		if b, err := json.Marshal(a); err == nil {
			h.Write(b)
		}
	}
	return scope + ":" + hex.EncodeToString(h.Sum(nil)[:16])
}

// ── Typed helpers ─────────────────────────────────────────────────────

// AlertState carries the fields a SOAR/SIEM alert update writes
// through Raft. The StateReplicator INSERT OR REPLACEs into the
// `alerts` table — schema lives in the main database migrations
// graph, not here.
type AlertState struct {
	ID          string    // primary key
	TenantID    string
	Name        string
	Severity    string
	Status      string    // open / acknowledged / investigating / closed / suppressed
	Host        string
	RawLog      string
	Description string
	UpdatedAt   time.Time
}

// ApplyAlertState replicates an alert's current state through Raft.
// Idempotent: the same (id, status, updated_at) tuple yields the
// same request ID, so a retry won't double-apply.
func (r *StateReplicator) ApplyAlertState(ctx context.Context, a AlertState) error {
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = time.Now().UTC()
	}
	const q = `
		INSERT OR REPLACE INTO alerts
			(id, tenant_id, name, severity, status, host, raw_log, description, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	return r.applyOrLocal(ctx,
		"alert", fmt.Sprintf("%s/%s", a.TenantID, a.ID), q,
		a.ID, a.TenantID, a.Name, a.Severity, a.Status, a.Host, a.RawLog,
		a.Description, a.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
}

// PlaybookDefinition carries the SOAR playbook payload to be replicated.
type PlaybookDefinition struct {
	ID        string // primary key
	TenantID  string
	Name      string
	Author    string
	Version   int
	Body      string // YAML (or JSON) DSL
	Enabled   bool
	UpdatedAt time.Time
}

// ApplyPlaybook replicates a playbook definition through Raft.
func (r *StateReplicator) ApplyPlaybook(ctx context.Context, p PlaybookDefinition) error {
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now().UTC()
	}
	enabled := 0
	if p.Enabled {
		enabled = 1
	}
	const q = `
		INSERT OR REPLACE INTO playbooks
			(id, tenant_id, name, author, version, body, enabled, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	return r.applyOrLocal(ctx,
		"playbook", fmt.Sprintf("%s/%s/v%d", p.TenantID, p.ID, p.Version), q,
		p.ID, p.TenantID, p.Name, p.Author, p.Version, p.Body, enabled,
		p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
}

// ThreatIntelIndicator carries one row of a TI feed snapshot.
type ThreatIntelIndicator struct {
	Value      string // canonical IOC (ip, domain, hash, etc.)
	Type       string // ip / domain / hash / url
	Source     string // feed name
	Severity   string // low / medium / high / critical
	CampaignID string
	FirstSeen  time.Time
	LastSeen   time.Time
}

// ApplyThreatIntel replicates a single indicator through Raft.
// Callers replicating large feed snapshots should batch via the
// caller's own transaction (this layer is per-row by design).
func (r *StateReplicator) ApplyThreatIntel(ctx context.Context, i ThreatIntelIndicator) error {
	if i.LastSeen.IsZero() {
		i.LastSeen = time.Now().UTC()
	}
	const q = `
		INSERT OR REPLACE INTO threat_intel_indicators
			(value, type, source, severity, campaign_id, first_seen, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	return r.applyOrLocal(ctx,
		"ti", fmt.Sprintf("%s/%s", i.Source, i.Value), q,
		i.Value, i.Type, i.Source, i.Severity, i.CampaignID,
		i.FirstSeen.UTC().Format(time.RFC3339Nano),
		i.LastSeen.UTC().Format(time.RFC3339Nano),
	)
}
