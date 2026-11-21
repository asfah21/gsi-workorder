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
	t.Setenv("ODOO_DB", "")
	t.Setenv("ODOO_SERVICE_LOGIN", "")
	t.Setenv("ODOO_SERVICE_PASSWORD", "")
	t.Setenv("ODOO_EMPLOYEE_NIK_FIELD", "")
	t.Setenv("ODOO_TOKEN", "")
	t.Setenv("SYNC_INTERVAL", "")
	t.Setenv("SYNC_CONCURRENCY", "")

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
	if cfg.OdooEmployeeNIKField != "nik" {
		t.Fatalf("OdooEmployeeNIKField = %q, want nik", cfg.OdooEmployeeNIKField)
	}
	if cfg.UserSyncEnabled() {
		t.Fatal("UserSyncEnabled() = true without worker JSON-RPC settings")
	}
	if cfg.SyncInterval.String() != "15m0s" {
		t.Fatalf("SyncInterval = %s, want 15m0s", cfg.SyncInterval)
	}
	if cfg.SyncConcurrency != 4 {
		t.Fatalf("SyncConcurrency = %d, want 4", cfg.SyncConcurrency)
	}
	if cfg.P2HSyncEnabled() {
		t.Fatal("P2HSyncEnabled() = true without token or service credentials")
	}
}

func TestLoadP2HSettings(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ODOO_TOKEN", "server-token")
	t.Setenv("SYNC_INTERVAL", "5m")
	t.Setenv("SYNC_CONCURRENCY", "2")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.P2HSyncEnabled() {
		t.Fatal("P2HSyncEnabled() = false, want true with ODOO_TOKEN")
	}
	if cfg.SyncInterval.String() != "5m0s" || cfg.SyncConcurrency != 2 {
		t.Fatalf("unexpected sync config: %s/%d", cfg.SyncInterval, cfg.SyncConcurrency)
	}
}

func TestLoadRejectsInvalidSyncSettings(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SYNC_INTERVAL", "30s")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for SYNC_INTERVAL below 1m")
	}

	t.Setenv("SYNC_INTERVAL", "15m")
	t.Setenv("SYNC_CONCURRENCY", "5")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error for SYNC_CONCURRENCY above 4")
	}
}

func TestLoadUserSyncSettings(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ODOO_DB", "odoo_new")
	t.Setenv("ODOO_SERVICE_LOGIN", "svc_gsi_workorder")
	t.Setenv("ODOO_SERVICE_PASSWORD", "secret")
	t.Setenv("ODOO_EMPLOYEE_NIK_FIELD", "nik")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.UserSyncEnabled() {
		t.Fatal("UserSyncEnabled() = false, want true")
	}
	if cfg.OdooDB != "odoo_new" || cfg.OdooEmployeeNIKField != "nik" {
		t.Fatalf("unexpected user sync config: %#v", cfg)
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
