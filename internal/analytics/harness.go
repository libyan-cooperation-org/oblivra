package analytics

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kingknull/oblivrashell/internal/monitoring"
	"github.com/kingknull/oblivrashell/internal/notifications"
)

// BenchmarkEvent represents a single data point in a security dataset.
type BenchmarkEvent struct {
	Type        string                   `json:"type"` // "log" or "telemetry"
	Payload     string                   `json:"payload,omitempty"`
	Telemetry   *monitoring.HostTelemetry `json:"telemetry,omitempty"`
	Host        string                   `json:"host"`
	SessionID   string                   `json:"session_id,omitempty"`
	ExpectAlert bool                     `json:"expect_alert"`
	ExpectedID  string                   `json:"expected_id,omitempty"`
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
}

// BenchmarkRunner orchestrates the evaluation of the detection engine.
type BenchmarkRunner struct {
	engine *AlertEngine
}

// NewBenchmarkRunner initializes a runner with a clean engine and mock dependencies.
func NewBenchmarkRunner() *BenchmarkRunner {
	// Use mock deps to avoid external side-effects during benchmarking
	mockNotifier := &mockNotifier{}
	mockAnalytics := &mockAnalytics{}
	
	ae := NewAlertEngine(mockNotifier, mockAnalytics)
	ae.ThrottlingEnabled = false

	return &BenchmarkRunner{
		engine: ae,
	}
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

	for _, e := range events {
		if e.ExpectAlert {
			result.ExpectedAlerts++
		}

		preHistory := len(r.engine.GetHistory())
		
		if e.Type == "log" {
			r.engine.ScanStream(e.SessionID, e.Host, e.Payload)
		} else if e.Type == "telemetry" && e.Telemetry != nil {
			r.engine.ScanTelemetry(e.Host, *e.Telemetry)
		}

		postHistory := r.engine.GetHistory()
		if len(postHistory) > preHistory {
			result.TotalAlerts++
			if e.ExpectAlert {
				result.TruePositives++
			} else {
				result.FalsePositives++
			}
		}
	}

	if result.TotalAlerts > 0 {
		result.Precision = float64(result.TruePositives) / float64(result.TotalAlerts)
	}
	if result.ExpectedAlerts > 0 {
		result.Recall = float64(result.TruePositives) / float64(result.ExpectedAlerts)
	}

	return result, nil
}

// --- Mocks ---

type mockNotifier struct {
	config notifications.NotificationConfig
}
func (m *mockNotifier) SendAlert(title, message string) {}
func (m *mockNotifier) UpdateConfig(cfg notifications.NotificationConfig) { m.config = cfg }
func (m *mockNotifier) GetConfig() notifications.NotificationConfig { return m.config }

type mockAnalytics struct{}
func (m *mockAnalytics) Open(dbPath string, encryptionKey []byte) error { return nil }
func (m *mockAnalytics) Ingest(sessionID, host, output string) {}
func (m *mockAnalytics) Search(rawQuery string, mode string, limit, offset int) ([]map[string]interface{}, error) { return nil, nil }
func (m *mockAnalytics) Close() error { return nil }
func (m *mockAnalytics) SaveConfig(key string, value string) error { return nil }
func (m *mockAnalytics) LoadConfig(key string) (string, error) { return "", nil }
func (m *mockAnalytics) SaveAlertEvent(triggerID, name, severity, host, sessionID, logLine string, sent bool) {}
func (m *mockAnalytics) GetAlertHistory(limit int) ([]map[string]interface{}, error) { return nil, nil }
func (m *mockAnalytics) IngestFrame(recordingID string, timestamp float64, frameType string, data string) {}
func (m *mockAnalytics) SaveRecording(id, sessionID, hostLabel string, cols, rows int, duration float64, eventCount int, status string) error { return nil }
func (m *mockAnalytics) GetRecordingMeta(id string) (map[string]interface{}, error) { return nil, nil }
func (m *mockAnalytics) ListRecordings() ([]map[string]interface{}, error) { return nil, nil }
func (m *mockAnalytics) DeleteRecording(id string) error { return nil }
func (m *mockAnalytics) GetRecordingFrames(recordingID string) ([]map[string]interface{}, error) { return nil, nil }
func (m *mockAnalytics) SearchRecordings(query string) ([]map[string]interface{}, error) { return nil, nil }
