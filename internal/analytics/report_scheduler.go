package analytics

import (
	"context"
	"time"

	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
)

// ReportScheduler manages the periodic execution of scheduled reports.
type ReportScheduler struct {
	repo    database.ReportStore
	factory *ReportFactory
	log     *logger.Logger
	stop    chan struct{}
}

func NewReportScheduler(repo database.ReportStore, factory *ReportFactory, log *logger.Logger) *ReportScheduler {
	return &ReportScheduler{
		repo:    repo,
		factory: factory,
		log:     log.WithPrefix("scheduler"),
		stop:    make(chan struct{}),
	}
}

func (s *ReportScheduler) Name() string { return "report-scheduler" }

func (s *ReportScheduler) Dependencies() []string { return []string{"analytics-service"} }

func (s *ReportScheduler) Start(ctx context.Context) error {
	s.log.Info("Report scheduler starting...")
	ticker := time.NewTicker(5 * time.Minute)
	
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.process(ctx)
			case <-s.stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (s *ReportScheduler) Stop(ctx context.Context) error {
	close(s.stop)
	return nil
}

func (s *ReportScheduler) process(ctx context.Context) {
	schedules, err := s.repo.GetDueSchedules(ctx)
	if err != nil {
		s.log.Error("Failed to fetch due schedules: %v", err)
		return
	}

	for _, sch := range schedules {
		s.log.Info("Executing scheduled report: %s (%s)", sch.Name, sch.ID)
		
		tenantCtx := database.WithTenant(ctx, sch.TenantID)
		go s.runReport(tenantCtx, sch)
	}
}

func (s *ReportScheduler) runReport(ctx context.Context, sch database.ReportSchedule) {
	start := time.Now().Add(-time.Duration(sch.IntervalMins) * time.Minute).Format(time.RFC3339)
	end := time.Now().Format(time.RFC3339)

	instance := &database.GeneratedReport{
		ScheduleID:  sch.ID,
		TemplateID:  sch.TemplateID,
		Title:       sch.Name,
		PeriodStart: start,
		PeriodEnd:   end,
		Status:      "pending",
	}

	if err := s.repo.CreateReportInstance(ctx, instance); err != nil {
		return
	}

	path, err := s.factory.GenerateHTML(ctx, sch.TemplateID, start, end)
	if err != nil {
		s.log.Error("Failed to generate report %s: %v", sch.ID, err)
		return
	}

	s.repo.MarkScheduleRun(ctx, sch.ID)
	s.log.Info("Report generated: %s", path)
}
