package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/kingknull/oblivra/internal/report"
)

type ReportService struct {
	log   *slog.Logger
	cases *InvestigationsService
	audit *AuditService
}

func NewReportService(log *slog.Logger, cases *InvestigationsService, audit *AuditService) *ReportService {
	return &ReportService{log: log, cases: cases, audit: audit}
}

func (s *ReportService) ServiceName() string { return "ReportService" }

// CaseHTML renders a sealed-or-open case to a self-contained HTML evidence
// package. The caller decides what to do with the bytes.
func (s *ReportService) CaseHTML(ctx context.Context, caseID string) ([]byte, error) {
	c, ok := s.cases.Get(caseID)
	if !ok {
		return nil, errors.New("case not found")
	}
	tl, err := s.cases.Timeline(ctx, caseID)
	if err != nil {
		return nil, err
	}
	conf, _ := s.cases.Confidence(ctx, caseID)
	root := ""
	if s.audit != nil {
		root = s.audit.Verify().RootHash
	}

	pkg := report.Package{
		Case: report.CaseInfo{
			ID:              c.ID,
			Title:           c.Title,
			OpenedBy:        c.OpenedBy,
			OpenedAt:        c.OpenedAt,
			State:           string(c.State),
			HostID:          c.Scope.HostID,
			From:            c.Scope.From,
			To:              c.Scope.To,
			AuditRootAtOpen: c.Scope.AuditRootAtOpen,
			SealedBy:        c.SealedBy,
			SealedAt:        c.SealedAt,
			Hypotheses:      mapHypotheses(c.Hypotheses),
			Annotations:     mapAnnotations(c.Annotations),
		},
		Timeline:    mapTimeline(tl),
		Confidence:  mapConfidence(conf),
		AuditRoot:   root,
		GeneratedAt: time.Now().UTC(),
		Verifier:    "oblivra-verify --hmac $OBLIVRA_AUDIT_KEY audit.log",
	}
	return report.Render(pkg)
}

func mapHypotheses(in []Hypothesis) []report.Hypothesis {
	out := make([]report.Hypothesis, len(in))
	for i, h := range in {
		out[i] = report.Hypothesis{
			ID: h.ID, Statement: h.Statement, Status: h.Status,
			CreatedBy: h.CreatedBy, CreatedAt: h.CreatedAt, UpdatedAt: h.UpdatedAt,
		}
	}
	return out
}

func mapAnnotations(in []Annotation) []report.Annotation {
	out := make([]report.Annotation, len(in))
	for i, a := range in {
		out[i] = report.Annotation{EventID: a.EventID, Body: a.Body, Author: a.Author, Timestamp: a.Timestamp}
	}
	return out
}

func mapTimeline(in []TimelineEntry) []report.TimelineEntry {
	out := make([]report.TimelineEntry, len(in))
	for i, t := range in {
		out[i] = report.TimelineEntry{
			Kind: t.Kind, Timestamp: t.Timestamp, Severity: t.Severity,
			Title: t.Title, Detail: t.Detail, RefID: t.RefID,
		}
	}
	return out
}

func mapConfidence(c *Confidence) *report.ConfidenceInfo {
	if c == nil {
		return nil
	}
	return &report.ConfidenceInfo{Score: c.Score, Explanation: c.Explanation}
}
