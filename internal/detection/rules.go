package detection

import (
	"fmt"
	"os"
	"path/filepath"
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

	Conditions map[string]string `yaml:"conditions"` // e.g. {"EventType": "failed_login", "User": "root"}

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
}

// RuleSequenceStep represents a required stage in a SequenceRule.
type RuleSequenceStep struct {
	StepID     string            `yaml:"step_id"`
	Conditions map[string]string `yaml:"conditions"`
}

// RuleEngine manages active detection rules and evaluating events.
type RuleEngine struct {
	rules    []Rule
	log      *logger.Logger
	verifier *RuleVerifier
	verdicts []ValidationResult // Stores validation outcomes for UI
}

// NewRuleEngine initializes a detection engine and loads YAML rules from a directory.
func NewRuleEngine(rulesDir string, log *logger.Logger) (*RuleEngine, error) {
	engine := &RuleEngine{
		rules:    make([]Rule, 0),
		log:      log,
		verifier: NewRuleVerifier(),
		verdicts: make([]ValidationResult, 0),
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
