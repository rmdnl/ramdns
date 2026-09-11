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

func TestSettingRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ramdns.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	if err := s.SetSetting(ctx, "upstream_timeout", "3s"); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	got, err := s.GetSetting(ctx, "upstream_timeout")
	if err != nil {
		t.Fatalf("GetSetting() error = %v", err)
	}

	if got.Key != "upstream_timeout" {
		t.Fatalf("Key = %q, want %q", got.Key, "upstream_timeout")
	}

	if got.Value != "3s" {
		t.Fatalf("Value = %q, want %q", got.Value, "3s")
	}

	if got.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt is zero")
	}

	if err := s.SetSetting(ctx, "upstream_timeout", "5s"); err != nil {
		t.Fatalf("SetSetting() update error = %v", err)
	}

	got, err = s.GetSetting(ctx, "upstream_timeout")
	if err != nil {
		t.Fatalf("GetSetting() after update error = %v", err)
	}

	if got.Value != "5s" {
		t.Fatalf("Value after update = %q, want %q", got.Value, "5s")
	}
}

func TestGetSettingMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ramdns.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	_, err = s.GetSetting(context.Background(), "does_not_exist")
	if err == nil {
		t.Fatal("GetSetting() expected error for missing setting")
	}
}
