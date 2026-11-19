package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	connectAttempts = 10
	connectDelay    = 2 * time.Second
	pingTimeout     = 3 * time.Second
)

// NewPool creates a pgx pool and waits for the database to accept connections.
// The retry loop keeps startup resilient while Docker Compose initializes
// PostgreSQL.
func NewPool(ctx context.Context, databaseURL string, logger *slog.Logger) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %s", sanitizeDatabaseError(err, databaseURL))
	}

	var lastErr error
	for attempt := 1; attempt <= connectAttempts; attempt++ {
		pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
			err = pool.Ping(pingCtx)
			cancel()
			if err == nil {
				logger.Info("database connected", "attempt", attempt)
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		logger.Warn("database not ready", "attempt", attempt, "error", sanitizeDatabaseError(err, databaseURL))

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(connectDelay):
		}
	}

	return nil, fmt.Errorf("connect to database after %d attempts: %s", connectAttempts, sanitizeDatabaseError(lastErr, databaseURL))
}
