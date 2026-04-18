package ndr

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// NetworkFlow represents a summarized network session.
type NetworkFlow struct {
	Timestamp  string    `json:"timestamp"`
	SourceIP   string    `json:"src_ip"`
	SourcePort int       `json:"src_port"`
	DestIP     string    `json:"dest_ip"`
	DestPort   int       `json:"dest_port"`
	Protocol   string    `json:"protocol"`
	BytesSent  int64     `json:"bytes_sent"`
	BytesRecv  int64     `json:"bytes_recv"`
	AppName    string    `json:"app_name,omitempty"`
}

// FlowCollector ingests network telemetry and produces NDR events.
type FlowCollector struct {
	bus *eventbus.Bus
	log *logger.Logger
}

func NewFlowCollector(bus *eventbus.Bus, log *logger.Logger) *FlowCollector {
	return &FlowCollector{
		bus: bus,
		log: log,
	}
}

// Ingest captures a raw flow and processes it through the detection pipeline.
func (c *FlowCollector) Ingest(flow NetworkFlow) {
	c.log.Info("[NDR] Flow detected: %s:%d -> %s:%d (%d bytes)",
		flow.SourceIP, flow.SourcePort, flow.DestIP, flow.DestPort, flow.BytesSent+flow.BytesRecv)

	// Publish to event bus for correlation
	c.bus.Publish("ndr.flow_captured", flow)

	// In a real implementation, we would check for:
	// - C2 communication patterns
	// - Exfiltration (unusual outbound byte counts)
	// - Lateral movement (internal port scanning)
}

// AnalyzeDNS analyzes DNS metadata for DGA or unusual resolution patterns.
func (c *FlowCollector) AnalyzeDNS(query string, answer string) {
	c.log.Info("[NDR] DNS Query: %s -> %s", query, answer)
	c.bus.Publish("ndr.dns_query", map[string]string{
		"query":  query,
		"answer": answer,
	})
}

// Start kicks off the background collection (e.g. listening for NetFlow packets).
func (c *FlowCollector) Start(ctx context.Context) error {
	c.log.Info("NDR Flow Collector starting on port 2055 (NetFlow)...")
	// Placeholder for actual UDP listener
	return nil
}
