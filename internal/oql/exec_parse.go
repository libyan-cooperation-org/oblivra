package oql

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// execParse implements the `parse json|xml|kv` OQL command (Phase 27.2.1).
// It reads a free-text source field on each row, runs the chosen
// structured parser, and merges the resulting key/value pairs back
// onto the row — either at the top level or under an `as <prefix>`
// dot-namespace.
//
// Failure mode: a row whose source field doesn't parse cleanly is
// passed through unchanged. We do NOT drop rows on parse failure,
// because real-world data is messy and the operator can chain a
// `where _parsed=true` clause downstream if they want to filter.
// Instead we set `_parse_error` on the row so the operator can see
// which records didn't parse.
func (ex *Executor) execParse(c *ParseCommand, rows []Row, prof *StageProfiler) ([]Row, error) {
	if c == nil {
		return rows, nil
	}
	fieldName := c.Field.Canonical()
	if fieldName == "" {
		fieldName = "_raw"
	}
	prefix := c.Output

	for _, row := range rows {
		prof.TrackRowIn()

		raw, ok := row[fieldName]
		if !ok || raw == nil {
			prof.TrackRowOut()
			continue
		}
		text := fmt.Sprint(raw)
		if text == "" {
			prof.TrackRowOut()
			continue
		}

		var (
			pairs map[string]interface{}
			err   error
		)
		switch c.Format {
		case ParseJSON:
			pairs, err = parseJSONFlat(text)
		case ParseXML:
			pairs, err = parseXMLFlat(text)
		case ParseKV:
			pairs = parseKVPairs(text)
		}
		if err != nil {
			row["_parse_error"] = err.Error()
			prof.TrackRowOut()
			continue
		}
		mergeFlatPairs(row, pairs, prefix)
		prof.TrackRowOut()
	}
	return rows, nil
}

// parseJSONFlat decodes a JSON value into a flat dot-path map.
//
//	{"user":"alice","ctx":{"ip":"10.0.0.1"}} →
//	  {"user":"alice","ctx.ip":"10.0.0.1"}
//
// Arrays surface as `<key>.<index>` paths; non-object roots become
// the single key `_value` so e.g. `parse json` on a stringified
// number still produces a usable field.
func parseJSONFlat(s string) (map[string]interface{}, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, err
	}
	out := make(map[string]interface{}, 8)
	flattenAny("", v, out)
	return out, nil
}

func flattenAny(prefix string, v interface{}, out map[string]interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		if len(t) == 0 {
			out[stripDotEnd(prefix)] = ""
			return
		}
		for k, child := range t {
			next := k
			if prefix != "" {
				next = prefix + "." + k
			}
			flattenAny(next, child, out)
		}
	case []interface{}:
		for i, child := range t {
			next := strconv.Itoa(i)
			if prefix != "" {
				next = prefix + "." + strconv.Itoa(i)
			}
			flattenAny(next, child, out)
		}
	default:
		key := prefix
		if key == "" {
			key = "_value"
		}
		out[key] = t
	}
}

func stripDotEnd(s string) string {
	for strings.HasSuffix(s, ".") {
		s = s[:len(s)-1]
	}
	return s
}

// parseXMLFlat decodes XML into a flat map. Element text content
// lands at the element's path; attributes land at `<path>.@<attr>`.
//
//	<user id="42"><name>alice</name></user> →
//	  {"user.name":"alice","user.@id":"42"}
//
// Best-effort: malformed XML is reported via the returned error.
func parseXMLFlat(s string) (map[string]interface{}, error) {
	dec := xml.NewDecoder(strings.NewReader(s))
	out := make(map[string]interface{}, 8)
	stack := make([]string, 0, 4)
	var charBuf strings.Builder

	for {
		tok, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			// Common case: caller passed something that started with
			// a non-XML byte. Treat as parse error rather than panic.
			if len(stack) == 0 && len(out) == 0 {
				return nil, err
			}
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			path := strings.Join(stack, ".")
			for _, a := range t.Attr {
				out[path+".@"+a.Name.Local] = a.Value
			}
			charBuf.Reset()
		case xml.CharData:
			charBuf.Write([]byte(t))
		case xml.EndElement:
			path := strings.Join(stack, ".")
			text := strings.TrimSpace(charBuf.String())
			if text != "" {
				out[path] = text
			}
			charBuf.Reset()
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("xml: no parseable content")
	}
	return out, nil
}

// parseKVPairs extracts `key=value` pairs from a free-text string.
// Recognised separators: whitespace, comma, semicolon. Quoted values
// (single or double) are preserved verbatim including embedded
// separators. Keys without an `=` are skipped.
//
//	`user=alice src_ip="10.0.0.1, scanner" action=allow` →
//	  {"user":"alice","src_ip":"10.0.0.1, scanner","action":"allow"}
func parseKVPairs(s string) map[string]interface{} {
	out := make(map[string]interface{}, 8)
	var (
		key       strings.Builder
		val       strings.Builder
		inKey     = true
		inQuote   byte
		afterEq  = false
	)
	flush := func() {
		k := strings.TrimSpace(key.String())
		v := val.String()
		if k != "" && afterEq {
			out[k] = v
		}
		key.Reset()
		val.Reset()
		inKey = true
		afterEq = false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote != 0 {
			if c == inQuote {
				inQuote = 0
				continue
			}
			val.WriteByte(c)
			continue
		}
		if c == '"' || c == '\'' {
			if !inKey {
				inQuote = c
				continue
			}
		}
		switch c {
		case ' ', '\t', ',', ';':
			if inKey && key.Len() == 0 {
				continue // skip leading separators
			}
			flush()
		case '=':
			if inKey {
				inKey = false
				afterEq = true
			} else {
				val.WriteByte(c)
			}
		default:
			if inKey {
				key.WriteByte(c)
			} else {
				val.WriteByte(c)
			}
		}
	}
	if key.Len() > 0 {
		flush()
	}
	return out
}

// mergeFlatPairs merges a flat key/value map back into a row.
// When `prefix` is non-empty every key is rewritten to `<prefix>.<key>`
// so the operator can disambiguate parsed fields from event-native
// fields.
func mergeFlatPairs(row Row, pairs map[string]interface{}, prefix string) {
	for k, v := range pairs {
		if prefix != "" {
			row[prefix+"."+k] = v
		} else {
			row[k] = v
		}
	}
}
