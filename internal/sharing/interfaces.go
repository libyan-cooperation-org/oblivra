package sharing




// RecordingProvider defines the interface for managing TTY recordings.
type RecordingProvider interface {
	StartRecording(sessionID, hostLabel string, cols, rows int) (*ActiveRecording, error)
	RecordOutput(sessionID string, data []byte)
	RecordInput(sessionID string, data []byte)
	StopRecording(sessionID string) (*RecordingMetadata, error)
	ListRecordings() ([]RecordingMetadata, error)
	DeleteRecording(id string) error
	GetRecordingFrames(id string) ([]map[string]interface{}, error)
	SearchRecordings(query string) ([]map[string]interface{}, error)
	GetRecordingMeta(id string) (map[string]interface{}, error)
	ExportRecording(id, destPath string) error
}




// SessionExecutor defines the interface for interacting with active SSH/Terminal sessions.
type SessionExecutor interface {
	SendInput(sessionID, data string) error
}
