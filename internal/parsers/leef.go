package parsers

import (
	"errors"
	"strings"

	"github.com/kingknull/oblivra/internal/events"
)

// Phase 98 — LEEF (Log Event Extended Format) parser.
//
// IBM QRadar's wire format. Two versions in the wild:
//
//	LEEF:1.0|Vendor|Product|Version|EventID|key=value<tab>key=value...
//	LEEF:2.0|Vendor|Product|Version|EventID|DelimChar|key=value<d>key=value...
//
// LEEF 2.0's sixth header field names the attribute delimiter — either a
// literal character or `x`-prefixed hex (`x09` = tab). LEEF 1.0 uses tab.
// Common attribute names (src, dst, usrName, srcPort, dstPort, proto,
// devTime, sev) map onto OBLIVRA's field conventions; everything else
// lands in Fields verbatim. Needed for QRadar-fed pipelines re-targeting
// OBLIVRA without a translation layer.

func parseLEEF(raw string) (*events.Event, error) {
	if !strings.HasPrefix(raw, "LEEF:") {
		return nil, errors.New("leef: missing LEEF: prefix")
	}
	version := strings.TrimPrefix(raw[:strings.IndexByte(raw, '|')], "LEEF:")

	// Header field count differs: 1.0 has 5 (payload is 6th), 2.0 has 6
	// (payload is 7th, preceded by the delimiter spec).
	nHeader := 6
	if strings.HasPrefix(version, "2") {
		nHeader = 7
	}
	parts := splitN(raw, '|', nHeader)
	if len(parts) < nHeader {
		return nil, errors.New("leef: truncated header")
	}

	delim := "\t"
	payload := parts[nHeader-1]
	if nHeader == 7 {
		delim = decodeLEEFDelim(parts[5])
	}

	ev := &events.Event{
		Source:    events.SourceFile,
		Raw:       raw,
		EventType: "leef:" + parts[4],
		Severity:  events.SeverityInfo,
		Fields: map[string]string{
			"leefVersion": version,
			"vendor":      parts[1],
			"product":     parts[2],
			"productVer":  parts[3],
			"leefEventId": parts[4],
		},
	}

	for _, kv := range strings.Split(payload, delim) {
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			continue
		}
		k := strings.TrimSpace(kv[:eq])
		v := strings.TrimSpace(kv[eq+1:])
		if k == "" || v == "" {
			continue
		}
		switch k {
		case "src":
			ev.Fields["srcIP"] = v
		case "dst":
			ev.Fields["dstIP"] = v
		case "usrName":
			ev.Fields["user"] = v
		case "sev":
			ev.Severity = leefSeverity(v)
			ev.Fields["sev"] = v
		default:
			ev.Fields[k] = v
		}
	}

	msg := parts[2] + " event " + parts[4]
	if u := ev.Fields["user"]; u != "" {
		msg += " user=" + u
	}
	if s := ev.Fields["srcIP"]; s != "" {
		msg += " src=" + s
	}
	if d := ev.Fields["dstIP"]; d != "" {
		msg += " dst=" + d
	}
	ev.Message = msg
	return ev, nil
}

// decodeLEEFDelim resolves the LEEF 2.0 delimiter spec: a literal char,
// or hex as `x09` / `0x09`. Empty spec falls back to tab.
func decodeLEEFDelim(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "\t"
	}
	hexPart := ""
	if strings.HasPrefix(spec, "0x") || strings.HasPrefix(spec, "0X") {
		hexPart = spec[2:]
	} else if (spec[0] == 'x' || spec[0] == 'X') && len(spec) > 1 {
		hexPart = spec[1:]
	}
	if hexPart != "" {
		var n int
		for _, c := range hexPart {
			switch {
			case c >= '0' && c <= '9':
				n = n*16 + int(c-'0')
			case c >= 'a' && c <= 'f':
				n = n*16 + int(c-'a'+10)
			case c >= 'A' && c <= 'F':
				n = n*16 + int(c-'A'+10)
			default:
				return "\t"
			}
		}
		if n > 0 && n < 128 {
			return string(rune(n))
		}
		return "\t"
	}
	return spec[:1]
}

// leefSeverity maps QRadar's 1-10 scale (same shape as CEF's).
func leefSeverity(s string) events.Severity {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return events.SeverityInfo
		}
		n = n*10 + int(c-'0')
	}
	return cefSeverity(n)
}
