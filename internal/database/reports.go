package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ReportRepository handles persistence for report templates, schedules, and instances.
type ReportRepository struct {
	db DatabaseStore
}

func NewReportRepository(db DatabaseStore) *ReportRepository {
	return &ReportRepository{db: db}
}

// Templates

func (r *ReportRepository) CreateTemplate(ctx context.Context, t *ReportTemplate) error {
	r.db.Lock()
	defer r.db.Unlock()

	now := time.Now().Format(time.RFC3339)
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.TenantID = MustTenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO report_templates (id, tenant_id, name, description, sections_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.TenantID, t.Name, t.Description, t.SectionsJSON, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *ReportRepository) GetTemplate(ctx context.Context, id string) (*ReportTemplate, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)
	var t ReportTemplate
	err = conn.QueryRow(`
		SELECT id, tenant_id, name, description, sections_json, created_at, updated_at
		FROM report_templates WHERE id = ? AND tenant_id = ?
	`, id, tenantID).Scan(&t.ID, &t.TenantID, &t.Name, &t.Description, &t.SectionsJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ReportRepository) ListTemplates(ctx context.Context) ([]ReportTemplate, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)
	rows, err := conn.Query(`
		SELECT id, tenant_id, name, description, sections_json, created_at, updated_at
		FROM report_templates WHERE tenant_id = ? ORDER BY name ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []ReportTemplate
	for rows.Next() {
		var t ReportTemplate
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Description, &t.SectionsJSON, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, nil
}

// Schedules

func (r *ReportRepository) CreateSchedule(ctx context.Context, s *ReportSchedule) error {
	r.db.Lock()
	defer r.db.Unlock()

	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	s.TenantID = MustTenantFromContext(ctx)
	s.CreatedAt = time.Now().Format(time.RFC3339)
	if s.NextRunAt == "" {
		s.NextRunAt = time.Now().Add(time.Duration(s.IntervalMins) * time.Minute).Format(time.RFC3339)
	}

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO report_schedules (id, tenant_id, template_id, name, interval_mins, next_run_at, recipients_json, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.TenantID, s.TemplateID, s.Name, s.IntervalMins, s.NextRunAt, s.Recipients, s.IsActive, s.CreatedAt)
	return err
}

func (r *ReportRepository) GetDueSchedules(ctx context.Context) ([]ReportSchedule, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	rows, err := conn.Query(`
		SELECT id, tenant_id, template_id, name, interval_mins, next_run_at, recipients_json, is_active, last_run_at, created_at
		FROM report_schedules WHERE is_active = 1 AND next_run_at <= ?
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []ReportSchedule
	for rows.Next() {
		var s ReportSchedule
		if err := rows.Scan(&s.ID, &s.TenantID, &s.TemplateID, &s.Name, &s.IntervalMins, &s.NextRunAt, &s.Recipients, &s.IsActive, &s.LastRunAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

func (r *ReportRepository) MarkScheduleRun(ctx context.Context, id string) error {
	r.db.Lock()
	defer r.db.Unlock()

	conn, err := r.db.Conn()
	if err != nil {
		return err
	}

	var interval int
	err = conn.QueryRow("SELECT interval_mins FROM report_schedules WHERE id = ?", id).Scan(&interval)
	if err != nil {
		return err
	}

	lastRun := time.Now().Format(time.RFC3339)
	nextRun := time.Now().Add(time.Duration(interval) * time.Minute).Format(time.RFC3339)

	_, err = r.db.ReplicatedExecContext(ctx, `
		UPDATE report_schedules SET last_run_at = ?, next_run_at = ? WHERE id = ?
	`, lastRun, nextRun, id)
	return err
}

// Generated Reports

func (r *ReportRepository) CreateReportInstance(ctx context.Context, g *GeneratedReport) error {
	r.db.Lock()
	defer r.db.Unlock()

	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	g.TenantID = MustTenantFromContext(ctx)
	if g.CreatedAt == "" {
		g.CreatedAt = time.Now().Format(time.RFC3339)
	}

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO generated_reports (id, tenant_id, schedule_id, template_id, title, period_start, period_end, file_path, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, g.ID, g.TenantID, g.ScheduleID, g.TemplateID, g.Title, g.PeriodStart, g.PeriodEnd, g.FilePath, g.Status, g.CreatedAt)
	return err
}

func (r *ReportRepository) ListReports(ctx context.Context, limit int) ([]GeneratedReport, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := MustTenantFromContext(ctx)
	rows, err := conn.Query(`
		SELECT id, tenant_id, schedule_id, template_id, title, period_start, period_end, file_path, status, created_at
		FROM generated_reports WHERE tenant_id = ? ORDER BY created_at DESC LIMIT ?
	`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []GeneratedReport
	for rows.Next() {
		var g GeneratedReport
		if err := rows.Scan(&g.ID, &g.TenantID, &g.ScheduleID, &g.TemplateID, &g.Title, &g.PeriodStart, &g.PeriodEnd, &g.FilePath, &g.Status, &g.CreatedAt); err != nil {
			return nil, err
		}
		reports = append(reports, g)
	}
	return reports, nil
}
