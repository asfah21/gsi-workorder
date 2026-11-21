package settings

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	KeyP2HAutoWO             = "p2h_auto_wo"
	KeyCandidateSLAHours     = "candidate_sla_hours"
	KeyCandidateBackfillDays = "candidate_backfill_days"

	defaultCandidateSLAHours     = 24
	defaultCandidateBackfillDays = 30
)

var (
	ErrValidation = errors.New("settings: validation failed")
	ErrNotFound   = errors.New("settings: not found")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "settings: validation failed" }
func (e *ValidationError) Unwrap() error { return ErrValidation }

type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type History struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	ChangedBy *int64    `json:"changed_by,omitempty"`
	ChangedAt time.Time `json:"changed_at"`
}

type Snapshot struct {
	P2HAutoWO             bool      `json:"p2h_auto_wo"`
	CandidateSLAHours     int       `json:"candidate_sla_hours"`
	CandidateBackfillDays int       `json:"candidate_backfill_days"`
	UpdatedAt             time.Time `json:"updated_at"`
	History               []History `json:"history"`
}

type Patch struct {
	P2HAutoWO             *bool
	CandidateSLAHours     *int
	CandidateBackfillDays *int
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Get(ctx context.Context) (Snapshot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT key, COALESCE(value, ''), COALESCE(updated_at, now())
		FROM app_settings
		WHERE key = ANY($1)
	`, []string{KeyP2HAutoWO, KeyCandidateSLAHours, KeyCandidateBackfillDays})
	if err != nil {
		return Snapshot{}, fmt.Errorf("get settings: %w", err)
	}
	defer rows.Close()

	values := map[string]Setting{}
	for rows.Next() {
		var item Setting
		if err := rows.Scan(&item.Key, &item.Value, &item.UpdatedAt); err != nil {
			return Snapshot{}, fmt.Errorf("scan setting: %w", err)
		}
		values[item.Key] = item
	}
	if err := rows.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("iterate settings: %w", err)
	}

	snapshot := Snapshot{
		P2HAutoWO:             parseBool(values[KeyP2HAutoWO].Value, false),
		CandidateSLAHours:     parseInt(values[KeyCandidateSLAHours].Value, defaultCandidateSLAHours),
		CandidateBackfillDays: parseInt(values[KeyCandidateBackfillDays].Value, defaultCandidateBackfillDays),
		UpdatedAt:             latestTime(values),
	}
	history, err := r.History(ctx, 50)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.History = history
	return snapshot, nil
}

func (r *Repository) History(ctx context.Context, limit int) ([]History, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, COALESCE(key, ''), COALESCE(old_value, ''), COALESCE(new_value, ''),
		       changed_by, changed_at
		FROM app_settings_history
		ORDER BY changed_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list settings history: %w", err)
	}
	defer rows.Close()

	items := make([]History, 0)
	for rows.Next() {
		var item History
		if err := rows.Scan(&item.ID, &item.Key, &item.OldValue, &item.NewValue,
			&item.ChangedBy, &item.ChangedAt); err != nil {
			return nil, fmt.Errorf("scan settings history: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settings history: %w", err)
	}
	return items, nil
}

func (r *Repository) Update(ctx context.Context, actorID int64, patch Patch) (Snapshot, error) {
	changes, fields := normalizePatch(patch)
	if len(fields) > 0 {
		return Snapshot{}, &ValidationError{Fields: fields}
	}
	if len(changes) == 0 {
		return r.Get(ctx)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("begin settings update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, change := range changes {
		var oldValue string
		err := tx.QueryRow(ctx, `
			SELECT COALESCE(value, '')
			FROM app_settings
			WHERE key = $1
			FOR UPDATE
		`, change.Key).Scan(&oldValue)
		if errors.Is(err, pgx.ErrNoRows) {
			oldValue = ""
		} else if err != nil {
			return Snapshot{}, fmt.Errorf("lock setting %s: %w", change.Key, err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO app_settings (key, value, updated_by, updated_at)
			VALUES ($1, $2, $3, now())
			ON CONFLICT (key) DO UPDATE
			SET value = EXCLUDED.value,
			    updated_by = EXCLUDED.updated_by,
			    updated_at = now()
		`, change.Key, change.Value, actorID); err != nil {
			return Snapshot{}, fmt.Errorf("update setting %s: %w", change.Key, err)
		}
		if oldValue == change.Value {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO app_settings_history (key, old_value, new_value, changed_by)
			VALUES ($1, $2, $3, $4)
		`, change.Key, oldValue, change.Value, actorID); err != nil {
			return Snapshot{}, fmt.Errorf("write setting history %s: %w", change.Key, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Snapshot{}, fmt.Errorf("commit settings update: %w", err)
	}
	return r.Get(ctx)
}

type change struct {
	Key   string
	Value string
}

func normalizePatch(patch Patch) ([]change, map[string]string) {
	fields := map[string]string{}
	changes := make([]change, 0, 3)
	if patch.P2HAutoWO != nil {
		changes = append(changes, change{Key: KeyP2HAutoWO, Value: strconv.FormatBool(*patch.P2HAutoWO)})
	}
	if patch.CandidateSLAHours != nil {
		if *patch.CandidateSLAHours < 1 || *patch.CandidateSLAHours > 168 {
			fields["candidate_sla_hours"] = "Harus antara 1 dan 168 jam."
		} else {
			changes = append(changes, change{Key: KeyCandidateSLAHours, Value: strconv.Itoa(*patch.CandidateSLAHours)})
		}
	}
	if patch.CandidateBackfillDays != nil {
		if *patch.CandidateBackfillDays < 0 || *patch.CandidateBackfillDays > 3650 {
			fields["candidate_backfill_days"] = "Harus antara 0 dan 3650 hari."
		} else {
			changes = append(changes, change{Key: KeyCandidateBackfillDays, Value: strconv.Itoa(*patch.CandidateBackfillDays)})
		}
	}
	return changes, fields
}

func parseBool(raw string, fallback bool) bool {
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func parseInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func latestTime(values map[string]Setting) time.Time {
	var latest time.Time
	for _, item := range values {
		if item.UpdatedAt.After(latest) {
			latest = item.UpdatedAt
		}
	}
	return latest
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context) (Snapshot, error) {
	return s.repo.Get(ctx)
}

func (s *Service) Update(ctx context.Context, actorID int64, patch Patch) (Snapshot, error) {
	if actorID <= 0 {
		return Snapshot{}, ErrValidation
	}
	return s.repo.Update(ctx, actorID, patch)
}
