package main

import (
	"strings"
	"testing"
)

func TestParseJournaldRecord_Sshd(t *testing.T) {
	line := []byte(`{"__CURSOR":"s=abc;i=42;t=900","__REALTIME_TIMESTAMP":"1714521600123456","_HOSTNAME":"web-01","_SYSTEMD_UNIT":"sshd.service","_PID":"1234","SYSLOG_IDENTIFIER":"sshd","MESSAGE":"Failed password for root from 10.0.0.42"}`)
	out, cursor, ok := parseJournaldRecord(line)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if cursor != "s=abc;i=42;t=900" {
		t.Errorf("cursor = %q", cursor)
	}
	// Output is a syslog-RFC3164-shape line so the server-side parser handles it natively.
	if !strings.Contains(out, "web-01 sshd.service[1234]: Failed password") {
		t.Errorf("out = %q", out)
	}
}

func TestParseJournaldRecord_NoMessage(t *testing.T) {
	// MESSAGE field absent → drop the record (no payload to ingest).
	line := []byte(`{"__CURSOR":"s=abc","_HOSTNAME":"x"}`)
	if _, _, ok := parseJournaldRecord(line); ok {
		t.Error("expected drop on missing MESSAGE")
	}
}

func TestParseJournaldRecord_FallbackIdentifier(t *testing.T) {
	// systemd-less services lack _SYSTEMD_UNIT; we should fall back to
	// SYSLOG_IDENTIFIER so the line still labels the producer.
	line := []byte(`{"__CURSOR":"c","SYSLOG_IDENTIFIER":"cron","_HOSTNAME":"db-01","MESSAGE":"job done"}`)
	out, _, ok := parseJournaldRecord(line)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if !strings.Contains(out, "cron: job done") {
		t.Errorf("out = %q", out)
	}
}

func TestStripFlag_RemovesFlagAndArg(t *testing.T) {
	in := []string{"--follow", "--since", "now", "--unit", "sshd.service"}
	out := stripFlag(in, "--since")
	want := []string{"--follow", "--unit", "sshd.service"}
	if len(out) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(out), len(want), out)
	}
	for i, v := range want {
		if out[i] != v {
			t.Errorf("out[%d] = %q, want %q", i, out[i], v)
		}
	}
}

func TestStripFlag_NoOp(t *testing.T) {
	in := []string{"--follow", "--unit", "sshd.service"}
	out := stripFlag(in, "--since") // not present
	if len(out) != len(in) {
		t.Errorf("expected no change, got %v", out)
	}
}
