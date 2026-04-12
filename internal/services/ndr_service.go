package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/ndr"
)

// NDRService exposes Network Detection & Response capabilities to the frontend.
type NDRService struct {
	collector       *ndr.FlowCollector
	dnsAnalyzer     *ndr.DNSAnalyzer
	tlsExtractor    *ndr.TLSMetadataExtractor
	lateralEngine   *ndr.LateralMovementEngine
	bus             *eventbus.Bus
	log             *logger.Logger
	ctx             context.Context

	flowHistory []ndr.NetworkFlow
}

func NewNDRService(collector *ndr.FlowCollector, bus *eventbus.Bus, log *logger.Logger) *NDRService {
	return &NDRService{
		collector:     collector,
		dnsAnalyzer:   ndr.NewDNSAnalyzer(bus, log),
		tlsExtractor:  ndr.NewTLSMetadataExtractor(bus, log),
		lateralEngine: ndr.NewLateralMovementEngine(bus, log),
		bus:           bus,
		log:           log.WithPrefix("ndr"),
		flowHistory:   make([]ndr.NetworkFlow, 0),
	}
}

func (s *NDRService) Name() string { return "ndr-service" }

// Dependencies returns service dependencies.
func (s *NDRService) Dependencies() []string {
	return []string{}
}

func (s *NDRService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.log.Info("NDR Service starting...")
	if err := s.collector.Start(ctx); err != nil {
		s.log.Error("Failed to start NDR collector: %v", err)
	}

	// Subscribe to internal flows for multi-hop analysis and sub-analyzers
	s.bus.Subscribe("ndr.flow_captured", func(event eventbus.Event) {
		flow, ok := event.Data.(ndr.NetworkFlow)
		if !ok {
			return
		}

		s.flowHistory = append(s.flowHistory, flow)
		if len(s.flowHistory) > 1000 {
			s.flowHistory = s.flowHistory[1:]
		}

		// Route through dedicated LateralMovementEngine for multi-hop correlation
		s.lateralEngine.ProcessFlow(flow)
		// Legacy inline check retained for compatibility
		s.detectLateralMovement(flow)
		
		if s.ctx != nil {
			EmitEvent("ndr:flow", flow)
		}
	})

	s.bus.Subscribe("ndr.dns_query", func(event eventbus.Event) {
		data, ok := event.Data.(map[string]string)
		if ok {
			s.dnsAnalyzer.ProcessQuery(data["query"], data["answer"])
		}
	})
	return nil
}

func (s *NDRService) detectLateralMovement(flow ndr.NetworkFlow) {
	// Rule: Detect internal port scanning/sweeping (same source to multiple internal dests in short window)
	internalIPPrefix := "192.168." // Simplified for demo
	if !strings.HasPrefix(flow.DestIP, internalIPPrefix) {
		return
	}

	window := 1 * time.Minute
	threshold := 10
	count := 0
	dests := make(map[string]bool)

	for i := len(s.flowHistory) - 1; i >= 0; i-- {
		prev := s.flowHistory[i]
		if time.Since(parseTime(prev.Timestamp)) > window {
			break
		}
		if prev.SourceIP == flow.SourceIP && strings.HasPrefix(prev.DestIP, internalIPPrefix) {
			dests[prev.DestIP] = true
		}
	}

	count = len(dests)
	if count >= threshold {
		s.log.Warn("⚠ Lateral Movement Detected: Source %s contacted %d unique internal hosts in 1m", flow.SourceIP, count)
		s.bus.Publish("siem.alert_fired", map[string]interface{}{
			"type":        "NDR_LATERAL_MOVEMENT",
			"severity":    "CRITICAL",
			"source_ip":   flow.SourceIP,
			"description": fmt.Sprintf("Source host contacted %d unique internal destinations. Potential scanning or lateral movement.", count),
			"evidence": []map[string]interface{}{
				{"key": "unique_destinations", "value": count, "threshold": threshold, "description": "Count of unique internal IP addresses contacted"},
				{"key": "window", "value": window.String(), "description": "Time window for correlation"},
			},
		})
	}
}

func (s *NDRService) Stop(ctx context.Context) error {
	s.log.Info("NDR Service shutting down...")
	return nil
}

// GetLiveTraffic returns a snapshot of recent network flows.
func (s *NDRService) GetLiveTraffic() ([]ndr.NetworkFlow, error) {
	// Return the in-memory cache of recent flows
	// In production, this would query the HotStore or an in-memory cache of recent flows.
	return s.flowHistory, nil
}


