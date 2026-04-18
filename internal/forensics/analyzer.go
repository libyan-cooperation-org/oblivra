package forensics

import (
	"fmt"
	"math"

	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/security"
)

// EntropySegment represents a chunk of data with its entropy value
type EntropySegment struct {
	Offset  int64   `json:"offset"`
	Size    int     `json:"size"`
	Entropy float64 `json:"entropy"`
}

// ForensicReport provides a deep analysis of a suspicious file or entity
type ForensicReport struct {
	Path           string           `json:"path"`
	TotalSize      int64            `json:"total_size"`
	OverallEntropy float64          `json:"overall_entropy"`
	Segments       []EntropySegment `json:"segments"`
	RiskScore      int              `json:"risk_score"`
	Mitigation     string           `json:"mitigation"`
}

// ForensicAnalyzer conducts deep investigative analysis
type ForensicAnalyzer struct {
	log *logger.Logger
}

// NewForensicAnalyzer creates a new forensic analyzer
func NewForensicAnalyzer(log *logger.Logger) *ForensicAnalyzer {
	return &ForensicAnalyzer{
		log: log,
	}
}

// AnalyzeFile performs a deep entropy profile of a target file
func (a *ForensicAnalyzer) AnalyzeFile(path string, data []byte) (*ForensicReport, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided for analysis")
	}

	overall := security.CalculateEntropy(data)

	// Segmented analysis
	segmentSize := 1024
	if len(data) < segmentSize {
		segmentSize = len(data)
	}

	segments := []EntropySegment{}
	for i := 0; i < len(data); i += segmentSize {
		end := i + segmentSize
		if end > len(data) {
			end = len(data)
		}

		ent := security.CalculateEntropy(data[i:end])
		segments = append(segments, EntropySegment{
			Offset:  int64(i),
			Size:    end - i,
			Entropy: ent,
		})
	}

	risk := 0
	if overall > 7.5 {
		risk = 90
	} else if overall > 6.0 {
		risk = 50
	} else {
		risk = 10
	}

	report := &ForensicReport{
		Path:           path,
		TotalSize:      int64(len(data)),
		OverallEntropy: math.Round(overall*100) / 100,
		Segments:       segments,
		RiskScore:      risk,
		Mitigation:     a.suggestMitigation(risk),
	}

	return report, nil
}

func (a *ForensicAnalyzer) suggestMitigation(risk int) string {
	if risk >= 80 {
		return "IMMEDIATE ISOLATION: High confidence encryption detected. Potential ransomware activity."
	}
	if risk >= 50 {
		return "INVESTIGATE: Elevated entropy levels. Check for compressed archives or custom binary formats."
	}
	return "BASELINE: Normal entropy levels observed."
}
