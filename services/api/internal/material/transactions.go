package material

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"gsi-workorder/internal/inventory"
)

type issueUsageDraft struct {
	MaterialItemID int64
	SparepartID    int64
	LocationID     int64
	Qty            float64
	ComponentID    *int64
	Notes          string
}

type issueUsageRow struct {
	ID             int64
	WOID           int64
	MaterialItemID *int64
	SparepartID    int64
	LocationID     int64
	Qty            float64
	ComponentID    *int64
	HM             *float64
	KM             *float64
	MovementType   string
}

func remainingIssuable(item lockedItem) float64 {
	return item.QtyRequired - (item.QtyIssued - item.QtyReturned)
}

func (r *Repository) Issue(ctx context.Context, id int64, actor Actor, input IssueInput) (Requirement, error) {
	if actor.Role != "admin" && actor.Role != "storekeeper" {
		return Requirement{}, ErrValidation
	}
	if len(input.Items) == 0 {
		return Requirement{}, &ValidationError{Fields: map[string]string{"items": "Minimal satu item issue wajib diisi."}}
	}
	seen := map[int64]struct{}{}
	for index, item := range input.Items {
		if item.MaterialItemID <= 0 || item.Qty <= 0 {
			return Requirement{}, &ValidationError{Fields: map[string]string{
				fmt.Sprintf("items.%d", index): "Item dan qty issue wajib valid.",
			}}
		}
		if _, exists := seen[item.MaterialItemID]; exists {
			return Requirement{}, &ValidationError{Fields: map[string]string{"items": "Item material duplikat."}}
		}
		seen[item.MaterialItemID] = struct{}{}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Requirement{}, fmt.Errorf("begin material issue: %w", err)
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
	if current.Status == "draft" || current.Status == "planned" || current.Status == "cancelled" || current.Status == "completed" {
		return Requirement{}, ErrInvalidState
	}
	items, err := loadLockedItems(ctx, tx, id)
	if err != nil {
		return Requirement{}, err
	}
	itemMap := make(map[int64]lockedItem, len(items))
	for _, item := range items {
		itemMap[item.ID] = item
	}
	hm, km, err := r.workOrderReadings(ctx, tx, current.WOID)
	if err != nil {
		return Requirement{}, err
	}
	ledgerItems := make([]inventory.PostItem, 0)
	usageDrafts := make([]issueUsageDraft, 0)
	issuedByItem := make(map[int64]float64)
	reservedConsumed := make(map[int64]float64)
	for _, request := range input.Items {
		item, ok := itemMap[request.MaterialItemID]
		if !ok {
			return Requirement{}, &ValidationError{Fields: map[string]string{"items": "Item material tidak ditemukan pada requirement ini."}}
		}
		if request.Qty > remainingIssuable(item) {
			return Requirement{}, &ValidationError{Fields: map[string]string{"qty": "Qty issue melebihi sisa kebutuhan."}}
		}
		remaining := request.Qty
		reservations, err := lockActiveReservations(ctx, tx, item)
		if err != nil {
			return Requirement{}, err
		}
		for _, reservation := range reservations {
			if remaining <= 0 {
				break
			}
			qty := reservation.Qty
			if qty > remaining {
				qty = remaining
			}
			locationID := reservation.LocationID
			reservationID := reservation.ID
			ledgerItems = append(ledgerItems, inventory.PostItem{
				SparepartID:    item.SparepartID,
				FromLocationID: &locationID,
				ReservationID:  &reservationID,
				Qty:            qty,
				Notes:          request.Notes,
			})
			usageDrafts = append(usageDrafts, issueUsageDraft{
				MaterialItemID: item.ID,
				SparepartID:    item.SparepartID,
				LocationID:     reservation.LocationID,
				Qty:            qty,
				ComponentID:    request.ComponentID,
				Notes:          request.Notes,
			})
			reservedConsumed[item.ID] += qty
			remaining -= qty
			if qty == reservation.Qty {
				if _, err := tx.Exec(ctx, `
					UPDATE inventory_reservations
					SET qty = 0, status = 'consumed', consumed_at = now()
					WHERE id = $1
				`, reservation.ID); err != nil {
					return Requirement{}, fmt.Errorf("consume reservation: %w", err)
				}
			} else {
				if _, err := tx.Exec(ctx, `
					UPDATE inventory_reservations SET qty = qty - $2 WHERE id = $1
				`, reservation.ID, qty); err != nil {
					return Requirement{}, fmt.Errorf("decrease reservation: %w", err)
				}
			}
		}
		if remaining > 0 {
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
				locationID := location.LocationID
				ledgerItems = append(ledgerItems, inventory.PostItem{
					SparepartID:    item.SparepartID,
					FromLocationID: &locationID,
					Qty:            qty,
					Notes:          request.Notes,
				})
				usageDrafts = append(usageDrafts, issueUsageDraft{
					MaterialItemID: item.ID,
					SparepartID:    item.SparepartID,
					LocationID:     location.LocationID,
					Qty:            qty,
					ComponentID:    request.ComponentID,
					Notes:          request.Notes,
				})
				remaining -= qty
			}
		}
		if remaining > 0 {
			return Requirement{}, inventory.ErrInsufficientStock
		}
		issuedByItem[item.ID] += request.Qty
	}
	if len(ledgerItems) == 0 {
		return Requirement{}, &ValidationError{Fields: map[string]string{"items": "Tidak ada qty issue yang dapat diproses."}}
	}
	notes := strings.TrimSpace(input.Notes)
	if notes == "" {
		notes = "Material issue " + current.Code
	}
	posted, err := r.inventory.PostTx(ctx, tx, inventory.Actor{ID: actor.ID, Role: actor.Role},
		inventory.PostInput{
			Type:           "issue",
			ReferenceType:  "material_requirement",
			ReferenceID:    &id,
			ReferenceNo:    current.Code,
			Notes:          notes,
			IdempotencyKey: input.IdempotencyKey,
			Items:          ledgerItems,
		})
	if err != nil {
		return Requirement{}, err
	}
	for index, draft := range usageDrafts {
		if index >= len(posted.ItemIDs) {
			return Requirement{}, fmt.Errorf("material issue usage mapping mismatch")
		}
		transactionItemID := posted.ItemIDs[index]
		if _, err := tx.Exec(ctx, `
			INSERT INTO work_order_sparepart_usage (
				wo_id, material_requirement_item_id, sparepart_id, component_id,
				warehouse_location_id, qty, hm, km, movement_type,
				inventory_transaction_item_id, used_by, notes
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'issue', $9, $10, NULLIF($11, ''))
		`, current.WOID, draft.MaterialItemID, draft.SparepartID, draft.ComponentID,
			draft.LocationID, draft.Qty, hm, km, transactionItemID, actor.ID,
			strings.TrimSpace(draft.Notes)); err != nil {
			return Requirement{}, fmt.Errorf("insert material issue usage: %w", err)
		}
	}
	for itemID, qty := range issuedByItem {
		if _, err := tx.Exec(ctx, `
			UPDATE material_requirement_items
			SET qty_issued = qty_issued + $2,
			    qty_reserved = GREATEST(qty_reserved - $3, 0)
			WHERE id = $1
		`, itemID, qty, reservedConsumed[itemID]); err != nil {
			return Requirement{}, fmt.Errorf("update material issued qty: %w", err)
		}
	}
	if err := r.recomputeLocked(ctx, tx, id, actor.ID, false, false, "Issue material"); err != nil {
		return Requirement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Requirement{}, fmt.Errorf("commit material issue: %w", err)
	}
	return r.Get(ctx, id)
}

type activeReservation struct {
	ID         int64
	LocationID int64
	Qty        float64
}

func lockActiveReservations(ctx context.Context, tx pgx.Tx, item lockedItem) ([]activeReservation, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, warehouse_location_id, qty::float8
		FROM inventory_reservations
		WHERE material_requirement_item_id = $1
		  AND status = 'active'
		  AND qty > 0
		ORDER BY reserved_at ASC, id ASC
		FOR UPDATE
	`, item.ID)
	if err != nil {
		return nil, fmt.Errorf("lock active reservations: %w", err)
	}
	defer rows.Close()
	items := make([]activeReservation, 0)
	for rows.Next() {
		var reservation activeReservation
		if err := rows.Scan(&reservation.ID, &reservation.LocationID, &reservation.Qty); err != nil {
			return nil, fmt.Errorf("scan active reservation: %w", err)
		}
		items = append(items, reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active reservations: %w", err)
	}
	return items, nil
}

func (r *Repository) Return(ctx context.Context, id int64, actor Actor, input ReturnInput) (Requirement, error) {
	if actor.Role != "admin" && actor.Role != "storekeeper" {
		return Requirement{}, ErrValidation
	}
	if len(input.Items) == 0 {
		return Requirement{}, &ValidationError{Fields: map[string]string{"items": "Minimal satu item return wajib diisi."}}
	}
	for index, item := range input.Items {
		if item.Qty <= 0 {
			return Requirement{}, &ValidationError{Fields: map[string]string{
				fmt.Sprintf("items.%d.qty", index): "Qty return harus lebih dari 0.",
			}}
		}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Requirement{}, fmt.Errorf("begin material return: %w", err)
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
	items, err := loadLockedItems(ctx, tx, id)
	if err != nil {
		return Requirement{}, err
	}
	itemMap := make(map[int64]lockedItem, len(items))
	for _, item := range items {
		itemMap[item.ID] = item
	}
	ledgerItems := make([]inventory.PostItem, 0)
	returnedByItem := make(map[int64]float64)
	type returnUsageDraft struct {
		MaterialItemID int64
		SparepartID    int64
		LocationID     int64
		Qty            float64
		ComponentID    *int64
		HM             *float64
		KM             *float64
		ReturnOfID     *int64
		Notes          string
	}
	drafts := make([]returnUsageDraft, 0)
	seenUsage := map[int64]struct{}{}
	for _, request := range input.Items {
		if request.UsageID != nil {
			if _, exists := seenUsage[*request.UsageID]; exists {
				return Requirement{}, &ValidationError{Fields: map[string]string{"usage_id": "Usage return duplikat."}}
			}
			seenUsage[*request.UsageID] = struct{}{}
		}
		var materialItem lockedItem
		var locationID int64
		var componentID *int64
		var hm, km *float64
		var returnOfID *int64
		notes := strings.TrimSpace(request.Notes)
		if request.UsageID != nil {
			usage, err := lockIssueUsage(ctx, tx, *request.UsageID)
			if err != nil {
				return Requirement{}, err
			}
			if usage.WOID != current.WOID || usage.MaterialItemID == nil {
				return Requirement{}, &ValidationError{Fields: map[string]string{"usage_id": "Usage issue bukan milik work order ini."}}
			}
			if usage.MovementType != "issue" {
				return Requirement{}, &ValidationError{Fields: map[string]string{"usage_id": "Usage harus bertipe issue."}}
			}
			item, ok := itemMap[*usage.MaterialItemID]
			if !ok {
				return Requirement{}, &ValidationError{Fields: map[string]string{"usage_id": "Usage bukan milik material requirement ini."}}
			}
			var alreadyReturned float64
			if err := tx.QueryRow(ctx, `
				SELECT COALESCE(sum(qty), 0)::float8
				FROM work_order_sparepart_usage
				WHERE return_of_usage_id = $1
			`, usage.ID).Scan(&alreadyReturned); err != nil {
				return Requirement{}, fmt.Errorf("count returned usage: %w", err)
			}
			if request.Qty > usage.Qty-alreadyReturned {
				return Requirement{}, &ValidationError{Fields: map[string]string{"qty": "Qty return melebihi sisa issue."}}
			}
			materialItem = item
			locationID = usage.LocationID
			componentID = usage.ComponentID
			hm = usage.HM
			km = usage.KM
			returnOfID = &usage.ID
		} else {
			if actor.Role != "admin" {
				return Requirement{}, &ValidationError{Fields: map[string]string{"usage_id": "Return wajib menunjuk usage issue; hanya admin yang boleh exception."}}
			}
			if request.MaterialItemID == nil || request.LocationID == nil || notes == "" {
				return Requirement{}, &ValidationError{Fields: map[string]string{"items": "Exception return admin wajib mengisi item, lokasi, dan alasan."}}
			}
			item, ok := itemMap[*request.MaterialItemID]
			if !ok {
				return Requirement{}, &ValidationError{Fields: map[string]string{"material_requirement_item_id": "Item material tidak ditemukan."}}
			}
			if request.Qty > item.QtyIssued-item.QtyReturned-returnedByItem[item.ID] {
				return Requirement{}, &ValidationError{Fields: map[string]string{"qty": "Qty return melebihi sisa issue."}}
			}
			var locationExists int64
			if err := tx.QueryRow(ctx, `
				SELECT wl.id
				FROM warehouse_locations wl
				JOIN warehouses w ON w.id = wl.warehouse_id
				WHERE wl.id = $1 AND wl.is_active = true AND w.is_active = true
			`, *request.LocationID).Scan(&locationExists); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return Requirement{}, &ValidationError{Fields: map[string]string{"warehouse_location_id": "Lokasi return tidak valid."}}
				}
				return Requirement{}, fmt.Errorf("check return location: %w", err)
			}
			materialItem = item
			locationID = *request.LocationID
		}
		ledgerItems = append(ledgerItems, inventory.PostItem{
			SparepartID:  materialItem.SparepartID,
			ToLocationID: &locationID,
			Qty:          request.Qty,
			Notes:        notes,
		})
		drafts = append(drafts, returnUsageDraft{
			MaterialItemID: materialItem.ID,
			SparepartID:    materialItem.SparepartID,
			LocationID:     locationID,
			Qty:            request.Qty,
			ComponentID:    componentID,
			HM:             hm,
			KM:             km,
			ReturnOfID:     returnOfID,
			Notes:          notes,
		})
		returnedByItem[materialItem.ID] += request.Qty
	}
	notes := strings.TrimSpace(input.Notes)
	if notes == "" {
		notes = "Material return " + current.Code
	}
	posted, err := r.inventory.PostTx(ctx, tx, inventory.Actor{ID: actor.ID, Role: actor.Role},
		inventory.PostInput{
			Type:           "return",
			ReferenceType:  "material_requirement",
			ReferenceID:    &id,
			ReferenceNo:    current.Code,
			Notes:          notes,
			IdempotencyKey: input.IdempotencyKey,
			Items:          ledgerItems,
		})
	if err != nil {
		return Requirement{}, err
	}
	for index, draft := range drafts {
		if index >= len(posted.ItemIDs) {
			return Requirement{}, fmt.Errorf("material return usage mapping mismatch")
		}
		transactionItemID := posted.ItemIDs[index]
		if _, err := tx.Exec(ctx, `
			INSERT INTO work_order_sparepart_usage (
				wo_id, material_requirement_item_id, sparepart_id, component_id,
				warehouse_location_id, qty, hm, km, movement_type,
				inventory_transaction_item_id, return_of_usage_id, used_by, notes
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'return', $9, $10, $11, NULLIF($12, ''))
		`, current.WOID, draft.MaterialItemID, draft.SparepartID, draft.ComponentID,
			draft.LocationID, draft.Qty, draft.HM, draft.KM, transactionItemID,
			draft.ReturnOfID, actor.ID, strings.TrimSpace(draft.Notes)); err != nil {
			return Requirement{}, fmt.Errorf("insert material return usage: %w", err)
		}
	}
	for itemID, qty := range returnedByItem {
		if _, err := tx.Exec(ctx, `
			UPDATE material_requirement_items
			SET qty_returned = qty_returned + $2
			WHERE id = $1
		`, itemID, qty); err != nil {
			return Requirement{}, fmt.Errorf("update material returned qty: %w", err)
		}
	}
	if err := r.recomputeLocked(ctx, tx, id, actor.ID, false, false, "Return material"); err != nil {
		return Requirement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Requirement{}, fmt.Errorf("commit material return: %w", err)
	}
	return r.Get(ctx, id)
}

func lockIssueUsage(ctx context.Context, tx pgx.Tx, id int64) (issueUsageRow, error) {
	var item issueUsageRow
	err := tx.QueryRow(ctx, `
		SELECT id, wo_id, material_requirement_item_id, sparepart_id,
		       warehouse_location_id, qty::float8, component_id,
		       hm::float8, km::float8, movement_type
		FROM work_order_sparepart_usage
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&item.ID, &item.WOID, &item.MaterialItemID, &item.SparepartID,
		&item.LocationID, &item.Qty, &item.ComponentID, &item.HM, &item.KM,
		&item.MovementType)
	if errors.Is(err, pgx.ErrNoRows) {
		return issueUsageRow{}, ErrNotFound
	}
	if err != nil {
		return issueUsageRow{}, fmt.Errorf("lock material usage: %w", err)
	}
	return item, nil
}

func (r *Repository) workOrderReadings(ctx context.Context, tx pgx.Tx, woID int64) (*float64, *float64, error) {
	var hm, km float64
	err := tx.QueryRow(ctx, `
		SELECT u.current_hm::float8, u.current_km::float8
		FROM work_orders wo
		JOIN units u ON u.id = wo.unit_id
		WHERE wo.id = $1
	`, woID).Scan(&hm, &km)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("load work-order readings: %w", err)
	}
	return &hm, &km, nil
}

func (r *Repository) Cancel(ctx context.Context, id int64, actor Actor, reason string) (Requirement, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return Requirement{}, ErrValidation
	}
	reason = strings.TrimSpace(reason)
	if len(reason) < 5 || len(reason) > 1000 {
		return Requirement{}, &ValidationError{Fields: map[string]string{"reason": "Alasan pembatalan wajib 5-1000 karakter."}}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Requirement{}, fmt.Errorf("begin material cancel: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := r.get(ctx, tx, id, true)
	if err != nil {
		return Requirement{}, err
	}
	if err := r.cancelTx(ctx, tx, current, actor, reason); err != nil {
		return Requirement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Requirement{}, fmt.Errorf("commit material cancel: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *Repository) cancelTx(ctx context.Context, tx pgx.Tx, current Requirement, actor Actor, reason string) error {
	if current.Status == "cancelled" || current.Status == "completed" {
		return ErrInvalidState
	}
	items, err := loadLockedItems(ctx, tx, current.ID)
	if err != nil {
		return err
	}
	netIssued := 0.0
	for _, item := range items {
		netIssued += item.QtyIssued - item.QtyReturned
	}
	if netIssued > 0 && actor.Role != "admin" {
		return &ValidationError{Fields: map[string]string{"reason": "Material dengan issue aktif hanya dapat dibatalkan admin."}}
	}
	if err := r.releaseAllReservationsTx(ctx, tx, current, actor, "cancelled", reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE material_requirements
		SET status = 'cancelled', status_changed_at = now(),
		    shortage_since = NULL, updated_at = now()
		WHERE id = $1
	`, current.ID); err != nil {
		return fmt.Errorf("cancel material requirement: %w", err)
	}
	return insertUpdate(ctx, tx, current.ID, actor.ID, current.Status, "cancelled", reason)
}

func (r *Repository) CompleteForWO(ctx context.Context, tx pgx.Tx, woID int64, actor Actor, note string) error {
	current, err := r.requirementForWOTx(ctx, tx, woID, true)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	if current.Status == "completed" || current.Status == "cancelled" {
		return nil
	}
	if current.Status == "shortage" && actor.Role != "admin" {
		return &ValidationError{Fields: map[string]string{"material": "Material masih shortage; hanya admin yang boleh override."}}
	}
	if err := r.releaseAllReservationsTx(ctx, tx, current, actor, "released", note); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE material_requirements
		SET status = 'completed', status_changed_at = now(),
		    shortage_since = NULL, updated_at = now()
		WHERE id = $1
	`, current.ID); err != nil {
		return fmt.Errorf("complete material requirement: %w", err)
	}
	return insertUpdate(ctx, tx, current.ID, actor.ID, current.Status, "completed", note)
}

func (r *Repository) CancelForWO(ctx context.Context, tx pgx.Tx, woID int64, actor Actor, reason string) error {
	current, err := r.requirementForWOTx(ctx, tx, woID, true)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	if current.Status == "cancelled" {
		return nil
	}
	return r.cancelTx(ctx, tx, current, actor, reason)
}

func (r *Repository) RequirementStatusForWO(ctx context.Context, tx pgx.Tx, woID int64) (string, bool, error) {
	current, err := r.requirementForWOTx(ctx, tx, woID, true)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	return current.Status, true, nil
}

func (r *Repository) requirementForWOTx(ctx context.Context, tx pgx.Tx, woID int64, forUpdate bool) (Requirement, error) {
	query := requirementSelect + ` WHERE mr.wo_id = $1 AND mr.status <> 'cancelled' ORDER BY mr.id DESC LIMIT 1`
	if forUpdate {
		query += ` FOR UPDATE OF mr`
	}
	item, err := scanRequirement(tx.QueryRow(ctx, query, woID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Requirement{}, ErrNotFound
	}
	if err != nil {
		return Requirement{}, fmt.Errorf("load work-order material requirement: %w", err)
	}
	return item, nil
}

func (r *Repository) CreateFromPlan(ctx context.Context, tx pgx.Tx, woID, planID, actorID int64) error {
	return SnapshotPlan(ctx, tx, woID, planID, actorID)
}

// SnapshotPlan copies pm_plan_materials into a draft material requirement.
// It is package-level so the PM repository can call it with its existing tx.
func SnapshotPlan(ctx context.Context, tx pgx.Tx, woID, planID, actorID int64) error {
	var existing int64
	err := tx.QueryRow(ctx, `
		SELECT id FROM material_requirements
		WHERE wo_id = $1 AND status <> 'cancelled'
		LIMIT 1
	`, woID).Scan(&existing)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("check PM material requirement: %w", err)
	}
	var itemCount int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM pm_plan_materials m
		JOIN spareparts s ON s.id = m.sparepart_id
		WHERE m.pm_plan_id = $1 AND s.is_active = true
	`, planID).Scan(&itemCount); err != nil {
		return fmt.Errorf("count PM plan materials: %w", err)
	}
	if itemCount == 0 {
		return nil
	}
	code, err := nextRequirementCode(ctx, tx)
	if err != nil {
		return err
	}
	var requirementID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO material_requirements (
			code, wo_id, source, status, requested_by, requested_at, notes
		)
		VALUES ($1, $2, 'pm', 'draft', $3, now(), 'Snapshot PM material plan')
		RETURNING id
	`, code, woID, nullActor(actorID)).Scan(&requirementID); err != nil {
		return fmt.Errorf("insert PM material requirement: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO material_requirement_items (
			material_requirement_id, sparepart_id, qty_required, is_optional, notes
		)
		SELECT $1, m.sparepart_id, m.qty, m.is_optional, m.notes
		FROM pm_plan_materials m
		JOIN spareparts s ON s.id = m.sparepart_id
		WHERE m.pm_plan_id = $2 AND s.is_active = true
	`, requirementID, planID); err != nil {
		return fmt.Errorf("snapshot PM material items: %w", err)
	}
	return insertUpdate(ctx, tx, requirementID, actorID, "", "draft", "Snapshot PM material plan")
}

func (r *Repository) RecomputeForSpareparts(ctx context.Context, sparepartIDs []int64) (RecomputeResult, error) {
	if len(sparepartIDs) == 0 {
		return RecomputeResult{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT mri.material_requirement_id
		FROM material_requirement_items mri
		JOIN material_requirements mr ON mr.id = mri.material_requirement_id
		WHERE mri.sparepart_id = ANY($1)
		  AND mr.status NOT IN ('draft', 'planned', 'completed', 'cancelled')
	`, sparepartIDs)
	if err != nil {
		return RecomputeResult{}, fmt.Errorf("find affected material requirements: %w", err)
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return RecomputeResult{}, fmt.Errorf("scan affected material requirement: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return RecomputeResult{}, fmt.Errorf("iterate affected material requirements: %w", err)
	}
	rows.Close()
	return r.recomputeIDs(ctx, ids)
}

func (r *Repository) RecomputeAll(ctx context.Context) (RecomputeResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM material_requirements
		WHERE status NOT IN ('draft', 'planned', 'completed', 'cancelled')
		ORDER BY id
	`)
	if err != nil {
		return RecomputeResult{}, fmt.Errorf("find material requirements: %w", err)
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return RecomputeResult{}, fmt.Errorf("scan material requirement: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return RecomputeResult{}, fmt.Errorf("iterate material requirements: %w", err)
	}
	rows.Close()
	return r.recomputeIDs(ctx, ids)
}

func (r *Repository) recomputeIDs(ctx context.Context, ids []int64) (RecomputeResult, error) {
	if len(ids) == 0 {
		return RecomputeResult{}, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RecomputeResult{}, fmt.Errorf("begin material recompute: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	updated := 0
	for _, id := range ids {
		var before string
		if err := tx.QueryRow(ctx, `SELECT status FROM material_requirements WHERE id = $1`, id).Scan(&before); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return RecomputeResult{}, fmt.Errorf("load material status for recompute: %w", err)
		}
		if err := r.recomputeLocked(ctx, tx, id, 0, false, false, "Stock change"); err != nil {
			return RecomputeResult{}, err
		}
		var after string
		if err := tx.QueryRow(ctx, `SELECT status FROM material_requirements WHERE id = $1`, id).Scan(&after); err != nil {
			return RecomputeResult{}, fmt.Errorf("read recomputed material status: %w", err)
		}
		if before != after {
			updated++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return RecomputeResult{}, fmt.Errorf("commit material recompute: %w", err)
	}
	return RecomputeResult{Updated: updated}, nil
}
