package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/kingknull/oblivra/internal/report"
)

type ReportService struct {
	log    *slog.Logger
	cases  *InvestigationsService
	audit  *AuditService
}

func NewReportService(log *slog.Logger, cases *InvestigationsService, audit *AuditService) *ReportService {
	return &ReportService{log: log, cases: cases, audit: audit}
}

func (s *ReportService) ServiceName() string { return "ReportService" }

// CaseHTML renders a sealed-or-open case to a self-contained HTML evidence
// package. Returns the bytes; the caller (HTTP handler / Wails) decides
// whether to write to disk, return as a download, etc.
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
		Case:        *c,
		Timeline:    tl,
		Confidence:  conf,
		AuditRoot:   root,
		GeneratedAt: time.Now().UTC(),
		Verifier:    "oblivra-verify --hmac $OBLIVRA_AUDIT_KEY audit.log",
	}
	return report.Render(pkg)
}
