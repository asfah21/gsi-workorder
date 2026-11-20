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
	"gsi-workorder/internal/config"
	"gsi-workorder/internal/db"
	"gsi-workorder/internal/httpx"
	"gsi-workorder/internal/odoo"
	"gsi-workorder/internal/store"
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

	logger.Info("starting api", "milestone", "m1a", "app_timezone", cfg.AppTimezone)

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

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpx.NewRouter(logger, checker, authService),
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
