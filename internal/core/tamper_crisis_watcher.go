package core

// Tamper-driven crisis auto-arm.
//
// When ≥ 2 unique hosts emit `tamper:detected` with severity
// "critical" within a rolling 1-hour window, engage Crisis Mode
// platform-wide. Real-world ransomware behaviour is rolling pwn
// across the fleet; the second tampered host is a strong signal that
// the first wasn't a one-off.
//
// Engagement publishes a `crisis:arm` bus event that crisisStore
// (frontend) and the Decision Panel pick up automatically.

import (
	"context"
	"sync"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// startTamperCrisisWatcher subscribes to tamper:detected events and
// triggers crisis:arm when the rolling-1h count of unique tampered
// hosts crosses the threshold. Single-fire per crisis episode — the
// watcher resets only after stand-down so we don't spam the same arm
// signal once active.
func startTamperCrisisWatcher(ctx context.Context, bus *eventbus.Bus, log *logger.Logger) {
	const (
		windowSeconds = 60 * 60 // 1 hour
		threshold     = 2
	)

	var (
		mu     sync.Mutex
		seen   = map[string]time.Time{} // agent_id → first seen in window
		armed  bool
	)

	prefix := log.WithPrefix("tamper.crisis")

	bus.Subscribe(eventbus.EventType("crisis:stand_down"), func(_ eventbus.Event) {
		mu.Lock()
		armed = false
		// Don't clear `seen` — the operator may want continuity in the
		// audit trail. The cleanup ticker below trims the map by age.
		mu.Unlock()
		prefix.Info("crisis stand-down received; auto-arm re-enabled")
	})

	bus.Subscribe(eventbus.EventType("tamper:detected"), func(ev eventbus.Event) {
		details, ok := ev.Data.(map[string]interface{})
		if !ok {
			return
		}
		severity, _ := details["severity"].(string)
		// Only critical-severity tamper indicators count toward the
		// crisis threshold — heartbeat-missed alerts at info severity
		// don't escalate (they're often a flapping network, not an
		// attack).
		if severity != "critical" {
			return
		}
		agentID, _ := details["agent_id"].(string)
		if agentID == "" {
			return
		}

		now := time.Now()
		mu.Lock()
		seen[agentID] = now
		// Trim out-of-window entries so the threshold check is honest.
		fresh := 0
		for id, t := range seen {
			if now.Sub(t).Seconds() > windowSeconds {
				delete(seen, id)
				continue
			}
			_ = id
			fresh++
		}
		shouldArm := !armed && fresh >= threshold
		if shouldArm {
			armed = true
		}
		mu.Unlock()

		if shouldArm {
			prefix.Warn("AUTO-ARMING crisis mode: %d unique hosts tampered in the last 1h", fresh)
			bus.Publish("crisis:arm", map[string]interface{}{
				"reason":    "tamper-spike",
				"detail":    "≥2 unique hosts emitted critical tamper indicators in the last hour",
				"hosts":     fresh,
				"timestamp": now.UTC().Format(time.RFC3339Nano),
			})
		}
	})

	// Periodic cleanup goroutine — drains stale entries even when no
	// tamper events fire so the map doesn't grow over a quiet day.
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				mu.Lock()
				for id, ts := range seen {
					if now.Sub(ts).Seconds() > windowSeconds {
						delete(seen, id)
					}
				}
				mu.Unlock()
			}
		}
	}()
}
