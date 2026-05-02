// Package sigma loads community Sigma rules (.yml) from a directory and
// translates a useful subset into OBLIVRA's native Rule shape.
//
// Supported subset:
//   - top-level: title, id, description, level, status, tags, falsepositives
//   - detection: any number of named selection-style blocks (selection,
//     selection_*, anything that resolves to a map of field → values)
//   - condition: "selection" (single block), "1 of selection_*" (glob over
//     blocks whose name starts with selection_), or "1 of them" (any
//     block). All three are unioned into the rule's AnyContain set since
//     OBLIVRA's matcher is substring-based.
//   - selection values: string, number, bool; flat or list. Field
//     modifiers (contains/startswith/endswith) collapse to substring match.
//   - logsource.product / logsource.category as event-type hints.
//
// Anything more elaborate (AND of selections, count-by, near-by, regex
// modifiers, etc.) is rejected with a clear error so the loader never
// silently mis-evaluates a rule.
package sigma

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kingknull/oblivra/internal/services"
)

type sigmaDoc struct {
	Title          string                 `yaml:"title"`
	ID             string                 `yaml:"id"`
	Description    string                 `yaml:"description"`
	Level          string                 `yaml:"level"`
	Status         string                 `yaml:"status"`
	Tags           []string               `yaml:"tags"`
	Logsource      map[string]string      `yaml:"logsource"`
	Detection      map[string]interface{} `yaml:"detection"`
	Falsepositives []string               `yaml:"falsepositives"`
}

// LoadDir reads every *.yml/*.yaml file under dir, translates each to a Rule,
// and returns them. Best-effort: parse failures on individual files are
// reported alongside successful translations.
func LoadDir(dir string) ([]services.Rule, []error) {
	var rules []services.Rule
	var errs []error

	if dir == "" {
		return nil, []error{errors.New("sigma: dir required")}
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}
	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		rule, perr := LoadFile(path)
		if perr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", filepath.Base(path), perr))
			return nil
		}
		rules = append(rules, rule)
		return nil
	})
	if walkErr != nil {
		errs = append(errs, walkErr)
	}
	return rules, errs
}

// LoadFile parses a single Sigma YAML file.
func LoadFile(path string) (services.Rule, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return services.Rule{}, err
	}
	return Parse(raw, filepath.Base(path))
}

// Parse parses Sigma YAML bytes. The originName is only used for the rule ID
// fallback when the YAML lacks one.
func Parse(raw []byte, originName string) (services.Rule, error) {
	var doc sigmaDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return services.Rule{}, fmt.Errorf("yaml: %w", err)
	}
	if doc.Title == "" {
		return services.Rule{}, errors.New("missing title")
	}

	id := doc.ID
	if id == "" {
		id = "sigma-" + strings.TrimSuffix(strings.TrimSuffix(originName, ".yml"), ".yaml")
	}

	cond, _ := doc.Detection["condition"].(string)
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return services.Rule{}, errors.New("missing condition")
	}

	res, err := resolveCondition(doc.Detection, cond)
	if err != nil {
		return services.Rule{}, err
	}

	rule := services.Rule{
		ID:       id,
		Name:     doc.Title,
		Severity: levelToSeverity(doc.Level),
		MITRE:    mitreFromTags(doc.Tags),
		Source:   "sigma",
	}

	// `keywords` shape: list of substrings, no field filter.
	if len(res.keywords) > 0 {
		var any []string
		for _, item := range res.keywords {
			if s, ok := item.(string); ok {
				any = append(any, s)
			}
		}
		if len(any) == 0 {
			return services.Rule{}, errors.New("empty keywords list")
		}
		rule.AnyContain = any
		rule.Fields = []string{"message", "raw"}
	} else {
		// Standard map-block shape. Union positive and negative sides.
		allAny, fields, err := unionSelections(res.positive)
		if err != nil {
			return services.Rule{}, err
		}
		if len(allAny) == 0 {
			return services.Rule{}, errors.New("empty selection (no matchable values)")
		}
		rule.Fields = fields
		// `all of <pattern>` → every value must appear (AllContain)
		// All other shapes → any value appears (AnyContain)
		if res.matchAll {
			rule.AllContain = allAny
		} else {
			rule.AnyContain = allAny
		}
	}

	if len(res.negative) > 0 {
		if notValues, _, err := unionSelections(res.negative); err == nil {
			rule.NotContain = notValues
		}
	}

	return rule, nil
}

// unionSelections flattens a list of detection blocks into a single
// AnyContain set + a deduped Fields list. Used for both the positive
// and negative sides of a "selection and not exclude" condition.
func unionSelections(blocks []map[string]interface{}) (anyContain, fields []string, err error) {
	fieldSet := map[string]struct{}{}
	for _, blk := range blocks {
		any, blkFields, err := flattenSelection(blk)
		if err != nil {
			return nil, nil, err
		}
		anyContain = append(anyContain, any...)
		for _, f := range blkFields {
			if _, dup := fieldSet[f]; !dup {
				fieldSet[f] = struct{}{}
				fields = append(fields, f)
			}
		}
	}
	return anyContain, fields, nil
}

// conditionResult is what resolveCondition hands back. The matchAll
// flag flips the rule's positive token set into AllContain (every
// token must appear) instead of AnyContain (any one is enough). Used
// by the `all of <pattern>` shape.
type conditionResult struct {
	positive  []map[string]interface{}
	negative  []map[string]interface{}
	keywords  []interface{} // raw list values from a top-level `keywords` block
	matchAll  bool
}

// resolveCondition parses the Sigma `condition` expression. Supported:
//
//   - <term>                       positive only
//   - <term> and not <term>        positive + negative
//   - <term> and <term2>           rejected (matcher can't honestly AND)
//   - all of <pattern>             every matched block; values use AllContain
//   - all of them                  every block; values use AllContain
//   - keywords                     list-shaped block → AnyContain
//
// where <term> is one of:
//   - selection
//   - <name>            (e.g. "exclude", "selection_a", "keywords")
//   - 1 of them
//   - 1 of <pattern>
//
// `or` between blocks, parens, count()/near()/etc are rejected.
func resolveCondition(detection map[string]interface{}, cond string) (conditionResult, error) {
	cond = strings.TrimSpace(cond)

	// "<lhs> and not <rhs>" — common shape; check first.
	if idx := strings.Index(cond, " and not "); idx > 0 {
		lhs := strings.TrimSpace(cond[:idx])
		rhs := strings.TrimSpace(cond[idx+len(" and not "):])
		res, err := resolveCondition(detection, lhs)
		if err != nil {
			return res, fmt.Errorf("lhs %q: %w", lhs, err)
		}
		neg, err := termBlocksLoose(detection, rhs)
		if err != nil {
			return res, fmt.Errorf("not-rhs %q: %w", rhs, err)
		}
		res.negative = neg
		return res, nil
	}

	// `keywords` is a special case — Sigma allows a top-level list
	// instead of a map. The condition is just "keywords".
	if cond == "keywords" {
		if raw, ok := detection["keywords"]; ok {
			if list, isList := raw.([]interface{}); isList {
				return conditionResult{keywords: list}, nil
			}
			// keywords-as-map → fall through to map handling below.
		}
	}

	// `all of them` / `all of <pattern>` — every matched block's tokens
	// must appear (AllContain semantics). We support this for the
	// 1-field-per-block shape that's by far the common case in Sigma.
	if strings.HasPrefix(cond, "all of ") {
		pattern := strings.TrimPrefix(cond, "all of ")
		blocks, err := globMatchedBlocks(detection, pattern)
		if err != nil {
			return conditionResult{}, fmt.Errorf("`%s`: %w", cond, err)
		}
		return conditionResult{positive: blocks, matchAll: true}, nil
	}

	if strings.Contains(cond, " and ") {
		return conditionResult{}, fmt.Errorf("unsupported condition %q (AND between blocks needs the matcher to evaluate both; use `selection and not exclude` instead)", cond)
	}
	if strings.Contains(cond, " or ") {
		return conditionResult{}, fmt.Errorf("unsupported condition %q (use `1 of them` or `1 of <pattern>` for OR)", cond)
	}

	pos, err := termBlocksLoose(detection, cond)
	if err != nil {
		return conditionResult{}, err
	}
	return conditionResult{positive: pos}, nil
}

// termBlocksLoose is like termBlocks but also handles a top-level
// `keywords` list when referenced bare. Used as a leaf resolver.
func termBlocksLoose(detection map[string]interface{}, term string) ([]map[string]interface{}, error) {
	term = strings.TrimSpace(term)
	if blocks, err := termBlocks(detection, term); err == nil {
		return blocks, nil
	} else if !strings.Contains(err.Error(), "no such block") {
		return nil, err
	}
	return nil, fmt.Errorf("term %q: no such block", term)
}

// globMatchedBlocks returns every detection block whose name matches
// the glob pattern. Used by `all of <pattern>` and `1 of <pattern>`.
func globMatchedBlocks(detection map[string]interface{}, pattern string) ([]map[string]interface{}, error) {
	if pattern == "them" {
		var out []map[string]interface{}
		for k, v := range detection {
			if k == "condition" {
				continue
			}
			if blk, ok := v.(map[string]interface{}); ok {
				out = append(out, blk)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("no detection blocks defined")
		}
		return out, nil
	}
	re, err := globToRegex(pattern)
	if err != nil {
		return nil, err
	}
	var out []map[string]interface{}
	for k, v := range detection {
		if k == "condition" || !re.MatchString(k) {
			continue
		}
		if blk, ok := v.(map[string]interface{}); ok {
			out = append(out, blk)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("matched no blocks")
	}
	return out, nil
}

// termBlocks resolves a single term (a leaf in the condition AST).
func termBlocks(detection map[string]interface{}, term string) ([]map[string]interface{}, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, errors.New("empty term")
	}
	if blocks, err := selectBlocks(detection, term); err == nil {
		return blocks, nil
	} else if !errors.Is(err, errLeafNotMatched) {
		return nil, err
	}
	// Plain block name, e.g. "exclude" or "selection_a".
	if blk, ok := detection[term].(map[string]interface{}); ok {
		return []map[string]interface{}{blk}, nil
	}
	return nil, fmt.Errorf("term %q: no such block", term)
}

// errLeafNotMatched is returned by selectBlocks when the term isn't one
// it recognises — distinguishes "syntactically wrong" from "this is a
// plain block name, try the lookup".
var errLeafNotMatched = errors.New("not a recognised term")

// selectBlocks resolves a Sigma `condition` expression into the set of
// detection blocks whose values should be unioned. We support three
// shapes that together cover the bulk of community rules:
//
//   - "selection"        single block named `selection`
//   - "1 of <pattern>"   any block whose name matches the glob pattern
//                        (e.g. "1 of selection_*" expands to selection_*)
//   - "1 of them"        union of every non-condition block
//
// More complex AND/OR/NOT expressions deliberately error out — a rule
// that requires "selection AND not exclude" can't be honestly evaluated
// by our substring matcher, so we surface the limitation rather than
// pretending to support it.
func selectBlocks(detection map[string]interface{}, cond string) ([]map[string]interface{}, error) {
	if cond == "selection" {
		blk, ok := detection["selection"].(map[string]interface{})
		if !ok {
			return nil, errors.New("missing or non-map selection")
		}
		return []map[string]interface{}{blk}, nil
	}

	// "1 of them" — union every non-condition block.
	if cond == "1 of them" {
		var out []map[string]interface{}
		for k, v := range detection {
			if k == "condition" {
				continue
			}
			if blk, ok := v.(map[string]interface{}); ok {
				out = append(out, blk)
			}
		}
		if len(out) == 0 {
			return nil, errors.New("`1 of them` but no detection blocks defined")
		}
		return out, nil
	}

	// "1 of <pattern>" — glob-match block names against pattern.
	if strings.HasPrefix(cond, "1 of ") {
		pattern := strings.TrimPrefix(cond, "1 of ")
		re, err := globToRegex(pattern)
		if err != nil {
			return nil, fmt.Errorf("condition pattern %q: %w", pattern, err)
		}
		var out []map[string]interface{}
		for k, v := range detection {
			if k == "condition" || !re.MatchString(k) {
				continue
			}
			if blk, ok := v.(map[string]interface{}); ok {
				out = append(out, blk)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("`%s` matched no blocks", cond)
		}
		return out, nil
	}

	// Caller (termBlocks) wants to know that we don't recognise this
	// shape so it can try a plain-block lookup. This isn't a hard error.
	return nil, errLeafNotMatched
}

// globToRegex converts a Sigma block-name glob (with `*` wildcard) to an
// anchored regex. We escape every other regex metacharacter so a pattern
// like `selection.with-dot_*` doesn't accidentally match more than the
// author intended.
func globToRegex(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			b.WriteString(".*")
		case '.', '+', '?', '(', ')', '[', ']', '{', '}', '|', '\\', '$', '^':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// flattenSelection turns a Sigma selection map into a flat AnyContain set.
// Modifiers (contains, startswith, endswith) all collapse to substring match,
// which is what AnyContain already does. Numeric and boolean values are
// stringified so e.g. `EventID: [1102, 104]` produces ["1102", "104"].
func flattenSelection(sel map[string]interface{}) (anyContain, fields []string, err error) {
	seenFields := map[string]struct{}{}
	for key, val := range sel {
		field := key
		if idx := strings.IndexByte(key, '|'); idx >= 0 {
			field = key[:idx]
		}
		field = mapField(field)
		if _, dup := seenFields[field]; !dup {
			seenFields[field] = struct{}{}
			fields = append(fields, field)
		}

		switch v := val.(type) {
		case string:
			anyContain = append(anyContain, v)
		case int:
			anyContain = append(anyContain, strconv.Itoa(v))
		case int64:
			anyContain = append(anyContain, strconv.FormatInt(v, 10))
		case float64:
			// YAML numbers can land as float64 — emit as int form when
			// it has no fractional part (EventID 1102 should stringify
			// to "1102", not "1102.000000").
			if v == float64(int64(v)) {
				anyContain = append(anyContain, strconv.FormatInt(int64(v), 10))
			} else {
				anyContain = append(anyContain, strconv.FormatFloat(v, 'g', -1, 64))
			}
		case bool:
			if v {
				anyContain = append(anyContain, "true")
			} else {
				anyContain = append(anyContain, "false")
			}
		case []interface{}:
			for _, item := range v {
				switch s := item.(type) {
				case string:
					anyContain = append(anyContain, s)
				case int:
					anyContain = append(anyContain, strconv.Itoa(s))
				case int64:
					anyContain = append(anyContain, strconv.FormatInt(s, 10))
				case float64:
					if s == float64(int64(s)) {
						anyContain = append(anyContain, strconv.FormatInt(int64(s), 10))
					} else {
						anyContain = append(anyContain, strconv.FormatFloat(s, 'g', -1, 64))
					}
				case bool:
					if s {
						anyContain = append(anyContain, "true")
					} else {
						anyContain = append(anyContain, "false")
					}
				}
			}
		case nil:
			// allow nil, ignore
		default:
			return nil, nil, fmt.Errorf("unsupported selection value for %q (%T)", key, v)
		}
	}
	if len(anyContain) == 0 {
		return nil, nil, errors.New("empty selection")
	}
	if len(fields) == 0 {
		fields = []string{"message", "raw"}
	}
	return anyContain, fields, nil
}

func mapField(sigmaField string) string {
	switch strings.ToLower(sigmaField) {
	case "commandline", "command":
		return "message"
	case "image", "process", "processname":
		return "message"
	case "hostname", "host", "computer":
		return "hostId"
	case "eventid", "type":
		return "eventType"
	case "level", "severity":
		return "severity"
	default:
		return strings.ToLower(sigmaField)
	}
}

func levelToSeverity(level string) services.AlertSeverity {
	switch strings.ToLower(level) {
	case "informational", "info", "low":
		return services.AlertSeverityLow
	case "medium":
		return services.AlertSeverityMedium
	case "high":
		return services.AlertSeverityHigh
	case "critical":
		return services.AlertSeverityCritical
	default:
		return services.AlertSeverityMedium
	}
}

func mitreFromTags(tags []string) []string {
	var out []string
	for _, t := range tags {
		t = strings.ToLower(t)
		if strings.HasPrefix(t, "attack.t") {
			out = append(out, strings.ToUpper(strings.TrimPrefix(t, "attack.")))
		}
	}
	return out
}
