package users

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID     int64  `json:"id"`
	NIK    string `json:"nik"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Active bool   `json:"is_active"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListActive returns active users in the requested roles. It is used by the
// work-order assignment and supervisor selectors.
func (r *Repository) ListActive(ctx context.Context, roles []string, query string, limit int) ([]User, int, error) {
	if len(roles) == 0 {
		roles = []string{"admin", "planner", "mechanic"}
	}
	if limit < 1 {
		limit = 100
	}
	if limit > 2000 {
		limit = 2000
	}

	args := []any{roles}
	where := []string{"is_active = true", "role = ANY($1)"}
	if query != "" {
		args = append(args, "%"+query+"%")
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR nik ILIKE $%d)", len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	args = append(args, limit)
	rows, err := r.pool.Query(ctx, `
		SELECT id, nik, COALESCE(name, ''), role, is_active
		FROM users
		WHERE `+whereSQL+`
		ORDER BY name ASC, id ASC
		LIMIT $`+fmt.Sprint(len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	items := make([]User, 0)
	for rows.Next() {
		var item User
		if err := rows.Scan(&item.ID, &item.NIK, &item.Name, &item.Role, &item.Active); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate users: %w", err)
	}
	return items, total, nil
}
