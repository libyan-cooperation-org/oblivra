package dag

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/analytics"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/engine/wasm"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// SIEMNode handles indexing events into the SIEM store.
type SIEMNode struct {
	BaseNode
	siem database.SIEMStore
	bus  eventbus.EventPublisher
	log  *logger.Logger
}

func NewSIEMNode(siem database.SIEMStore, bus eventbus.EventPublisher, log *logger.Logger) *SIEMNode {
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
		TenantID:  evt.TenantID,
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

	tenantID := evt.TenantID
	if tenantID == "" {
		tenantID = "default_tenant"
	}

	sessionID := evt.SessionId
	if sessionID == "" {
		sessionID = "syslog"
	}

	n.analytics.Ingest(database.WithTenant(ctx, tenantID), sessionID, evt.Host, evt.RawLine)
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

// DetectionNode handles real-time security rule evaluation.
type DetectionNode struct {
	BaseNode
	engine *detection.Evaluator
	bus    eventbus.EventPublisher
	log    *logger.Logger
}

func NewDetectionNode(e *detection.Evaluator, bus eventbus.EventPublisher, log *logger.Logger) *DetectionNode {
	return &DetectionNode{
		BaseNode: BaseNode{nodeName: "Detection_Engine"},
		engine:   e,
		bus:      bus,
		log:      log,
	}
}

func (n *DetectionNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	if n.engine == nil {
		return []*Event{evt}, nil
	}

	detEvt := detection.Event{
		TenantID:  evt.TenantID,
		EventType: evt.EventType,
		SourceIP:  evt.SourceIp,
		User:      evt.User,
		HostID:    evt.Host,
		RawLog:    evt.RawLine,
		Timestamp: evt.Timestamp,
	}

	matches := n.engine.ProcessEvent(detEvt)
	for _, match := range matches {
		if n.bus != nil {
			// Publish match for AlertingService to pick up
			n.bus.Publish("detection.match", match)
		}
	}

	return []*Event{evt}, nil
}

// ── Identity Enrichment Node ─────────────────────────────────────────────────

type UserResolver interface {
	GetUserByEmail(ctx context.Context, email string) (*database.User, error)
}

type IdentityEnrichmentNode struct {
	BaseNode
	resolver UserResolver
	cache    sync.Map
	log      *logger.Logger
}

func NewIdentityEnrichmentNode(r UserResolver, log *logger.Logger) *IdentityEnrichmentNode {
	return &IdentityEnrichmentNode{
		BaseNode: BaseNode{nodeName: "Identity_Enrichment"},
		resolver: r,
		log:      log,
	}
}

func (n *IdentityEnrichmentNode) Process(ctx context.Context, evt *Event) ([]*Event, error) {
	if evt.User == "" || n.resolver == nil {
		return []*Event{evt}, nil
	}

	// Cache lookup
	if val, ok := n.cache.Load(evt.User); ok {
		n.enrich(evt, val.(*database.User))
		return []*Event{evt}, nil
	}

	// DB lookup with retry
	var user *database.User
	var err error
	for i := 0; i < 3; i++ {
		user, err = n.resolver.GetUserByEmail(ctx, evt.User)
		if err == nil {
			break
		}
		// Only retry if it's likely a locking issue (heuristic)
		if i < 2 {
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
			continue
		}
	}

	if err != nil {
		// User not found or error, proceed without enrichment
		return []*Event{evt}, nil
	}

	// Ingress caching
	n.cache.Store(evt.User, user)
	n.enrich(evt, user)

	return []*Event{evt}, nil
}

func (n *IdentityEnrichmentNode) enrich(evt *Event, u *database.User) {
	// Inject SCIM metadata into event metadata
	if evt.Metadata == nil {
		evt.Metadata = make(map[string]string)
	}
	evt.Metadata["identity.display_name"] = u.DisplayName
	evt.Metadata["identity.title"] = u.Title
	evt.Metadata["identity.department"] = u.Department
	evt.Metadata["identity.organization"] = u.Organization
	evt.Metadata["identity.active"] = fmt.Sprintf("%v", u.Active)
	
	if u.GroupsJSON != "" {
		evt.Metadata["identity.groups"] = u.GroupsJSON
	}
}
