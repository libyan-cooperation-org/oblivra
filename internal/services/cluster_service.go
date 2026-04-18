package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kingknull/oblivrashell/internal/cluster"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/platform"
	"github.com/kingknull/oblivrashell/internal/search"
)

type ClusterService struct {
	BaseService
	db     database.DatabaseStore
	bus    *eventbus.Bus
	log    *logger.Logger
	server *http.Server
	federator *search.Federator
	ctx    context.Context
}

func NewClusterService(db database.DatabaseStore, bus *eventbus.Bus, log *logger.Logger, federator *search.Federator) *ClusterService {
	return &ClusterService{
		db:        db,
		bus:       bus,
		log:       log.WithPrefix("cluster"),
		federator: federator,
	}
}

func (s *ClusterService) Name() string { return "cluster-service" }

// Dependencies returns service dependencies
func (s *ClusterService) Dependencies() []string {
	return []string{"eventbus", "database"}
}

func (s *ClusterService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.bus.Subscribe(eventbus.EventVaultUnlocked, s.onVaultUnlocked)
	return nil
}

func (s *ClusterService) onVaultUnlocked(event eventbus.Event) {
	raftID := os.Getenv("OBLIVRA_RAFT_ID")
	if raftID == "" {
		s.log.Debug("OBLIVRA_RAFT_ID not set, skipping cluster initialization")
		return
	}

	raftBind := os.Getenv("OBLIVRA_RAFT_BIND")
	if raftBind == "" {
		raftBind = "127.0.0.1:15300" // Default bind address
	}
	raftJoin := os.Getenv("OBLIVRA_RAFT_JOIN")

	raftDir := os.Getenv("OBLIVRA_RAFT_DIR")
	if raftDir == "" {
		raftDir = filepath.Join(platform.DataDir(), "raft")
	}

	httpPort := os.Getenv("OBLIVRA_RAFT_HTTP_PORT")
	if httpPort == "" {
		httpPort = "15400"
	}

	s.log.Info("Initializing Raft cluster node (ID: %s, Bind: %s)", raftID, raftBind)

	dbImpl, ok := s.db.(*database.Database)
	if !ok {
		s.log.Error("Database is not of expected type *database.Database")
		return
	}

	cfg := cluster.Config{
		NodeID:   raftID,
		BindAddr: raftBind,
		BaseDir:  raftDir,
		JoinAddr: raftJoin,
	}

	cm, err := cluster.NewNode(cfg, dbImpl.DB(), s.log)
	if err != nil {
		s.log.Error("Failed to initialize Raft node: %v", err)
		return
	}

	s.db.SetClusterManager(cm)
	s.log.Info("Cluster Manager attached to database")

	// Start HTTP server for /join endpoint
	mux := http.NewServeMux()
	mux.Handle("/join", cluster.NewHandler(cm, s.log))

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", httpPort),
		Handler: mux,
	}

	go func() {
		s.log.Info("Starting Raft HTTP server on :%s", httpPort)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("Raft HTTP server error: %v", err)
		}
	}()

	// If join address is provided, try to join the cluster
	if raftJoin != "" && raftJoin != raftBind {
		go func() {
			time.Sleep(2 * time.Second) // Wait for node to stabilize
			s.log.Info("Attempting to join cluster at %s", raftJoin)
			// Assuming JoinCluster requires 3 arguments: joinAddr, nodeID, bindAddr
			err := cluster.JoinCluster(raftJoin, raftID, raftBind)
			if err != nil {
				s.log.Error("Failed to join cluster: %v", err)
			} else {
				s.log.Info("Successfully joined cluster")
			}
		}()
	}

	// Start node synchronization loop for search federation
	go s.syncNodesLoop(s.ctx)
}

func (s *ClusterService) syncNodesLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncNodesWithFederator()
		}
	}
}

func (s *ClusterService) syncNodesWithFederator() {
	if s.federator == nil {
		return
	}

	cm := s.db.ClusterManager()
	if cm == nil {
		return
	}

	nodes := cm.Nodes()
	for _, nodeAddr := range nodes {
		// Convert Raft address (e.g. :15300) to API address (e.g. :8080)
		// For MVP, we assume they are on the same host and main API is 8080.
		// A more robust way would be to broadcast the API port via Raft FSM.
		host, _, err := net.SplitHostPort(nodeAddr)
		if err != nil {
			host = nodeAddr // Fallback
		}
		
		apiAddr := fmt.Sprintf("http://%s:8080", host)
		// s.federator.AddPeer(nodeAddr, apiAddr)
		// TODO: Implement a method to set/refresh peers instead of just AddPeer (to avoid duplicates)
		s.federator.AddPeer(nodeAddr, apiAddr)
	}
}

func (s *ClusterService) Stop(ctx context.Context) error {
	if s.server != nil {
		s.log.Info("Shutting down Raft HTTP server")
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			s.log.Error("Failed to cleanly shutdown Raft HTTP server: %v", err)
		}
	}
	return nil
}
