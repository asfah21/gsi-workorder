package db

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migrate applies all pending SQL migrations. It is called by the API at
// startup; the worker never runs migrations.
func Migrate(databaseURL, migrationsDir string, logger *slog.Logger) error {
	sourceURL, err := fileSourceURL(migrationsDir)
	if err != nil {
		return err
	}

	runner, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("create migration runner: %s", sanitizeDatabaseError(err, databaseURL))
	}
	defer func() {
		sourceErr, databaseErr := runner.Close()
		if sourceErr != nil {
			logger.Warn("close migration source", "error", sourceErr)
		}
		if databaseErr != nil {
			logger.Warn("close migration database", "error", databaseErr)
		}
	}()

	version, dirty, err := runner.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read migration version: %s", sanitizeDatabaseError(err, databaseURL))
	}
	if dirty {
		return fmt.Errorf("database is dirty at migration version %d", version)
	}
	logger.Info("database migration state", "version", version, "dirty", dirty)

	if err := runner.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("database schema is up to date")
			return nil
		}
		return fmt.Errorf("apply database migrations: %s", sanitizeDatabaseError(err, databaseURL))
	}

	version, dirty, err = runner.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read migration version after apply: %s", sanitizeDatabaseError(err, databaseURL))
	}
	logger.Info("database migrations applied", "version", version, "dirty", dirty)
	return nil
}

func fileSourceURL(dir string) (string, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve migrations directory: %w", err)
	}

	// On Linux this becomes file:///app/migrations. On Windows the drive
	// letter becomes the URL host, for example file://D:/project/migrations,
	// which is what golang-migrate expects.
	return "file://" + filepath.ToSlash(absolute), nil
}
