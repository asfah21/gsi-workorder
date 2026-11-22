package units

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const unitSelect = `
	SELECT u.id, u.odoo_unit_id, u.code,
	       COALESCE(u.unit_type, ''), COALESCE(u.merk, ''), COALESCE(u.class_unit, ''),
	       COALESCE(u.license_plate, ''), COALESCE(u.site_location, ''),
	       u.unit_model_id, COALESCE(um.manufacturer, ''), COALESCE(um.model, ''),
	       u.engine_model_id, COALESCE(em.manufacturer, ''), COALESCE(em.model, ''),
	       COALESCE(u.serial_number, ''), COALESCE(u.engine_serial_number, ''),
	       u.current_hm, u.current_km, u.hm_updated_at, u.km_updated_at,
	       u.is_breakdown, u.is_active, u.synced_at
	FROM units u
	LEFT JOIN unit_models um ON um.id = u.unit_model_id
	LEFT JOIN engine_models em ON em.id = u.engine_model_id
`

const unitInsertSQL = `
	INSERT INTO units (
		code, unit_type, merk, class_unit, license_plate, site_location,
		unit_model_id, engine_model_id, serial_number, engine_serial_number,
		is_breakdown, is_active
	)
	VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''),
	        NULLIF($6, ''), $7, $8, NULLIF($9, ''), NULLIF($10, ''), $11, $12)
	RETURNING id
`

func (r *PostgresRepository) ListUnitModels(ctx context.Context, filter ModelFilter) ([]UnitModel, int, error) {
	args := make([]any, 0, 3)
	where := []string{"1 = 1"}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		where = append(where, fmt.Sprintf("(manufacturer ILIKE $%d OR model ILIKE $%d OR COALESCE(category, '') ILIKE $%d)", len(args), len(args), len(args)))
	}
	if filter.ActiveOnly {
		where = append(where, "is_active = true")
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM unit_models WHERE "+strings.Join(where, " AND "), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count unit models: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	orderColumn := unitModelSortColumn(filter.Sort)
	orderDirection := "ASC"
	if filter.Order == "desc" {
		orderDirection = "DESC"
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, manufacturer, model, COALESCE(category, ''), COALESCE(notes, ''),
		       is_active, created_at, updated_at
		FROM unit_models
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY `+orderColumn+` `+orderDirection+`, id ASC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list unit models: %w", err)
	}
	defer rows.Close()

	models := make([]UnitModel, 0)
	for rows.Next() {
		var model UnitModel
		if err := rows.Scan(&model.ID, &model.Manufacturer, &model.Model, &model.Category,
			&model.Notes, &model.IsActive, &model.CreatedAt, &model.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan unit model: %w", err)
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate unit models: %w", err)
	}
	return models, total, nil
}

func (r *PostgresRepository) GetUnitModel(ctx context.Context, id int64) (UnitModel, error) {
	var model UnitModel
	err := r.pool.QueryRow(ctx, `
		SELECT id, manufacturer, model, COALESCE(category, ''), COALESCE(notes, ''),
		       is_active, created_at, updated_at
		FROM unit_models
		WHERE id = $1
	`, id).Scan(&model.ID, &model.Manufacturer, &model.Model, &model.Category,
		&model.Notes, &model.IsActive, &model.CreatedAt, &model.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return UnitModel{}, ErrNotFound
	}
	if err != nil {
		return UnitModel{}, fmt.Errorf("get unit model: %w", err)
	}
	return model, nil
}

func (r *PostgresRepository) CreateUnitModel(ctx context.Context, model UnitModel) (UnitModel, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO unit_models (manufacturer, model, category, notes, is_active)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5)
		RETURNING id, created_at, updated_at
	`, model.Manufacturer, model.Model, model.Category, model.Notes, model.IsActive).Scan(
		&model.ID, &model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		return UnitModel{}, mapWriteError("create unit model", err)
	}
	return model, nil
}

func (r *PostgresRepository) UpdateUnitModel(ctx context.Context, model UnitModel) (UnitModel, error) {
	err := r.pool.QueryRow(ctx, `
		UPDATE unit_models
		SET manufacturer = $2,
		    model = $3,
		    category = NULLIF($4, ''),
		    notes = NULLIF($5, ''),
		    is_active = $6,
		    updated_at = now()
		WHERE id = $1
		RETURNING created_at, updated_at
	`, model.ID, model.Manufacturer, model.Model, model.Category, model.Notes, model.IsActive).Scan(
		&model.CreatedAt, &model.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return UnitModel{}, ErrNotFound
	}
	if err != nil {
		return UnitModel{}, mapWriteError("update unit model", err)
	}
	return model, nil
}

func (r *PostgresRepository) ListEngineModels(ctx context.Context, filter ModelFilter) ([]EngineModel, int, error) {
	args := make([]any, 0, 3)
	where := []string{"1 = 1"}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		where = append(where, fmt.Sprintf("(manufacturer ILIKE $%d OR model ILIKE $%d OR COALESCE(fuel_type, '') ILIKE $%d)", len(args), len(args), len(args)))
	}
	if filter.ActiveOnly {
		where = append(where, "is_active = true")
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM engine_models WHERE "+strings.Join(where, " AND "), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count engine models: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	orderColumn := engineModelSortColumn(filter.Sort)
	orderDirection := "ASC"
	if filter.Order == "desc" {
		orderDirection = "DESC"
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, manufacturer, model, COALESCE(fuel_type, ''), displacement_cc,
		       COALESCE(notes, ''), is_active, created_at, updated_at
		FROM engine_models
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY `+orderColumn+` `+orderDirection+`, id ASC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list engine models: %w", err)
	}
	defer rows.Close()

	models := make([]EngineModel, 0)
	for rows.Next() {
		var model EngineModel
		var displacement *int32
		if err := rows.Scan(&model.ID, &model.Manufacturer, &model.Model, &model.FuelType,
			&displacement, &model.Notes, &model.IsActive, &model.CreatedAt, &model.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan engine model: %w", err)
		}
		if displacement != nil {
			value := int(*displacement)
			model.DisplacementCC = &value
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate engine models: %w", err)
	}
	return models, total, nil
}

func (r *PostgresRepository) GetEngineModel(ctx context.Context, id int64) (EngineModel, error) {
	var model EngineModel
	var displacement *int32
	err := r.pool.QueryRow(ctx, `
		SELECT id, manufacturer, model, COALESCE(fuel_type, ''), displacement_cc,
		       COALESCE(notes, ''), is_active, created_at, updated_at
		FROM engine_models
		WHERE id = $1
	`, id).Scan(&model.ID, &model.Manufacturer, &model.Model, &model.FuelType,
		&displacement, &model.Notes, &model.IsActive, &model.CreatedAt, &model.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return EngineModel{}, ErrNotFound
	}
	if err != nil {
		return EngineModel{}, fmt.Errorf("get engine model: %w", err)
	}
	if displacement != nil {
		value := int(*displacement)
		model.DisplacementCC = &value
	}
	return model, nil
}

func (r *PostgresRepository) CreateEngineModel(ctx context.Context, model EngineModel) (EngineModel, error) {
	var displacement any
	if model.DisplacementCC != nil {
		displacement = *model.DisplacementCC
	}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO engine_models (manufacturer, model, fuel_type, displacement_cc, notes, is_active)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6)
		RETURNING id, created_at, updated_at
	`, model.Manufacturer, model.Model, model.FuelType, displacement, model.Notes, model.IsActive).Scan(
		&model.ID, &model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		return EngineModel{}, mapWriteError("create engine model", err)
	}
	return model, nil
}

func (r *PostgresRepository) UpdateEngineModel(ctx context.Context, model EngineModel) (EngineModel, error) {
	var displacement any
	if model.DisplacementCC != nil {
		displacement = *model.DisplacementCC
	}
	err := r.pool.QueryRow(ctx, `
		UPDATE engine_models
		SET manufacturer = $2,
		    model = $3,
		    fuel_type = NULLIF($4, ''),
		    displacement_cc = $5,
		    notes = NULLIF($6, ''),
		    is_active = $7,
		    updated_at = now()
		WHERE id = $1
		RETURNING created_at, updated_at
	`, model.ID, model.Manufacturer, model.Model, model.FuelType, displacement, model.Notes, model.IsActive).Scan(
		&model.CreatedAt, &model.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return EngineModel{}, ErrNotFound
	}
	if err != nil {
		return EngineModel{}, mapWriteError("update engine model", err)
	}
	return model, nil
}

func (r *PostgresRepository) ListUnits(ctx context.Context, filter UnitFilter) ([]Unit, int, error) {
	args := make([]any, 0, 8)
	where := []string{"1 = 1"}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		where = append(where, fmt.Sprintf("(u.code ILIKE $%d OR COALESCE(u.license_plate, '') ILIKE $%d OR COALESCE(u.site_location, '') ILIKE $%d OR COALESCE(u.merk, '') ILIKE $%d)", len(args), len(args), len(args), len(args)))
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		where = append(where, fmt.Sprintf("u.is_active = $%d", len(args)))
	}
	if filter.UnitModelID != nil {
		args = append(args, *filter.UnitModelID)
		where = append(where, fmt.Sprintf("u.unit_model_id = $%d", len(args)))
	}
	if filter.SiteLocation != "" {
		args = append(args, "%"+filter.SiteLocation+"%")
		where = append(where, fmt.Sprintf("COALESCE(u.site_location, '') ILIKE $%d", len(args)))
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM units u WHERE "+strings.Join(where, " AND "), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count units: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	orderColumn := unitSortColumn(filter.Sort)
	orderDirection := "ASC"
	if filter.Order == "desc" {
		orderDirection = "DESC"
	}

	rows, err := r.pool.Query(ctx, unitSelect+`
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY `+orderColumn+` `+orderDirection+`, u.id ASC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list units: %w", err)
	}
	defer rows.Close()

	result := make([]Unit, 0)
	for rows.Next() {
		unit, err := scanUnit(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, unit)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate units: %w", err)
	}
	return result, total, nil
}

func (r *PostgresRepository) GetUnit(ctx context.Context, id int64) (Unit, error) {
	unit, err := scanUnit(r.pool.QueryRow(ctx, unitSelect+" WHERE u.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Unit{}, ErrNotFound
	}
	if err != nil {
		return Unit{}, err
	}
	return unit, nil
}

func (r *PostgresRepository) CreateUnit(ctx context.Context, unit Unit) (Unit, error) {
	var id int64
	err := r.pool.QueryRow(ctx, unitInsertSQL, unit.Code, unit.UnitType, unit.Merk, unit.ClassUnit, unit.LicensePlate, unit.SiteLocation,
		nullableID(unit.UnitModelID), nullableID(unit.EngineModelID), unit.SerialNumber,
		unit.EngineSerialNumber, unit.IsBreakdown, unit.IsActive).Scan(&id)
	if err != nil {
		return Unit{}, mapWriteError("create unit", err)
	}
	return r.GetUnit(ctx, id)
}

func (r *PostgresRepository) CreateUnits(ctx context.Context, items []Unit) ([]Unit, error) {
	if len(items) == 0 {
		return []Unit{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create units: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created := make([]Unit, 0, len(items))
	for _, item := range items {
		var id int64
		err := tx.QueryRow(ctx, unitInsertSQL, item.Code, item.UnitType, item.Merk, item.ClassUnit,
			item.LicensePlate, item.SiteLocation, nullableID(item.UnitModelID),
			nullableID(item.EngineModelID), item.SerialNumber, item.EngineSerialNumber,
			item.IsBreakdown, item.IsActive).Scan(&id)
		if err != nil {
			return nil, mapWriteError("create unit", err)
		}
		item.ID = id
		created = append(created, item)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create units: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) ExistingUnitCodes(ctx context.Context, codes []string) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(codes))
	if len(codes) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT code FROM units WHERE code = ANY($1)`, codes)
	if err != nil {
		return nil, fmt.Errorf("list existing unit codes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("scan existing unit code: %w", err)
		}
		result[code] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate existing unit codes: %w", err)
	}
	return result, nil
}

func (r *PostgresRepository) UpdateUnit(ctx context.Context, unit Unit) (Unit, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE units
		SET code = $2,
		    unit_type = NULLIF($3, ''),
		    merk = NULLIF($4, ''),
		    class_unit = NULLIF($5, ''),
		    license_plate = NULLIF($6, ''),
		    site_location = NULLIF($7, ''),
		    unit_model_id = $8,
		    engine_model_id = $9,
		    serial_number = NULLIF($10, ''),
		    engine_serial_number = NULLIF($11, ''),
		    is_breakdown = $12,
		    is_active = $13
		WHERE id = $1
	`, unit.ID, unit.Code, unit.UnitType, unit.Merk, unit.ClassUnit, unit.LicensePlate,
		unit.SiteLocation, nullableID(unit.UnitModelID), nullableID(unit.EngineModelID),
		unit.SerialNumber, unit.EngineSerialNumber, unit.IsBreakdown, unit.IsActive)
	if err != nil {
		return Unit{}, mapWriteError("update unit", err)
	}
	return r.GetUnit(ctx, unit.ID)
}

func (r *PostgresRepository) ListMeterReadings(ctx context.Context, unitID int64, limit int) ([]MeterReading, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, unit_id, reading_type, value, recorded_at, source_type, source_id,
		       COALESCE(notes, ''), created_at
		FROM unit_meter_readings
		WHERE unit_id = $1
		ORDER BY recorded_at DESC, id DESC
		LIMIT $2
	`, unitID, limit)
	if err != nil {
		return nil, fmt.Errorf("list meter readings: %w", err)
	}
	defer rows.Close()

	readings := make([]MeterReading, 0)
	for rows.Next() {
		var reading MeterReading
		if err := rows.Scan(&reading.ID, &reading.UnitID, &reading.ReadingType, &reading.Value,
			&reading.RecordedAt, &reading.SourceType, &reading.SourceID, &reading.Notes,
			&reading.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan meter reading: %w", err)
		}
		readings = append(readings, reading)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate meter readings: %w", err)
	}
	return readings, nil
}

func (r *PostgresRepository) AddManualReading(ctx context.Context, unitID int64, input MeterReadingInput, actorRole string) (MeterReading, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return MeterReading{}, fmt.Errorf("begin meter reading: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentHM, currentKM float64
	var hmUpdatedAt, kmUpdatedAt *time.Time
	var isActive bool
	err = tx.QueryRow(ctx, `
		SELECT current_hm, current_km, hm_updated_at, km_updated_at, is_active
		FROM units
		WHERE id = $1
		FOR UPDATE
	`, unitID).Scan(&currentHM, &currentKM, &hmUpdatedAt, &kmUpdatedAt, &isActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return MeterReading{}, ErrNotFound
	}
	if err != nil {
		return MeterReading{}, fmt.Errorf("lock unit for reading: %w", err)
	}
	if !isActive {
		return MeterReading{}, ErrInactive
	}

	current := currentHM
	currentUpdatedAt := hmUpdatedAt
	if input.ReadingType == "km" {
		current = currentKM
		currentUpdatedAt = kmUpdatedAt
	}
	if input.Value < current {
		if actorRole != "admin" {
			return MeterReading{}, ErrReadingRegression
		}
		if len(strings.TrimSpace(input.Notes)) < 5 {
			return MeterReading{}, &ValidationError{Fields: map[string]string{
				"notes": "Alasan koreksi reading minimal 5 karakter",
			}}
		}
	}

	reading := MeterReading{
		UnitID:      unitID,
		ReadingType: input.ReadingType,
		Value:       input.Value,
		RecordedAt:  input.RecordedAt,
		SourceType:  "manual",
		Notes:       input.Notes,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO unit_meter_readings (unit_id, reading_type, value, recorded_at, source_type, notes)
		VALUES ($1, $2, $3, $4, 'manual', NULLIF($5, ''))
		RETURNING id, created_at
	`, unitID, input.ReadingType, input.Value, input.RecordedAt, input.Notes).Scan(
		&reading.ID, &reading.CreatedAt)
	if err != nil {
		return MeterReading{}, fmt.Errorf("insert meter reading: %w", err)
	}

	if currentUpdatedAt == nil || !input.RecordedAt.Before(*currentUpdatedAt) {
		if input.ReadingType == "hm" {
			_, err = tx.Exec(ctx, `UPDATE units SET current_hm = $2, hm_updated_at = $3 WHERE id = $1`,
				unitID, input.Value, input.RecordedAt)
		} else {
			_, err = tx.Exec(ctx, `UPDATE units SET current_km = $2, km_updated_at = $3 WHERE id = $1`,
				unitID, input.Value, input.RecordedAt)
		}
		if err != nil {
			return MeterReading{}, fmt.Errorf("update unit meter cache: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return MeterReading{}, fmt.Errorf("commit meter reading: %w", err)
	}
	return reading, nil
}

func (r *PostgresRepository) DashboardSummary(ctx context.Context) (DashboardSummary, error) {
	var summary DashboardSummary
	err := r.pool.QueryRow(ctx, `
		SELECT
			count(*),
			count(*) FILTER (WHERE is_active),
			count(*) FILTER (WHERE is_breakdown AND is_active),
			(SELECT count(*) FROM unit_models WHERE is_active),
			(SELECT count(*) FROM engine_models WHERE is_active),
			(SELECT count(*) FROM unit_meter_readings WHERE recorded_at >= now() - interval '30 days')
		FROM units
	`).Scan(&summary.UnitsTotal, &summary.UnitsActive, &summary.UnitsBreakdown,
		&summary.UnitModels, &summary.EngineModels, &summary.ReadingsLast30Days)
	if err != nil {
		return DashboardSummary{}, fmt.Errorf("dashboard summary: %w", err)
	}
	return summary, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUnit(row rowScanner) (Unit, error) {
	var unit Unit
	var unitModelID, engineModelID *int64
	var unitModelManufacturer, unitModelName string
	var engineModelManufacturer, engineModelName string
	var odooUnitID *int64
	var syncedAt *time.Time

	err := row.Scan(
		&unit.ID, &odooUnitID, &unit.Code,
		&unit.UnitType, &unit.Merk, &unit.ClassUnit, &unit.LicensePlate, &unit.SiteLocation,
		&unitModelID, &unitModelManufacturer, &unitModelName,
		&engineModelID, &engineModelManufacturer, &engineModelName,
		&unit.SerialNumber, &unit.EngineSerialNumber,
		&unit.CurrentHM, &unit.CurrentKM, &unit.HMUpdatedAt, &unit.KMUpdatedAt,
		&unit.IsBreakdown, &unit.IsActive, &syncedAt,
	)
	if err != nil {
		return Unit{}, err
	}
	unit.OdooUnitID = odooUnitID
	unit.SyncedAt = syncedAt
	unit.UnitModelID = unitModelID
	if unitModelID != nil {
		unit.UnitModel = &UnitModelSummary{
			ID:           *unitModelID,
			Manufacturer: unitModelManufacturer,
			Model:        unitModelName,
		}
	}
	unit.EngineModelID = engineModelID
	if engineModelID != nil {
		unit.EngineModel = &EngineModelSummary{
			ID:           *engineModelID,
			Manufacturer: engineModelManufacturer,
			Model:        engineModelName,
		}
	}
	return unit, nil
}

func unitSortColumn(value string) string {
	switch value {
	case "unit_type":
		return "COALESCE(u.unit_type, '')"
	case "model":
		return "COALESCE(um.manufacturer, '') || ' ' || COALESCE(um.model, '')"
	case "site_location":
		return "COALESCE(u.site_location, '')"
	case "current_hm":
		return "u.current_hm"
	case "current_km":
		return "u.current_km"
	case "is_active":
		return "u.is_active"
	case "is_breakdown":
		return "u.is_breakdown"
	default:
		return "u.code"
	}
}

func unitModelSortColumn(value string) string {
	switch value {
	case "model":
		return "model"
	case "category":
		return "COALESCE(category, '')"
	case "is_active":
		return "is_active"
	case "updated_at":
		return "updated_at"
	default:
		return "manufacturer"
	}
}

func engineModelSortColumn(value string) string {
	switch value {
	case "model":
		return "model"
	case "fuel_type":
		return "COALESCE(fuel_type, '')"
	case "displacement_cc":
		return "displacement_cc"
	case "is_active":
		return "is_active"
	case "updated_at":
		return "updated_at"
	default:
		return "manufacturer"
	}
}

func nullableID(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func mapWriteError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: duplicate data", ErrConflict)
		case "23503":
			return fmt.Errorf("%w: referenced record not found", ErrValidation)
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}
