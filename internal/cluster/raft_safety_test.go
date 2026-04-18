package cluster

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/kingknull/oblivrashell/internal/logger"
	_ "github.com/mattn/go-sqlite3"
)

// setupMockNode provisions an isolated SQLite-backed Raft node connected to an In-Memory Transport
func setupMockNode(t *testing.T, id string, transport *raft.InmemTransport) (*Node, *sql.DB, string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, fmt.Sprintf("oblivra_%s.db", id))

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("failed to open sqlite for %s: %v", id, err)
	}

	// Create a table for replication tests
	_, err = db.Exec("CREATE TABLE test_raft (id INTEGER PRIMARY KEY, state TEXT)")
	if err != nil {
		t.Fatalf("failed to init schema logic for %s: %v", id, err)
	}

	log, err := logger.New(logger.Config{
		Level:      logger.InfoLevel,
		OutputPath: filepath.Join(tempDir, "test.log"),
		JSON:       false,
	})
	if err != nil {
		t.Fatalf("failed to init logger for %s: %v", id, err)
	}
	fsm := NewFSM(db, dbPath, log)

	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(id)
	// Speed up testing elections
	raftConfig.ElectionTimeout = 200 * time.Millisecond
	raftConfig.HeartbeatTimeout = 50 * time.Millisecond
	raftConfig.LeaderLeaseTimeout = 50 * time.Millisecond
	raftConfig.CommitTimeout = 5 * time.Millisecond

	logStore := raft.NewInmemStore()
	stableStore := raft.NewInmemStore()
	snapshotStore := raft.NewDiscardSnapshotStore()

	r, err := raft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		t.Fatalf("failed to initialize raft for %s: %v", id, err)
	}

	return &Node{
		raft: r,
		log:  log,
	}, db, tempDir
}

// TestRaftSplitBrain validates that during a Network Partition, the minority partition
// drops leadership and rejects writes, while the majority partition elects a new leader and continues.
func TestRaftSplitBrain(t *testing.T) {
	addrA, addrB, addrC := raft.ServerAddress("NodeA"), raft.ServerAddress("NodeB"), raft.ServerAddress("NodeC")

	_, transportA := raft.NewInmemTransport(addrA)
	_, transportB := raft.NewInmemTransport(addrB)
	_, transportC := raft.NewInmemTransport(addrC)

	// Interconnect all
	transportA.Connect(addrB, transportB)
	transportA.Connect(addrC, transportC)
	transportB.Connect(addrA, transportA)
	transportB.Connect(addrC, transportC)
	transportC.Connect(addrA, transportA)
	transportC.Connect(addrB, transportB)

	nodeA, dbA, _ := setupMockNode(t, "NodeA", transportA)
	defer dbA.Close()
	defer nodeA.Shutdown()
	nodeB, dbB, _ := setupMockNode(t, "NodeB", transportB)
	defer dbB.Close()
	defer nodeB.Shutdown()
	nodeC, dbC, _ := setupMockNode(t, "NodeC", transportC)
	defer dbC.Close()
	defer nodeC.Shutdown()

	// Bootstrap cluster
	configuration := raft.Configuration{
		Servers: []raft.Server{
			{ID: raft.ServerID("NodeA"), Address: addrA},
			{ID: raft.ServerID("NodeB"), Address: addrB},
			{ID: raft.ServerID("NodeC"), Address: addrC},
		},
	}

	nodeA.raft.BootstrapCluster(configuration)

	// Wait for election
	time.Sleep(1 * time.Second)

	// Find the initial leader
	var leader *Node
	var leaderAddr raft.ServerAddress
	if nodeA.IsLeader() {
		leader, leaderAddr = nodeA, addrA
	} else if nodeB.IsLeader() {
		leader, leaderAddr = nodeB, addrB
	} else if nodeC.IsLeader() {
		leader, leaderAddr = nodeC, addrC
	}

	if leader == nil {
		t.Fatal("Failed to elect a leader during normal operations")
	}

	// 1. Simulate Split Brain (Partition the Leader from the Followers)
	// We disconnect the leader from both followers
	var followers []*raft.InmemTransport
	var fAddrs []raft.ServerAddress

	if leaderAddr == addrA {
		transportA.DisconnectAll()
		followers = append(followers, transportB, transportC)
		fAddrs = append(fAddrs, addrB, addrC)
	} else if leaderAddr == addrB {
		transportB.DisconnectAll()
		followers = append(followers, transportA, transportC)
		fAddrs = append(fAddrs, addrA, addrC)
	} else {
		transportC.DisconnectAll()
		followers = append(followers, transportA, transportB)
		fAddrs = append(fAddrs, addrA, addrB)
	}

	// 2. Wait for followers to realize the leader is gone and start absolute split vote
	time.Sleep(1 * time.Second)

	// Verify the isolated leader lost its state
	if leader.IsLeader() {
		t.Fatal("Isolated minority leader failed to step down")
	}

	// Verify the isolated leader CANNOT accept writes
	_, _, err := leader.ApplyWrite(context.Background(), "test-req-1", "INSERT INTO test_raft (state) VALUES (?)", "isolated_write")
	if err == nil {
		t.Fatal("Isolated minority leader accepted a write that could corrupt DB state")
	}

	// Verify the majority partition elected a NEW leader
	newLeaderElected := false
	if followers[0].LocalAddr() == addrA || followers[1].LocalAddr() == addrA {
		if nodeA.IsLeader() { newLeaderElected = true }
	}
	if followers[0].LocalAddr() == addrB || followers[1].LocalAddr() == addrB {
		if nodeB.IsLeader() { newLeaderElected = true }
	}
	if followers[0].LocalAddr() == addrC || followers[1].LocalAddr() == addrC {
		if nodeC.IsLeader() { newLeaderElected = true }
	}

	if !newLeaderElected {
		t.Fatal("Majority partition failed to elect a new leader during split-brain")
	}
}
