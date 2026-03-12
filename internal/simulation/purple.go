package simulation



// CoverageReport summarizes detection posture against MITRE ATT&CK.
type CoverageReport struct {
	TotalTechniques   int              `json:"total_techniques"`
	CoveredTechniques int              `json:"covered_techniques"`
	GapTechniques     int              `json:"gap_techniques"`
	CoveragePercent   float64          `json:"coverage_percent"`
	TacticBreakdown   []TacticCoverage `json:"tactic_breakdown"`
	CoveredIDs        []string         `json:"covered_ids"`
	GapIDs            []string         `json:"gap_ids"`
}

// TacticCoverage represents per-tactic detection coverage.
type TacticCoverage struct {
	TacticID   string            `json:"tactic_id"`
	Tactic     string            `json:"tactic"`
	Total      int               `json:"total"`
	Covered    int               `json:"covered"`
	Percent    float64           `json:"percent"`
	Techniques []TechniqueStatus `json:"techniques"`
}

// TechniqueStatus indicates whether a specific technique has active detection.
type TechniqueStatus struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Covered  bool   `json:"covered"`
	Scenario string `json:"scenario,omitempty"` // linked scenario ID
}

// ValidationRun captures the outcome of a continuous validation pass.
type ValidationRun struct {
	ID             string    `json:"id"`
	Timestamp      string    `json:"timestamp"`
	TotalScenarios int       `json:"total_scenarios"`
	Detected       int       `json:"detected"`
	Missed         int       `json:"missed"`
	PassRate       float64   `json:"pass_rate"`
	CoverageIndex  float64   `json:"coverage_index"`
	DurationMs     int64     `json:"duration_ms"`
}

// PurpleTeamReport is the composite scoring model for the Purple Team dashboard.
type PurpleTeamReport struct {
	ResilienceScore   float64         `json:"resilience_score"`
	ResilienceGrade   string          `json:"resilience_grade"`
	DetectionRate     float64         `json:"detection_rate"`
	CoverageIndex     float64         `json:"coverage_index"`
	MeanResponseMs    int64           `json:"mean_response_ms"`
	LastValidation    *ValidationRun  `json:"last_validation,omitempty"`
	ValidationHistory []ValidationRun `json:"validation_history"`
	Coverage          CoverageReport  `json:"coverage"`
}

// gradeFromScore converts a 0-100 resilience score to a letter grade.
func gradeFromScore(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}
