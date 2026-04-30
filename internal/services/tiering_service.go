package services

import (
	"context"
	"log/slog"

	"github.com/kingknull/oblivra/internal/storage/tiering"
)

type TieringService struct {
	log      *slog.Logger
	migrator *tiering.Migrator
}

func NewTieringService(log *slog.Logger, m *tiering.Migrator) *TieringService {
	return &TieringService{log: log, migrator: m}
}

func (s *TieringService) ServiceName() string { return "TieringService" }

// Promote runs one migration pass on demand.
func (s *TieringService) Promote(ctx context.Context) (int, error) {
	if s.migrator == nil {
		return 0, nil
	}
	return s.migrator.Run(ctx)
}

func (s *TieringService) Stats() tiering.Stats {
	if s.migrator == nil {
		return tiering.Stats{}
	}
	return s.migrator.Stats()
}

// VerifyWarm runs a cross-tier integrity check on recent warm Parquet files.
func (s *TieringService) VerifyWarm(maxFiles int) (tiering.VerifyResult, error) {
	if s.migrator == nil {
		return tiering.VerifyResult{}, nil
	}
	return s.migrator.Verify(maxFiles)
}
