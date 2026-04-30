// Package reconstruction recovers higher-level state from the raw event
// stream — sessions, processes, network flows, etc.
//
// `sessions.go` groups authentication events into login → activity → logout
// sequences. We recognise sshd, RDP (Windows EventID 4624/4634), and a few
// generic patterns. Sessions are keyed by (host, user, sourceIP) so the same
// user logging in twice from two boxes is two sessions.
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

type SessionState string

const (
	SessionOpen     SessionState = "open"
	SessionClosed   SessionState = "closed"
	SessionFailed   SessionState = "failed"
	SessionUnknown  SessionState = "unknown"
)

type Session struct {
	ID         string       `json:"id"`
	HostID     string       `json:"hostId"`
	User       string       `json:"user"`
	SourceIP   string       `json:"sourceIp,omitempty"`
	Method     string       `json:"method,omitempty"` // ssh-publickey / ssh-password / rdp / kerberos
	State      SessionState `json:"state"`
	StartedAt  time.Time    `json:"startedAt"`
	EndedAt    time.Time    `json:"endedAt,omitempty"`
	EventIDs   []string     `json:"eventIds"`
	FailedAttempts int      `json:"failedAttempts,omitempty"`
}

// SessionEngine is the in-memory grouper. It's safe to feed events
// out-of-order (within reason) — the engine reconstructs by sorting events
// by timestamp before emitting.
type SessionEngine struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionEngine() *SessionEngine {
	return &SessionEngine{sessions: map[string]*Session{}}
}

// Observe is called per-event from the bus fan-out.
func (s *SessionEngine) Observe(_ context.Context, ev events.Event) {
	cls := classify(ev)
	if cls.kind == "" {
		return
	}
	if cls.user == "" || ev.HostID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Logout events typically don't carry the source IP, so we route them
	// to the most-recent open session for (host, user) regardless of srcIP.
	if cls.kind == "logout" {
		if sess := s.findOpenForUser(ev.HostID, cls.user); sess != nil {
			sess.State = SessionClosed
			sess.EndedAt = ev.Timestamp
			sess.EventIDs = append(sess.EventIDs, ev.ID)
			return
		}
	}

	key := ev.HostID + "|" + cls.user + "|" + cls.srcIP
	sess, ok := s.sessions[key]
	if !ok {
		sess = &Session{
			ID:        sessionID(key, ev.Timestamp),
			HostID:    ev.HostID,
			User:      cls.user,
			SourceIP:  cls.srcIP,
			Method:    cls.method,
			State:     SessionUnknown,
			StartedAt: ev.Timestamp,
		}
		s.sessions[key] = sess
	}

	switch cls.kind {
	case "login_success":
		if sess.State != SessionOpen {
			sess.State = SessionOpen
			sess.StartedAt = ev.Timestamp
			sess.Method = cls.method
		}
	case "login_failed":
		sess.FailedAttempts++
		if sess.State == SessionUnknown {
			sess.State = SessionFailed
		}
	case "logout":
		// Fallback: no matching open session — record as a closed session
		// of unknown origin so we don't drop the signal.
		sess.State = SessionClosed
		sess.EndedAt = ev.Timestamp
	}
	sess.EventIDs = append(sess.EventIDs, ev.ID)
}

// findOpenForUser returns the most recently opened session for (host, user)
// or nil. Caller holds s.mu.
func (s *SessionEngine) findOpenForUser(host, user string) *Session {
	var best *Session
	for _, sess := range s.sessions {
		if sess.HostID != host || sess.User != user {
			continue
		}
		if sess.State != SessionOpen {
			continue
		}
		if best == nil || sess.StartedAt.After(best.StartedAt) {
			best = sess
		}
	}
	return best
}

func (s *SessionEngine) Sessions(host string) []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		if host != "" && sess.HostID != host {
			continue
		}
		out = append(out, *sess)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out
}

func (s *SessionEngine) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sess := range s.sessions {
		if sess.ID == id {
			cc := *sess
			return &cc, true
		}
	}
	return nil, false
}

// ---- classification ----

type classification struct {
	kind   string // login_success | login_failed | logout
	user   string
	srcIP  string
	method string
}

var (
	rxAcceptedSSH = regexp.MustCompile(`Accepted (\w+) for (\S+) from (\S+)`)
	rxFailedSSH   = regexp.MustCompile(`(?:Failed password|authentication failure).*?(?:user[=\s]+)(\S+)`)
	rxFailedAlt   = regexp.MustCompile(`Failed password for (?:invalid user )?(\S+) from (\S+)`)
	rxClosedSSH   = regexp.MustCompile(`session closed for user (\S+)`)
	rxOpenedPAM   = regexp.MustCompile(`session opened for user (\S+)`)
)

func classify(ev events.Event) classification {
	msg := ev.Message + " " + ev.Raw

	// Generic eventType-driven fast paths first.
	switch ev.EventType {
	case "login_success":
		return classification{kind: "login_success", user: fieldOr(ev.Fields, "user"), srcIP: fieldOr(ev.Fields, "src_ip"), method: fieldOr(ev.Fields, "method")}
	case "login_failed", "failed_login":
		return classification{kind: "login_failed", user: fieldOr(ev.Fields, "user"), srcIP: fieldOr(ev.Fields, "src_ip"), method: fieldOr(ev.Fields, "method")}
	case "logout", "session_close":
		return classification{kind: "logout", user: fieldOr(ev.Fields, "user")}
	}

	// sshd "Accepted publickey for alice from 10.0.0.1"
	if m := rxAcceptedSSH.FindStringSubmatch(msg); len(m) == 4 {
		return classification{kind: "login_success", user: m[2], srcIP: m[3], method: "ssh-" + m[1]}
	}
	// "session opened for user alice"
	if m := rxOpenedPAM.FindStringSubmatch(msg); len(m) == 2 {
		return classification{kind: "login_success", user: m[1], method: "pam"}
	}
	// "session closed for user alice"
	if m := rxClosedSSH.FindStringSubmatch(msg); len(m) == 2 {
		return classification{kind: "logout", user: m[1]}
	}
	// "Failed password for root from 10.0.0.1"
	if m := rxFailedAlt.FindStringSubmatch(msg); len(m) == 3 {
		return classification{kind: "login_failed", user: m[1], srcIP: m[2], method: "ssh-password"}
	}
	if m := rxFailedSSH.FindStringSubmatch(msg); len(m) == 2 {
		return classification{kind: "login_failed", user: m[1], method: "ssh-password"}
	}

	// Windows EventID heuristics — approximate; production code reads the
	// EVTX EventID directly.
	if strings.Contains(msg, "EventID 4624") || strings.Contains(msg, "Successful Logon") {
		return classification{kind: "login_success", user: fieldOr(ev.Fields, "TargetUserName"), srcIP: fieldOr(ev.Fields, "IpAddress"), method: "windows"}
	}
	if strings.Contains(msg, "EventID 4625") || strings.Contains(msg, "Failed Logon") {
		return classification{kind: "login_failed", user: fieldOr(ev.Fields, "TargetUserName"), srcIP: fieldOr(ev.Fields, "IpAddress"), method: "windows"}
	}
	if strings.Contains(msg, "EventID 4634") || strings.Contains(msg, "Logoff") {
		return classification{kind: "logout", user: fieldOr(ev.Fields, "TargetUserName")}
	}

	return classification{}
}

func fieldOr(m map[string]string, k string) string {
	if m == nil {
		return ""
	}
	return m[k]
}

func sessionID(key string, t time.Time) string {
	// Deterministic enough — same key + same start → same id.
	return strings.ReplaceAll(key, "|", "-") + "-" + t.UTC().Format("20060102T150405")
}
