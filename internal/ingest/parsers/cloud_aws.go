package parsers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
)

// CloudTrailParser parses AWS CloudTrail JSON events
type CloudTrailParser struct{}

func (p *CloudTrailParser) Name() string {
	return "AWSCloudTrail"
}

func (p *CloudTrailParser) CanParse(line string) bool {
	trim := strings.TrimSpace(line)
	if !strings.HasPrefix(trim, "{") {
		return false
	}
	return strings.Contains(line, `"eventSource"`) && strings.Contains(line, `"userIdentity"`)
}

func (p *CloudTrailParser) Parse(info Info, event *database.HostEvent) error {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(info.RawLine), &payload); err != nil {
		return err
	}
	if payload == nil {
		return fmt.Errorf("json is null")
	}

	event.EventType = "cloud_audit"

	if eventName, ok := payload["eventName"].(string); ok {
		event.EventType = "aws_" + strings.ToLower(eventName)
	}

	if srcIP, ok := payload["sourceIPAddress"].(string); ok {
		event.SourceIP = srcIP
	}

	if identity, ok := payload["userIdentity"].(map[string]interface{}); ok {
		if arn, ok := identity["arn"].(string); ok {
			event.User = arn
		} else if userName, ok := identity["userName"].(string); ok {
			event.User = userName
		}
	}

	return nil
}
