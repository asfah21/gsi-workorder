package usersync

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) UpsertUser(ctx context.Context, user OdooUser, fallbackNIK string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE odoo_user_id = $1
	`, user.OdooID).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
		err = r.pool.QueryRow(ctx, `
			SELECT id
			FROM users
			WHERE lower(btrim(nik)) = lower(btrim($1))
			LIMIT 1
		`, user.Login).Scan(&id)
	}
	if errors.Is(err, pgx.ErrNoRows) && strings.TrimSpace(fallbackNIK) != "" {
		err = r.pool.QueryRow(ctx, `
			SELECT id
			FROM users
			WHERE lower(btrim(nik)) = lower(btrim($1))
			LIMIT 1
		`, fallbackNIK).Scan(&id)
	}

	switch {
	case err == nil:
		_, err = r.pool.Exec(ctx, `
			UPDATE users
			SET odoo_user_id = $2,
			    name = $3,
			    email = $4,
			    is_active = $5,
			    synced_at = now(),
			    nik = CASE WHEN nik IS NULL OR btrim(nik) = '' THEN btrim($6) ELSE nik END
			WHERE id = $1
		`, id, user.OdooID, user.Name, user.Email, user.Active, user.Login)
		if err != nil {
			return 0, fmt.Errorf("update user %d: %w", id, err)
		}
		return id, nil
	case errors.Is(err, pgx.ErrNoRows):
		err = r.pool.QueryRow(ctx, `
			INSERT INTO users (odoo_user_id, nik, name, email, role, is_active, synced_at)
			VALUES ($1, btrim($2), $3, $4, 'mechanic', $5, now())
			RETURNING id
		`, user.OdooID, strings.TrimSpace(user.Login), user.Name, user.Email, user.Active).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("insert user %d: %w", user.OdooID, err)
		}
		return id, nil
	default:
		return 0, fmt.Errorf("find local user for Odoo user %d: %w", user.OdooID, err)
	}
}

func (r *PostgresRepository) UpsertEmployee(ctx context.Context, employee OdooEmployee, localUserID *int64) error {
	var userID any
	if localUserID != nil {
		userID = *localUserID
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO employees (
			odoo_employee_id, nik, name, job_title, department, site_location,
			user_id, is_active, synced_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		ON CONFLICT (odoo_employee_id) DO UPDATE
		SET nik = EXCLUDED.nik,
		    name = EXCLUDED.name,
		    job_title = EXCLUDED.job_title,
		    department = EXCLUDED.department,
		    site_location = EXCLUDED.site_location,
		    user_id = EXCLUDED.user_id,
		    is_active = EXCLUDED.is_active,
		    synced_at = now()
	`, employee.OdooID, employee.NIK, employee.Name, employee.JobTitle,
		employee.Department, employee.SiteLocation, userID, employee.Active)
	if err != nil {
		return fmt.Errorf("upsert employee %d: %w", employee.OdooID, err)
	}
	return nil
}

func (r *PostgresRepository) StartSyncRun(ctx context.Context, kind string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO sync_runs (kind, status, started_at)
		VALUES ($1, 'running', now())
		RETURNING id
	`, kind).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert sync run: %w", err)
	}
	return id, nil
}

func (r *PostgresRepository) FinishSyncRun(ctx context.Context, id int64, status, errorMessage string) error {
	var errorValue any
	if strings.TrimSpace(errorMessage) != "" {
		errorValue = errorMessage
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE sync_runs
		SET status = $2, finished_at = now(), error = $3
		WHERE id = $1
	`, id, status, errorValue)
	if err != nil {
		return fmt.Errorf("update sync run: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SetSyncState(ctx context.Context, key, value string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sync_state (key, value, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = now()
	`, key, value)
	if err != nil {
		return fmt.Errorf("upsert sync state %q: %w", key, err)
	}
	return nil
}
