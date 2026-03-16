package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kingknull/oblivrashell/internal/cluster"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// RegistryCommand is the operation type stored in the Raft log for plugin state changes.
type RegistryCommand string

const (
	CmdActivate   RegistryCommand = "activate"
	CmdDeactivate RegistryCommand = "deactivate"
)

// RegistryLogEntry is the payload written to the Raft cluster for plugin state mutations.
type RegistryLogEntry struct {
	Op        RegistryCommand `json:"op"`
	PluginID  string          `json:"plugin_id"`
	Timestamp string          `json:"timestamp"`
	NodeID    string          `json:"node_id"`
}

// ClusterRegistry wraps Registry with Raft replication so that plugin activation
// state is consistent across all cluster nodes. A node failure is invisible to
// analysts — any peer that becomes leader already has the correct plugin state.
type ClusterRegistry struct {
	*Registry
	cm     cluster.Manager
	log    *logger.Logger
	nodeID string
}

// NewClusterRegistry creates a cluster-aware plugin registry.
// If cm is nil (single-node mode) it falls back to the local Registry.
func NewClusterRegistry(r *Registry, cm cluster.Manager, nodeID string, log *logger.Logger) *ClusterRegistry {
	return &ClusterRegistry{
		Registry: r,
		cm:       cm,
		log:      log.WithPrefix("cluster-plugin"),
		nodeID:   nodeID,
	}
}

// Activate activates the plugin locally and replicates the state change to all peers.
func (cr *ClusterRegistry) Activate(id string) error {
	// Perform the local activation first — if it fails, don't replicate.
	if err := cr.Registry.Activate(id); err != nil {
		return err
	}

	// Replicate to the cluster so peers synchronise their state.
	if err := cr.replicate(CmdActivate, id); err != nil {
		// Log the replication failure but don't roll back the local activation.
		// The cluster FSM will reconcile on the next snapshot restore.
		cr.log.Warn("[CLUSTER-PLUGIN] Failed to replicate Activate(%s) to cluster: %v — local state preserved", id, err)
	}
	return nil
}

// Deactivate deactivates the plugin locally and replicates the state change.
func (cr *ClusterRegistry) Deactivate(id string) error {
	if err := cr.Registry.Deactivate(id); err != nil {
		return err
	}

	if err := cr.replicate(CmdDeactivate, id); err != nil {
		cr.log.Warn("[CLUSTER-PLUGIN] Failed to replicate Deactivate(%s) to cluster: %v — local state preserved", id, err)
	}
	return nil
}

// ApplyFromCluster is called by the Raft FSM on follower nodes to apply a
// replicated plugin state change without triggering another replication round.
// Signature uses flat args to satisfy the cluster.pluginRegistryApplier interface
// without creating an import cycle.
func (cr *ClusterRegistry) ApplyFromCluster(op, pluginID, timestamp, nodeID string) error {
	switch RegistryCommand(op) {
	case CmdActivate:
		cr.log.Info("[CLUSTER-PLUGIN] Applying replicated Activate(%s) from node %s at %s", pluginID, nodeID, timestamp)
		return cr.Registry.Activate(pluginID)
	case CmdDeactivate:
		cr.log.Info("[CLUSTER-PLUGIN] Applying replicated Deactivate(%s) from node %s at %s", pluginID, nodeID, timestamp)
		return cr.Registry.Deactivate(pluginID)
	default:
		return fmt.Errorf("unknown plugin registry command: %q", op)
	}
}

// replicate writes a plugin state change to the Raft log.
// It is a no-op if no cluster manager is configured (single-node mode).
func (cr *ClusterRegistry) replicate(op RegistryCommand, pluginID string) error {
	if cr.cm == nil {
		return nil // single-node mode — no replication needed
	}

	if !cr.cm.IsLeader() {
		// Forward to leader is handled by the transport layer; log the attempt.
		return fmt.Errorf("%w: cannot replicate plugin state from follower — redirect to %s",
			cluster.ErrNotLeader, cr.cm.LeaderAddr())
	}

	entry := RegistryLogEntry{
		Op:        op,
		PluginID:  pluginID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		NodeID:    cr.nodeID,
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal plugin registry entry: %w", err)
	}

	// Use a dedicated query format so the FSM can distinguish plugin ops from SQL ops.
	query := fmt.Sprintf("--plugin-registry-- %s", string(payload))
	_, _, err = cr.cm.ApplyWrite(context.Background(), query)
	return err
}
