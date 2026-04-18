package services

import (
	"context"
	"sync"

	"github.com/kingknull/oblivrashell/internal/api"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/oql"
	"github.com/kingknull/oblivrashell/internal/platform"
)

// DashboardService manages the custom Dashboard Studio lifecycle and data orchestration.
type DashboardService struct {
	BaseService
	repo      database.DashboardStore
	analytics *AnalyticsService
	log       *logger.Logger
	pool      *platform.WorkerPool
}

func NewDashboardService(
	repo database.DashboardStore,
	analytics *AnalyticsService,
	log *logger.Logger,
) *DashboardService {
	// Sized for concurrent widget queries
	pool := platform.NewWorkerPool("dashboard-queries", 8)
	pool.Start()

	return &DashboardService{
		repo:      repo,
		analytics: analytics,
		log:       log.WithPrefix("dashboard-studio"),
		pool:      pool,
	}
}

func (s *DashboardService) Name() string { return "dashboard-studio-service" }

func (s *DashboardService) Stop(ctx context.Context) error {
	if s.pool != nil {
		s.pool.Stop()
	}
	return nil
}

// Management

func (s *DashboardService) CreateDashboard(ctx context.Context, d *database.Dashboard) error {
	return s.repo.Create(ctx, d)
}

func (s *DashboardService) GetDashboard(ctx context.Context, id string) (*database.Dashboard, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *DashboardService) ListDashboards(ctx context.Context) ([]database.Dashboard, error) {
	return s.repo.List(ctx)
}

func (s *DashboardService) UpdateDashboard(ctx context.Context, d *database.Dashboard) error {
	return s.repo.Update(ctx, d)
}

func (s *DashboardService) DeleteDashboard(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Widget Management

func (s *DashboardService) AddWidget(ctx context.Context, w *database.DashboardWidget) error {
	return s.repo.AddWidget(ctx, w)
}

func (s *DashboardService) UpdateWidget(ctx context.Context, w *database.DashboardWidget) error {
	return s.repo.UpdateWidget(ctx, w)
}

func (s *DashboardService) DeleteWidget(ctx context.Context, dashboardID, widgetID string) error {
	return s.repo.DeleteWidget(ctx, dashboardID, widgetID)
}

// Data Orchestration

// GetDashboardData executes OQL queries for ALL widgets in a dashboard in parallel.
func (s *DashboardService) GetDashboardData(ctx context.Context, id string) (*api.DashboardData, error) {
	dash, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	widgets, err := s.repo.GetWidgets(ctx, dash.ID)
	if err != nil {
		return nil, err
	}

	results := make(map[string]*oql.QueryResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, w := range widgets {
		wg.Add(1)
		widget := w // capture
		
		// Use Submit for bounded concurrency
		s.pool.Submit(func() {
			defer wg.Done()
			
			res, err := s.analytics.RunOQL(ctx, widget.QueryOQL)
			if err != nil {
				s.log.Warn("Widget %s query failed: %v", widget.ID, err)
				return
			}

			mu.Lock()
			results[widget.ID] = res
			mu.Unlock()
		})
	}

	wg.Wait()

	return &api.DashboardData{
		DashboardID: dash.ID,
		Results:     results,
	}, nil
}
