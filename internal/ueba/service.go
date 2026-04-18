package ueba

import (
	"context"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"sync"
)

// UEBAService manages behavioral analytics and ML-based anomaly detection.
type UEBAService struct {
	baseline   *BaselineStore
	forest     *IsolationForest
	itdr       *ITDRManager
	peerGroups *PeerGroupManager
	hostRepo   database.HostStore
	bus        *eventbus.Bus
	log        *logger.Logger

	// Control chan for training loop
	stopChan chan struct{}

	anomalies []map[string]interface{}
	anomalyMu  sync.RWMutex
}

func NewUEBAService(hostRepo database.HostStore, bus *eventbus.Bus, store KVStore, log *logger.Logger) *UEBAService {
	baseline := NewBaselineStore(store)
	return &UEBAService{
		baseline:   baseline,
		forest:     NewIsolationForest(100, 256), // 100 trees, 256 subsample
		itdr:       NewITDRManager(baseline),
		peerGroups: NewPeerGroupManager(),
		hostRepo:   hostRepo,
		bus:        bus,
		log:        log.WithPrefix("ueba"),
		stopChan:   make(chan struct{}),
	}
}

func (s *UEBAService) Startup(ctx context.Context) {
	s.log.Info("Starting UEBA service behavioral loop...")

	// Restore profiles from disk
	if err := s.baseline.LoadAll(); err != nil {
		s.log.Error("Failed to restore behavioral baselines: %v", err)
	} else {
		s.log.Info("Restored %d behavioral profiles from hot store", len(s.baseline.GetAllProfiles()))
	}

	// Start background training loop (rebuild forest every hour)
	go s.trainingLoop()
	// Start background persistence loop (flush profiles to disk every 5 minutes)
	go s.persistenceLoop()

	// Subscribe to all security events for real-time profiling
	s.bus.Subscribe(eventbus.EventType("siem.event_indexed"), func(event eventbus.Event) {
		defer func() {
			if r := recover(); r != nil {
				s.log.Debug("Recovered from panic in UEBA process: %v", r)
			}
		}()
		if hEvent, ok := event.Data.(database.HostEvent); ok {
			s.ProcessEvent(&hEvent)
		}
	})
}

func (s *UEBAService) Shutdown() {
	close(s.stopChan)
}

func (s *UEBAService) Name() string {
	return "UEBAService"
}

func (s *UEBAService) ProcessEvent(event *database.HostEvent) {
	// 1. Update behavioral profile
	p := s.baseline.GetOrCreateProfile(event.User, "user")
	hp := s.baseline.GetOrCreateProfile(event.HostID, "host")

	// Assign peer group if not set
	if hp.PeerGroupID == "" {
		host, err := s.hostRepo.GetByID(context.Background(), event.HostID)
		if err == nil && host != nil {
			hp.PeerGroupID = host.Category
			if hp.PeerGroupID == "" {
				hp.PeerGroupID = "default"
			}
		}
	}

	// Extract features (placeholder logic for now)
	p.UpdateFeature("event_frequency", p.FeatureVectors["event_frequency"]+1)
	hp.UpdateFeature("event_frequency", hp.FeatureVectors["event_frequency"]+1)

	// 2. Peer Group Analysis
	var peerRisk float64
	if hp.PeerGroupID != "" {
		group := s.peerGroups.GetOrCreateGroup(hp.PeerGroupID)
		group.Update(hp.FeatureVectors)
		peerRisk = group.GetDeviation(hp.FeatureVectors)
	}

	// 3. Run ITDR heuristics
	itdrScore := s.itdr.AnalyzeEvent(event)

	// 4. Score with Isolation Forest
	anomalyScore := s.forest.Score(p)

	// 5. Aggregate risk
	// Weighting: 40% Anomaly, 30% Peer Deviation, 30% ITDR
	totalRisk := (anomalyScore * 0.4) + (peerRisk * 0.3) + (itdrScore * 0.3)
	if totalRisk > 1.0 {
		totalRisk = 1.0
	}

	p.SetRiskScore(totalRisk)
	hp.SetRiskScore(totalRisk) // Both entity and host share the risk for this event

	// 6. If high risk, publish a security anomaly alert
	if totalRisk > 0.8 {
		evidence := []map[string]interface{}{
			{"key": "anomaly_score", "value": anomalyScore, "threshold": 0.5, "description": "Isolation Forest deviation score"},
			{"key": "peer_risk", "value": peerRisk, "threshold": 0.3, "description": "Deviation from peer group baseline"},
			{"key": "itdr_score", "value": itdrScore, "threshold": 0.3, "description": "Identity Threat detection heuristics"},
		}

		anomaly := map[string]interface{}{
			"entity_id":     event.User,
			"entity_type":   "user",
			"risk_score":    totalRisk,
			"peer_group_id": hp.PeerGroupID,
			"event":         event,
			"evidence":      evidence,
			"timestamp":     time.Now().Format(time.RFC3339),
		}

		s.anomalyMu.Lock()
		s.anomalies = append(s.anomalies, anomaly)
		if len(s.anomalies) > 100 {
			s.anomalies = s.anomalies[1:]
		}
		s.anomalyMu.Unlock()

		s.bus.Publish(eventbus.EventType("siem.anomaly_detected"), anomaly)
	}
}

func (s *UEBAService) trainingLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.log.Info("Retraining Isolation Forest on %d profiles...", len(s.baseline.GetAllProfiles()))
			s.forest.Train(s.baseline.GetAllProfiles())
		case <-s.stopChan:
			return
		}
	}
}

func (s *UEBAService) persistenceLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			profiles := s.baseline.GetAllProfiles()
			if len(profiles) > 0 {
				s.log.Debug("Flushing %d profiles to behavioral hot store...", len(profiles))
				for _, p := range profiles {
					if err := s.baseline.Save(p.ID); err != nil {
						s.log.Error("Failed to persist profile %s: %v", p.ID, err)
					}
				}
			}
		case <-s.stopChan:
			return
		}
	}
}

// GetProfiles returns the current behavioral profiles.
func (s *UEBAService) GetProfiles() []*EntityProfile {
	return s.baseline.GetAllProfiles()
}

// GetAnomalies returns the recent anomalies ring buffer.
func (s *UEBAService) GetAnomalies() []map[string]interface{} {
	s.anomalyMu.RLock()
	defer s.anomalyMu.RUnlock()
	out := make([]map[string]interface{}, len(s.anomalies))
	copy(out, s.anomalies)
	return out
}
