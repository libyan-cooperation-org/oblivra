package parsers

import (
	"testing"

	"github.com/kingknull/oblivrashell/internal/database"
)

func TestCloudTrailParser(t *testing.T) {
	rawLog := `{
		"eventVersion": "1.08",
		"userIdentity": {
			"type": "IAMUser",
			"userName": "Alice",
			"arn": "arn:aws:iam::123456789012:user/Alice"
		},
		"eventTime": "2023-01-01T12:00:00Z",
		"eventSource": "iam.amazonaws.com",
		"eventName": "CreateUser",
		"sourceIPAddress": "192.168.1.100",
		"userAgent": "aws-cli/2.0.0"
	}`

	parser := &CloudTrailParser{}
	if !parser.CanParse(rawLog) {
		t.Fatal("Parser missed valid CloudTrail JSON")
	}

	event := &database.HostEvent{}
	info := Info{RawLine: rawLog}
	err := parser.Parse(info, event)

	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if event.EventType != "aws_createuser" {
		t.Errorf("Expected event_type 'aws_createuser', got '%s'", event.EventType)
	}
	if event.SourceIP != "192.168.1.100" {
		t.Errorf("Expected source_ip '192.168.1.100', got '%s'", event.SourceIP)
	}
	if event.User != "arn:aws:iam::123456789012:user/Alice" {
		t.Errorf("Expected user ARN, got '%s'", event.User)
	}
}

func TestWindowsEventParser(t *testing.T) {
	rawLog := `{
		"@timestamp": "2023-01-01T12:00:00.000Z",
		"winlog": {
			"channel": "Security",
			"event_id": 4625,
			"event_data": {
				"TargetUserName": "Administrator",
				"IpAddress": "10.0.0.50"
			}
		}
	}`

	parser := &WindowsEventParser{}
	if !parser.CanParse(rawLog) {
		t.Fatal("Parser missed valid Winlogbeat JSON")
	}

	event := &database.HostEvent{}
	info := Info{RawLine: rawLog}
	err := parser.Parse(info, event)

	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if event.EventType != "windows_login_failed" {
		t.Errorf("Expected event_type 'windows_login_failed', got '%s'", event.EventType)
	}
	if event.SourceIP != "10.0.0.50" {
		t.Errorf("Expected source_ip '10.0.0.50', got '%s'", event.SourceIP)
	}
	if event.User != "Administrator" {
		t.Errorf("Expected user 'Administrator', got '%s'", event.User)
	}
}

func TestNetworkFirewallParser(t *testing.T) {
	rawLog := `1,2023/01/01 12:00:00,0018C103328,TRAFFIC,drop,1,2023/01/01 12:00:00,192.168.1.200,8.8.8.8,0.0.0.0,0.0.0.0,Rule1,vsys1,Trust,Untrust,ethernet1/1,`
	parser := NewNetworkFirewallParser()

	if !parser.CanParse(rawLog) {
		t.Fatal("Parser missed valid Palo Alto log")
	}

	event := &database.HostEvent{}
	info := Info{RawLine: rawLog}
	err := parser.Parse(info, event)

	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}
	if event.EventType != "network_connection_blocked" {
		t.Errorf("Expected network_connection_blocked, got '%s'", event.EventType)
	}
	if event.SourceIP != "192.168.1.200" {
		t.Errorf("Expected srcip '192.168.1.200', got '%s'", event.SourceIP)
	}
}
