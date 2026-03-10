package app

import (
	"context"

	"github.com/kingknull/oblivrashell/internal/ledger"
	"github.com/wailsapp/wails/v2/pkg/logger"
)

type LedgerService struct {
	ctx   context.Context
	log   logger.Logger
	chain *ledger.Chain
}

func NewLedgerService() *LedgerService {
	return &LedgerService{
		chain: ledger.NewChain(),
	}
}

// RegisterCtx supports the interface check in ServiceRegistry
func (s *LedgerService) RegisterCtx(ctx context.Context, logger logger.Logger) {
	s.ctx = ctx
	s.log = logger
	s.log.Info("LedgerService initialized with Genesis Block")
}

func (s *LedgerService) Name() string {
	return "LedgerService"
}

func (s *LedgerService) Startup(ctx context.Context) {
	s.ctx = ctx
}

func (s *LedgerService) Shutdown() {
}

// GetChain returns the entire verifiable chain.
func (s *LedgerService) GetChain() []ledger.Block {
	return s.chain.GetBlocks()
}

// AppendEvidence enables the front-end or backend to push evidence onto the ledger manually.
func (s *LedgerService) AppendEvidence(payload string, dataType string) *ledger.Block {
	return s.chain.AddBlock([]byte(payload), dataType)
}

// VerifyChain kicks off a cryptographic verification map of the ledger from genesis to HEAD.
func (s *LedgerService) VerifyChain() string {
	err := s.chain.Verify()
	if err != nil {
		s.log.Error("Ledger Verification Failed: " + err.Error())
		return err.Error()
	}
	s.log.Info("Ledger verified successfully")
	return "VALID"
}

// ExportChain generates a JSON payload for external audits.
func (s *LedgerService) ExportChain() string {
	bytes, err := s.chain.Export()
	if err != nil {
		return ""
	}
	return string(bytes)
}
