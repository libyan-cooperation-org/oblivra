package mcp

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// HostIsolator defines the interface for host isolation operations
type HostIsolator interface {
	IsolateHost(hostID string, reason string) error
}

// ThreatIntel defines the interface for indicator enrichment
type ThreatIntel interface {
	Match(iocType, value string) (any, bool) // Using any to avoid direct dependency on Indicator struct if possible, or we can keep it
	MatchAny(value string) (any, bool)
}

// DefaultEngine implements the ExecutionEngine interface by routing to OBLIVRA services
type DefaultEngine struct {
	siem     database.SIEMStore
	isolator HostIsolator
	intel    ThreatIntel
	bus      *eventbus.Bus
	log      *logger.Logger
}

// NewDefaultEngine creates a new DefaultEngine
func NewDefaultEngine(siem database.SIEMStore, isolator HostIsolator, intel ThreatIntel, bus *eventbus.Bus, log *logger.Logger) *DefaultEngine {
	return &DefaultEngine{
		siem:     siem,
		isolator: isolator,
		intel:    intel,
		bus:      bus,
		log:      log.WithPrefix("mcp-engine"),
	}
}

// Execute runs the actual tool logic
func (e *DefaultEngine) Execute(ctx context.Context, tool string, params map[string]any) (any, error) {
	switch tool {
	case "siem_search":
		query, _ := params["query"].(string)
		limit, _ := params["limit"].(float64)
		if limit == 0 {
			limit = 100
		}
		events, err := e.siem.SearchHostEvents(ctx, query, int(limit))
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"count":  len(events),
			"events": events,
		}, nil

	case "get_alerts":
		status, _ := params["status"].(string)
		severity, _ := params["severity"].(string)
		limit, _ := params["limit"].(float64)
		if limit == 0 {
			limit = 100
		}
		
		query := "EventType:security_alert"
		if status != "" {
			query = fmt.Sprintf("%s AND Status:%s", query, status)
		}
		if severity != "" {
			query = fmt.Sprintf("%s AND Severity:%s", query, severity)
		}
		
		alerts, err := e.siem.SearchHostEvents(ctx, query, int(limit))
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"count":  len(alerts),
			"alerts": alerts,
		}, nil

	case "enrich_indicator":
		iocType, _ := params["type"].(string)
		value, _ := params["value"].(string)
		
		// Map MCP types to OBLIVRA types if necessary
		// For now assume they match or use MatchAny
		ind, found := e.intel.Match(iocType, value)
		if !found {
			ind, found = e.intel.MatchAny(value)
		}
		
		if !found {
			return map[string]any{"found": false}, nil
		}
		return map[string]any{
			"found":     true,
			"indicator": ind,
		}, nil

	case "isolate_host":
		hostID, _ := params["host_id"].(string)
		reason, _ := params["reason"].(string)
		
		err := e.isolator.IsolateHost(hostID, reason)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"status": "isolated",
			"host_id": hostID,
		}, nil

	default:
		return nil, fmt.Errorf("execution logic not implemented for tool: %s", tool)
	}
}

// Simulate calculates the potential impact of a tool without executing it
func (e *DefaultEngine) Simulate(ctx context.Context, tool string, params map[string]any) (any, error) {
	impact := map[string]any{
		"affected_hosts":   0,
		"alerts_generated": 0,
		"cost_estimate":    0,
	}

	switch tool {
	case "siem_search":
		impact["cost_estimate"] = 10
	case "get_alerts":
		impact["cost_estimate"] = 5
	case "enrich_indicator":
		impact["cost_estimate"] = 2
	case "isolate_host":
		impact["affected_hosts"] = 1
		impact["alerts_generated"] = 2
		impact["cost_estimate"] = 500
	case "run_playbook":
		impact["affected_hosts"] = 5
		impact["alerts_generated"] = 10
		impact["cost_estimate"] = 200
	}

	return map[string]any{
		"impact": impact,
	}, nil
}
