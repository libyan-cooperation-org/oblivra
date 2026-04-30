package services

import (
	"context"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/events"
)

// UebaProfile keeps a rolling event-rate baseline per entity (host or user).
// We track events-per-minute, store a running mean + stddev over the last 60
// minutes, and flag any minute that exceeds mean+3σ.
type UebaProfile struct {
	Entity      string    `json:"entity"`
	Tenant      string    `json:"tenant"`
	WindowsSeen int64     `json:"windowsSeen"`
	Mean        float64   `json:"mean"`
	StdDev      float64   `json:"stdDev"`
	LastEPM     int       `json:"lastEpm"`
	LastSpike   float64   `json:"lastSpike"` // z-score of most recent window
	UpdatedAt   time.Time `json:"updatedAt"`

	values []float64
	bucket time.Time
	count  int
}

type UebaAnomaly struct {
	Entity     string    `json:"entity"`
	Tenant     string    `json:"tenant"`
	EPM        int       `json:"epm"`
	Mean       float64   `json:"mean"`
	StdDev     float64   `json:"stdDev"`
	ZScore     float64   `json:"zScore"`
	DetectedAt time.Time `json:"detectedAt"`
}

type UebaService struct {
	log    *slog.Logger
	alerts *AlertService

	mu        sync.Mutex
	profiles  map[string]*UebaProfile
	anomalies []UebaAnomaly
}

func NewUebaService(log *slog.Logger, alerts *AlertService) *UebaService {
	return &UebaService{log: log, alerts: alerts, profiles: map[string]*UebaProfile{}}
}

func (s *UebaService) ServiceName() string { return "UebaService" }

// Observe is called from the ingest fan-out for every event.
func (s *UebaService) Observe(ctx context.Context, ev events.Event) {
	entity := ev.HostID
	if entity == "" {
		return
	}
	now := ev.Timestamp.Truncate(time.Minute)
	key := ev.TenantID + ":" + entity

	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.profiles[key]
	if !ok {
		p = &UebaProfile{Entity: entity, Tenant: ev.TenantID, bucket: now}
		s.profiles[key] = p
	}
	if p.bucket.IsZero() {
		p.bucket = now
	}
	if !now.Equal(p.bucket) {
		s.closeBucket(ctx, p, p.count)
		p.bucket = now
		p.count = 0
	}
	p.count++
	p.UpdatedAt = ev.Timestamp
}

func (s *UebaService) closeBucket(ctx context.Context, p *UebaProfile, count int) {
	p.LastEPM = count
	p.WindowsSeen++
	if p.WindowsSeen <= 3 {
		p.values = append(p.values, float64(count))
		p.Mean, p.StdDev = stats(p.values)
		return
	}
	mean, sd := p.Mean, p.StdDev
	z := 0.0
	if sd > 0 {
		z = (float64(count) - mean) / sd
	}
	p.LastSpike = z
	if z >= 3 {
		anom := UebaAnomaly{
			Entity: p.Entity, Tenant: p.Tenant,
			EPM: count, Mean: mean, StdDev: sd, ZScore: z,
			DetectedAt: time.Now().UTC(),
		}
		s.anomalies = append(s.anomalies, anom)
		if len(s.anomalies) > 1000 {
			s.anomalies = s.anomalies[len(s.anomalies)-1000:]
		}
		if s.alerts != nil {
			s.alerts.Raise(ctx, Alert{
				TenantID: p.Tenant,
				RuleID:   "ueba-rate-anomaly",
				RuleName: "UEBA event-rate anomaly",
				Severity: AlertSeverityMedium,
				HostID:   p.Entity,
				Message:  "event-rate spike (z=" + strconv.FormatFloat(z, 'f', 2, 64) + ")",
				MITRE:    []string{"behavioral.anomaly"},
			})
		}
	}
	p.values = append(p.values, float64(count))
	if len(p.values) > 60 {
		p.values = p.values[len(p.values)-60:]
	}
	p.Mean, p.StdDev = stats(p.values)
}

func (s *UebaService) Profiles() []UebaProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]UebaProfile, 0, len(s.profiles))
	for _, p := range s.profiles {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSpike > out[j].LastSpike })
	return out
}

func (s *UebaService) Anomalies() []UebaAnomaly {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]UebaAnomaly, len(s.anomalies))
	copy(out, s.anomalies)
	return out
}

func stats(vs []float64) (mean, sd float64) {
	if len(vs) == 0 {
		return 0, 0
	}
	for _, v := range vs {
		mean += v
	}
	mean /= float64(len(vs))
	for _, v := range vs {
		d := v - mean
		sd += d * d
	}
	sd = math.Sqrt(sd / float64(len(vs)))
	return
}
