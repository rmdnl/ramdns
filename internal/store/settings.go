package store

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Setting struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

func (s *Store) GetSetting(ctx context.Context, key string) (Setting, error) {
	if key == "" {
		return Setting{}, errors.New("setting key is empty")
	}

	var setting Setting
	var updatedAt string

	err := s.db.QueryRowContext(
		ctx,
		`SELECT key, value, updated_at
		 FROM settings
		 WHERE key = ?`,
		key,
	).Scan(&setting.Key, &setting.Value, &updatedAt)
	if err != nil {
		return Setting{}, fmt.Errorf("get setting %q: %w", key, err)
	}

	setting.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return Setting{}, fmt.Errorf("parse setting %q timestamp: %w", key, err)
	}

	return setting, nil
}

func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	if key == "" {
		return errors.New("setting key is empty")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO settings(key, value, updated_at)
		 VALUES(?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET
		     value = excluded.value,
		     updated_at = excluded.updated_at`,
		key,
		value,
		now,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}

	return nil
}

func (s *Store) ListSettings(ctx context.Context) ([]Setting, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT key, value, updated_at
		 FROM settings
		 ORDER BY key`,
	)
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()

	var result []Setting

	for rows.Next() {
		var (
			setting   Setting
			updatedAt string
		)

		if err := rows.Scan(
			&setting.Key,
			&setting.Value,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}

		setting.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse setting %q timestamp: %w", setting.Key, err)
		}

		result = append(result, setting)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settings: %w", err)
	}

	return result, nil
}
