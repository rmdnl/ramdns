package store

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Adlist struct {
	ID        int64
	Name      string
	URL       string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Store) CreateAdlist(ctx context.Context, name, url string, enabled bool) (Adlist, error) {
	if name == "" {
		return Adlist{}, errors.New("adlist name is empty")
	}
	if url == "" {
		return Adlist{}, errors.New("adlist URL is empty")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO adlists(name, url, enabled, created_at, updated_at)
		 VALUES(?, ?, ?, ?, ?)`,
		name,
		url,
		boolToInt(enabled),
		now,
		now,
	)
	if err != nil {
		return Adlist{}, fmt.Errorf("create adlist: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Adlist{}, fmt.Errorf("get adlist ID: %w", err)
	}

	return Adlist{
		ID:        id,
		Name:      name,
		URL:       url,
		Enabled:   enabled,
		CreatedAt: parseTime(now),
		UpdatedAt: parseTime(now),
	}, nil
}

func (s *Store) ListAdlists(ctx context.Context) ([]Adlist, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, url, enabled, created_at, updated_at
		 FROM adlists
		 ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list adlists: %w", err)
	}
	defer rows.Close()

	var result []Adlist

	for rows.Next() {
		var (
			adlist    Adlist
			enabled   int
			createdAt string
			updatedAt string
		)

		if err := rows.Scan(
			&adlist.ID,
			&adlist.Name,
			&adlist.URL,
			&enabled,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan adlist: %w", err)
		}

		adlist.Enabled = enabled != 0

		var err error
		adlist.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse adlist created_at: %w", err)
		}

		adlist.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse adlist updated_at: %w", err)
		}

		result = append(result, adlist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate adlists: %w", err)
	}

	return result, nil
}

func (s *Store) GetAdlist(ctx context.Context, id int64) (Adlist, error) {
	if id <= 0 {
		return Adlist{}, errors.New("adlist ID is invalid")
	}

	var (
		adlist    Adlist
		enabled   int
		createdAt string
		updatedAt string
	)

	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, url, enabled, created_at, updated_at
		 FROM adlists
		 WHERE id = ?`,
		id,
	).Scan(
		&adlist.ID,
		&adlist.Name,
		&adlist.URL,
		&enabled,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return Adlist{}, fmt.Errorf("get adlist %d: %w", id, err)
	}

	adlist.Enabled = enabled != 0

	var errParse error
	adlist.CreatedAt, errParse = time.Parse(time.RFC3339Nano, createdAt)
	if errParse != nil {
		return Adlist{}, fmt.Errorf("parse adlist created_at: %w", errParse)
	}

	adlist.UpdatedAt, errParse = time.Parse(time.RFC3339Nano, updatedAt)
	if errParse != nil {
		return Adlist{}, fmt.Errorf("parse adlist updated_at: %w", errParse)
	}

	return adlist, nil
}

func (s *Store) SetAdlistEnabled(ctx context.Context, id int64, enabled bool) error {
	if id <= 0 {
		return errors.New("adlist ID is invalid")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE adlists
		 SET enabled = ?, updated_at = ?
		 WHERE id = ?`,
		boolToInt(enabled),
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("update adlist %d: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check adlist %d update: %w", id, err)
	}

	if affected == 0 {
		return fmt.Errorf("adlist %d not found", id)
	}

	return nil
}

func (s *Store) DeleteAdlist(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("adlist ID is invalid")
	}

	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM adlists WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete adlist %d: %w", id, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check adlist %d deletion: %w", id, err)
	}

	if affected == 0 {
		return fmt.Errorf("adlist %d not found", id)
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func parseTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, value)
	return t
}
