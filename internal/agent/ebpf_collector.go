//go:build !linux
// +build !linux

package agent

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// EBPFCollector is a no-op placeholder on non-Linux platforms.
// The real implementation lives in ebpf_collector_linux.go.
type EBPFCollector struct {
	hostname string
	log      *logger.Logger
}

func NewEBPFCollector(hostname string, log *logger.Logger) *EBPFCollector {
	return &EBPFCollector{hostname: hostname, log: log}
}

func (c *EBPFCollector) Name() string { return "ebpf" }

func (c *EBPFCollector) Start(ctx context.Context, ch chan<- Event) error {
	c.log.Info("[eBPF] Not available on %s — collector disabled", runtimeGOOS())
	<-ctx.Done()
	return nil
}

func (c *EBPFCollector) Stop() {}
