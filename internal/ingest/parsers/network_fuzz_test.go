package parsers

import (
	"testing"

	"github.com/kingknull/oblivrashell/internal/database"
)

// FuzzNetworkFirewallParser bombards the Palo Alto CSV heuristics with malformed inputs
// Objective: Ensure catastrophic backtracking or array out-of-bounds panics never occur.
func FuzzNetworkFirewallParser(f *testing.F) {
	// Provide standard structural seeds
	f.Add("1,2023/01/01 12:00:00,0018C103328,TRAFFIC,drop,1,2023/01/01 12:00:00,192.168.1.200,8.8.8.8,0.0.0.0,0.0.0.0,Rule1,vsys1,Trust,Untrust,ethernet1/1,")
	f.Add("filterlog: 5,,,1000000103,igb0,match,block,in,4,0x0,,250,30704,0,none,17,udp,139,10.0.0.1,10.0.0.255,137,137,209")
	f.Add("THREAT,vulnerability,1,2023/01/01 12:00:00,10.0.0.1,10.0.0.2,")

	parser := NewNetworkFirewallParser()

	f.Fuzz(func(t *testing.T, payload string) {
		// Ignore inputs the parser immediately rejects to speed up deep logic fuzzing
		if !parser.CanParse(payload) {
			return
		}

		event := &database.HostEvent{}
		info := Info{RawLine: payload}
		
		// The fuzzer asserts this function NEVER panics when digesting garbage bytes
		err := parser.Parse(info, event)
		if err != nil {
			// Expected behavior: it might return an error, but it must not panic.
			return
		}
	})
}
