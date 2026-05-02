package listeners

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// TestDecodeWriteRequest_Roundtrip hand-builds a Prometheus
// WriteRequest payload (one TimeSeries with two labels and two
// samples), then runs it through our minimal decoder and checks the
// shape. We do this without the official prompb dependency to make
// sure our wire-format reader stays correct as the codebase evolves.
func TestDecodeWriteRequest_Roundtrip(t *testing.T) {
	// Build a sample WriteRequest by hand.
	//
	//   tag = (field << 3) | wire_type
	//
	// timeseries field 1 (length-delimited) → contains
	//   labels[]   field 1 (length-delimited)
	//     name  field 1 (length-delimited): "__name__"
	//     value field 2 (length-delimited): "http_requests_total"
	//   labels[]   field 1 (length-delimited)
	//     name  field 1: "method"
	//     value field 2: "GET"
	//   samples[]  field 2 (length-delimited)
	//     value     field 1 (fixed64 / double): 42.5
	//     timestamp field 2 (varint):           1714521600000
	//   samples[]  field 2
	//     value     field 1: 100
	//     timestamp field 2: 1714521660000

	enc := func(tag, wt int) []byte {
		return varint(uint64(tag<<3) | uint64(wt))
	}
	str := func(tag int, s string) []byte {
		out := append(enc(tag, 2), varint(uint64(len(s)))...)
		return append(out, s...)
	}
	lbl := func(name, val string) []byte {
		body := append(str(1, name), str(2, val)...)
		out := append(enc(1, 2), varint(uint64(len(body)))...)
		return append(out, body...)
	}
	smp := func(value float64, tsMs int64) []byte {
		var fixed [8]byte
		binary.LittleEndian.PutUint64(fixed[:], math.Float64bits(value))
		body := append(enc(1, 1), fixed[:]...)
		body = append(body, enc(2, 0)...)
		body = append(body, varint(uint64(tsMs))...)
		out := append(enc(2, 2), varint(uint64(len(body)))...)
		return append(out, body...)
	}
	tsBody := bytes.NewBuffer(nil)
	tsBody.Write(lbl("__name__", "http_requests_total"))
	tsBody.Write(lbl("method", "GET"))
	tsBody.Write(smp(42.5, 1714521600000))
	tsBody.Write(smp(100, 1714521660000))

	wr := bytes.NewBuffer(nil)
	wr.Write(enc(1, 2))
	wr.Write(varint(uint64(tsBody.Len())))
	wr.Write(tsBody.Bytes())

	req, err := decodeWriteRequest(wr.Bytes())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(req.timeSeries) != 1 {
		t.Fatalf("timeseries = %d, want 1", len(req.timeSeries))
	}
	ts := req.timeSeries[0]
	if len(ts.labels) != 2 {
		t.Errorf("labels = %d, want 2", len(ts.labels))
	}
	if ts.labels[0].name != "__name__" || ts.labels[0].value != "http_requests_total" {
		t.Errorf("label[0] = %v", ts.labels[0])
	}
	if ts.labels[1].name != "method" || ts.labels[1].value != "GET" {
		t.Errorf("label[1] = %v", ts.labels[1])
	}
	if len(ts.samples) != 2 {
		t.Fatalf("samples = %d, want 2", len(ts.samples))
	}
	if ts.samples[0].value != 42.5 {
		t.Errorf("sample[0].value = %v", ts.samples[0].value)
	}
	if ts.samples[0].timestampMs != 1714521600000 {
		t.Errorf("sample[0].ts = %d", ts.samples[0].timestampMs)
	}
	if ts.samples[1].value != 100 {
		t.Errorf("sample[1].value = %v", ts.samples[1].value)
	}
}

// TestDecodeWriteRequest_SkipsUnknownFields makes sure we ignore
// metadata (field 3) and other future additions instead of erroring.
func TestDecodeWriteRequest_SkipsUnknownFields(t *testing.T) {
	enc := func(tag, wt int) []byte { return varint(uint64(tag<<3) | uint64(wt)) }
	// Just metadata field 3 (length-delimited) with empty body — must skip.
	buf := append(enc(3, 2), varint(0)...)
	if _, err := decodeWriteRequest(buf); err != nil {
		t.Errorf("expected no error skipping unknown field, got %v", err)
	}
}

// TestDecodeWriteRequest_TruncatedBody errors out cleanly.
func TestDecodeWriteRequest_TruncatedBody(t *testing.T) {
	enc := func(tag, wt int) []byte { return varint(uint64(tag<<3) | uint64(wt)) }
	// Claim length 100 but provide 0 bytes.
	buf := append(enc(1, 2), varint(100)...)
	if _, err := decodeWriteRequest(buf); err == nil {
		t.Error("expected error for truncated body")
	}
}

func TestFormatFloat(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{42, "42"},
		{42.5, "42.5"},
		{0.001, "0.001"},
		{math.NaN(), "NaN"},
		{math.Inf(1), "+Inf"},
		{math.Inf(-1), "-Inf"},
	}
	for _, c := range cases {
		if got := formatFloat(c.in); got != c.want {
			t.Errorf("formatFloat(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// varint encodes a uint64 in protobuf little-endian-base128 form.
func varint(x uint64) []byte {
	var buf [10]byte
	i := 0
	for x >= 0x80 {
		buf[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	buf[i] = byte(x)
	return buf[:i+1]
}
