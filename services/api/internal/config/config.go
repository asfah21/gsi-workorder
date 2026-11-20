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
	}, nil
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
