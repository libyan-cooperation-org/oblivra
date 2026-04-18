package parsers

import (
	"testing"

	"github.com/kingknull/oblivrashell/internal/database"
)

// FuzzSyslogParser bombards the generic RFC5424 Syslog parser with structural anomalies
// to guarantee the ingestion engine drops malicious inputs without panicking.
func FuzzSyslogParser(f *testing.F) {
	// Provide standard structural seeds
	f.Add(`<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - 'su root' failed for lonvick on /dev/pts/8`)
	f.Add(`<165>1 2003-08-24T05:14:15.000003-07:00 192.0.2.1 myproc 8710 - - %% It's time to make the do-nuts.`)
	f.Add(`malformed syslog payload without brackets`)

	parser := NewLinuxAuthParser()

	f.Fuzz(func(t *testing.T, payload string) {
		if !parser.CanParse(payload) {
			return
		}

		event := &database.HostEvent{}
		info := Info{RawLine: payload}
		
		// The fuzzer asserts the string splitting logic and date parsing
		// will never enter an unbounded panic state or stack overflow.
		err := parser.Parse(info, event)
		if err != nil {
			return
		}
	})
}
