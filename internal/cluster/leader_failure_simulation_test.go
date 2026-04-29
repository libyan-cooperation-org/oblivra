//go:build cgo

// CGO-only — same constraint as raft_safety_test.go (uses setupMockNode
// which depends on the SQLite CGO driver).
package cluster

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/hashicorp/raft"
)

// TestLeaderFailureIdempotency validates that when a leader fails mid-operation
// and the client retries the same RequestID on a new leader, the FSM correctly
// identifies the duplicate and prevents double-processing.
func TestLeaderFailureIdempotency(t *testing.T) {
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
	defer nodeA.Shutdown()
	nodeB, dbB, _ := setupMockNode(t, "NodeB", transportB)
	defer nodeB.Shutdown()
	nodeC, dbC, _ := setupMockNode(t, "NodeC", transportC)
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

	// 1. Identify initial leader
	var leader *Node
	if nodeA.IsLeader() {
		leader = nodeA
	} else if nodeB.IsLeader() {
		leader = nodeB
	} else if nodeC.IsLeader() {
		leader = nodeC
	}

	if leader == nil {
		t.Fatal("Failed to elect initial leader")
	}

	// 2. Perform a successful write with RequestID
	reqID := "unique-request-123"
	query := "INSERT INTO test_raft (state) VALUES (?)"
	_, _, err := leader.ApplyWrite(context.Background(), reqID, query, "first-attempt")
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// 3. Simulate "Ghost" applied write:
	// In Raft, it's possible for a log to be committed on followers but the leader 
	// fails before observing the commit or replying.
	// We'll simulate this by Manually injecting the SAME RequestID via another node's raft.
	// (Normally this happens via replication, but we want to force the 'Applied' check).
	
	// 4. Force a Leader Change (Kill current leader)
	leader.Shutdown()
	time.Sleep(1 * time.Second) // Wait for re-election

	// 5. Find NEW leader
	var newLeader *Node
	if nodeA.raft.State() == raft.Leader { newLeader = nodeA }
	if nodeB.raft.State() == raft.Leader { newLeader = nodeB }
	if nodeC.raft.State() == raft.Leader { newLeader = nodeC }

	if newLeader == nil {
		t.Fatal("Failed to elect new leader after failure")
	}

	// 6. Retry the SAME RequestID on the NEW leader
	// This simulates a client retrying a "timed out" or "connection reset" request.
	_, _, err = newLeader.ApplyWrite(context.Background(), reqID, query, "retry-attempt")
	if err != nil {
		t.Fatalf("Retry write failed: %v", err)
	}

	// 7. Verify Data Integrity: There should be exactly ONE row with value 'first-attempt'
	// and NO row with 'retry-attempt' (because the second execution was skipped).
	
	count := 0
	err = dbA.QueryRow("SELECT COUNT(*) FROM test_raft").Scan(&count)
	if err != nil { t.Fatalf("query dbA: %v", err) }
	
	if count != 1 {
		t.Errorf("Idempotency FAILURE: Found %d rows in DB, expected 1", count)
	}

	var val string
	err = dbA.QueryRow("SELECT state FROM test_raft LIMIT 1").Scan(&val)
	if err != nil { t.Fatalf("query val: %v", err) }
	
	if val != "first-attempt" {
		t.Errorf("Data corruption: row contains '%s', expected 'first-attempt'", val)
	}

	// 8. Verify Followers also have the same state and NO duplicates
	for _, db := range []*sql.DB{dbB, dbC} {
		var fCount int
		err = db.QueryRow("SELECT COUNT(*) FROM test_raft").Scan(&fCount)
		if err != nil { t.Fatalf("query follower db: %v", err) }
		if fCount != 1 {
			t.Errorf("Follower DB consistency failure: Found %d rows, expected 1", fCount)
		}
	}
}
