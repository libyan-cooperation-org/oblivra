package ingest

import (
	"testing"
)

// FuzzAutoParse targets the central log multiplexer.
// It ensures that no combination of characters, malformed JSON,
// or incomplete CEF/LEEF headers can cause a panic or OOM.
func FuzzAutoParse(f *testing.F) {
	// Seed with valid examples from each format
	f.Add(`{"event_type": "windows_login", "user": "admin", "winlog": {"event_id": 4624}}`)
	f.Add(`CEF:0|Binary Ninja|OBLIVRA|1.0|LOGIN_FAIL|Failed Login|10|src=10.0.0.1 duser=root`)
	f.Add(`LEEF:1.0|Microsoft|Windows|2022|4625|cat=Auth	srcIP=1.1.1.1	usrName=guest`)
	f.Add(`<34>1 2003-10-11T22:14:15.003Z host1 sshd[123]: Failed password for root from 1.2.3.4`)
	f.Add(`random garbage string that is not a log`)
	f.Add(`{ "incomplete": "json" `)
	f.Add(`CEF:0|incomplete|header`)

	f.Fuzz(func(t *testing.T, data string) {
		// AutoParse is the main entry point for the ingestion pipeline.
		// It must NEVER panic regardless of input.
		_ = AutoParse(data)
	})
}
