package reconstruction

import (
	"context"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// NetworkStitcher correlates DNS lookups + connection events + NetFlow rows
// into 5-tuple stories so an analyst can answer "who did host X talk to and
// what name did they resolve first?" without manually pivoting between
// three views.
type NetworkStitcher struct {
	mu      sync.RWMutex
	flows   map[string]*Flow // 5-tuple-key → flow
	dns     map[string][]DNSAnswer
	timeout time.Duration
}

func NewNetworkStitcher() *NetworkStitcher {
	return &NetworkStitcher{
		flows:   map[string]*Flow{},
		dns:     map[string][]DNSAnswer{},
		timeout: 30 * time.Minute,
	}
}

type Flow struct {
	Key        string    `json:"key"`
	HostID     string    `json:"hostId,omitempty"`
	SrcIP      string    `json:"srcIp"`
	DstIP      string    `json:"dstIp"`
	SrcPort    int       `json:"srcPort,omitempty"`
	DstPort    int       `json:"dstPort,omitempty"`
	Protocol   string    `json:"protocol,omitempty"`
	FirstSeen  time.Time `json:"firstSeen"`
	LastSeen   time.Time `json:"lastSeen"`
	Packets    int64     `json:"packets,omitempty"`
	Bytes      int64     `json:"bytes,omitempty"`
	EventIDs   []string  `json:"eventIds,omitempty"`
	ResolvedAs []string  `json:"resolvedAs,omitempty"` // DNS names that resolved to DstIP
}

type DNSAnswer struct {
	Query    string    `json:"query"`
	IP       string    `json:"ip"`
	HostID   string    `json:"hostId,omitempty"`
	Observed time.Time `json:"observed"`
	EventID  string    `json:"eventId"`
}

// Observe is called per-event from the bus fan-out.
func (n *NetworkStitcher) Observe(_ context.Context, ev events.Event) {
	if name, ip, ok := classifyDNS(ev); ok {
		n.recordDNS(DNSAnswer{Query: name, IP: ip, HostID: ev.HostID, Observed: ev.Timestamp, EventID: ev.ID})
		return
	}
	if f, ok := classifyFlow(ev); ok {
		n.recordFlow(f, ev.ID)
	}
}

// RecordFlow is the explicit entry point for NetFlow records that arrive on
// the dedicated NDR path. The platform stack calls this directly so we don't
// duplicate work classifying the same record twice.
func (n *NetworkStitcher) RecordFlow(srcIP, dstIP string, srcPort, dstPort int, proto string, when time.Time, bytes, packets int64, hostID, evID string) {
	f := Flow{
		HostID: hostID, SrcIP: srcIP, DstIP: dstIP,
		SrcPort: srcPort, DstPort: dstPort, Protocol: proto,
		FirstSeen: when, LastSeen: when,
		Bytes: bytes, Packets: packets,
	}
	n.recordFlow(f, evID)
}

func (n *NetworkStitcher) recordDNS(a DNSAnswer) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.dns[a.IP] = append(n.dns[a.IP], a)
	n.gcLocked()
	// Backfill any flows that already used this IP.
	for _, f := range n.flows {
		if f.DstIP == a.IP && !contains(f.ResolvedAs, a.Query) {
			f.ResolvedAs = append(f.ResolvedAs, a.Query)
		}
	}
}

func (n *NetworkStitcher) recordFlow(f Flow, evID string) {
	key := flowKey(f.SrcIP, f.DstIP, f.SrcPort, f.DstPort, f.Protocol)
	f.Key = key
	n.mu.Lock()
	defer n.mu.Unlock()
	existing, ok := n.flows[key]
	if !ok {
		f.EventIDs = []string{evID}
		// Stitch any DNS answer we already saw.
		for _, ans := range n.dns[f.DstIP] {
			if !contains(f.ResolvedAs, ans.Query) {
				f.ResolvedAs = append(f.ResolvedAs, ans.Query)
			}
		}
		n.flows[key] = &f
		n.gcLocked()
		return
	}
	if f.FirstSeen.Before(existing.FirstSeen) {
		existing.FirstSeen = f.FirstSeen
	}
	if f.LastSeen.After(existing.LastSeen) {
		existing.LastSeen = f.LastSeen
	}
	existing.Bytes += f.Bytes
	existing.Packets += f.Packets
	if !contains(existing.EventIDs, evID) {
		existing.EventIDs = append(existing.EventIDs, evID)
	}
	n.gcLocked()
}

// gcLocked drops anything older than the timeout window. Caller holds n.mu.
func (n *NetworkStitcher) gcLocked() {
	if n.timeout <= 0 {
		return
	}
	cut := time.Now().Add(-n.timeout)
	for k, f := range n.flows {
		if f.LastSeen.Before(cut) {
			delete(n.flows, k)
		}
	}
	for ip, answers := range n.dns {
		var keep []DNSAnswer
		for _, a := range answers {
			if a.Observed.After(cut) {
				keep = append(keep, a)
			}
		}
		if len(keep) == 0 {
			delete(n.dns, ip)
		} else {
			n.dns[ip] = keep
		}
	}
}

// FlowsByHost returns all flows the stitcher remembers, optionally filtered
// to one host. Newest-first.
func (n *NetworkStitcher) FlowsByHost(host string) []Flow {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make([]Flow, 0, len(n.flows))
	for _, f := range n.flows {
		if host != "" && f.HostID != host && f.SrcIP != host && f.DstIP != host {
			continue
		}
		out = append(out, *f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	return out
}

// DNSByQuery returns the DNS answers we've seen for a given hostname.
func (n *NetworkStitcher) DNSByQuery(query string) []DNSAnswer {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := []DNSAnswer{}
	for _, list := range n.dns {
		for _, a := range list {
			if a.Query == query {
				out = append(out, a)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Observed.After(out[j].Observed) })
	return out
}

// ---- classification ----

var (
	rxDNSQueryAns = regexp.MustCompile(`(?i)(?:DNS|query)\s+(\S+)\s+(?:resolved|->|=>|=)\s*(\d+\.\d+\.\d+\.\d+)`)
	rxConnLine    = regexp.MustCompile(`(?i)\bsrc=(\d+\.\d+\.\d+\.\d+).*?\bdst=(\d+\.\d+\.\d+\.\d+).*?\bproto=(\w+)`)
)

func classifyDNS(ev events.Event) (name, ip string, ok bool) {
	// 1) field-level (CEF / structured)
	if q, ok1 := ev.Fields["dns_query"]; ok1 {
		if a, ok2 := ev.Fields["dns_answer"]; ok2 {
			return q, a, true
		}
	}
	// 2) message regex
	src := ev.Message + " " + ev.Raw
	if m := rxDNSQueryAns.FindStringSubmatch(src); len(m) == 3 {
		return m[1], m[2], true
	}
	return "", "", false
}

func classifyFlow(ev events.Event) (Flow, bool) {
	src := ev.Message + " " + ev.Raw
	if m := rxConnLine.FindStringSubmatch(src); len(m) == 4 {
		return Flow{
			HostID: ev.HostID, SrcIP: m[1], DstIP: m[2], Protocol: m[3],
			FirstSeen: ev.Timestamp, LastSeen: ev.Timestamp,
		}, true
	}
	if s := ev.Fields["src"]; s != "" {
		if d := ev.Fields["dst"]; d != "" {
			f := Flow{
				HostID: ev.HostID, SrcIP: s, DstIP: d,
				FirstSeen: ev.Timestamp, LastSeen: ev.Timestamp,
				Protocol: ev.Fields["proto"],
			}
			return f, true
		}
	}
	return Flow{}, false
}

func flowKey(src, dst string, sp, dp int, proto string) string {
	return src + ":" + itoa(sp) + "→" + dst + ":" + itoa(dp) + "/" + proto
}

func contains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return ""
	}
	// short, dependency-free
	buf := [12]byte{}
	i := len(buf)
	if n < 0 {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
