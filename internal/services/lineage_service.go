package services

import (
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// LineageNode is one process observed in a log message. We synthesise the tree
// from PID/PPID values seen on a host — this is best-effort and lossy by
// nature, since logs only show what they show.
type LineageNode struct {
	HostID    string    `json:"hostId"`
	PID       int       `json:"pid"`
	PPID      int       `json:"ppid"`
	Name      string    `json:"name,omitempty"`
	Command   string    `json:"command,omitempty"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	Events    int       `json:"events"`
}

// Tree is one host's process forest, sorted parents-first.
type Tree struct {
	HostID string         `json:"hostId"`
	Nodes  []*LineageNode `json:"nodes"`
}

type LineageService struct {
	log *slog.Logger
	mu  sync.RWMutex
	// host → pid → node
	byHost map[string]map[int]*LineageNode
}

func NewLineageService(log *slog.Logger) *LineageService {
	return &LineageService{log: log, byHost: map[string]map[int]*LineageNode{}}
}

func (s *LineageService) ServiceName() string { return "LineageService" }

// Observe is called per-event from the bus fan-out.
func (s *LineageService) Observe(ev events.Event) {
	if ev.HostID == "" {
		return
	}
	pid, ppid, name, cmd := extractProc(ev)
	if pid <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	hosts, ok := s.byHost[ev.HostID]
	if !ok {
		hosts = map[int]*LineageNode{}
		s.byHost[ev.HostID] = hosts
	}
	n, ok := hosts[pid]
	if !ok {
		n = &LineageNode{HostID: ev.HostID, PID: pid, FirstSeen: ev.Timestamp}
		hosts[pid] = n
	}
	if ppid > 0 {
		n.PPID = ppid
	}
	if name != "" {
		n.Name = name
	}
	if cmd != "" {
		n.Command = cmd
	}
	n.LastSeen = ev.Timestamp
	n.Events++
}

func (s *LineageService) Tree(host string) Tree {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hosts := s.byHost[host]
	out := Tree{HostID: host}
	for _, n := range hosts {
		out.Nodes = append(out.Nodes, n)
	}
	sort.Slice(out.Nodes, func(i, j int) bool {
		if out.Nodes[i].PPID != out.Nodes[j].PPID {
			return out.Nodes[i].PPID < out.Nodes[j].PPID
		}
		return out.Nodes[i].PID < out.Nodes[j].PID
	})
	return out
}

func (s *LineageService) Hosts() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.byHost))
	for k := range s.byHost {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// extractProc tries a few common log shapes to pull pid/ppid/name/cmd.
//
// Shapes recognised:
//
//	sshd[1234]: ...
//	process_creation pid=4221 ppid=812 name=cmd.exe cmd="cmd /c whoami"
//	[pid 1234] something
//	(EVT 4688) NewProcessId: 0x4d2 ParentProcessId: 0x12 ... Image: foo.exe
var (
	bracketPID = regexp.MustCompile(`(?:^|\s)(\w+)\[(\d+)\]`)
	kvPID      = regexp.MustCompile(`(?i)\bpid\s*=\s*(\d+)`)
	kvPPID     = regexp.MustCompile(`(?i)\bppid\s*=\s*(\d+)`)
	kvName     = regexp.MustCompile(`(?i)\bname\s*=\s*([^\s]+)`)
	kvImage    = regexp.MustCompile(`(?i)\bimage\s*[:=]\s*([^\s,]+)`)
	kvCmd      = regexp.MustCompile(`(?i)\b(cmd|commandline)\s*=\s*"([^"]+)"`)
	hexNewPID  = regexp.MustCompile(`(?i)NewProcessId:\s*0x([0-9a-f]+)`)
	hexPPID    = regexp.MustCompile(`(?i)ParentProcessId:\s*0x([0-9a-f]+)`)
)

func extractProc(ev events.Event) (pid, ppid int, name, cmd string) {
	src := ev.Message + " " + ev.Raw + " "
	for k, v := range ev.Fields {
		src += k + "=" + v + " "
	}

	if m := bracketPID.FindStringSubmatch(src); len(m) == 3 {
		name = m[1]
		pid = atoi(m[2])
	}
	if m := kvPID.FindStringSubmatch(src); len(m) == 2 {
		pid = atoi(m[1])
	}
	if m := kvPPID.FindStringSubmatch(src); len(m) == 2 {
		ppid = atoi(m[1])
	}
	if name == "" {
		if m := kvName.FindStringSubmatch(src); len(m) == 2 {
			name = m[1]
		} else if m := kvImage.FindStringSubmatch(src); len(m) == 2 {
			name = m[1]
		}
	}
	if m := kvCmd.FindStringSubmatch(src); len(m) == 3 {
		cmd = m[2]
	}
	if pid == 0 {
		if m := hexNewPID.FindStringSubmatch(src); len(m) == 2 {
			pid = hexAtoi(m[1])
		}
	}
	if ppid == 0 {
		if m := hexPPID.FindStringSubmatch(src); len(m) == 2 {
			ppid = hexAtoi(m[1])
		}
	}
	return pid, ppid, name, cmd
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func hexAtoi(s string) int {
	n, _ := strconv.ParseInt(s, 16, 64)
	return int(n)
}
