package platform

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/httpserver"
)

// TestPlatformIntegration spins up the full service stack against an
// ephemeral data directory and exercises the in-process composition that
// individual unit tests can't cover: ingest → detection rule fires →
// alert raised → audit chain anchored → reconstruction observes → trust
// graded → tamper signal flagged.
func TestPlatformIntegration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OBLIVRA_DATA_DIR", dir)

	logger := slog.New(slog.NewTextHandler(silent{}, nil))
	stack, err := New(Options{Logger: logger, InMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	defer stack.Close()

	// Wrap with the HTTP server so we exercise auditmw + the JSON shapes the
	// REST handlers produce. Bound to a random port via httptest.
	srv := httpserver.New(logger, httpserver.Deps{
		System:         stack.System,
		Siem:           stack.Siem,
		Alerts:         stack.Alerts,
		Intel:          stack.Intel,
		Rules:          stack.Rules,
		Audit:          stack.Audit,
		Fleet:          stack.Fleet,
		Ueba:           stack.Ueba,
		Ndr:            stack.Ndr,
		Foren:          stack.Foren,
		Tier:           stack.Tier,
		Lineage:        stack.Lineage,
		Vault:          stack.Vault,
		Timeline:       stack.Timeline,
		Investigations: stack.Investigations,
		Reconstruction: stack.Reconstruction,
		TenantPolicy:   stack.TenantPolicy,
		Trust:          stack.Trust,
		Quality:        stack.Quality,
		Graph:          stack.Graph,
		Import:         stack.Import,
		Report:         stack.Report,
		Tamper:         stack.Tamper,
		Bus:            stack.Bus,
	})
	httpSrv := httptest.NewServer(srv.Handler())
	defer httpSrv.Close()

	post := func(path string, body any) (int, []byte) {
		var b []byte
		if body != nil {
			b, _ = json.Marshal(body)
		}
		req, _ := http.NewRequest("POST", httpSrv.URL+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		out, _ := readAll(resp.Body)
		return resp.StatusCode, out
	}
	get := func(path string) (int, []byte) {
		resp, err := http.Get(httpSrv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		out, _ := readAll(resp.Body)
		return resp.StatusCode, out
	}

	// 1. Ingest an event that fires multiple builtin rules + leaves a tamper
	//    signal (auditctl -D + Failed password from a known-bad IP).
	probes := []map[string]any{
		{
			"source": "rest", "hostId": "web-01", "severity": "warning",
			"message": "sshd Failed password for root from 198.51.100.7 port 22",
		},
		{
			"source": "rest", "hostId": "web-01", "severity": "info",
			"message": "auditctl -D — auditd rules cleared by attacker",
		},
		{
			"source": "rest", "hostId": "win-01", "eventType": "process_creation",
			"message": "powershell.exe -nop -enc JABzPSdpZW...",
		},
	}
	for _, p := range probes {
		if code, body := post("/api/v1/siem/ingest", p); code != 202 {
			t.Fatalf("ingest %v: status %d body=%s", p, code, string(body))
		}
	}

	// Give the async fan-out a moment.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if stack.Alerts.Count() >= 3 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// 2. Alerts fired (sshd brute-force + IOC match + powershell-encoded).
	if got := stack.Alerts.Count(); got < 2 {
		t.Errorf("expected ≥2 alerts to fire, got %d", got)
	}

	// 3. Audit chain holds.
	if r := stack.Audit.Verify(); !r.OK {
		t.Fatalf("audit chain broken: brokenAt=%d", r.BrokenAt)
	}

	// 4. Trust grades populated.
	code, body := get("/api/v1/trust/summary")
	if code != 200 {
		t.Fatalf("trust summary: %d %s", code, body)
	}
	var summary struct {
		Verified, Consistent, Suspicious, Untrusted int
	}
	_ = json.Unmarshal(body, &summary)
	total := summary.Verified + summary.Consistent + summary.Suspicious + summary.Untrusted
	if total < 3 {
		t.Errorf("trust summary should have ≥3 records, got total=%d", total)
	}

	// 5. Tamper findings raised on the auditctl -D message.
	code, body = get("/api/v1/forensics/tamper")
	if code != 200 {
		t.Fatalf("tamper findings: %d", code)
	}
	if !bytes.Contains(body, []byte("auditd")) {
		t.Errorf("expected auditd-disabled finding, got %s", string(body))
	}

	// 6. Open a case with the cutoff *before* a fresh ingest, then confirm
	//    the case timeline excludes it.
	code, body = post("/api/v1/cases", map[string]any{
		"title": "integration", "hostId": "web-01",
	})
	if code != 201 {
		t.Fatalf("open case: %d %s", code, body)
	}
	var caseResp map[string]any
	_ = json.Unmarshal(body, &caseResp)
	caseID, _ := caseResp["id"].(string)
	if caseID == "" {
		t.Fatal("case id missing")
	}

	// Sleep so the next event's receivedAt is after the case cutoff.
	time.Sleep(50 * time.Millisecond)
	post("/api/v1/siem/ingest", map[string]any{
		"source": "rest", "hostId": "web-01", "message": "POST-CASE-EVENT",
	})
	time.Sleep(50 * time.Millisecond)

	code, body = get("/api/v1/cases/" + caseID + "/timeline")
	if code != 200 {
		t.Fatalf("case timeline: %d %s", code, body)
	}
	if bytes.Contains(body, []byte("POST-CASE-EVENT")) {
		t.Fatal("snapshot leak: post-case event surfaced through case timeline")
	}

	// 7. Generate the HTML evidence package.
	code, body = get("/api/v1/cases/" + caseID + "/report.html")
	if code != 200 {
		t.Fatalf("report html: %d", code)
	}
	if !bytes.Contains(body, []byte("OBLIVRA Evidence Package")) {
		t.Errorf("evidence HTML body missing header")
	}

	// 8. Audit chain still verifies after every action.
	if r := stack.Audit.Verify(); !r.OK {
		t.Fatalf("audit chain broken after full flow: brokenAt=%d", r.BrokenAt)
	}

	// 9. Confirm a case-open audit entry made it into the chain.
	recent := stack.Audit.Recent(50)
	foundOpen := false
	for _, e := range recent {
		if e.Action == "investigation.open" {
			foundOpen = true
			break
		}
	}
	if !foundOpen {
		t.Errorf("investigation.open missing from audit chain")
	}
}

// silent is a no-op io.Writer for slog so the test output isn't drowned.
type silent struct{}

func (silent) Write(p []byte) (int, error) { return len(p), nil }

// readAll is the same as io.ReadAll — written inline to keep the test file's
// imports minimal.
func readAll(r interface {
	Read(p []byte) (n int, err error)
}) ([]byte, error) {
	buf := make([]byte, 0, 1<<14)
	tmp := make([]byte, 4<<10)
	for {
		n, err := r.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return buf, nil
}

