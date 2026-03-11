package detection

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// BenchmarkEvent represents a single data point in a security dataset.
type BenchmarkEvent struct {
	Type        string            `json:"type"` // "log"
	Payload     string            `json:"payload"`
	Host        string            `json:"host"`
	SourceIP    string            `json:"source_ip,omitempty"`
	EventType   string            `json:"event_type,omitempty"`
	User        string            `json:"user,omitempty"`
	ExpectAlert bool              `json:"expect_alert"`
	ExpectedID  string            `json:"expected_id,omitempty"`
}

// BenchmarkResult contains the findings of a dataset run.
type BenchmarkResult struct {
	TotalEvents     int     `json:"total_events"`
	ExpectedAlerts  int     `json:"expected_alerts"`
	TotalAlerts     int     `json:"total_alerts"`
	TruePositives   int     `json:"true_positives"`
	FalsePositives  int     `json:"false_positives"`
	Precision       float64 `json:"precision"`
	Recall          float64 `json:"recall"`
	Duration        time.Duration `json:"duration"`
}

// BenchmarkRunner orchestrates the evaluation of the detection engine.
type BenchmarkRunner struct {
	evaluator *Evaluator
	log       *logger.Logger
}

// NewBenchmarkRunner initializes a runner with a live evaluator pointing to a rules directory.
func NewBenchmarkRunner(rulesDir string, log *logger.Logger) (*BenchmarkRunner, error) {
	eval, err := NewEvaluator(rulesDir, log)
	if err != nil {
		return nil, err
	}
	return &BenchmarkRunner{
		evaluator: eval,
		log:       log,
	}, nil
}

// RunBenchmark executes a dataset simulation.
func (r *BenchmarkRunner) RunBenchmark(filePath string) (*BenchmarkResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read dataset: %w", err)
	}

	var events []BenchmarkEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("unmarshal dataset: %w", err)
	}

	result := &BenchmarkResult{TotalEvents: len(events)}
	start := time.Now()

	for _, e := range events {
		if e.ExpectAlert {
			result.ExpectedAlerts++
		}

		// Map BenchmarkEvent to detection.Event
		evt := Event{
			EventType: e.EventType,
			SourceIP:  e.SourceIP,
			User:      e.User,
			HostID:    e.Host,
			RawLog:    e.Payload,
			Timestamp: time.Now(),
		}

		// If no EventType provided, we simulate the first step of parsing
		if evt.EventType == "" {
			if e.Payload != "" {
				// Naive type extraction for benchmarking if not provided
				if containsOne(e.Payload, "Failed password", "authentication failure") {
					evt.EventType = "failed_login"
				} else if containsOne(e.Payload, "Accepted password") {
					evt.EventType = "successful_login"
				}
			}
		}

		matches := r.evaluator.ProcessEvent(evt)
		if len(matches) > 0 {
			for _, m := range matches {
				result.TotalAlerts++
				if e.ExpectAlert && (e.ExpectedID == "" || m.RuleID == e.ExpectedID) {
					result.TruePositives++
				} else {
					result.FalsePositives++
				}
			}
		}
	}

	result.Duration = time.Since(start)
	if result.TotalAlerts > 0 {
		result.Precision = float64(result.TruePositives) / float64(result.TotalAlerts)
	}
	if result.ExpectedAlerts > 0 {
		result.Recall = float64(result.TruePositives) / float64(result.ExpectedAlerts)
	}

	return result, nil
}

func containsOne(s string, terms ...string) bool {
	for _, t := range terms {
		if contains(s, t) {
			return true
		}
	}
	return false
}

func contains(s, term string) bool {
	if len(term) == 0 {
		return true
	}
	if len(s) < len(term) {
		return false
	}
	// Linear scan — avoids importing strings to keep the package dependency-free
	for i := 0; i <= len(s)-len(term); i++ {
		if s[i:i+len(term)] == term {
			return true
		}
	}
	return false
}
