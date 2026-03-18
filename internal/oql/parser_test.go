package oql

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "basic search",
			input: "source=firewall",
		},
		{
			name:  "search with pipe and stats",
			input: "source=firewall | stats count by src_ip",
		},
		{
			name:  "complex pipeline",
			input: "source=firewall | where dst_port=443 | stats count by src_ip | sort -count",
		},
		{
			name:  "eval and rename",
			input: "source=auth | eval user_lower=lower(user) | rename user_lower as u",
		},
		{
			name:  "subsearch",
			input: "source=dns [ search source=threat_intel | table ip ]",
		},
		{
			name:  "timechart",
			input: "source=access | timechart span=1h count by status",
		},
		{
			name:  "multi-eval",
			input: "source=logs | eval a=1, b=2, c=a+b",
		},
		{
			name:    "invalid syntax",
			input:   "source=firewall |",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && q == nil {
				t.Error("Parse() returned nil query without error")
			}
		})
	}
}
