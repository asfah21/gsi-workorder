package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) CreateOpname(ctx context.Context, actor Actor, input OpnameCreateInput) (OpnameSession, error) {
	if !canCountStock(actor.Role) {
		return OpnameSession{}, ErrValidation
	}
	if input.WarehouseID <= 0 {
		return OpnameSession{}, &ValidationError{Fields: map[string]string{"warehouse_id": "Gudang wajib dipilih."}}
	}
	input.Notes = trim(input.Notes)
	if len(input.Notes) > 1000 {
		return OpnameSession{}, &ValidationError{Fields: map[string]string{"notes": "Catatan maksimal 1000 karakter."}}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return OpnameSession{}, fmt.Errorf("begin stock opname create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var warehouseCode string
	if err := tx.QueryRow(ctx, `
		SELECT code FROM warehouses WHERE id = $1 AND is_active = true
	`, input.WarehouseID).Scan(&warehouseCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OpnameSession{}, &ValidationError{Fields: map[string]string{"warehouse_id": "Gudang tidak ditemukan atau tidak aktif."}}
		}
		return OpnameSession{}, fmt.Errorf("check opname warehouse: %w", err)
	}
	code, err := nextOpnameCode(ctx, tx)
	if err != nil {
		return OpnameSession{}, err
	}
	var id int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO stock_opname_sessions (
			code, warehouse_id, status, started_at, notes, created_by
		)
		VALUES ($1, $2, 'counting', now(), NULLIF($3, ''), $4)
		RETURNING id
	`, code, input.WarehouseID, input.Notes, actor.ID).Scan(&id); err != nil {
		return OpnameSession{}, fmt.Errorf("insert stock opname session: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO stock_opname_items (session_id, sparepart_id, location_id, system_qty)
		SELECT $1, x.sparepart_id, x.location_id, x.system_qty
		FROM (
			SELECT b.sparepart_id, b.warehouse_location_id AS location_id, b.on_hand AS system_qty
			FROM inventory_balances b
			JOIN spareparts s ON s.id = b.sparepart_id
			JOIN warehouse_locations wl ON wl.id = b.warehouse_location_id
			JOIN warehouses w ON w.id = wl.warehouse_id
			WHERE wl.warehouse_id = $2
			  AND s.is_active = true
			  AND wl.is_active = true
			  AND w.is_active = true
			UNION ALL
			SELECT p.sparepart_id, p.warehouse_location_id, 0::numeric(14,3)
			FROM inventory_stock_policies p
			JOIN spareparts s ON s.id = p.sparepart_id
			JOIN warehouse_locations wl ON wl.id = p.warehouse_location_id
			JOIN warehouses w ON w.id = wl.warehouse_id
			WHERE wl.warehouse_id = $2
			  AND p.is_active = true
			  AND s.is_active = true
			  AND wl.is_active = true
			  AND w.is_active = true
			  AND NOT EXISTS (
				SELECT 1 FROM inventory_balances b
				WHERE b.sparepart_id = p.sparepart_id
				  AND b.warehouse_location_id = p.warehouse_location_id
			  )
		) x
	`, id, input.WarehouseID); err != nil {
		return OpnameSession{}, fmt.Errorf("snapshot stock opname items: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OpnameSession{}, fmt.Errorf("commit stock opname create: %w", err)
	}
	return r.GetOpnameSession(ctx, id)
}

func (r *Repository) ListOpnameSessions(ctx context.Context, filter OpnameFilter) ([]OpnameSession, int, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	args := make([]any, 0, 3)
	where := []string{"1 = 1"}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("s.status = $%d", len(args)))
	}
	if filter.WarehouseID != nil {
		args = append(args, *filter.WarehouseID)
		where = append(where, fmt.Sprintf("s.warehouse_id = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM stock_opname_sessions s WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stock opname sessions: %w", err)
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, opnameSessionSelect+`
		WHERE `+whereSQL+`
		ORDER BY s.created_at DESC, s.id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stock opname sessions: %w", err)
	}
	defer rows.Close()
	items := make([]OpnameSession, 0)
	for rows.Next() {
		item, err := scanOpnameSession(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan stock opname session: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate stock opname sessions: %w", err)
	}
	return items, total, nil
}

func (r *Repository) GetOpnameSession(ctx context.Context, id int64) (OpnameSession, error) {
	item, err := scanOpnameSession(r.pool.QueryRow(ctx, opnameSessionSelect+` WHERE s.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return OpnameSession{}, ErrNotFound
	}
	if err != nil {
		return OpnameSession{}, fmt.Errorf("get stock opname session: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT si.id, si.session_id, si.sparepart_id, sp.code, sp.name, sp.uom,
		       si.location_id, w.code, wl.code, wl.name,
		       si.system_qty::float8, si.counted_qty::float8, si.difference::float8,
		       COALESCE(si.notes, '')
		FROM stock_opname_items si
		JOIN spareparts sp ON sp.id = si.sparepart_id
		JOIN warehouse_locations wl ON wl.id = si.location_id
		JOIN warehouses w ON w.id = wl.warehouse_id
		WHERE si.session_id = $1
		ORDER BY sp.code ASC, w.code ASC, wl.code ASC
	`, id)
	if err != nil {
		return OpnameSession{}, fmt.Errorf("list stock opname items: %w", err)
	}
	defer rows.Close()
	lines := make([]OpnameItem, 0)
	for rows.Next() {
		var line OpnameItem
		if err := rows.Scan(&line.ID, &line.SessionID, &line.SparepartID,
			&line.SparepartCode, &line.SparepartName, &line.UOM,
			&line.LocationID, &line.WarehouseCode, &line.LocationCode, &line.LocationName,
			&line.SystemQty, &line.CountedQty, &line.Difference, &line.Notes); err != nil {
			return OpnameSession{}, fmt.Errorf("scan stock opname item: %w", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return OpnameSession{}, fmt.Errorf("iterate stock opname items: %w", err)
	}
	item.Items = lines
	item.ItemCount = len(lines)
	item.CountedCount = 0
	for _, line := range lines {
		if line.CountedQty != nil {
			item.CountedCount++
		}
	}
	return item, nil
}

func (r *Repository) CountOpname(ctx context.Context, id int64, actor Actor, input OpnameCountInput) (OpnameSession, error) {
	if !canCountStock(actor.Role) {
		return OpnameSession{}, ErrValidation
	}
	if len(input.Items) == 0 {
		return OpnameSession{}, &ValidationError{Fields: map[string]string{"items": "Minimal satu item hitung wajib diisi."}}
	}
	if len(input.Items) > 500 {
		return OpnameSession{}, &ValidationError{Fields: map[string]string{"items": "Maksimal 500 item per penyimpanan."}}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return OpnameSession{}, fmt.Errorf("begin stock opname count: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	if err := tx.QueryRow(ctx, `
		SELECT status FROM stock_opname_sessions WHERE id = $1 FOR UPDATE
	`, id).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OpnameSession{}, ErrNotFound
		}
		return OpnameSession{}, fmt.Errorf("lock stock opname session: %w", err)
	}
	if status != "counting" && status != "review" {
		return OpnameSession{}, ErrInvalidState
	}
	for _, item := range input.Items {
		if item.ItemID <= 0 {
			return OpnameSession{}, &ValidationError{Fields: map[string]string{"items": "Item opname tidak valid."}}
		}
		if item.CountedQty < 0 {
			return OpnameSession{}, &ValidationError{Fields: map[string]string{"counted_qty": "Qty hitung tidak boleh negatif."}}
		}
		if len(trim(item.Notes)) > 500 {
			return OpnameSession{}, &ValidationError{Fields: map[string]string{"notes": "Catatan item maksimal 500 karakter."}}
		}
		tag, err := tx.Exec(ctx, `
			UPDATE stock_opname_items
			SET counted_qty = $3, notes = NULLIF($4, '')
			WHERE id = $1 AND session_id = $2
		`, item.ItemID, id, item.CountedQty, trim(item.Notes))
		if err != nil {
			return OpnameSession{}, fmt.Errorf("update stock opname item: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return OpnameSession{}, ErrNotFound
		}
	}
	var total, counted int
	if err := tx.QueryRow(ctx, `
		SELECT count(*), count(counted_qty)
		FROM stock_opname_items
		WHERE session_id = $1
	`, id).Scan(&total, &counted); err != nil {
		return OpnameSession{}, fmt.Errorf("count stock opname progress: %w", err)
	}
	nextStatus := "counting"
	if total == 0 || counted == total {
		nextStatus = "review"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE stock_opname_sessions SET status = $2, updated_at = now() WHERE id = $1
	`, id, nextStatus); err != nil {
		return OpnameSession{}, fmt.Errorf("update stock opname status: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OpnameSession{}, fmt.Errorf("commit stock opname count: %w", err)
	}
	return r.GetOpnameSession(ctx, id)
}

func (r *Repository) PostOpname(ctx context.Context, id int64, actor Actor) (OpnameSession, error) {
	if !canApproveStock(actor.Role) {
		return OpnameSession{}, ErrValidation
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return OpnameSession{}, fmt.Errorf("begin stock opname post: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var code, status string
	if err := tx.QueryRow(ctx, `
		SELECT code, status FROM stock_opname_sessions WHERE id = $1 FOR UPDATE
	`, id).Scan(&code, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OpnameSession{}, ErrNotFound
		}
		return OpnameSession{}, fmt.Errorf("lock stock opname session: %w", err)
	}
	if status != "counting" && status != "review" {
		return OpnameSession{}, ErrInvalidState
	}
	rows, err := tx.Query(ctx, `
		SELECT sparepart_id, location_id, system_qty::float8, counted_qty::float8
		FROM stock_opname_items
		WHERE session_id = $1
	`, id)
	if err != nil {
		return OpnameSession{}, fmt.Errorf("list stock opname differences: %w", err)
	}
	type difference struct {
		SparepartID int64
		LocationID  int64
		SystemQty   float64
		CountedQty  float64
	}
	differences := make([]difference, 0)
	for rows.Next() {
		var item difference
		var counted *float64
		if err := rows.Scan(&item.SparepartID, &item.LocationID, &item.SystemQty, &counted); err != nil {
			rows.Close()
			return OpnameSession{}, fmt.Errorf("scan stock opname difference: %w", err)
		}
		if counted == nil {
			rows.Close()
			return OpnameSession{}, &ValidationError{Fields: map[string]string{"items": "Semua item harus dihitung sebelum posting."}}
		}
		item.CountedQty = *counted
		differences = append(differences, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return OpnameSession{}, fmt.Errorf("iterate stock opname differences: %w", err)
	}
	rows.Close()

	items := make([]PostItem, 0, len(differences))
	for _, item := range differences {
		diff := item.CountedQty - item.SystemQty
		if diff == 0 {
			continue
		}
		line := PostItem{
			SparepartID: item.SparepartID,
			Qty:         diff,
			Notes:       fmt.Sprintf("Opname: sistem %.3f, hitung %.3f", item.SystemQty, item.CountedQty),
		}
		if diff < 0 {
			locationID := item.LocationID
			line.FromLocationID = &locationID
			line.Qty = -diff
		} else {
			locationID := item.LocationID
			line.ToLocationID = &locationID
		}
		items = append(items, line)
	}
	if len(items) > 0 {
		if err := validatePostReferences(ctx, tx, items); err != nil {
			return OpnameSession{}, err
		}
		if _, err := r.postLedgerTx(ctx, tx, actor.ID, "opname", "stock_opname",
			&id, code, "Stock opname "+code, "", items, nil, false); err != nil {
			return OpnameSession{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE stock_opname_sessions
		SET status = 'posted', posted_at = now(), posted_by = $2, updated_at = now()
		WHERE id = $1
	`, id, actor.ID); err != nil {
		return OpnameSession{}, fmt.Errorf("mark stock opname posted: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OpnameSession{}, fmt.Errorf("commit stock opname post: %w", err)
	}
	return r.GetOpnameSession(ctx, id)
}

func (r *Repository) CancelOpname(ctx context.Context, id int64, actor Actor, reason string) (OpnameSession, error) {
	if !canApproveStock(actor.Role) {
		return OpnameSession{}, ErrValidation
	}
	reason = trim(reason)
	if reason == "" || len(reason) > 1000 {
		return OpnameSession{}, &ValidationError{Fields: map[string]string{"reason": "Alasan pembatalan wajib 1-1000 karakter."}}
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE stock_opname_sessions
		SET status = 'cancelled', cancel_reason = $2, cancelled_by = $3,
		    cancelled_at = now(), updated_at = now()
		WHERE id = $1 AND status IN ('draft', 'counting', 'review')
	`, id, reason, actor.ID)
	if err != nil {
		return OpnameSession{}, fmt.Errorf("cancel stock opname: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return OpnameSession{}, ErrInvalidState
	}
	return r.GetOpnameSession(ctx, id)
}

const opnameSessionSelect = `
	SELECT s.id, s.code, s.warehouse_id, w.code, w.name, s.status,
	       s.started_at, s.posted_at, s.posted_by, COALESCE(u.name, ''),
	       COALESCE(s.notes, ''), COALESCE(s.cancel_reason, ''), s.cancelled_at,
	       s.created_by, s.created_at, s.updated_at,
	       (SELECT count(*) FROM stock_opname_items si WHERE si.session_id = s.id),
	       (SELECT count(*) FROM stock_opname_items si
	        WHERE si.session_id = s.id AND si.counted_qty IS NOT NULL)
	FROM stock_opname_sessions s
	JOIN warehouses w ON w.id = s.warehouse_id
	LEFT JOIN users u ON u.id = s.posted_by
`

func scanOpnameSession(row interface{ Scan(...any) error }) (OpnameSession, error) {
	var item OpnameSession
	if err := row.Scan(&item.ID, &item.Code, &item.WarehouseID, &item.WarehouseCode,
		&item.WarehouseName, &item.Status, &item.StartedAt, &item.PostedAt,
		&item.PostedBy, &item.PostedByName, &item.Notes, &item.CancelReason,
		&item.CancelledAt, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
		&item.ItemCount, &item.CountedCount); err != nil {
		return OpnameSession{}, err
	}
	item.Items = make([]OpnameItem, 0)
	return item, nil
}

func nextOpnameCode(ctx context.Context, tx pgx.Tx) (string, error) {
	var code string
	if err := tx.QueryRow(ctx, `
		SELECT 'OPN-' || to_char(now() AT TIME ZONE 'Asia/Makassar', 'YYYYMM') || '-'
		       || lpad(nextval('stock_opname_code_seq')::text, 4, '0')
	`).Scan(&code); err != nil {
		return "", fmt.Errorf("next stock opname code: %w", err)
	}
	return code, nil
}
