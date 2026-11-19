package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort          = "8080"
	defaultMigrationsDir = "migrations"
	shutdownTimeout      = 10 * time.Second
)

// Config holds the runtime configuration needed by the API and worker.
type Config struct {
	DatabaseURL     string
	Port            string
	MigrationsDir   string
	ShutdownTimeout time.Duration
}

// Load reads configuration from environment variables. Secrets are never
// hardcoded; DATABASE_URL is required for both API and worker.
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

	return Config{
		DatabaseURL:     databaseURL,
		Port:            port,
		MigrationsDir:   migrationsDir,
		ShutdownTimeout: shutdownTimeout,
	}, nil
}
