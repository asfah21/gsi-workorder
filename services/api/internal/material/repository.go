package material

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/inventory"
	"gsi-workorder/internal/notifications"
)

type Repository struct {
	pool      *pgxpool.Pool
	inventory *inventory.Repository
	loc       *time.Location
}

func NewRepository(pool *pgxpool.Pool, inv *inventory.Repository, loc *time.Location) *Repository {
	if loc == nil {
		loc = time.UTC
	}
	return &Repository{pool: pool, inventory: inv, loc: loc}
}

type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

const requirementSelect = `
	SELECT mr.id, mr.code, mr.wo_id, wo.code, wo.title, COALESCE(u.code, ''),
	       mr.source, mr.status, mr.status_changed_at, mr.shortage_since,
	       mr.requested_by, COALESCE(rb.name, ''), mr.requested_at,
	       COALESCE(mr.notes, ''), mr.created_at, mr.updated_at,
	       (SELECT count(*) FROM material_requirement_items i
	        WHERE i.material_requirement_id = mr.id),
	       (SELECT count(*) FROM material_requirement_items i
	        WHERE i.material_requirement_id = mr.id AND NOT i.is_optional),
	       (SELECT count(*) FROM v_material_readiness r
	        WHERE r.material_requirement_id = mr.id
	          AND NOT r.is_optional AND r.shortage > 0),
	       COALESCE((SELECT r.derived_status FROM v_material_readiness r
	                 WHERE r.material_requirement_id = mr.id LIMIT 1), mr.status)
	FROM material_requirements mr
	JOIN work_orders wo ON wo.id = mr.wo_id
	LEFT JOIN units u ON u.id = wo.unit_id
	LEFT JOIN users rb ON rb.id = mr.requested_by
`

const itemSelect = `
	SELECT r.item_id, r.material_requirement_id, r.sparepart_id,
	       r.sparepart_code, r.sparepart_name, r.uom,
	       r.qty_required, r.qty_reserved, r.qty_issued, r.qty_returned,
	       r.net_issued, r.available, r.coverage, r.shortage,
	       r.is_optional, r.preferred_warehouse_id,
	       COALESCE(r.preferred_warehouse_code, ''), r.notes
	FROM v_material_readiness r
	WHERE r.material_requirement_id = $1
	ORDER BY r.is_optional ASC, r.sparepart_code ASC
`

func (r *Repository) List(ctx context.Context, filter Filter) ([]Requirement, int, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	args := make([]any, 0, 5)
	where := []string{"1 = 1"}
	if filter.WOID != nil {
		args = append(args, *filter.WOID)
		where = append(where, fmt.Sprintf("mr.wo_id = $%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("mr.status = $%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Query)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf(`(mr.code ILIKE $%d OR wo.code ILIKE $%d
			OR wo.title ILIKE $%d OR COALESCE(mr.notes, '') ILIKE $%d)`, n, n, n, n))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM material_requirements mr
		JOIN work_orders wo ON wo.id = mr.wo_id
		WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count material requirements: %w", err)
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, requirementSelect+`
		WHERE `+whereSQL+`
		ORDER BY mr.created_at DESC, mr.id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list material requirements: %w", err)
	}
	defer rows.Close()
	items := make([]Requirement, 0)
	for rows.Next() {
		item, err := scanRequirement(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan material requirement: %w", err)
		}
		itemsItems, err := r.listItems(ctx, item.ID)
		if err != nil {
			return nil, 0, err
		}
		usages, err := listUsages(ctx, r.pool, item.ID)
		if err != nil {
			return nil, 0, err
		}
		item.Items = itemsItems
		item.Usages = usages
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate material requirements: %w", err)
	}
	return items, total, nil
}

func (r *Repository) Get(ctx context.Context, id int64) (Requirement, error) {
	return r.get(ctx, r.pool, id, false)
}

func (r *Repository) get(ctx context.Context, q querier, id int64, forUpdate bool) (Requirement, error) {
	query := requirementSelect + ` WHERE mr.id = $1`
	if forUpdate {
		query += ` FOR UPDATE OF mr`
	}
	item, err := scanRequirement(q.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Requirement{}, ErrNotFound
	}
	if err != nil {
		return Requirement{}, fmt.Errorf("get material requirement: %w", err)
	}
	items, err := listItems(ctx, q, id)
	if err != nil {
		return Requirement{}, err
	}
	usages, err := listUsages(ctx, q, id)
	if err != nil {
		return Requirement{}, err
	}
	item.Items = items
	item.Usages = usages
	return item, nil
}

func (r *Repository) listItems(ctx context.Context, id int64) ([]Item, error) {
	return listItems(ctx, r.pool, id)
}

func listItems(ctx context.Context, q querier, id int64) ([]Item, error) {
	rows, err := q.Query(ctx, itemSelect, id)
	if err != nil {
		return nil, fmt.Errorf("list material items: %w", err)
	}
	defer rows.Close()
	items := make([]Item, 0)
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.RequirementID, &item.SparepartID,
			&item.SparepartCode, &item.SparepartName, &item.UOM,
			&item.QtyRequired, &item.QtyReserved, &item.QtyIssued, &item.QtyReturned,
			&item.NetIssued, &item.Available, &item.Coverage, &item.Shortage,
			&item.IsOptional, &item.PreferredWarehouseID,
			&item.PreferredWarehouseCode, &item.Notes); err != nil {
			return nil, fmt.Errorf("scan material item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate material items: %w", err)
	}
	return items, nil
}

func (r *Repository) ListUpdates(ctx context.Context, id int64) ([]UpdateRow, error) {
	if err := r.requireRequirement(ctx, id); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.material_requirement_id, u.user_id, COALESCE(usr.name, ''),
		       COALESCE(u.status_from, ''), u.status_to, COALESCE(u.note, ''), u.created_at
		FROM material_requirement_updates u
		LEFT JOIN users usr ON usr.id = u.user_id
		WHERE u.material_requirement_id = $1
		ORDER BY u.created_at DESC, u.id DESC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("list material updates: %w", err)
	}
	defer rows.Close()
	items := make([]UpdateRow, 0)
	for rows.Next() {
		var item UpdateRow
		if err := rows.Scan(&item.ID, &item.RequirementID, &item.UserID, &item.UserName,
			&item.StatusFrom, &item.StatusTo, &item.Note, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan material update: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate material updates: %w", err)
	}
	return items, nil
}

func (r *Repository) ListUsages(ctx context.Context, id int64) ([]Usage, error) {
	if err := r.requireRequirement(ctx, id); err != nil {
		return nil, err
	}
	return listUsages(ctx, r.pool, id)
}

func listUsages(ctx context.Context, q querier, id int64) ([]Usage, error) {
	rows, err := q.Query(ctx, `
		SELECT u.id, u.wo_id, u.material_requirement_item_id,
		       u.sparepart_id, s.code, s.name,
		       u.component_id, COALESCE(c.name, ''),
		       u.warehouse_location_id, w.code, wl.code,
		       u.qty::float8, u.hm::float8, u.km::float8, u.movement_type,
		       u.inventory_transaction_item_id, u.return_of_usage_id,
		       u.used_by, COALESCE(usr.name, ''), u.used_at, COALESCE(u.notes, '')
		FROM work_order_sparepart_usage u
		JOIN work_orders wo ON wo.id = u.wo_id
		JOIN material_requirement_items mri ON mri.id = u.material_requirement_item_id
		JOIN spareparts s ON s.id = u.sparepart_id
		JOIN warehouse_locations wl ON wl.id = u.warehouse_location_id
		JOIN warehouses w ON w.id = wl.warehouse_id
		LEFT JOIN components c ON c.id = u.component_id
		LEFT JOIN users usr ON usr.id = u.used_by
		WHERE mri.material_requirement_id = $1
		ORDER BY u.used_at DESC, u.id DESC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("list material usages: %w", err)
	}
	defer rows.Close()
	items := make([]Usage, 0)
	for rows.Next() {
		var item Usage
		if err := rows.Scan(&item.ID, &item.WOID, &item.RequirementItemID,
			&item.SparepartID, &item.SparepartCode, &item.SparepartName,
			&item.ComponentID, &item.ComponentName,
			&item.LocationID, &item.WarehouseCode, &item.LocationCode,
			&item.Qty, &item.HM, &item.KM, &item.MovementType,
			&item.InventoryItemID, &item.ReturnOfUsageID,
			&item.UsedBy, &item.UsedByName, &item.UsedAt, &item.Notes); err != nil {
			return nil, fmt.Errorf("scan material usage: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate material usages: %w", err)
	}
	return items, nil
}

func scanRequirement(row interface{ Scan(...any) error }) (Requirement, error) {
	var item Requirement
	if err := row.Scan(&item.ID, &item.Code, &item.WOID, &item.WOCode, &item.WOTitle,
		&item.UnitCode, &item.Source, &item.Status, &item.StatusChangedAt,
		&item.ShortageSince, &item.RequestedBy, &item.RequestedByName,
		&item.RequestedAt, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
		&item.ItemCount, &item.RequiredCount, &item.ShortageCount,
		&item.DerivedStatus); err != nil {
		return Requirement{}, err
	}
	item.Items = make([]Item, 0)
	item.Usages = make([]Usage, 0)
	return item, nil
}

func (r *Repository) requireRequirement(ctx context.Context, id int64) error {
	var exists int64
	if err := r.pool.QueryRow(ctx, `SELECT id FROM material_requirements WHERE id = $1`, id).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("check material requirement: %w", err)
	}
	return nil
}

func (r *Repository) Create(ctx context.Context, actor Actor, input CreateInput) (Requirement, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return Requirement{}, ErrValidation
	}
	if input.WOID <= 0 {
		return Requirement{}, &ValidationError{Fields: map[string]string{"wo_id": "Work order wajib dipilih."}}
	}
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "manual"
	}
	if source != "manual" && source != "pm" {
		return Requirement{}, &ValidationError{Fields: map[string]string{"source": "Source material tidak valid."}}
	}
	normalizedItems, err := normalizeItems(input.Items)
	if err != nil {
		return Requirement{}, err
	}
	notes := strings.TrimSpace(input.Notes)
	if len(notes) > 1000 {
		return Requirement{}, &ValidationError{Fields: map[string]string{"notes": "Catatan maksimal 1000 karakter."}}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Requirement{}, fmt.Errorf("begin material create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var woStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM work_orders WHERE id = $1 FOR UPDATE`, input.WOID).Scan(&woStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Requirement{}, &ValidationError{Fields: map[string]string{"wo_id": "Work order tidak ditemukan."}}
		}
		return Requirement{}, fmt.Errorf("lock material work order: %w", err)
	}
	if woStatus == "done" || woStatus == "cancelled" {
		return Requirement{}, &ValidationError{Fields: map[string]string{"wo_id": "Work order sudah selesai atau dibatalkan."}}
	}
	if err := validateItemsReferences(ctx, tx, normalizedItems); err != nil {
		return Requirement{}, err
	}
	code, err := nextRequirementCode(ctx, tx)
	if err != nil {
		return Requirement{}, err
	}
	var id int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO material_requirements (
			code, wo_id, source, status, requested_by, requested_at, notes
		)
		VALUES ($1, $2, $3, 'draft', $4, now(), NULLIF($5, ''))
		RETURNING id
	`, code, input.WOID, source, actor.ID, notes).Scan(&id); err != nil {
		if isUniqueViolation(err) {
			return Requirement{}, ErrConflict
		}
		return Requirement{}, fmt.Errorf("insert material requirement: %w", err)
	}
	if err := insertItems(ctx, tx, id, normalizedItems); err != nil {
		return Requirement{}, err
	}
	if err := insertUpdate(ctx, tx, id, actor.ID, "", "draft", "Material requirement dibuat"); err != nil {
		return Requirement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Requirement{}, fmt.Errorf("commit material create: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *Repository) Update(ctx context.Context, id int64, actor Actor, patch PatchInput) (Requirement, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return Requirement{}, ErrValidation
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Requirement{}, fmt.Errorf("begin material update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := r.get(ctx, tx, id, true)
	if err != nil {
		return Requirement{}, err
	}
	if current.Status == "cancelled" || current.Status == "completed" {
		return Requirement{}, ErrInvalidState
	}
	if patch.Items != nil {
		if current.Status != "draft" && current.Status != "planned" {
			return Requirement{}, &ValidationError{Fields: map[string]string{"items": "Item hanya dapat diubah pada status draft/planned."}}
		}
		normalized, err := normalizeItems(*patch.Items)
		if err != nil {
			return Requirement{}, err
		}
		if err := validateItemsReferences(ctx, tx, normalized); err != nil {
			return Requirement{}, err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM material_requirement_items WHERE material_requirement_id = $1`, id); err != nil {
			return Requirement{}, fmt.Errorf("replace material items: %w", err)
		}
		if err := insertItems(ctx, tx, id, normalized); err != nil {
			return Requirement{}, err
		}
		current.ItemCount = len(normalized)
	}
	if patch.Notes != nil {
		notes := strings.TrimSpace(*patch.Notes)
		if len(notes) > 1000 {
			return Requirement{}, &ValidationError{Fields: map[string]string{"notes": "Catatan maksimal 1000 karakter."}}
		}
		if _, err := tx.Exec(ctx, `UPDATE material_requirements SET notes = NULLIF($2, ''), updated_at = now() WHERE id = $1`, id, notes); err != nil {
			return Requirement{}, fmt.Errorf("update material notes: %w", err)
		}
	}
	if patch.Status != nil && strings.TrimSpace(*patch.Status) != "" && *patch.Status != current.Status {
		if err := r.transitionLocked(ctx, tx, current, actor, strings.TrimSpace(*patch.Status), "Perubahan status manual"); err != nil {
			return Requirement{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Requirement{}, fmt.Errorf("commit material update: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *Repository) transitionLocked(ctx context.Context, tx pgx.Tx, current Requirement, actor Actor, target, note string) error {
	switch {
	case current.Status == "draft" && target == "planned":
		if current.ItemCount == 0 {
			return &ValidationError{Fields: map[string]string{"items": "Minimal satu item sebelum planned."}}
		}
		return r.setStatusLocked(ctx, tx, current, actor.ID, target, note, false)
	case current.Status == "planned" && target == "checking_stock":
		return r.recomputeLocked(ctx, tx, current.ID, actor.ID, true, true, note)
	case current.Status == "checking_stock" && target == "checking_stock":
		return r.recomputeLocked(ctx, tx, current.ID, actor.ID, true, true, note)
	case (current.Status == "ready" || current.Status == "partially_ready" || current.Status == "shortage") && target == "checking_stock":
		return r.recomputeLocked(ctx, tx, current.ID, actor.ID, true, true, note)
	default:
		return &ValidationError{Fields: map[string]string{"status": "Transisi status material tidak valid."}}
	}
}

func (r *Repository) CheckStock(ctx context.Context, id int64, actor Actor) (Requirement, error) {
	if actor.Role != "admin" && actor.Role != "planner" && actor.Role != "storekeeper" {
		return Requirement{}, ErrValidation
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Requirement{}, fmt.Errorf("begin material check stock: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := r.get(ctx, tx, id, true)
	if err != nil {
		return Requirement{}, err
	}
	if current.Status == "cancelled" || current.Status == "completed" {
		return Requirement{}, ErrInvalidState
	}
	if current.ItemCount == 0 {
		return Requirement{}, &ValidationError{Fields: map[string]string{"items": "Material requirement belum memiliki item."}}
	}
	if err := r.recomputeLocked(ctx, tx, id, actor.ID, true, true, "Check stock"); err != nil {
		return Requirement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Requirement{}, fmt.Errorf("commit material check stock: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *Repository) recomputeLocked(ctx context.Context, tx pgx.Tx, id, actorID int64, allowWorkflow, forceAudit bool, note string) error {
	var currentStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM material_requirements WHERE id = $1 FOR UPDATE`, id).Scan(&currentStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("lock material requirement status: %w", err)
	}
	if currentStatus == "cancelled" || currentStatus == "completed" {
		return nil
	}
	if !allowWorkflow && (currentStatus == "draft" || currentStatus == "planned") {
		return nil
	}
	derived, missing, err := derivedStatus(ctx, tx, id)
	if err != nil {
		return err
	}
	next := derived
	if missing {
		if currentStatus == "draft" || currentStatus == "planned" {
			next = currentStatus
		} else {
			next = "ready"
		}
	}
	if !allowWorkflow && (next == "draft" || next == "planned" || next == "checking_stock") {
		return nil
	}
	var shortageSince *time.Time
	if next == "shortage" {
		if currentStatus != "shortage" {
			now := time.Now().UTC()
			shortageSince = &now
		} else {
			var existing *time.Time
			if err := tx.QueryRow(ctx, `SELECT shortage_since FROM material_requirements WHERE id = $1`, id).Scan(&existing); err != nil {
				return fmt.Errorf("read material shortage since: %w", err)
			}
			shortageSince = existing
		}
	}
	if next != currentStatus {
		if _, err := tx.Exec(ctx, `
			UPDATE material_requirements
			SET status = $2, status_changed_at = now(), shortage_since = $3, updated_at = now()
			WHERE id = $1
		`, id, next, shortageSince); err != nil {
			return fmt.Errorf("update material readiness status: %w", err)
		}
		if err := insertUpdate(ctx, tx, id, actorID, currentStatus, next, note); err != nil {
			return err
		}
		if err := r.notifyStatus(ctx, tx, id, next); err != nil {
			return err
		}
	} else if next == "shortage" && shortageSince != nil {
		if _, err := tx.Exec(ctx, `UPDATE material_requirements SET shortage_since = $2, updated_at = now() WHERE id = $1`, id, shortageSince); err != nil {
			return fmt.Errorf("set material shortage since: %w", err)
		}
	}
	if next != "shortage" {
		if _, err := tx.Exec(ctx, `UPDATE material_requirements SET shortage_since = NULL WHERE id = $1 AND shortage_since IS NOT NULL`, id); err != nil {
			return fmt.Errorf("clear material shortage since: %w", err)
		}
	}
	if forceAudit && next == currentStatus {
		if err := insertUpdate(ctx, tx, id, actorID, currentStatus, next, note); err != nil {
			return err
		}
	}
	return nil
}

func derivedStatus(ctx context.Context, q querier, id int64) (string, bool, error) {
	var status string
	err := q.QueryRow(ctx, `
		SELECT derived_status FROM v_material_readiness
		WHERE material_requirement_id = $1
		LIMIT 1
	`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", true, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read material readiness: %w", err)
	}
	return status, false, nil
}

func (r *Repository) setStatusLocked(ctx context.Context, tx pgx.Tx, current Requirement, actorID int64, target, note string, allowShortage bool) error {
	if target == "shortage" && !allowShortage {
		return &ValidationError{Fields: map[string]string{"status": "Shortage hanya boleh dihitung sistem."}}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE material_requirements
		SET status = $2, status_changed_at = now(), updated_at = now()
		WHERE id = $1
	`, current.ID, target); err != nil {
		return fmt.Errorf("set material status: %w", err)
	}
	if err := insertUpdate(ctx, tx, current.ID, actorID, current.Status, target, note); err != nil {
		return err
	}
	return r.notifyStatus(ctx, tx, current.ID, target)
}

func insertUpdate(ctx context.Context, tx pgx.Tx, requirementID, actorID int64, from, to, note string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO material_requirement_updates (
			material_requirement_id, user_id, status_from, status_to, note
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''))
	`, requirementID, nullActor(actorID), from, to, strings.TrimSpace(note)); err != nil {
		return fmt.Errorf("append material update: %w", err)
	}
	return nil
}

func (r *Repository) notifyStatus(ctx context.Context, tx pgx.Tx, id int64, status string) error {
	roles := []string{}
	switch status {
	case "shortage", "partially_ready":
		roles = []string{"planner", "storekeeper"}
	case "ready", "reserved":
		roles = []string{"planner", "mechanic"}
	default:
		return nil
	}
	var code, woCode string
	if err := tx.QueryRow(ctx, `
		SELECT mr.code, wo.code
		FROM material_requirements mr
		JOIN work_orders wo ON wo.id = mr.wo_id
		WHERE mr.id = $1
	`, id).Scan(&code, &woCode); err != nil {
		return fmt.Errorf("load material notification target: %w", err)
	}
	day := time.Now().In(r.loc).Format("2006-01-02")
	title := "Material " + strings.ReplaceAll(status, "_", " ")
	body := fmt.Sprintf("%s untuk %s berstatus %s.", code, woCode, status)
	_, err := notifications.CreateForRoles(ctx, tx, roles, notifications.CreateInput{
		Type:      "material_" + status,
		Title:     title,
		Body:      body,
		RefType:   "material_requirement",
		RefID:     &id,
		DedupeKey: fmt.Sprintf("mat:%d:%s:%s", id, status, day),
	}, nil)
	if err != nil {
		return fmt.Errorf("notify material status: %w", err)
	}
	return nil
}

func nullActor(id int64) *int64 {
	if id <= 0 {
		return nil
	}
	return &id
}

func nextRequirementCode(ctx context.Context, tx pgx.Tx) (string, error) {
	var code string
	if err := tx.QueryRow(ctx, `
		SELECT 'MR-' || to_char(now() AT TIME ZONE 'Asia/Makassar', 'YYYYMM') || '-'
		       || lpad(nextval('material_requirement_code_seq')::text, 4, '0')
	`).Scan(&code); err != nil {
		return "", fmt.Errorf("next material requirement code: %w", err)
	}
	return code, nil
}

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func normalizeItems(input []ItemInput) ([]ItemInput, error) {
	fields := map[string]string{}
	seen := map[int64]struct{}{}
	items := make([]ItemInput, 0, len(input))
	for index, item := range input {
		item.Notes = strings.TrimSpace(item.Notes)
		if item.SparepartID <= 0 {
			fields[fmt.Sprintf("items.%d.sparepart_id", index)] = "Sparepart wajib dipilih."
		}
		if item.QtyRequired <= 0 {
			fields[fmt.Sprintf("items.%d.qty_required", index)] = "Qty required harus lebih dari 0."
		}
		if len(item.Notes) > 500 {
			fields[fmt.Sprintf("items.%d.notes", index)] = "Catatan maksimal 500 karakter."
		}
		if _, exists := seen[item.SparepartID]; item.SparepartID > 0 && exists {
			fields[fmt.Sprintf("items.%d.sparepart_id", index)] = "Sparepart duplikat."
		}
		if item.SparepartID > 0 {
			seen[item.SparepartID] = struct{}{}
		}
		items = append(items, item)
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}
	return items, nil
}

func validateItemsReferences(ctx context.Context, tx pgx.Tx, items []ItemInput) error {
	for _, item := range items {
		var exists int64
		if err := tx.QueryRow(ctx, `SELECT id FROM spareparts WHERE id = $1 AND is_active = true`, item.SparepartID).Scan(&exists); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return &ValidationError{Fields: map[string]string{"sparepart_id": "Sparepart tidak ditemukan atau tidak aktif."}}
			}
			return fmt.Errorf("check material sparepart: %w", err)
		}
		if item.PreferredWarehouseID != nil {
			if err := tx.QueryRow(ctx, `SELECT id FROM warehouses WHERE id = $1 AND is_active = true`, *item.PreferredWarehouseID).Scan(&exists); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return &ValidationError{Fields: map[string]string{"preferred_warehouse_id": "Gudang tidak ditemukan atau tidak aktif."}}
				}
				return fmt.Errorf("check material warehouse: %w", err)
			}
		}
	}
	return nil
}

func insertItems(ctx context.Context, tx pgx.Tx, requirementID int64, items []ItemInput) error {
	for _, item := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO material_requirement_items (
				material_requirement_id, sparepart_id, qty_required,
				is_optional, preferred_warehouse_id, notes
			)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		`, requirementID, item.SparepartID, item.QtyRequired, item.IsOptional,
			item.PreferredWarehouseID, item.Notes); err != nil {
			if isUniqueViolation(err) {
				return ErrConflict
			}
			return fmt.Errorf("insert material item: %w", err)
		}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	type sqlState interface{ SQLState() string }
	var state sqlState
	return errors.As(err, &state) && state.SQLState() == "23505"
}
