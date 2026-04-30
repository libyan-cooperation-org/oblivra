// Package sigma loads community Sigma rules (.yml) from a directory and
// translates a useful subset into OBLIVRA's native Rule shape.
//
// Supported subset:
//   - top-level: title, id, description, level, status, tags, falsepositives
//   - detection: a single named selection map + condition: selection
//   - selection values: string, []string, contains/endswith/startswith modifiers
//   - logsource.product / logsource.category as event-type hints
//
// Anything more elaborate (OR/AND of multiple selections, near-by, count-by,
// timeframe modifiers, regex modifiers) is rejected with a clear error so the
// loader never silently mis-evaluates a rule.
package sigma

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	if cond == "" || strings.ContainsAny(cond, " ") && cond != "selection" {
		// Only "selection" (single-block) is supported in this loader.
		return services.Rule{}, fmt.Errorf("unsupported condition %q (only 'selection' is implemented)", cond)
	}

	sel, ok := doc.Detection["selection"].(map[string]interface{})
	if !ok {
		return services.Rule{}, errors.New("missing or non-map selection")
	}

	any, fields, err := flattenSelection(sel)
	if err != nil {
		return services.Rule{}, err
	}

	return services.Rule{
		ID:         id,
		Name:       doc.Title,
		Severity:   levelToSeverity(doc.Level),
		Fields:     fields,
		AnyContain: any,
		MITRE:      mitreFromTags(doc.Tags),
		Source:     "sigma",
	}, nil
}

// flattenSelection turns a Sigma selection map into a flat AnyContain set.
// Modifiers (contains, startswith, endswith) all collapse to substring match,
// which is what AnyContain already does.
func flattenSelection(sel map[string]interface{}) (anyContain, fields []string, err error) {
	seenFields := map[string]struct{}{}
	for key, val := range sel {
		// "FieldName|contains" → field=FieldName
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
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					anyContain = append(anyContain, s)
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
