package agent

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// MetricsCollector collects system-level metrics (CPU, memory, disk, network).
type MetricsCollector struct {
	hostname string
	interval time.Duration
	log      *logger.Logger
}

func NewMetricsCollector(hostname string, interval time.Duration, log *logger.Logger) *MetricsCollector {
	return &MetricsCollector{hostname: hostname, interval: interval, log: log}
}

func (c *MetricsCollector) Name() string { return "metrics" }

func (c *MetricsCollector) Start(ctx context.Context, ch chan<- Event) error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			ch <- c.collectSystemMetrics()
		}
	}
}

func (c *MetricsCollector) Stop() {}

func (c *MetricsCollector) collectSystemMetrics() Event {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return Event{
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    "metrics",
		Type:      "system_metrics",
		Host:      c.hostname,
		Data: map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"heap_alloc": m.HeapAlloc,
			"heap_sys":   m.HeapSys,
			"gc_count":   m.NumGC,
			"num_cpu":    runtime.NumCPU(),
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
		},
	}
}

// FileTailCollector tails log files and sends lines as events.
type FileTailCollector struct {
	hostname string
	paths    []string
	log      *logger.Logger
}

func NewFileTailCollector(hostname string, paths []string, log *logger.Logger) *FileTailCollector {
	return &FileTailCollector{hostname: hostname, paths: paths, log: log}
}

func (c *FileTailCollector) Name() string { return "file_tail" }

func (c *FileTailCollector) Start(ctx context.Context, ch chan<- Event) error {
	// For each path, check if it exists and start tailing
	for _, path := range c.paths {
		if _, err := os.Stat(path); err != nil {
			c.log.Warn("Skipping path %s: %v", path, err)
			continue
		}
		go c.tailFile(ctx, path, ch)
	}
	<-ctx.Done()
	return nil
}

func (c *FileTailCollector) Stop() {}

func (c *FileTailCollector) tailFile(ctx context.Context, path string, ch chan<- Event) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	// Seek to end
	f.Seek(0, 2)

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := f.Read(buf)
			if err != nil || n == 0 {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			ch <- Event{
				Timestamp: time.Now().Format(time.RFC3339),
				Source:    "file_tail",
				Type:      "log_line",
				Host:      c.hostname,
				Data: map[string]interface{}{
					"path": path,
					"line": string(buf[:n]),
				},
			}
		}
	}
}

// FIMCollector monitors critical files for integrity changes.
type FIMCollector struct {
	hostname string
	paths    []string
	log      *logger.Logger
}

func NewFIMCollector(hostname string, paths []string, log *logger.Logger) *FIMCollector {
	return &FIMCollector{hostname: hostname, paths: paths, log: log}
}

func (c *FIMCollector) Name() string { return "fim" }

func (c *FIMCollector) Start(ctx context.Context, ch chan<- Event) error {
	// Initial scan — record baselines
	baseline := make(map[string]int64) // path → modtime as unix
	for _, path := range c.paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		baseline[path] = info.ModTime().Unix()
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for _, path := range c.paths {
				info, err := os.Stat(path)
				if err != nil {
					continue
				}
				modTime := info.ModTime().Unix()
				oldMod, exists := baseline[path]
				if exists && modTime != oldMod {
					ch <- Event{
						Timestamp: time.Now().Format(time.RFC3339),
						Source:    "fim",
						Type:      "file_modified",
						Host:      c.hostname,
						Data: map[string]interface{}{
							"path":         path,
							"old_modified": time.Unix(oldMod, 0).Format(time.RFC3339),
							"new_modified": info.ModTime().Format(time.RFC3339),
							"size_bytes":   info.Size(),
						},
					}
					baseline[path] = modTime
				} else if !exists {
					baseline[path] = modTime
				}
			}
		}
	}
}

func (c *FIMCollector) Stop() {}

// EventLogCollector collects Windows Event Log entries.
type EventLogCollector struct {
	hostname string
	log      *logger.Logger
}

func NewEventLogCollector(hostname string, log *logger.Logger) *EventLogCollector {
	return &EventLogCollector{hostname: hostname, log: log}
}

func (c *EventLogCollector) Name() string { return "eventlog" }

func (c *EventLogCollector) Start(ctx context.Context, ch chan<- Event) error {
	// Windows Event Log collection — platform-specific
	// On non-Windows, this is a no-op
	if runtime.GOOS != "windows" {
		<-ctx.Done()
		return nil
	}

	// Poll every 10 seconds for new Security events
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Placeholder: Windows Event Log reading requires wevtutil or win32 API
			// Will be implemented with platform-specific build tags
			ch <- Event{
				Timestamp: time.Now().Format(time.RFC3339),
				Source:    "eventlog",
				Type:      "windows_event",
				Host:      c.hostname,
				Data: map[string]interface{}{
					"channel": "Security",
					"status":  "collector_running",
				},
			}
		}
	}
}

func (c *EventLogCollector) Stop() {}
