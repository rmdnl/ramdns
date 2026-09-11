package store

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type CustomRule struct {
	ID        int64
	Domain    string
	RuleType  string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Store) CreateCustomRule(ctx context.Context, domain, ruleType string, enabled bool) (CustomRule, error) {
	if domain == "" {
		return CustomRule{}, errors.New("custom rule domain is empty")
	}

	if ruleType != "exact" && ruleType != "domain" {
		return CustomRule{}, fmt.Errorf("invalid custom rule type %q", ruleType)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO custom_rules(domain, rule_type, enabled, created_at, updated_at)
		 VALUES(?, ?, ?, ?, ?)`,
		domain,
		ruleType,
		boolToInt(enabled),
		now,
		now,
	)
	if err != nil {
		return CustomRule{}, fmt.Errorf("create custom rule: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return CustomRule{}, fmt.Errorf("get custom rule ID: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339Nano, now)

	return CustomRule{
		ID:        id,
		Domain:    domain,
		RuleType:  ruleType,
		Enabled:   enabled,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}, nil
}

func (s *Store) ListCustomRules(ctx context.Context) ([]CustomRule, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, domain, rule_type, enabled, created_at, updated_at
		 FROM custom_rules
		 ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list custom rules: %w", err)
	}
	defer rows.Close()

	result := make([]CustomRule, 0)

	for rows.Next() {
		var (
			rule      CustomRule
			enabled   int
			createdAt string
			updatedAt string
		)

		if err := rows.Scan(
			&rule.ID,
			&rule.Domain,
			&rule.RuleType,
			&enabled,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan custom rule: %w", err)
		}

		rule.Enabled = enabled != 0

		var err error

		rule.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse custom rule created_at: %w", err)
		}

		rule.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse custom rule updated_at: %w", err)
		}

		result = append(result, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate custom rules: %w", err)
	}

	return result, nil
}

func (s *Store) GetCustomRule(ctx context.Context, id int64) (CustomRule, error) {
	if id <= 0 {
		return CustomRule{}, errors.New("custom rule ID is invalid")
	}

	var (
		rule      CustomRule
		enabled   int
		createdAt string
		updatedAt string
	)

	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, domain, rule_type, enabled, created_at, updated_at
		 FROM custom_rules
		 WHERE id = ?`,
		id,
	).Scan(
		&rule.ID,
		&rule.Domain,
		&rule.RuleType,
		&enabled,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return CustomRule{}, fmt.Errorf("get custom rule %d: %w", id, err)
	}

	rule.Enabled = enabled != 0

	var errParse error

	rule.CreatedAt, errParse = time.Parse(time.RFC3339Nano, createdAt)
	if errParse != nil {
		return CustomRule{}, fmt.Errorf("parse custom rule created_at: %w", errParse)
	}

	rule.UpdatedAt, errParse = time.Parse(time.RFC3339Nano, updatedAt)
	if errParse != nil {
		return CustomRule{}, fmt.Errorf("parse custom rule updated_at: %w", errParse)
	}

	return rule, nil
}

func (s *Store) SetCustomRuleEnabled(ctx context.Context, id int64, enabled bool) error {
	if id <= 0 {
		return errors.New("custom rule ID is invalid")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE custom_rules
		 SET enabled = ?, updated_at = ?
		 WHERE id = ?`,
		boolToInt(enabled),
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("update custom rule %d: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check custom rule %d update: %w", id, err)
	}

	if affected == 0 {
		return fmt.Errorf("custom rule %d not found", id)
	}

	return nil
}

func (s *Store) DeleteCustomRule(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("custom rule ID is invalid")
	}

	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM custom_rules WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete custom rule %d: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check custom rule %d deletion: %w", id, err)
	}

	if affected == 0 {
		return fmt.Errorf("custom rule %d not found", id)
	}

	return nil
}
