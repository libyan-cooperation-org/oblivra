package database

import (
	"context"
	"database/sql"
	"time"
)

type suppressionRepository struct {
	db DatabaseStore
}

func NewSuppressionRepository(db DatabaseStore) SuppressionStore {
	return &suppressionRepository{db: db}
}

func (r *suppressionRepository) List(ctx context.Context) ([]SuppressionRule, error) {
	tenantID := MustTenantFromContext(ctx)
	rows, err := r.db.QueryContext(ctx, 
		"SELECT id, tenant_id, label, description, rule_id, field, value, is_regex, expires_at, is_active, last_matched_at, created_at, updated_at FROM suppression_rules WHERE tenant_id = ?", 
		tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []SuppressionRule
	for rows.Next() {
		var rule SuppressionRule
		err := rows.Scan(&rule.ID, &rule.TenantID, &rule.Label, &rule.Description, &rule.RuleID, &rule.Field, &rule.Value, &rule.IsRegex, &rule.ExpiresAt, &rule.IsActive, &rule.LastMatchedAt, &rule.CreatedAt, &rule.UpdatedAt)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *suppressionRepository) GetByID(ctx context.Context, id string) (*SuppressionRule, error) {
	tenantID := MustTenantFromContext(ctx)
	row := r.db.QueryRowContext(ctx, 
		"SELECT id, tenant_id, label, description, rule_id, field, value, is_regex, expires_at, is_active, last_matched_at, created_at, updated_at FROM suppression_rules WHERE tenant_id = ? AND id = ?", 
		tenantID, id)
	
	var rule SuppressionRule
	err := row.Scan(&rule.ID, &rule.TenantID, &rule.Label, &rule.Description, &rule.RuleID, &rule.Field, &rule.Value, &rule.IsRegex, &rule.ExpiresAt, &rule.IsActive, &rule.LastMatchedAt, &rule.CreatedAt, &rule.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *suppressionRepository) GetByRuleID(ctx context.Context, ruleID string) ([]SuppressionRule, error) {
	tenantID := MustTenantFromContext(ctx)
	rows, err := r.db.QueryContext(ctx, 
		"SELECT id, tenant_id, label, description, rule_id, field, value, is_regex, expires_at, is_active, last_matched_at, created_at, updated_at FROM suppression_rules WHERE tenant_id = ? AND rule_id = ? AND is_active = 1", 
		tenantID, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []SuppressionRule
	for rows.Next() {
		var rule SuppressionRule
		err := rows.Scan(&rule.ID, &rule.TenantID, &rule.Label, &rule.Description, &rule.RuleID, &rule.Field, &rule.Value, &rule.IsRegex, &rule.ExpiresAt, &rule.IsActive, &rule.LastMatchedAt, &rule.CreatedAt, &rule.UpdatedAt)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *suppressionRepository) GetMatch(ctx context.Context, ruleID, field, value string) (*SuppressionRule, error) {
	tenantID := MustTenantFromContext(ctx)
	// We check for rules that match the specific ruleID or are global (ruleID is empty/null)
	// and match the field and value.
	// Note: Regex matching is handled in the service layer for complexity reasons, 
	// but we could do basic SQL LIKE if needed.
	row := r.db.QueryRowContext(ctx, 
		`SELECT id, tenant_id, label, description, rule_id, field, value, is_regex, expires_at, is_active, last_matched_at, created_at, updated_at 
		 FROM suppression_rules 
		 WHERE tenant_id = ? AND field = ? AND value = ? AND is_active = 1 
		 AND (rule_id = ? OR rule_id IS NULL OR rule_id = '')
		 AND (expires_at IS NULL OR expires_at = '' OR expires_at > ?)
		 LIMIT 1`, 
		tenantID, field, value, ruleID, time.Now().Format(time.RFC3339))
	
	var rule SuppressionRule
	err := row.Scan(&rule.ID, &rule.TenantID, &rule.Label, &rule.Description, &rule.RuleID, &rule.Field, &rule.Value, &rule.IsRegex, &rule.ExpiresAt, &rule.IsActive, &rule.LastMatchedAt, &rule.CreatedAt, &rule.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *suppressionRepository) Upsert(ctx context.Context, rule *SuppressionRule) error {
	rule.TenantID = MustTenantFromContext(ctx)
	now := time.Now().Format(time.RFC3339)
	if rule.CreatedAt == "" {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	_, err := r.db.ReplicatedExecContext(ctx, 
		`INSERT INTO suppression_rules (id, tenant_id, label, description, rule_id, field, value, is_regex, expires_at, is_active, last_matched_at, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET 
		 label=excluded.label, 
		 description=excluded.description, 
		 rule_id=excluded.rule_id, 
		 field=excluded.field, 
		 value=excluded.value, 
		 is_regex=excluded.is_regex, 
		 expires_at=excluded.expires_at, 
		 is_active=excluded.is_active, 
		 updated_at=excluded.updated_at`,
		rule.ID, rule.TenantID, rule.Label, rule.Description, rule.RuleID, rule.Field, rule.Value, rule.IsRegex, rule.ExpiresAt, rule.IsActive, rule.LastMatchedAt, rule.CreatedAt, rule.UpdatedAt)
	
	return err
}

func (r *suppressionRepository) Delete(ctx context.Context, id string) error {
	tenantID := MustTenantFromContext(ctx)
	_, err := r.db.ReplicatedExecContext(ctx, "DELETE FROM suppression_rules WHERE tenant_id = ? AND id = ?", tenantID, id)
	return err
}

func (r *suppressionRepository) MarkMatched(ctx context.Context, id string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.ReplicatedExecContext(ctx, "UPDATE suppression_rules SET last_matched_at = ? WHERE id = ?", now, id)
	return err
}
