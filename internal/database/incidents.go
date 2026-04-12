package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// ensure Database implements IncidentStore at compile time
var _ IncidentStore = (*Database)(nil)

// Upsert inserts a new incident or updates an existing one if the ID matches.
func (d *Database) Upsert(ctx context.Context, incident *Incident) error {
	db, err := d.Conn()
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	tacticsJSON, _ := json.Marshal(incident.MitreTactics)
	techniquesJSON, _ := json.Marshal(incident.MitreTechniques)

	query := `
		INSERT INTO incidents (
			id, rule_id, group_key, status, severity, description,
			title, first_seen_at, last_seen_at, event_count, owner,
			mitre_tactics, mitre_techniques, resolution_reason,
			triage_score, triage_reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status=excluded.status,
			severity=excluded.severity,
			description=excluded.description,
			title=excluded.title,
			last_seen_at=excluded.last_seen_at,
			event_count=excluded.event_count,
			owner=excluded.owner,
			mitre_tactics=excluded.mitre_tactics,
			mitre_techniques=excluded.mitre_techniques,
			resolution_reason=excluded.resolution_reason,
			triage_score=excluded.triage_score,
			triage_reason=excluded.triage_reason
	`

	_, err = db.ExecContext(ctx, query,
		incident.ID, incident.RuleID, incident.GroupKey, incident.Status,
		incident.Severity, incident.Description, incident.Title,
		incident.FirstSeenAt, incident.LastSeenAt, incident.EventCount,
		incident.Owner, string(tacticsJSON), string(techniquesJSON),
		incident.ResolutionReason, incident.TriageScore, incident.TriageReason,
	)
	return err
}

// GetByID retrieves a specific incident.
func (d *Database) GetByID(ctx context.Context, id string) (*Incident, error) {
	db, err := d.Conn()
	if err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	row := db.QueryRowContext(ctx, "SELECT id, rule_id, group_key, status, severity, description, title, first_seen_at, last_seen_at, event_count, owner, mitre_tactics, mitre_techniques, resolution_reason, triage_score, triage_reason FROM incidents WHERE id = ?", id)
	return scanIncident(row)
}

// GetByRuleAndGroup fetches the active/latest incident for a specific rule & entity to allow appending.
func (d *Database) GetByRuleAndGroup(ctx context.Context, ruleID string, groupKey string) (*Incident, error) {
	db, err := d.Conn()
	if err != nil {
		return nil, err
	}
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Only fetch New, Active, or Investigating incidents to group with. If it's Closed, it starts a new Incident.
	query := `
		SELECT id, rule_id, group_key, status, severity, description, title, 
		       first_seen_at, last_seen_at, event_count, owner, mitre_tactics, 
			   mitre_techniques, resolution_reason, triage_score, triage_reason 
		FROM incidents 
		WHERE rule_id = ? AND group_key = ? AND status != 'Closed'
		ORDER BY last_seen_at DESC LIMIT 1
	`
	row := db.QueryRowContext(ctx, query, ruleID, groupKey)
	incident, err := scanIncident(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // Return nil if no active incident to append to
	}
	return incident, err
}

// Search retrieves incidents, optionally filtered by status or owner.
func (d *Database) Search(ctx context.Context, status string, owner string, limit int) ([]Incident, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
		SELECT id, rule_id, group_key, status, severity, description, title, 
		       first_seen_at, last_seen_at, event_count, owner, mitre_tactics, 
			   mitre_techniques, resolution_reason, triage_score, triage_reason 
		FROM incidents WHERE 1=1
	`
	var args []interface{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	if owner != "" {
		query += " AND owner = ?"
		args = append(args, owner)
	}

	query += " ORDER BY last_seen_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		incident, err := scanIncidentList(rows)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, *incident)
	}
	return incidents, nil
}

// UpdateStatus fast-paths a status change (e.g., closing an incident).
func (d *Database) UpdateStatus(ctx context.Context, id string, status string, reason string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.ExecContext(ctx, "UPDATE incidents SET status = ?, resolution_reason = ? WHERE id = ?", status, reason, id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanIncident(row rowScanner) (*Incident, error) {
	var i Incident
	var tacticsStr, techniquesStr string
	var firstSeen, lastSeen time.Time

	err := row.Scan(
		&i.ID, &i.RuleID, &i.GroupKey, &i.Status, &i.Severity, &i.Description,
		&i.Title, &firstSeen, &lastSeen, &i.EventCount, &i.Owner,
		&tacticsStr, &techniquesStr, &i.ResolutionReason, &i.TriageScore, &i.TriageReason,
	)
	if err != nil {
		return nil, err
	}

	i.FirstSeenAt = firstSeen.Format(time.RFC3339)
	i.LastSeenAt = lastSeen.Format(time.RFC3339)
	_ = json.Unmarshal([]byte(tacticsStr), &i.MitreTactics)
	_ = json.Unmarshal([]byte(techniquesStr), &i.MitreTechniques)

	if i.MitreTactics == nil {
		i.MitreTactics = []string{}
	}
	if i.MitreTechniques == nil {
		i.MitreTechniques = []string{}
	}

	return &i, nil
}

func scanIncidentList(rows *sql.Rows) (*Incident, error) {
	return scanIncident(rows)
}
