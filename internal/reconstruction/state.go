package reconstruction

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/storage/hot"
)

// ProcessSnapshot is the reconstructed view of running processes on a host
// at a given moment. "Running" = we saw a creation event but no matching
// exit before T.
type ProcessSnapshot struct {
	HostID   string         `json:"hostId"`
	At       time.Time      `json:"at"`
	Running  []ProcessEntry `json:"running"`
	Exited   []ProcessEntry `json:"exited"`
}

type ProcessEntry struct {
	PID     int       `json:"pid"`
	PPID    int       `json:"ppid,omitempty"`
	Image   string    `json:"image,omitempty"`
	Command string    `json:"command,omitempty"`
	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended,omitempty"`
	EventID string    `json:"eventId,omitempty"`
}

// StateService produces a process-state snapshot for a host at time T.
type StateService struct {
	hot *hot.Store
}

func NewStateService(h *hot.Store) *StateService {
	return &StateService{hot: h}
}

// Reconstruct walks events for the host up to `at` and returns the implied
// running/exited process sets.
func (s *StateService) Reconstruct(ctx context.Context, tenantID, hostID string, at time.Time) (*ProcessSnapshot, error) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	evs, err := s.hot.Range(ctx, hot.RangeOpts{
		TenantID: tenantID,
		From:     time.Unix(0, 0),
		To:       at,
		Limit:    50000,
	})
	if err != nil {
		return nil, err
	}

	// pid → entry
	live := map[int]*ProcessEntry{}
	exited := []ProcessEntry{}

	for _, ev := range evs {
		if ev.HostID != hostID {
			continue
		}
		kind, pid, ppid, image, cmd := classifyProcess(ev)
		if pid <= 0 {
			continue
		}
		switch kind {
		case "create":
			live[pid] = &ProcessEntry{
				PID: pid, PPID: ppid, Image: image, Command: cmd,
				Started: ev.Timestamp, EventID: ev.ID,
			}
		case "exit":
			if entry, ok := live[pid]; ok {
				entry.Ended = ev.Timestamp
				exited = append(exited, *entry)
				delete(live, pid)
			}
		}
	}

	running := make([]ProcessEntry, 0, len(live))
	for _, e := range live {
		running = append(running, *e)
	}
	sort.Slice(running, func(i, j int) bool { return running[i].PID < running[j].PID })
	sort.Slice(exited, func(i, j int) bool { return exited[i].Ended.After(exited[j].Ended) })
	return &ProcessSnapshot{
		HostID:  hostID,
		At:      at,
		Running: running,
		Exited:  exited,
	}, nil
}

// ---- classification ----

var (
	rxNewPIDHex   = regexp.MustCompile(`(?i)NewProcessId:\s*0x([0-9a-f]+)`)
	rxParentHex   = regexp.MustCompile(`(?i)ParentProcessId:\s*0x([0-9a-f]+)`)
	rxImageField  = regexp.MustCompile(`(?i)Image[:\s=]+([^\s,]+)`)
	rxCmdField    = regexp.MustCompile(`(?i)CommandLine[:\s=]+"?([^"\n]+)"?`)
	rxBracketProc = regexp.MustCompile(`(\w+)\[(\d+)\]`)
	rxKernelExit  = regexp.MustCompile(`(?i)\bpid\s*=\s*(\d+).*\b(exit|terminate|killed)`)
)

func classifyProcess(ev events.Event) (kind string, pid, ppid int, image, cmd string) {
	if ev.EventType != "" {
		switch strings.ToLower(ev.EventType) {
		case "process_creation", "process.create", "process_start":
			return finishCreate(ev)
		case "process_exit", "process.exit", "process_stop":
			return "exit", parsePID(ev), 0, "", ""
		}
	}

	src := ev.Message + " " + ev.Raw
	for k, v := range ev.Fields {
		src += " " + k + "=" + v
	}
	lower := strings.ToLower(src)

	if strings.Contains(lower, "newprocessid") || strings.Contains(lower, "process_creation") || strings.Contains(lower, "process started") {
		return finishCreate(ev)
	}
	if m := rxKernelExit.FindStringSubmatch(src); len(m) == 3 {
		p, _ := strconv.Atoi(m[1])
		return "exit", p, 0, "", ""
	}
	return "", 0, 0, "", ""
}

func finishCreate(ev events.Event) (string, int, int, string, string) {
	src := ev.Message + " " + ev.Raw
	for k, v := range ev.Fields {
		src += " " + k + "=" + v
	}

	pid := 0
	ppid := 0
	if m := rxNewPIDHex.FindStringSubmatch(src); len(m) == 2 {
		n, _ := strconv.ParseInt(m[1], 16, 64)
		pid = int(n)
	}
	if m := rxParentHex.FindStringSubmatch(src); len(m) == 2 {
		n, _ := strconv.ParseInt(m[1], 16, 64)
		ppid = int(n)
	}
	if pid == 0 {
		pid = parsePID(ev)
	}
	if pid == 0 {
		return "", 0, 0, "", ""
	}
	image := ""
	if m := rxImageField.FindStringSubmatch(src); len(m) == 2 {
		image = m[1]
	} else if m := rxBracketProc.FindStringSubmatch(src); len(m) == 3 {
		image = m[1]
	}
	cmd := ""
	if m := rxCmdField.FindStringSubmatch(src); len(m) == 2 {
		cmd = strings.TrimSpace(m[1])
	}
	return "create", pid, ppid, image, cmd
}

func parsePID(ev events.Event) int {
	src := ev.Message + " " + ev.Raw
	if v, ok := ev.Fields["pid"]; ok {
		n, _ := strconv.Atoi(v)
		if n > 0 {
			return n
		}
	}
	if m := rxBracketProc.FindStringSubmatch(src); len(m) == 3 {
		n, _ := strconv.Atoi(m[2])
		return n
	}
	if i := strings.Index(src, "pid="); i >= 0 {
		end := i + 4
		for end < len(src) && src[end] >= '0' && src[end] <= '9' {
			end++
		}
		n, _ := strconv.Atoi(src[i+4 : end])
		return n
	}
	return 0
}
