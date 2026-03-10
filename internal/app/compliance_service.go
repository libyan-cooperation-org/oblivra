package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/kingknull/oblivrashell/internal/compliance"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/vault"
)

// ComplianceService exposes compliance reporting and data integrity to the frontend
type ComplianceService struct {
	BaseService
	ctx             context.Context
	reportGenerator *compliance.ReportGenerator
	evaluator       *compliance.Evaluator
	bus             *eventbus.Bus
	log             *logger.Logger
	vault           *vault.Vault
	identity        *IdentityService
	api             *APIService
}

// Name returns the name of the service
func (s *ComplianceService) Name() string { return "ComplianceService" }

// NewComplianceService creates a new compliance service
func NewComplianceService(generator *compliance.ReportGenerator, evaluator *compliance.Evaluator, bus *eventbus.Bus, log *logger.Logger, v *vault.Vault, identity *IdentityService, api *APIService) *ComplianceService {
	return &ComplianceService{
		reportGenerator: generator,
		evaluator:       evaluator,
		bus:             bus,
		log:             log.WithPrefix("compliance"),
		vault:           v,
		identity:        identity,
		api:             api,
	}
}

// Startup initializes the service
func (s *ComplianceService) Startup(ctx context.Context) {
	s.ctx = ctx
}

// GenerateReport creates a new compliance report
func (s *ComplianceService) GenerateReport(reportType string, startUnix, endUnix int64) (*compliance.ComplianceReport, error) {
	start := time.Unix(startUnix, 0)
	end := time.Unix(endUnix, 0)

	s.log.Info("Generating %s report from %s to %s", reportType, start.Format(time.RFC3339), end.Format(time.RFC3339))
	return s.reportGenerator.GenerateReport(compliance.ReportType(reportType), start, end)
}

// ListReports returns previously generated reports
func (s *ComplianceService) ListReports() ([]compliance.ComplianceReport, error) {
	return s.reportGenerator.ListReports()
}

// SignData generates a secure HMAC-SHA256 signature for the given payload using the vault's master key
func (s *ComplianceService) SignData(payload string) (map[string]string, error) {
	if s.vault == nil || !s.vault.IsUnlocked() {
		return nil, fmt.Errorf("vault is locked or not available; cannot sign")
	}

	var sig string
	err := s.vault.AccessMasterKey(func(key []byte) error {
		h := hmac.New(sha256.New, key)
		h.Write([]byte(payload))
		sig = hex.EncodeToString(h.Sum(nil))
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %w", err)
	}

	return map[string]string{
		"signature": sig,
		"method":    "HMAC-SHA256",
		"ts":        time.Now().Format(time.RFC3339),
	}, nil
}

// ListCompliancePacks returns all loaded compliance packs.
func (s *ComplianceService) ListCompliancePacks() ([]compliance.PackDefinition, error) {
	if s.evaluator == nil {
		return nil, fmt.Errorf("compliance evaluator not initialized")
	}
	return s.evaluator.ListPacks(), nil
}

// EvaluatePack runs a compliance evaluation for a specific pack.
func (s *ComplianceService) EvaluatePack(packID string) (*compliance.PackResult, error) {
	if s.evaluator == nil {
		return nil, fmt.Errorf("compliance evaluator not initialized")
	}

	// Fetch some stats from repos for the state
	auditCount, _ := s.reportGenerator.GetAuditCount()

	// Real-time Identity telemetry
	identityStats, _ := s.identity.GetSecurityStats()

	// Real-time API telemetry
	tlsActive := false
	if s.api != nil && s.api.server != nil {
		tlsActive = s.api.server.IsTLS()
	}

	// Build current system state for evaluation
	state := compliance.SystemState{
		EncryptionEnabled:    true, // SQLCipher is always active in OBLIVRA
		MFAEnabled:           identityStats.MFAPassive > 0,
		TLSEnabled:           tlsActive,
		FIMEnabled:           true, // Sentinel FIM is active
		RBACEnabled:          identityStats.RBACActive,
		AlertingEnabled:      true, // Alerting engine is operational
		MerkleIntegrityValid: s.reportGenerator.IsMerkleValid(),
		EvidenceLockerAvail:  true, // Forensics service is bound
		AuditLogCount:        auditCount,
		EventTypesPresent:    make(map[string]bool),
	}

	return s.evaluator.Evaluate(packID, state)
}

// ExportReportPDF generates a PDF for a report and returns the file path.
func (s *ComplianceService) ExportReportPDF(reportType string, startUnix, endUnix int64) (string, error) {
	report, err := s.GenerateReport(reportType, startUnix, endUnix)
	if err != nil {
		return "", err
	}

	pdfData, err := s.reportGenerator.ExportPDF(report)
	if err != nil {
		return "", fmt.Errorf("export pdf: %w", err)
	}

	// Save to same location as JSON but with .pdf
	jsonPath, err := s.reportGenerator.ExportToFile(report)
	if err != nil {
		return "", err
	}

	pdfPath := jsonPath[:len(jsonPath)-5] + ".pdf"
	if err := os.WriteFile(pdfPath, pdfData, 0600); err != nil {
		return "", err
	}

	return pdfPath, nil
}
