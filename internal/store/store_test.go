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

func TestAdlistCRUD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ramdns.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	created, err := s.CreateAdlist(
		ctx,
		"HaGeZi Multi",
		"https://example.com/adlist.txt",
		true,
	)
	if err != nil {
		t.Fatalf("CreateAdlist() error = %v", err)
	}

	if created.ID <= 0 {
		t.Fatalf("created ID = %d, want > 0", created.ID)
	}

	if created.Name != "HaGeZi Multi" {
		t.Fatalf("created Name = %q", created.Name)
	}

	if !created.Enabled {
		t.Fatal("created Enabled = false, want true")
	}

	got, err := s.GetAdlist(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAdlist() error = %v", err)
	}

	if got.URL != "https://example.com/adlist.txt" {
		t.Fatalf("URL = %q", got.URL)
	}

	if err := s.SetAdlistEnabled(ctx, created.ID, false); err != nil {
		t.Fatalf("SetAdlistEnabled() error = %v", err)
	}

	got, err = s.GetAdlist(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAdlist() after disable error = %v", err)
	}

	if got.Enabled {
		t.Fatal("Enabled = true after disable, want false")
	}

	adlists, err := s.ListAdlists(ctx)
	if err != nil {
		t.Fatalf("ListAdlists() error = %v", err)
	}

	if len(adlists) != 1 {
		t.Fatalf("ListAdlists() returned %d rows, want 1", len(adlists))
	}

	if err := s.DeleteAdlist(ctx, created.ID); err != nil {
		t.Fatalf("DeleteAdlist() error = %v", err)
	}

	if _, err := s.GetAdlist(ctx, created.ID); err == nil {
		t.Fatal("GetAdlist() after delete expected error")
	}
}

func TestCreateAdlistRejectsInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ramdns.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	if _, err := s.CreateAdlist(ctx, "", "https://example.com/list.txt", true); err == nil {
		t.Fatal("CreateAdlist() expected error for empty name")
	}

	if _, err := s.CreateAdlist(ctx, "Test", "", true); err == nil {
		t.Fatal("CreateAdlist() expected error for empty URL")
	}
}

func TestCustomRuleCRUD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ramdns.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	created, err := s.CreateCustomRule(ctx, "ads.example.com", "domain", true)
	if err != nil {
		t.Fatalf("CreateCustomRule() error = %v", err)
	}

	if created.ID <= 0 {
		t.Fatalf("created ID = %d, want > 0", created.ID)
	}

	got, err := s.GetCustomRule(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetCustomRule() error = %v", err)
	}

	if got.Domain != "ads.example.com" {
		t.Fatalf("Domain = %q, want %q", got.Domain, "ads.example.com")
	}

	if got.RuleType != "domain" {
		t.Fatalf("RuleType = %q, want %q", got.RuleType, "domain")
	}

	if !got.Enabled {
		t.Fatal("Enabled = false, want true")
	}

	if err := s.SetCustomRuleEnabled(ctx, created.ID, false); err != nil {
		t.Fatalf("SetCustomRuleEnabled() error = %v", err)
	}

	got, err = s.GetCustomRule(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetCustomRule() after disable error = %v", err)
	}

	if got.Enabled {
		t.Fatal("Enabled = true after disable, want false")
	}

	rules, err := s.ListCustomRules(ctx)
	if err != nil {
		t.Fatalf("ListCustomRules() error = %v", err)
	}

	if len(rules) != 1 {
		t.Fatalf("ListCustomRules() returned %d rows, want 1", len(rules))
	}

	if err := s.DeleteCustomRule(ctx, created.ID); err != nil {
		t.Fatalf("DeleteCustomRule() error = %v", err)
	}

	if _, err := s.GetCustomRule(ctx, created.ID); err == nil {
		t.Fatal("GetCustomRule() after delete expected error")
	}
}

func TestCreateCustomRuleRejectsInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ramdns.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	if _, err := s.CreateCustomRule(ctx, "", "domain", true); err == nil {
		t.Fatal("CreateCustomRule() expected error for empty domain")
	}

	if _, err := s.CreateCustomRule(ctx, "example.com", "invalid", true); err == nil {
		t.Fatal("CreateCustomRule() expected error for invalid rule type")
	}
}
