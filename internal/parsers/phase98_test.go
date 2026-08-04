package parsers

import (
	"strings"
	"testing"

	"github.com/kingknull/oblivra/internal/events"
)

// ---- Windows Event XML ----

const winevt4624 = `<Event xmlns='http://schemas.microsoft.com/win/2004/08/events/event'><System><Provider Name='Microsoft-Windows-Security-Auditing'/><EventID>4624</EventID><Level>0</Level><TimeCreated SystemTime='2026-07-30T08:12:44.123456700Z'/><EventRecordID>982341</EventRecordID><Channel>Security</Channel><Computer>DC01.corp.local</Computer></System><EventData><Data Name='TargetUserName'>alice</Data><Data Name='LogonType'>10</Data><Data Name='IpAddress'>10.0.0.5</Data></EventData></Event>`

func TestWinEvtLogonSuccess(t *testing.T) {
	ev, err := Parse(winevt4624, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != "winevt:4624" {
		t.Errorf("eventType = %q", ev.EventType)
	}
	if ev.HostID != "DC01.corp.local" {
		t.Errorf("hostId = %q", ev.HostID)
	}
	if ev.Fields["TargetUserName"] != "alice" || ev.Fields["IpAddress"] != "10.0.0.5" {
		t.Errorf("fields = %+v", ev.Fields)
	}
	if ev.Fields["EventID"] != "4624" || ev.Fields["Channel"] != "Security" {
		t.Errorf("system fields = %+v", ev.Fields)
	}
	if ev.Timestamp.IsZero() || ev.Timestamp.Year() != 2026 {
		t.Errorf("timestamp = %v", ev.Timestamp)
	}
	// Message must satisfy the session classifier's substring heuristics.
	if !strings.Contains(ev.Message, "EventID 4624") || !strings.Contains(ev.Message, "Successful Logon") {
		t.Errorf("message = %q", ev.Message)
	}
	if ev.Raw != winevt4624 {
		t.Error("raw not preserved")
	}
}

func TestWinEvtFailedLogonSeverity(t *testing.T) {
	line := `<Event><System><EventID>4625</EventID><Level>0</Level><Computer>DC01</Computer></System><EventData><Data Name='TargetUserName'>administrator</Data><Data Name='IpAddress'>203.0.113.66</Data></EventData></Event>`
	ev, err := Parse(line, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Severity != events.SeverityWarn {
		t.Errorf("4625 severity = %q, want warning", ev.Severity)
	}
	if !strings.Contains(ev.Message, "Failed Logon") {
		t.Errorf("message = %q", ev.Message)
	}
}

func TestWinEvtLevelMapping(t *testing.T) {
	for level, want := range map[string]events.Severity{
		"1": events.SeverityCritical,
		"2": events.SeverityError,
		"3": events.SeverityWarn,
		"4": events.SeverityInfo,
		"5": events.SeverityDebug,
	} {
		line := `<Event><System><EventID>1000</EventID><Level>` + level + `</Level></System></Event>`
		ev, err := Parse(line, FormatWinEvt)
		if err != nil {
			t.Fatal(err)
		}
		if ev.Severity != want {
			t.Errorf("level %s → %q, want %q", level, ev.Severity, want)
		}
	}
}

func TestWinEvtXMLDeclPrefix(t *testing.T) {
	line := `<?xml version="1.0"?><Event><System><EventID>7045</EventID><Computer>WS17</Computer></System><EventData><Data Name='ServiceName'>EvilSvc</Data></EventData></Event>`
	ev, err := Parse(line, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != "winevt:7045" || ev.Fields["ServiceName"] != "EvilSvc" {
		t.Errorf("ev = %+v", ev)
	}
}

func TestWinEvtMalformedFallsToPlain(t *testing.T) {
	ev, err := Parse(`<Event><System><oops`, FormatWinEvt)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != "plain" {
		t.Errorf("eventType = %q, want plain fallback", ev.EventType)
	}
}

// ---- CloudTrail ----

const ctLogin = `{"eventVersion":"1.08","userIdentity":{"type":"IAMUser","arn":"arn:aws:iam::111122223333:user/alice","accountId":"111122223333","userName":"alice"},"eventTime":"2026-07-30T08:15:00Z","eventSource":"signin.amazonaws.com","eventName":"ConsoleLogin","awsRegion":"us-east-1","sourceIPAddress":"203.0.113.5","responseElements":{"ConsoleLogin":"Success"}}`

func TestCloudTrailConsoleLogin(t *testing.T) {
	ev, err := Parse(ctLogin, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != "cloudtrail:ConsoleLogin" {
		t.Errorf("eventType = %q", ev.EventType)
	}
	if ev.Fields["user"] != "alice" || ev.Fields["srcIP"] != "203.0.113.5" {
		t.Errorf("fields = %+v", ev.Fields)
	}
	if ev.Fields["awsRegion"] != "us-east-1" || ev.Fields["eventSource"] != "signin.amazonaws.com" {
		t.Errorf("fields = %+v", ev.Fields)
	}
	if ev.Severity != events.SeverityInfo {
		t.Errorf("severity = %q", ev.Severity)
	}
	if ev.Timestamp.IsZero() {
		t.Error("timestamp not parsed")
	}
}

func TestCloudTrailFailureSeverity(t *testing.T) {
	failed := strings.Replace(ctLogin, `"Success"`, `"Failure"`, 1)
	ev, err := Parse(failed, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Severity != events.SeverityWarn {
		t.Errorf("failure severity = %q, want warning", ev.Severity)
	}
	if !strings.Contains(ev.Message, "result=Failure") {
		t.Errorf("message = %q", ev.Message)
	}
}

func TestCloudTrailErrorCode(t *testing.T) {
	line := `{"eventVersion":"1.08","userIdentity":{"type":"IAMUser","arn":"arn:aws:iam::1:user/mallory","userName":"mallory"},"eventTime":"2026-07-30T09:45:00Z","eventSource":"iam.amazonaws.com","eventName":"CreateAccessKey","sourceIPAddress":"203.0.113.77","errorCode":"AccessDenied","requestParameters":{"userName":"admin"}}`
	ev, err := Parse(line, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Severity != events.SeverityWarn || ev.Fields["errorCode"] != "AccessDenied" {
		t.Errorf("ev = %+v", ev)
	}
	// scalar requestParameters flattened under req.
	if ev.Fields["req.userName"] != "admin" {
		t.Errorf("req params = %+v", ev.Fields)
	}
}

func TestCloudTrailARNFallbackUser(t *testing.T) {
	line := `{"eventVersion":"1.09","userIdentity":{"type":"Root","arn":"arn:aws:iam::1:root"},"eventTime":"2026-07-30T10:00:00Z","eventSource":"cloudtrail.amazonaws.com","eventName":"StopLogging","sourceIPAddress":"203.0.113.88"}`
	ev, err := Parse(line, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Fields["user"] != "arn:aws:iam::1:root" {
		t.Errorf("user = %q, want ARN fallback", ev.Fields["user"])
	}
}

func TestCloudTrailSniffVsPlainJSON(t *testing.T) {
	// Ordinary JSON without CloudTrail markers must keep going to parseJSON.
	ev, err := Parse(`{"message":"hello","host":"web-01"}`, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != "json" {
		t.Errorf("plain JSON routed to %q", ev.EventType)
	}
}

// ---- LEEF ----

func TestLEEF20TabDelim(t *testing.T) {
	line := "LEEF:2.0|IBM|QRadar|9.2|AuthFailure|x09|src=192.0.2.10\tdst=10.0.0.4\tusrName=svc-backup\tsev=8"
	ev, err := Parse(line, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != "leef:AuthFailure" {
		t.Errorf("eventType = %q", ev.EventType)
	}
	if ev.Fields["srcIP"] != "192.0.2.10" || ev.Fields["dstIP"] != "10.0.0.4" || ev.Fields["user"] != "svc-backup" {
		t.Errorf("fields = %+v", ev.Fields)
	}
	if ev.Severity != events.SeverityError { // sev=8 → error tier
		t.Errorf("severity = %q", ev.Severity)
	}
	if ev.Fields["vendor"] != "IBM" || ev.Fields["product"] != "QRadar" {
		t.Errorf("header fields = %+v", ev.Fields)
	}
}

func TestLEEF20CustomDelim(t *testing.T) {
	line := "LEEF:2.0|PaloAlto|PAN-OS|11.0|THREAT|^|src=203.0.113.9^usrName=jdoe^sev=9^act=blocked"
	ev, err := Parse(line, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Fields["srcIP"] != "203.0.113.9" || ev.Fields["user"] != "jdoe" || ev.Fields["act"] != "blocked" {
		t.Errorf("fields = %+v", ev.Fields)
	}
	if ev.Severity != events.SeverityCritical { // sev=9 → critical
		t.Errorf("severity = %q", ev.Severity)
	}
}

func TestLEEF10DefaultTab(t *testing.T) {
	line := "LEEF:1.0|Fortinet|FortiGate|7.4|traffic-deny|src=198.51.100.7\tdst=10.2.0.8\tact=deny"
	ev, err := Parse(line, FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != "leef:traffic-deny" || ev.Fields["srcIP"] != "198.51.100.7" || ev.Fields["act"] != "deny" {
		t.Errorf("ev fields = %+v", ev.Fields)
	}
}

func TestLEEFTruncatedFallsToPlain(t *testing.T) {
	ev, err := Parse("LEEF:2.0|OnlyVendor", FormatAuto)
	if err != nil {
		t.Fatal(err)
	}
	if ev.EventType != "plain" {
		t.Errorf("eventType = %q, want plain fallback", ev.EventType)
	}
}
