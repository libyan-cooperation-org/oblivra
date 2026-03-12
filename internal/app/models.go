package app

import (
	"context"
	"sync"
)

// FileInfo represents a file on either a local or remote system
type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	Mode    string    `json:"mode"`
	ModTime string    `json:"mod_time"`
	IsDir   bool      `json:"is_dir"`
}

type TransferType string
type TransferStatus string

const (
	TransferDownload TransferType = "download"
	TransferUpload   TransferType = "upload"

	StatusQueued     TransferStatus = "queued"
	StatusInProgress TransferStatus = "in_progress"
	StatusCompleted  TransferStatus = "completed"
	StatusFailed     TransferStatus = "failed"
	StatusCancelled  TransferStatus = "cancelled"
)

type TransferJob struct {
	ID          string         `json:"id"`
	SessionID   string         `json:"session_id"`
	Type        TransferType   `json:"type"`
	RemotePath  string         `json:"remote_path"`
	LocalPath   string         `json:"local_path"`
	Filename    string         `json:"filename"`
	TotalBytes  int64          `json:"total_bytes"`
	BytesCopied int64          `json:"bytes_copied"`
	SpeedBytesS float64        `json:"speed_bytes_s"`
	Status      TransferStatus `json:"status"`
	Error       string         `json:"error,omitempty"`
	StartedAt   string         `json:"started_at"`
	CompletedAt *string        `json:"completed_at,omitempty"`
	cancelFn    context.CancelFunc
	ctx         context.Context
}

// MultiExecResult holds the result from one host
type MultiExecResult struct {
	HostID    string `json:"host_id"`
	HostLabel string `json:"host_label"`
	Hostname  string `json:"hostname"`
	Output    string `json:"output"`
	ExitCode  int    `json:"exit_code"`
	Error     string `json:"error,omitempty"`
	Duration  string `json:"duration"`
	Status    string `json:"status"` // "success", "error", "timeout", "pending", "running"
}

// MultiExecJob tracks a multi-host execution job
type MultiExecJob struct {
	ID        string            `json:"id"`
	Command   string            `json:"command"`
	HostIDs   []string          `json:"host_ids"`
	Results   []MultiExecResult `json:"results"`
	StartedAt string            `json:"started_at"`
	EndedAt   *string           `json:"ended_at,omitempty"`
	Status    string            `json:"status"` // "running", "completed", "partial"
	mu        sync.Mutex
}

// AIResponse holds the result from an AI model
type AIResponse struct {
	Text       string `json:"text"`
	RawCommand string `json:"raw_command,omitempty"`
}
