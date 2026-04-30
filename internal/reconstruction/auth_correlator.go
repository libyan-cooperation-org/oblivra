package reconstruction

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// AuthEvent is one normalised auth observation across protocols.
type AuthEvent struct {
	User      string    `json:"user"`
	HostID    string    `json:"hostId"`
	SourceIP  string    `json:"sourceIp,omitempty"`
	Protocol  string    `json:"protocol"` // ssh | rdp | kerberos | web | pam
	Result    string    `json:"result"`   // success | failure
	Timestamp time.Time `json:"timestamp"`
	EventID   string    `json:"eventId"`
}

// AuthChain is a per-user, per-day grouping of all auth events across every
// protocol — the analyst-facing "show me everywhere alice logged in today"
// view. It's how cross-protocol attacks (kerberoast → SMB lateral → RDP) get
// surfaced as one story instead of three disconnected rule-fires.
type AuthChain struct {
	User    string      `json:"user"`
	Day     string      `json:"day"` // YYYY-MM-DD
	Events  []AuthEvent `json:"events"`
	Hosts   []string    `json:"hosts"`
	IPs     []string    `json:"ips"`
	Protocols []string  `json:"protocols"`
	Failures int        `json:"failures"`
	Successes int       `json:"successes"`
}

type AuthCorrelator struct {
	mu     sync.RWMutex
	byKey  map[string]*AuthChain // user|day → chain
	cap    int
}

func NewAuthCorrelator() *AuthCorrelator {
	return &AuthCorrelator{byKey: map[string]*AuthChain{}, cap: 5000}
}

// Observe is called per-event from the bus fan-out.
func (c *AuthCorrelator) Observe(_ context.Context, ev events.Event) {
	auth := classifyAuthCross(ev)
	if auth == nil {
		return
	}
	day := auth.Timestamp.UTC().Format("2006-01-02")
	key := auth.User + "|" + day

	c.mu.Lock()
	defer c.mu.Unlock()
	chain, ok := c.byKey[key]
	if !ok {
		chain = &AuthChain{User: auth.User, Day: day}
		c.byKey[key] = chain
		// Cap the table to avoid unbounded growth.
		if len(c.byKey) > c.cap {
			// Drop the oldest day key.
			var oldest string
			for k := range c.byKey {
				if oldest == "" || k < oldest {
					oldest = k
				}
			}
			delete(c.byKey, oldest)
		}
	}
	chain.Events = append(chain.Events, *auth)
	if !contains(chain.Hosts, auth.HostID) {
		chain.Hosts = append(chain.Hosts, auth.HostID)
	}
	if auth.SourceIP != "" && !contains(chain.IPs, auth.SourceIP) {
		chain.IPs = append(chain.IPs, auth.SourceIP)
	}
	if !contains(chain.Protocols, auth.Protocol) {
		chain.Protocols = append(chain.Protocols, auth.Protocol)
	}
	if auth.Result == "success" {
		chain.Successes++
	} else {
		chain.Failures++
	}
}

// ChainsByUser returns every (day, chain) for a user, newest day first.
func (c *AuthCorrelator) ChainsByUser(user string) []AuthChain {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := []AuthChain{}
	for _, chain := range c.byKey {
		if chain.User != user {
			continue
		}
		cc := *chain
		out = append(out, cc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Day > out[j].Day })
	return out
}

// MultiProtocol returns chains where the user authenticated via 2+ distinct
// protocols on the same day — a strong signal of lateral movement.
func (c *AuthCorrelator) MultiProtocol(limit int) []AuthChain {
	if limit <= 0 {
		limit = 50
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := []AuthChain{}
	for _, chain := range c.byKey {
		if len(chain.Protocols) >= 2 {
			cc := *chain
			out = append(out, cc)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day > out[j].Day
		}
		return len(out[i].Hosts) > len(out[j].Hosts)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func classifyAuthCross(ev events.Event) *AuthEvent {
	cls := classify(ev) // re-use the sshd/PAM/Windows recognizer
	if cls.kind == "" {
		// Try kerberos / web / SSO heuristics.
		msg := strings.ToLower(ev.Message + " " + ev.Raw)
		switch {
		case strings.Contains(msg, "kerberos"), strings.Contains(msg, "tgt request"):
			return authFromKerberos(ev)
		case strings.Contains(msg, "saml") || strings.Contains(msg, "oidc") || strings.Contains(msg, "oauth"):
			return authFromSSO(ev)
		}
		return nil
	}
	a := &AuthEvent{
		User: cls.user, HostID: ev.HostID, SourceIP: cls.srcIP,
		Timestamp: ev.Timestamp, EventID: ev.ID,
	}
	switch cls.kind {
	case "login_success":
		a.Result = "success"
	case "login_failed":
		a.Result = "failure"
	case "logout":
		a.Result = "success"
	}
	switch {
	case strings.HasPrefix(cls.method, "ssh"):
		a.Protocol = "ssh"
	case cls.method == "rdp" || strings.Contains(cls.method, "windows"):
		a.Protocol = "rdp"
	case cls.method == "pam":
		a.Protocol = "pam"
	default:
		a.Protocol = cls.method
		if a.Protocol == "" {
			a.Protocol = "unknown"
		}
	}
	if a.User == "" {
		return nil
	}
	return a
}

func authFromKerberos(ev events.Event) *AuthEvent {
	a := &AuthEvent{HostID: ev.HostID, Protocol: "kerberos", Timestamp: ev.Timestamp, EventID: ev.ID}
	if v := fieldOr(ev.Fields, "user"); v != "" {
		a.User = v
	} else if v := fieldOr(ev.Fields, "TargetUserName"); v != "" {
		a.User = v
	}
	a.SourceIP = fieldOr(ev.Fields, "src_ip")
	if strings.Contains(strings.ToLower(ev.Message), "fail") {
		a.Result = "failure"
	} else {
		a.Result = "success"
	}
	if a.User == "" {
		return nil
	}
	return a
}

func authFromSSO(ev events.Event) *AuthEvent {
	a := &AuthEvent{HostID: ev.HostID, Protocol: "web-sso", Timestamp: ev.Timestamp, EventID: ev.ID}
	a.User = fieldOr(ev.Fields, "user")
	a.SourceIP = fieldOr(ev.Fields, "src_ip")
	if strings.Contains(strings.ToLower(ev.Message), "denied") || strings.Contains(strings.ToLower(ev.Message), "fail") {
		a.Result = "failure"
	} else {
		a.Result = "success"
	}
	if a.User == "" {
		return nil
	}
	return a
}
