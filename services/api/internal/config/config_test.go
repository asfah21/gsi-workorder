package config

import (
	"strings"
	"testing"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://gsi:secret@db:5432/workorder?sslmode=disable")
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("ODOO_BASE_URL", "http://10.10.11.2:8068")
}

func TestLoadDefaults(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PORT", "")
	t.Setenv("MIGRATIONS_DIR", "")
	t.Setenv("APP_TIMEZONE", "")
	t.Setenv("SEED_ADMIN_NIK", "")
	t.Setenv("SEED_ADMIN_NAME", "")

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
	if cfg.AppTimezone != defaultAppTimezone {
		t.Fatalf("AppTimezone = %q, want %q", cfg.AppTimezone, defaultAppTimezone)
	}
	if cfg.AppLocation == nil || cfg.AppLocation.String() != defaultAppTimezone {
		t.Fatalf("AppLocation = %v, want %q", cfg.AppLocation, defaultAppTimezone)
	}
	if cfg.OdooBaseURL != "http://10.10.11.2:8068" {
		t.Fatalf("OdooBaseURL = %q", cfg.OdooBaseURL)
	}
	if cfg.SeedAdminName != "Seed Admin" {
		t.Fatalf("SeedAdminName = %q, want Seed Admin", cfg.SeedAdminName)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DATABASE_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error when DATABASE_URL is empty")
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("PORT", "70000")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for an invalid PORT")
	}
}

func TestLoadRequiresStrongJWTSecret(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("JWT_SECRET", strings.Repeat("x", minimumJWTSecretLen-1))

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for a short JWT_SECRET")
	}
}

func TestLoadRequiresOdooBaseURL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ODOO_BASE_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error when ODOO_BASE_URL is empty")
	}
}

func TestLoadRejectsInvalidOdooBaseURL(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ODOO_BASE_URL", "10.10.11.2:8068")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for a URL without scheme")
	}
}

func TestLoadRejectsInvalidTimezone(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_TIMEZONE", "Asia/Not_A_Place")

	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for an invalid APP_TIMEZONE")
	}
}
