package forensics

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/kingknull/oblivrashell/internal/logger"
)

// Collector defines the interface for trusted forensic evidence acquisition.
type Collector interface {
	// AcquireDisk streams a raw image of a block device or logical volume.
	AcquireDisk(ctx context.Context, devicePath string, writer io.Writer) error
	
	// AcquireMemory dumps the physical memory of the local machine.
	AcquireMemory(ctx context.Context, writer io.Writer) error
}

// LocalCollector implements forensic acquisition using native OS calls.
type LocalCollector struct {
	log *logger.Logger
}

// NewLocalCollector creates a new forensic collector for the local system.
func NewLocalCollector(log *logger.Logger) *LocalCollector {
	return &LocalCollector{
		log: log.WithPrefix("forensics-collector"),
	}
}

// AcquireDisk performs a raw sector-by-sector copy of a target device.
// [CAUTION] Requires administrative privileges (root/SYSTEM).
func (c *LocalCollector) AcquireDisk(ctx context.Context, devicePath string, writer io.Writer) error {
	c.log.Info("[FORENSICS] Starting disk acquisition: %s", devicePath)
	
	f, err := os.Open(devicePath)
	if err != nil {
		return fmt.Errorf("open device: %w", err)
	}
	defer f.Close()

	// Use a large buffer for high-throughput acquisition (1MB)
	buf := make([]byte, 1024*1024)
	totalBytes := int64(0)

	for {
		select {
		case <-ctx.Done():
			c.log.Warn("[FORENSICS] Disk acquisition cancelled")
			return ctx.Err()
		default:
			n, err := f.Read(buf)
			if n > 0 {
				if _, werr := writer.Write(buf[:n]); werr != nil {
					return fmt.Errorf("write image: %w", werr)
				}
				totalBytes += int64(n)
			}
			if err == io.EOF {
				c.log.Info("[FORENSICS] Disk acquisition complete. Total bytes: %d", totalBytes)
				return nil
			}
			if err != nil {
				return fmt.Errorf("read device: %w", err)
			}
		}
	}
}

// AcquireMemory dumps physical memory. 
// Implementation varies significantly by OS (e.g., /dev/mem on Linux, WinPMEM on Windows).
func (c *LocalCollector) AcquireMemory(ctx context.Context, writer io.Writer) error {
	c.log.Info("[FORENSICS] Starting memory dump")

	switch runtime.GOOS {
	case "linux":
		return c.acquireMemoryLinux(ctx, writer)
	case "windows":
		return c.acquireMemoryWindows(ctx, writer)
	default:
		return fmt.Errorf("memory acquisition not supported on %s", runtime.GOOS)
	}
}

func (c *LocalCollector) acquireMemoryLinux(ctx context.Context, writer io.Writer) error {
	// Traditional Linux memory acquisition via /dev/mem or /proc/kcore
	return c.AcquireDisk(ctx, "/dev/mem", writer)
}

func (c *LocalCollector) acquireMemoryWindows(ctx context.Context, writer io.Writer) error {
	// Windows memory acquisition typically requires a kernel driver.
	// For this implementation, we assume a trusted driver (like WinPMEM) is reachable via a device node.
	return c.AcquireDisk(ctx, "\\\\.\\pmem", writer)
}
