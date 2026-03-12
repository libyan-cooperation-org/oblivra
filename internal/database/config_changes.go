package database

import (
	"context"
	"fmt"
	"time"
)

// RecordChange persists a configuration change audit record.
func (d *Database) RecordChange(ctx context.Context, change *ConfigChange) error {
	db, err := d.Conn()
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if change.ID == "" {
		change.ID = fmt.Sprintf("cfg-%d", time.Now().UnixNano())
	}
	if change.Timestamp == "" {
		change.Timestamp = time.Now().Format(time.RFC3339)
	}

	query := `
		INSERT INTO config_changes (
			id, timestamp, category, key, old_value, new_value, risk_score, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = db.ExecContext(ctx, query,
		change.ID, change.Timestamp, change.Category, change.Key,
		change.OldValue, change.NewValue, change.RiskScore, change.Status,
	)
	return err
}

// GetChanges retrieves configuration changes, optionally filtered by category.
func (d *Database) GetChanges(ctx context.Context, category string, limit int) ([]ConfigChange, error) {
	db, err := d.Conn()
	if err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := "SELECT id, timestamp, category, key, old_value, new_value, risk_score, status FROM config_changes"
	var args []interface{}

	if category != "" {
		query += " WHERE category = ?"
		args = append(args, category)
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []ConfigChange
	for rows.Next() {
		var c ConfigChange
		err := rows.Scan(
			&c.ID, &c.Timestamp, &c.Category, &c.Key,
			&c.OldValue, &c.NewValue, &c.RiskScore, &c.Status,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	return changes, nil
}
