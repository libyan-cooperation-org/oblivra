package parsers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
)

// AzureActivityParser parses Azure resource manager Activity Logs
type AzureActivityParser struct{}

func (p *AzureActivityParser) Name() string {
	return "AzureActivityLog"
}

func (p *AzureActivityParser) CanParse(line string) bool {
	trim := strings.TrimSpace(line)
	if !strings.HasPrefix(trim, "{") {
		return false
	}
	return strings.Contains(line, `"caller"`) && strings.Contains(line, `"resourceId"`)
}

func (p *AzureActivityParser) Parse(info Info, event *database.HostEvent) error {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(info.RawLine), &payload); err != nil {
		return err
	}
	if payload == nil {
		return fmt.Errorf("json is null")
	}

	event.EventType = "azure_activity"

	if caller, ok := payload["caller"].(string); ok {
		event.User = caller
	}

	if ip, ok := payload["callerIpAddress"].(string); ok {
		event.SourceIP = ip
	}

	if op, ok := payload["operationName"].(map[string]interface{}); ok {
		if val, ok := op["value"].(string); ok {
			// e.g., Microsoft.Compute/virtualMachines/write
			event.EventType = "azure_" + strings.ToLower(strings.ReplaceAll(val, "/", "_"))
		}
	} else if opStr, ok := payload["operationName"].(string); ok {
		event.EventType = "azure_" + strings.ToLower(strings.ReplaceAll(opStr, "/", "_"))
	}

	return nil
}
