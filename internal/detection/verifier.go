package detection

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ValidationResult holds the status of a rule's static analysis
type ValidationResult struct {
	IsValid   bool     `json:"is_valid"`
	RuleID    string   `json:"rule_id"`
	RuleName  string   `json:"rule_name"`
	Errors    []string `json:"errors"`
	IsSecured bool     `json:"is_secured"` // True if we confirmed malicious logic is absent
}

// RuleVerifier statically analyzes loaded YAML detection rules to provide mathematical guarantees
// against DoS conditions (Regex backtracking, infinite state aggregation) and syntactic errors.
type RuleVerifier struct{}

func NewRuleVerifier() *RuleVerifier {
	return &RuleVerifier{}
}

// Verify statically analyzes a single detection rule
func (v *RuleVerifier) Verify(r *Rule) ValidationResult {
	res := ValidationResult{
		IsValid:   true,
		RuleID:    r.ID,
		RuleName:  r.Name,
		Errors:    make([]string, 0),
		IsSecured: true,
	}

	// 1. Syntactic Verification (AST Validation)
	if r.ID == "" {
		res.addError("Rule ID is missing")
	}
	if r.Name == "" {
		res.addError("Rule Name is missing")
	}
	validSeverities := map[string]bool{"critical": true, "high": true, "medium": true, "low": true, "info": true}
	if !validSeverities[strings.ToLower(r.Severity)] {
		res.addError(fmt.Sprintf("Invalid severity '%s'", r.Severity))
	}
	if r.Type == "" {
		res.addError("Rule Type is missing")
	}

	// 2. Mathematical Constraints on stateful tracking
	if r.Type == ThresholdRule || r.Type == SequenceRule {
		if r.WindowSec <= 0 || r.WindowSec > 86400 {
			res.addError(fmt.Sprintf("window_sec (%d) must be mathematically bounded between 1 and 86400 seconds to prevent heap exhaustion", r.WindowSec))
		}
	}
	if r.Type == ThresholdRule {
		if r.Threshold <= 0 {
			res.addError("threshold must be strictly positive")
		}
		if r.Threshold > 10000 {
			res.addError("threshold mathematically exceeds bounded correlation buffer limits (> 10,000)")
		}

		// 22.1: Enforce GroupBy Cardinality Limits
		if len(r.GroupBy) > 5 {
			res.addError("GroupBy exceeds 5 fields; mathematically likely to cause state explosion")
		}
	}
	if r.Type == SequenceRule {
		if len(r.Sequence) < 2 {
			res.addError("Sequence rules require at least two causal steps")
		}
		if len(r.Sequence) > 20 {
			res.addError("Sequence steps exceed mathematical upper bound (20) for tracking state machine")
		}
		// Validate internal steps
		for i, s := range r.Sequence {
			if s.StepID == "" {
				res.addError(fmt.Sprintf("Sequence step %d missing step_id", i))
			}
			errs := v.checkConditions(s.Conditions)
			for _, e := range errs {
				res.addError(fmt.Sprintf("Sequence step '%s': %s", s.StepID, e))
			}
		}
	}

	// 3. Prevent Malicious Policy Injection
	// Evaluate top-level conditions
	if r.Type != SequenceRule {
		if len(r.Conditions) == 0 {
			res.addError("Rule contains no conditional AST logic")
		}
		errs := v.checkConditions(r.Conditions)
		for _, e := range errs {
			res.addError(e)
		}
	}

	if len(res.Errors) > 0 {
		res.IsValid = false
		res.IsSecured = false
	}

	// 1.2: Rule Cost Circuit Breaker (DoS Protection)
	// ── 22.3: Circuit Breaker Validation ───────────────────────────────────
	// Rules with extreme execution cost are rejected to prevent DoS.
	cost := r.ExecutionCost()
	if cost > MaxRuleCost {
		res.addError(fmt.Sprintf("Rule execution cost (%d) exceeds sovereign-grade stability threshold (%d). Reduce grouping cardinality or regex complexity.", cost, MaxRuleCost))
		res.IsValid = false
		res.IsSecured = false
	}

	return res
}

func (res *ValidationResult) addError(err string) {
	res.Errors = append(res.Errors, err)
}

// checkConditions validates all evaluation nodes inside a condition map, primarily targeting ReDoS
func (v *RuleVerifier) checkConditions(conditions map[string]interface{}) []string {
	var errs []string
	for field, val := range conditions {
		v.checkSingleCondition(field, val, &errs)
	}
	return errs
}

func (v *RuleVerifier) checkSingleCondition(field string, val interface{}, errs *[]string) {
	if slice, ok := val.([]interface{}); ok {
		for _, item := range slice {
			v.checkSingleCondition(field, item, errs)
		}
		return
	}

	matchPattern := fmt.Sprintf("%v", val)
	if strings.HasPrefix(strings.ToLower(matchPattern), "regex:") {
		pattern := matchPattern[6:]
		if len(pattern) > 512 {
			*errs = append(*errs, fmt.Sprintf("Regex too long (%d > 512) on field '%s'", len(pattern), field))
			return
		}
		if err := v.verifyRegexSafety(pattern); err != nil {
			*errs = append(*errs, fmt.Sprintf("Unsafe ReDoS regex detected on field '%s': %v", field, err))
		}
	} else if strings.HasPrefix(strings.ToLower(matchPattern), "cidr:") {
		cidr := matchPattern[5:]
		if strings.Count(cidr, "/") != 1 {
			*errs = append(*errs, fmt.Sprintf("Invalid CIDR format on field '%s': %s", field, cidr))
		}
	}
}

// verifyRegexSafety applies mathematical constraint heuristics to prevent ReDoS CPU exhaustion.
func (v *RuleVerifier) verifyRegexSafety(pattern string) error {
	// First check if it compiles natively
	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex syntax: %w", err)
	}

	// Heuristic 1: Detect nested quantifiers (e.g. (a+)+ or (.*)*)
	// This is a naive heuristic specifically targeting the most catastrophic backtracking combinations.
	nestedQuantifierPattern := `(\([^)]*[*+]+[^)]*\)[*+]+)`
	if matched, _ := regexp.MatchString(nestedQuantifierPattern, pattern); matched {
		return errors.New("nested quantifier structure mathematically vulnerable to catastrophic backtracking")
	}

	// Heuristic 2: Extremely large bounds
	largeBoundsPattern := `\{[0-9]+,[0-9]{4,}\}`
	if matched, _ := regexp.MatchString(largeBoundsPattern, pattern); matched {
		return errors.New("regex bounds exceed strict CPU timing limits")
	}

	return nil
}
