package detection

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kingknull/oblivrashell/internal/logger"
	"gopkg.in/yaml.v3"
)

// RuleType defines the category of the detection rule.
type RuleType string

const (
	ThresholdRule   RuleType = "threshold"
	FrequencyRule   RuleType = "frequency"
	SequenceRule    RuleType = "sequence"
	CorrelationRule RuleType = "correlation"
)

// Rule represents a parsed detection rule from YAML.
type Rule struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Severity    string   `yaml:"severity"` // low, medium, high, critical
	Type        RuleType `yaml:"type"`

	// 22.4: Semantic version (semver) for rule versioning and rollback.
	// Format: "1.0.0" — incremented on every rule change.
	Version string `yaml:"version"`

	Conditions map[string]interface{} `yaml:"conditions"` // e.g. {"EventType": "failed_login", "User": "root"}

	// MITRE Framework
	MitreTactics    []string `yaml:"mitre_tactics"`    // e.g. ["Initial Access", "Credential Access"]
	MitreTechniques []string `yaml:"mitre_techniques"` // e.g. ["T1078", "T1110"]

	// Threshold/Frequency Specifics
	Threshold int `yaml:"threshold"`  // Number of occurrences
	WindowSec int `yaml:"window_sec"` // Evaluation window in seconds

	// Sequence Specifics
	Sequence []RuleSequenceStep `yaml:"sequence"` // Define explicit causal chains

	// Grouping (e.g. group by source_ip to track per-ip frequency)
	GroupBy []string `yaml:"group_by"`

	// Deduplication
	DedupWindowSec int `yaml:"dedup_window_sec"` // Prevent alert spam

	// 22.5: Sharding Hint. If true, rule is evaluated in the CorrelationHub, not in local shards.
	IsGlobal bool `yaml:"is_global"`
}

// RuleSequenceStep represents a required stage in a SequenceRule.
type RuleSequenceStep struct {
	StepID     string                 `yaml:"step_id"`
	Conditions map[string]interface{} `yaml:"conditions"`
}

// RuleEngine manages active detection rules and evaluating events.
type RuleEngine struct {
	rules    []Rule
	log      *logger.Logger
	verifier *RuleVerifier
	verdicts []ValidationResult // Stores validation outcomes for UI

	// 22.4: previous version of each rule keyed by rule ID, enabling rollback.
	previousRules map[string]Rule
}

// NewRuleEngine initializes a detection engine and loads YAML rules from a directory.
func NewRuleEngine(rulesDir string, log *logger.Logger) (*RuleEngine, error) {
	engine := &RuleEngine{
		rules:         make([]Rule, 0),
		log:           log,
		verifier:      NewRuleVerifier(),
		verdicts:      make([]ValidationResult, 0),
		previousRules: make(map[string]Rule),
	}

	if err := engine.LoadRules(rulesDir); err != nil {
		return nil, fmt.Errorf("failed to load detection rules: %w", err)
	}

	return engine, nil
}

// LoadRules scans the target directory for .yaml files and parses them into memory.
func (e *RuleEngine) LoadRules(rulesDir string) error {
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			// If dir doesn't exist, create it so users have a place to put rules
			if err := os.MkdirAll(rulesDir, 0700); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
			rulePath := filepath.Join(rulesDir, entry.Name())
			if err := e.loadRuleFile(rulePath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *RuleEngine) loadRuleFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var rule Rule
	if err := yaml.Unmarshal(data, &rule); err != nil {
		return fmt.Errorf("failed to parse rule %s: %w", path, err)
	}

	if rule.ID == "" || rule.Name == "" || rule.Type == "" {
		// Provide a baseline rejection for completely malformed YAML structures
		e.log.Error("[DETECTION] Rule %s is missing required core fields", path)
		e.verdicts = append(e.verdicts, ValidationResult{
			IsValid:   false,
			RuleID:    "UNKNOWN",
			RuleName:  filepath.Base(path),
			Errors:    []string{"Missing core YAML fields (id, name, type)"},
			IsSecured: false,
		})
		return fmt.Errorf("invalid rule in %s: missing required fields", path)
	}

	result := e.verifier.Verify(&rule)
	e.verdicts = append(e.verdicts, result)

	if !result.IsValid {
		e.log.Error("[DETECTION] Ignored invalid rule '%s': %v", rule.Name, result.Errors)
		return nil // Don't crash loading, just ignore the malformed rule
	}

	e.rules = append(e.rules, rule)
	return nil
}

// GetRules returns all currently loaded rules.
func (e *RuleEngine) GetRules() []Rule {
	return e.rules
}

// LoadSigmaDirectory converts and loads all Sigma rule files (.yml/.yaml) from a directory.
// Sigma rules are automatically transpiled to the native Oblivra rule format.
// Files that fail transpilation are logged and skipped; they do not block other rules.
func (e *RuleEngine) LoadSigmaDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			e.log.Info("[SIGMA] Directory %s does not exist, skipping", dir)
			return nil
		}
		return fmt.Errorf("read sigma dir %s: %w", dir, err)
	}

	loaded := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := e.LoadSigmaFile(path); err != nil {
			e.log.Warn("[SIGMA] Skipping %s: %v", entry.Name(), err)
			continue
		}
		loaded++
	}

	e.log.Info("[SIGMA] Loaded %d rules from %s", loaded, dir)
	return nil
}

// LoadSigmaFile parses a single Sigma rule file and converts it to the native format.
func (e *RuleEngine) LoadSigmaFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	rule, err := TranspileSigma(data)
	if err != nil {
		return fmt.Errorf("transpile: %w", err)
	}

	result := e.verifier.Verify(rule)
	e.verdicts = append(e.verdicts, result)

	if !result.IsValid {
		e.log.Warn("[SIGMA] Invalid transpiled rule '%s': %v", rule.Name, result.Errors)
		return fmt.Errorf("validation failed: %v", result.Errors)
	}

	// Avoid duplicates when hot-reloading
	for _, existing := range e.rules {
		if existing.ID == rule.ID {
			e.log.Info("[SIGMA] Rule %s already loaded, skipping duplicate", rule.ID)
			return nil
		}
	}

	e.rules = append(e.rules, *rule)
	e.log.Info("[SIGMA] Loaded rule: %s [%s]", rule.Name, rule.Severity)
	return nil
}

// GetVerificationResults returns the static analysis verdicts of loaded rule files
func (e *RuleEngine) GetVerificationResults() []ValidationResult {
	return e.verdicts
}

// ── 22.4: Rule Versioning & Rollback ────────────────────────────────────────────────

// UpsertRule replaces an existing rule by ID (saving the previous version)
// or adds a new rule. Returns whether the rule was updated (vs. inserted).
func (e *RuleEngine) UpsertRule(rule Rule) bool {
	if rule.Version == "" {
		rule.Version = "1.0.0"
	}
	for i, existing := range e.rules {
		if existing.ID == rule.ID {
			// Archive current version before replacing
			e.previousRules[rule.ID] = existing
			e.rules[i] = rule
			e.log.Info("[RULES] Updated rule %s: %s → %s", rule.ID, existing.Version, rule.Version)
			return true
		}
	}
	e.rules = append(e.rules, rule)
	e.log.Info("[RULES] Added new rule %s v%s", rule.ID, rule.Version)
	return false
}

// RollbackRule restores the previous version of a rule.
// Returns the restored rule and true if a previous version existed.
func (e *RuleEngine) RollbackRule(ruleID string) (*Rule, bool) {
	prev, ok := e.previousRules[ruleID]
	if !ok {
		return nil, false
	}
	for i, r := range e.rules {
		if r.ID == ruleID {
			e.rules[i] = prev
			delete(e.previousRules, ruleID)
			e.log.Info("[RULES] Rolled back rule %s to v%s", ruleID, prev.Version)
			return &prev, true
		}
	}
	return nil, false
}

// GetPreviousVersion returns the archived previous version of a rule, if any.
func (e *RuleEngine) GetPreviousVersion(ruleID string) (*Rule, bool) {
	prev, ok := e.previousRules[ruleID]
	if !ok {
		return nil, false
	}
	return &prev, true
}

// ── 22.4: MITRE Coverage Gap Report ──────────────────────────────────────────────

// MITRETechniqueScore represents the detection coverage for one ATT&CK technique.
type MITRETechniqueScore struct {
	TechniqueID   string   `json:"technique_id"`
	TechniqueName string   `json:"technique_name"`
	RuleCount     int      `json:"rule_count"`
	RuleIDs       []string `json:"rule_ids"`
	CoverageLevel string   `json:"coverage_level"` // "none", "partial", "covered"
}

// MITREGapReport contains per-technique coverage scores for Navigator export.
type MITREGapReport struct {
	Techniques    []MITRETechniqueScore `json:"techniques"`
	CoveredCount  int                   `json:"covered_count"`
	PartialCount  int                   `json:"partial_count"`
	UncoveredCount int                  `json:"uncovered_count"`
	TotalTechniques int                 `json:"total_techniques"`
	NavigatorLayer  map[string]interface{} `json:"navigator_layer"`
}

// GenerateMITREGapReport produces a MITRE Navigator-compatible coverage analysis
// across all loaded rules. Returns the full coverage breakdown and a Navigator layer.
func (e *RuleEngine) GenerateMITREGapReport() *MITREGapReport {
	// All known MITRE ATT&CK techniques (subset — expandable)
	allTechniques := GetAllTechniques()

	// Build a map of technique ID → matching rules
	techRules := make(map[string][]string)
	for _, rule := range e.rules {
		for _, tech := range rule.MitreTechniques {
			techRules[tech] = append(techRules[tech], rule.ID)
		}
	}

	var scores []MITRETechniqueScore
	covered, partial, uncovered := 0, 0, 0

	for techID, techName := range allTechniques {
		ruleIDs := techRules[techID]
		count := len(ruleIDs)
		level := "none"
		switch {
		case count == 0:
			level = "none"
			uncovered++
		case count == 1:
			level = "partial"
			partial++
		default:
			level = "covered"
			covered++
		}
		scores = append(scores, MITRETechniqueScore{
			TechniqueID:   techID,
			TechniqueName: techName,
			RuleCount:     count,
			RuleIDs:       ruleIDs,
			CoverageLevel: level,
		})
	}

	// Build MITRE Navigator layer JSON structure
	navTechniques := make([]map[string]interface{}, 0, len(scores))
	for _, s := range scores {
		color := ""
		switch s.CoverageLevel {
		case "covered":
			color = "#00ff88" // green
		case "partial":
			color = "#ffaa00" // amber
		case "none":
			color = "" // no highlight
		}
		entry := map[string]interface{}{
			"techniqueID": s.TechniqueID,
			"tactic":      "",
			"score":       s.RuleCount,
			"color":       color,
			"comment":     fmt.Sprintf("%d rule(s): %s", s.RuleCount, strings.Join(s.RuleIDs, ", ")),
			"enabled":     true,
			"metadata":    []interface{}{},
		}
		navTechniques = append(navTechniques, entry)
	}

	navLayer := map[string]interface{}{
		"name":       "OBLIVRA Detection Coverage",
		"version":    "4.5",
		"domain":     "enterprise-attack",
		"description": fmt.Sprintf("Auto-generated by OBLIVRA. %d/%d techniques covered.", covered, len(allTechniques)),
		"techniques": navTechniques,
		"gradient": map[string]interface{}{
			"colors":  []string{"#ff3355", "#ffaa00", "#00ff88"},
			"minValue": 0,
			"maxValue": 3,
		},
	}

	return &MITREGapReport{
		Techniques:      scores,
		CoveredCount:    covered,
		PartialCount:    partial,
		UncoveredCount:  uncovered,
		TotalTechniques: len(allTechniques),
		NavigatorLayer:  navLayer,
	}
}

// ── 22.4: Rule Test Framework ──────────────────────────────────────────────────────────

// RuleTestFixture is a single event + expected match result.
type RuleTestFixture struct {
	Description string                 `yaml:"description" json:"description"`
	Event       map[string]interface{} `yaml:"event"       json:"event"`
	ExpectMatch bool                   `yaml:"expect_match" json:"expect_match"`
}

// RuleTestResult is the outcome of running one fixture against a rule.
type RuleTestResult struct {
	Description string `json:"description"`
	Expected    bool   `json:"expected"`
	Got         bool   `json:"got"`
	Passed      bool   `json:"passed"`
}

// RuleTestSuiteResult is the aggregate result of running all fixtures for a rule.
type RuleTestSuiteResult struct {
	RuleID   string           `json:"rule_id"`
	RuleName string           `json:"rule_name"`
	Passed   int              `json:"passed"`
	Failed   int              `json:"failed"`
	Total    int              `json:"total"`
	Results  []RuleTestResult `json:"results"`
}

// TestRule runs a rule against a set of test fixtures and returns the results.
// fixtures is a slice of RuleTestFixture; the engine evaluates each event against
// the rule's conditions using the same matching logic as live detection.
func (e *RuleEngine) TestRule(rule *Rule, fixtures []RuleTestFixture) *RuleTestSuiteResult {
	suite := &RuleTestSuiteResult{
		RuleID:   rule.ID,
		RuleName: rule.Name,
	}

	for _, fix := range fixtures {
		// Simulate matching: check all conditions against the fixture event
		matched := matchRuleConditions(rule, fix.Event)
		passed := matched == fix.ExpectMatch
		suite.Results = append(suite.Results, RuleTestResult{
			Description: fix.Description,
			Expected:    fix.ExpectMatch,
			Got:         matched,
			Passed:      passed,
		})
		if passed {
			suite.Passed++
		} else {
			suite.Failed++
		}
		suite.Total++
	}
	return suite
}

// matchRuleConditions evaluates a rule's conditions map against a test event map.
// Uses the same field-matching semantics as the live detection engine.
func matchRuleConditions(rule *Rule, event map[string]interface{}) bool {
	for field, expected := range rule.Conditions {
		eventVal, ok := event[field]
		if !ok {
			// Field missing from event — no match
			return false
		}
		expStr, ok := expected.(string)
		if !ok {
			continue
		}
		evtStr := fmt.Sprintf("%v", eventVal)
		// Support the same "regex:" prefix as the live engine
		if strings.HasPrefix(expStr, "regex:") {
			pattern := strings.TrimPrefix(expStr, "regex:")
			matched, err := regexp.MatchString(pattern, evtStr)
			if err != nil || !matched {
				return false
			}
		} else if !strings.EqualFold(evtStr, expStr) {
			return false
		}
	}
	return true
}
