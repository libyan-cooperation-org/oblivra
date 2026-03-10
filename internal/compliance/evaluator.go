package compliance

import (
	"embed"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed packs/*.yaml
var packsFS embed.FS

// CheckType defines the kind of automated compliance check.
type CheckType string

const (
	CheckAuditLogExists       CheckType = "audit_log_exists"
	CheckMinLogRetentionDays  CheckType = "min_log_retention_days"
	CheckEncryptionEnabled    CheckType = "encryption_enabled"
	CheckMFAEnabled           CheckType = "mfa_enabled"
	CheckTLSEnabled           CheckType = "tls_enabled"
	CheckFIMEnabled           CheckType = "fim_enabled"
	CheckRBACEnabled          CheckType = "rbac_enabled"
	CheckAlertingEnabled      CheckType = "alerting_enabled"
	CheckFailedLoginThreshold CheckType = "failed_login_threshold"
	CheckMerkleIntegrity      CheckType = "merkle_integrity_valid"
	CheckEvidenceLocker       CheckType = "evidence_locker_available"
	CheckCompliancePackLoaded CheckType = "compliance_pack_loaded"
	CheckAccessReview         CheckType = "access_review_exists"
	CheckVulnScan             CheckType = "vulnerability_scan_exists"
)

// PackDefinition is the YAML schema for a compliance pack.
type PackDefinition struct {
	ID          string       `yaml:"id"          json:"id"`
	Name        string       `yaml:"name"        json:"name"`
	Version     string       `yaml:"version"     json:"version"`
	Description string       `yaml:"description" json:"description"`
	Category    string       `yaml:"category"    json:"category"`
	Controls    []ControlDef `yaml:"controls"    json:"controls"`
}

// ControlDef is a single compliance control within a pack.
type ControlDef struct {
	ID          string     `yaml:"id"          json:"id"`
	Title       string     `yaml:"title"       json:"title"`
	Description string     `yaml:"description" json:"description,omitempty"`
	Checks      []CheckDef `yaml:"checks"      json:"checks"`
}

// CheckDef is an automated check within a control.
type CheckDef struct {
	Type        CheckType              `yaml:"type"        json:"type"`
	Description string                 `yaml:"description" json:"description"`
	Params      map[string]interface{} `yaml:"params" json:"params,omitempty"`
}

// EvalResult is the outcome of evaluating a single check.
type EvalResult struct {
	CheckType   CheckType `json:"check_type"`
	Description string    `json:"description"`
	Passed      bool      `json:"passed"`
	Details     string    `json:"details,omitempty"`
}

// ControlResult is the outcome of evaluating a single control.
type ControlResult struct {
	ID     string       `json:"id"`
	Title  string       `json:"title"`
	Passed bool         `json:"passed"`
	Checks []EvalResult `json:"checks"`
}

// PackResult is the full evaluation of a compliance pack.
type PackResult struct {
	PackID         string          `json:"pack_id"`
	PackName       string          `json:"pack_name"`
	EvaluatedAt    time.Time       `json:"evaluated_at"`
	TotalControls  int             `json:"total_controls"`
	PassedControls int             `json:"passed_controls"`
	FailedControls int             `json:"failed_controls"`
	Score          float64         `json:"score"` // 0.0 – 100.0
	Controls       []ControlResult `json:"controls"`
}

// SystemState captures the current system capabilities for evaluation.
// The evaluator checks these flags against pack requirements.
type SystemState struct {
	EncryptionEnabled    bool
	MFAEnabled           bool
	TLSEnabled           bool
	FIMEnabled           bool
	RBACEnabled          bool
	AlertingEnabled      bool
	MerkleIntegrityValid bool
	EvidenceLockerAvail  bool
	AuditLogCount        int64
	OldestLogAge         time.Duration // age of the oldest audit log
	EventTypesPresent    map[string]bool
}

// Evaluator loads compliance packs and evaluates them against system state.
type Evaluator struct {
	packs []PackDefinition
}

// NewEvaluator creates an evaluator and loads all embedded packs.
func NewEvaluator() (*Evaluator, error) {
	e := &Evaluator{}

	entries, err := packsFS.ReadDir("packs")
	if err != nil {
		return nil, fmt.Errorf("read packs dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := packsFS.ReadFile("packs/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read pack %s: %w", entry.Name(), err)
		}

		var pack PackDefinition
		if err := yaml.Unmarshal(data, &pack); err != nil {
			return nil, fmt.Errorf("parse pack %s: %w", entry.Name(), err)
		}
		e.packs = append(e.packs, pack)
	}

	return e, nil
}

// ListPacks returns all loaded compliance packs.
func (e *Evaluator) ListPacks() []PackDefinition {
	return e.packs
}

// GetPack returns a specific pack by ID.
func (e *Evaluator) GetPack(id string) (*PackDefinition, error) {
	for i := range e.packs {
		if e.packs[i].ID == id {
			return &e.packs[i], nil
		}
	}
	return nil, fmt.Errorf("compliance pack %q not found", id)
}

// Evaluate runs all checks in a pack against the current system state.
func (e *Evaluator) Evaluate(packID string, state SystemState) (*PackResult, error) {
	pack, err := e.GetPack(packID)
	if err != nil {
		return nil, err
	}

	result := &PackResult{
		PackID:      pack.ID,
		PackName:    pack.Name,
		EvaluatedAt: time.Now(),
	}

	for _, ctrl := range pack.Controls {
		cr := ControlResult{
			ID:    ctrl.ID,
			Title: ctrl.Title,
		}

		allPassed := true
		for _, check := range ctrl.Checks {
			er := e.evaluateCheck(check, state)
			cr.Checks = append(cr.Checks, er)
			if !er.Passed {
				allPassed = false
			}
		}

		cr.Passed = allPassed
		result.Controls = append(result.Controls, cr)
		result.TotalControls++
		if allPassed {
			result.PassedControls++
		} else {
			result.FailedControls++
		}
	}

	if result.TotalControls > 0 {
		result.Score = float64(result.PassedControls) / float64(result.TotalControls) * 100
	}

	return result, nil
}

// EvaluateAll runs all loaded packs against the system state.
func (e *Evaluator) EvaluateAll(state SystemState) ([]PackResult, error) {
	var results []PackResult
	for _, pack := range e.packs {
		r, err := e.Evaluate(pack.ID, state)
		if err != nil {
			return nil, err
		}
		results = append(results, *r)
	}
	return results, nil
}

func (e *Evaluator) evaluateCheck(check CheckDef, state SystemState) EvalResult {
	result := EvalResult{
		CheckType:   check.Type,
		Description: check.Description,
	}

	switch check.Type {
	case CheckEncryptionEnabled:
		result.Passed = state.EncryptionEnabled
		if !result.Passed {
			result.Details = "Database encryption is not active"
		}

	case CheckMFAEnabled:
		result.Passed = state.MFAEnabled
		if !result.Passed {
			result.Details = "Multi-factor authentication is not configured"
		}

	case CheckTLSEnabled:
		result.Passed = state.TLSEnabled
		if !result.Passed {
			result.Details = "TLS is not enabled for external listeners"
		}

	case CheckFIMEnabled:
		result.Passed = state.FIMEnabled
		if !result.Passed {
			result.Details = "File integrity monitoring is not active"
		}

	case CheckRBACEnabled:
		result.Passed = state.RBACEnabled
		if !result.Passed {
			result.Details = "Role-based access controls not enforced"
		}

	case CheckAlertingEnabled:
		result.Passed = state.AlertingEnabled
		if !result.Passed {
			result.Details = "Security alerting engine is not operational"
		}

	case CheckMerkleIntegrity:
		result.Passed = state.MerkleIntegrityValid
		if !result.Passed {
			result.Details = "Merkle tree integrity verification failed or not configured"
		}

	case CheckEvidenceLocker:
		result.Passed = state.EvidenceLockerAvail
		if !result.Passed {
			result.Details = "Evidence locker is not available"
		}

	case CheckCompliancePackLoaded:
		result.Passed = len(e.packs) > 0
		if !result.Passed {
			result.Details = "No compliance packs loaded"
		}

	case CheckAccessReview:
		// Passes if audit log has access review entries
		result.Passed = state.EventTypesPresent["access_review"]
		if !result.Passed {
			result.Details = "No access review events found in audit log"
		}

	case CheckVulnScan:
		result.Passed = state.EventTypesPresent["vulnerability_scan"]
		if !result.Passed {
			result.Details = "No vulnerability scan events found"
		}

	case CheckAuditLogExists:
		if types, ok := check.Params["event_types"]; ok {
			typeList, _ := toStringSlice(types)
			if len(typeList) == 1 && typeList[0] == "*" {
				result.Passed = state.AuditLogCount > 0
				if !result.Passed {
					result.Details = "No audit log entries found"
				}
			} else {
				allFound := true
				var missing []string
				for _, et := range typeList {
					if !state.EventTypesPresent[et] {
						allFound = false
						missing = append(missing, et)
					}
				}
				result.Passed = allFound
				if !result.Passed {
					result.Details = fmt.Sprintf("Missing event types: %v", missing)
				}
			}
		} else {
			result.Passed = state.AuditLogCount > 0
		}

	case CheckMinLogRetentionDays:
		days := 90 // default
		if d, ok := check.Params["days"]; ok {
			if df, ok := d.(int); ok {
				days = df
			} else if df, ok := d.(float64); ok {
				days = int(df)
			}
		}
		required := time.Duration(days) * 24 * time.Hour
		result.Passed = state.OldestLogAge >= required
		if !result.Passed {
			actualDays := int(state.OldestLogAge.Hours() / 24)
			result.Details = fmt.Sprintf("Oldest log is %d days old, required %d", actualDays, days)
		}

	case CheckFailedLoginThreshold:
		// This check passes if alerting is enabled (threshold detection is a feature of the alerting engine)
		result.Passed = state.AlertingEnabled
		if !result.Passed {
			result.Details = "Failed login threshold alerting not configured"
		}

	default:
		result.Passed = false
		result.Details = fmt.Sprintf("Unknown check type: %s", check.Type)
	}

	return result
}

func toStringSlice(v interface{}) ([]string, bool) {
	switch val := v.(type) {
	case []interface{}:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result, true
	case []string:
		return val, true
	default:
		return nil, false
	}
}
