package analytics

import "github.com/kingknull/oblivrashell/internal/monitoring"

// Engine defines the interface for terminal logging and forensic analysis.
type Engine interface {
	Open(dbPath string, encryptionKey []byte) error
	Ingest(sessionID, host, output string)
	Search(rawQuery string, mode string, limit, offset int) ([]map[string]interface{}, error)
	Close() error
	SaveConfig(key string, value string) error
	LoadConfig(key string) (string, error)
	SaveAlertEvent(triggerID, name, severity, host, sessionID, logLine string, sent bool)
	GetAlertHistory(limit int) ([]map[string]interface{}, error)
	IngestFrame(recordingID string, timestamp float64, frameType string, data string)
	SaveRecording(id, sessionID, hostLabel string, cols, rows int, duration float64, eventCount int, status string) error
	GetRecordingMeta(id string) (map[string]interface{}, error)
	ListRecordings() ([]map[string]interface{}, error)
	DeleteRecording(id string) error
	GetRecordingFrames(recordingID string) ([]map[string]interface{}, error)
	SearchRecordings(query string) ([]map[string]interface{}, error)
}

// AlertProvider defines the interface for scanning logs and telemetry for threats.
type AlertProvider interface {
	AddTrigger(id, name, pattern, severity string) error
	RemoveTrigger(id string)
	GetTriggers() []Trigger
	GetMetricTriggers() []MetricTrigger
	UpdateMetricTrigger(mt MetricTrigger)
	RemoveMetricTrigger(id string)
	SetMetricTriggers(trigs []MetricTrigger)
	GetHistory() []AlertEvent
	ScanStream(sessionID, hostLabel, line string)
	ScanTelemetry(hostID string, t monitoring.HostTelemetry)
}
