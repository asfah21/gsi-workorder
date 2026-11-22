package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) ListWarehouses(ctx context.Context, query string, activeOnly bool, page, limit int) ([]Warehouse, int, error) {
	page, limit = normalizePage(page, limit)
	args := make([]any, 0, 3)
	where := []string{"1 = 1"}
	if query != "" {
		args = append(args, "%"+trim(query)+"%")
		where = append(where, fmt.Sprintf("(code ILIKE $%d OR name ILIKE $%d OR COALESCE(site_location, '') ILIKE $%d)",
			len(args), len(args), len(args)))
	}
	if activeOnly {
		where = append(where, "is_active = true")
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM warehouses WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count warehouses: %w", err)
	}
	args = append(args, limit, (page-1)*limit)
	rows, err := r.pool.Query(ctx, `
		SELECT w.id, w.code, w.name, COALESCE(w.site_location, ''), w.is_active,
		       (SELECT count(*) FROM warehouse_locations wl WHERE wl.warehouse_id = w.id),
		       w.created_at, w.updated_at
		FROM warehouses w
		WHERE `+whereSQL+`
		ORDER BY w.code ASC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list warehouses: %w", err)
	}
	defer rows.Close()
	items := make([]Warehouse, 0)
	for rows.Next() {
		var item Warehouse
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.SiteLocation,
			&item.IsActive, &item.LocationCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan warehouse: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate warehouses: %w", err)
	}
	return items, total, nil
}

func (r *Repository) CreateWarehouse(ctx context.Context, actor Actor, input WarehouseInput) (Warehouse, error) {
	if !canManageMaster(actor.Role) {
		return Warehouse{}, ErrValidation
	}
	code, name, site, err := normalizeWarehouse(input.Code, input.Name, input.SiteLocation)
	if err != nil {
		return Warehouse{}, err
	}
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}
	var id int64
	err = r.pool.QueryRow(ctx, `
		INSERT INTO warehouses (code, name, site_location, is_active)
		VALUES ($1, $2, NULLIF($3, ''), $4)
		RETURNING id
	`, code, name, site, active).Scan(&id)
	if isUniqueViolation(err) {
		return Warehouse{}, ErrConflict
	}
	if err != nil {
		return Warehouse{}, fmt.Errorf("create warehouse: %w", err)
	}
	return r.getWarehouse(ctx, id)
}

func (r *Repository) UpdateWarehouse(ctx context.Context, id int64, actor Actor, patch WarehousePatch) (Warehouse, error) {
	if !canManageMaster(actor.Role) {
		return Warehouse{}, ErrValidation
	}
	current, err := r.getWarehouse(ctx, id)
	if err != nil {
		return Warehouse{}, err
	}
	if patch.Code != nil {
		current.Code = *patch.Code
	}
	if patch.Name != nil {
		current.Name = *patch.Name
	}
	if patch.SiteLocation != nil {
		current.SiteLocation = *patch.SiteLocation
	}
	if patch.IsActive != nil {
		current.IsActive = *patch.IsActive
	}
	code, name, site, err := normalizeWarehouse(current.Code, current.Name, current.SiteLocation)
	if err != nil {
		return Warehouse{}, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE warehouses
		SET code = $2, name = $3, site_location = NULLIF($4, ''), is_active = $5, updated_at = now()
		WHERE id = $1
	`, id, code, name, site, current.IsActive)
	if isUniqueViolation(err) {
		return Warehouse{}, ErrConflict
	}
	if err != nil {
		return Warehouse{}, fmt.Errorf("update warehouse: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Warehouse{}, ErrNotFound
	}
	return r.getWarehouse(ctx, id)
}

func (r *Repository) getWarehouse(ctx context.Context, id int64) (Warehouse, error) {
	var item Warehouse
	err := r.pool.QueryRow(ctx, `
		SELECT w.id, w.code, w.name, COALESCE(w.site_location, ''), w.is_active,
		       (SELECT count(*) FROM warehouse_locations wl WHERE wl.warehouse_id = w.id),
		       w.created_at, w.updated_at
		FROM warehouses w
		WHERE w.id = $1
	`, id).Scan(&item.ID, &item.Code, &item.Name, &item.SiteLocation, &item.IsActive,
		&item.LocationCount, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Warehouse{}, ErrNotFound
	}
	if err != nil {
		return Warehouse{}, fmt.Errorf("get warehouse: %w", err)
	}
	return item, nil
}

func (r *Repository) ListLocations(ctx context.Context, warehouseID *int64, query string, activeOnly bool) ([]Location, error) {
	args := make([]any, 0, 3)
	where := []string{"1 = 1"}
	if warehouseID != nil {
		if err := r.requireWarehouse(ctx, *warehouseID); err != nil {
			return nil, err
		}
		args = append(args, *warehouseID)
		where = append(where, fmt.Sprintf("wl.warehouse_id = $%d", len(args)))
	}
	if query != "" {
		args = append(args, "%"+trim(query)+"%")
		where = append(where, fmt.Sprintf(`(wl.code ILIKE $%d OR wl.name ILIKE $%d
			OR w.code ILIKE $%d OR w.name ILIKE $%d)`,
			len(args), len(args), len(args), len(args)))
	}
	if activeOnly {
		where = append(where, "wl.is_active = true AND w.is_active = true")
	}
	rows, err := r.pool.Query(ctx, locationSelect+` WHERE `+strings.Join(where, " AND ")+`
		ORDER BY w.code ASC, wl.code ASC
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list warehouse locations: %w", err)
	}
	defer rows.Close()
	items := make([]Location, 0)
	for rows.Next() {
		item, err := scanLocation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan warehouse location: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate warehouse locations: %w", err)
	}
	return items, nil
}

func (r *Repository) CreateLocation(ctx context.Context, actor Actor, input LocationInput) (Location, error) {
	if !canManageMaster(actor.Role) {
		return Location{}, ErrValidation
	}
	if err := r.requireWarehouse(ctx, input.WarehouseID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return Location{}, &ValidationError{Fields: map[string]string{"warehouse_id": "Gudang tidak ditemukan."}}
		}
		return Location{}, err
	}
	code, name, locationType, err := normalizeLocation(input.Code, input.Name, input.LocationType)
	if err != nil {
		return Location{}, err
	}
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}
	var id int64
	err = r.pool.QueryRow(ctx, `
		INSERT INTO warehouse_locations (warehouse_id, code, name, location_type, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, input.WarehouseID, code, name, locationType, active).Scan(&id)
	if isUniqueViolation(err) {
		return Location{}, ErrConflict
	}
	if err != nil {
		return Location{}, fmt.Errorf("create warehouse location: %w", err)
	}
	return r.getLocation(ctx, id)
}

func (r *Repository) UpdateLocation(ctx context.Context, id int64, actor Actor, patch LocationPatch) (Location, error) {
	if !canManageMaster(actor.Role) {
		return Location{}, ErrValidation
	}
	current, err := r.getLocation(ctx, id)
	if err != nil {
		return Location{}, err
	}
	if patch.Code != nil {
		current.Code = *patch.Code
	}
	if patch.Name != nil {
		current.Name = *patch.Name
	}
	if patch.LocationType != nil {
		current.LocationType = *patch.LocationType
	}
	if patch.IsActive != nil {
		current.IsActive = *patch.IsActive
	}
	code, name, locationType, err := normalizeLocation(current.Code, current.Name, current.LocationType)
	if err != nil {
		return Location{}, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE warehouse_locations
		SET code = $2, name = $3, location_type = $4, is_active = $5, updated_at = now()
		WHERE id = $1
	`, id, code, name, locationType, current.IsActive)
	if isUniqueViolation(err) {
		return Location{}, ErrConflict
	}
	if err != nil {
		return Location{}, fmt.Errorf("update warehouse location: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Location{}, ErrNotFound
	}
	return r.getLocation(ctx, id)
}

func (r *Repository) getLocation(ctx context.Context, id int64) (Location, error) {
	item, err := scanLocation(r.pool.QueryRow(ctx, locationSelect+` WHERE wl.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Location{}, ErrNotFound
	}
	if err != nil {
		return Location{}, fmt.Errorf("get warehouse location: %w", err)
	}
	return item, nil
}

const locationSelect = `
	SELECT wl.id, wl.warehouse_id, w.code, w.name, wl.code, wl.name,
	       wl.location_type, wl.is_active, wl.created_at, wl.updated_at
	FROM warehouse_locations wl
	JOIN warehouses w ON w.id = wl.warehouse_id
`

func scanLocation(row interface{ Scan(...any) error }) (Location, error) {
	var item Location
	if err := row.Scan(&item.ID, &item.WarehouseID, &item.WarehouseCode, &item.WarehouseName,
		&item.Code, &item.Name, &item.LocationType, &item.IsActive,
		&item.CreatedAt, &item.UpdatedAt); err != nil {
		return Location{}, err
	}
	return item, nil
}

func (r *Repository) requireWarehouse(ctx context.Context, id int64) error {
	var exists int64
	if err := r.pool.QueryRow(ctx, `SELECT id FROM warehouses WHERE id = $1`, id).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("check warehouse: %w", err)
	}
	return nil
}

func normalizeWarehouse(code, name, site string) (string, string, string, error) {
	fields := map[string]string{}
	code = strings.ToUpper(trim(code))
	name = trim(name)
	site = trim(site)
	if code == "" || len(code) > 40 {
		fields["code"] = "Kode gudang wajib 1-40 karakter."
	}
	if name == "" || len(name) > 150 {
		fields["name"] = "Nama gudang wajib 1-150 karakter."
	}
	if len(site) > 150 {
		fields["site_location"] = "Site location maksimal 150 karakter."
	}
	if len(fields) > 0 {
		return "", "", "", &ValidationError{Fields: fields}
	}
	return code, name, site, nil
}

func normalizeLocation(code, name, locationType string) (string, string, string, error) {
	fields := map[string]string{}
	code = strings.ToUpper(trim(code))
	name = trim(name)
	locationType = strings.ToLower(trim(locationType))
	if locationType == "" {
		locationType = "bin"
	}
	if code == "" || len(code) > 40 {
		fields["code"] = "Kode lokasi wajib 1-40 karakter."
	}
	if name == "" || len(name) > 150 {
		fields["name"] = "Nama lokasi wajib 1-150 karakter."
	}
	if locationType != "rack" && locationType != "bin" && locationType != "yard" && locationType != "staging" {
		fields["location_type"] = "Tipe lokasi tidak valid."
	}
	if len(fields) > 0 {
		return "", "", "", &ValidationError{Fields: fields}
	}
	return code, name, locationType, nil
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

func (r *Repository) ListPolicies(ctx context.Context, filter PolicyFilter) ([]Policy, int, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	args := make([]any, 0, 5)
	where := []string{"1 = 1"}
	if filter.Query != "" {
		args = append(args, "%"+trim(filter.Query)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf(`(s.code ILIKE $%d OR s.name ILIKE $%d
			OR wl.code ILIKE $%d OR wl.name ILIKE $%d OR w.code ILIKE $%d)`,
			n, n, n, n, n))
	}
	if filter.SparepartID != nil {
		args = append(args, *filter.SparepartID)
		where = append(where, fmt.Sprintf("p.sparepart_id = $%d", len(args)))
	}
	if filter.LocationID != nil {
		args = append(args, *filter.LocationID)
		where = append(where, fmt.Sprintf("p.warehouse_location_id = $%d", len(args)))
	}
	if filter.ActiveOnly {
		where = append(where, "p.is_active = true")
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM inventory_stock_policies p
		JOIN spareparts s ON s.id = p.sparepart_id
		JOIN warehouse_locations wl ON wl.id = p.warehouse_location_id
		JOIN warehouses w ON w.id = wl.warehouse_id
		WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stock policies: %w", err)
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, policySelect+`
		WHERE `+whereSQL+`
		ORDER BY s.code ASC, w.code ASC, wl.code ASC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stock policies: %w", err)
	}
	defer rows.Close()
	items := make([]Policy, 0)
	for rows.Next() {
		item, err := scanPolicy(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan stock policy: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate stock policies: %w", err)
	}
	return items, total, nil
}

func (r *Repository) UpsertPolicy(ctx context.Context, actor Actor, input PolicyInput) (Policy, error) {
	if !canManageMaster(actor.Role) {
		return Policy{}, ErrValidation
	}
	if err := r.validatePolicy(ctx, input.SparepartID, input.LocationID, input.MinQty, input.CriticalQty, input.ReorderQty); err != nil {
		return Policy{}, err
	}
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO inventory_stock_policies (
			sparepart_id, warehouse_location_id, min_qty, critical_qty, reorder_qty, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (sparepart_id, warehouse_location_id) DO UPDATE
		SET min_qty = EXCLUDED.min_qty,
		    critical_qty = EXCLUDED.critical_qty,
		    reorder_qty = EXCLUDED.reorder_qty,
		    is_active = EXCLUDED.is_active,
		    updated_at = now()
		RETURNING id
	`, input.SparepartID, input.LocationID, input.MinQty, input.CriticalQty,
		input.ReorderQty, active).Scan(&id)
	if err != nil {
		return Policy{}, fmt.Errorf("upsert stock policy: %w", err)
	}
	return r.getPolicy(ctx, id)
}

func (r *Repository) UpdatePolicy(ctx context.Context, id int64, actor Actor, patch PolicyPatch) (Policy, error) {
	if !canManageMaster(actor.Role) {
		return Policy{}, ErrValidation
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Policy{}, fmt.Errorf("begin stock policy update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current Policy
	if err := tx.QueryRow(ctx, policySelect+` WHERE p.id = $1 FOR UPDATE`, id).Scan(
		&current.ID, &current.SparepartID, &current.SparepartCode, &current.SparepartName,
		&current.UOM, &current.LocationID, &current.WarehouseID, &current.WarehouseCode,
		&current.LocationCode, &current.LocationName, &current.MinQty, &current.CriticalQty,
		&current.ReorderQty, &current.IsActive, &current.CreatedAt, &current.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Policy{}, ErrNotFound
		}
		return Policy{}, fmt.Errorf("lock stock policy: %w", err)
	}
	if patch.MinQty != nil {
		current.MinQty = *patch.MinQty
	}
	if patch.CriticalQty != nil {
		current.CriticalQty = *patch.CriticalQty
	}
	if patch.ClearReorder {
		current.ReorderQty = nil
	} else if patch.ReorderQty != nil {
		current.ReorderQty = patch.ReorderQty
	}
	if patch.IsActive != nil {
		current.IsActive = *patch.IsActive
	}
	if err := validatePolicyValues(current.MinQty, current.CriticalQty, current.ReorderQty); err != nil {
		return Policy{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE inventory_stock_policies
		SET min_qty = $2, critical_qty = $3, reorder_qty = $4, is_active = $5, updated_at = now()
		WHERE id = $1
	`, id, current.MinQty, current.CriticalQty, current.ReorderQty, current.IsActive); err != nil {
		return Policy{}, fmt.Errorf("update stock policy: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Policy{}, fmt.Errorf("commit stock policy update: %w", err)
	}
	return r.getPolicy(ctx, id)
}

func (r *Repository) getPolicy(ctx context.Context, id int64) (Policy, error) {
	item, err := scanPolicy(r.pool.QueryRow(ctx, policySelect+` WHERE p.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Policy{}, ErrNotFound
	}
	if err != nil {
		return Policy{}, fmt.Errorf("get stock policy: %w", err)
	}
	return item, nil
}

const policySelect = `
	SELECT p.id, p.sparepart_id, s.code, s.name, s.uom,
	       p.warehouse_location_id, w.id, w.code, wl.code, wl.name,
	       p.min_qty::float8, p.critical_qty::float8, p.reorder_qty::float8,
	       p.is_active, p.created_at, p.updated_at
	FROM inventory_stock_policies p
	JOIN spareparts s ON s.id = p.sparepart_id
	JOIN warehouse_locations wl ON wl.id = p.warehouse_location_id
	JOIN warehouses w ON w.id = wl.warehouse_id
`

func scanPolicy(row interface{ Scan(...any) error }) (Policy, error) {
	var item Policy
	if err := row.Scan(&item.ID, &item.SparepartID, &item.SparepartCode, &item.SparepartName,
		&item.UOM, &item.LocationID, &item.WarehouseID, &item.WarehouseCode,
		&item.LocationCode, &item.LocationName, &item.MinQty, &item.CriticalQty,
		&item.ReorderQty, &item.IsActive, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Policy{}, err
	}
	return item, nil
}

func (r *Repository) validatePolicy(ctx context.Context, sparepartID, locationID int64, minQty, criticalQty float64, reorderQty *float64) error {
	if err := validatePolicyValues(minQty, criticalQty, reorderQty); err != nil {
		return err
	}
	var exists int64
	if err := r.pool.QueryRow(ctx, `SELECT id FROM spareparts WHERE id = $1 AND is_active = true`, sparepartID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &ValidationError{Fields: map[string]string{"sparepart_id": "Sparepart tidak ditemukan atau tidak aktif."}}
		}
		return fmt.Errorf("check sparepart for policy: %w", err)
	}
	if err := r.pool.QueryRow(ctx, `
		SELECT id FROM warehouse_locations WHERE id = $1 AND is_active = true
	`, locationID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &ValidationError{Fields: map[string]string{"warehouse_location_id": "Lokasi tidak ditemukan atau tidak aktif."}}
		}
		return fmt.Errorf("check location for policy: %w", err)
	}
	return nil
}

func validatePolicyValues(minQty, criticalQty float64, reorderQty *float64) error {
	fields := map[string]string{}
	if minQty < 0 {
		fields["min_qty"] = "Minimum stock tidak boleh negatif."
	}
	if criticalQty < 0 {
		fields["critical_qty"] = "Critical stock tidak boleh negatif."
	}
	if criticalQty > minQty {
		fields["critical_qty"] = "Critical stock tidak boleh melebihi minimum stock."
	}
	if reorderQty != nil && *reorderQty < 0 {
		fields["reorder_qty"] = "Reorder qty tidak boleh negatif."
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func (r *Repository) ListBalances(ctx context.Context, filter BalanceFilter) ([]Balance, int, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	args := make([]any, 0, 6)
	where := []string{"1 = 1"}
	if filter.Query != "" {
		args = append(args, "%"+trim(filter.Query)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf(`(sparepart_code ILIKE $%d OR sparepart_name ILIKE $%d
			OR location_code ILIKE $%d OR location_name ILIKE $%d OR warehouse_code ILIKE $%d)`,
			n, n, n, n, n))
	}
	if filter.WarehouseID != nil {
		args = append(args, *filter.WarehouseID)
		where = append(where, fmt.Sprintf("warehouse_id = $%d", len(args)))
	}
	if filter.LocationID != nil {
		args = append(args, *filter.LocationID)
		where = append(where, fmt.Sprintf("warehouse_location_id = $%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("stock_status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM v_stock_balance WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stock balances: %w", err)
	}
	orderColumn := balanceSortColumn(filter.Sort)
	orderDirection := "ASC"
	if strings.EqualFold(filter.Order, "desc") {
		orderDirection = "DESC"
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, `
		SELECT sparepart_id, sparepart_code, sparepart_name, category, uom, sparepart_active,
		       warehouse_location_id, warehouse_id, warehouse_code, warehouse_name,
		       location_code, location_name, location_type,
		       on_hand::float8, reserved::float8, available::float8,
		       policy_id, min_qty::float8, critical_qty::float8, reorder_qty::float8,
		       stock_status, balance_updated_at
		FROM v_stock_balance
		WHERE `+whereSQL+`
		ORDER BY `+orderColumn+` `+orderDirection+`, sparepart_code ASC, warehouse_code ASC, location_code ASC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stock balances: %w", err)
	}
	defer rows.Close()
	items := make([]Balance, 0)
	for rows.Next() {
		var item Balance
		if err := rows.Scan(&item.SparepartID, &item.SparepartCode, &item.SparepartName,
			&item.Category, &item.UOM, &item.SparepartActive, &item.LocationID,
			&item.WarehouseID, &item.WarehouseCode, &item.WarehouseName,
			&item.LocationCode, &item.LocationName, &item.LocationType,
			&item.OnHand, &item.Reserved, &item.Available, &item.PolicyID,
			&item.MinQty, &item.CriticalQty, &item.ReorderQty, &item.StockStatus,
			&item.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan stock balance: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate stock balances: %w", err)
	}
	return items, total, nil
}

func balanceSortColumn(sort string) string {
	switch strings.ToLower(trim(sort)) {
	case "sparepart_name":
		return "sparepart_name"
	case "available":
		return "available"
	case "on_hand":
		return "on_hand"
	case "reserved":
		return "reserved"
	case "stock_status":
		return "CASE stock_status WHEN 'critical' THEN 1 WHEN 'low' THEN 2 WHEN 'ok' THEN 3 ELSE 4 END"
	case "location_code":
		return "location_code"
	case "warehouse_code":
		return "warehouse_code"
	case "updated_at":
		return "balance_updated_at"
	default:
		return "sparepart_code"
	}
}
