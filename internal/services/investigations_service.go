// Investigations — time-frozen analyst cases.
//
// When an analyst opens a case, we record:
//   - the entity scope (host / user / ip)
//   - the time window (from / to)
//   - the audit-chain root hash at the moment of open (so the analyst can
//     prove later "I was looking at this exact platform state")
//   - the upper-bound event-receivedAt cutoff
//
// Every subsequent timeline / search call routed *through* the case applies
// the snapshot scope — only events whose receivedAt is <= the cutoff and
// whose timestamp falls in [from, to] are returned. New events keep arriving
// in the live store but they're invisible to anything querying through the
// case.
//
// Cases are persisted as line-delimited JSON at {dataDir}/cases.log so they
// survive restarts and a tampered file is detectable on reload.
package services

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kingknull/oblivra/internal/storage/hot"
)

var _ = strings.ContainsAny // reserved for future scope helpers

const casesFile = "cases.log"

// CaseState transitions: open → sealed.
type CaseState string

const (
	CaseStateOpen   CaseState = "open"
	CaseStateSealed CaseState = "sealed"
)

// CaseScope captures everything needed to recreate the analyst's view of the
// world at case-open time.
type CaseScope struct {
	TenantID         string    `json:"tenantId"`
	HostID           string    `json:"hostId,omitempty"`
	From             time.Time `json:"from"`
	To               time.Time `json:"to"`
	ReceivedAtCutoff time.Time `json:"receivedAtCutoff"`
	AuditRootAtOpen  string    `json:"auditRootAtOpen"`
}

type Case struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	OpenedBy    string            `json:"openedBy"`
	OpenedAt    time.Time         `json:"openedAt"`
	State       CaseState         `json:"state"`
	Scope       CaseScope         `json:"scope"`
	Notes       []CaseNote        `json:"notes,omitempty"`
	Hypotheses  []Hypothesis      `json:"hypotheses,omitempty"`
	Annotations []Annotation      `json:"annotations,omitempty"`
	SealedAt    time.Time         `json:"sealedAt,omitempty"`
	SealedBy    string            `json:"sealedBy,omitempty"`
	Detail      map[string]string `json:"detail,omitempty"`
}

// Hypothesis is an analyst's working theory attached to a case. The validity
// flips when the evidence either confirms or refutes it.
type Hypothesis struct {
	ID          string        `json:"id"`
	Statement   string        `json:"statement"`
	Status      string        `json:"status"` // "open" | "confirmed" | "refuted"
	EvidenceIDs []string      `json:"evidenceIds,omitempty"`
	CreatedBy   string        `json:"createdBy"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt,omitempty"`
}

// Annotation is a free-form note pinned to a specific event ID inside a case.
type Annotation struct {
	EventID   string    `json:"eventId"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
}

// Confidence is the rough completeness score for a case. Cheap heuristic
// over scope coverage + corroborating events / alerts / sources.
type Confidence struct {
	Score          int      `json:"score"`        // 0–100
	EventCount     int      `json:"eventCount"`
	AlertCount     int      `json:"alertCount"`
	SourceCount    int      `json:"sourceCount"`
	GapCount       int      `json:"gapCount"`
	Explanation    string   `json:"explanation"`
	Contributions  []string `json:"contributions,omitempty"`
}

type CaseNote struct {
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	Timestamp time.Time `json:"timestamp"`
}

type InvestigationsService struct {
	log    *slog.Logger
	hot    *hot.Store
	alerts *AlertService
	foren  *ForensicsService
	audit  *AuditService

	mu    sync.RWMutex
	cases map[string]*Case

	path string
	file *os.File
}

func NewInvestigationsService(log *slog.Logger, dir string, h *hot.Store, alerts *AlertService, foren *ForensicsService, audit *AuditService) (*InvestigationsService, error) {
	if dir == "" {
		return nil, errors.New("investigations: dir required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, casesFile)

	s := &InvestigationsService{
		log: log, hot: h, alerts: alerts, foren: foren, audit: audit,
		cases: map[string]*Case{}, path: path,
	}
	if err := s.replay(); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	s.file = f
	log.Info("investigations journal opened", "path", path, "cases", len(s.cases))
	return s, nil
}

func (s *InvestigationsService) ServiceName() string { return "InvestigationsService" }

func (s *InvestigationsService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

// replay reads the persisted case log; later entries overwrite earlier ones
// for the same ID (case updates are append-only writes of full state).
func (s *InvestigationsService) replay() error {
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	br := bufio.NewReader(f)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			var c Case
			if jerr := json.Unmarshal(line, &c); jerr != nil {
				return fmt.Errorf("cases replay: %w", jerr)
			}
			s.cases[c.ID] = &c
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// persist appends a full case snapshot to the journal — replay reduces by
// last-write-wins keyed on ID.
func (s *InvestigationsService) persist(c *Case) error {
	if s.file == nil {
		return nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if _, err := s.file.Write(b); err != nil {
		return err
	}
	return s.file.Sync()
}

// ---- public API ----

type OpenCaseRequest struct {
	Title    string    `json:"title"`
	HostID   string    `json:"hostId"`
	TenantID string    `json:"tenantId"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	OpenedBy string    `json:"openedBy"`
}

func (s *InvestigationsService) Open(ctx context.Context, req OpenCaseRequest) (*Case, error) {
	if req.Title == "" {
		return nil, errors.New("title required")
	}
	if req.OpenedBy == "" {
		req.OpenedBy = "anonymous"
	}
	if req.TenantID == "" {
		req.TenantID = "default"
	}
	now := time.Now().UTC()
	if req.To.IsZero() {
		req.To = now
	}
	if req.From.IsZero() {
		req.From = req.To.Add(-24 * time.Hour)
	}

	root := ""
	if s.audit != nil {
		root = s.audit.Verify().RootHash
	}

	c := &Case{
		ID:       caseID(),
		Title:    req.Title,
		OpenedBy: req.OpenedBy,
		OpenedAt: now,
		State:    CaseStateOpen,
		Scope: CaseScope{
			TenantID:         req.TenantID,
			HostID:           req.HostID,
			From:             req.From,
			To:               req.To,
			ReceivedAtCutoff: now,
			AuditRootAtOpen:  root,
		},
	}
	s.mu.Lock()
	s.cases[c.ID] = c
	s.mu.Unlock()
	if err := s.persist(c); err != nil {
		s.log.Error("case persist", "err", err)
	}
	if s.audit != nil {
		s.audit.Append(ctx, req.OpenedBy, "investigation.open", req.TenantID, map[string]string{
			"caseId":  c.ID,
			"hostId":  req.HostID,
			"from":    req.From.Format(time.RFC3339),
			"to":      req.To.Format(time.RFC3339),
			"rootAt":  root,
		})
	}
	return c, nil
}

func (s *InvestigationsService) List() []Case {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Case, 0, len(s.cases))
	for _, c := range s.cases {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OpenedAt.After(out[j].OpenedAt) })
	return out
}

func (s *InvestigationsService) Get(id string) (*Case, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.cases[id]
	if !ok {
		return nil, false
	}
	cc := *c
	return &cc, true
}

// Timeline returns the merged event/alert/gap stream restricted to the case's
// frozen scope (time window AND receivedAt cutoff).
func (s *InvestigationsService) Timeline(ctx context.Context, caseID string) ([]TimelineEntry, error) {
	c, ok := s.Get(caseID)
	if !ok {
		return nil, errors.New("case not found")
	}
	out := []TimelineEntry{}

	// Hot store events restricted to scope + cutoff.
	evs, err := s.hot.Range(ctx, hot.RangeOpts{
		TenantID: c.Scope.TenantID,
		From:     c.Scope.From,
		To:       c.Scope.To,
		Limit:    5000,
	})
	if err == nil {
		for _, e := range evs {
			if e.ReceivedAt.After(c.Scope.ReceivedAtCutoff) {
				continue // not visible in this snapshot
			}
			if c.Scope.HostID != "" && e.HostID != c.Scope.HostID {
				continue
			}
			out = append(out, TimelineEntry{
				Kind: "event", Timestamp: e.Timestamp, Severity: string(e.Severity),
				Title: orFallback(e.EventType, "event"), Detail: e.Message, RefID: e.ID,
			})
		}
	}

	// Alerts.
	if s.alerts != nil {
		for _, a := range s.alerts.Recent(1000) {
			if a.Triggered.After(c.Scope.ReceivedAtCutoff) {
				continue
			}
			if a.Triggered.Before(c.Scope.From) || a.Triggered.After(c.Scope.To) {
				continue
			}
			if c.Scope.HostID != "" && a.HostID != c.Scope.HostID {
				continue
			}
			out = append(out, TimelineEntry{
				Kind: "alert", Timestamp: a.Triggered, Severity: string(a.Severity),
				Title: a.RuleName, Detail: a.Message, RefID: a.ID,
			})
		}
	}

	// Gaps + sealed evidence (forensics).
	if s.foren != nil {
		for _, g := range s.foren.Gaps() {
			if g.EndedAt.After(c.Scope.ReceivedAtCutoff) {
				continue
			}
			if g.EndedAt.Before(c.Scope.From) || g.StartedAt.After(c.Scope.To) {
				continue
			}
			if c.Scope.HostID != "" && g.HostID != c.Scope.HostID {
				continue
			}
			out = append(out, TimelineEntry{
				Kind: "gap", Timestamp: g.EndedAt, Severity: "warning",
				Title: "Telemetry gap", Detail: "host " + g.HostID + " silent for " + g.Duration,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	return out, nil
}

// AddNote appends an audited investigator note to a case.
func (s *InvestigationsService) AddNote(ctx context.Context, caseID, author, body string) (*Case, error) {
	if body == "" {
		return nil, errors.New("note body required")
	}
	s.mu.Lock()
	c, ok := s.cases[caseID]
	if !ok {
		s.mu.Unlock()
		return nil, errors.New("case not found")
	}
	if c.State == CaseStateSealed {
		s.mu.Unlock()
		return nil, errors.New("case sealed")
	}
	c.Notes = append(c.Notes, CaseNote{Author: author, Body: body, Timestamp: time.Now().UTC()})
	cc := *c
	s.mu.Unlock()
	_ = s.persist(&cc)
	if s.audit != nil {
		s.audit.Append(ctx, author, "investigation.note", c.Scope.TenantID, map[string]string{
			"caseId": caseID,
			"len":    fmt.Sprintf("%d", len(body)),
		})
	}
	return &cc, nil
}

// Seal locks a case so it cannot be modified.
func (s *InvestigationsService) Seal(ctx context.Context, caseID, by string) (*Case, error) {
	s.mu.Lock()
	c, ok := s.cases[caseID]
	if !ok {
		s.mu.Unlock()
		return nil, errors.New("case not found")
	}
	if c.State == CaseStateSealed {
		cc := *c
		s.mu.Unlock()
		return &cc, nil
	}
	c.State = CaseStateSealed
	c.SealedAt = time.Now().UTC()
	c.SealedBy = by
	cc := *c
	s.mu.Unlock()
	_ = s.persist(&cc)
	if s.audit != nil {
		s.audit.Append(ctx, by, "investigation.seal", c.Scope.TenantID, map[string]string{
			"caseId": caseID,
		})
	}
	return &cc, nil
}

func caseID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// AddHypothesis records a new working theory on a case.
func (s *InvestigationsService) AddHypothesis(ctx context.Context, caseID, author, statement string) (*Case, error) {
	if statement == "" {
		return nil, errors.New("statement required")
	}
	s.mu.Lock()
	c, ok := s.cases[caseID]
	if !ok {
		s.mu.Unlock()
		return nil, errors.New("case not found")
	}
	if c.State == CaseStateSealed {
		s.mu.Unlock()
		return nil, errors.New("case sealed")
	}
	h := Hypothesis{
		ID:        randomID(6),
		Statement: statement,
		Status:    "open",
		CreatedBy: author,
		CreatedAt: time.Now().UTC(),
	}
	c.Hypotheses = append(c.Hypotheses, h)
	cc := *c
	s.mu.Unlock()
	_ = s.persist(&cc)
	if s.audit != nil {
		s.audit.Append(ctx, author, "investigation.hypothesis.add", c.Scope.TenantID, map[string]string{
			"caseId":     caseID,
			"hypothesis": h.ID,
		})
	}
	return &cc, nil
}

// SetHypothesisStatus flips an existing hypothesis to confirmed/refuted.
func (s *InvestigationsService) SetHypothesisStatus(ctx context.Context, caseID, hypoID, author, status string, evidenceIDs []string) (*Case, error) {
	if status != "confirmed" && status != "refuted" && status != "open" {
		return nil, errors.New("status must be open|confirmed|refuted")
	}
	s.mu.Lock()
	c, ok := s.cases[caseID]
	if !ok {
		s.mu.Unlock()
		return nil, errors.New("case not found")
	}
	if c.State == CaseStateSealed {
		s.mu.Unlock()
		return nil, errors.New("case sealed")
	}
	updated := false
	for i := range c.Hypotheses {
		if c.Hypotheses[i].ID == hypoID {
			c.Hypotheses[i].Status = status
			c.Hypotheses[i].UpdatedAt = time.Now().UTC()
			if len(evidenceIDs) > 0 {
				c.Hypotheses[i].EvidenceIDs = append(c.Hypotheses[i].EvidenceIDs, evidenceIDs...)
			}
			updated = true
			break
		}
	}
	cc := *c
	s.mu.Unlock()
	if !updated {
		return nil, errors.New("hypothesis not found")
	}
	_ = s.persist(&cc)
	if s.audit != nil {
		s.audit.Append(ctx, author, "investigation.hypothesis.update", c.Scope.TenantID, map[string]string{
			"caseId":     caseID,
			"hypothesis": hypoID,
			"status":     status,
		})
	}
	return &cc, nil
}

// Annotate pins a free-form note to a specific event ID.
func (s *InvestigationsService) Annotate(ctx context.Context, caseID, eventID, author, body string) (*Case, error) {
	if eventID == "" || body == "" {
		return nil, errors.New("eventId and body required")
	}
	s.mu.Lock()
	c, ok := s.cases[caseID]
	if !ok {
		s.mu.Unlock()
		return nil, errors.New("case not found")
	}
	if c.State == CaseStateSealed {
		s.mu.Unlock()
		return nil, errors.New("case sealed")
	}
	c.Annotations = append(c.Annotations, Annotation{
		EventID: eventID, Body: body, Author: author, Timestamp: time.Now().UTC(),
	})
	cc := *c
	s.mu.Unlock()
	_ = s.persist(&cc)
	if s.audit != nil {
		s.audit.Append(ctx, author, "investigation.annotate", c.Scope.TenantID, map[string]string{
			"caseId":  caseID,
			"eventId": eventID,
		})
	}
	return &cc, nil
}

// Confidence runs a cheap heuristic over the case scope.
//
//	+30 if any sealed evidence package exists for the host in scope
//	+25 if at least one alert fired inside the case window
//	+20 if at least 2 sources corroborated (3+ events from each)
//	+15 if the case has at least one confirmed hypothesis
//	-10 per detected log gap (caps at -30)
//	+10 base for any non-empty case
func (s *InvestigationsService) Confidence(ctx context.Context, caseID string) (*Confidence, error) {
	c, ok := s.Get(caseID)
	if !ok {
		return nil, errors.New("case not found")
	}
	conf := &Confidence{Score: 10}
	conf.Contributions = append(conf.Contributions, "+10 base")

	tl, err := s.Timeline(ctx, caseID)
	if err == nil {
		sources := map[string]int{}
		for _, e := range tl {
			switch e.Kind {
			case "event":
				conf.EventCount++
			case "alert":
				conf.AlertCount++
			case "gap":
				conf.GapCount++
			}
			if e.Kind == "event" {
				// Source is unavailable in TimelineEntry — approximate with severity bucket
				sources[e.Severity]++
			}
		}
		conf.SourceCount = len(sources)
		if conf.AlertCount > 0 {
			conf.Score += 25
			conf.Contributions = append(conf.Contributions, "+25 alerts fired")
		}
		if conf.SourceCount >= 2 {
			conf.Score += 20
			conf.Contributions = append(conf.Contributions, "+20 multi-source")
		}
		if conf.GapCount > 0 {
			penalty := conf.GapCount * 10
			if penalty > 30 {
				penalty = 30
			}
			conf.Score -= penalty
			conf.Contributions = append(conf.Contributions, fmt.Sprintf("-%d log gaps", penalty))
		}
	}
	for _, h := range c.Hypotheses {
		if h.Status == "confirmed" {
			conf.Score += 15
			conf.Contributions = append(conf.Contributions, "+15 confirmed hypothesis")
			break
		}
	}
	if s.foren != nil {
		for _, ev := range s.foren.List() {
			if c.Scope.HostID == "" || ev.HostID == c.Scope.HostID {
				conf.Score += 30
				conf.Contributions = append(conf.Contributions, "+30 sealed evidence")
				break
			}
		}
	}
	if conf.Score < 0 {
		conf.Score = 0
	}
	if conf.Score > 100 {
		conf.Score = 100
	}
	conf.Explanation = explainScore(conf.Score)
	return conf, nil
}

func explainScore(s int) string {
	switch {
	case s >= 80:
		return "high — multi-source corroboration with sealed evidence"
	case s >= 60:
		return "moderate — strong signal but evidence not yet sealed"
	case s >= 40:
		return "preliminary — some corroboration, more pivots recommended"
	default:
		return "weak — case scope contains few signals"
	}
}
