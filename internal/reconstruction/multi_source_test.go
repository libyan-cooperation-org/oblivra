package reconstruction

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/storage/hot"
)

// TestMultiSourceReconstruction is the forensic-grade demonstration
// that the platform's "any input, any vendor" claim survives contact
// with the data: it seeds a single host's process tree using FOUR
// different ingest shapes — Windows Sysmon EventID 1, Linux auditd
// SYSCALL execve, classic syslog with `sshd[pid]` decoration, and the
// native OBLIVRA agent JSON — and asserts the reconstructed state
// snapshot contains all four processes regardless of source.
//
// The point of the test is that an analyst in a heterogeneous estate
// (mixed Win/Linux, mixed forwarders) cannot tell — from the
// reconstructed view alone — which ingest path produced each event.
// The platform unifies them.
func TestMultiSourceReconstruction(t *testing.T) {
	store, err := hot.Open(hot.Options{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	host := "host-01"
	t0 := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	// 1) Windows Sysmon-style: hex NewProcessId, hex ParentProcessId,
	//    Image + CommandLine fields. Goes through SourceAgent shape.
	put(t, store, &events.Event{
		Source:    events.SourceAgent,
		HostID:    host,
		EventType: "process_creation",
		Message:   "NewProcessId: 0x4d2 ParentProcessId: 0x32c Image: C:\\Windows\\System32\\cmd.exe CommandLine: \"cmd.exe /c whoami\"",
		Timestamp: t0,
	})

	// 2) Linux auditd-style: pid= and ppid= in fields, raw kernel
	//    message in Raw. Comes off the syslog UDP listener so source
	//    is SourceSyslog.
	put(t, store, &events.Event{
		Source:    events.SourceSyslog,
		HostID:    host,
		EventType: "process_creation",
		Message:   "audit: type=SYSCALL execve",
		Raw:       "type=SYSCALL msg=audit(1714563600.000:42): syscall=execve pid=4242 ppid=1 exe=/usr/bin/curl",
		Fields:    map[string]string{"pid": "4242", "ppid": "1"},
		Timestamp: t0.Add(1 * time.Second),
	})

	// 3) Classic syslog process_started decoration — the parser
	//    falls back to the bracketed `sshd[5678]` form when no
	//    EventType is set. Tests the keyword-based detection path.
	put(t, store, &events.Event{
		Source:    events.SourceREST,
		HostID:    host,
		Message:   "sshd[5678]: process started for user alice",
		Timestamp: t0.Add(2 * time.Second),
	})

	// 4) Native OBLIVRA agent JSON — eventType + pid field.
	put(t, store, &events.Event{
		Source:    events.SourceAgent,
		HostID:    host,
		EventType: "process.create",
		Message:   "process started",
		Fields:    map[string]string{"pid": "9999", "image": "/opt/oblivra/oblivra-agent"},
		Timestamp: t0.Add(3 * time.Second),
	})

	svc := NewStateService(store)
	snap, err := svc.Reconstruct(context.Background(), "default", host, t0.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}

	// Assert all four PIDs land in Running. We use a set lookup so
	// ordering of the parser internals doesn't matter.
	wantPIDs := map[int]string{
		0x4d2: "Sysmon hex pid",
		4242:  "auditd pid= field",
		5678:  "syslog sshd[5678]",
		9999:  "native agent pid field",
	}
	gotPIDs := map[int]bool{}
	for _, p := range snap.Running {
		gotPIDs[p.PID] = true
	}
	for pid, label := range wantPIDs {
		if !gotPIDs[pid] {
			t.Errorf("missing pid %d (%s) — multi-source unification failed", pid, label)
		}
	}
	if len(snap.Running) != 4 {
		t.Errorf("expected 4 running, got %d: %+v", len(snap.Running), snap.Running)
	}

	// Now seed exit events in mixed shapes too — and confirm the
	// reconstructor pairs creates with exits across source formats.
	put(t, store, &events.Event{ // 1: native exit eventType
		Source: events.SourceAgent, HostID: host,
		EventType: "process_exit",
		Message:   "process exited",
		Fields:    map[string]string{"pid": "1234"},
		Timestamp: t0.Add(20 * time.Second),
	})
	put(t, store, &events.Event{ // 2: kernel exit phrasing
		Source:    events.SourceSyslog,
		HostID:    host,
		Message:   "kernel: pid=4242 exit code 0",
		Timestamp: t0.Add(21 * time.Second),
	})

	snap, err = svc.Reconstruct(context.Background(), "default", host, t0.Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	exitedPIDs := map[int]bool{}
	for _, p := range snap.Exited {
		exitedPIDs[p.PID] = true
	}
	if !exitedPIDs[0x4d2] {
		t.Errorf("Sysmon pid 0x4d2 didn't transition to exited (native exit eventType not paired)")
	}
	if !exitedPIDs[4242] {
		t.Errorf("auditd pid 4242 didn't transition to exited (kernel exit phrasing not paired)")
	}

	// And the two unpaired creates (5678, 9999) must STILL be running.
	stillRunning := map[int]bool{}
	for _, p := range snap.Running {
		stillRunning[p.PID] = true
	}
	if !stillRunning[5678] || !stillRunning[9999] {
		t.Errorf("unpaired creates lost their running state: running=%v", snap.Running)
	}
}

// TestMultiSourceReconstruction_TenantAwareness pins a critical
// invariant: even when two tenants share the same hostname (which is
// common in MSP / multi-region deployments), reconstruction never
// merges processes across tenants. Without this property, an analyst
// querying tenant-A's host-01 could see processes from tenant-B's
// host-01 in the snapshot.
func TestMultiSourceReconstruction_TenantAwareness(t *testing.T) {
	store, err := hot.Open(hot.Options{InMemory: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	host := "shared-host"
	t0 := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	put(t, store, &events.Event{
		TenantID: "tenant-a", Source: events.SourceAgent, HostID: host,
		EventType: "process_creation",
		Message:   "NewProcessId: 0x111 ParentProcessId: 0x1 Image: a.exe",
		Timestamp: t0,
	})
	put(t, store, &events.Event{
		TenantID: "tenant-b", Source: events.SourceAgent, HostID: host,
		EventType: "process_creation",
		Message:   "NewProcessId: 0x222 ParentProcessId: 0x1 Image: b.exe",
		Timestamp: t0,
	})

	svc := NewStateService(store)
	a, err := svc.Reconstruct(context.Background(), "tenant-a", host, t0.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Running) != 1 || a.Running[0].PID != 0x111 {
		t.Fatalf("tenant-a reconstruction leaked / missing: %+v", a.Running)
	}
	for _, p := range a.Running {
		if !strings.Contains(strings.ToLower(p.Image), "a.exe") && p.Image != "" {
			t.Fatalf("tenant-a snapshot contains foreign process: %+v", p)
		}
	}

	b, _ := svc.Reconstruct(context.Background(), "tenant-b", host, t0.Add(time.Second))
	if len(b.Running) != 1 || b.Running[0].PID != 0x222 {
		t.Fatalf("tenant-b reconstruction leaked / missing: %+v", b.Running)
	}
}

// put is a tiny helper so the test bodies stay readable.
func put(t *testing.T, store *hot.Store, ev *events.Event) {
	t.Helper()
	if ev.TenantID == "" {
		ev.TenantID = "default"
	}
	if err := ev.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := store.Put(ev); err != nil {
		t.Fatalf("put: %v", err)
	}
}
