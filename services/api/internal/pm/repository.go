package pm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/notifications"
	"gsi-workorder/internal/pmcalc"
	"gsi-workorder/internal/settings"
	"gsi-workorder/internal/workorders"
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

const planSelect = `
	SELECT p.id, p.name, COALESCE(p.description, ''), p.unit_id, COALESCE(u.code, ''),
	       p.unit_model_id,
	       TRIM(COALESCE(um.manufacturer, '') || CASE WHEN um.model IS NULL THEN '' ELSE ' ' || um.model END),
	       p.status, p.effective_from, p.effective_to,
	       p.interval_hm, p.interval_km, p.interval_days, p.interval_rule,
	       p.upcoming_days, p.due_soon_days, p.upcoming_hm, p.upcoming_km,
	       p.due_soon_hm, p.due_soon_km, p.auto_generate_wo, p.default_priority,
	       p.default_assignee_id, COALESCE(da.name, ''),
	       p.default_supervisor_id, COALESCE(ds.name, ''),
	       p.created_by, COALESCE(cb.name, ''), p.created_at, p.updated_at
	FROM pm_plans p
	LEFT JOIN units u ON u.id = p.unit_id
	LEFT JOIN unit_models um ON um.id = p.unit_model_id
	LEFT JOIN users da ON da.id = p.default_assignee_id
	LEFT JOIN users ds ON ds.id = p.default_supervisor_id
	LEFT JOIN users cb ON cb.id = p.created_by
`

func (r *Repository) ListPlans(ctx context.Context, filter PlanFilter) ([]Plan, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 25
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	args := make([]any, 0, 6)
	where := []string{"1 = 1"}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("p.status = $%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		where = append(where, fmt.Sprintf(`(p.name ILIKE $%d OR COALESCE(p.description, '') ILIKE $%d
			OR COALESCE(u.code, '') ILIKE $%d OR COALESCE(um.model, '') ILIKE $%d)`,
			len(args), len(args), len(args), len(args)))
	}
	if filter.UnitID != nil {
		args = append(args, *filter.UnitID)
		where = append(where, fmt.Sprintf("p.unit_id = $%d", len(args)))
	}
	if filter.UnitModelID != nil {
		args = append(args, *filter.UnitModelID)
		where = append(where, fmt.Sprintf("p.unit_model_id = $%d", len(args)))
	}

	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM pm_plans p
		LEFT JOIN units u ON u.id = p.unit_id
		LEFT JOIN unit_models um ON um.id = p.unit_model_id
		WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count PM plans: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	orderColumn := planSortColumn(filter.Sort)
	orderDirection := "ASC"
	if strings.EqualFold(filter.Order, "desc") {
		orderDirection = "DESC"
	}
	rows, err := r.pool.Query(ctx, planSelect+`
		WHERE `+whereSQL+`
		ORDER BY `+orderColumn+` `+orderDirection+`, p.id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list PM plans: %w", err)
	}
	defer rows.Close()

	items := make([]Plan, 0)
	for rows.Next() {
		item, err := scanPlan(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan PM plan: %w", err)
		}
		checklist, err := r.checklist(ctx, item.ID)
		if err != nil {
			return nil, 0, err
		}
		item.Checklist = checklist
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate PM plans: %w", err)
	}
	return items, total, nil
}

func (r *Repository) GetPlan(ctx context.Context, id int64) (Plan, error) {
	item, err := scanPlan(r.pool.QueryRow(ctx, planSelect+` WHERE p.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrNotFound
	}
	if err != nil {
		return Plan{}, fmt.Errorf("get PM plan: %w", err)
	}
	checklist, err := r.checklist(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	item.Checklist = checklist
	return item, nil
}

func (r *Repository) CreatePlan(ctx context.Context, actor Actor, input PlanInput) (Plan, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return Plan{}, ErrValidation
	}
	normalized, err := normalizePlanInput(input)
	if err != nil {
		return Plan{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("begin PM plan create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := validatePlanTarget(ctx, tx, normalized.UnitID, normalized.UnitModelID); err != nil {
		return Plan{}, err
	}
	if err := validatePlanUsers(ctx, tx, normalized.DefaultAssigneeID, normalized.DefaultSupervisorID); err != nil {
		return Plan{}, err
	}

	var id int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO pm_plans (
			name, description, unit_id, unit_model_id, status, effective_from, effective_to,
			interval_hm, interval_km, interval_days, interval_rule,
			upcoming_days, due_soon_days, upcoming_hm, upcoming_km, due_soon_hm, due_soon_km,
			auto_generate_wo, default_priority, default_assignee_id, default_supervisor_id,
			created_by, updated_by
		)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7, $8, $9, $10, $11,
		        $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $22)
		RETURNING id
	`, normalized.Name, normalized.Description, normalized.UnitID, normalized.UnitModelID,
		normalized.Status, dateValue(normalized.EffectiveFrom), nullDate(normalized.EffectiveTo),
		normalized.IntervalHM, normalized.IntervalKM, normalized.IntervalDays, normalized.IntervalRule,
		normalized.UpcomingDays, normalized.DueSoonDays, normalized.UpcomingHM, normalized.UpcomingKM,
		normalized.DueSoonHM, normalized.DueSoonKM, normalized.AutoGenerateWO,
		normalized.DefaultPriority, normalized.DefaultAssigneeID, normalized.DefaultSupervisorID,
		actor.ID).Scan(&id); err != nil {
		return Plan{}, fmt.Errorf("insert PM plan: %w", err)
	}
	if err := replaceChecklist(ctx, tx, id, normalized.Checklist); err != nil {
		return Plan{}, err
	}
	if normalized.Status == "active" {
		if err := ensurePlanSchedulesTx(ctx, tx, id, r.loc); err != nil {
			return Plan{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Plan{}, fmt.Errorf("commit PM plan create: %w", err)
	}
	return r.GetPlan(ctx, id)
}

func (r *Repository) UpdatePlan(ctx context.Context, id int64, actor Actor, patch PlanPatch) (Plan, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return Plan{}, ErrValidation
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("begin PM plan update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var locked int64
	if err := tx.QueryRow(ctx, `SELECT id FROM pm_plans WHERE id = $1 FOR UPDATE`, id).Scan(&locked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Plan{}, ErrNotFound
		}
		return Plan{}, fmt.Errorf("lock PM plan: %w", err)
	}
	current, err := r.getPlanTx(ctx, tx, id)
	if err != nil {
		return Plan{}, err
	}
	next := current
	if patch.Name != nil {
		next.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.Description != nil {
		next.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.Status != nil {
		next.Status = strings.TrimSpace(*patch.Status)
	}
	if patch.EffectiveFrom != nil {
		next.EffectiveFrom = *patch.EffectiveFrom
	}
	if patch.ClearEffectiveTo {
		next.EffectiveTo = nil
	} else if patch.EffectiveTo != nil {
		next.EffectiveTo = patch.EffectiveTo
	}
	applyNullableFloat(&next.IntervalHM, patch.IntervalHM, patch.ClearIntervalHM)
	applyNullableFloat(&next.IntervalKM, patch.IntervalKM, patch.ClearIntervalKM)
	applyNullableInt(&next.IntervalDays, patch.IntervalDays, patch.ClearIntervalDays)
	if patch.IntervalRule != nil {
		next.IntervalRule = strings.TrimSpace(*patch.IntervalRule)
	}
	applyNullableInt(&next.UpcomingDays, patch.UpcomingDays, patch.ClearUpcomingDays)
	applyNullableInt(&next.DueSoonDays, patch.DueSoonDays, patch.ClearDueSoonDays)
	applyNullableFloat(&next.UpcomingHM, patch.UpcomingHM, patch.ClearUpcomingHM)
	applyNullableFloat(&next.UpcomingKM, patch.UpcomingKM, patch.ClearUpcomingKM)
	applyNullableFloat(&next.DueSoonHM, patch.DueSoonHM, patch.ClearDueSoonHM)
	applyNullableFloat(&next.DueSoonKM, patch.DueSoonKM, patch.ClearDueSoonKM)
	if patch.AutoGenerateWO != nil {
		next.AutoGenerateWO = *patch.AutoGenerateWO
	}
	if patch.DefaultPriority != nil {
		next.DefaultPriority = strings.TrimSpace(*patch.DefaultPriority)
	}
	if patch.ClearDefaultAssignee {
		next.DefaultAssigneeID = nil
	} else if patch.DefaultAssigneeID != nil {
		next.DefaultAssigneeID = patch.DefaultAssigneeID
	}
	if patch.ClearDefaultSupervisor {
		next.DefaultSupervisorID = nil
	} else if patch.DefaultSupervisorID != nil {
		next.DefaultSupervisorID = patch.DefaultSupervisorID
	}
	if !validStatus(next.Status) {
		return Plan{}, &ValidationError{Fields: map[string]string{"status": "Status plan tidak valid."}}
	}
	if next.Status == "active" && current.Status != "active" {
		if err := validatePlanReadiness(next); err != nil {
			return Plan{}, err
		}
	}
	if err := validatePlanValues(next); err != nil {
		return Plan{}, err
	}
	if err := validatePlanUsers(ctx, tx, next.DefaultAssigneeID, next.DefaultSupervisorID); err != nil {
		return Plan{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE pm_plans
		SET name = $2, description = NULLIF($3, ''), status = $4,
		    effective_from = $5, effective_to = $6,
		    interval_hm = $7, interval_km = $8, interval_days = $9, interval_rule = $10,
		    upcoming_days = $11, due_soon_days = $12,
		    upcoming_hm = $13, upcoming_km = $14, due_soon_hm = $15, due_soon_km = $16,
		    auto_generate_wo = $17, default_priority = $18,
		    default_assignee_id = $19, default_supervisor_id = $20,
		    updated_by = $21, updated_at = now()
		WHERE id = $1
	`, id, next.Name, next.Description, next.Status, dateValue(next.EffectiveFrom),
		nullDate(next.EffectiveTo), next.IntervalHM, next.IntervalKM, next.IntervalDays,
		next.IntervalRule, next.UpcomingDays, next.DueSoonDays, next.UpcomingHM,
		next.UpcomingKM, next.DueSoonHM, next.DueSoonKM, next.AutoGenerateWO,
		next.DefaultPriority, next.DefaultAssigneeID, next.DefaultSupervisorID, actor.ID); err != nil {
		return Plan{}, fmt.Errorf("update PM plan: %w", err)
	}
	if patch.Checklist != nil {
		normalized := PlanInput{Checklist: *patch.Checklist}
		for index := range normalized.Checklist {
			normalized.Checklist[index].Task = strings.TrimSpace(normalized.Checklist[index].Task)
			normalized.Checklist[index].Standard = strings.TrimSpace(normalized.Checklist[index].Standard)
			if normalized.Checklist[index].Sequence <= 0 {
				normalized.Checklist[index].Sequence = index + 1
			}
			if normalized.Checklist[index].Task == "" || len(normalized.Checklist[index].Task) > 500 {
				return Plan{}, &ValidationError{Fields: map[string]string{"checklist": "Task checklist wajib 1-500 karakter."}}
			}
		}
		if err := replaceChecklist(ctx, tx, id, normalized.Checklist); err != nil {
			return Plan{}, err
		}
	}
	if next.Status == "active" {
		if err := ensurePlanSchedulesTx(ctx, tx, id, r.loc); err != nil {
			return Plan{}, err
		}
		if err := recalcPlanSchedulesTx(ctx, tx, id, r.loc, r.loadThresholdDefaults(ctx)); err != nil {
			return Plan{}, err
		}
	}
	if next.Status != "active" && current.Status == "active" {
		if _, err := tx.Exec(ctx, `
			UPDATE pm_schedules
			SET status = 'cancelled', cancel_reason = 'Plan tidak aktif',
			    cancelled_by = $2, cancelled_at = now(), updated_at = now()
			WHERE pm_plan_id = $1 AND status IN ('scheduled', 'due', 'overdue')
		`, id, actor.ID); err != nil {
			return Plan{}, fmt.Errorf("cancel schedules for inactive plan: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Plan{}, fmt.Errorf("commit PM plan update: %w", err)
	}
	return r.GetPlan(ctx, id)
}

func (r *Repository) SetPlanStatus(ctx context.Context, id int64, actor Actor, status string) (Plan, error) {
	status = strings.TrimSpace(status)
	if status != "active" && status != "inactive" && status != "draft" {
		return Plan{}, &ValidationError{Fields: map[string]string{"status": "Status plan tidak valid."}}
	}
	patch := PlanPatch{Status: &status}
	return r.UpdatePlan(ctx, id, actor, patch)
}

func (r *Repository) getPlanTx(ctx context.Context, tx pgx.Tx, id int64) (Plan, error) {
	item, err := scanPlan(tx.QueryRow(ctx, planSelect+` WHERE p.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, ErrNotFound
	}
	if err != nil {
		return Plan{}, fmt.Errorf("get PM plan: %w", err)
	}
	checklist, err := checklistQuery(ctx, tx, id)
	if err != nil {
		return Plan{}, err
	}
	item.Checklist = checklist
	return item, nil
}

func (r *Repository) checklist(ctx context.Context, planID int64) ([]ChecklistItem, error) {
	return checklistQuery(ctx, r.pool, planID)
}

func (r *Repository) GetPlanChecklist(ctx context.Context, planID int64) ([]ChecklistItem, error) {
	var exists int
	if err := r.pool.QueryRow(ctx, `SELECT 1 FROM pm_plans WHERE id = $1`, planID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("check PM plan: %w", err)
	}
	return r.checklist(ctx, planID)
}

func (r *Repository) AddChecklistItem(ctx context.Context, planID int64, actor Actor, input ChecklistInput) (ChecklistItem, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return ChecklistItem{}, ErrValidation
	}
	task := strings.TrimSpace(input.Task)
	standard := strings.TrimSpace(input.Standard)
	if task == "" || len(task) > 500 {
		return ChecklistItem{}, &ValidationError{Fields: map[string]string{"task": "Task checklist wajib 1-500 karakter."}}
	}
	if len(standard) > 500 {
		return ChecklistItem{}, &ValidationError{Fields: map[string]string{"standard": "Standard checklist maksimal 500 karakter."}}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ChecklistItem{}, fmt.Errorf("begin PM checklist add: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists int
	if err := tx.QueryRow(ctx, `SELECT 1 FROM pm_plans WHERE id = $1 FOR UPDATE`, planID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChecklistItem{}, ErrNotFound
		}
		return ChecklistItem{}, fmt.Errorf("lock PM plan checklist: %w", err)
	}
	sequence := input.Sequence
	if sequence <= 0 {
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(MAX(sequence), 0) + 1 FROM pm_plan_checklist_items WHERE pm_plan_id = $1
		`, planID).Scan(&sequence); err != nil {
			return ChecklistItem{}, fmt.Errorf("next PM checklist sequence: %w", err)
		}
	}
	var item ChecklistItem
	if err := tx.QueryRow(ctx, `
		INSERT INTO pm_plan_checklist_items (pm_plan_id, sequence, component_id, task, standard)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		RETURNING id, pm_plan_id, sequence, component_id, task, COALESCE(standard, ''), created_at, updated_at
	`, planID, sequence, input.ComponentID, task, standard).Scan(
		&item.ID, &item.PlanID, &item.Sequence, &item.ComponentID, &item.Task,
		&item.Standard, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return ChecklistItem{}, fmt.Errorf("add PM checklist item: %w", err)
	}
	if item.ComponentID != nil {
		_ = tx.QueryRow(ctx, `
			SELECT TRIM(COALESCE(code, '') || CASE WHEN name IS NULL THEN '' ELSE ' - ' || name END)
			FROM components WHERE id = $1
		`, *item.ComponentID).Scan(&item.Component)
	}
	if err := tx.Commit(ctx); err != nil {
		return ChecklistItem{}, fmt.Errorf("commit PM checklist add: %w", err)
	}
	return item, nil
}

type rowQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func checklistQuery(ctx context.Context, q rowQuerier, planID int64) ([]ChecklistItem, error) {
	rows, err := q.Query(ctx, `
		SELECT c.id, c.pm_plan_id, c.sequence, c.component_id,
		       TRIM(COALESCE(comp.code, '') || CASE WHEN comp.name IS NULL THEN '' ELSE ' - ' || comp.name END),
		       c.task, COALESCE(c.standard, ''), c.created_at, c.updated_at
		FROM pm_plan_checklist_items c
		LEFT JOIN components comp ON comp.id = c.component_id
		WHERE c.pm_plan_id = $1
		ORDER BY c.sequence ASC, c.id ASC
	`, planID)
	if err != nil {
		return nil, fmt.Errorf("list PM checklist: %w", err)
	}
	defer rows.Close()
	items := make([]ChecklistItem, 0)
	for rows.Next() {
		var item ChecklistItem
		if err := rows.Scan(&item.ID, &item.PlanID, &item.Sequence, &item.ComponentID,
			&item.Component, &item.Task, &item.Standard, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan PM checklist: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanPlan(row interface{ Scan(...any) error }) (Plan, error) {
	var item Plan
	var effectiveFrom, effectiveTo pgtype.Date
	var intervalHM, intervalKM, upcomingHM, upcomingKM, dueSoonHM, dueSoonKM pgtype.Numeric
	var intervalDays, upcomingDays, dueSoonDays pgtype.Int4
	err := row.Scan(
		&item.ID, &item.Name, &item.Description, &item.UnitID, &item.UnitCode,
		&item.UnitModelID, &item.UnitModelName, &item.Status, &effectiveFrom, &effectiveTo,
		&intervalHM, &intervalKM, &intervalDays, &item.IntervalRule,
		&upcomingDays, &dueSoonDays, &upcomingHM, &upcomingKM, &dueSoonHM, &dueSoonKM,
		&item.AutoGenerateWO, &item.DefaultPriority,
		&item.DefaultAssigneeID, &item.DefaultAssigneeName,
		&item.DefaultSupervisorID, &item.DefaultSupervisorName,
		&item.CreatedBy, &item.CreatedByName, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return Plan{}, err
	}
	if effectiveFrom.Valid {
		item.EffectiveFrom = dateOnly(effectiveFrom.Time)
	}
	item.EffectiveTo = datePointer(effectiveTo)
	item.IntervalHM = numericPointer(intervalHM)
	item.IntervalKM = numericPointer(intervalKM)
	item.IntervalDays = intPointer(intervalDays)
	item.UpcomingDays = intPointer(upcomingDays)
	item.DueSoonDays = intPointer(dueSoonDays)
	item.UpcomingHM = numericPointer(upcomingHM)
	item.UpcomingKM = numericPointer(upcomingKM)
	item.DueSoonHM = numericPointer(dueSoonHM)
	item.DueSoonKM = numericPointer(dueSoonKM)
	return item, nil
}

func validatePlanTarget(ctx context.Context, tx pgx.Tx, unitID, modelID *int64) error {
	var exists int
	if unitID != nil {
		if err := tx.QueryRow(ctx, `SELECT 1 FROM units WHERE id = $1 AND is_active = true`, *unitID).Scan(&exists); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ValidationError{Fields: map[string]string{"unit_id": "Unit tidak ditemukan atau tidak aktif."}}
			}
			return fmt.Errorf("validate PM unit: %w", err)
		}
	}
	if modelID != nil {
		if err := tx.QueryRow(ctx, `SELECT 1 FROM unit_models WHERE id = $1 AND is_active = true`, *modelID).Scan(&exists); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ValidationError{Fields: map[string]string{"unit_model_id": "Model unit tidak ditemukan atau tidak aktif."}}
			}
			return fmt.Errorf("validate PM model: %w", err)
		}
	}
	return nil
}

func validatePlanUsers(ctx context.Context, tx pgx.Tx, assigneeID, supervisorID *int64) error {
	if assigneeID != nil {
		var exists int
		if err := tx.QueryRow(ctx, `
			SELECT 1 FROM users WHERE id = $1 AND is_active = true
			AND role IN ('admin', 'planner', 'mechanic')
		`, *assigneeID).Scan(&exists); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ValidationError{Fields: map[string]string{"default_assignee_id": "User assignee tidak ditemukan atau tidak aktif."}}
			}
			return fmt.Errorf("validate PM assignee: %w", err)
		}
	}
	if supervisorID != nil {
		var exists int
		if err := tx.QueryRow(ctx, `
			SELECT 1 FROM users WHERE id = $1 AND is_active = true
			AND role IN ('admin', 'planner')
		`, *supervisorID).Scan(&exists); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ValidationError{Fields: map[string]string{"default_supervisor_id": "User supervisor tidak ditemukan atau tidak aktif."}}
			}
			return fmt.Errorf("validate PM supervisor: %w", err)
		}
	}
	return nil
}

func validatePlanReadiness(plan Plan) error {
	fields := map[string]string{}
	if plan.IntervalHM == nil && plan.IntervalKM == nil && plan.IntervalDays == nil {
		fields["interval"] = "Minimal satu interval wajib diisi sebelum plan diaktifkan."
	}
	if plan.EffectiveFrom.IsZero() {
		fields["effective_from"] = "Tanggal efektif wajib diisi."
	}
	if plan.UnitID == nil && plan.UnitModelID == nil {
		fields["target"] = "Plan belum memiliki target."
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func validatePlanValues(plan Plan) error {
	fields := map[string]string{}
	if len(plan.Name) < 3 || len(plan.Name) > 200 {
		fields["name"] = "Nama plan wajib 3-200 karakter."
	}
	if len(plan.Description) > 2000 {
		fields["description"] = "Deskripsi maksimal 2000 karakter."
	}
	if plan.EffectiveTo != nil && plan.EffectiveTo.Before(plan.EffectiveFrom) {
		fields["effective_to"] = "Tanggal berakhir tidak boleh sebelum tanggal efektif."
	}
	if plan.IntervalHM != nil && *plan.IntervalHM <= 0 {
		fields["interval_hm"] = "Interval HM harus lebih dari 0."
	}
	if plan.IntervalKM != nil && *plan.IntervalKM <= 0 {
		fields["interval_km"] = "Interval KM harus lebih dari 0."
	}
	if plan.IntervalDays != nil && *plan.IntervalDays <= 0 {
		fields["interval_days"] = "Interval hari harus lebih dari 0."
	}
	if plan.IntervalHM == nil && plan.IntervalKM == nil && plan.IntervalDays == nil {
		fields["interval"] = "Minimal satu interval wajib diisi."
	}
	if plan.IntervalRule != "first_reached" && plan.IntervalRule != "all_required" {
		fields["interval_rule"] = "Aturan interval tidak valid."
	}
	if !validPriority(plan.DefaultPriority) {
		fields["default_priority"] = "Prioritas default tidak valid."
	}
	if plan.UpcomingDays != nil && (*plan.UpcomingDays < 0 || *plan.UpcomingDays > 365) {
		fields["upcoming_days"] = "Upcoming days harus antara 0 dan 365."
	}
	if plan.DueSoonDays != nil && (*plan.DueSoonDays < 0 || *plan.DueSoonDays > 365) {
		fields["due_soon_days"] = "Due soon days harus antara 0 dan 365."
	}
	if plan.UpcomingDays != nil && plan.DueSoonDays != nil && *plan.DueSoonDays > *plan.UpcomingDays {
		fields["due_soon_days"] = "Due soon tidak boleh lebih besar dari upcoming."
	}
	if plan.UpcomingHM != nil && *plan.UpcomingHM < 0 {
		fields["upcoming_hm"] = "Upcoming HM tidak valid."
	}
	if plan.UpcomingKM != nil && *plan.UpcomingKM < 0 {
		fields["upcoming_km"] = "Upcoming KM tidak valid."
	}
	if plan.DueSoonHM != nil && *plan.DueSoonHM < 0 {
		fields["due_soon_hm"] = "Due soon HM tidak valid."
	}
	if plan.DueSoonKM != nil && *plan.DueSoonKM < 0 {
		fields["due_soon_km"] = "Due soon KM tidak valid."
	}
	if plan.UpcomingHM != nil && plan.DueSoonHM != nil && *plan.DueSoonHM > *plan.UpcomingHM {
		fields["due_soon_hm"] = "Due soon HM tidak boleh lebih besar dari upcoming HM."
	}
	if plan.UpcomingKM != nil && plan.DueSoonKM != nil && *plan.DueSoonKM > *plan.UpcomingKM {
		fields["due_soon_km"] = "Due soon KM tidak boleh lebih besar dari upcoming KM."
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func applyNullableFloat(target **float64, value *float64, clear bool) {
	if clear {
		*target = nil
		return
	}
	if value != nil {
		copy := *value
		*target = &copy
	}
}

func applyNullableInt(target **int, value *int, clear bool) {
	if clear {
		*target = nil
		return
	}
	if value != nil {
		copy := *value
		*target = &copy
	}
}

func numericPointer(value pgtype.Numeric) *float64 {
	if !value.Valid {
		return nil
	}
	converted, err := value.Float64Value()
	if err != nil || !converted.Valid {
		return nil
	}
	result := converted.Float64
	return &result
}

func intPointer(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int32)
	return &result
}

func datePointer(value pgtype.Date) *time.Time {
	if !value.Valid {
		return nil
	}
	result := dateOnly(value.Time)
	return &result
}

func timestampPointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func dateValue(value time.Time) pgtype.Date {
	if value.IsZero() {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: dateOnly(value), Valid: true}
}

func nullDate(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return pgtype.Date{Time: dateOnly(*value), Valid: true}
}

func planSortColumn(raw string) string {
	switch raw {
	case "name":
		return "p.name"
	case "status":
		return "p.status"
	case "effective_from":
		return "p.effective_from"
	case "updated_at":
		return "p.updated_at"
	default:
		return "p.created_at"
	}
}

func replaceChecklist(ctx context.Context, tx pgx.Tx, planID int64, items []ChecklistInput) error {
	if _, err := tx.Exec(ctx, `DELETE FROM pm_plan_checklist_items WHERE pm_plan_id = $1`, planID); err != nil {
		return fmt.Errorf("clear PM checklist: %w", err)
	}
	for index, item := range items {
		sequence := item.Sequence
		if sequence <= 0 {
			sequence = index + 1
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO pm_plan_checklist_items (pm_plan_id, sequence, component_id, task, standard)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		`, planID, sequence, item.ComponentID, item.Task, item.Standard); err != nil {
			return fmt.Errorf("insert PM checklist: %w", err)
		}
	}
	return nil
}

type planCycle struct {
	UnitID        *int64
	UnitModelID   *int64
	Status        string
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	IntervalHM    *float64
	IntervalKM    *float64
	IntervalDays  *int
}

func loadPlanCycle(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, planID int64) (planCycle, error) {
	var item planCycle
	var from, to pgtype.Date
	var hm, km pgtype.Numeric
	var days pgtype.Int4
	err := q.QueryRow(ctx, `
		SELECT unit_id, unit_model_id, status, effective_from, effective_to,
		       interval_hm, interval_km, interval_days
		FROM pm_plans
		WHERE id = $1
	`, planID).Scan(&item.UnitID, &item.UnitModelID, &item.Status, &from, &to, &hm, &km, &days)
	if errors.Is(err, pgx.ErrNoRows) {
		return planCycle{}, ErrNotFound
	}
	if err != nil {
		return planCycle{}, fmt.Errorf("load PM plan cycle: %w", err)
	}
	if from.Valid {
		item.EffectiveFrom = dateOnly(from.Time)
	}
	item.EffectiveTo = datePointer(to)
	item.IntervalHM = numericPointer(hm)
	item.IntervalKM = numericPointer(km)
	item.IntervalDays = intPointer(days)
	return item, nil
}

func ensurePlanSchedulesTx(ctx context.Context, tx pgx.Tx, planID int64, loc *time.Location) error {
	cycle, err := loadPlanCycle(ctx, tx, planID)
	if err != nil {
		return err
	}
	if cycle.Status != "active" {
		return nil
	}

	unitIDs := make([]int64, 0)
	if cycle.UnitID != nil {
		var active bool
		if err := tx.QueryRow(ctx, `SELECT is_active FROM units WHERE id = $1`, *cycle.UnitID).Scan(&active); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("load PM unit: %w", err)
		}
		if active {
			unitIDs = append(unitIDs, *cycle.UnitID)
		}
	} else if cycle.UnitModelID != nil {
		rows, err := tx.Query(ctx, `
			SELECT id FROM units WHERE unit_model_id = $1 AND is_active = true ORDER BY id
		`, *cycle.UnitModelID)
		if err != nil {
			return fmt.Errorf("list units for PM model: %w", err)
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return fmt.Errorf("scan PM unit: %w", err)
			}
			unitIDs = append(unitIDs, id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate PM units: %w", err)
		}
		rows.Close()
	}

	for _, unitID := range unitIDs {
		var exists int
		if err := tx.QueryRow(ctx, `
			SELECT 1 FROM pm_schedules
			WHERE pm_plan_id = $1 AND unit_id = $2
			  AND status IN ('scheduled', 'due', 'overdue', 'wo_created')
		`, planID, unitID).Scan(&exists); err == nil {
			continue
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("check PM schedule: %w", err)
		}

		var currentHM, currentKM float64
		if err := tx.QueryRow(ctx, `SELECT current_hm, current_km FROM units WHERE id = $1`, unitID).Scan(&currentHM, &currentKM); err != nil {
			return fmt.Errorf("load PM unit readings: %w", err)
		}
		dueHM, dueKM, dueDate := computeDue(cycle.IntervalHM, cycle.IntervalKM, cycle.IntervalDays, currentHM, currentKM, cycle.EffectiveFrom)
		if _, err := tx.Exec(ctx, `
			INSERT INTO pm_schedules (
				pm_plan_id, unit_id, cycle_no, status, due_hm, due_km, due_date
			)
			VALUES ($1, $2, 1, 'scheduled', $3, $4, $5)
			ON CONFLICT DO NOTHING
		`, planID, unitID, dueHM, dueKM, nullDate(dueDate)); err != nil {
			return fmt.Errorf("create PM schedule: %w", err)
		}
	}
	return nil
}

func recalcPlanSchedulesTx(ctx context.Context, tx pgx.Tx, planID int64, loc *time.Location, defaults thresholdDefaults) error {
	cycle, err := loadPlanCycle(ctx, tx, planID)
	if err != nil || cycle.Status != "active" {
		return err
	}
	rows, err := tx.Query(ctx, `
		SELECT id, unit_id, due_hm, due_km, due_date,
		       last_completed_hm, last_completed_km, last_completed_at,
		       override_reason IS NOT NULL OR overridden_at IS NOT NULL
		FROM pm_schedules
		WHERE pm_plan_id = $1 AND status IN ('scheduled', 'due', 'overdue')
		FOR UPDATE
	`, planID)
	if err != nil {
		return fmt.Errorf("list schedules for recalculation: %w", err)
	}
	type row struct {
		id             int64
		unitID         int64
		dueHM, dueKM   *float64
		dueDate        pgtype.Date
		lastHM, lastKM *float64
		lastAt         pgtype.Timestamptz
		hasOverride    bool
	}
	items := make([]row, 0)
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.id, &item.unitID, &item.dueHM, &item.dueKM, &item.dueDate,
			&item.lastHM, &item.lastKM, &item.lastAt, &item.hasOverride); err != nil {
			rows.Close()
			return fmt.Errorf("scan PM schedule recalc: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate PM schedule recalc: %w", err)
	}
	rows.Close()

	for _, item := range items {
		if item.hasOverride {
			continue
		}
		var currentHM, currentKM float64
		if err := tx.QueryRow(ctx, `SELECT current_hm, current_km FROM units WHERE id = $1`, item.unitID).Scan(&currentHM, &currentKM); err != nil {
			return fmt.Errorf("load recalc unit: %w", err)
		}
		baseHM := currentHM
		baseKM := currentKM
		baseDate := cycle.EffectiveFrom
		if item.lastHM != nil {
			baseHM = *item.lastHM
		}
		if item.lastKM != nil {
			baseKM = *item.lastKM
		}
		if item.lastAt.Valid {
			baseDate = dateOnlyIn(item.lastAt.Time, loc)
		}
		dueHM, dueKM, dueDate := computeDue(cycle.IntervalHM, cycle.IntervalKM, cycle.IntervalDays, baseHM, baseKM, baseDate)
		if _, err := tx.Exec(ctx, `
			UPDATE pm_schedules SET due_hm = $2, due_km = $3, due_date = $4, updated_at = now()
			WHERE id = $1
		`, item.id, dueHM, dueKM, nullDate(dueDate)); err != nil {
			return fmt.Errorf("recalculate PM schedule: %w", err)
		}
		if err := recalculateOneStatus(ctx, tx, item.id, loc, defaults); err != nil {
			return err
		}
	}
	return nil
}

func recalculateOneStatus(ctx context.Context, tx pgx.Tx, id int64, loc *time.Location, defaults thresholdDefaults) error {
	record, err := loadScheduleRecord(ctx, tx, id)
	if err != nil {
		return err
	}
	if record.Status == "completed" || record.Status == "skipped" || record.Status == "cancelled" || record.Status == "wo_created" {
		return nil
	}
	result := pmcalc.Evaluate(pmcalc.Plan{
		IntervalRule:        record.rule,
		UpcomingDays:        record.upcomingDays,
		DueSoonDays:         record.dueSoonDays,
		DefaultUpcomingDays: &defaults.UpcomingDays,
		DefaultDueSoonDays:  &defaults.DueSoonDays,
		UpcomingHM:          record.upcomingHM,
		UpcomingKM:          record.upcomingKM,
		DueSoonHM:           record.dueSoonHM,
		DueSoonKM:           record.dueSoonKM,
	}, pmcalc.Schedule{
		DueHM:   record.DueHM,
		DueKM:   record.DueKM,
		DueDate: record.DueDate,
	}, record.currentHM, record.currentKM, dateOnlyIn(time.Now(), loc))
	status := "scheduled"
	if result.Overdue {
		status = "overdue"
	} else if result.Due {
		status = "due"
	}
	if _, err := tx.Exec(ctx, `UPDATE pm_schedules SET status = $2, updated_at = now() WHERE id = $1`, id, status); err != nil {
		return fmt.Errorf("recalculate PM schedule status: %w", err)
	}
	return nil
}

func computeDue(intervalHM, intervalKM *float64, intervalDays *int, currentHM, currentKM float64, baseDate time.Time) (*float64, *float64, *time.Time) {
	var dueHM, dueKM *float64
	var dueDate *time.Time
	if intervalHM != nil {
		value := currentHM + *intervalHM
		dueHM = &value
	}
	if intervalKM != nil {
		value := currentKM + *intervalKM
		dueKM = &value
	}
	if intervalDays != nil {
		value := baseDate.AddDate(0, 0, *intervalDays)
		dueDate = &value
	}
	return dueHM, dueKM, dueDate
}

const scheduleSelect = `
	SELECT s.id, s.pm_plan_id, p.name, s.unit_id, u.code,
	       TRIM(COALESCE(um.manufacturer, '') || CASE WHEN um.model IS NULL THEN '' ELSE ' ' || um.model END),
	       s.cycle_no, s.status, s.due_hm, s.due_km, s.due_date,
	       s.last_completed_hm, s.last_completed_km, s.last_completed_at,
	       COALESCE(s.override_reason, ''), s.overridden_at,
	       COALESCE(s.skip_reason, ''), s.skipped_at,
	       COALESCE(s.cancel_reason, ''), s.cancelled_at,
	       wo.id, COALESCE(wo.code, ''), s.created_at, s.updated_at,
	       p.interval_rule, p.upcoming_days, p.due_soon_days,
	       p.upcoming_hm, p.upcoming_km, p.due_soon_hm, p.due_soon_km,
	       p.auto_generate_wo, u.current_hm, u.current_km
	FROM pm_schedules s
	JOIN pm_plans p ON p.id = s.pm_plan_id
	JOIN units u ON u.id = s.unit_id
	LEFT JOIN unit_models um ON um.id = u.unit_model_id
	LEFT JOIN work_orders wo ON wo.pm_schedule_id = s.id AND wo.status <> 'cancelled'
`

type scheduleRecord struct {
	Schedule
	rule         string
	upcomingDays *int
	dueSoonDays  *int
	upcomingHM   *float64
	upcomingKM   *float64
	dueSoonHM    *float64
	dueSoonKM    *float64
	autoGenerate bool
	currentHM    float64
	currentKM    float64
}

func scanSchedule(row interface{ Scan(...any) error }) (scheduleRecord, error) {
	var record scheduleRecord
	var dueHM, dueKM, lastHM, lastKM pgtype.Numeric
	var dueDate pgtype.Date
	var lastAt, overriddenAt, skippedAt, cancelledAt pgtype.Timestamptz
	var overrideReason, skipReason, cancelReason string
	var woID *int64
	var woCode string
	var rule string
	var upcomingDays, dueSoonDays pgtype.Int4
	var upcomingHM, upcomingKM, dueSoonHM, dueSoonKM pgtype.Numeric
	err := row.Scan(
		&record.ID, &record.PlanID, &record.PlanName, &record.UnitID, &record.UnitCode,
		&record.UnitModelName, &record.CycleNo, &record.Status, &dueHM, &dueKM, &dueDate,
		&lastHM, &lastKM, &lastAt, &overrideReason, &overriddenAt,
		&skipReason, &skippedAt, &cancelReason, &cancelledAt,
		&woID, &woCode, &record.CreatedAt, &record.UpdatedAt,
		&rule, &upcomingDays, &dueSoonDays, &upcomingHM, &upcomingKM,
		&dueSoonHM, &dueSoonKM, &record.autoGenerate, &record.currentHM, &record.currentKM,
	)
	if err != nil {
		return scheduleRecord{}, err
	}
	record.DueHM = numericPointer(dueHM)
	record.DueKM = numericPointer(dueKM)
	record.DueDate = datePointer(dueDate)
	record.LastCompletedHM = numericPointer(lastHM)
	record.LastCompletedKM = numericPointer(lastKM)
	record.LastCompletedAt = timestampPointer(lastAt)
	record.OverrideReason = overrideReason
	record.OverriddenAt = timestampPointer(overriddenAt)
	record.SkipReason = skipReason
	record.SkippedAt = timestampPointer(skippedAt)
	record.CancelReason = cancelReason
	record.CancelledAt = timestampPointer(cancelledAt)
	record.WorkOrderID = woID
	record.WorkOrderCode = woCode
	record.rule = rule
	record.upcomingDays = intPointer(upcomingDays)
	record.dueSoonDays = intPointer(dueSoonDays)
	record.upcomingHM = numericPointer(upcomingHM)
	record.upcomingKM = numericPointer(upcomingKM)
	record.dueSoonHM = numericPointer(dueSoonHM)
	record.dueSoonKM = numericPointer(dueSoonKM)
	return record, nil
}

type thresholdDefaults struct {
	UpcomingDays int
	DueSoonDays  int
}

func (r *Repository) loadThresholdDefaults(ctx context.Context) thresholdDefaults {
	defaults := thresholdDefaults{UpcomingDays: 30, DueSoonDays: 7}
	rows, err := r.pool.Query(ctx, `
		SELECT key, COALESCE(value, '') FROM app_settings
		WHERE key IN ($1, $2)
	`, settings.KeyPMUpcomingDays, settings.KeyPMDueSoonDays)
	if err != nil {
		return defaults
	}
	defer rows.Close()
	for rows.Next() {
		var key, raw string
		if err := rows.Scan(&key, &raw); err != nil {
			continue
		}
		value, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || value <= 0 {
			continue
		}
		switch key {
		case settings.KeyPMUpcomingDays:
			defaults.UpcomingDays = value
		case settings.KeyPMDueSoonDays:
			defaults.DueSoonDays = value
		}
	}
	return defaults
}

func (r *Repository) evaluateRecord(record *scheduleRecord, today time.Time, defaults thresholdDefaults) {
	if record.Status == "completed" || record.Status == "skipped" || record.Status == "cancelled" || record.Status == "wo_created" {
		record.Bucket = record.Status
		return
	}
	result := pmcalc.Evaluate(pmcalc.Plan{
		IntervalRule:        record.rule,
		UpcomingDays:        record.upcomingDays,
		DueSoonDays:         record.dueSoonDays,
		DefaultUpcomingDays: &defaults.UpcomingDays,
		DefaultDueSoonDays:  &defaults.DueSoonDays,
		UpcomingHM:          record.upcomingHM,
		UpcomingKM:          record.upcomingKM,
		DueSoonHM:           record.dueSoonHM,
		DueSoonKM:           record.dueSoonKM,
	}, pmcalc.Schedule{
		DueHM:   record.DueHM,
		DueKM:   record.DueKM,
		DueDate: record.DueDate,
	}, record.currentHM, record.currentKM, today)
	record.Bucket = result.Bucket
	record.DaysToDue = result.DaysToDue
}

func (r *Repository) ListSchedules(ctx context.Context, filter ScheduleFilter) ([]Schedule, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 25
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	args := make([]any, 0, 5)
	where := []string{"1 = 1"}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("s.status = $%d", len(args)))
	}
	if filter.UnitID != nil {
		args = append(args, *filter.UnitID)
		where = append(where, fmt.Sprintf("s.unit_id = $%d", len(args)))
	}
	if filter.PlanID != nil {
		args = append(args, *filter.PlanID)
		where = append(where, fmt.Sprintf("s.pm_plan_id = $%d", len(args)))
	}
	if filter.ActiveOnly {
		where = append(where, "s.status IN ('scheduled', 'due', 'overdue', 'wo_created')")
	}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		where = append(where, fmt.Sprintf("(p.name ILIKE $%d OR u.code ILIKE $%d)", len(args), len(args)))
	}

	rows, err := r.pool.Query(ctx, scheduleSelect+`
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY s.due_date ASC NULLS LAST, s.id ASC
		LIMIT 10000
	`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list PM schedules: %w", err)
	}
	defer rows.Close()

	today := dateOnlyIn(time.Now(), r.loc)
	defaults := r.loadThresholdDefaults(ctx)
	items := make([]Schedule, 0)
	for rows.Next() {
		record, err := scanSchedule(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan PM schedule: %w", err)
		}
		r.evaluateRecord(&record, today, defaults)
		if filter.Bucket != "" && record.Bucket != filter.Bucket {
			continue
		}
		items = append(items, record.Schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate PM schedules: %w", err)
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if left.DueDate == nil && right.DueDate != nil {
			return false
		}
		if left.DueDate != nil && right.DueDate == nil {
			return true
		}
		if left.DueDate != nil && right.DueDate != nil && !left.DueDate.Equal(*right.DueDate) {
			return left.DueDate.Before(*right.DueDate)
		}
		return left.ID < right.ID
	})
	total := len(items)
	start := (filter.Page - 1) * filter.Limit
	if start > total {
		start = total
	}
	end := start + filter.Limit
	if end > total {
		end = total
	}
	return items[start:end], total, nil
}

func (r *Repository) GetSchedule(ctx context.Context, id int64) (Schedule, error) {
	record, err := loadScheduleRecord(ctx, r.pool, id)
	if err != nil {
		return Schedule{}, err
	}
	r.evaluateRecord(&record, dateOnlyIn(time.Now(), r.loc), r.loadThresholdDefaults(ctx))
	plan, err := r.GetPlan(ctx, record.PlanID)
	if err != nil {
		return Schedule{}, err
	}
	record.Plan = &plan
	return record.Schedule, nil
}

type singleRowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func loadScheduleRecord(ctx context.Context, q singleRowQuerier, id int64) (scheduleRecord, error) {
	record, err := scanSchedule(q.QueryRow(ctx, scheduleSelect+` WHERE s.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return scheduleRecord{}, ErrNotFound
	}
	if err != nil {
		return scheduleRecord{}, fmt.Errorf("get PM schedule: %w", err)
	}
	return record, nil
}

func (r *Repository) GenerateWO(ctx context.Context, id int64, actor Actor) (GenerateResult, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return GenerateResult{}, ErrValidation
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("begin PM generate WO: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		unitID                       int64
		cycleNo                      int
		status                       string
		planName, priority, unitCode string
		assigneeID, supervisorID     *int64
		dueDate                      pgtype.Date
	)
	err = tx.QueryRow(ctx, `
		SELECT s.unit_id, s.cycle_no, s.status, p.name, p.default_priority,
		       p.default_assignee_id, p.default_supervisor_id, u.code, s.due_date
		FROM pm_schedules s
		JOIN pm_plans p ON p.id = s.pm_plan_id
		JOIN units u ON u.id = s.unit_id
		WHERE s.id = $1
		FOR UPDATE OF s
	`, id).Scan(&unitID, &cycleNo, &status, &planName, &priority,
		&assigneeID, &supervisorID, &unitCode, &dueDate)
	if errors.Is(err, pgx.ErrNoRows) {
		return GenerateResult{}, ErrNotFound
	}
	if err != nil {
		return GenerateResult{}, fmt.Errorf("load PM schedule for generation: %w", err)
	}
	if status != "scheduled" && status != "due" && status != "overdue" {
		return GenerateResult{}, ErrConflict
	}
	var existing int64
	err = tx.QueryRow(ctx, `
		SELECT id FROM work_orders
		WHERE pm_schedule_id = $1 AND status <> 'cancelled'
		LIMIT 1
	`, id).Scan(&existing)
	if err == nil {
		return GenerateResult{}, ErrConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return GenerateResult{}, fmt.Errorf("check PM work order: %w", err)
	}

	now := time.Now()
	createdBy := &actor.ID
	if actor.ID <= 0 {
		createdBy = nil
	}
	order, err := workorders.Create(ctx, tx, workorders.CreateInput{
		Source:          "pm",
		UnitID:          &unitID,
		PMScheduleID:    &id,
		Title:           fmt.Sprintf("PM: %s - %s", planName, unitCode),
		Description:     fmt.Sprintf("Preventive maintenance cycle #%d", cycleNo),
		Priority:        priority,
		Status:          "open",
		MaintenanceType: "preventive",
		AssigneeID:      assigneeID,
		SupervisorID:    supervisorID,
		DueDate:         datePointer(dueDate),
		CreatedBy:       createdBy,
		Now:             now,
		Location:        r.loc,
	})
	if err != nil {
		return GenerateResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO work_order_checklist_items (
			wo_id, pm_plan_checklist_item_id, sequence, component_id, task, standard
		)
		SELECT $1, c.id, c.sequence, c.component_id, c.task, c.standard
		FROM pm_plan_checklist_items c
		JOIN pm_schedules s ON s.pm_plan_id = c.pm_plan_id
		WHERE s.id = $2
		ORDER BY c.sequence
	`, order.ID, id); err != nil {
		return GenerateResult{}, fmt.Errorf("snapshot PM checklist: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pm_schedules SET status = 'wo_created', updated_at = now() WHERE id = $1
	`, id); err != nil {
		return GenerateResult{}, fmt.Errorf("mark PM schedule WO created: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return GenerateResult{}, fmt.Errorf("commit PM generate WO: %w", err)
	}
	schedule, err := r.GetSchedule(ctx, id)
	if err != nil {
		return GenerateResult{}, err
	}
	return GenerateResult{Schedule: schedule, WorkOrder: order}, nil
}

func (r *Repository) OverrideDue(ctx context.Context, id int64, actor Actor, input OverrideInput) (Schedule, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return Schedule{}, ErrValidation
	}
	reason := strings.TrimSpace(input.Reason)
	if len(reason) < 5 {
		return Schedule{}, &ValidationError{Fields: map[string]string{"reason": "Alasan override wajib minimal 5 karakter."}}
	}
	if input.DueHM == nil && input.DueKM == nil && input.DueDate == nil {
		return Schedule{}, &ValidationError{Fields: map[string]string{"due": "Minimal satu due HM, KM, atau tanggal wajib diisi."}}
	}
	if input.DueHM != nil && *input.DueHM < 0 {
		return Schedule{}, &ValidationError{Fields: map[string]string{"due_hm": "Due HM tidak valid."}}
	}
	if input.DueKM != nil && *input.DueKM < 0 {
		return Schedule{}, &ValidationError{Fields: map[string]string{"due_km": "Due KM tidak valid."}}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Schedule{}, fmt.Errorf("begin PM override: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM pm_schedules WHERE id = $1 FOR UPDATE`, id).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Schedule{}, ErrNotFound
		}
		return Schedule{}, fmt.Errorf("lock PM schedule: %w", err)
	}
	if status == "completed" || status == "skipped" || status == "cancelled" || status == "wo_created" {
		return Schedule{}, ErrConflict
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pm_schedules
		SET due_hm = COALESCE($2, due_hm),
		    due_km = COALESCE($3, due_km),
		    due_date = COALESCE($4, due_date),
		    override_reason = $5,
		    overridden_by = $6,
		    overridden_at = now(),
		    status = 'scheduled',
		    updated_at = now()
		WHERE id = $1
	`, id, input.DueHM, input.DueKM, nullDate(input.DueDate), reason, actor.ID); err != nil {
		return Schedule{}, fmt.Errorf("override PM due: %w", err)
	}
	if err := r.refreshScheduleStatusTx(ctx, tx, id); err != nil {
		return Schedule{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Schedule{}, fmt.Errorf("commit PM override: %w", err)
	}
	return r.GetSchedule(ctx, id)
}

func (r *Repository) Skip(ctx context.Context, id int64, actor Actor, reason string) (Schedule, error) {
	return r.finishSchedule(ctx, id, actor, "skipped", strings.TrimSpace(reason))
}

func (r *Repository) Cancel(ctx context.Context, id int64, actor Actor, reason string) (Schedule, error) {
	return r.finishSchedule(ctx, id, actor, "cancelled", strings.TrimSpace(reason))
}

func (r *Repository) finishSchedule(ctx context.Context, id int64, actor Actor, status, reason string) (Schedule, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return Schedule{}, ErrValidation
	}
	if len(reason) < 5 {
		return Schedule{}, &ValidationError{Fields: map[string]string{"reason": "Alasan wajib minimal 5 karakter."}}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Schedule{}, fmt.Errorf("begin PM schedule finish: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current string
	if err := tx.QueryRow(ctx, `SELECT status FROM pm_schedules WHERE id = $1 FOR UPDATE`, id).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Schedule{}, ErrNotFound
		}
		return Schedule{}, fmt.Errorf("lock PM schedule: %w", err)
	}
	if current == "completed" || current == "skipped" || current == "cancelled" {
		return Schedule{}, ErrConflict
	}
	var woID int64
	err = tx.QueryRow(ctx, `
		SELECT id FROM work_orders WHERE pm_schedule_id = $1 AND status <> 'cancelled' LIMIT 1
	`, id).Scan(&woID)
	if err == nil {
		return Schedule{}, ErrConflict
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Schedule{}, fmt.Errorf("check PM schedule work order: %w", err)
	}
	if status == "skipped" {
		_, err = tx.Exec(ctx, `
			UPDATE pm_schedules
			SET status = 'skipped', skip_reason = $2, skipped_by = $3, skipped_at = now(), updated_at = now()
			WHERE id = $1
		`, id, reason, actor.ID)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE pm_schedules
			SET status = 'cancelled', cancel_reason = $2, cancelled_by = $3,
			    cancelled_at = now(), updated_at = now()
			WHERE id = $1
		`, id, reason, actor.ID)
	}
	if err != nil {
		return Schedule{}, fmt.Errorf("finish PM schedule: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Schedule{}, fmt.Errorf("commit PM schedule finish: %w", err)
	}
	return r.GetSchedule(ctx, id)
}

func (r *Repository) ensureAllSchedules(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT id FROM pm_plans WHERE status = 'active' ORDER BY id`)
	if err != nil {
		return fmt.Errorf("list active PM plans: %w", err)
	}
	planIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan active PM plan: %w", err)
		}
		planIDs = append(planIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate active PM plans: %w", err)
	}
	rows.Close()

	for _, planID := range planIDs {
		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin PM schedule ensure: %w", err)
		}
		if err := ensurePlanSchedulesTx(ctx, tx, planID, r.loc); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit PM schedule ensure: %w", err)
		}
	}
	return nil
}

func (r *Repository) EvaluateDue(ctx context.Context, now time.Time) (EvaluateResult, error) {
	if err := r.ensureAllSchedules(ctx); err != nil {
		return EvaluateResult{}, err
	}
	var pmAutoRaw string
	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(value, 'false') FROM app_settings WHERE key = $1
	`, settings.KeyPMAutoWO).Scan(&pmAutoRaw); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return EvaluateResult{}, fmt.Errorf("load PM auto setting: %w", err)
	}
	pmAuto := strings.EqualFold(pmAutoRaw, "true")

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return EvaluateResult{}, fmt.Errorf("begin PM evaluation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, scheduleSelect+`
		WHERE p.status = 'active'
		  AND s.status IN ('scheduled', 'due', 'overdue')
		  AND (p.effective_to IS NULL OR p.effective_to >= $1)
		ORDER BY s.id
	`, dateOnlyIn(now, r.loc))
	if err != nil {
		return EvaluateResult{}, fmt.Errorf("list PM schedules for evaluation: %w", err)
	}
	records := make([]scheduleRecord, 0)
	for rows.Next() {
		record, err := scanSchedule(rows)
		if err != nil {
			rows.Close()
			return EvaluateResult{}, fmt.Errorf("scan PM evaluation schedule: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return EvaluateResult{}, fmt.Errorf("iterate PM evaluation schedules: %w", err)
	}
	rows.Close()

	result := EvaluateResult{}
	toGenerate := make([]int64, 0)
	today := dateOnlyIn(now, r.loc)
	defaults := r.loadThresholdDefaults(ctx)
	for _, record := range records {
		r.evaluateRecord(&record, today, defaults)
		nextStatus := "scheduled"
		if record.Bucket == pmcalc.BucketOverdue {
			nextStatus = "overdue"
		} else if record.Bucket == pmcalc.BucketDue {
			nextStatus = "due"
		}
		if nextStatus != record.Status {
			if _, err := tx.Exec(ctx, `
				UPDATE pm_schedules SET status = $2, updated_at = now() WHERE id = $1
			`, record.ID, nextStatus); err != nil {
				return EvaluateResult{}, fmt.Errorf("update PM schedule status: %w", err)
			}
			result.Updated++
		}

		switch record.Bucket {
		case pmcalc.BucketUpcoming, pmcalc.BucketDueSoon, pmcalc.BucketDue, pmcalc.BucketOverdue:
			created, err := notifications.CreateForRoles(ctx, tx, []string{"admin", "planner"}, notifications.CreateInput{
				Type:      "pm_due",
				Title:     fmt.Sprintf("PM %s: %s", record.UnitCode, record.PlanName),
				Body:      fmt.Sprintf("Siklus #%d untuk %s berstatus %s.", record.CycleNo, record.UnitCode, bucketLabel(record.Bucket)),
				RefType:   "pm_schedule",
				RefID:     &record.ID,
				DedupeKey: fmt.Sprintf("pm:%d:%d:%s", record.ID, record.CycleNo, record.Bucket),
			}, nil)
			if err != nil {
				return EvaluateResult{}, err
			}
			result.Notified += created
		}
		if record.autoGenerate && pmAuto && (record.Bucket == pmcalc.BucketDue || record.Bucket == pmcalc.BucketOverdue) {
			toGenerate = append(toGenerate, record.ID)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return EvaluateResult{}, fmt.Errorf("commit PM evaluation: %w", err)
	}

	for _, scheduleID := range toGenerate {
		generated, err := r.GenerateWO(ctx, scheduleID, Actor{Role: "admin"})
		if err != nil {
			if errors.Is(err, ErrConflict) {
				continue
			}
			return result, err
		}
		if generated.WorkOrder.ID > 0 {
			result.AutoGenerated++
		}
	}
	return result, nil
}

func bucketLabel(bucket string) string {
	switch bucket {
	case pmcalc.BucketUpcoming:
		return "upcoming"
	case pmcalc.BucketDueSoon:
		return "due soon"
	case pmcalc.BucketDue:
		return "due"
	case pmcalc.BucketOverdue:
		return "overdue"
	default:
		return bucket
	}
}

func (r *Repository) Compliance(ctx context.Context, filter ComplianceFilter) (Compliance, error) {
	args := make([]any, 0, 4)
	where := []string{"1 = 1"}
	if filter.PlanID != nil {
		args = append(args, *filter.PlanID)
		where = append(where, fmt.Sprintf("s.pm_plan_id = $%d", len(args)))
	}
	if filter.UnitID != nil {
		args = append(args, *filter.UnitID)
		where = append(where, fmt.Sprintf("s.unit_id = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		where = append(where, fmt.Sprintf("s.created_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		where = append(where, fmt.Sprintf("s.created_at <= $%d", len(args)))
	}

	rows, err := r.pool.Query(ctx, scheduleSelect+`
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY p.name, s.unit_id, s.cycle_no
		LIMIT 10000
	`, args...)
	if err != nil {
		return Compliance{}, fmt.Errorf("list PM compliance: %w", err)
	}
	defer rows.Close()

	grouped := map[int64]*ComplianceRow{}
	total := Compliance{}
	for rows.Next() {
		record, err := scanSchedule(rows)
		if err != nil {
			return Compliance{}, fmt.Errorf("scan PM compliance: %w", err)
		}
		row := grouped[record.PlanID]
		if row == nil {
			row = &ComplianceRow{PlanID: record.PlanID, PlanName: record.PlanName, Target: record.UnitCode}
			grouped[record.PlanID] = row
		}
		row.Total++
		total.Total++
		late := false
		if record.Status == "completed" {
			row.Completed++
			total.Completed++
			if record.DueDate != nil && record.LastCompletedAt != nil {
				if dateOnlyIn(*record.LastCompletedAt, r.loc).After(dateOnly(*record.DueDate)) {
					late = true
				}
			}
			if record.DueHM != nil && record.LastCompletedHM != nil && *record.LastCompletedHM > *record.DueHM {
				late = true
			}
			if record.DueKM != nil && record.LastCompletedKM != nil && *record.LastCompletedKM > *record.DueKM {
				late = true
			}
			if !late {
				row.OnTime++
				total.OnTime++
			}
		}
		if record.Status == "overdue" {
			row.Overdue++
			total.Overdue++
		}
		if record.Status == "skipped" {
			row.Missed++
			total.Missed++
		}
	}
	if err := rows.Err(); err != nil {
		return Compliance{}, fmt.Errorf("iterate PM compliance: %w", err)
	}

	result := Compliance{
		Rows:      make([]ComplianceRow, 0, len(grouped)),
		Total:     total.Total,
		Completed: total.Completed,
		OnTime:    total.OnTime,
		Overdue:   total.Overdue,
		Missed:    total.Missed,
	}
	for _, row := range grouped {
		row.CompletionRate = percentage(row.Completed, row.Total)
		row.OnTimeRate = percentage(row.OnTime, row.Completed)
		result.Rows = append(result.Rows, *row)
	}
	sort.Slice(result.Rows, func(i, j int) bool {
		if result.Rows[i].PlanName == result.Rows[j].PlanName {
			return result.Rows[i].PlanID < result.Rows[j].PlanID
		}
		return result.Rows[i].PlanName < result.Rows[j].PlanName
	})
	result.CompletionRate = percentage(result.Completed, result.Total)
	result.OnTimeRate = percentage(result.OnTime, result.Completed)
	return result, nil
}

func percentage(part, total int) float64 {
	if total <= 0 {
		return 0
	}
	value := float64(part) / float64(total) * 100
	return math.Round(value*10) / 10
}

func (r *Repository) refreshScheduleStatusTx(ctx context.Context, tx pgx.Tx, id int64) error {
	record, err := loadScheduleRecord(ctx, tx, id)
	if err != nil {
		return err
	}
	if record.Status == "completed" || record.Status == "skipped" || record.Status == "cancelled" || record.Status == "wo_created" {
		return nil
	}
	r.evaluateRecord(&record, dateOnlyIn(time.Now(), r.loc), r.loadThresholdDefaults(ctx))
	status := "scheduled"
	if record.Bucket == pmcalc.BucketOverdue {
		status = "overdue"
	} else if record.Bucket == pmcalc.BucketDue {
		status = "due"
	}
	if _, err := tx.Exec(ctx, `UPDATE pm_schedules SET status = $2, updated_at = now() WHERE id = $1`, id, status); err != nil {
		return fmt.Errorf("refresh PM schedule status: %w", err)
	}
	return nil
}
