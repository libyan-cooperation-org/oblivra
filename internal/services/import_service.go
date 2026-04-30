package services

import (
	"context"
	"io"
	"log/slog"

	"github.com/kingknull/oblivra/internal/importer"
	"github.com/kingknull/oblivra/internal/ingest"
	"github.com/kingknull/oblivra/internal/parsers"
)

type ImportService struct {
	log      *slog.Logger
	pipeline *ingest.Pipeline
}

func NewImportService(log *slog.Logger, p *ingest.Pipeline) *ImportService {
	return &ImportService{log: log, pipeline: p}
}

func (s *ImportService) ServiceName() string { return "ImportService" }

// Run streams r into the platform under the given tenant + source label.
// `format` may be empty (auto-detect).
func (s *ImportService) Run(ctx context.Context, r io.Reader, tenantID, source string, format string) (importer.Summary, error) {
	imp := importer.New(s.pipeline, importer.Options{
		TenantID: tenantID,
		Source:   source,
		Format:   parsers.Format(format),
		Logger:   s.log,
	})
	return imp.Run(ctx, r)
}
