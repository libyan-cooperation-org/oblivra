package parsers

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// Phase 42 — auditd text-format reader.
//
// The Linux audit subsystem writes /var/log/audit/audit.log in a
// space-separated key=value format introduced by Steve Grubb. Format:
//
//   type=SYSCALL msg=audit(1714521600.123:456): arch=c000003e syscall=2
//     success=yes exit=3 pid=1234 auid=1000 uid=0 gid=0 comm="cat"
//     exe="/usr/bin/cat" key=(null)
//
// Two pieces of metadata always show up: the `type` field (event class —
// SYSCALL, PATH, EXECVE, USER_AUTH, CRED_DISP, etc.) and the
// `msg=audit(<unix_seconds>.<ms>:<seqid>):` group. The remainder is
// plain key=value with optional double-quoted values.
//
// We parse the text variant directly because the binary auditd format
// requires libauparse (C). The text variant is what most operators ship
// to log shippers anyway.
//
// Each parsed line becomes an event with:
//   - EventType   = type (lowercased)
//   - Timestamp   = parsed from msg=audit(...)
//   - Severity    = info, except CRED_*/USER_AUTH failures → warning
//   - Fields      = every other key=value pair
//   - HostID      = empty (the source line itself doesn't carry it;
//                   callers stamp it via Provenance)

var auditdMsgPrefix = []byte("msg=audit(")

func parseAuditd(raw string) (*events.Event, error) {
	if !strings.HasPrefix(raw, "type=") {
		return nil, errors.New("auditd: missing type= prefix")
	}
	ev := &events.Event{
		Raw:      raw,
		Source:   events.SourceREST,
		Severity: events.SeverityInfo,
		Fields:   map[string]string{},
	}
	tokens := tokenizeAuditd(raw)
	for _, t := range tokens {
		eq := strings.IndexByte(t, '=')
		if eq <= 0 {
			continue
		}
		k := t[:eq]
		v := strings.Trim(t[eq+1:], `"`)
		switch k {
		case "type":
			ev.EventType = "auditd:" + strings.ToLower(v)
		case "msg":
			ts, seq, ok := parseAuditMsg(v)
			if ok {
				ev.Timestamp = ts
				ev.Fields["auditSeq"] = seq
			}
		case "comm":
			ev.Fields["comm"] = v
		case "exe":
			ev.Fields["exe"] = v
		case "uid", "auid", "euid", "suid", "fsuid":
			ev.Fields[k] = v
		case "pid", "ppid":
			ev.Fields[k] = v
		case "key":
			if v != "(null)" {
				ev.Fields["auditKey"] = v
			}
		case "success":
			ev.Fields["success"] = v
			if v == "no" && (ev.EventType == "auditd:user_auth" ||
				ev.EventType == "auditd:cred_disp" ||
				ev.EventType == "auditd:user_login") {
				ev.Severity = events.SeverityWarn
			}
		default:
			ev.Fields[k] = v
		}
	}
	if ev.EventType == "" {
		return nil, errors.New("auditd: no type=")
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	ev.Message = compactAuditMessage(ev)
	return ev, nil
}

// tokenizeAuditd splits a line into KV tokens, respecting double-quoted
// values that may contain spaces (e.g. comm="my prog"). msg=audit(...)
// is a special case — its value contains parentheses; we slurp until
// the closing `):` pair.
func tokenizeAuditd(s string) []string {
	var out []string
	i := 0
	for i < len(s) {
		// Skip leading whitespace.
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		// Handle the "msg=audit(...)" group as one token.
		if i+len(auditdMsgPrefix) <= len(s) && s[i:i+len(auditdMsgPrefix)] == string(auditdMsgPrefix) {
			end := strings.Index(s[i:], "):")
			if end >= 0 {
				out = append(out, s[i:i+end+1])
				i += end + 2
				continue
			}
		}
		start := i
		quoted := false
		for i < len(s) {
			c := s[i]
			if c == '"' {
				quoted = !quoted
				i++
				continue
			}
			if !quoted && (c == ' ' || c == '\t') {
				break
			}
			i++
		}
		out = append(out, s[start:i])
	}
	return out
}

// parseAuditMsg unpacks the "audit(<unix>.<ms>:<seq>)" stamp.
func parseAuditMsg(v string) (time.Time, string, bool) {
	v = strings.TrimPrefix(v, "audit(")
	v = strings.TrimSuffix(v, ")")
	colon := strings.IndexByte(v, ':')
	if colon < 0 {
		return time.Time{}, "", false
	}
	tsStr := v[:colon]
	seq := v[colon+1:]
	dot := strings.IndexByte(tsStr, '.')
	var sec, msec int64
	if dot < 0 {
		s, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			return time.Time{}, "", false
		}
		sec = s
	} else {
		s, err := strconv.ParseInt(tsStr[:dot], 10, 64)
		if err != nil {
			return time.Time{}, "", false
		}
		sec = s
		ms, err := strconv.ParseInt(tsStr[dot+1:], 10, 64)
		if err == nil {
			msec = ms
		}
	}
	return time.Unix(sec, msec*int64(time.Millisecond)).UTC(), seq, true
}

// compactAuditMessage builds a short, human-readable summary so the live
// tail UI doesn't dump the full key=value soup as the headline.
func compactAuditMessage(ev *events.Event) string {
	parts := []string{strings.TrimPrefix(ev.EventType, "auditd:")}
	if c, ok := ev.Fields["comm"]; ok && c != "" {
		parts = append(parts, "comm="+c)
	}
	if exe, ok := ev.Fields["exe"]; ok && exe != "" {
		parts = append(parts, "exe="+exe)
	}
	if uid, ok := ev.Fields["uid"]; ok {
		parts = append(parts, "uid="+uid)
	}
	if s, ok := ev.Fields["success"]; ok {
		parts = append(parts, "success="+s)
	}
	return strings.Join(parts, " ")
}
