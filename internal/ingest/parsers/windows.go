package parsers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
)

// WindowsEventParser handles JSON-formatted Windows Event Logs
// Typically shipped via Winlogbeat or similar agents
type WindowsEventParser struct{}

func (p *WindowsEventParser) Name() string {
	return "WindowsEventLog"
}

func (p *WindowsEventParser) CanParse(line string) bool {
	// A simple heuristic: Is it JSON and does it contain "winlog"?
	trim := strings.TrimSpace(line)
	if !strings.HasPrefix(trim, "{") {
		return false
	}
	return strings.Contains(line, `"winlog"`) || strings.Contains(line, `"event_id"`)
}

func (p *WindowsEventParser) Parse(info Info, event *database.HostEvent) error {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(info.RawLine), &payload); err != nil {
		return err
	}
	if payload == nil {
		return fmt.Errorf("json is null")
	}

	event.EventType = "windows_event"

	// Try Winlogbeat format
	if winlog, ok := payload["winlog"].(map[string]interface{}); ok {
		if eventInfo, ok := winlog["event_data"].(map[string]interface{}); ok {
			if ip, ok := eventInfo["IpAddress"].(string); ok && ip != "-" {
				event.SourceIP = ip
			}
			if user, ok := eventInfo["TargetUserName"].(string); ok {
				event.User = user
			}
		}

		if id, ok := winlog["event_id"].(float64); ok {
			event.EventType = p.mapEventID(int(id))
		}
	} else {
		// Try generic JSON Windows event
		if id, ok := payload["EventID"].(float64); ok {
			event.EventType = p.mapEventID(int(id))
		}
	}

	return nil
}

func (p *WindowsEventParser) mapEventID(id int) string {
	switch id {
	case 4624:
		return "windows_login_success"
	case 4625:
		return "windows_login_failed"
	case 4688:
		return "windows_process_creation"
	case 1102:
		return "windows_audit_log_cleared"
	default:
		return "windows_event"
	}
}
