package cluster

import (
	"context"
	"errors"
)

var (
	// ErrNotLeader is returned when a write operation is attempted on a follower node
	ErrNotLeader = errors.New("node is not the raft leader")
)

// Manager defines the interface for repositories to interact with the cluster
type Manager interface {
	// ApplyWrite replicate a SQL command through the Raft log.
	// If the current node is not the leader, it will return ErrNotLeader.
	// requestID is used for idempotency; if empty, it will be ignored (least safe).
	ApplyWrite(ctx context.Context, requestID string, query string, args ...interface{}) (int64, int64, error)

	// IsLeader returns true if the current node is the Raft leader
	IsLeader() bool

	// LeaderAddr returns the address of the current leader
	LeaderAddr() string
}
