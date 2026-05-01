package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/services"
)

// Phase 41 — Universal Forwarder Compatibility.
//
// Two compatibility endpoints so existing forwarders can ship to OBLIVRA
// without re-tooling:
//
//   POST /services/collector/event   — Splunk HEC (newline-delimited or
//                                      single envelope; Authorization:
//                                      Splunk <token>)
//
//   POST /v1/logs                    — OpenTelemetry OTLP/HTTP logs
//                                      (gzip-encoded protobuf is rejected;
//                                      send JSON-encoded OTLP). Sufficient
//                                      for OpenTelemetry Collector with
//                                      `otlphttp/json` exporter.

// hecHandler accepts the canonical Splunk HEC envelope:
//
//	{"event": ..., "host": "...", "source": "...", "sourcetype": "...",
//	 "index": "...", "time": <epoch>, "fields": {...}}
//
// Either a single object OR a stream of objects (no wrapper array). The
// latter is what UF actually sends.
func (s *Server) hecHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.acceptHECAuth(r) {
			writeError(w, http.StatusUnauthorized, "splunk auth required")
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, 16<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		// HEC is two-shape: either bare JSON or NDJSON. We handle both via a
		// streaming decoder.
		dec := json.NewDecoder(strings.NewReader(string(body)))
		written := 0
		for {
			var raw struct {
				Time       any            `json:"time"`
				Host       string         `json:"host"`
				Source     string         `json:"source"`
				Sourcetype string         `json:"sourcetype"`
				Index      string         `json:"index"`
				Event      any            `json:"event"`
				Fields     map[string]any `json:"fields"`
			}
			if err := dec.Decode(&raw); err != nil {
				if err == io.EOF {
					break
				}
				writeError(w, http.StatusBadRequest, "hec parse: "+err.Error())
				return
			}
			ts := splunkTime(raw.Time)
			ev := events.Event{
				TenantID:   firstNonEmpty(raw.Index, "default"),
				Timestamp:  ts,
				ReceivedAt: time.Now().UTC(),
				Source:     events.SourceREST,
				HostID:     raw.Host,
				EventType:  firstNonEmpty(raw.Sourcetype, "splunk:hec"),
				Severity:   events.SeverityInfo,
				Message:    splunkEventBody(raw.Event),
				Fields:     stringifyFields(raw.Fields),
				Provenance: events.Provenance{
					IngestPath: "splunk-hec",
					Peer:       strings.SplitN(r.RemoteAddr, ":", 2)[0],
					Format:     "splunk-hec",
				},
			}
			if _, err := s.siem.Ingest(r.Context(), ev); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			written++
		}
		writeJSON(w, http.StatusOK, map[string]any{"text": "Success", "code": 0, "written": written})
	}
}

// otlpLogsHandler accepts a single OTLP/HTTP logs request encoded as JSON.
// This is what `otlphttp/json` produces. The proto encoding is intentionally
// not supported — we'd need protobuf codegen.
//
// The OTLP shape is:
//
//	{ "resourceLogs": [
//	    { "resource": {"attributes": [...]},
//	      "scopeLogs": [
//	        { "logRecords": [ {"timeUnixNano": "...", "body": {...}, "severityText": "...", "attributes": [...]}, ... ] }
//	      ] } ] }
func (s *Server) otlpLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, 16<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		var doc struct {
			ResourceLogs []struct {
				Resource struct {
					Attributes []otlpKV `json:"attributes"`
				} `json:"resource"`
				ScopeLogs []struct {
					LogRecords []struct {
						TimeUnixNano string   `json:"timeUnixNano"`
						SeverityText string   `json:"severityText"`
						Body         otlpAny  `json:"body"`
						Attributes   []otlpKV `json:"attributes"`
					} `json:"logRecords"`
				} `json:"scopeLogs"`
			} `json:"resourceLogs"`
		}
		if err := json.Unmarshal(body, &doc); err != nil {
			writeError(w, http.StatusBadRequest, "otlp parse: "+err.Error())
			return
		}
		written := 0
		peer := strings.SplitN(r.RemoteAddr, ":", 2)[0]
		for _, rl := range doc.ResourceLogs {
			resAttrs := otlpAttrsToMap(rl.Resource.Attributes)
			host := resAttrs["host.name"]
			service := resAttrs["service.name"]
			for _, sl := range rl.ScopeLogs {
				for _, rec := range sl.LogRecords {
					attrs := otlpAttrsToMap(rec.Attributes)
					for k, v := range resAttrs {
						if _, dup := attrs[k]; !dup {
							attrs[k] = v
						}
					}
					ts := otlpNanoTime(rec.TimeUnixNano)
					ev := events.Event{
						TenantID:   "default",
						Timestamp:  ts,
						ReceivedAt: time.Now().UTC(),
						Source:     events.SourceREST,
						HostID:     host,
						EventType:  firstNonEmpty(service, "otlp"),
						Severity:   otlpSeverity(rec.SeverityText),
						Message:    rec.Body.String(),
						Fields:     attrs,
						Provenance: events.Provenance{
							IngestPath: "otlp-http",
							Peer:       peer,
							Format:     "otlp",
						},
					}
					if _, err := s.siem.Ingest(r.Context(), ev); err != nil {
						writeError(w, http.StatusInternalServerError, err.Error())
						return
					}
					written++
				}
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"written": written})
	}
}

// ---- HEC helpers ----

func (s *Server) acceptHECAuth(r *http.Request) bool {
	// HEC sends "Authorization: Splunk <token>". Map it to an OBLIVRA bearer
	// by stripping the prefix and letting the existing auth middleware
	// handle the rest. If our auth middleware is disabled we accept any.
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Splunk ") {
		r.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(h, "Splunk "))
	}
	if s.auth == nil || !s.auth.Required() {
		return true
	}
	// Defer to the standard middleware via a synthetic check.
	return r.Header.Get("Authorization") != ""
}

func splunkTime(v any) time.Time {
	switch t := v.(type) {
	case float64:
		// HEC times are seconds since epoch (with optional fraction).
		sec := int64(t)
		nsec := int64((t - float64(sec)) * 1e9)
		return time.Unix(sec, nsec).UTC()
	case string:
		if t == "" {
			return time.Now().UTC()
		}
		if ts, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return ts.UTC()
		}
		return time.Now().UTC()
	default:
		return time.Now().UTC()
	}
}

func splunkEventBody(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func stringifyFields(in map[string]any) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		switch t := v.(type) {
		case string:
			out[k] = t
		case nil:
			// skip
		default:
			b, _ := json.Marshal(t)
			out[k] = string(b)
		}
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// ---- OTLP helpers ----

type otlpKV struct {
	Key   string  `json:"key"`
	Value otlpAny `json:"value"`
}

type otlpAny struct {
	StringValue *string  `json:"stringValue,omitempty"`
	IntValue    *string  `json:"intValue,omitempty"` // OTLP encodes int64 as a string
	DoubleValue *float64 `json:"doubleValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
}

func (a otlpAny) String() string {
	switch {
	case a.StringValue != nil:
		return *a.StringValue
	case a.IntValue != nil:
		return *a.IntValue
	case a.DoubleValue != nil:
		return fmt.Sprintf("%g", *a.DoubleValue)
	case a.BoolValue != nil:
		if *a.BoolValue {
			return "true"
		}
		return "false"
	}
	return ""
}

func otlpAttrsToMap(in []otlpKV) map[string]string {
	out := make(map[string]string, len(in))
	for _, kv := range in {
		out[kv.Key] = kv.Value.String()
	}
	return out
}

func otlpNanoTime(s string) time.Time {
	if s == "" {
		return time.Now().UTC()
	}
	var ns int64
	if _, err := fmt.Sscanf(s, "%d", &ns); err != nil || ns == 0 {
		return time.Now().UTC()
	}
	return time.Unix(0, ns).UTC()
}

func otlpSeverity(s string) events.Severity {
	switch strings.ToLower(s) {
	case "trace", "debug", "trace2", "trace3", "trace4", "debug2", "debug3", "debug4":
		return events.SeverityDebug
	case "info", "info2", "info3", "info4":
		return events.SeverityInfo
	case "warn", "warning", "warn2", "warn3", "warn4":
		return events.SeverityWarn
	case "error", "error2", "error3", "error4":
		return events.SeverityError
	case "fatal", "fatal2", "fatal3", "fatal4":
		return events.SeverityCritical
	default:
		return events.SeverityInfo
	}
}

// silenceUnused so import-trim doesn't strip services
var _ = services.NewSiemService
