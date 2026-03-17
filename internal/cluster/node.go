package cluster

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type Config struct {
	NodeID   string
	BindAddr string
	BaseDir  string
	JoinAddr string // Non-empty if joining an existing cluster
}

type Node struct {
	raft *raft.Raft
	log  *logger.Logger
}

func NewNode(cfg Config, db *sql.DB, log *logger.Logger) (*Node, error) {
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(cfg.NodeID)

	// In testing we might want lower timeouts, but we leave defaults for now.
	// We require at least one other node to commit if we are in a cluster, but if we are starting standalone:
	// We will bootstrap immediately if JoinAddr is empty.

	logStorePath := filepath.Join(cfg.BaseDir, "raft-log.db")
	stableStorePath := filepath.Join(cfg.BaseDir, "raft-stable.db")
	snapshotStorePath := filepath.Join(cfg.BaseDir, "raft-snapshots")

	if err := os.MkdirAll(snapshotStorePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create snapshot dir: %w", err)
	}

	logStore, err := raftboltdb.NewBoltStore(logStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log store: %w", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(stableStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create stable store: %w", err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(snapshotStorePath, 2, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot store: %w", err)
	}

	addr, err := net.ResolveTCPAddr("tcp", cfg.BindAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve TCP address: %w", err)
	}

	transport, err := raft.NewTCPTransport(cfg.BindAddr, addr, 3, 10*time.Second, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP transport: %w", err)
	}

	dbPath := filepath.Join(cfg.BaseDir, "oblivra.db")
	fsm := NewFSM(db, dbPath, log)

	r, err := raft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create raft node: %w", err)
	}

	hasState, err := raft.HasExistingState(logStore, stableStore, snapshotStore)
	if err != nil {
		return nil, fmt.Errorf("check existing raft state: %w", err)
	}

	// Bootstrap cluster if we are starting a brand new one and not joining and no state exists
	if cfg.JoinAddr == "" && !hasState {
		cfg := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raftConfig.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		r.BootstrapCluster(cfg)
		log.Info("[RAFT] Bootstrapped new single-node cluster (Leader: %s)", raftConfig.LocalID)
	}

	return &Node{
		raft: r,
		log:  log,
	}, nil
}

func (n *Node) IsLeader() bool {
	return n.raft.State() == raft.Leader
}

func (n *Node) LeaderAddr() string {
	leaderAddr, _ := n.raft.LeaderWithID()
	return string(leaderAddr)
}

func (n *Node) ApplyWrite(ctx context.Context, query string, args ...interface{}) (int64, int64, error) {
	if !n.IsLeader() {
		return 0, 0, ErrNotLeader
	}

	cmd := SQLWriteCommand{
		Query: query,
		Args:  args,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return 0, 0, fmt.Errorf("marshal raft command: %w", err)
	}

	af := n.raft.Apply(data, 5*time.Second)
	if err := af.Error(); err != nil {
		return 0, 0, fmt.Errorf("raft apply: %w", err)
	}

	// Retrieve the response from the FSM
	if resp := af.Response(); resp != nil {
		if applyResp, ok := resp.(FSMApplyResponse); ok {
			return applyResp.LastInsertId, applyResp.RowsAffected, applyResp.Err
		}
	}

	return 0, 0, nil
}
func (n *Node) Shutdown() error {
	future := n.raft.Shutdown()
	if err := future.Error(); err != nil {
		return fmt.Errorf("raft shutdown: %w", err)
	}
	return nil
}
