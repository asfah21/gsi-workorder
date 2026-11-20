package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using the M0 users table plus the
// M1A normalized-NIK unique index.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) FindByNIK(ctx context.Context, nik string) (User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, COALESCE(nik, ''), COALESCE(name, ''), COALESCE(email, ''),
		       role, is_active
		FROM users
		WHERE lower(btrim(nik)) = lower(btrim($1))
		LIMIT 1
	`, nik)
	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("query user by nik: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id int64) (User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, COALESCE(nik, ''), COALESCE(name, ''), COALESCE(email, ''),
		       role, is_active
		FROM users
		WHERE id = $1
	`, id)
	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("query user by id: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) TouchLogin(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("update last_login_at: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdateRole(ctx context.Context, id int64, role string) (User, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE users
		SET role = $2
		WHERE id = $1
		RETURNING id, COALESCE(nik, ''), COALESCE(name, ''), COALESCE(email, ''),
		          role, is_active
	`, id, role)
	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("update user role: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) EnsureAdmin(ctx context.Context, nik, name string) (User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin seed admin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Bootstrap must not keep re-promoting the seed user on every restart.
	// If an active admin already exists, leave roles untouched.
	var existingID int64
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE role = 'admin' AND is_active
		ORDER BY id
		LIMIT 1
		FOR UPDATE
	`).Scan(&existingID)
	if err == nil {
		row := tx.QueryRow(ctx, `
			SELECT id, COALESCE(nik, ''), COALESCE(name, ''), COALESCE(email, ''),
			       role, is_active
			FROM users
			WHERE id = $1
		`, existingID)
		user, scanErr := scanUser(row)
		if scanErr != nil {
			return User{}, fmt.Errorf("read existing admin: %w", scanErr)
		}
		if err := tx.Commit(ctx); err != nil {
			return User{}, fmt.Errorf("commit seed admin check: %w", err)
		}
		return user, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, fmt.Errorf("check existing admin: %w", err)
	}

	var id int64
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE lower(btrim(nik)) = lower(btrim($1))
		FOR UPDATE
	`, nik).Scan(&id)
	switch {
	case err == nil:
		row := tx.QueryRow(ctx, `
			UPDATE users
			SET role = 'admin',
			    is_active = true,
			    name = CASE WHEN btrim($2) = '' THEN name ELSE $2 END,
			    synced_at = now()
			WHERE id = $1
			RETURNING id, COALESCE(nik, ''), COALESCE(name, ''), COALESCE(email, ''),
			          role, is_active
		`, id, name)
		user, scanErr := scanUser(row)
		if scanErr != nil {
			return User{}, fmt.Errorf("promote seed admin: %w", scanErr)
		}
		if err := tx.Commit(ctx); err != nil {
			return User{}, fmt.Errorf("commit seed admin: %w", err)
		}
		return user, nil
	case errors.Is(err, pgx.ErrNoRows):
		row := tx.QueryRow(ctx, `
			INSERT INTO users (nik, name, role, is_active, synced_at)
			VALUES (btrim($1), $2, 'admin', true, now())
			RETURNING id, COALESCE(nik, ''), COALESCE(name, ''), COALESCE(email, ''),
			          role, is_active
		`, nik, name)
		user, scanErr := scanUser(row)
		if scanErr != nil {
			return User{}, fmt.Errorf("insert seed admin: %w", scanErr)
		}
		if err := tx.Commit(ctx); err != nil {
			return User{}, fmt.Errorf("commit seed admin: %w", err)
		}
		return user, nil
	default:
		return User{}, fmt.Errorf("find seed admin: %w", err)
	}
}

func scanUser(row pgx.Row) (User, error) {
	var user User
	err := row.Scan(&user.ID, &user.NIK, &user.Name, &user.Email, &user.Role, &user.IsActive)
	if err != nil {
		return User{}, err
	}
	return user, nil
}
