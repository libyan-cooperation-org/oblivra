package services

import (
	"log/slog"
	"runtime"
	"time"
)

const Version = "0.1.0-phase0"

type SystemInfo struct {
	Version    string `json:"version"`
	GoVersion  string `json:"goVersion"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	NumCPU     int    `json:"numCpu"`
	StartedAt  string `json:"startedAt"`
	Goroutines int    `json:"goroutines"`
}

type Health struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type SystemService struct {
	log       *slog.Logger
	startedAt time.Time
}

func NewSystemService(log *slog.Logger) *SystemService {
	return &SystemService{log: log, startedAt: time.Now().UTC()}
}

func (s *SystemService) ServiceName() string { return "SystemService" }

func (s *SystemService) Info() SystemInfo {
	return SystemInfo{
		Version:    Version,
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		NumCPU:     runtime.NumCPU(),
		StartedAt:  s.startedAt.Format(time.RFC3339),
		Goroutines: runtime.NumGoroutine(),
	}
}

func (s *SystemService) Ping() Health {
	return Health{Status: "ok", Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}
}

// OsInstallDate returns the approximate date the OS was installed.
// Implementation is in os_install_windows.go and os_install_other.go.
func (s *SystemService) OsInstallDate() time.Time {
	return osInstallDate()
}
