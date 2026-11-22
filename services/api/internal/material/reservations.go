package material

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"gsi-workorder/internal/inventory"
)

const reservationSelect = `
	SELECT ir.id, ir.material_requirement_item_id, mri.material_requirement_id,
	       mr.code, mr.wo_id, wo.code,
	       ir.sparepart_id, s.code, s.name, s.uom,
	       ir.warehouse_location_id, w.code, wl.code, wl.name,
	       ir.qty::float8, ir.status,
	       ir.reserved_by, COALESCE(usr.name, ''), ir.reserved_at,
	       ir.released_at, ir.consumed_at, COALESCE(ir.notes, '')
	FROM inventory_reservations ir
	JOIN material_requirement_items mri ON mri.id = ir.material_requirement_item_id
	JOIN material_requirements mr ON mr.id = mri.material_requirement_id
	JOIN work_orders wo ON wo.id = mr.wo_id
	JOIN spareparts s ON s.id = ir.sparepart_id
	JOIN warehouse_locations wl ON wl.id = ir.warehouse_location_id
	JOIN warehouses w ON w.id = wl.warehouse_id
	LEFT JOIN users usr ON usr.id = ir.reserved_by
`

func (r *Repository) ListReservations(ctx context.Context, filter ReservationFilter) ([]Reservation, int, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	args := make([]any, 0, 6)
	where := []string{"1 = 1"}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("ir.status = $%d", len(args)))
	}
	if filter.SparepartID != nil {
		args = append(args, *filter.SparepartID)
		where = append(where, fmt.Sprintf("ir.sparepart_id = $%d", len(args)))
	}
	if filter.RequirementID != nil {
		args = append(args, *filter.RequirementID)
		where = append(where, fmt.Sprintf("mri.material_requirement_id = $%d", len(args)))
	}
	if filter.WOID != nil {
		args = append(args, *filter.WOID)
		where = append(where, fmt.Sprintf("mr.wo_id = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM inventory_reservations ir
		JOIN material_requirement_items mri ON mri.id = ir.material_requirement_item_id
		JOIN material_requirements mr ON mr.id = mri.material_requirement_id
		WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count inventory reservations: %w", err)
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, reservationSelect+`
		WHERE `+whereSQL+`
		ORDER BY ir.reserved_at DESC, ir.id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list inventory reservations: %w", err)
	}
	defer rows.Close()
	items := make([]Reservation, 0)
	for rows.Next() {
		item, err := scanReservation(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan inventory reservation: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate inventory reservations: %w", err)
	}
	return items, total, nil
}

func (r *Repository) getReservation(ctx context.Context, q querier, id int64, forUpdate bool) (Reservation, error) {
	query := reservationSelect + ` WHERE ir.id = $1`
	if forUpdate {
		query += ` FOR UPDATE OF ir`
	}
	item, err := scanReservation(q.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Reservation{}, ErrNotFound
	}
	if err != nil {
		return Reservation{}, fmt.Errorf("get inventory reservation: %w", err)
	}
	return item, nil
}

func scanReservation(row interface{ Scan(...any) error }) (Reservation, error) {
	var item Reservation
	if err := row.Scan(&item.ID, &item.RequirementItemID, &item.RequirementID,
		&item.RequirementCode, &item.WOID, &item.WOCode,
		&item.SparepartID, &item.SparepartCode, &item.SparepartName, &item.UOM,
		&item.LocationID, &item.WarehouseCode, &item.LocationCode, &item.LocationName,
		&item.Qty, &item.Status, &item.ReservedBy, &item.ReservedByName,
		&item.ReservedAt, &item.ReleasedAt, &item.ConsumedAt, &item.Notes); err != nil {
		return Reservation{}, err
	}
	return item, nil
}

type lockedItem struct {
	ID                   int64
	SparepartID          int64
	QtyRequired          float64
	QtyReserved          float64
	QtyIssued            float64
	QtyReturned          float64
	IsOptional           bool
	PreferredWarehouseID *int64
}

func loadLockedItems(ctx context.Context, tx pgx.Tx, requirementID int64) ([]lockedItem, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, sparepart_id, qty_required::float8, qty_reserved::float8,
		       qty_issued::float8, qty_returned::float8, is_optional,
		       preferred_warehouse_id
		FROM material_requirement_items
		WHERE material_requirement_id = $1
		ORDER BY is_optional ASC, id ASC
		FOR UPDATE
	`, requirementID)
	if err != nil {
		return nil, fmt.Errorf("lock material items: %w", err)
	}
	defer rows.Close()
	items := make([]lockedItem, 0)
	for rows.Next() {
		var item lockedItem
		if err := rows.Scan(&item.ID, &item.SparepartID, &item.QtyRequired,
			&item.QtyReserved, &item.QtyIssued, &item.QtyReturned,
			&item.IsOptional, &item.PreferredWarehouseID); err != nil {
			return nil, fmt.Errorf("scan locked material item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locked material items: %w", err)
	}
	return items, nil
}

type availableLocation struct {
	LocationID    int64
	WarehouseCode string
	LocationCode  string
	Available     float64
}

func lockAvailableLocations(ctx context.Context, tx pgx.Tx, item lockedItem) ([]availableLocation, error) {
	rows, err := tx.Query(ctx, `
		SELECT b.warehouse_location_id, w.code, wl.code,
		       (b.on_hand - b.reserved)::float8
		FROM inventory_balances b
		JOIN warehouse_locations wl ON wl.id = b.warehouse_location_id
		JOIN warehouses w ON w.id = wl.warehouse_id
		WHERE b.sparepart_id = $1
		  AND wl.is_active = true
		  AND w.is_active = true
		  AND ($2::bigint IS NULL OR w.id = $2)
		ORDER BY w.code ASC, wl.code ASC
		FOR UPDATE OF b
	`, item.SparepartID, item.PreferredWarehouseID)
	if err != nil {
		return nil, fmt.Errorf("lock material stock: %w", err)
	}
	defer rows.Close()
	locations := make([]availableLocation, 0)
	for rows.Next() {
		var location availableLocation
		if err := rows.Scan(&location.LocationID, &location.WarehouseCode,
			&location.LocationCode, &location.Available); err != nil {
			return nil, fmt.Errorf("scan material stock: %w", err)
		}
		locations = append(locations, location)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate material stock: %w", err)
	}
	return locations, nil
}

func remainingRequired(item lockedItem) float64 {
	return item.QtyRequired - item.QtyReserved - (item.QtyIssued - item.QtyReturned)
}

func (r *Repository) Reserve(ctx context.Context, id int64, actor Actor, input ReserveInput) (Requirement, error) {
	if actor.Role != "admin" && actor.Role != "planner" && actor.Role != "storekeeper" {
		return Requirement{}, ErrValidation
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Requirement{}, fmt.Errorf("begin material reserve: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if input.IdempotencyKey != "" {
		found, err := r.inventory.CheckIdempotency(ctx, tx, input.IdempotencyKey)
		if err != nil {
			return Requirement{}, err
		}
		if found {
			return r.Get(ctx, id)
		}
	}
	current, err := r.get(ctx, tx, id, true)
	if err != nil {
		return Requirement{}, err
	}
	if current.Status == "cancelled" || current.Status == "completed" || current.Status == "draft" || current.Status == "planned" {
		return Requirement{}, ErrInvalidState
	}
	items, err := loadLockedItems(ctx, tx, id)
	if err != nil {
		return Requirement{}, err
	}
	if len(items) == 0 {
		return Requirement{}, &ValidationError{Fields: map[string]string{"items": "Material requirement belum memiliki item."}}
	}
	ledgerItems := make([]inventory.PostItem, 0)
	for _, item := range items {
		remaining := remainingRequired(item)
		if remaining <= 0 {
			continue
		}
		locations, err := lockAvailableLocations(ctx, tx, item)
		if err != nil {
			return Requirement{}, err
		}
		for _, location := range locations {
			if remaining <= 0 {
				break
			}
			if location.Available <= 0 {
				continue
			}
			qty := location.Available
			if qty > remaining {
				qty = remaining
			}
			var reservationID int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO inventory_reservations (
					material_requirement_item_id, sparepart_id, warehouse_location_id,
					qty, status, reserved_by, notes
				)
				VALUES ($1, $2, $3, $4, 'active', $5, NULLIF($6, ''))
				ON CONFLICT (material_requirement_item_id, warehouse_location_id)
					WHERE status = 'active'
				DO UPDATE SET qty = inventory_reservations.qty + EXCLUDED.qty,
				              reserved_by = EXCLUDED.reserved_by,
				              reserved_at = now()
				RETURNING id
			`, item.ID, item.SparepartID, location.LocationID, qty, actor.ID,
				strings.TrimSpace(input.Notes)).Scan(&reservationID); err != nil {
				return Requirement{}, fmt.Errorf("upsert inventory reservation: %w", err)
			}
			locationID := location.LocationID
			reservation := reservationID
			ledgerItems = append(ledgerItems, inventory.PostItem{
				SparepartID:    item.SparepartID,
				FromLocationID: &locationID,
				ReservationID:  &reservation,
				Qty:            qty,
				Notes:          "Material reservation",
			})
			if _, err := tx.Exec(ctx, `
				UPDATE material_requirement_items
				SET qty_reserved = qty_reserved + $2
				WHERE id = $1
			`, item.ID, qty); err != nil {
				return Requirement{}, fmt.Errorf("update material reserved qty: %w", err)
			}
			remaining -= qty
		}
	}
	if len(ledgerItems) > 0 {
		if _, err := r.inventory.PostTx(ctx, tx, inventory.Actor{ID: actor.ID, Role: actor.Role},
			inventory.PostInput{
				Type:           "reserve",
				ReferenceType:  "material_requirement",
				ReferenceID:    &id,
				ReferenceNo:    current.Code,
				Notes:          strings.TrimSpace(input.Notes),
				IdempotencyKey: input.IdempotencyKey,
				Items:          ledgerItems,
			}); err != nil {
			return Requirement{}, err
		}
	}
	if err := r.recomputeLocked(ctx, tx, id, actor.ID, false, false, "Reserve stock"); err != nil {
		return Requirement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Requirement{}, fmt.Errorf("commit material reserve: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *Repository) CreateReservation(ctx context.Context, actor Actor, input CreateReservationInput) (Reservation, error) {
	if actor.Role != "admin" && actor.Role != "planner" && actor.Role != "storekeeper" {
		return Reservation{}, ErrValidation
	}
	if input.MaterialItemID <= 0 || input.LocationID <= 0 || input.Qty <= 0 {
		return Reservation{}, &ValidationError{Fields: map[string]string{"items": "Item, lokasi, dan qty reservasi wajib valid."}}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Reservation{}, fmt.Errorf("begin reservation create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var requirementID int64
	if err := tx.QueryRow(ctx, `
		SELECT material_requirement_id FROM material_requirement_items
		WHERE id = $1
	`, input.MaterialItemID).Scan(&requirementID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Reservation{}, ErrNotFound
		}
		return Reservation{}, fmt.Errorf("load reservation material item: %w", err)
	}
	current, err := r.get(ctx, tx, requirementID, true)
	if err != nil {
		return Reservation{}, err
	}
	if current.Status == "cancelled" || current.Status == "completed" || current.Status == "draft" || current.Status == "planned" {
		return Reservation{}, ErrInvalidState
	}
	items, err := loadLockedItems(ctx, tx, requirementID)
	if err != nil {
		return Reservation{}, err
	}
	var selected lockedItem
	for _, item := range items {
		if item.ID == input.MaterialItemID {
			selected = item
			break
		}
	}
	if selected.ID == 0 {
		return Reservation{}, ErrNotFound
	}
	if input.Qty > remainingRequired(selected) {
		return Reservation{}, &ValidationError{Fields: map[string]string{"qty": "Qty reservasi melebihi sisa kebutuhan."}}
	}
	var available float64
	if err := tx.QueryRow(ctx, `
		SELECT (b.on_hand - b.reserved)::float8
		FROM inventory_balances b
		JOIN warehouse_locations wl ON wl.id = b.warehouse_location_id
		JOIN warehouses w ON w.id = wl.warehouse_id
		WHERE b.sparepart_id = $1 AND b.warehouse_location_id = $2
		  AND wl.is_active = true AND w.is_active = true
		FOR UPDATE OF b
	`, selected.SparepartID, input.LocationID).Scan(&available); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Reservation{}, &ValidationError{Fields: map[string]string{"warehouse_location_id": "Lokasi tidak memiliki saldo stok."}}
		}
		return Reservation{}, fmt.Errorf("lock reservation balance: %w", err)
	}
	if available < input.Qty {
		return Reservation{}, inventory.ErrInsufficientStock
	}
	var reservationID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO inventory_reservations (
			material_requirement_item_id, sparepart_id, warehouse_location_id,
			qty, status, reserved_by, notes
		)
		VALUES ($1, $2, $3, $4, 'active', $5, NULLIF($6, ''))
		ON CONFLICT (material_requirement_item_id, warehouse_location_id)
			WHERE status = 'active'
		DO UPDATE SET qty = inventory_reservations.qty + EXCLUDED.qty,
		              reserved_by = EXCLUDED.reserved_by,
		              reserved_at = now()
		RETURNING id
	`, selected.ID, selected.SparepartID, input.LocationID, input.Qty, actor.ID,
		strings.TrimSpace(input.Notes)).Scan(&reservationID); err != nil {
		return Reservation{}, fmt.Errorf("insert reservation: %w", err)
	}
	locationID := input.LocationID
	reservationRef := reservationID
	if _, err := r.inventory.PostTx(ctx, tx, inventory.Actor{ID: actor.ID, Role: actor.Role},
		inventory.PostInput{
			Type:          "reserve",
			ReferenceType: "material_requirement",
			ReferenceID:   &requirementID,
			ReferenceNo:   current.Code,
			Notes:         strings.TrimSpace(input.Notes),
			Items: []inventory.PostItem{{
				SparepartID:    selected.SparepartID,
				FromLocationID: &locationID,
				ReservationID:  &reservationRef,
				Qty:            input.Qty,
				Notes:          "Material reservation",
			}},
		}); err != nil {
		return Reservation{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE material_requirement_items
		SET qty_reserved = qty_reserved + $2
		WHERE id = $1
	`, selected.ID, input.Qty); err != nil {
		return Reservation{}, fmt.Errorf("update reserved qty: %w", err)
	}
	if err := r.recomputeLocked(ctx, tx, requirementID, actor.ID, false, false, "Reserve stock"); err != nil {
		return Reservation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, fmt.Errorf("commit reservation create: %w", err)
	}
	return r.getReservation(ctx, r.pool, reservationID, false)
}

func (r *Repository) ReleaseReservation(ctx context.Context, id int64, actor Actor, reason string) (Reservation, error) {
	if actor.Role != "admin" && actor.Role != "planner" && actor.Role != "storekeeper" {
		return Reservation{}, ErrValidation
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Reservation{}, fmt.Errorf("begin reservation release: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	initial, err := r.getReservation(ctx, tx, id, false)
	if err != nil {
		return Reservation{}, err
	}
	requirement, err := r.get(ctx, tx, initial.RequirementID, true)
	if err != nil {
		return Reservation{}, err
	}
	reservation, err := r.getReservation(ctx, tx, id, true)
	if err != nil {
		return Reservation{}, err
	}
	if reservation.Status != "active" {
		return Reservation{}, ErrInvalidState
	}
	locationID := reservation.LocationID
	reservationID := reservation.ID
	if _, err := r.inventory.PostTx(ctx, tx, inventory.Actor{ID: actor.ID, Role: actor.Role},
		inventory.PostInput{
			Type:          "release",
			ReferenceType: "material_requirement",
			ReferenceID:   &requirement.ID,
			ReferenceNo:   requirement.Code,
			Notes:         strings.TrimSpace(reason),
			Items: []inventory.PostItem{{
				SparepartID:    reservation.SparepartID,
				FromLocationID: &locationID,
				ReservationID:  &reservationID,
				Qty:            reservation.Qty,
				Notes:          strings.TrimSpace(reason),
			}},
		}); err != nil {
		return Reservation{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE inventory_reservations
		SET qty = 0, status = 'released', released_at = now(),
		    notes = COALESCE(NULLIF($2, ''), notes)
		WHERE id = $1
	`, id, strings.TrimSpace(reason)); err != nil {
		return Reservation{}, fmt.Errorf("mark reservation released: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE material_requirement_items
		SET qty_reserved = GREATEST(qty_reserved - $2, 0)
		WHERE id = $1
	`, reservation.RequirementItemID, reservation.Qty); err != nil {
		return Reservation{}, fmt.Errorf("decrease material reserved qty: %w", err)
	}
	if err := r.recomputeLocked(ctx, tx, requirement.ID, actor.ID, false, false, "Release reservation"); err != nil {
		return Reservation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, fmt.Errorf("commit reservation release: %w", err)
	}
	return r.getReservation(ctx, r.pool, id, false)
}

func (r *Repository) releaseAllReservationsTx(ctx context.Context, tx pgx.Tx, requirement Requirement, actor Actor, targetStatus, note string) error {
	if _, err := loadLockedItems(ctx, tx, requirement.ID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `
		SELECT ir.id, ir.material_requirement_item_id, ir.sparepart_id,
		       ir.warehouse_location_id, ir.qty::float8
		FROM inventory_reservations ir
		JOIN material_requirement_items mri ON mri.id = ir.material_requirement_item_id
		WHERE mri.material_requirement_id = $1 AND ir.status = 'active'
		ORDER BY ir.id
		FOR UPDATE OF ir
	`, requirement.ID)
	if err != nil {
		return fmt.Errorf("lock active reservations: %w", err)
	}
	type active struct {
		ID          int64
		ItemID      int64
		SparepartID int64
		LocationID  int64
		Qty         float64
	}
	activeRows := make([]active, 0)
	for rows.Next() {
		var item active
		if err := rows.Scan(&item.ID, &item.ItemID, &item.SparepartID,
			&item.LocationID, &item.Qty); err != nil {
			rows.Close()
			return fmt.Errorf("scan active reservation: %w", err)
		}
		activeRows = append(activeRows, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate active reservations: %w", err)
	}
	rows.Close()
	if len(activeRows) == 0 {
		return nil
	}
	ledgerItems := make([]inventory.PostItem, 0, len(activeRows))
	for _, item := range activeRows {
		locationID := item.LocationID
		reservationID := item.ID
		ledgerItems = append(ledgerItems, inventory.PostItem{
			SparepartID:    item.SparepartID,
			FromLocationID: &locationID,
			ReservationID:  &reservationID,
			Qty:            item.Qty,
			Notes:          note,
		})
	}
	if _, err := r.inventory.PostTx(ctx, tx, inventory.Actor{ID: actor.ID, Role: "admin"},
		inventory.PostInput{
			Type:          "release",
			ReferenceType: "material_requirement",
			ReferenceID:   &requirement.ID,
			ReferenceNo:   requirement.Code,
			Notes:         note,
			Items:         ledgerItems,
		}); err != nil {
		return err
	}
	for _, item := range activeRows {
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_reservations
			SET qty = 0, status = $2, released_at = now()
			WHERE id = $1
		`, item.ID, targetStatus); err != nil {
			return fmt.Errorf("close reservation: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE material_requirement_items
			SET qty_reserved = GREATEST(qty_reserved - $2, 0)
			WHERE id = $1
		`, item.ItemID, item.Qty); err != nil {
			return fmt.Errorf("decrease material reserved qty: %w", err)
		}
	}
	return nil
}
