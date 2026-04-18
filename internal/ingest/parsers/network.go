package parsers

import (
	"regexp"
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
)

// NetworkFirewallParser handles common perimeter logs like Palo Alto or pfSense
type NetworkFirewallParser struct {
	// Simple CSV/Syslog heuristic matching for basic traffic logs
	paloRegex *regexp.Regexp
}

func NewNetworkFirewallParser() *NetworkFirewallParser {
	return &NetworkFirewallParser{
		// Regex to grab SrcIP, DstIP, Action from standard Palo Alto Traffic log format
		paloRegex: regexp.MustCompile(`TRAFFIC,(start|end|drop|deny),[^,]+,([^,]+),([^,]+),[^,]*,[^,]*,[^,]*,[^,]*,[^,]*,[^,]*,([^,]+),([^,]+),`),
	}
}

func (p *NetworkFirewallParser) Name() string {
	return "NetworkFirewall"
}

func (p *NetworkFirewallParser) CanParse(line string) bool {
	// Heuristic: comma separated, contains "TRAFFIC" or "THREAT" typical of network appliances
	return strings.Contains(line, "TRAFFIC,") || strings.Contains(line, "THREAT,") || strings.Contains(line, "filterlog:")
}

func (p *NetworkFirewallParser) Parse(info Info, event *database.HostEvent) error {
	line := info.RawLine

	if match := p.paloRegex.FindStringSubmatch(line); len(match) >= 4 {
		action := match[1]
		srcIP := match[3]
		// dstIP := match[4]

		event.SourceIP = srcIP

		switch action {
		case "drop", "deny":
			event.EventType = "network_connection_blocked"
		case "start", "end":
			event.EventType = "network_connection_allowed"
		default:
			event.EventType = "network_traffic"
		}
		return nil
	}

	event.EventType = "network_event"
	return nil
}
