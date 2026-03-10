package parsers

import (
	"strings"

	"github.com/kingknull/oblivrashell/internal/database"
)

// Info holds context about the raw log being parsed (time, source IP of syslog origin, etc)
type Info struct {
	RawLine  string
	OriginIP string
}

// Parser defines the interface that all specific log format parsers must implement
type Parser interface {
	Name() string
	CanParse(line string) bool
	Parse(info Info, event *database.HostEvent) error
}

// Registry holds the ordered list of parsers to attempt
type Registry struct {
	parsers []Parser
}

// NewRegistry initializes the standard suite of OBLIVRA SIEM parsers
func NewRegistry() *Registry {
	r := &Registry{}
	// Order matters: More specific formats (JSON, structured) should be checked before generic regex fallbacks
	r.Register(&CloudTrailParser{})
	r.Register(&AzureActivityParser{})
	r.Register(&WindowsEventParser{})
	r.Register(NewNetworkFirewallParser())
	r.Register(NewLinuxAuthParser()) // Fallback for most syslog
	return r
}

func (r *Registry) Register(p Parser) {
	r.parsers = append(r.parsers, p)
}

// Process attempts to parse the line with the first matching parser.
// If no specific parser matches, it falls back to a generic syslog/raw assumption.
func (r *Registry) Process(info Info, event *database.HostEvent) bool {
	// Treat the payload as generic initially
	event.RawLog = strings.TrimSpace(info.RawLine)
	event.SourceIP = info.OriginIP
	event.EventType = "unknown"

	for _, p := range r.parsers {
		if p.CanParse(info.RawLine) {
			if err := p.Parse(info, event); err == nil {
				// We successfully parsed it structured
				return true
			}
		}
	}

	// Unparsed/Unknown format
	return false
}
