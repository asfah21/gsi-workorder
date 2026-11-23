package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"

	"gsi-workorder/internal/config"
	"gsi-workorder/internal/db"
	"gsi-workorder/internal/inventory"
	"gsi-workorder/internal/material"
	"gsi-workorder/internal/odoo"
	"gsi-workorder/internal/p2h"
	"gsi-workorder/internal/pm"
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
		case "sync-p2h":
			if err := runP2HSync(logger, false, os.Args[2:]); err != nil {
				logger.Error("P2H sync failed", "error", err)
				os.Exit(1)
			}
			return
		case "sync-full":
			if err := runP2HSync(logger, true, os.Args[2:]); err != nil {
				logger.Error("full P2H sync failed", "error", err)
				os.Exit(1)
			}
			return
		case "backfill-p2h-readings":
			if err := runP2HReadingsBackfill(logger); err != nil {
				logger.Error("P2H reading backfill failed", "error", err)
				os.Exit(1)
			}
			return
		case "remind-candidates":
			if err := runCandidateReminders(logger); err != nil {
				logger.Error("candidate reminder failed", "error", err)
				os.Exit(1)
			}
			return
		case "evaluate-pm":
			if err := runPMEvaluation(logger); err != nil {
				logger.Error("PM evaluation failed", "error", err)
				os.Exit(1)
			}
			return
		case "alert-stock":
			if err := runStockAlerts(logger); err != nil {
				logger.Error("stock alert failed", "error", err)
				os.Exit(1)
			}
			return
		case "recompute-material":
			if err := runMaterialRecompute(logger); err != nil {
				logger.Error("material recompute failed", "error", err)
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

	if cfg.P2HSyncEnabled() {
		p2hSource := "jsonrpc"
		if cfg.OdooDB == "" || cfg.OdooServiceLogin == "" || cfg.OdooServicePassword == "" {
			p2hSource = "rest"
			logger.Warn("P2H JSON-RPC detail disabled: ODOO_DB or service credentials missing; using REST")
		}
		logger.Info("P2H sync source selected", "source", p2hSource)
		syncService := newP2HSyncService(cfg, pool, logger, p2hSource)
		var p2hLock sync.Mutex
		runP2H := func() {
			if !p2hLock.TryLock() {
				logger.Warn("P2H sync skipped: previous sync still running")
				return
			}
			defer p2hLock.Unlock()

			syncCtx, cancel := context.WithTimeout(ctx, 2*time.Hour)
			defer cancel()
			if err := syncService.Sync(syncCtx, false); err != nil {
				if errors.Is(err, p2h.ErrSyncInProgress) {
					logger.Warn("P2H sync skipped: another sync is running")
				} else {
					logger.Error("P2H sync failed", "error", err)
				}
			}
		}
		go runP2H()
		if _, err := scheduler.AddFunc("@every "+cfg.SyncInterval.String(), runP2H); err != nil {
			return fmt.Errorf("schedule P2H sync: %w", err)
		}
		if _, err := scheduler.AddFunc("0 * * * *", func() {
			reminderCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			if err := syncService.SendReminders(reminderCtx); err != nil {
				logger.Error("P2H reminder failed", "error", err)
			}
		}); err != nil {
			return fmt.Errorf("schedule P2H reminder: %w", err)
		}
		jobs += 2
	} else {
		logger.Warn("P2H sync disabled: ODOO_TOKEN or ODOO_SERVICE_LOGIN/ODOO_SERVICE_PASSWORD is missing")
	}

	pmRepository := pm.NewRepository(pool, cfg.AppLocation)
	if _, err := scheduler.AddFunc("0 7 * * *", func() {
		pmCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
		defer cancel()
		result, err := pmRepository.EvaluateDue(pmCtx, time.Now())
		if err != nil {
			logger.Error("PM evaluation failed", "error", err)
			return
		}
		logger.Info("PM evaluation finished",
			"updated", result.Updated,
			"notified", result.Notified,
			"auto_generated", result.AutoGenerated,
		)
	}); err != nil {
		return fmt.Errorf("schedule PM evaluation: %w", err)
	}
	jobs++

	inventoryRepository := inventory.NewRepository(pool, cfg.AppLocation)
	if _, err := scheduler.AddFunc("30 7 * * *", func() {
		alertCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		result, err := inventoryRepository.SendLowStockAlerts(alertCtx, time.Now())
		if err != nil {
			logger.Error("stock alert failed", "error", err)
			return
		}
		logger.Info("stock alert finished",
			"low", result.Low,
			"critical", result.Critical,
			"notified", result.Notified,
		)
	}); err != nil {
		return fmt.Errorf("schedule stock alert: %w", err)
	}
	jobs++

	scheduler.Start()
	logger.Info("worker started", "jobs", jobs, "milestone", "m6")
	<-ctx.Done()
	logger.Info("worker shutdown signal received")
	return nil
}

func runPMEvaluation(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	result, err := pm.NewRepository(pool, cfg.AppLocation).EvaluateDue(ctx, time.Now())
	if err != nil {
		return err
	}
	logger.Info("PM evaluation finished",
		"updated", result.Updated,
		"notified", result.Notified,
		"auto_generated", result.AutoGenerated,
	)
	return nil
}

func runStockAlerts(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	result, err := inventory.NewRepository(pool, cfg.AppLocation).SendLowStockAlerts(ctx, time.Now())
	if err != nil {
		return err
	}
	logger.Info("stock alert finished",
		"low", result.Low,
		"critical", result.Critical,
		"notified", result.Notified,
	)
	return nil
}

func runMaterialRecompute(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	inv := inventory.NewRepository(pool, cfg.AppLocation)
	result, err := material.NewRepository(pool, inv, cfg.AppLocation).RecomputeAll(ctx)
	if err != nil {
		return err
	}
	logger.Info("material recompute finished", "updated", result.Updated)
	return nil
}

func runP2HReadingsBackfill(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	result, err := p2h.NewRepository(pool, cfg.AppLocation).BackfillReadings(ctx)
	if err != nil {
		return err
	}
	logger.Info("P2H reading backfill finished",
		"reports", result.Reports,
		"readings_written", result.Readings,
		"units_refreshed", result.Units,
		"anomalies", result.Anomalies,
	)
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

func runP2HSync(logger *slog.Logger, forceDetails bool, args []string) error {
	flags := flag.NewFlagSet("p2h-sync", flag.ContinueOnError)
	sourceFlag := flags.String("source", "jsonrpc", "detail source: jsonrpc or rest")
	if err := flags.Parse(args); err != nil {
		return err
	}
	source, err := normalizeP2HSource(*sourceFlag)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if !cfg.P2HSyncEnabled() {
		return fmt.Errorf("P2H sync config incomplete: set ODOO_TOKEN or ODOO_SERVICE_LOGIN/ODOO_SERVICE_PASSWORD")
	}
	if source == "jsonrpc" && (cfg.OdooDB == "" || cfg.OdooServiceLogin == "" || cfg.OdooServicePassword == "") {
		return fmt.Errorf("JSON-RPC detail requires ODOO_DB, ODOO_SERVICE_LOGIN, and ODOO_SERVICE_PASSWORD; use --source=rest to keep the old detail API")
	}
	logger.Info("P2H sync starting", "source", source, "force_details", forceDetails)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	return newP2HSyncService(cfg, pool, logger, source).Sync(ctx, forceDetails)
}

func normalizeP2HSource(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "jsonrpc":
		return "jsonrpc", nil
	case "rest":
		return "rest", nil
	default:
		return "", fmt.Errorf("invalid --source %q: use jsonrpc or rest", raw)
	}
}

func runCandidateReminders(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	return newP2HSyncService(cfg, pool, logger, "rest").SendReminders(ctx)
}

func newP2HSyncService(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger, source string) *p2h.SyncService {
	client := p2h.NewClient(
		cfg.OdooBaseURL,
		cfg.OdooTimeout,
		cfg.OdooToken,
		cfg.OdooServiceLogin,
		cfg.OdooServicePassword,
	)
	repo := p2h.NewRepository(pool, cfg.AppLocation)

	var detail p2h.DetailLoader
	if source == "jsonrpc" {
		detail = p2h.NewJSONRPCClient(
			cfg.OdooBaseURL,
			cfg.OdooDB,
			cfg.OdooServiceLogin,
			cfg.OdooServicePassword,
			120*time.Second,
		)
	}
	return p2h.NewSyncServiceWithDetail(client, detail, repo, cfg.SyncConcurrency, cfg.AppLocation, logger)
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
