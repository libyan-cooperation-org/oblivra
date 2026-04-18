package detection

import "testing"

// FuzzSigmaTranspile ensures the Sigma transpiler never panics on arbitrary input.
// Run with: go test -fuzz=FuzzSigmaTranspile ./internal/detection/
func FuzzSigmaTranspile(f *testing.F) {
	// Seed corpus — valid and invalid Sigma documents
	seeds := []string{
		sigmaTestRuleMimikatz,
		sigmaTestRuleSSH,
		sigmaTestRuleDeprecated,
		`title: Empty Detection
detection:
  condition: selection`,
		`{}`,
		`title: Malformed YAML
detection: [invalid`,
		`title: No Condition
detection:
  selection:
    CommandLine|contains: test`,
		`title: All Modifiers
id: fuzz-seed-1
status: stable
level: high
logsource:
  category: process_creation
detection:
  sel1:
    CommandLine|startswith: 'powershell'
  sel2:
    CommandLine|endswith: '.ps1'
  sel3:
    CommandLine|re: '(?i)invoke-?expression'
  sel4|all:
    CommandLine|contains:
      - bypass
      - hidden
  condition: sel1 or sel2 or sel3 or sel4`,
	}

	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic regardless of input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("TranspileSigma panicked on input: %q\npanic: %v", data, r)
			}
		}()
		// We only care that it doesn't panic — errors are acceptable
		_, _ = TranspileSigma(data)
	})
}
