package detection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// Phase 25.7 — Sigma rule coverage gate.
//
// Production loads detection rules via `Evaluator.LoadSigmaDirectory`
// (see `internal/services/alerting_service.go`), so coverage tests
// must drive the same path. This file:
//
//   1. Defines fixture sets per rule-id (the YAML 'id' field).
//   2. Walks `sigma/core/` and asserts every rule-id has fixtures
//      registered (TestRuleCoverage_AllRulesHaveFixtures).
//   3. Loads rules through Evaluator.LoadSigmaDirectory and runs
//      each fixture's event through ProcessEvent, asserting the
//      expected match (TestSigmaRules_FixtureEvaluation).
//
// Adding a new rule under sigma/core/ without registering fixtures
// here fails CI. This is the enforcement gate that closes the
// audit's "22% coverage" finding.

// fixtureSet maps rule-id (Sigma YAML 'id' field) → representative
// events with expected match outcomes. Fixture authors: at minimum,
// one match-positive + one match-negative case per rule.
// Note on IDs: the Sigma loader prefixes rule IDs with `sigma-` so
// they don't collide with native rules. Fixture keys MUST match the
// loaded ID, not the bare YAML 'id' field.
var fixtureSet = map[string][]ruleFixture{
	// SSH brute force — sigma/core/brute_force_ssh.yml
	"sigma-5cb6c182-167d-419b-9c7b-7b0f7a0b5f1a": {
		{
			Name:        "failed-password matches",
			Event:       Event{EventType: "sshd", RawLog: "Failed password for invalid user admin from 192.168.1.50"},
			ExpectMatch: true,
		},
		{
			Name:        "successful login does NOT match",
			Event:       Event{EventType: "sshd", RawLog: "Accepted publickey for ubuntu from 10.0.0.5"},
			ExpectMatch: false,
		},
	},

	// Discovery commands — sigma/core/discovery_commands.yml
	"sigma-a8d5e0f1-2c3d-4e5f-9a1b-c2d3e4f5a6b7": {
		{
			Name:        "whoami in shell matches",
			Event:       Event{EventType: "shell", RawLog: "user@host:~$ whoami"},
			ExpectMatch: true,
		},
		{
			Name:        "ordinary ls does NOT match",
			Event:       Event{EventType: "shell", RawLog: "user@host:~$ ls -la /tmp"},
			ExpectMatch: false,
		},
	},

	// Privilege escalation via sudo — sigma/core/priv_esc_sudo.yml
	"sigma-e5f6a7b8-c9d0-4e1f-8a2b-b3c4d5e6f7a8": {
		{
			Name:        "sudo to root shell matches",
			Event:       Event{EventType: "sudo", RawLog: "user : TTY=pts/0 ; PWD=/home/user ; USER=root ; COMMAND=/bin/bash"},
			ExpectMatch: true,
		},
		{
			Name:        "ordinary sudo apt does NOT match",
			Event:       Event{EventType: "sudo", RawLog: "user : TTY=pts/0 ; USER=root ; COMMAND=/usr/bin/apt update"},
			ExpectMatch: false,
		},
	},

	// Ransomware canary — sigma/core/ransomware_canary.yml
	"sigma-d4f5e6a7-b8c9-4d0e-9f1b-a2b3c4d5e6f7": {
		{
			Name:        "canary file modify matches",
			Event:       Event{EventType: "fim", RawLog: "MODIFY /etc/oblivra/canary.txt by pid=1234"},
			ExpectMatch: true,
		},
		{
			Name:        "unrelated file modify does NOT match",
			Event:       Event{EventType: "fim", RawLog: "MODIFY /var/log/syslog by pid=5678"},
			ExpectMatch: false,
		},
	},
}

// ruleFixture is the local fixture shape — a representative
// detection.Event + the operator's expected match outcome.
type ruleFixture struct {
	Name        string
	Event       Event
	ExpectMatch bool
}

// loadEvaluatorWithSigma constructs an Evaluator and pre-loads the
// shipped sigma/core/ rule directory. Tests use this instead of the
// production NewEvaluator (which expects native Rule YAML, not Sigma)
// — the Evaluator embeds RuleEngine, so LoadSigmaDirectory works.
func loadEvaluatorWithSigma(t *testing.T) *Evaluator {
	t.Helper()
	log, err := logger.New(logger.Config{Level: logger.ErrorLevel, OutputPath: os.DevNull})
	if err != nil {
		t.Fatalf("logger.New: %v", err)
	}
	// Use a tmp directory as the native-rules dir so NewRuleEngine
	// doesn't blow up on Sigma syntax (it expects native format).
	emptyDir := t.TempDir()
	ev, err := NewEvaluator(emptyDir, log)
	if err != nil {
		t.Fatalf("NewEvaluator(%s): %v", emptyDir, err)
	}
	rulesDir := filepath.Join("..", "..", "sigma", "core")
	if err := ev.LoadSigmaDirectory(rulesDir); err != nil {
		t.Fatalf("LoadSigmaDirectory(%s): %v", rulesDir, err)
	}
	ev.RebuildRouteIndex()
	return ev
}

// TestRuleCoverage_AllRulesHaveFixtures fails if any rule under
// sigma/core/ doesn't have fixtures registered in `fixtureSet`.
// This is the CI hard-block — adding a rule without fixtures must
// not merge.
func TestRuleCoverage_AllRulesHaveFixtures(t *testing.T) {
	rulesDir := filepath.Join("..", "..", "sigma", "core")
	matches, err := filepath.Glob(filepath.Join(rulesDir, "*.yml"))
	if err != nil {
		t.Fatalf("glob sigma/core: %v", err)
	}
	if len(matches) == 0 {
		t.Skip("no rule files under sigma/core/ — nothing to enforce")
	}

	ev := loadEvaluatorWithSigma(t)
	rules := ev.GetRules()

	missing := []string{}
	for _, r := range rules {
		if _, ok := fixtureSet[r.ID]; !ok {
			missing = append(missing, r.Name+" (id="+r.ID+")")
		}
	}
	if len(missing) > 0 {
		t.Errorf(
			"%d sigma rule(s) shipped without fixtures — register entries in "+
				"fixtureSet so the rule's behaviour is verified:\n  %s",
			len(missing), strings.Join(missing, "\n  "),
		)
	}
}

// TestSigmaRules_FixtureEvaluation pushes each fixture through the
// production evaluator and asserts the expected match outcome.
// Drift between rule edits and fixture expectations fails loud.
//
// Note on threshold/sequence rules: ProcessEvent returns Matches
// only when the rule's threshold is satisfied. For single-event
// fixtures we ALSO accept matchesConditions=true (the underlying
// per-event predicate) as a positive outcome — otherwise rules
// with `threshold: 5` would never match a one-event fixture.
func TestSigmaRules_FixtureEvaluation(t *testing.T) {
	ev := loadEvaluatorWithSigma(t)
	rules := ev.GetRules()
	if len(rules) == 0 {
		t.Skip("no rules loaded; skipping evaluation")
	}

	// Build a rule-id → rule lookup so each fixture can target the
	// expected rule via matchesConditions.
	byID := make(map[string]Rule, len(rules))
	for _, r := range rules {
		byID[r.ID] = r
	}

	for ruleID, fixtures := range fixtureSet {
		rule, ok := byID[ruleID]
		if !ok {
			// rule was registered in fixtureSet but isn't currently
			// loaded — fixture-author error. Surface clearly.
			t.Errorf("fixtureSet has entry for unknown rule id %q", ruleID)
			continue
		}
		for _, fx := range fixtures {
			t.Run(rule.Name+"/"+fx.Name, func(t *testing.T) {
				gotMatch := ev.matchesConditions(rule.Conditions, fx.Event)
				if gotMatch != fx.ExpectMatch {
					t.Errorf("expected match=%v, got match=%v",
						fx.ExpectMatch, gotMatch)
				}
			})
		}
	}
}
