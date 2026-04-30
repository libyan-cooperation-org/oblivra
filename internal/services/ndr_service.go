package services

import (
	"log/slog"
	"sort"
	"sync"
	"time"
)

// NetFlowRecord is a minimal flow record. Real NetFlow/IPFIX packet decode is
// deferred — for now agents and integrations push flows here as JSON.
type NetFlowRecord struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	SrcIP     string    `json:"srcIp"`
	DstIP     string    `json:"dstIp"`
	SrcPort   int       `json:"srcPort"`
	DstPort   int       `json:"dstPort"`
	Protocol  string    `json:"protocol"`
	Bytes     int64     `json:"bytes"`
	Packets   int64     `json:"packets"`
}

type NdrPair struct {
	Pair  string `json:"pair"`
	Bytes int64  `json:"bytes"`
	Flows int64  `json:"flows"`
}

type NdrService struct {
	log *slog.Logger
	mu  sync.RWMutex
	all []NetFlowRecord
	cap int
}

func NewNdrService(log *slog.Logger) *NdrService {
	return &NdrService{log: log, cap: 5000}
}

func (s *NdrService) ServiceName() string { return "NdrService" }

func (s *NdrService) Record(r NetFlowRecord) {
	s.mu.Lock()
	s.all = append(s.all, r)
	if len(s.all) > s.cap {
		s.all = s.all[len(s.all)-s.cap:]
	}
	s.mu.Unlock()
}

func (s *NdrService) Recent(limit int) []NetFlowRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.all)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]NetFlowRecord, 0, limit)
	for i := n - 1; i >= n-limit; i-- {
		out = append(out, s.all[i])
	}
	return out
}

// TopTalkers aggregates flows by src→dst pair and returns the heaviest.
func (s *NdrService) TopTalkers(limit int) []NdrPair {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agg := map[string]*NdrPair{}
	for _, f := range s.all {
		key := f.SrcIP + "→" + f.DstIP
		p, ok := agg[key]
		if !ok {
			p = &NdrPair{Pair: key}
			agg[key] = p
		}
		p.Bytes += f.Bytes
		p.Flows++
	}
	out := make([]NdrPair, 0, len(agg))
	for _, v := range agg {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
