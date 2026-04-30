package reconstruction

import (
	"context"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// EntityProfile rolls up everything we know about a single host / user / IP
// into one analyst-facing summary. Phase 39 calls this the "entity forensic
// profile" — it's what the analyst opens first when an alert names a host.
type EntityProfile struct {
	Kind          string            `json:"kind"` // host | user | ip
	ID            string            `json:"id"`
	FirstSeen     time.Time         `json:"firstSeen"`
	LastSeen      time.Time         `json:"lastSeen"`
	Events        int64             `json:"events"`
	Sources       []string          `json:"sources"`
	TopEventTypes []KV              `json:"topEventTypes"`
	TopFields     map[string][]KV   `json:"topFields,omitempty"`
	RelatedHosts  []string          `json:"relatedHosts,omitempty"`
	RelatedUsers  []string          `json:"relatedUsers,omitempty"`
}

type KV struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type entityState struct {
	first       time.Time
	last        time.Time
	events      int64
	sources     map[string]struct{}
	eventTypes  map[string]int64
	fieldHits   map[string]map[string]int64 // field name → value → count
	related     map[string]struct{}
}

type EntityIndex struct {
	mu       sync.RWMutex
	hosts    map[string]*entityState
	users    map[string]*entityState
	ips      map[string]*entityState
}

func NewEntityIndex() *EntityIndex {
	return &EntityIndex{
		hosts: map[string]*entityState{},
		users: map[string]*entityState{},
		ips:   map[string]*entityState{},
	}
}

// Observe is called per-event from the bus fan-out.
func (e *EntityIndex) Observe(_ context.Context, ev events.Event) {
	host := ev.HostID
	user := pickUser(ev)
	ip := pickIP(ev)

	e.mu.Lock()
	defer e.mu.Unlock()
	if host != "" {
		s := e.upsert(e.hosts, host)
		bumpAll(s, ev)
		if user != "" {
			s.related[user] = struct{}{}
		}
	}
	if user != "" {
		s := e.upsert(e.users, user)
		bumpAll(s, ev)
		if host != "" {
			s.related[host] = struct{}{}
		}
	}
	if ip != "" && net.ParseIP(ip) != nil {
		s := e.upsert(e.ips, ip)
		bumpAll(s, ev)
		if host != "" {
			s.related[host] = struct{}{}
		}
	}
}

func (e *EntityIndex) upsert(m map[string]*entityState, id string) *entityState {
	s, ok := m[id]
	if !ok {
		s = &entityState{
			sources:    map[string]struct{}{},
			eventTypes: map[string]int64{},
			fieldHits:  map[string]map[string]int64{},
			related:    map[string]struct{}{},
		}
		m[id] = s
	}
	return s
}

func bumpAll(s *entityState, ev events.Event) {
	s.events++
	if s.first.IsZero() || ev.Timestamp.Before(s.first) {
		s.first = ev.Timestamp
	}
	if ev.Timestamp.After(s.last) {
		s.last = ev.Timestamp
	}
	s.sources[string(ev.Source)] = struct{}{}
	if ev.EventType != "" {
		s.eventTypes[ev.EventType]++
	}
	for k, v := range ev.Fields {
		if _, ok := s.fieldHits[k]; !ok {
			s.fieldHits[k] = map[string]int64{}
		}
		s.fieldHits[k][v]++
	}
}

// Profile returns a snapshot for a (kind, id) pair.
func (e *EntityIndex) Profile(kind, id string) *EntityProfile {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var s *entityState
	switch kind {
	case "host":
		s = e.hosts[id]
	case "user":
		s = e.users[id]
	case "ip":
		s = e.ips[id]
	default:
		return nil
	}
	if s == nil {
		return nil
	}
	p := &EntityProfile{
		Kind: kind, ID: id, FirstSeen: s.first, LastSeen: s.last, Events: s.events,
		TopFields: map[string][]KV{},
	}
	for src := range s.sources {
		p.Sources = append(p.Sources, src)
	}
	sort.Strings(p.Sources)
	p.TopEventTypes = topKV(s.eventTypes, 5)
	for f, vals := range s.fieldHits {
		p.TopFields[f] = topKV(vals, 5)
	}
	for r := range s.related {
		switch kind {
		case "host":
			p.RelatedUsers = append(p.RelatedUsers, r)
		case "user", "ip":
			p.RelatedHosts = append(p.RelatedHosts, r)
		}
	}
	sort.Strings(p.RelatedHosts)
	sort.Strings(p.RelatedUsers)
	return p
}

// List returns up-to-N profiles per kind, newest-first.
func (e *EntityIndex) List(kind string, limit int) []EntityProfile {
	if limit <= 0 {
		limit = 100
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	var src map[string]*entityState
	switch kind {
	case "host":
		src = e.hosts
	case "user":
		src = e.users
	case "ip":
		src = e.ips
	default:
		return nil
	}
	out := make([]EntityProfile, 0, len(src))
	for id, s := range src {
		out = append(out, EntityProfile{
			Kind: kind, ID: id, FirstSeen: s.first, LastSeen: s.last, Events: s.events,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func topKV(m map[string]int64, n int) []KV {
	out := make([]KV, 0, len(m))
	for k, v := range m {
		out = append(out, KV{Key: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func pickUser(ev events.Event) string {
	for _, k := range []string{"user", "userId", "username", "TargetUserName", "src_user"} {
		if v, ok := ev.Fields[k]; ok && v != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func pickIP(ev events.Event) string {
	for _, k := range []string{"src_ip", "src", "srcIp", "client_ip", "IpAddress", "ip"} {
		if v, ok := ev.Fields[k]; ok && v != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
