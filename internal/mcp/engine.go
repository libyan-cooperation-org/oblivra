package mcp

import (
	"context"
	"fmt"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// Phase 36.7: ForensicEngine interface (IsolateHost / KillProcess) removed.
// The MCP execution surface is now read-only — siem_search / get_alerts /
// enrich_indicator. Active response (host isolation, process termination,
// playbook execution) was deleted with the broad scope cut. Pair MCP-driven
// detection with an external SOAR for response automation.

// ThreatIntel defines the interface for indicator enrichment
type ThreatIntel interface {
	Match(iocType, value string) (any, bool) // Using any to avoid direct dependency on Indicator struct if possible, or we can keep it
	MatchAny(value string) (any, bool)
}

// DefaultEngine implements the ExecutionEngine interface by routing to OBLIVRA services
type DefaultEngine struct {
	siem  database.SIEMStore
	intel ThreatIntel
	bus   *eventbus.Bus
	log   *logger.Logger
}

// NewDefaultEngine creates a new DefaultEngine
func NewDefaultEngine(siem database.SIEMStore, intel ThreatIntel, bus *eventbus.Bus, log *logger.Logger) *DefaultEngine {
	return &DefaultEngine{
		siem:  siem,
		intel: intel,
		bus:   bus,
		log:   log.WithPrefix("mcp-engine"),
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

	// Phase 36.7: isolate_host / quarantine_host / kill_process tools removed.
	// MCP is now a read-only execution surface — pair with external SOAR for
	// active response.

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
	// Phase 36.7: isolate_host / quarantine_host / kill_process / run_playbook
	// simulation cases removed — response tools deleted.
	}

	return map[string]any{
		"impact": impact,
	}, nil
}
