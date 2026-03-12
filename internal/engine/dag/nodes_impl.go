package dag

import (
	"context"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/engine/wasm"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// SIEMNode handles indexing events into the SIEM store.
type SIEMNode struct {
	BaseNode
	siem database.SIEMStore
	bus  *eventbus.Bus
	log  *logger.Logger
}

func NewSIEMNode(siem database.SIEMStore, bus *eventbus.Bus, log *logger.Logger) *SIEMNode {
	return &SIEMNode{
		BaseNode: BaseNode{nodeName: "SIEM_Destination"},
		siem:     siem,
		bus:      bus,
		log:      log,
	}
}

func (n *SIEMNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	if n.siem == nil {
		return nil, nil
	}

	hostEvent := &database.HostEvent{
		HostID:    evt.Host,
		Timestamp: evt.Timestamp,
		EventType: evt.EventType,
		SourceIP:  evt.SourceIp,
		User:      evt.User,
		RawLog:    evt.RawLine,
	}

	insertCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := n.siem.InsertHostEvent(insertCtx, hostEvent); err != nil {
		n.log.Error("[DAG] Failed to index SIEM event: %v", err)
	} else if n.bus != nil {
		n.bus.Publish("siem.event_indexed", *hostEvent)
	}

	return nil, nil // Leaf node
}

// AnalyticsNode handles indexing events into the Analytics engine.
type AnalyticsNode struct {
	BaseNode
	analytics *analytics.AnalyticsEngine
}

func NewAnalyticsNode(ae *analytics.AnalyticsEngine) *AnalyticsNode {
	return &AnalyticsNode{
		BaseNode:  BaseNode{nodeName: "Analytics_Destination"},
		analytics: ae,
	}
}

func (n *AnalyticsNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	if n.analytics == nil {
		return nil, nil
	}

	sessionID := evt.SessionId
	if sessionID == "" {
		sessionID = "syslog"
	}

	n.analytics.Ingest(sessionID, evt.Host, evt.RawLine)
	return nil, nil // Leaf node
}

// WASMFilterNode executes sandboxed plugins.
type WASMFilterNode struct {
	BaseNode
	wasm *wasm.PluginManager
	log  *logger.Logger
}

func NewWASMFilterNode(wm *wasm.PluginManager, log *logger.Logger) *WASMFilterNode {
	return &WASMFilterNode{
		BaseNode: BaseNode{nodeName: "WASM_Filter"},
		wasm:     wm,
		log:      log,
	}
}

func (n *WASMFilterNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	if n.wasm == nil {
		return []*Event{evt}, nil
	}

	dropped := false
	wasmCtx := wasm.WithEvent(ctx, evt.RawLine, &dropped)

	if err := n.wasm.ExecuteAll(wasmCtx); err != nil {
		n.log.Error("[DAG] WASM plugin execution error: %v", err)
	}

	if dropped {
		return nil, nil // Event filtered out
	}

	return []*Event{evt}, nil
}
