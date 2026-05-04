//go:build windows

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// runWinlog tails a Windows Event Log channel using wevtutil.exe as the
// subprocess — no CGO, no DLL loading, same static-binary guarantee as
// the journald reader on Linux.
//
// Strategy:
//   - On first run with startFrom:"beginning" → backfill every record in the
//     channel (including rotated archive logs) then switch to live poll.
//   - On first run with startFrom:"tail" (default) → record the current high-
//     water EventRecordID and only forward events written *after* this run.
//   - On restart → resume from the cursor file (last EventRecordID flushed).
//
// Live tail is implemented as a poll loop (wevtutil has no --follow flag).
// Default poll interval 2 s; kept intentionally short so analysts see events
// within one heartbeat tick.
//
// Channels to configure in agent.yml:
//
//	- type: winlog
//	  channel: "Security"
//	- type: winlog
//	  channel: "System"
//	- type: winlog
//	  channel: "Application"
//	- type: winlog
//	  channel: "Microsoft-Windows-Sysmon/Operational"
//	- type: winlog
//	  channel: "Microsoft-Windows-PowerShell/Operational"
func (t *Tailer) runWinlog(ctx context.Context) error {
	channel := t.in.Channel
	if channel == "" {
		channel = "Security"
	}

	// Verify wevtutil is reachable (it is on all modern Windows, but fail
	// fast and clearly if someone is running under Wine or a container).
	if _, err := exec.LookPath("wevtutil.exe"); err != nil {
		return fmt.Errorf("winlog: wevtutil.exe not found in PATH — Windows Event Log unavailable")
	}

	cursorFile := filepath.Join(t.stateDir, "winlog-"+sanitizeChannelName(channel)+".cursor")
	lastID := readWinlogCursor(cursorFile)

	switch {
	case lastID > 0:
		// Resume: ship everything after the last-seen record ID.
		log.Printf("winlog %s: resuming from EventRecordID %d", channel, lastID)
		newID := t.backfillWinlog(ctx, channel, lastID, cursorFile)
		if newID > lastID {
			lastID = newID
		}

	case t.in.StartFrom == "beginning":
		// Day-zero backfill: ship every record in the channel, oldest first.
		// wevtutil /rd:false = oldest first (ascending).
		log.Printf("winlog %s: backfilling from the beginning", channel)
		newID := t.backfillWinlog(ctx, channel, 0, cursorFile)
		lastID = newID

	default:
		// Default (tail): seed the cursor to the current high-water mark so
		// we only forward events from this moment onward.
		hwm := winlogHighWater(channel)
		log.Printf("winlog %s: tailing from EventRecordID %d", channel, hwm)
		if hwm > 0 {
			writeWinlogCursor(cursorFile, hwm)
			lastID = hwm
		}
	}

	// Live poll loop.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			newID := t.pollWinlog(ctx, channel, lastID, cursorFile)
			if newID > lastID {
				lastID = newID
			}
		}
	}
}

// backfillWinlog ships all events in `channel` with EventRecordID > afterID.
// afterID == 0 means "from the very first record".
// Returns the highest EventRecordID shipped (0 if nothing was shipped).
func (t *Tailer) backfillWinlog(ctx context.Context, channel string, afterID int64, cursorFile string) int64 {
	query := winlogXPathQuery(afterID)
	args := []string{
		"qe", channel,
		"/f:RenderedXml",
		"/rd:false", // oldest-first
		"/q:" + query,
	}
	cmd := exec.CommandContext(ctx, "wevtutil.exe", args...)
	out, err := cmd.Output()
	if err != nil {
		// Exit code 2 = "no events matching query" — completely normal.
		if isNoEventsError(err) {
			return afterID
		}
		log.Printf("winlog backfill %s: wevtutil: %v", channel, err)
		return afterID
	}
	return t.consumeWinlogXML(ctx, out, channel, cursorFile)
}

// pollWinlog queries for events written since lastID and ships them.
// Returns the new high-water EventRecordID (or lastID if nothing arrived).
func (t *Tailer) pollWinlog(ctx context.Context, channel string, lastID int64, cursorFile string) int64 {
	if ctx.Err() != nil {
		return lastID
	}
	query := winlogXPathQuery(lastID)
	args := []string{
		"qe", channel,
		"/f:RenderedXml",
		"/rd:false",
		"/c:500", // cap per poll; prevents a burst from blocking the main loop
		"/q:" + query,
	}
	cmd := exec.CommandContext(ctx, "wevtutil.exe", args...)
	out, err := cmd.Output()
	if err != nil {
		if isNoEventsError(err) {
			return lastID
		}
		log.Printf("winlog poll %s: %v", channel, err)
		return lastID
	}
	newID := t.consumeWinlogXML(ctx, out, channel, cursorFile)
	if newID > lastID {
		return newID
	}
	return lastID
}

// consumeWinlogXML parses the raw wevtutil XML output (multiple concatenated
// <Event> documents), converts each to an agent event, and enqueues it.
// Returns the highest EventRecordID seen.
func (t *Tailer) consumeWinlogXML(ctx context.Context, raw []byte, channel string, cursorFile string) int64 {
	var highID int64

	// wevtutil outputs bare <Event> elements concatenated without a root
	// wrapper.  Wrap them so the XML decoder can stream them as tokens.
	wrapped := bytes.NewReader(bytes.Join([][]byte{
		[]byte("<Events>"),
		raw,
		[]byte("</Events>"),
	}, nil))

	dec := xml.NewDecoder(wrapped)
	dec.Strict = false // tolerate malformed records

	for {
		if ctx.Err() != nil {
			break
		}
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Event" {
			continue
		}
		var ev winEvtXML
		if err := dec.DecodeElement(&ev, &se); err != nil {
			continue
		}

		msg := winlogFormatMessage(&ev, channel)
		t.enqueue("winlog:"+channel, msg)

		if ev.System.EventRecordID > highID {
			highID = ev.System.EventRecordID
		}
	}

	if highID > 0 {
		writeWinlogCursor(cursorFile, highID)
	}
	return highID
}

// winlogFormatMessage converts a parsed Windows event to a human-readable
// syslog-style line that matches what the platform parsers expect.
// Format: <timestamp> <Computer> <Channel>[<EventID>]: <message>
// Fields from EventData are injected as key=value pairs after the message.
func winlogFormatMessage(ev *winEvtXML, channel string) string {
	ts := parseWinlogTime(ev.System.TimeCreated.SystemTime)

	// Build the EventData payload as key=value pairs.
	var dataItems []string
	for _, d := range ev.Data.Data {
		if d.Name != "" && strings.TrimSpace(d.Value) != "" {
			dataItems = append(dataItems, d.Name+"="+strings.TrimSpace(d.Value))
		}
	}
	for _, d := range ev.UserData.Data {
		if d.Name != "" && strings.TrimSpace(d.Value) != "" {
			dataItems = append(dataItems, d.Name+"="+strings.TrimSpace(d.Value))
		}
	}

	description := winlogEventDescription(ev.System.EventID)
	var body strings.Builder
	if description != "" {
		body.WriteString(description)
	}
	if len(dataItems) > 0 {
		if body.Len() > 0 {
			body.WriteString(": ")
		}
		body.WriteString(strings.Join(dataItems, " "))
	}
	if body.Len() == 0 {
		body.WriteString(fmt.Sprintf("EventID=%d", ev.System.EventID))
	}

	sev := winlogSeverity(ev.System.Level, ev.System.EventID)

	fields := map[string]string{
		"sourceType":   winlogSourceType(channel),
		"agentSource":  "winlog:" + channel,
		"winChannel":   ev.System.Channel,
		"winEventID":   strconv.Itoa(ev.System.EventID),
		"winProvider":  ev.System.Provider.Name,
		"winRecordID":  strconv.FormatInt(ev.System.EventRecordID, 10),
		"winComputer":  ev.System.Computer,
		"winKeywords":  ev.System.Keywords,
		"severity":     sev,
	}

	doc := map[string]any{
		"source":    "agent",
		"eventType": fmt.Sprintf("win:event:%d", ev.System.EventID),
		"severity":  sev,
		"hostId":    ev.System.Computer,
		"timestamp": ts.UTC().Format(time.RFC3339Nano),
		"message": fmt.Sprintf("%s %s %s[%d]: %s",
			ts.Format("Jan _2 15:04:05"),
			ev.System.Computer,
			shortChannelName(channel),
			ev.System.EventID,
			body.String(),
		),
		"raw":    body.String(),
		"fields": fields,
	}
	b, _ := json.Marshal(doc)
	return string(b)
}

// ── XML structs ────────────────────────────────────────────────────────────

type winEvtXML struct {
	XMLName xml.Name     `xml:"Event"`
	System  winEvtSystem `xml:"System"`
	Data    winEvtData   `xml:"EventData"`
	UserData winEvtData  `xml:"UserData"`
}

type winEvtSystem struct {
	Provider     struct{ Name string `xml:"Name,attr"` } `xml:"Provider"`
	EventID      int    `xml:"EventID"`
	Version      int    `xml:"Version"`
	Level        int    `xml:"Level"`
	Task         int    `xml:"Task"`
	Keywords     string `xml:"Keywords"`
	TimeCreated  struct{ SystemTime string `xml:"SystemTime,attr"` } `xml:"TimeCreated"`
	EventRecordID int64 `xml:"EventRecordID"`
	Channel      string `xml:"Channel"`
	Computer     string `xml:"Computer"`
}

type winEvtData struct {
	Data []struct {
		Name  string `xml:"Name,attr"`
		Value string `xml:",chardata"`
	} `xml:"Data"`
}

// ── Helpers ────────────────────────────────────────────────────────────────

// winlogXPathQuery returns an XPath filter selecting events with
// EventRecordID > afterID.  afterID == 0 returns all events.
func winlogXPathQuery(afterID int64) string {
	if afterID <= 0 {
		return "*"
	}
	return fmt.Sprintf("*[System[EventRecordID > %d]]", afterID)
}

// winlogHighWater returns the EventRecordID of the most-recent event in
// channel (used to seed the "tail" cursor on first run).
func winlogHighWater(channel string) int64 {
	// Query the single newest event.
	cmd := exec.Command("wevtutil.exe", "qe", channel, "/f:RenderedXml", "/rd:true", "/c:1")
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	wrapped := bytes.NewReader(append(append([]byte("<Events>"), out...), []byte("</Events>")...))
	dec := xml.NewDecoder(wrapped)
	dec.Strict = false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Event" {
			continue
		}
		var ev winEvtXML
		if dec.DecodeElement(&ev, &se) == nil {
			return ev.System.EventRecordID
		}
	}
	return 0
}

func parseWinlogTime(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	// Windows uses 2006-01-02T15:04:05.0000000Z
	for _, layout := range []string{
		"2006-01-02T15:04:05.9999999Z",
		"2006-01-02T15:04:05.999999999Z07:00",
		time.RFC3339Nano,
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Now()
}

func sanitizeChannelName(ch string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", " ", "_", ":", "-")
	return strings.ToLower(r.Replace(ch))
}

func shortChannelName(ch string) string {
	parts := strings.Split(ch, "/")
	return parts[len(parts)-1]
}

func winlogSourceType(channel string) string {
	m := map[string]string{
		"Security":    "windows:security",
		"System":      "windows:system",
		"Application": "windows:application",
		"Microsoft-Windows-Sysmon/Operational":                          "windows:sysmon",
		"Microsoft-Windows-PowerShell/Operational":                       "windows:powershell",
		"Microsoft-Windows-Windows Defender/Operational":                 "windows:defender",
		"Microsoft-Windows-TaskScheduler/Operational":                    "windows:scheduler",
		"Microsoft-Windows-TerminalServices-RemoteConnectionManager/Operational": "windows:rdp",
		"Microsoft-Windows-WMI-Activity/Operational":                     "windows:wmi",
	}
	if st, ok := m[channel]; ok {
		return st
	}
	return "windows:" + sanitizeChannelName(channel)
}

// winlogSeverity maps Windows Event Level (0-5) and specific EventIDs
// to the platform severity vocabulary.
func winlogSeverity(level, eventID int) string {
	// EventID-specific overrides take priority.
	criticalEventIDs := map[int]bool{
		1102: true, // Security audit log cleared
		517:  true, // (XP) audit log cleared
		4616: true, // System time changed
	}
	highEventIDs := map[int]bool{
		4625: true, // Failed logon
		4648: true, // Logon using explicit credentials
		4697: true, // Service installed
		4698: true, // Scheduled task created
		4719: true, // Audit policy changed
		4720: true, // User account created
		4722: true, // User account enabled
		4723: true, // Password change attempt
		4724: true, // Password reset
		4725: true, // User account disabled
		4726: true, // User account deleted
		4728: true, // Member added to global security group
		4732: true, // Member added to local group
		4756: true, // Member added to universal group
		7045: true, // New service installed (System log)
		4688: true, // Process creation
		4104: true, // PowerShell script block logging
		4103: true, // PowerShell module logging
	}
	if criticalEventIDs[eventID] {
		return "critical"
	}
	if highEventIDs[eventID] {
		return "warning"
	}
	switch level {
	case 1:
		return "critical"
	case 2:
		return "error"
	case 3:
		return "warning"
	case 5:
		return "debug"
	default:
		return "info"
	}
}

// winlogEventDescription maps common EventIDs to readable descriptions.
// This avoids having to call EvtFormatMessage via Win32 API — we keep it
// simple with a curated map covering the most forensically relevant IDs.
func winlogEventDescription(id int) string {
	descriptions := map[int]string{
		// ── Security: Logon/Logoff ──────────────────────────────────────
		4624: "An account was successfully logged on",
		4625: "An account failed to log on",
		4634: "An account was logged off",
		4647: "User initiated logoff",
		4648: "A logon was attempted using explicit credentials",
		4649: "A replay attack was detected",
		4672: "Special privileges assigned to new logon",
		4675: "SIDs were filtered",
		// ── Security: Account Management ────────────────────────────────
		4720: "A user account was created",
		4722: "A user account was enabled",
		4723: "An attempt was made to change an account password",
		4724: "An attempt was made to reset an account password",
		4725: "A user account was disabled",
		4726: "A user account was deleted",
		4728: "A member was added to a security-enabled global group",
		4729: "A member was removed from a security-enabled global group",
		4732: "A member was added to a security-enabled local group",
		4733: "A member was removed from a security-enabled local group",
		4756: "A member was added to a security-enabled universal group",
		// ── Security: Privilege Use ──────────────────────────────────────
		4673: "A privileged service was called",
		4674: "An operation was attempted on a privileged object",
		// ── Security: Process Tracking ───────────────────────────────────
		4688: "A new process has been created",
		4689: "A process has exited",
		// ── Security: Audit Policy ───────────────────────────────────────
		4715: "The audit policy (SACL) on an object was changed",
		4719: "System audit policy was changed",
		4817: "Auditing settings on object were changed",
		1102: "The audit log was cleared",
		// ── Security: Kerberos ──────────────────────────────────────────
		4768: "A Kerberos authentication ticket (TGT) was requested",
		4769: "A Kerberos service ticket was requested",
		4771: "Kerberos pre-authentication failed",
		4776: "The computer attempted to validate credentials",
		// ── Security: Object Access ──────────────────────────────────────
		4656: "A handle to an object was requested",
		4663: "An attempt was made to access an object",
		4670: "Permissions on an object were changed",
		// ── Security: Policy Change ──────────────────────────────────────
		4616: "The system time was changed",
		4697: "A service was installed in the system",
		4698: "A scheduled task was created",
		4699: "A scheduled task was deleted",
		4700: "A scheduled task was enabled",
		4701: "A scheduled task was disabled",
		// ── System: Service Control Manager ─────────────────────────────
		7034: "A service terminated unexpectedly",
		7036: "A service entered a state",
		7040: "The start type of a service was changed",
		7045: "A new service was installed",
		// ── System: Kernel ───────────────────────────────────────────────
		6005: "The Event Log service was started",
		6006: "The Event Log service was stopped",
		6013: "System uptime",
		41:   "The system has rebooted without cleanly shutting down",
		1074: "The process initiated a restart/shutdown",
		// ── Sysmon ───────────────────────────────────────────────────────
		1:  "Sysmon: Process Create",
		2:  "Sysmon: File creation time changed",
		3:  "Sysmon: Network connection",
		5:  "Sysmon: Process terminated",
		6:  "Sysmon: Driver loaded",
		7:  "Sysmon: Image loaded",
		8:  "Sysmon: CreateRemoteThread",
		10: "Sysmon: Process accessed",
		11: "Sysmon: File created",
		12: "Sysmon: Registry object added/deleted",
		13: "Sysmon: Registry value set",
		15: "Sysmon: File stream created",
		17: "Sysmon: Pipe created",
		18: "Sysmon: Pipe connected",
		22: "Sysmon: DNS query",
		23: "Sysmon: File delete",
		25: "Sysmon: Process tampering",
		// ── PowerShell ───────────────────────────────────────────────────
		4103: "PowerShell: Module logging",
		4104: "PowerShell: Script block logging",
		// ── WMI ──────────────────────────────────────────────────────────
		5857: "WMI: Provider started",
		5858: "WMI: Provider error",
		5859: "WMI: Filter-consumer binding",
		5860: "WMI: Temporary subscription created",
		5861: "WMI: Permanent subscription created",
	}
	if desc, ok := descriptions[id]; ok {
		return desc
	}
	return ""
}

// isNoEventsError returns true when wevtutil exits because the query
// matched zero records — that's normal and should not be logged as an error.
func isNoEventsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no events") ||
		strings.Contains(msg, "error code 15007") || // ERROR_EVT_QUERY_RESULT_STALE
		strings.Contains(msg, "error code 15002")    // ERROR_EVT_INVALID_QUERY
}

// ── Cursor persistence ─────────────────────────────────────────────────────

func readWinlogCursor(path string) int64 {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	id, _ := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	return id
}

func writeWinlogCursor(path string, id int64) {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strconv.FormatInt(id, 10)+"\n"), 0o600); err != nil {
		log.Printf("winlog cursor write: %v", err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		log.Printf("winlog cursor rename: %v", err)
		_ = os.Remove(tmp)
	}
}

// ── Channel discovery ──────────────────────────────────────────────────────

// discoverWinlogChannels returns the subset of well-known channels that
// actually exist on this host. Called by setup to build the inputs list.
func discoverWinlogChannels() []string {
	// Priority-ordered list — Security is always first for forensics.
	wellKnown := []string{
		"Security",
		"System",
		"Application",
		"Microsoft-Windows-Sysmon/Operational",
		"Microsoft-Windows-PowerShell/Operational",
		"Microsoft-Windows-TaskScheduler/Operational",
		"Microsoft-Windows-Windows Defender/Operational",
		"Microsoft-Windows-TerminalServices-RemoteConnectionManager/Operational",
		"Microsoft-Windows-WMI-Activity/Operational",
		"Microsoft-Windows-DNS-Client/Operational",
	}

	// `wevtutil gl <channel>` exits 0 if the channel exists, non-zero otherwise.
	var found []string
	for _, ch := range wellKnown {
		cmd := exec.Command("wevtutil.exe", "gl", ch)
		if err := cmd.Run(); err == nil {
			found = append(found, ch)
		}
	}
	return found
}

// scanWevtutil is a helper used in tests to read all XML from a wevtutil
// output stream without shipping (used in runTest mode).
func scanWevtutil(data []byte) []string {
	wrapped := bytes.NewReader(append(append([]byte("<Events>"), data...), []byte("</Events>")...))
	dec := xml.NewDecoder(wrapped)
	dec.Strict = false
	var out []string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Event" {
			continue
		}
		var ev winEvtXML
		if dec.DecodeElement(&ev, &se) == nil {
			out = append(out, winlogFormatMessage(&ev, ev.System.Channel))
		}
	}
	return out
}

// drainWevtutilStderr logs stderr lines from a wevtutil command.
func drainWevtutilStderr(r *bufio.Reader) {
	for {
		line, err := r.ReadString('\n')
		if line != "" {
			log.Printf("wevtutil: %s", strings.TrimRight(line, "\r\n"))
		}
		if err != nil {
			return
		}
	}
}
