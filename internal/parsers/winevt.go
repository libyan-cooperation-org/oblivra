package parsers

import (
	"encoding/xml"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// Phase 98 — Windows Event Log XML parser.
//
// Parses the Windows XML event shape produced by Windows Event Collector
// (WEC) forwarding, `wevtutil qe /f:xml`, Winlogbeat's original XML, and
// the OBLIVRA agent's future winlog input. One event per line (the usual
// shipper framing); the <Events> batch wrapper is handled by callers that
// split it, or by a line holding exactly one <Event> element.
//
//	<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'>
//	  <System>
//	    <Provider Name='Microsoft-Windows-Security-Auditing' .../>
//	    <EventID>4624</EventID>
//	    <Level>0</Level>
//	    <TimeCreated SystemTime='2026-08-01T12:00:00.000000000Z'/>
//	    <Channel>Security</Channel>
//	    <Computer>DC01.corp.local</Computer>
//	  </System>
//	  <EventData>
//	    <Data Name='TargetUserName'>alice</Data>
//	    <Data Name='IpAddress'>10.0.0.5</Data>
//	  </EventData>
//	</Event>
//
// Every <Data Name=...> lands verbatim in Fields — the reconstruction
// engine's session/cmdline classifiers key on TargetUserName / IpAddress /
// CommandLine / ParentProcessName, so no renaming happens here. Message is
// synthesised as "EventID <id> <provider>" (plus the well-known label for
// classic security IDs) because the XML shape carries no rendered text.

type winEvent struct {
	System struct {
		Provider struct {
			Name string `xml:"Name,attr"`
		} `xml:"Provider"`
		EventID     string `xml:"EventID"`
		Level       string `xml:"Level"`
		Task        string `xml:"Task"`
		Keywords    string `xml:"Keywords"`
		TimeCreated struct {
			SystemTime string `xml:"SystemTime,attr"`
		} `xml:"TimeCreated"`
		EventRecordID string `xml:"EventRecordID"`
		Channel       string `xml:"Channel"`
		Computer      string `xml:"Computer"`
		Security      struct {
			UserID string `xml:"UserID,attr"`
		} `xml:"Security"`
	} `xml:"System"`
	EventData struct {
		Data []struct {
			Name  string `xml:"Name,attr"`
			Value string `xml:",chardata"`
		} `xml:"Data"`
	} `xml:"EventData"`
}

// wellKnownEventIDs gives the classic security log IDs a human label so the
// synthesised message reads like the rendered Windows text the session
// classifier's substring heuristics expect.
var wellKnownEventIDs = map[string]string{
	"1102": "The audit log was cleared",
	"4624": "Successful Logon",
	"4625": "Failed Logon",
	"4634": "Logoff",
	"4648": "Logon with explicit credentials",
	"4672": "Special privileges assigned to new logon",
	"4688": "A new process has been created",
	"4720": "A user account was created",
	"4726": "A user account was deleted",
	"7045": "A service was installed",
}

func parseWinEvt(raw string) (*events.Event, error) {
	// Tolerate an XML declaration prefix and the <Events> batch wrapper
	// around a single record.
	body := strings.TrimSpace(raw)
	if i := strings.Index(body, "<Event"); i > 0 {
		body = body[i:]
	}
	var we winEvent
	if err := xml.Unmarshal([]byte(body), &we); err != nil {
		return nil, errors.New("winevt: " + err.Error())
	}
	if we.System.EventID == "" {
		return nil, errors.New("winevt: no EventID")
	}

	ev := &events.Event{
		Source:    events.SourceFile,
		Raw:       raw,
		EventType: "winevt:" + we.System.EventID,
		Severity:  winLevelSeverity(we.System.Level),
		Fields: map[string]string{
			"EventID": we.System.EventID,
		},
	}
	if ts, err := time.Parse(time.RFC3339Nano, we.System.TimeCreated.SystemTime); err == nil {
		ev.Timestamp = ts
	}
	if we.System.Computer != "" {
		ev.HostID = we.System.Computer
	}
	setIf := func(k, v string) {
		if v != "" {
			ev.Fields[k] = v
		}
	}
	setIf("Provider", we.System.Provider.Name)
	setIf("Channel", we.System.Channel)
	setIf("Level", we.System.Level)
	setIf("Task", we.System.Task)
	setIf("EventRecordID", we.System.EventRecordID)
	setIf("UserID", we.System.Security.UserID)
	for _, d := range we.EventData.Data {
		if d.Name != "" {
			ev.Fields[d.Name] = strings.TrimSpace(d.Value)
		}
	}

	// Well-known process-creation shape: promote EventID 4688's process
	// fields to the names the cmdline reconstructor scans for.
	label := wellKnownEventIDs[we.System.EventID]
	msg := "EventID " + we.System.EventID
	if label != "" {
		msg += " " + label
	}
	if p := we.System.Provider.Name; p != "" && label == "" {
		msg += " " + p
	}
	if u := ev.Fields["TargetUserName"]; u != "" {
		msg += " user=" + u
	}
	if ip := ev.Fields["IpAddress"]; ip != "" && ip != "-" {
		msg += " from " + ip
	}
	if cl := ev.Fields["CommandLine"]; cl != "" {
		msg += " CommandLine=" + cl
	}
	ev.Message = msg

	// Security auditing failures (4625, 1102, 4726) read as warnings so
	// they surface in default triage views.
	switch we.System.EventID {
	case "4625", "1102", "4726":
		if sevRank(ev.Severity) < sevRank(events.SeverityWarn) {
			ev.Severity = events.SeverityWarn
		}
	}
	return ev, nil
}

// winLevelSeverity maps the numeric <Level> to OBLIVRA severity.
// 0 = LogAlways, 1 = Critical, 2 = Error, 3 = Warning, 4 = Info, 5 = Verbose.
func winLevelSeverity(level string) events.Severity {
	n, err := strconv.Atoi(level)
	if err != nil {
		return events.SeverityInfo
	}
	switch n {
	case 1:
		return events.SeverityCritical
	case 2:
		return events.SeverityError
	case 3:
		return events.SeverityWarn
	case 5:
		return events.SeverityDebug
	default: // 0 (LogAlways) and 4 (Informational)
		return events.SeverityInfo
	}
}

// sevRank orders severities for max() comparisons.
func sevRank(s events.Severity) int {
	switch s {
	case events.SeverityDebug:
		return 0
	case events.SeverityInfo:
		return 1
	case events.SeverityNotice:
		return 2
	case events.SeverityWarn:
		return 3
	case events.SeverityError:
		return 4
	case events.SeverityCritical:
		return 5
	case events.SeverityAlert:
		return 6
	default:
		return 1
	}
}
