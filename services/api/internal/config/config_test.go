package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://gsi:secret@db:5432/workorder?sslmode=disable")
	t.Setenv("PORT", "")
	t.Setenv("MIGRATIONS_DIR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.MigrationsDir != "migrations" {
		t.Fatalf("MigrationsDir = %q, want migrations", cfg.MigrationsDir)
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("DatabaseURL is empty")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error when DATABASE_URL is empty")
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://gsi:secret@db:5432/workorder?sslmode=disable")
	t.Setenv("PORT", "70000")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for an invalid PORT")
	}
}
