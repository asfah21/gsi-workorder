package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"

	"gsi-workorder/internal/config"
	"gsi-workorder/internal/db"
	"gsi-workorder/internal/odoo"
	"gsi-workorder/internal/usersync"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "health":
			if err := health(logger); err != nil {
				logger.Error("worker health check failed", "error", err)
				os.Exit(1)
			}
			return
		case "run":
			// Fall through to the default run mode.
		case "sync-users":
			if err := runUserSync(logger); err != nil {
				logger.Error("user sync failed", "error", err)
				os.Exit(1)
			}
			return
		default:
			logger.Error("unknown worker command", "command", os.Args[1])
			os.Exit(2)
		}
	}

	if err := run(logger); err != nil {
		logger.Error("worker stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	scheduler := cron.New(cron.WithLocation(cfg.AppLocation))
	defer scheduler.Stop()

	jobs := 0
	if cfg.UserSyncEnabled() {
		syncService := newUserSyncService(cfg, pool, logger)
		var syncLock sync.Mutex
		runSync := func() {
			if !syncLock.TryLock() {
				logger.Warn("user sync skipped: previous sync still running")
				return
			}
			defer syncLock.Unlock()

			syncCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			if err := syncService.Sync(syncCtx); err != nil {
				logger.Error("user sync failed", "error", err)
			}
		}

		go runSync()
		if _, err := scheduler.AddFunc("0 */6 * * *", runSync); err != nil {
			return fmt.Errorf("schedule user sync: %w", err)
		}
		jobs = 1
	} else {
		logger.Warn("user sync disabled: ODOO_DB, ODOO_SERVICE_LOGIN, ODOO_SERVICE_PASSWORD, or ODOO_EMPLOYEE_NIK_FIELD is missing")
	}

	scheduler.Start()
	logger.Info("worker started", "jobs", jobs, "milestone", "m1b")
	<-ctx.Done()
	logger.Info("worker shutdown signal received")
	return nil
}

func runUserSync(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if !cfg.UserSyncEnabled() {
		return fmt.Errorf("user sync config incomplete: set ODOO_DB, ODOO_SERVICE_LOGIN, ODOO_SERVICE_PASSWORD, and ODOO_EMPLOYEE_NIK_FIELD")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	return newUserSyncService(cfg, pool, logger).Sync(ctx)
}

func newUserSyncService(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) *usersync.Service {
	return usersync.NewService(
		odoo.NewClient(cfg.OdooBaseURL, cfg.OdooTimeout),
		usersync.NewPostgresRepository(pool),
		cfg.OdooDB,
		cfg.OdooServiceLogin,
		cfg.OdooServicePassword,
		cfg.OdooEmployeeNIKField,
		logger,
	)
}

func health(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
