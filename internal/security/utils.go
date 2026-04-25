package security

import (
	"fmt"
	"time"
)

// parseTime parses an RFC3339 timestamp string.
// SA-10: returns an explicit error instead of silently returning zero time,
// which could cause callers to misinterpret epoch (1970) as a valid expiry.
func parseTime(ts string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}, fmt.Errorf("parseTime: invalid RFC3339 timestamp %q: %w", ts, err)
	}
	return t, nil
}
