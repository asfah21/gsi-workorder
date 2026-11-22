package inventory

import (
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
	loc  *time.Location
}

func NewRepository(pool *pgxpool.Pool, loc *time.Location) *Repository {
	if loc == nil {
		loc = time.UTC
	}
	return &Repository{pool: pool, loc: loc}
}

func canManageMaster(role string) bool {
	return role == "admin" || role == "planner"
}

func canMoveStock(role string) bool {
	return role == "admin" || role == "storekeeper"
}

func canApproveStock(role string) bool {
	return role == "admin" || role == "planner"
}

func canCountStock(role string) bool {
	return role == "admin" || role == "storekeeper"
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

func trim(value string) string {
	return strings.TrimSpace(value)
}
