package parsers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
)

// GCPAuditParser parses Google Cloud Platform Audit Log JSON events.
// GCP audit logs use the Cloud Audit Logs schema with protoPayload wrapping.
type GCPAuditParser struct{}

func (p *GCPAuditParser) Name() string {
	return "GCPAuditLog"
}

func (p *GCPAuditParser) CanParse(line string) bool {
	trim := strings.TrimSpace(line)
	if !strings.HasPrefix(trim, "{") {
		return false
	}
	return strings.Contains(line, `"protoPayload"`) && strings.Contains(line, `"insertId"`)
}

func (p *GCPAuditParser) Parse(info Info, event *database.HostEvent) error {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(info.RawLine), &payload); err != nil {
		return err
	}
	if payload == nil {
		return fmt.Errorf("json is null")
	}

	event.EventType = "gcp_audit"

	proto, ok := payload["protoPayload"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Extract method name (e.g. google.iam.admin.v1.CreateServiceAccount)
	if methodName, ok := proto["methodName"].(string); ok {
		parts := strings.Split(methodName, ".")
		short := parts[len(parts)-1]
		event.EventType = "gcp_" + strings.ToLower(short)
	}

	// Extract caller IP from requestMetadata
	if reqMeta, ok := proto["requestMetadata"].(map[string]interface{}); ok {
		if callerIP, ok := reqMeta["callerIp"].(string); ok {
			event.SourceIP = callerIP
		}
	}

	// Extract principal email from authenticationInfo
	if authInfo, ok := proto["authenticationInfo"].(map[string]interface{}); ok {
		if email, ok := authInfo["principalEmail"].(string); ok {
			event.User = email
		}
	}

	return nil
}
