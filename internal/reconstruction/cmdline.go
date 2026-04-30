package reconstruction

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// CmdLine records a reconstructed command-line invocation.
type CmdLine struct {
	HostID    string    `json:"hostId"`
	User      string    `json:"user,omitempty"`
	PID       int       `json:"pid,omitempty"`
	PPID      int       `json:"ppid,omitempty"`
	Image     string    `json:"image,omitempty"`
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
	EventID   string    `json:"eventId"`
	Suspicious bool     `json:"suspicious,omitempty"`
}

// CmdLineEngine harvests command-line invocations from process_creation
// events (Windows EventID 4688, Linux execve / auditd) and ranks them by
// suspicion score. The score is a small handful of LOLBin / encoded-command
// heuristics — same shape as the builtin detection rules but specialised
// for command-line surfacing.
type CmdLineEngine struct {
	mu  sync.RWMutex
	all []CmdLine
	cap int
}

func NewCmdLineEngine() *CmdLineEngine {
	return &CmdLineEngine{cap: 5000}
}

var (
	rxCmdLine = regexp.MustCompile(`(?i)CommandLine[:=\s]+"?([^"\n]+)"?`)
	rxImage   = regexp.MustCompile(`(?i)Image[:=\s]+([^\s,]+)`)
	rxExecve  = regexp.MustCompile(`execve\("?([^"\s]+)"?(?:[,\s]+"?([^"\n]*?)"?)?\)`)
)

func (e *CmdLineEngine) Observe(_ context.Context, ev events.Event) {
	cmd, image, ok := extractCommand(ev)
	if !ok {
		return
	}
	cl := CmdLine{
		HostID:    ev.HostID,
		PID:       parsePID(ev),
		Image:     image,
		Command:   cmd,
		Timestamp: ev.Timestamp,
		EventID:   ev.ID,
		User:      pickUser(ev),
	}
	cl.Suspicious = scoreSuspicious(cmd) > 0
	e.mu.Lock()
	e.all = append(e.all, cl)
	if len(e.all) > e.cap {
		e.all = e.all[len(e.all)-e.cap:]
	}
	e.mu.Unlock()
}

func extractCommand(ev events.Event) (cmd, image string, ok bool) {
	src := ev.Message + " " + ev.Raw
	for k, v := range ev.Fields {
		src += " " + k + "=" + v
	}
	if v, has := ev.Fields["CommandLine"]; has && v != "" {
		cmd = v
	}
	if v, has := ev.Fields["command"]; has && v != "" {
		cmd = v
	}
	if cmd == "" {
		if m := rxCmdLine.FindStringSubmatch(src); len(m) == 2 {
			cmd = strings.TrimSpace(m[1])
		}
	}
	if cmd == "" {
		if m := rxExecve.FindStringSubmatch(src); len(m) >= 2 {
			cmd = m[1]
			if len(m) >= 3 && m[2] != "" {
				cmd += " " + m[2]
			}
		}
	}
	if v, has := ev.Fields["Image"]; has {
		image = v
	}
	if image == "" {
		if m := rxImage.FindStringSubmatch(src); len(m) == 2 {
			image = m[1]
		}
	}
	return cmd, image, cmd != ""
}

// scoreSuspicious is intentionally cheap. The Sigma rules already do the
// real work — this just lights up the cmdline list with a flag so the
// analyst knows which entries are worth opening first.
func scoreSuspicious(cmd string) int {
	c := strings.ToLower(cmd)
	hits := 0
	for _, needle := range []string{
		"powershell -enc", "powershell.exe -encodedcommand",
		"iex (", "downloadstring", "invoke-expression",
		"certutil -decode", "certutil -urlcache",
		"mshta http", "mshta javascript:",
		"rundll32 ", "wmic /node:",
		"vssadmin delete", "wbadmin delete",
		"bcdedit /set", "schtasks /create",
		"netsh advfirewall set",
		"curl ", "wget ",
		"| bash", "| sh", "| zsh",
	} {
		if strings.Contains(c, needle) {
			hits++
		}
	}
	return hits
}

// ByHost returns the most recent N invocations on a host (or all hosts if "").
func (e *CmdLineEngine) ByHost(host string, limit int) []CmdLine {
	if limit <= 0 {
		limit = 100
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]CmdLine, 0, limit)
	for i := len(e.all) - 1; i >= 0 && len(out) < limit; i-- {
		c := e.all[i]
		if host != "" && c.HostID != host {
			continue
		}
		out = append(out, c)
	}
	return out
}

// Suspicious returns the most-recently-flagged invocations across the fleet.
func (e *CmdLineEngine) Suspicious(limit int) []CmdLine {
	if limit <= 0 {
		limit = 100
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]CmdLine, 0, limit)
	for i := len(e.all) - 1; i >= 0 && len(out) < limit; i-- {
		if e.all[i].Suspicious {
			out = append(out, e.all[i])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	return out
}
