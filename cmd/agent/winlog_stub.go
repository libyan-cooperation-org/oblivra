//go:build !windows

package main

import (
	"context"
	"fmt"
)

// runWinlog is unavailable on non-Windows platforms.
func (t *Tailer) runWinlog(_ context.Context) error {
	return fmt.Errorf("winlog: Windows Event Log is only available on Windows (current OS: linux/darwin/other)")
}

// discoverWinlogChannels returns nil on non-Windows platforms.
func discoverWinlogChannels() []string { return nil }
