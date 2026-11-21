package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gsi-workorder/internal/auth"
	"gsi-workorder/internal/candidates"
	"gsi-workorder/internal/config"
	"gsi-workorder/internal/db"
	"gsi-workorder/internal/httpx"
	"gsi-workorder/internal/notifications"
	"gsi-workorder/internal/odoo"
	"gsi-workorder/internal/p2h"
	"gsi-workorder/internal/settings"
	"gsi-workorder/internal/store"
	"gsi-workorder/internal/units"
	"gsi-workorder/internal/users"
	"gsi-workorder/internal/workorders"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("api stopped with error", "error", err)
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

	logger.Info("starting api", "milestone", "m3", "app_timezone", cfg.AppTimezone)

	if err := db.Migrate(cfg.DatabaseURL, cfg.MigrationsDir, logger); err != nil {
		return err
	}

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	queries := store.New(pool)
	checker := httpx.CheckFunc(func(ctx context.Context) error {
		_, err := queries.HealthCheck(ctx)
		if err != nil {
			logger.Warn("health check failed", "error", err)
		}
		return err
	})

	authService := auth.NewService(
		auth.NewPostgresRepository(pool),
		odoo.NewClient(cfg.OdooBaseURL, cfg.OdooTimeout),
		auth.NewTokenManager(cfg.JWTSecret),
		logger,
	)

	if cfg.SeedAdminNIK != "" {
		admin, err := authService.EnsureAdmin(ctx, cfg.SeedAdminNIK, cfg.SeedAdminName)
		if err != nil {
			return fmt.Errorf("seed admin: %w", err)
		}
		logger.Info("seed admin ensured", "user_id", admin.ID)
	}

	unitService := units.NewService(units.NewPostgresRepository(pool), logger)
	candidateService := candidates.NewService(candidates.NewRepository(pool, cfg.AppLocation))
	notificationService := notifications.NewService(notifications.NewRepository(pool))
	settingsService := settings.NewService(settings.NewRepository(pool))
	p2hService := p2h.NewReportService(p2h.NewRepository(pool, cfg.AppLocation))
	workOrderService := workorders.NewRepository(pool, cfg.AppLocation)
	userDirectoryService := users.NewRepository(pool)

	server := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpx.NewRouterWithM2(logger, checker, authService, unitService, httpx.M2Services{
			Candidates:    candidateService,
			Notifications: notificationService,
			Settings:      settingsService,
			P2H:           p2hService,
			WorkOrders:    workOrderService,
			Users:         userDirectoryService,
			UploadDir:     cfg.UploadDir,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("api listening", "addr", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", err)
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	logger.Info("api stopped")
	return nil
}
