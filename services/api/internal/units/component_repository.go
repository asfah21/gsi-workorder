package units

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const componentSelect = `
	SELECT id, code, name, COALESCE(category, ''), component_class,
	       default_lifetime_hm, default_lifetime_days, default_lifetime_km, is_active
	FROM components
`

const componentInstallationSelect = `
	SELECT i.id, i.unit_id, u.code, i.component_id, c.code, c.name,
	       COALESCE(c.component_class, ''), i.serial_number, i.installed_at,
	       i.installed_hm, i.installed_km, i.removed_at, i.removed_hm, i.removed_km,
	       i.lifetime_hm, i.lifetime_days, i.lifetime_km, i.status,
	       i.source_wo_id, COALESCE(i.notes, ''), i.created_by, i.created_at, i.updated_at
	FROM unit_component_installations i
	JOIN units u ON u.id = i.unit_id
	JOIN components c ON c.id = i.component_id
`

func (r *PostgresRepository) ListComponents(ctx context.Context, activeOnly bool) ([]Component, error) {
	query := componentSelect
	if activeOnly {
		query += " WHERE is_active = true"
	}
	query += " ORDER BY code ASC LIMIT 500"
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list components: %w", err)
	}
	defer rows.Close()

	items := make([]Component, 0)
	for rows.Next() {
		item, err := scanComponent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan component: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate components: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) GetComponent(ctx context.Context, id int64) (Component, error) {
	item, err := scanComponent(r.pool.QueryRow(ctx, componentSelect+" WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Component{}, ErrNotFound
	}
	if err != nil {
		return Component{}, fmt.Errorf("get component: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) CreateComponent(ctx context.Context, component Component) (Component, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO components (
			code, name, category, component_class,
			default_lifetime_hm, default_lifetime_days, default_lifetime_km, is_active
		)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, $8)
		RETURNING id
	`, component.Code, component.Name, component.Category, component.ComponentClass,
		component.DefaultLifetimeHM, component.DefaultLifetimeDays,
		component.DefaultLifetimeKM, component.IsActive).Scan(&id)
	if err != nil {
		return Component{}, mapWriteError("create component", err)
	}
	return r.GetComponent(ctx, id)
}

func (r *PostgresRepository) UpdateComponent(ctx context.Context, component Component) (Component, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE components
		SET code = $2,
		    name = $3,
		    category = NULLIF($4, ''),
		    component_class = NULLIF($5, ''),
		    default_lifetime_hm = $6,
		    default_lifetime_days = $7,
		    default_lifetime_km = $8,
		    is_active = $9,
		    updated_at = now()
		WHERE id = $1
	`, component.ID, component.Code, component.Name, component.Category,
		component.ComponentClass, component.DefaultLifetimeHM,
		component.DefaultLifetimeDays, component.DefaultLifetimeKM, component.IsActive)
	if err != nil {
		return Component{}, mapWriteError("update component", err)
	}
	if tag.RowsAffected() == 0 {
		return Component{}, ErrNotFound
	}
	return r.GetComponent(ctx, component.ID)
}

func (r *PostgresRepository) ListComponentInstallations(ctx context.Context, unitID int64, activeOnly bool) ([]ComponentInstallation, error) {
	where := "i.unit_id = $1"
	if activeOnly {
		where += " AND i.status = 'active'"
	}
	rows, err := r.pool.Query(ctx, componentInstallationSelect+`
		WHERE `+where+`
		ORDER BY (i.status = 'active') DESC, i.installed_at DESC, i.id DESC
		LIMIT 500
	`, unitID)
	if err != nil {
		return nil, fmt.Errorf("list component installations: %w", err)
	}
	defer rows.Close()

	items := make([]ComponentInstallation, 0)
	for rows.Next() {
		item, err := scanComponentInstallation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan component installation: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate component installations: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) GetComponentInstallation(ctx context.Context, id int64) (ComponentInstallation, error) {
	item, err := scanComponentInstallation(r.pool.QueryRow(ctx, componentInstallationSelect+" WHERE i.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ComponentInstallation{}, ErrNotFound
	}
	if err != nil {
		return ComponentInstallation{}, fmt.Errorf("get component installation: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) CreateComponentInstallation(ctx context.Context, input ComponentInstallationCreate) (ComponentInstallation, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO unit_component_installations (
			unit_id, component_id, serial_number, installed_at, installed_hm, installed_km,
			lifetime_hm, lifetime_days, lifetime_km, status, source_wo_id, notes, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active', $10, NULLIF($11, ''), $12)
		RETURNING id
	`, input.UnitID, input.ComponentID, input.SerialNumber, input.InstalledAt,
		input.InstalledHM, input.InstalledKM, input.LifetimeHM, input.LifetimeDays,
		input.LifetimeKM, nullableID(input.SourceWOID), input.Notes, nullableID(input.CreatedBy)).Scan(&id)
	if err != nil {
		return ComponentInstallation{}, mapWriteError("create component installation", err)
	}
	return r.GetComponentInstallation(ctx, id)
}

// CloseAndInstallComponent closes one active installation and creates the next
// installation in the same transaction. Replacement and transfer share this
// path so history cannot be half-written.
func (r *PostgresRepository) CloseAndInstallComponent(ctx context.Context, oldID int64, close ComponentInstallationClose, create ComponentInstallationCreate) (ComponentInstallation, error) {
	if close.Status != installationStatusRemoved && close.Status != installationStatusTransferred {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{
			"status": "Status penutupan instalasi tidak valid",
		}}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ComponentInstallation{}, fmt.Errorf("begin component installation transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var oldUnitID, oldComponentID int64
	var oldStatus string
	var oldInstalledAt time.Time
	var oldInstalledHM, oldInstalledKM float64
	err = tx.QueryRow(ctx, `
		SELECT unit_id, component_id, status, installed_at, installed_hm, installed_km
		FROM unit_component_installations
		WHERE id = $1
		FOR UPDATE
	`, oldID).Scan(&oldUnitID, &oldComponentID, &oldStatus, &oldInstalledAt, &oldInstalledHM, &oldInstalledKM)
	if errors.Is(err, pgx.ErrNoRows) {
		return ComponentInstallation{}, ErrNotFound
	}
	if err != nil {
		return ComponentInstallation{}, fmt.Errorf("lock component installation: %w", err)
	}
	if oldStatus != installationStatusActive {
		return ComponentInstallation{}, fmt.Errorf("%w: component installation is not active", ErrConflict)
	}
	if create.ComponentID != oldComponentID {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{
			"component_id": "Komponen pengganti harus sama dengan komponen lama",
		}}
	}
	if close.RemovedAt.Before(oldInstalledAt) {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{
			"removed_at": "Tanggal pelepasan tidak boleh sebelum tanggal pemasangan",
		}}
	}
	if close.RemovedHM != nil && *close.RemovedHM < oldInstalledHM {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{
			"removed_hm": "HM pelepasan lebih kecil dari HM pemasangan",
		}}
	}
	if close.RemovedKM != nil && *close.RemovedKM < oldInstalledKM {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{
			"removed_km": "KM pelepasan lebih kecil dari KM pemasangan",
		}}
	}

	tag, err := tx.Exec(ctx, `
		UPDATE unit_component_installations
		SET status = $2,
		    removed_at = $3,
		    removed_hm = $4,
		    removed_km = $5,
		    updated_at = now()
		WHERE id = $1
	`, oldID, close.Status, close.RemovedAt, close.RemovedHM, close.RemovedKM)
	if err != nil {
		return ComponentInstallation{}, mapWriteError("close component installation", err)
	}
	if tag.RowsAffected() == 0 {
		return ComponentInstallation{}, ErrNotFound
	}

	var newID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO unit_component_installations (
			unit_id, component_id, serial_number, installed_at, installed_hm, installed_km,
			lifetime_hm, lifetime_days, lifetime_km, status, source_wo_id, notes, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active', $10, NULLIF($11, ''), $12)
		RETURNING id
	`, create.UnitID, create.ComponentID, create.SerialNumber, create.InstalledAt,
		create.InstalledHM, create.InstalledKM, create.LifetimeHM, create.LifetimeDays,
		create.LifetimeKM, nullableID(create.SourceWOID), create.Notes, nullableID(create.CreatedBy)).Scan(&newID)
	if err != nil {
		return ComponentInstallation{}, mapWriteError("create replacement component installation", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ComponentInstallation{}, fmt.Errorf("commit component installation transition: %w", err)
	}
	return r.GetComponentInstallation(ctx, newID)
}

func (r *PostgresRepository) ListComponentLifetime(ctx context.Context, unitID int64, activeOnly bool) ([]ComponentLifetime, error) {
	where := "unit_id = $1"
	if activeOnly {
		where += " AND status = 'active'"
	}
	rows, err := r.pool.Query(ctx, `
		SELECT installation_id, unit_id, unit_code, unit_current_hm, unit_current_km,
		       component_id, component_code, component_name, category, component_class,
		       serial_number, installed_at, installed_hm, installed_km,
		       removed_at, removed_hm, removed_km, status,
		       lifetime_hm, lifetime_days, lifetime_km,
		       used_hm, used_km, used_days,
		       remaining_hm, remaining_days, remaining_km
		FROM v_unit_component_lifetime
		WHERE `+where+`
		ORDER BY (status = 'active') DESC, installed_at DESC, installation_id DESC
		LIMIT 500
	`, unitID)
	if err != nil {
		return nil, fmt.Errorf("list component lifetime: %w", err)
	}
	defer rows.Close()

	items := make([]ComponentLifetime, 0)
	for rows.Next() {
		var item ComponentLifetime
		var removedHM, removedKM pgtype.Numeric
		var lifetimeHM, lifetimeKM pgtype.Numeric
		var lifetimeDays pgtype.Int4
		var remainingHM, remainingDays, remainingKM pgtype.Numeric
		if err := rows.Scan(
			&item.InstallationID, &item.UnitID, &item.UnitCode,
			&item.UnitCurrentHM, &item.UnitCurrentKM,
			&item.ComponentID, &item.ComponentCode, &item.ComponentName,
			&item.Category, &item.ComponentClass, &item.SerialNumber,
			&item.InstalledAt, &item.InstalledHM, &item.InstalledKM,
			&item.RemovedAt, &removedHM, &removedKM, &item.Status,
			&lifetimeHM, &lifetimeDays, &lifetimeKM,
			&item.UsedHM, &item.UsedKM, &item.UsedDays,
			&remainingHM, &remainingDays, &remainingKM,
		); err != nil {
			return nil, fmt.Errorf("scan component lifetime: %w", err)
		}
		item.RemovedHM = numericPointer(removedHM)
		item.RemovedKM = numericPointer(removedKM)
		item.LifetimeHM = numericPointer(lifetimeHM)
		item.LifetimeDays = intPointer(lifetimeDays)
		item.LifetimeKM = numericPointer(lifetimeKM)
		item.RemainingHM = numericPointer(remainingHM)
		item.RemainingDays = numericPointer(remainingDays)
		item.RemainingKM = numericPointer(remainingKM)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate component lifetime: %w", err)
	}
	return items, nil
}

func scanComponent(row rowScanner) (Component, error) {
	var item Component
	var componentClass pgtype.Text
	var lifetimeHM, lifetimeKM pgtype.Numeric
	var lifetimeDays pgtype.Int4
	err := row.Scan(
		&item.ID, &item.Code, &item.Name, &item.Category, &componentClass,
		&lifetimeHM, &lifetimeDays, &lifetimeKM, &item.IsActive,
	)
	if err != nil {
		return Component{}, err
	}
	item.ComponentClass = componentClass.String
	item.DefaultLifetimeHM = numericPointer(lifetimeHM)
	item.DefaultLifetimeDays = intPointer(lifetimeDays)
	item.DefaultLifetimeKM = numericPointer(lifetimeKM)
	return item, nil
}

func scanComponentInstallation(row rowScanner) (ComponentInstallation, error) {
	var item ComponentInstallation
	var removedHM, removedKM pgtype.Numeric
	var lifetimeHM, lifetimeKM pgtype.Numeric
	var lifetimeDays pgtype.Int4
	err := row.Scan(
		&item.ID, &item.UnitID, &item.UnitCode, &item.ComponentID,
		&item.ComponentCode, &item.ComponentName, &item.ComponentClass,
		&item.SerialNumber, &item.InstalledAt, &item.InstalledHM, &item.InstalledKM,
		&item.RemovedAt, &removedHM, &removedKM,
		&lifetimeHM, &lifetimeDays, &lifetimeKM,
		&item.Status, &item.SourceWOID, &item.Notes, &item.CreatedBy,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return ComponentInstallation{}, err
	}
	item.RemovedHM = numericPointer(removedHM)
	item.RemovedKM = numericPointer(removedKM)
	item.LifetimeHM = numericPointer(lifetimeHM)
	item.LifetimeDays = intPointer(lifetimeDays)
	item.LifetimeKM = numericPointer(lifetimeKM)
	return item, nil
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
