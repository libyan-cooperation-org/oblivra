package parsers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRealSamples walks internal/parsers/testdata and confirms every line in
// every sample file parses without going through the plain fallback. Catches
// production-format drift the synthetic tests would miss.
func TestRealSamples(t *testing.T) {
	root := "testdata"
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
			ev, err := Parse(line, FormatAuto)
			if err != nil || ev == nil {
				t.Errorf("%s line %d: parse failed: %v", path, i+1, err)
				continue
			}
			expected := strings.Split(filepath.ToSlash(path), "/")[1]
			// EventType for plain fallback is "plain" — anything else means the
			// real parser succeeded.
			if ev.EventType == "plain" {
				t.Errorf("%s line %d (expected %s): fell back to plain", path, i+1, expected)
			}
			if ev.Raw == "" {
				t.Errorf("%s line %d: raw not preserved", path, i+1)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
