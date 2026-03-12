package sharing

import "time"

func parseTime(ts string) time.Time {
	t, _ := time.Parse(time.RFC3339, ts)
	if t.IsZero() {
		t, _ = time.Parse("2006-01-02 15:04:05", ts)
	}
	return t
}
