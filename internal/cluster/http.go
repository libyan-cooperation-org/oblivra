package cluster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/raft"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// JoinRequest is the payload sent by a new node to join the cluster
type JoinRequest struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"` // The raft TCP address
}

// Handler provides the HTTP interface for cluster management.
type Handler struct {
	node *Node
	log  *logger.Logger
}

func NewHandler(node *Node, log *logger.Logger) *Handler {
	return &Handler{
		node: node,
		log:  log,
	}
}

// ServeHTTP handles the /join endpoint.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("[RAFT-HTTP] Failed to parse apply request: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	h.log.Info("[RAFT-HTTP] Received join request from Node: %s, Address: %s", req.NodeID, req.Address)

	if !h.node.IsLeader() {
		// In a production system, we would proxy this to the leader via 301 Redirect or internal RPC.
		w.Header().Set("X-Raft-Leader", h.node.LeaderAddr())
		http.Error(w, "not the leader", http.StatusServiceUnavailable)
		return
	}

	// Tell Raft to add a new voter
	cf := h.node.raft.AddVoter(raft.ServerID(req.NodeID), raft.ServerAddress(req.Address), 0, 0)
	if err := cf.Error(); err != nil {
		h.log.Error("[RAFT-HTTP] Failed to add voter %s to cluster: %v", req.NodeID, err)
		http.Error(w, "failed to add node", http.StatusInternalServerError)
		return
	}

	h.log.Info("[RAFT-HTTP] Added voter %s to the cluster successfully", req.NodeID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"joined"}`))
}

// JoinCluster makes an HTTP request to the leader's /join API
func JoinCluster(leaderAddr string, localNodeID string, localRaftAddr string) error {
	reqPayload := JoinRequest{
		NodeID:  localNodeID,
		Address: localRaftAddr,
	}

	p, err := json.Marshal(reqPayload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/join", leaderAddr)
	resp, err := http.Post(url, "application/json", bytes.NewReader(p)) // simplified: real implementation might need TLS and a robust HTTP client
	if err != nil {
		return fmt.Errorf("failed to call join API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("leader rejected join request (status %d)", resp.StatusCode)
	}

	return nil
}
