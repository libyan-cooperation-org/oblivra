package agent

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// ─────────────────────────────────────────────────────────────────────────────
// stopOnce — safe single-close helper used by every collector's stop channel
// ─────────────────────────────────────────────────────────────────────────────

// stopOnce wraps a chan struct{} with sync.Once so calling Stop() multiple
// times (e.g. from hotToggle and from agent.Stop()) never panics.
type stopOnce struct {
	ch   chan struct{}
	once sync.Once
}

func newStopOnce() stopOnce {
	return stopOnce{ch: make(chan struct{})}
}

func (s *stopOnce) stop() {
	s.once.Do(func() { close(s.ch) })
}

func (s *stopOnce) C() <-chan struct{} {
	return s.ch
}

// ─────────────────────────────────────────────────────────────────────────────
// MetricsCollector — real OS-level system metrics using stdlib + /proc
// ─────────────────────────────────────────────────────────────────────────────

// MetricsCollector collects real OS-level system metrics.
// On Linux it reads /proc/meminfo, /proc/stat, /proc/net/dev and /proc/loadavg.
// On all platforms it also reports Go runtime stats.
type MetricsCollector struct {
	hostname     string
	agentID      string
	interval     time.Duration
	log          *logger.Logger
	stop         stopOnce
	prevCPUTotal uint64
	prevCPUIdle  uint64
}

func NewMetricsCollector(hostname, agentID string, interval time.Duration, log *logger.Logger) *MetricsCollector {
	return &MetricsCollector{
		hostname: hostname,
		agentID:  agentID,
		interval: interval,
		log:      log,
		stop:     newStopOnce(),
	}
}

func (c *MetricsCollector) Name() string { return "metrics" }
func (c *MetricsCollector) Stop()        { c.stop.stop() }

func (c *MetricsCollector) Start(ctx context.Context, ch chan<- Event) error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-c.stop.C():
			return nil
		case <-ticker.C:
			evt := c.collect()
			select {
			case ch <- evt:
			default:
				c.log.Warn("[metrics] channel full, dropping event")
			}
		}
	}
}

func (c *MetricsCollector) collect() Event {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	data := map[string]interface{}{
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"go_goroutines": runtime.NumGoroutine(),
		"go_heap_alloc": m.HeapAlloc,
		"go_heap_sys":   m.HeapSys,
		"go_gc_count":   m.NumGC,
		"num_cpu":       runtime.NumCPU(),
		"process_count": countProcesses(),
	}

	if runtime.GOOS == "linux" {
		if memFree, memTotal, memAvail := readProcMeminfo(); memTotal > 0 {
			data["mem_total_bytes"]    = memTotal
			data["mem_free_bytes"]     = memFree
			data["mem_available_bytes"]= memAvail
			data["mem_used_bytes"]     = memTotal - memFree
			data["mem_used_percent"]   = float64(memTotal-memAvail) / float64(memTotal) * 100
		}
		if cpuPct := c.readCPUPercent(); cpuPct >= 0 {
			data["cpu_percent"] = cpuPct
		}
		if l1, l5, l15 := readLoadAvg(); l1 >= 0 {
			data["load_avg_1"]  = l1
			data["load_avg_5"]  = l5
			data["load_avg_15"] = l15
		}
		if rxB, txB := readNetIO(); rxB >= 0 {
			data["net_bytes_recv"] = rxB
			data["net_bytes_sent"] = txB
		}
		if diskFree, diskTotal := statfsDiskUsage("/"); diskTotal > 0 {
			data["disk_total_bytes"]   = diskTotal
			data["disk_free_bytes"]    = diskFree
			data["disk_used_bytes"]    = diskTotal - diskFree
			data["disk_used_percent"]  = float64(diskTotal-diskFree) / float64(diskTotal) * 100
		}
	}

	return Event{
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    "metrics",
		Type:      "system_metrics",
		Host:      c.hostname,
		AgentID:   c.agentID,
		Data:      data,
	}
}

func readProcMeminfo() (free, total, available uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(parts[1], 10, 64)
		val *= 1024
		switch parts[0] {
		case "MemTotal:":
			total = val
		case "MemFree:":
			free = val
		case "MemAvailable:":
			available = val
		}
	}
	return
}

func (c *MetricsCollector) readCPUPercent() float64 {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return -1
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return -1
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return -1
	}
	var vals [10]uint64
	for i := 1; i < len(fields) && i <= 10; i++ {
		vals[i-1], _ = strconv.ParseUint(fields[i], 10, 64)
	}
	idle := vals[3] + vals[4]
	var total uint64
	for _, v := range vals {
		total += v
	}
	if c.prevCPUTotal == 0 {
		c.prevCPUTotal, c.prevCPUIdle = total, idle
		return 0
	}
	dTotal := total - c.prevCPUTotal
	dIdle := idle - c.prevCPUIdle
	c.prevCPUTotal, c.prevCPUIdle = total, idle
	if dTotal == 0 {
		return 0
	}
	return (1.0 - float64(dIdle)/float64(dTotal)) * 100
}

func readLoadAvg() (load1, load5, load15 float64) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return -1, -1, -1
	}
	parts := strings.Fields(string(data))
	if len(parts) < 3 {
		return -1, -1, -1
	}
	load1, _ = strconv.ParseFloat(parts[0], 64)
	load5, _ = strconv.ParseFloat(parts[1], 64)
	load15, _ = strconv.ParseFloat(parts[2], 64)
	return
}

func readNetIO() (rxBytes, txBytes int64) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return -1, -1
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Scan() // header 1
	scanner.Scan() // header 2
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 10 {
			continue
		}
		iface := strings.TrimSuffix(parts[0], ":")
		if iface == "lo" {
			continue
		}
		rx, _ := strconv.ParseInt(parts[1], 10, 64)
		tx, _ := strconv.ParseInt(parts[9], 10, 64)
		rxBytes += rx
		txBytes += tx
	}
	return
}

func countProcesses() int {
	if runtime.GOOS != "linux" {
		return 0
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			if _, err := strconv.Atoi(e.Name()); err == nil {
				n++
			}
		}
	}
	return n
}

// ─────────────────────────────────────────────────────────────────────────────
// FileTailCollector — line-buffered, rotation-aware log file tailing
// ─────────────────────────────────────────────────────────────────────────────

type FileTailCollector struct {
	hostname string
	agentID  string
	paths    []string
	log      *logger.Logger
	stop     stopOnce
}

func NewFileTailCollector(hostname, agentID string, paths []string, log *logger.Logger) *FileTailCollector {
	return &FileTailCollector{
		hostname: hostname,
		agentID:  agentID,
		paths:    paths,
		log:      log,
		stop:     newStopOnce(),
	}
}

func (c *FileTailCollector) Name() string { return "file_tail" }
func (c *FileTailCollector) Stop()        { c.stop.stop() }

func (c *FileTailCollector) Start(ctx context.Context, ch chan<- Event) error {
	for _, path := range c.paths {
		if _, err := os.Stat(path); err != nil {
			c.log.Warn("[file_tail] Skipping %s: %v", path, err)
			continue
		}
		go c.tailFile(ctx, path, ch)
	}
	select {
	case <-ctx.Done():
	case <-c.stop.C():
	}
	return nil
}

func (c *FileTailCollector) tailFile(ctx context.Context, path string, ch chan<- Event) {
	f, err := os.Open(path)
	if err != nil {
		c.log.Warn("[file_tail] Cannot open %s: %v", path, err)
		return
	}
	defer f.Close()

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		c.log.Warn("[file_tail] Seek failed %s: %v", path, err)
		return
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 128*1024), 128*1024)

	poll := time.NewTicker(250 * time.Millisecond)
	defer poll.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stop.C():
			return
		case <-poll.C:
			// Log rotation check
			if fi1, err1 := f.Stat(); err1 == nil {
				if fi2, err2 := os.Stat(path); err2 == nil && !os.SameFile(fi1, fi2) {
					c.log.Info("[file_tail] Rotation detected: %s", path)
					f.Close()
					if f, err = os.Open(path); err != nil {
						c.log.Warn("[file_tail] Reopen failed %s: %v", path, err)
						return
					}
					scanner = bufio.NewScanner(f)
					scanner.Buffer(make([]byte, 128*1024), 128*1024)
				}
			}
			// Drain new lines
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}
				evt := Event{
					Timestamp: time.Now().Format(time.RFC3339),
					Source:    "file_tail",
					Type:      "log_line",
					Host:      c.hostname,
					AgentID:   c.agentID,
					Data:      map[string]interface{}{"path": path, "line": line},
				}
				select {
				case ch <- evt:
				case <-ctx.Done():
					return
				case <-c.stop.C():
					return
				default:
					c.log.Warn("[file_tail] channel full, dropping line from %s", path)
				}
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FIMCollector — SHA-256 content-hash based integrity monitoring
// ─────────────────────────────────────────────────────────────────────────────

type fileBaseline struct {
	Hash    string
	ModTime int64
	Size    int64
}

// FIMCollector monitors files and directories for content changes.
// Uses SHA-256 rather than modtime — modtime is trivially reset by attackers.
type FIMCollector struct {
	hostname string
	agentID  string
	paths    []string
	log      *logger.Logger
	stop     stopOnce
}

func NewFIMCollector(hostname, agentID string, paths []string, log *logger.Logger) *FIMCollector {
	return &FIMCollector{
		hostname: hostname,
		agentID:  agentID,
		paths:    paths,
		log:      log,
		stop:     newStopOnce(),
	}
}

func (c *FIMCollector) Name() string { return "fim" }
func (c *FIMCollector) Stop()        { c.stop.stop() }

func (c *FIMCollector) Start(ctx context.Context, ch chan<- Event) error {
	baseline := make(map[string]fileBaseline)
	for _, p := range c.paths {
		if b, err := c.hashPath(p); err == nil {
			baseline[p] = b
		}
	}
	c.log.Info("[fim] Baseline established: %d paths", len(baseline))

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-c.stop.C():
			return nil
		case <-ticker.C:
			c.scan(baseline, ch)
		}
	}
}

func (c *FIMCollector) scan(baseline map[string]fileBaseline, ch chan<- Event) {
	for _, p := range c.paths {
		b, err := c.hashPath(p)
		if err != nil {
			if old, ok := baseline[p]; ok {
				c.emit(ch, "file_deleted", p, old.Hash, "", old.Size, 0)
				delete(baseline, p)
			}
			continue
		}
		old, exists := baseline[p]
		if !exists {
			c.emit(ch, "file_created", p, "", b.Hash, 0, b.Size)
			baseline[p] = b
			continue
		}
		if b.Hash != old.Hash {
			c.emit(ch, "file_modified", p, old.Hash, b.Hash, old.Size, b.Size)
			baseline[p] = b
		}
	}
}

func (c *FIMCollector) emit(ch chan<- Event, evType, path, oldHash, newHash string, oldSize, newSize int64) {
	select {
	case ch <- Event{
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    "fim",
		Type:      evType,
		Host:      c.hostname,
		AgentID:   c.agentID,
		Data: map[string]interface{}{
			"path":     path,
			"old_hash": oldHash,
			"new_hash": newHash,
			"old_size": oldSize,
			"new_size": newSize,
		},
	}:
	default:
		c.log.Warn("[fim] channel full, dropping event for %s", path)
	}
}

func (c *FIMCollector) hashPath(path string) (fileBaseline, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return fileBaseline{}, err
	}
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return fileBaseline{}, err
		}
		h := sha256.New()
		for _, e := range entries {
			fmt.Fprintf(h, "%s:%v:%d\n", e.Name(), e.IsDir(), e.Type())
		}
		return fileBaseline{
			Hash:    hex.EncodeToString(h.Sum(nil)),
			ModTime: info.ModTime().Unix(),
			Size:    info.Size(),
		}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return fileBaseline{}, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fileBaseline{}, err
	}
	return fileBaseline{
		Hash:    hex.EncodeToString(h.Sum(nil)),
		ModTime: info.ModTime().Unix(),
		Size:    info.Size(),
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// EventLogCollector — Windows Event Log (platform-aware stub)
// ─────────────────────────────────────────────────────────────────────────────

type EventLogCollector struct {
	hostname string
	agentID  string
	log      *logger.Logger
	stop     stopOnce
}

func NewEventLogCollector(hostname, agentID string, log *logger.Logger) *EventLogCollector {
	return &EventLogCollector{
		hostname: hostname,
		agentID:  agentID,
		log:      log,
		stop:     newStopOnce(),
	}
}

func (c *EventLogCollector) Name() string { return "eventlog" }
func (c *EventLogCollector) Stop()        { c.stop.stop() }

func (c *EventLogCollector) Start(ctx context.Context, ch chan<- Event) error {
	if runtime.GOOS != "windows" {
		select {
		case <-ctx.Done():
		case <-c.stop.C():
		}
		return nil
	}
	// Windows-specific implementation in collectors_windows.go provides real
	// Event Log streaming. This fallback heartbeat fires if that file isn't built.
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-c.stop.C():
			return nil
		case <-ticker.C:
			select {
			case ch <- Event{
				Timestamp: time.Now().Format(time.RFC3339),
				Source:    "eventlog",
				Type:      "windows_event_heartbeat",
				Host:      c.hostname,
				AgentID:   c.agentID,
				Data:      map[string]interface{}{"channel": "Security", "status": "running"},
			}:
			default:
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// readProcFile — shared /proc reader used by eBPF fallback and collectors
// ─────────────────────────────────────────────────────────────────────────────

// readProcFile reads a small /proc file and returns its content trimmed of
// trailing newlines and null bytes. Returns empty string on error.
func readProcFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(data), "\n\x00")
}
