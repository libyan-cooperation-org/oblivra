package analytics

import (
	"context"
	"github.com/kingknull/oblivrashell/internal/monitoring"
)

// Engine defines the interface for terminal logging and forensic analysis.
type Engine interface {
	Open(dbPath string, encryptionKey []byte) error
	Ingest(ctx context.Context, sessionID, host, output string)
	Search(ctx context.Context, rawQuery string, mode string, limit, offset int) ([]map[string]interface{}, error)
	Close() error
	SaveConfig(ctx context.Context, key string, value string) error
	LoadConfig(ctx context.Context, key string) (string, error)
	SaveAlertEvent(ctx context.Context, triggerID, name, severity, host, sessionID, logLine string, sent bool)
	GetAlertHistory(ctx context.Context, limit int) ([]map[string]interface{}, error)
	IngestFrame(ctx context.Context, recordingID string, timestamp float64, frameType string, data string)
	SaveRecording(ctx context.Context, id, sessionID, hostLabel string, cols, rows int, duration float64, eventCount int, status string) error
	GetRecordingMeta(ctx context.Context, id string) (map[string]interface{}, error)
	ListRecordings(ctx context.Context) ([]map[string]interface{}, error)
	DeleteRecording(ctx context.Context, id string) error
	GetRecordingFrames(ctx context.Context, recordingID string) ([]map[string]interface{}, error)
	SearchRecordings(ctx context.Context, query string) ([]map[string]interface{}, error)
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
	ScanStream(ctx context.Context, sessionID, hostLabel, line string)
	ScanTelemetry(ctx context.Context, hostID string, t monitoring.HostTelemetry)
}
