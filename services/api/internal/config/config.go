package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort          = "8080"
	defaultMigrationsDir = "migrations"
	defaultAppTimezone   = "Asia/Makassar"
	shutdownTimeout      = 10 * time.Second
	minimumJWTSecretLen  = 32
	odooRequestTimeout   = 10 * time.Second
)

// Config holds the runtime configuration needed by the API and worker.
type Config struct {
	DatabaseURL     string
	Port            string
	MigrationsDir   string
	ShutdownTimeout time.Duration

	// Authentication and Odoo.
	JWTSecret   string
	OdooBaseURL string
	OdooTimeout time.Duration

	// AppTimezone is used for business dates and display formatting.
	// AppLocation is the resolved IANA location.
	AppTimezone string
	AppLocation *time.Location

	// SeedAdminNIK optionally bootstraps/promotes a local admin after sync.
	// It never stores a password; login still validates against Odoo.
	SeedAdminNIK  string
	SeedAdminName string

	// Odoo JSON-RPC settings for the worker user/employee sync. They are
	// optional for the API so login remains available even before M1B env is
	// configured; the worker requires them to run a sync.
	OdooDB               string
	OdooServiceLogin     string
	OdooServicePassword  string
	OdooEmployeeNIKField string
}

// Load reads configuration from environment variables. Secrets are never
// hardcoded; DATABASE_URL, JWT_SECRET, and ODOO_BASE_URL are required.
func Load() (Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return Config{}, fmt.Errorf("PORT must be a number between 1 and 65535")
	}

	migrationsDir := strings.TrimSpace(os.Getenv("MIGRATIONS_DIR"))
	if migrationsDir == "" {
		migrationsDir = defaultMigrationsDir
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) < minimumJWTSecretLen {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least %d characters", minimumJWTSecretLen)
	}

	odooBaseURL := strings.TrimSpace(os.Getenv("ODOO_BASE_URL"))
	if err := validateHTTPBaseURL(odooBaseURL); err != nil {
		return Config{}, fmt.Errorf("ODOO_BASE_URL: %w", err)
	}

	appTimezone := strings.TrimSpace(os.Getenv("APP_TIMEZONE"))
	if appTimezone == "" {
		appTimezone = defaultAppTimezone
	}
	appLocation, err := time.LoadLocation(appTimezone)
	if err != nil {
		return Config{}, fmt.Errorf("APP_TIMEZONE %q is invalid: %w", appTimezone, err)
	}

	seedAdminNIK := strings.TrimSpace(os.Getenv("SEED_ADMIN_NIK"))
	seedAdminName := strings.TrimSpace(os.Getenv("SEED_ADMIN_NAME"))
	if seedAdminName == "" {
		seedAdminName = "Seed Admin"
	}

	odooDB := strings.TrimSpace(os.Getenv("ODOO_DB"))
	odooServiceLogin := strings.TrimSpace(os.Getenv("ODOO_SERVICE_LOGIN"))
	odooServicePassword := os.Getenv("ODOO_SERVICE_PASSWORD")
	odooEmployeeNIKField := strings.TrimSpace(os.Getenv("ODOO_EMPLOYEE_NIK_FIELD"))
	if odooEmployeeNIKField == "" {
		odooEmployeeNIKField = "nik"
	}

	return Config{
		DatabaseURL:     databaseURL,
		Port:            port,
		MigrationsDir:   migrationsDir,
		ShutdownTimeout: shutdownTimeout,

		JWTSecret:   jwtSecret,
		OdooBaseURL: strings.TrimRight(odooBaseURL, "/"),
		OdooTimeout: odooRequestTimeout,

		AppTimezone: appTimezone,
		AppLocation: appLocation,

		SeedAdminNIK:  seedAdminNIK,
		SeedAdminName: seedAdminName,

		OdooDB:               odooDB,
		OdooServiceLogin:     odooServiceLogin,
		OdooServicePassword:  odooServicePassword,
		OdooEmployeeNIKField: odooEmployeeNIKField,
	}, nil
}

// UserSyncEnabled reports whether all worker JSON-RPC settings are present.
func (c Config) UserSyncEnabled() bool {
	return c.OdooDB != "" &&
		c.OdooServiceLogin != "" &&
		c.OdooServicePassword != "" &&
		c.OdooEmployeeNIKField != ""
}

func validateHTTPBaseURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("must be a valid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("host is required")
	}
	return nil
}
