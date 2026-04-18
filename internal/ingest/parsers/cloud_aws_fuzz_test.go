package parsers

import (
	"testing"

	"github.com/kingknull/oblivrashell/internal/database"
)

// FuzzCloudTrailParser bombards the generic AWS CloudTrail JSON parser with malformed
// or deeply nested malicious payloads to ensure it never encounters OOM, stack exhaustion,
// or catastrophic failure.
func FuzzCloudTrailParser(f *testing.F) {
	// Provide standard structural seeds
	f.Add(`{"eventVersion":"1.08","userIdentity":{"type":"IAMUser","userName":"Alice","arn":"arn:aws:iam::123456789012:user/Alice"},"eventTime":"2023-01-01T12:00:00Z","eventSource":"iam.amazonaws.com","eventName":"CreateUser","sourceIPAddress":"192.168.1.100","userAgent":"aws-cli/2.0.0"}`)
	f.Add(`{"eventName": "ConsoleLogin", "sourceIPAddress": "1.1.1.1"}`)
	f.Add(`malformed json body`)

	parser := &CloudTrailParser{}

	f.Fuzz(func(t *testing.T, payload string) {
		// Ignore string payloads our fast-path analyzer immediately rejects
		if !parser.CanParse(payload) {
			return
		}

		event := &database.HostEvent{}
		info := Info{RawLine: payload}
		
		// The fuzzer asserts `json.Unmarshal` will safely reject and return `error`
		// instead of crashing the Goroutine when confronted with garbage bytes.
		err := parser.Parse(info, event)
		if err != nil {
			return
		}
	})
}
