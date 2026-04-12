package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// DashboardRepository handles persistence for the custom Dashboard Studio.
type DashboardRepository struct {
	db DatabaseStore
}

func NewDashboardRepository(db DatabaseStore) *DashboardRepository {
	return &DashboardRepository{db: db}
}

func (r *DashboardRepository) Create(ctx context.Context, d *Dashboard) error {
	r.db.Lock()
	defer r.db.Unlock()

	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	d.TenantID = TenantFromContext(ctx)
	now := time.Now().Format(time.RFC3339)
	d.CreatedAt = now
	d.UpdatedAt = now

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO dashboards (id, tenant_id, name, description, layout, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, d.ID, d.TenantID, d.Name, d.Description, d.Layout, d.OwnerID, d.CreatedAt, d.UpdatedAt)
	return err
}

func (r *DashboardRepository) GetByID(ctx context.Context, id string) (*Dashboard, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)
	var d Dashboard
	err = conn.QueryRow(`
		SELECT id, tenant_id, name, description, layout, owner_id, created_at, updated_at
		FROM dashboards WHERE id = ? AND tenant_id = ?
	`, id, tenantID).Scan(&d.ID, &d.TenantID, &d.Name, &d.Description, &d.Layout, &d.OwnerID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Fetch widgets
	widgets, err := r.GetWidgets(ctx, d.ID)
	if err == nil {
		d.Widgets = widgets
	}

	return &d, nil
}

func (r *DashboardRepository) List(ctx context.Context) ([]Dashboard, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	tenantID := TenantFromContext(ctx)
	rows, err := conn.Query(`
		SELECT id, tenant_id, name, description, layout, owner_id, created_at, updated_at
		FROM dashboards WHERE tenant_id = ? ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dashboards []Dashboard
	for rows.Next() {
		var d Dashboard
		if err := rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.Description, &d.Layout, &d.OwnerID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		dashboards = append(dashboards, d)
	}
	return dashboards, nil
}

func (r *DashboardRepository) Update(ctx context.Context, d *Dashboard) error {
	r.db.Lock()
	defer r.db.Unlock()

	d.UpdatedAt = time.Now().Format(time.RFC3339)
	tenantID := TenantFromContext(ctx)

	_, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE dashboards SET name = ?, description = ?, layout = ?, updated_at = ?
		WHERE id = ? AND tenant_id = ?
	`, d.Name, d.Description, d.Layout, d.UpdatedAt, d.ID, tenantID)
	return err
}

func (r *DashboardRepository) Delete(ctx context.Context, id string) error {
	r.db.Lock()
	defer r.db.Unlock()

	tenantID := TenantFromContext(ctx)
	_, err := r.db.ReplicatedExecContext(ctx, "DELETE FROM dashboards WHERE id = ? AND tenant_id = ?", id, tenantID)
	return err
}

// Widget Management

func (r *DashboardRepository) AddWidget(ctx context.Context, w *DashboardWidget) error {
	r.db.Lock()
	defer r.db.Unlock()

	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	w.CreatedAt = time.Now().Format(time.RFC3339)

	_, err := r.db.ReplicatedExecContext(ctx, `
		INSERT INTO dashboard_widgets (id, dashboard_id, title, viz_type, query_oql, layout_x, layout_y, layout_w, layout_h, refresh_interval_secs, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, w.ID, w.DashboardID, w.Title, w.VizType, w.QueryOQL, w.LayoutX, w.LayoutY, w.LayoutW, w.LayoutH, w.RefreshIntervalSecs, w.CreatedAt)
	return err
}

func (r *DashboardRepository) GetWidgets(ctx context.Context, dashboardID string) ([]DashboardWidget, error) {
	r.db.RLock()
	defer r.db.RUnlock()

	conn, err := r.db.Conn()
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(`
		SELECT id, dashboard_id, title, viz_type, query_oql, layout_x, layout_y, layout_w, layout_h, refresh_interval_secs, created_at
		FROM dashboard_widgets WHERE dashboard_id = ? ORDER BY layout_y ASC, layout_x ASC
	`, dashboardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var widgets []DashboardWidget
	for rows.Next() {
		var w DashboardWidget
		if err := rows.Scan(&w.ID, &w.DashboardID, &w.Title, &w.VizType, &w.QueryOQL, &w.LayoutX, &w.LayoutY, &w.LayoutW, &w.LayoutH, &w.RefreshIntervalSecs, &w.CreatedAt); err != nil {
			return nil, err
		}
		widgets = append(widgets, w)
	}
	return widgets, nil
}

func (r *DashboardRepository) UpdateWidget(ctx context.Context, w *DashboardWidget) error {
	r.db.Lock()
	defer r.db.Unlock()

	_, err := r.db.ReplicatedExecContext(ctx, `
		UPDATE dashboard_widgets SET title = ?, viz_type = ?, query_oql = ?, layout_x = ?, layout_y = ?, layout_w = ?, layout_h = ?, refresh_interval_secs = ?
		WHERE id = ? AND dashboard_id = ?
	`, w.Title, w.VizType, w.QueryOQL, w.LayoutX, w.LayoutY, w.LayoutW, w.LayoutH, w.RefreshIntervalSecs, w.ID, w.DashboardID)
	return err
}

func (r *DashboardRepository) DeleteWidget(ctx context.Context, dashboardID, widgetID string) error {
	r.db.Lock()
	defer r.db.Unlock()

	_, err := r.db.ReplicatedExecContext(ctx, "DELETE FROM dashboard_widgets WHERE id = ? AND dashboard_id = ?", widgetID, dashboardID)
	return err
}
