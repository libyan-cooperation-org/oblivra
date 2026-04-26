//go:build windows

// Windows historical backfill — drives the BackfillCollector on Windows.
//
// Strategy: shell out to `wevtutil qe <Channel> /q:"*[System[TimeCreated[timediff(@SystemTime) <= <ms>]]]" /f:RenderedXml /e:Events`
// for the three core event log channels (System / Security / Application),
// stream the resulting XML, and emit one event per record.
//
// We use wevtutil instead of native EvtQuery (golang.org/x/sys/windows
// doesn't expose it cleanly) because:
//   - it ships with every supported Windows version, no extra deps
//   - it's the same tool sysadmins and security tooling already use
//   - it cleanly handles the lookback window via the timediff XPath
//
// Best-effort: any missing channel or wevtutil error is logged WARN
// and the scan continues with the next channel.

package agent

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// runPlatformBackfill walks Windows Event Log channels.
func (c *BackfillCollector) runPlatformBackfill(ctx context.Context, ch chan<- Event) (int, error) {
	if _, err := exec.LookPath("wevtutil.exe"); err != nil {
		return 0, fmt.Errorf("wevtutil not found: %w", err)
	}

	channels := []struct {
		name     string
		category string
	}{
		{"System", "system"},
		{"Security", "security"},
		{"Application", "system"},
	}

	count := 0
	for _, c2 := range channels {
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		case <-c.stop.C():
			return count, nil
		default:
		}
		n, err := c.scanWindowsChannel(ctx, ch, c2.name, c2.category)
		if err != nil {
			c.log.Warn("backfill: wevtutil channel scan failed",
				"channel", c2.name, "error", err)
			continue
		}
		count += n
	}
	return count, nil
}

// scanWindowsChannel queries one event log channel via wevtutil.
//
// The XPath filter:
//
//	*[System[TimeCreated[timediff(@SystemTime) <= <ms>]]]
//
// matches every entry whose timediff from "now" is at most `lookback`
// milliseconds. wevtutil renders each event as an `<Event>` XML
// fragment under an `<Events>` root.
func (c *BackfillCollector) scanWindowsChannel(
	ctx context.Context, ch chan<- Event,
	channel, category string,
) (int, error) {
	lookbackMs := strconv.FormatInt(c.lookback.Milliseconds(), 10)
	xpath := fmt.Sprintf(
		"*[System[TimeCreated[timediff(@SystemTime) <= %s]]]", lookbackMs,
	)

	cmd := exec.CommandContext(ctx, "wevtutil.exe",
		"qe", channel,
		"/q:"+xpath,
		"/f:RenderedXml",
		"/e:Events",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start wevtutil: %w", err)
	}

	count := 0

	// wevtutil with /e:Events wraps everything in <Events>...</Events>.
	// Stream-decode one <Event> at a time so we don't buffer megabytes.
	dec := xml.NewDecoder(bufio.NewReaderSize(stdout, 1024*1024))
	for {
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return count, ctx.Err()
		case <-c.stop.C():
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return count, nil
		default:
		}

		tok, err := dec.Token()
		if err != nil {
			break // EOF or decode error → end of stream
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Event" {
			continue
		}

		var ev winEvent
		if err := dec.DecodeElement(&ev, &se); err != nil {
			continue
		}

		ts, _ := time.Parse(time.RFC3339Nano, ev.System.TimeCreated.SystemTime)
		if ts.IsZero() {
			ts = time.Now()
		}

		// Windows EventLevel mapping — wevtutil renders Level as a
		// numeric string (1=Critical, 2=Error, 3=Warning, 4=Info, 5=Verbose).
		severity := MapSeverity("windows", ev.System.Level)

		extra := map[string]interface{}{
			"channel":     channel,
			"event_id":    ev.System.EventID,
			"provider":    ev.System.Provider.Name,
			"computer":    ev.System.Computer,
			"task":        ev.System.Task,
			"opcode":      ev.System.Opcode,
			"raw_message": ev.RenderingInfo.Message,
		}

		if err := c.emit(ctx, ch, ts, "eventlog/"+strings.ToLower(channel),
			"log.eventlog", severity, category,
			ev.RenderingInfo.Message, extra); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return count, err
		}
		count++
	}

	if err := cmd.Wait(); err != nil {
		c.log.Debug("backfill: wevtutil exit", "channel", channel, "error", err)
	}
	return count, nil
}

// winEvent matches the rendered-XML shape wevtutil produces. We only
// pull the fields we care about — there are dozens more available
// (UserData, EventData, Security, etc.) but they're not yet consumed
// by the dashboard.
type winEvent struct {
	XMLName xml.Name `xml:"Event"`
	System  struct {
		Provider struct {
			Name string `xml:"Name,attr"`
		} `xml:"Provider"`
		EventID     string `xml:"EventID"`
		Level       string `xml:"Level"`
		Task        string `xml:"Task"`
		Opcode      string `xml:"Opcode"`
		Computer    string `xml:"Computer"`
		TimeCreated struct {
			SystemTime string `xml:"SystemTime,attr"`
		} `xml:"TimeCreated"`
	} `xml:"System"`
	RenderingInfo struct {
		Message string `xml:"Message"`
	} `xml:"RenderingInfo"`
}
