package oql

import (
	"context"
	"github.com/kingknull/oblivrashell/internal/auth"
)

// DataSource provides a way for the OQL executor to fetch raw data.
type DataSource interface {
	// Fetch retrieves rows matching the initial search expression and time range.
	Fetch(ctx context.Context, search SearchExpr, timeRange TimeRange) ([]Row, error)
}

// InMemSource is a simple implementation of DataSource that wraps a static slice of rows.
type InMemSource struct {
	Rows []Row
}

func (s *InMemSource) Fetch(ctx context.Context, search SearchExpr, timeRange TimeRange) ([]Row, error) {
	user := auth.UserFromContext(ctx)
	if user == nil {
		return nil, nil // Or return unauthorized error? For internal, let's just return empty.
	}

	// Filter rows by TenantID to ensure isolation
	filtered := make([]Row, 0, len(s.Rows))
	for _, row := range s.Rows {
		tid, ok := row["tenant_id"].(string)
		if !ok || tid == user.TenantID || user.TenantID == "system" {
			filtered = append(filtered, row)
		}
	}

	return filtered, nil
}
