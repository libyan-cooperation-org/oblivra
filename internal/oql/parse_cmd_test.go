package oql

import (
	"testing"
)

// TestParseCommand_Grammar verifies the new `parse` pipe-command
// (Phase 27.2.1) is accepted by the parser and produces the expected
// AST shape.
func TestParseCommand_Grammar(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantFmt  ParseFormat
		wantSrc  string
		wantOut  string
	}{
		{"json default field", `source=logs | parse json`, ParseJSON, "_raw", ""},
		{"json explicit field", `source=logs | parse json message`, ParseJSON, "message", ""},
		{"json with prefix", `source=logs | parse json message as evt`, ParseJSON, "message", "evt"},
		{"xml default", `source=logs | parse xml`, ParseXML, "_raw", ""},
		{"xml explicit + prefix", `source=logs | parse xml body as parsed`, ParseXML, "body", "parsed"},
		{"kv message", `source=logs | parse kv message`, ParseKV, "message", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := Parse(tc.input, nil)
			if err != nil {
				t.Fatalf("Parse(%q) failed: %v", tc.input, err)
			}
			if len(q.Commands) != 1 {
				t.Fatalf("got %d commands, want 1", len(q.Commands))
			}
			pc, ok := q.Commands[0].(*ParseCommand)
			if !ok {
				t.Fatalf("got %T, want *ParseCommand", q.Commands[0])
			}
			if pc.Format != tc.wantFmt {
				t.Errorf("Format: got %d, want %d", pc.Format, tc.wantFmt)
			}
			if pc.Field.Canonical() != tc.wantSrc {
				t.Errorf("Field: got %q, want %q", pc.Field.Canonical(), tc.wantSrc)
			}
			if pc.Output != tc.wantOut {
				t.Errorf("Output: got %q, want %q", pc.Output, tc.wantOut)
			}
		})
	}
}

// TestParseJSONFlat checks the flatten helper that backs `parse json`.
// Nested objects must collapse into dot-paths so downstream `where`
// and `eval` clauses can reference them like ordinary fields.
func TestParseJSONFlat(t *testing.T) {
	in := `{"user":"alice","ctx":{"ip":"10.0.0.1","port":443},"tags":["a","b"]}`
	out, err := parseJSONFlat(in)
	if err != nil {
		t.Fatalf("parseJSONFlat: %v", err)
	}
	want := map[string]interface{}{
		"user":     "alice",
		"ctx.ip":   "10.0.0.1",
		"ctx.port": float64(443),
		"tags.0":   "a",
		"tags.1":   "b",
	}
	for k, v := range want {
		got, ok := out[k]
		if !ok {
			t.Errorf("missing %q", k)
			continue
		}
		if got != v {
			t.Errorf("%q: got %v, want %v", k, got, v)
		}
	}
}

// TestParseKVPairs checks quoted-value handling and separator
// flexibility — these are the two highest-risk behaviours that
// real-world KV log lines exercise.
func TestParseKVPairs(t *testing.T) {
	in := `user=alice src_ip="10.0.0.1, scanner" action=allow rule=R-001`
	out := parseKVPairs(in)
	if out["user"] != "alice" {
		t.Errorf("user: got %v, want alice", out["user"])
	}
	if out["src_ip"] != "10.0.0.1, scanner" {
		t.Errorf("quoted src_ip lost embedded comma: got %v", out["src_ip"])
	}
	if out["action"] != "allow" {
		t.Errorf("action: got %v, want allow", out["action"])
	}
	if out["rule"] != "R-001" {
		t.Errorf("rule: got %v, want R-001", out["rule"])
	}
}
