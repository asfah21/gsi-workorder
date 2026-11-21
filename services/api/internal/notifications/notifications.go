package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Notification struct {
	ID        int64      `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	RefType   string     `json:"ref_type"`
	RefID     *int64     `json:"ref_id,omitempty"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateInput struct {
	Type    string
	Title   string
	Body    string
	RefType string
	RefID   *int64
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateForRoles inserts one in-app notification for every active user in the
// supplied roles. It is used inside the same transaction as the candidate/WO
// change so a notification is never written for a rolled-back decision.
func CreateForRoles(
	ctx context.Context,
	tx pgx.Tx,
	roles []string,
	input CreateInput,
	excludeUserID *int64,
) (int, error) {
	rows, err := tx.Query(ctx, `
		SELECT id
		FROM users
		WHERE is_active = true
		  AND role = ANY($1)
		  AND ($2::bigint IS NULL OR id <> $2)
	`, roles, excludeUserID)
	if err != nil {
		return 0, fmt.Errorf("list notification recipients: %w", err)
	}
	defer rows.Close()

	userIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("scan notification recipient: %w", err)
		}
		userIDs = append(userIDs, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate notification recipients: %w", err)
	}
	return CreateForUsers(ctx, tx, userIDs, input)
}

func CreateForUsers(ctx context.Context, tx pgx.Tx, userIDs []int64, input CreateInput) (int, error) {
	unique := make(map[int64]struct{}, len(userIDs))
	ids := make([]int64, 0, len(userIDs))
	for _, id := range userIDs {
		if id <= 0 {
			continue
		}
		if _, exists := unique[id]; exists {
			continue
		}
		unique[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, userID := range ids {
		batch.Queue(`
			INSERT INTO notifications (user_id, type, title, body, ref_type, ref_id)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, userID, input.Type, input.Title, input.Body, input.RefType, input.RefID)
	}
	results := tx.SendBatch(ctx, batch)
	for range ids {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return 0, fmt.Errorf("insert notification: %w", err)
		}
	}
	if err := results.Close(); err != nil {
		return 0, fmt.Errorf("close notification batch: %w", err)
	}
	return len(ids), nil
}

func (r *Repository) List(ctx context.Context, userID int64, page, limit int, unreadOnly bool) ([]Notification, int, int, error) {
	filter := ""
	if unreadOnly {
		filter = " AND read_at IS NULL"
	}

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM notifications
		WHERE user_id = $1`+filter, userID).Scan(&total); err != nil {
		return nil, 0, 0, fmt.Errorf("count notifications: %w", err)
	}

	var unread int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM notifications
		WHERE user_id = $1 AND read_at IS NULL
	`, userID).Scan(&unread); err != nil {
		return nil, 0, 0, fmt.Errorf("count unread notifications: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, COALESCE(type, ''), COALESCE(title, ''), COALESCE(body, ''),
		       COALESCE(ref_type, ''), ref_id, read_at, created_at
		FROM notifications
		WHERE user_id = $1`+filter+`
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	items := make([]Notification, 0)
	for rows.Next() {
		var item Notification
		if err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Body,
			&item.RefType, &item.RefID, &item.ReadAt, &item.CreatedAt); err != nil {
			return nil, 0, 0, fmt.Errorf("scan notification: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("iterate notifications: %w", err)
	}
	return items, total, unread, nil
}

func (r *Repository) MarkRead(ctx context.Context, userID int64, ids []int64, all bool) (int64, error) {
	var updated int64
	if all {
		tag, err := r.pool.Exec(ctx, `
			UPDATE notifications
			SET read_at = now()
			WHERE user_id = $1 AND read_at IS NULL
		`, userID)
		if err != nil {
			return 0, fmt.Errorf("mark notifications read: %w", err)
		}
		updated = tag.RowsAffected()
	} else {
		tag, err := r.pool.Exec(ctx, `
			UPDATE notifications
			SET read_at = now()
			WHERE user_id = $1
			  AND read_at IS NULL
			  AND id = ANY($2)
		`, userID, ids)
		if err != nil {
			return 0, fmt.Errorf("mark notifications read: %w", err)
		}
		updated = tag.RowsAffected()
	}
	return updated, nil
}
