// Forensics service — log-derived evidence + log-gap detection.
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
	"github.com/kingknull/oblivra/internal/storage/hot"
)

type EvidenceItem struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	HostID    string    `json:"hostId"`
	Title     string    `json:"title"`
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	EventIDs  []string  `json:"eventIds"`
	Hash      string    `json:"hash"`
	SealedAt  time.Time `json:"sealedAt"`
	Sealed    bool      `json:"sealed"`
}

type LogGap struct {
	HostID    string    `json:"hostId"`
	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt"`
	Duration  string    `json:"duration"`
}

type ForensicsService struct {
	hot   *hot.Store
	audit *AuditService

	mu       sync.RWMutex
	items    map[string]*EvidenceItem
	lastSeen map[string]time.Time // host → last event seen
	gaps     []LogGap
}

func NewForensicsService(hot *hot.Store, audit *AuditService) *ForensicsService {
	return &ForensicsService{
		hot:      hot,
		audit:    audit,
		items:    map[string]*EvidenceItem{},
		lastSeen: map[string]time.Time{},
	}
}

func (s *ForensicsService) ServiceName() string { return "ForensicsService" }

// Observe is called per-event so we can track gaps in host telemetry.
const gapThreshold = 5 * time.Minute

func (s *ForensicsService) Observe(ev events.Event) {
	if ev.HostID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, ok := s.lastSeen[ev.HostID]
	if ok && ev.Timestamp.Sub(prev) > gapThreshold {
		gap := LogGap{
			HostID:    ev.HostID,
			StartedAt: prev,
			EndedAt:   ev.Timestamp,
			Duration:  ev.Timestamp.Sub(prev).Round(time.Second).String(),
		}
		s.gaps = append(s.gaps, gap)
		if len(s.gaps) > 1000 {
			s.gaps = s.gaps[len(s.gaps)-1000:]
		}
	}
	s.lastSeen[ev.HostID] = ev.Timestamp
}

func (s *ForensicsService) Gaps() []LogGap {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]LogGap, len(s.gaps))
	copy(out, s.gaps)
	return out
}

// CollectByHost gathers events for a given host between [from, to] and seals
// them into an evidence package.
func (s *ForensicsService) CollectByHost(ctx context.Context, tenantID, hostID, title string, from, to time.Time) (*EvidenceItem, error) {
	evs, err := s.hot.Range(ctx, hot.RangeOpts{
		TenantID: tenantID,
		From:     from,
		To:       to,
		Limit:    10000,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(evs))
	for _, e := range evs {
		if e.HostID != hostID {
			continue
		}
		ids = append(ids, e.ID)
	}
	sort.Strings(ids)
	hash := hashIDs(ids)
	item := &EvidenceItem{
		ID:       randomID(8),
		TenantID: tenantID,
		HostID:   hostID,
		Title:    title,
		From:     from,
		To:       to,
		EventIDs: ids,
		Hash:     hash,
		SealedAt: time.Now().UTC(),
		Sealed:   true,
	}
	s.mu.Lock()
	s.items[item.ID] = item
	s.mu.Unlock()
	if s.audit != nil {
		_ = s.audit.Append(ctx, "system", "evidence.seal", tenantID, map[string]string{
			"id":     item.ID,
			"hostId": hostID,
			"title":  title,
			"hash":   hash,
			"events": strconv.Itoa(len(ids)),
		})
	}
	return item, nil
}

func (s *ForensicsService) List() []EvidenceItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]EvidenceItem, 0, len(s.items))
	for _, v := range s.items {
		out = append(out, *v)
	}
	return out
}

func hashIDs(ids []string) string {
	b, _ := json.Marshal(ids)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
