// Package listeners — Prometheus remote_write receiver.
//
// Prometheus and most modern metric agents (Vector, Telegraf, Grafana
// Agent, Mimir, etc.) can push to a remote_write endpoint. The wire
// format is a Snappy-compressed protobuf `WriteRequest`. We decode it
// locally and emit one OBLIVRA event per sample, so metrics get the
// same WAL → hot store → audit chain treatment as logs. That's the
// differentiator: tamper-evident metrics, not just tamper-evident logs.
//
// We hand-roll a minimal protobuf decoder for the four message types
// we need (WriteRequest / TimeSeries / Sample / Label) to keep the
// dependency surface tiny — pulling in the full prometheus/prometheus
// tree just for these would add ~50 transitive deps.
package listeners

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/golang/snappy"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/ingest"
)

// Handler returns an http.HandlerFunc that accepts POST requests at
// /api/v1/metrics/remote_write. Decompresses + decodes the body, emits
// one event per sample, returns 204 on success per the Prometheus spec.
func PromRemoteWriteHandler(log *slog.Logger, p *ingest.Pipeline, tenant string) http.HandlerFunc {
	if tenant == "" {
		tenant = "default"
	}
	var counter atomic.Int64
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}

		// Cap body size — Prometheus flushes ~1MB chunks; a hostile
		// client could try to OOM us with a giant batch otherwise.
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32<<20))
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}

		raw, err := snappy.Decode(nil, body)
		if err != nil {
			http.Error(w, "snappy decode: "+err.Error(), http.StatusBadRequest)
			return
		}

		req, err := decodeWriteRequest(raw)
		if err != nil {
			http.Error(w, "protobuf decode: "+err.Error(), http.StatusBadRequest)
			return
		}

		written := 0
		for _, ts := range req.timeSeries {
			labels := map[string]string{}
			var name string
			for _, l := range ts.labels {
				if l.name == "__name__" {
					name = l.value
					continue
				}
				labels[l.name] = l.value
			}
			if name == "" {
				continue // malformed; skip
			}
			for _, s := range ts.samples {
				ev := &events.Event{
					Source:    events.SourceREST,
					EventType: "metric:" + name,
					TenantID:  tenant,
					Severity:  metricSeverity(name, s.value),
					Message:   fmt.Sprintf("%s = %s", name, formatFloat(s.value)),
					Timestamp: time.Unix(s.timestampMs/1000, (s.timestampMs%1000)*int64(time.Millisecond)).UTC(),
					Fields:    map[string]string{"__name__": name, "value": formatFloat(s.value)},
				}
				for k, v := range labels {
					ev.Fields[k] = v
				}
				if hostL, ok := labels["instance"]; ok && ev.HostID == "" {
					ev.HostID = hostL
				}
				ev.Provenance.IngestPath = "prom-remote-write"
				ev.Provenance.Format = "prometheus.WriteRequest"
				if err := p.Submit(r.Context(), ev); err != nil {
					log.Warn("prom remote_write submit", "err", err)
					continue
				}
				written++
				counter.Add(1)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
		_ = written
	}
}

// ---- minimal protobuf decoder for WriteRequest -------------------------
//
// The Prometheus remote-write proto schema (relevant subset):
//
//   message WriteRequest {
//     repeated TimeSeries timeseries = 1;
//     // metadata field 3 ignored
//   }
//   message TimeSeries {
//     repeated Label  labels  = 1;
//     repeated Sample samples = 2;
//   }
//   message Label  { string name = 1; string value = 2; }
//   message Sample { double value = 1; int64  timestamp = 2; } // ms since epoch
//
// Wire format primer:
//   tag = (field_number << 3) | wire_type
//   wire_type 0 = varint, 1 = fixed64, 2 = length-delimited
//
// We only decode field numbers we care about and skip unknown fields.

type writeRequest struct {
	timeSeries []timeSeries
}

type timeSeries struct {
	labels  []label
	samples []sample
}

type label struct {
	name  string
	value string
}

type sample struct {
	value       float64
	timestampMs int64
}

func decodeWriteRequest(buf []byte) (writeRequest, error) {
	var req writeRequest
	for len(buf) > 0 {
		tag, wireType, n, err := readTag(buf)
		if err != nil {
			return req, err
		}
		buf = buf[n:]
		switch {
		case tag == 1 && wireType == 2:
			body, rest, err := readLenDelim(buf)
			if err != nil {
				return req, err
			}
			ts, err := decodeTimeSeries(body)
			if err != nil {
				return req, err
			}
			req.timeSeries = append(req.timeSeries, ts)
			buf = rest
		default:
			rest, err := skipField(buf, wireType)
			if err != nil {
				return req, err
			}
			buf = rest
		}
	}
	return req, nil
}

func decodeTimeSeries(buf []byte) (timeSeries, error) {
	var ts timeSeries
	for len(buf) > 0 {
		tag, wireType, n, err := readTag(buf)
		if err != nil {
			return ts, err
		}
		buf = buf[n:]
		switch {
		case tag == 1 && wireType == 2:
			body, rest, err := readLenDelim(buf)
			if err != nil {
				return ts, err
			}
			lbl, err := decodeLabel(body)
			if err != nil {
				return ts, err
			}
			ts.labels = append(ts.labels, lbl)
			buf = rest
		case tag == 2 && wireType == 2:
			body, rest, err := readLenDelim(buf)
			if err != nil {
				return ts, err
			}
			s, err := decodeSample(body)
			if err != nil {
				return ts, err
			}
			ts.samples = append(ts.samples, s)
			buf = rest
		default:
			rest, err := skipField(buf, wireType)
			if err != nil {
				return ts, err
			}
			buf = rest
		}
	}
	return ts, nil
}

func decodeLabel(buf []byte) (label, error) {
	var l label
	for len(buf) > 0 {
		tag, wireType, n, err := readTag(buf)
		if err != nil {
			return l, err
		}
		buf = buf[n:]
		switch {
		case tag == 1 && wireType == 2:
			body, rest, err := readLenDelim(buf)
			if err != nil {
				return l, err
			}
			l.name = string(body)
			buf = rest
		case tag == 2 && wireType == 2:
			body, rest, err := readLenDelim(buf)
			if err != nil {
				return l, err
			}
			l.value = string(body)
			buf = rest
		default:
			rest, err := skipField(buf, wireType)
			if err != nil {
				return l, err
			}
			buf = rest
		}
	}
	return l, nil
}

func decodeSample(buf []byte) (sample, error) {
	var s sample
	for len(buf) > 0 {
		tag, wireType, n, err := readTag(buf)
		if err != nil {
			return s, err
		}
		buf = buf[n:]
		switch {
		case tag == 1 && wireType == 1:
			if len(buf) < 8 {
				return s, errors.New("sample value: truncated")
			}
			s.value = math.Float64frombits(binary.LittleEndian.Uint64(buf[:8]))
			buf = buf[8:]
		case tag == 2 && wireType == 0:
			v, n, err := readVarint(buf)
			if err != nil {
				return s, err
			}
			s.timestampMs = int64(v)
			buf = buf[n:]
		default:
			rest, err := skipField(buf, wireType)
			if err != nil {
				return s, err
			}
			buf = rest
		}
	}
	return s, nil
}

// ---- protobuf wire-format primitives ----

func readTag(buf []byte) (tag int, wireType int, n int, err error) {
	v, n, err := readVarint(buf)
	if err != nil {
		return 0, 0, 0, err
	}
	return int(v >> 3), int(v & 0x7), n, nil
}

func readVarint(buf []byte) (uint64, int, error) {
	var x uint64
	var shift uint
	for i := 0; i < len(buf); i++ {
		b := buf[i]
		x |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return x, i + 1, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, 0, errors.New("varint overflow")
		}
	}
	return 0, 0, errors.New("varint truncated")
}

func readLenDelim(buf []byte) (body, rest []byte, err error) {
	n, off, err := readVarint(buf)
	if err != nil {
		return nil, nil, err
	}
	if uint64(len(buf[off:])) < n {
		return nil, nil, errors.New("length-delimited truncated")
	}
	return buf[off : off+int(n)], buf[off+int(n):], nil
}

func skipField(buf []byte, wireType int) ([]byte, error) {
	switch wireType {
	case 0: // varint
		_, n, err := readVarint(buf)
		if err != nil {
			return nil, err
		}
		return buf[n:], nil
	case 1: // fixed64
		if len(buf) < 8 {
			return nil, errors.New("fixed64: truncated")
		}
		return buf[8:], nil
	case 2: // length-delimited
		_, rest, err := readLenDelim(buf)
		return rest, err
	case 5: // fixed32
		if len(buf) < 4 {
			return nil, errors.New("fixed32: truncated")
		}
		return buf[4:], nil
	default:
		return nil, fmt.Errorf("unsupported wire type %d", wireType)
	}
}

// ---- formatting helpers ----

func formatFloat(v float64) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	if math.IsInf(v, 1) {
		return "+Inf"
	}
	if math.IsInf(v, -1) {
		return "-Inf"
	}
	// Use 'g' so integers stay clean ("5") and floats stay readable.
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// metricSeverity makes a *very* light interpretation of the metric so
// the existing severity-based UI still works for metrics. Real
// thresholds belong in saved-search alert rules; this just colours
// `_errors`-suffixed counters higher than a plain gauge.
func metricSeverity(name string, _ float64) events.Severity {
	if len(name) > 7 && name[len(name)-7:] == "_errors" {
		return events.SeverityWarn
	}
	return events.SeverityInfo
}
