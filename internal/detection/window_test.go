package detection_test

import (
	"testing"
	"time"

	"github.com/kingknull/oblivrashell/internal/detection"
	"github.com/kingknull/oblivrashell/internal/logger"
)

func TestDetection_TumblingWindow(t *testing.T) {
	tmpDir := t.TempDir()
	writeRule(t, tmpDir, "tumbling_test.yaml", `
id: "tumbling_test"
name: "Tumbling Window Test"
severity: "medium"
type: "threshold"
threshold: 3
window_sec: 10
window_type: "tumbling"
conditions:
  EventType: "test_event"
`)
	log := logger.NewStdoutLogger()
	ev, err := detection.NewEvaluator(tmpDir, log)
	if err != nil { t.Fatalf("NewEvaluator: %v", err) }

	// Base time at a 10s boundary
	baseTime := time.Date(2026, 4, 18, 20, 0, 0, 0, time.UTC)

	sendAt := func(offset int) []detection.Match {
		ts := baseTime.Add(time.Duration(offset) * time.Second).Format(time.RFC3339)
		return ev.ProcessEvent(detection.Event{
			EventType: "test_event",
			Timestamp: ts,
		})
	}

	if len(sendAt(2)) > 0 { t.Fatal("unexpected alert at 2s") }
	if len(sendAt(5)) > 0 { t.Fatal("unexpected alert at 5s") }
	if len(sendAt(12)) > 0 { t.Fatal("unexpected alert at 12s") }
	if len(sendAt(15)) > 0 { t.Fatal("unexpected alert at 15s") }
	matches := sendAt(18)
	if len(matches) == 0 { t.Fatal("expected alert at 18s") }
}

func TestDetection_WatermarkTolerance(t *testing.T) {
	tmpDir := t.TempDir()
	writeRule(t, tmpDir, "watermark_test.yaml", `
id: "watermark_test"
name: "Watermark Test"
severity: "medium"
type: "threshold"
threshold: 2
window_sec: 60
watermark_sec: 10
conditions:
  EventType: "test_event"
`)
	log := logger.NewStdoutLogger()
	ev, err := detection.NewEvaluator(tmpDir, log)
	if err != nil { t.Fatalf("NewEvaluator: %v", err) }

	baseTime := time.Date(2026, 4, 18, 20, 0, 0, 0, time.UTC)

	sendAt := func(offset int) []detection.Match {
		ts := baseTime.Add(time.Duration(offset) * time.Second).Format(time.RFC3339)
		return ev.ProcessEvent(detection.Event{
			EventType: "test_event",
			Timestamp: ts,
		})
	}

	sendAt(20)
	matches := sendAt(15)
	if len(matches) == 0 { t.Fatal("expected alert: 15s event should be within watermark") }

	ev, _ = detection.NewEvaluator(tmpDir, log)
	sendAt(30)
	matches = sendAt(15)
	if len(matches) > 0 { t.Fatal("unexpected alert: 15s event should be dropped") }
}

func TestDetection_DeterministicReplay(t *testing.T) {
	tmpDir := t.TempDir()
	writeRule(t, tmpDir, "replay_test.yaml", `
id: "replay_test"
name: "Replay Test"
severity: "medium"
type: "threshold"
threshold: 3
window_sec: 60
watermark_sec: 30
conditions:
  EventType: "test_event"
`)
	log := logger.NewStdoutLogger()
	ev, err := detection.NewEvaluator(tmpDir, log)
	if err != nil { t.Fatalf("NewEvaluator: %v", err) }

	baseTime := time.Date(2026, 4, 18, 20, 0, 0, 0, time.UTC)

	ev.ProcessEvent(detection.Event{EventType: "test_event", Timestamp: baseTime.Add(20 * time.Second).Format(time.RFC3339)})
	ev.ProcessEvent(detection.Event{EventType: "test_event", Timestamp: baseTime.Add(10 * time.Second).Format(time.RFC3339)})
	matches := ev.ProcessEvent(detection.Event{EventType: "test_event", Timestamp: baseTime.Add(15 * time.Second).Format(time.RFC3339)})

	if len(matches) == 0 {
		t.Fatalf("expected alert after 3 events. Rules: %d", len(ev.GetRules()))
	}
}
