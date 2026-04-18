package sharing




// RecordingProvider defines the interface for managing TTY recordings.
type RecordingProvider interface {
	StartRecording(tenantID, sessionID, hostLabel string, cols, rows int) (*ActiveRecording, error)
	RecordOutput(sessionID string, data []byte)
	RecordInput(sessionID string, data []byte)
	StopRecording(sessionID string) (*RecordingMetadata, error)
	ListRecordings(tenantID string) ([]RecordingMetadata, error)
	DeleteRecording(tenantID, id string) error
	GetRecordingFrames(tenantID, id string) ([]map[string]interface{}, error)
	SearchRecordings(tenantID, query string) ([]map[string]interface{}, error)
	GetRecordingMeta(tenantID, id string) (map[string]interface{}, error)
	ExportRecording(tenantID, id, destPath string) error
}




// SessionExecutor defines the interface for interacting with active SSH/Terminal sessions.
type SessionExecutor interface {
	SendInput(sessionID, data string) error
}
