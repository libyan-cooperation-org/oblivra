package agent

import (
	"context"
	"runtime"
	"time"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// EBPFCollector collects kernel-level security telemetry via eBPF on Linux.
// On non-Linux systems, this is a no-op placeholder.
type EBPFCollector struct {
	hostname string
	log      *logger.Logger
}

func NewEBPFCollector(hostname string, log *logger.Logger) *EBPFCollector {
	return &EBPFCollector{hostname: hostname, log: log}
}

func (c *EBPFCollector) Name() string { return "ebpf" }

func (c *EBPFCollector) Start(ctx context.Context, ch chan<- Event) error {
	if runtime.GOOS != "linux" {
		c.log.Info("eBPF is only available on Linux, skipping")
		<-ctx.Done()
		return nil
	}

	// On Linux, this would attach eBPF programs to:
	// - sys_enter_execve (process execution)
	// - tcp_connect (outbound connections)
	// - security_file_open (file access)
	// - sys_enter_ptrace (anti-debugging detection)
	//
	// Implementation requires:
	// - github.com/cilium/ebpf (pure Go eBPF library)
	// - Pre-compiled .o BPF programs or BTF-based CO-RE
	// - CAP_BPF or root privileges
	//
	// For now this is a stub that emits placeholder events.

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			ch <- Event{
				Timestamp: time.Now(),
				Source:    "ebpf",
				Type:      "kernel_telemetry",
				Host:      c.hostname,
				Data: map[string]interface{}{
					"probes_attached": 0,
					"events_captured": 0,
					"status":          "stub_not_implemented",
					"os":              runtime.GOOS,
				},
			}
		}
	}
}

func (c *EBPFCollector) Stop() {}
