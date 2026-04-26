//go:build !linux && !windows && !darwin

// Fallback: on every other GOOS (FreeBSD, OpenBSD, etc.) the historical
// backfill is a no-op. The agent still functions — only the
// backfill collector reports zero events and immediately marks itself
// complete.
//
// This stub exists so `go build ./...` succeeds on every GOOS the
// rest of the project supports.

package agent

import "context"

func (c *BackfillCollector) runPlatformBackfill(_ context.Context, _ chan<- Event) (int, error) {
	c.log.Info("backfill: no platform implementation for this GOOS, skipping")
	return 0, nil
}
