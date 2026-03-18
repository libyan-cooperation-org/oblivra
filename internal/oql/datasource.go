package oql

import (
	"context"
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
	// For in-memory, we assume the data is already filtered or will be filtered by the executor.
	return s.Rows, nil
}
