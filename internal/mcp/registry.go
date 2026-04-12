package mcp

import (
	"sync"
)

// ToolRegistry manages the set of available MCP tools
type ToolRegistry struct {
	tools map[string]ToolDefinition
	mu    sync.RWMutex
}

// NewToolRegistry creates a new ToolRegistry with core OBLIVRA tools
func NewToolRegistry() *ToolRegistry {
	r := &ToolRegistry{
		tools: make(map[string]ToolDefinition),
	}
	r.registerCoreTools()
	return r
}

// Register adds a new tool to the registry
func (r *ToolRegistry) Register(def ToolDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[def.Name] = def
}

// GetTool retrieves a tool definition by name
func (r *ToolRegistry) GetTool(name string) (ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.tools[name]
	return def, ok
}

// ListTools returns all registered tool definitions
func (r *ToolRegistry) ListTools() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

func (r *ToolRegistry) registerCoreTools() {
	// 4.1 SIEM Search
	r.Register(ToolDefinition{
		Name:        "siem_search",
		Version:     "1.0",
		Description: "Search SIEM events using OQL or keywords",
		Category:    "siem",
		RiskLevel:   "low",
		InputSchema: map[string]any{
			"query":      "string",
			"time_range": "object { from: RFC3339, to: RFC3339 }",
			"limit":      "int",
			"fields":     "[]string",
		},
		Constraints: ToolConstraints{
			MaxCost:   100,
			RateLimit: "10/s",
		},
	})

	// 4.2 Alerts Query
	r.Register(ToolDefinition{
		Name:        "get_alerts",
		Version:     "1.0",
		Description: "Retrieve security alerts filterable by status and severity",
		Category:    "siem",
		RiskLevel:   "low",
		InputSchema: map[string]any{
			"status":   "open|closed|investigating",
			"severity": "low|medium|high|critical",
			"limit":    "int",
		},
		Constraints: ToolConstraints{
			MaxCost:   50,
			RateLimit: "20/s",
		},
	})

	// 4.3 Enrichment
	r.Register(ToolDefinition{
		Name:        "enrich_indicator",
		Version:     "1.0",
		Description: "Enrich indicators (IP, Domain, Hash) with threat intel",
		Category:    "intel",
		RiskLevel:   "low",
		InputSchema: map[string]any{
			"type":  "ip|domain|hash",
			"value": "string",
		},
		Constraints: ToolConstraints{
			MaxCost:   20,
			RateLimit: "50/s",
		},
	})

	// 4.4 Host Isolation (CRITICAL)
	r.Register(ToolDefinition{
		Name:             "quarantine_host",
		Version:          "1.0",
		Description:      "Isolate a host from the network to contain a threat (Agent-managed)",
		Category:         "response",
		RiskLevel:        "critical",
		RequiresApproval: true,
		InputSchema: map[string]any{
			"host_id": "string",
			"reason":  "string",
		},
		Constraints: ToolConstraints{
			MaxCost:   500,
			RateLimit: "1/m",
		},
	})

	// 4.5 Process Termination (CRITICAL)
	r.Register(ToolDefinition{
		Name:             "kill_process",
		Version:          "1.0",
		Description:      "Forcefully terminate a process by ID on a remote host",
		Category:         "response",
		RiskLevel:        "critical",
		RequiresApproval: true,
		InputSchema: map[string]any{
			"host_id": "string",
			"pid":     "int",
			"reason":  "string",
		},
		Constraints: ToolConstraints{
			MaxCost:   300,
			RateLimit: "10/m",
		},
	})

	// 4.6 Run Playbook
	r.Register(ToolDefinition{
		Name:             "run_playbook",
		Version:          "1.0",
		Description:      "Execute an automated response playbook",
		Category:         "response",
		RiskLevel:        "high",
		RequiresApproval: true,
		InputSchema: map[string]any{
			"playbook_id": "string",
			"target":      "string",
			"params":      "object",
		},
		Constraints: ToolConstraints{
			MaxCost:   200,
			RateLimit: "5/m",
		},
	})

	// 4.6 UEBA Risk
	r.Register(ToolDefinition{
		Name:        "get_entity_risk",
		Version:     "1.0",
		Description: "Get the behavior-based risk score for a user or host",
		Category:    "ueba",
		RiskLevel:   "low",
		InputSchema: map[string]any{
			"entity_id": "string",
			"type":      "user|host",
		},
		Constraints: ToolConstraints{
			MaxCost:   30,
			RateLimit: "30/s",
		},
	})
}
