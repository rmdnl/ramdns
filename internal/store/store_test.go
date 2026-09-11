package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ramdns.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	var count int
	if err := s.DB().QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`,
	).Scan(&count); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}

	if count != 1 {
		t.Fatalf("migration count = %d, want 1", count)
	}

	tables := []string{
		"settings",
		"adlists",
		"custom_rules",
	}

	for _, table := range tables {
		var exists int

		err := s.DB().QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`,
			table,
		).Scan(&exists)

		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}

		if exists != 1 {
			t.Fatalf("table %s does not exist", table)
		}
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatal("Open(\"\") expected error")
	}
}
