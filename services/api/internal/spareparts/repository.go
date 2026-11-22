package spareparts

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const sparepartSelect = `
	SELECT s.id, s.code, s.name, COALESCE(s.category, ''), s.uom,
	       COALESCE(s.description, ''), s.is_active, COALESCE(pn.part_number, ''),
	       (SELECT count(*) FROM sparepart_compatibility sc WHERE sc.sparepart_id = s.id),
	       (SELECT count(*) FROM sparepart_compatibility sc
	        WHERE sc.sparepart_id = s.id
	          AND sc.status IN ('approved_compatible', 'conditional_compatible')),
	       s.created_at, s.updated_at
	FROM spareparts s
	LEFT JOIN LATERAL (
		SELECT part_number
		FROM sparepart_numbers
		WHERE sparepart_id = s.id
		ORDER BY is_primary DESC, id ASC
		LIMIT 1
	) pn ON true
`

func (r *Repository) List(ctx context.Context, filter Filter) ([]Sparepart, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 25
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	args := make([]any, 0, 8)
	where := []string{"1 = 1"}
	if filter.Query != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Query)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf(`(
			s.code ILIKE $%d OR s.name ILIKE $%d OR COALESCE(s.category, '') ILIKE $%d
			OR COALESCE(s.description, '') ILIKE $%d
			OR EXISTS (
				SELECT 1 FROM sparepart_numbers num
				WHERE num.sparepart_id = s.id
				  AND (num.part_number ILIKE $%d OR COALESCE(num.brand, '') ILIKE $%d)
			)
			OR EXISTS (
				SELECT 1
				FROM sparepart_compatibility sc
				LEFT JOIN unit_models um ON um.id = sc.unit_model_id
				LEFT JOIN engine_models em ON em.id = sc.engine_model_id
				WHERE sc.sparepart_id = s.id
				  AND (COALESCE(um.manufacturer, '') ILIKE $%d
				       OR COALESCE(um.model, '') ILIKE $%d
				       OR COALESCE(em.manufacturer, '') ILIKE $%d
				       OR COALESCE(em.model, '') ILIKE $%d
				       OR COALESCE(sc.configuration, '') ILIKE $%d)
			)
		)`, n, n, n, n, n, n, n, n, n, n, n))
	}
	if filter.Category != "" {
		args = append(args, filter.Category)
		where = append(where, fmt.Sprintf("s.category = $%d", len(args)))
	}
	if filter.ActiveOnly != nil {
		args = append(args, *filter.ActiveOnly)
		where = append(where, fmt.Sprintf("s.is_active = $%d", len(args)))
	}
	if filter.UnitModelID != nil {
		args = append(args, *filter.UnitModelID)
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM sparepart_compatibility sc
			WHERE sc.sparepart_id = s.id AND sc.unit_model_id = $%d
		)`, len(args)))
	}
	if filter.EngineModelID != nil {
		args = append(args, *filter.EngineModelID)
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM sparepart_compatibility sc
			WHERE sc.sparepart_id = s.id AND sc.engine_model_id = $%d
		)`, len(args)))
	}
	if filter.CompatibilityStatus != "" {
		args = append(args, filter.CompatibilityStatus)
		where = append(where, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM sparepart_compatibility sc
			WHERE sc.sparepart_id = s.id AND sc.status = $%d
		)`, len(args)))
	}

	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM spareparts s WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count spareparts: %w", err)
	}

	orderColumn := sparepartSortColumn(filter.Sort)
	orderDirection := "ASC"
	if strings.EqualFold(filter.Order, "desc") {
		orderDirection = "DESC"
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, sparepartSelect+`
		WHERE `+whereSQL+`
		ORDER BY `+orderColumn+` `+orderDirection+`, s.id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list spareparts: %w", err)
	}
	defer rows.Close()

	items := make([]Sparepart, 0)
	for rows.Next() {
		item, err := scanSparepart(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan sparepart: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate spareparts: %w", err)
	}
	return items, total, nil
}

func (r *Repository) Get(ctx context.Context, id int64) (Sparepart, error) {
	item, err := scanSparepart(r.pool.QueryRow(ctx, sparepartSelect+` WHERE s.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Sparepart{}, ErrNotFound
	}
	if err != nil {
		return Sparepart{}, fmt.Errorf("get sparepart: %w", err)
	}
	numbers, err := r.ListNumbers(ctx, id)
	if err != nil {
		return Sparepart{}, err
	}
	alternatives, err := r.ListAlternatives(ctx, id)
	if err != nil {
		return Sparepart{}, err
	}
	compatibility, err := r.ListCompatibility(ctx, id)
	if err != nil {
		return Sparepart{}, err
	}
	item.Numbers = numbers
	item.Alternatives = alternatives
	item.Compatibility = compatibility
	return item, nil
}

func (r *Repository) Create(ctx context.Context, actor Actor, input Input) (Sparepart, error) {
	if !canManage(actor.Role) {
		return Sparepart{}, ErrValidation
	}
	normalized, err := normalizeInput(input)
	if err != nil {
		return Sparepart{}, err
	}
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}
	var id int64
	err = r.pool.QueryRow(ctx, `
		INSERT INTO spareparts (code, name, category, uom, description, is_active)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6)
		RETURNING id
	`, normalized.Code, normalized.Name, normalized.Category, normalized.UOM,
		normalized.Description, active).Scan(&id)
	if isUniqueViolation(err) {
		return Sparepart{}, ErrConflict
	}
	if err != nil {
		return Sparepart{}, fmt.Errorf("create sparepart: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *Repository) Update(ctx context.Context, id int64, actor Actor, patch Patch) (Sparepart, error) {
	if !canManage(actor.Role) {
		return Sparepart{}, ErrValidation
	}
	current, err := r.Get(ctx, id)
	if err != nil {
		return Sparepart{}, err
	}
	next := current
	if patch.Code != nil {
		next.Code = strings.ToUpper(strings.TrimSpace(*patch.Code))
	}
	if patch.Name != nil {
		next.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.Category != nil {
		next.Category = strings.TrimSpace(*patch.Category)
	}
	if patch.UOM != nil {
		next.UOM = strings.TrimSpace(*patch.UOM)
	}
	if patch.Description != nil {
		next.Description = strings.TrimSpace(*patch.Description)
	}
	if patch.IsActive != nil {
		next.IsActive = *patch.IsActive
	}
	normalized, err := normalizeInput(Input{
		Code:        next.Code,
		Name:        next.Name,
		Category:    next.Category,
		UOM:         next.UOM,
		Description: next.Description,
	})
	if err != nil {
		return Sparepart{}, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE spareparts
		SET code = $2, name = $3, category = NULLIF($4, ''), uom = $5,
		    description = NULLIF($6, ''), is_active = $7, updated_at = now()
		WHERE id = $1
	`, id, normalized.Code, normalized.Name, normalized.Category, normalized.UOM,
		normalized.Description, next.IsActive)
	if isUniqueViolation(err) {
		return Sparepart{}, ErrConflict
	}
	if err != nil {
		return Sparepart{}, fmt.Errorf("update sparepart: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Sparepart{}, ErrNotFound
	}
	return r.Get(ctx, id)
}

func (r *Repository) ListNumbers(ctx context.Context, sparepartID int64) ([]Number, error) {
	if err := r.requireSparepart(ctx, sparepartID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, sparepart_id, part_number, COALESCE(brand, ''), number_type,
		       is_primary, COALESCE(notes, ''), created_at
		FROM sparepart_numbers
		WHERE sparepart_id = $1
		ORDER BY is_primary DESC, part_number ASC, id ASC
	`, sparepartID)
	if err != nil {
		return nil, fmt.Errorf("list sparepart numbers: %w", err)
	}
	defer rows.Close()
	items := make([]Number, 0)
	for rows.Next() {
		var item Number
		if err := rows.Scan(&item.ID, &item.SparepartID, &item.PartNumber, &item.Brand,
			&item.NumberType, &item.IsPrimary, &item.Notes, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan sparepart number: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sparepart numbers: %w", err)
	}
	return items, nil
}

func (r *Repository) CreateNumber(ctx context.Context, sparepartID int64, actor Actor, input NumberInput) (Number, error) {
	if !canManage(actor.Role) {
		return Number{}, ErrValidation
	}
	normalized, err := normalizeNumber(input)
	if err != nil {
		return Number{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Number{}, fmt.Errorf("begin sparepart number create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists int64
	if err := tx.QueryRow(ctx, `SELECT id FROM spareparts WHERE id = $1 FOR UPDATE`, sparepartID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Number{}, ErrNotFound
		}
		return Number{}, fmt.Errorf("lock sparepart: %w", err)
	}
	if normalized.IsPrimary {
		if _, err := tx.Exec(ctx, `UPDATE sparepart_numbers SET is_primary = false WHERE sparepart_id = $1`, sparepartID); err != nil {
			return Number{}, fmt.Errorf("clear primary sparepart number: %w", err)
		}
	}
	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO sparepart_numbers (sparepart_id, part_number, brand, number_type, is_primary, notes)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, NULLIF($6, ''))
		RETURNING id
	`, sparepartID, normalized.PartNumber, normalized.Brand, normalized.NumberType,
		normalized.IsPrimary, normalized.Notes).Scan(&id)
	if isUniqueViolation(err) {
		return Number{}, ErrConflict
	}
	if err != nil {
		return Number{}, fmt.Errorf("create sparepart number: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Number{}, fmt.Errorf("commit sparepart number create: %w", err)
	}
	return r.getNumber(ctx, sparepartID, id)
}

func (r *Repository) UpdateNumber(ctx context.Context, sparepartID, numberID int64, actor Actor, patch NumberPatch) (Number, error) {
	if !canManage(actor.Role) {
		return Number{}, ErrValidation
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Number{}, fmt.Errorf("begin sparepart number update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists int64
	if err := tx.QueryRow(ctx, `SELECT id FROM spareparts WHERE id = $1 FOR UPDATE`, sparepartID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Number{}, ErrNotFound
		}
		return Number{}, fmt.Errorf("lock sparepart: %w", err)
	}
	number, err := scanNumber(tx.QueryRow(ctx, `
		SELECT id, sparepart_id, part_number, COALESCE(brand, ''), number_type,
		       is_primary, COALESCE(notes, ''), created_at
		FROM sparepart_numbers
		WHERE id = $1 AND sparepart_id = $2
		FOR UPDATE
	`, numberID, sparepartID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Number{}, ErrNotFound
	}
	if err != nil {
		return Number{}, fmt.Errorf("get sparepart number: %w", err)
	}
	if patch.PartNumber != nil {
		number.PartNumber = strings.TrimSpace(*patch.PartNumber)
	}
	if patch.Brand != nil {
		number.Brand = strings.TrimSpace(*patch.Brand)
	}
	if patch.NumberType != nil {
		number.NumberType = strings.TrimSpace(*patch.NumberType)
	}
	if patch.IsPrimary != nil {
		number.IsPrimary = *patch.IsPrimary
	}
	if patch.Notes != nil {
		number.Notes = strings.TrimSpace(*patch.Notes)
	}
	normalized, err := normalizeNumber(NumberInput{
		PartNumber: number.PartNumber,
		Brand:      number.Brand,
		NumberType: number.NumberType,
		IsPrimary:  number.IsPrimary,
		Notes:      number.Notes,
	})
	if err != nil {
		return Number{}, err
	}
	if normalized.IsPrimary {
		if _, err := tx.Exec(ctx, `
			UPDATE sparepart_numbers SET is_primary = false
			WHERE sparepart_id = $1 AND id <> $2
		`, sparepartID, numberID); err != nil {
			return Number{}, fmt.Errorf("clear primary sparepart number: %w", err)
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE sparepart_numbers
		SET part_number = $2, brand = NULLIF($3, ''), number_type = $4,
		    is_primary = $5, notes = NULLIF($6, '')
		WHERE id = $1
	`, numberID, normalized.PartNumber, normalized.Brand, normalized.NumberType,
		normalized.IsPrimary, normalized.Notes)
	if isUniqueViolation(err) {
		return Number{}, ErrConflict
	}
	if err != nil {
		return Number{}, fmt.Errorf("update sparepart number: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Number{}, fmt.Errorf("commit sparepart number update: %w", err)
	}
	return r.getNumber(ctx, sparepartID, numberID)
}

func (r *Repository) ListAlternatives(ctx context.Context, sparepartID int64) ([]Alternative, error) {
	if err := r.requireSparepart(ctx, sparepartID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, alternativeSelect+`
		WHERE a.sparepart_id = $1 OR a.alternative_sparepart_id = $1
		ORDER BY a.id DESC
	`, sparepartID)
	if err != nil {
		return nil, fmt.Errorf("list sparepart alternatives: %w", err)
	}
	defer rows.Close()
	items := make([]Alternative, 0)
	for rows.Next() {
		item, err := scanAlternative(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sparepart alternative: %w", err)
		}
		if item.SparepartID == sparepartID {
			item.Direction = "outgoing"
		} else {
			item.Direction = "incoming"
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sparepart alternatives: %w", err)
	}
	return items, nil
}

func (r *Repository) CreateAlternative(ctx context.Context, sparepartID int64, actor Actor, input AlternativeInput) (Alternative, error) {
	if !canManage(actor.Role) {
		return Alternative{}, ErrValidation
	}
	relationType := strings.TrimSpace(input.RelationType)
	if relationType == "" {
		relationType = "equivalent"
	}
	if !validRelationType(relationType) {
		return Alternative{}, &ValidationError{Fields: map[string]string{"relation_type": "Tipe relasi tidak valid."}}
	}
	if input.AlternativeSparepartID <= 0 || input.AlternativeSparepartID == sparepartID {
		return Alternative{}, &ValidationError{Fields: map[string]string{"alternative_sparepart_id": "Sparepart alternatif wajib dipilih dan berbeda."}}
	}
	if err := r.requireSparepart(ctx, sparepartID); err != nil {
		return Alternative{}, err
	}
	if err := r.requireSparepart(ctx, input.AlternativeSparepartID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return Alternative{}, &ValidationError{Fields: map[string]string{"alternative_sparepart_id": "Sparepart alternatif tidak ditemukan."}}
		}
		return Alternative{}, err
	}
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO sparepart_alternatives (sparepart_id, alternative_sparepart_id, relation_type, notes)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		RETURNING id
	`, sparepartID, input.AlternativeSparepartID, relationType, strings.TrimSpace(input.Notes)).Scan(&id)
	if isUniqueViolation(err) {
		return Alternative{}, ErrConflict
	}
	if err != nil {
		return Alternative{}, fmt.Errorf("create sparepart alternative: %w", err)
	}
	return r.getAlternative(ctx, sparepartID, id)
}

func (r *Repository) VerifyAlternative(ctx context.Context, sparepartID, alternativeID int64, actor Actor, input VerifyInput) (Alternative, error) {
	if !canManage(actor.Role) {
		return Alternative{}, ErrValidation
	}
	status := strings.TrimSpace(input.Status)
	if !validVerificationStatus(status) || status == "pending_verification" {
		return Alternative{}, &ValidationError{Fields: map[string]string{"status": "Status verifikasi tidak valid."}}
	}
	notes := strings.TrimSpace(input.Notes)
	if status == "conditional_compatible" && notes == "" {
		return Alternative{}, &ValidationError{Fields: map[string]string{"notes": "Catatan kondisi wajib untuk Conditional Compatible."}}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Alternative{}, fmt.Errorf("begin alternative verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var existingNotes string
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(notes, '')
		FROM sparepart_alternatives
		WHERE id = $1 AND sparepart_id = $2
		FOR UPDATE
	`, alternativeID, sparepartID).Scan(&existingNotes); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Alternative{}, ErrNotFound
		}
		return Alternative{}, fmt.Errorf("lock sparepart alternative: %w", err)
	}
	if notes == "" {
		notes = existingNotes
	}
	if status == "conditional_compatible" && notes == "" {
		return Alternative{}, &ValidationError{Fields: map[string]string{"notes": "Catatan kondisi wajib untuk Conditional Compatible."}}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sparepart_alternatives
		SET verification_status = $3,
		    notes = NULLIF($4, ''),
		    verified_by = $5, verified_at = now(), updated_at = now()
		WHERE id = $1 AND sparepart_id = $2
	`, alternativeID, sparepartID, status, notes, actor.ID); err != nil {
		return Alternative{}, fmt.Errorf("verify sparepart alternative: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Alternative{}, fmt.Errorf("commit alternative verification: %w", err)
	}
	return r.getAlternative(ctx, sparepartID, alternativeID)
}

func (r *Repository) ListCompatibility(ctx context.Context, sparepartID int64) ([]Compatibility, error) {
	if err := r.requireSparepart(ctx, sparepartID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, compatibilitySelect+`
		WHERE sc.sparepart_id = $1
		ORDER BY sc.status, sc.id DESC
	`, sparepartID)
	if err != nil {
		return nil, fmt.Errorf("list sparepart compatibility: %w", err)
	}
	defer rows.Close()
	items := make([]Compatibility, 0)
	for rows.Next() {
		item, err := scanCompatibility(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sparepart compatibility: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sparepart compatibility: %w", err)
	}
	return items, nil
}

func (r *Repository) CreateCompatibility(ctx context.Context, sparepartID int64, actor Actor, input CompatibilityInput) (Compatibility, error) {
	if !canManage(actor.Role) {
		return Compatibility{}, ErrValidation
	}
	configuration := strings.TrimSpace(input.Configuration)
	if input.UnitModelID == nil && input.EngineModelID == nil && configuration == "" {
		return Compatibility{}, &ValidationError{Fields: map[string]string{
			"target": "Minimal satu target (unit model, engine model, atau konfigurasi) wajib diisi.",
		}}
	}
	if err := r.requireSparepart(ctx, sparepartID); err != nil {
		return Compatibility{}, err
	}
	if input.UnitModelID != nil {
		if err := r.requireActiveReference(ctx, "unit_models", *input.UnitModelID); err != nil {
			return Compatibility{}, &ValidationError{Fields: map[string]string{"unit_model_id": "Unit model tidak ditemukan atau tidak aktif."}}
		}
	}
	if input.EngineModelID != nil {
		if err := r.requireActiveReference(ctx, "engine_models", *input.EngineModelID); err != nil {
			return Compatibility{}, &ValidationError{Fields: map[string]string{"engine_model_id": "Engine model tidak ditemukan atau tidak aktif."}}
		}
	}
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO sparepart_compatibility (
			sparepart_id, unit_model_id, engine_model_id, configuration, source, notes
		)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''))
		RETURNING id
	`, sparepartID, input.UnitModelID, input.EngineModelID, configuration,
		strings.TrimSpace(input.Source), strings.TrimSpace(input.Notes)).Scan(&id)
	if isUniqueViolation(err) {
		return Compatibility{}, ErrConflict
	}
	if err != nil {
		return Compatibility{}, fmt.Errorf("create sparepart compatibility: %w", err)
	}
	return r.getCompatibility(ctx, sparepartID, id)
}

func (r *Repository) VerifyCompatibility(ctx context.Context, sparepartID, compatibilityID int64, actor Actor, input VerifyInput) (Compatibility, error) {
	if !canManage(actor.Role) {
		return Compatibility{}, ErrValidation
	}
	status := strings.TrimSpace(input.Status)
	if !validVerificationStatus(status) || status == "pending_verification" {
		return Compatibility{}, &ValidationError{Fields: map[string]string{"status": "Status verifikasi tidak valid."}}
	}
	conditionNote := strings.TrimSpace(input.ConditionNote)
	if status == "conditional_compatible" && conditionNote == "" {
		return Compatibility{}, &ValidationError{Fields: map[string]string{"condition_note": "Catatan kondisi wajib untuk Conditional Compatible."}}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Compatibility{}, fmt.Errorf("begin compatibility verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var exists int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM sparepart_compatibility
		WHERE id = $1 AND sparepart_id = $2
		FOR UPDATE
	`, compatibilityID, sparepartID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Compatibility{}, ErrNotFound
		}
		return Compatibility{}, fmt.Errorf("lock sparepart compatibility: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sparepart_compatibility
		SET status = $3,
		    condition_note = NULLIF($4, ''),
		    notes = CASE WHEN $5 = '' THEN notes ELSE $5 END,
		    source = CASE WHEN $6 = '' THEN source ELSE $6 END,
		    verified_by = $7, verified_at = now(), updated_at = now()
		WHERE id = $1 AND sparepart_id = $2
	`, compatibilityID, sparepartID, status, conditionNote,
		strings.TrimSpace(input.Notes), strings.TrimSpace(input.Source), actor.ID); err != nil {
		return Compatibility{}, fmt.Errorf("verify sparepart compatibility: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Compatibility{}, fmt.Errorf("commit compatibility verification: %w", err)
	}
	return r.getCompatibility(ctx, sparepartID, compatibilityID)
}

const alternativeSelect = `
	SELECT a.id, a.sparepart_id, sp.code, sp.name,
	       a.alternative_sparepart_id, alt.code, alt.name,
	       a.relation_type, a.verification_status, COALESCE(a.notes, ''),
	       a.verified_by, COALESCE(u.name, ''), a.verified_at, a.created_at, a.updated_at
	FROM sparepart_alternatives a
	JOIN spareparts sp ON sp.id = a.sparepart_id
	JOIN spareparts alt ON alt.id = a.alternative_sparepart_id
	LEFT JOIN users u ON u.id = a.verified_by
`

const compatibilitySelect = `
	SELECT sc.id, sc.sparepart_id, sc.unit_model_id,
	       TRIM(COALESCE(um.manufacturer, '') || CASE WHEN um.model IS NULL THEN '' ELSE ' ' || um.model END),
	       sc.engine_model_id,
	       TRIM(COALESCE(em.manufacturer, '') || CASE WHEN em.model IS NULL THEN '' ELSE ' ' || em.model END),
	       COALESCE(sc.configuration, ''), sc.status, COALESCE(sc.condition_note, ''),
	       COALESCE(sc.source, ''), COALESCE(sc.notes, ''), sc.verified_by, COALESCE(u.name, ''),
	       sc.verified_at, sc.created_at, sc.updated_at
	FROM sparepart_compatibility sc
	LEFT JOIN unit_models um ON um.id = sc.unit_model_id
	LEFT JOIN engine_models em ON em.id = sc.engine_model_id
	LEFT JOIN users u ON u.id = sc.verified_by
`

func (r *Repository) getNumber(ctx context.Context, sparepartID, numberID int64) (Number, error) {
	item, err := scanNumber(r.pool.QueryRow(ctx, `
		SELECT id, sparepart_id, part_number, COALESCE(brand, ''), number_type,
		       is_primary, COALESCE(notes, ''), created_at
		FROM sparepart_numbers
		WHERE id = $1 AND sparepart_id = $2
	`, numberID, sparepartID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Number{}, ErrNotFound
	}
	if err != nil {
		return Number{}, fmt.Errorf("get sparepart number: %w", err)
	}
	return item, nil
}

func (r *Repository) getAlternative(ctx context.Context, sparepartID, id int64) (Alternative, error) {
	item, err := scanAlternative(r.pool.QueryRow(ctx, alternativeSelect+`
		WHERE a.id = $1 AND a.sparepart_id = $2
	`, id, sparepartID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Alternative{}, ErrNotFound
	}
	if err != nil {
		return Alternative{}, fmt.Errorf("get sparepart alternative: %w", err)
	}
	item.Direction = "outgoing"
	return item, nil
}

func (r *Repository) getCompatibility(ctx context.Context, sparepartID, id int64) (Compatibility, error) {
	item, err := scanCompatibility(r.pool.QueryRow(ctx, compatibilitySelect+`
		WHERE sc.id = $1 AND sc.sparepart_id = $2
	`, id, sparepartID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Compatibility{}, ErrNotFound
	}
	if err != nil {
		return Compatibility{}, fmt.Errorf("get sparepart compatibility: %w", err)
	}
	return item, nil
}

func (r *Repository) requireSparepart(ctx context.Context, id int64) error {
	var exists int64
	if err := r.pool.QueryRow(ctx, `SELECT id FROM spareparts WHERE id = $1`, id).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("check sparepart: %w", err)
	}
	return nil
}

func (r *Repository) requireActiveReference(ctx context.Context, table string, id int64) error {
	if table != "unit_models" && table != "engine_models" {
		return ErrValidation
	}
	var exists int64
	if err := r.pool.QueryRow(ctx, `SELECT id FROM `+table+` WHERE id = $1 AND is_active = true`, id).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("check %s: %w", table, err)
	}
	return nil
}

func normalizeInput(input Input) (Input, error) {
	fields := map[string]string{}
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Category = strings.TrimSpace(input.Category)
	input.UOM = strings.TrimSpace(input.UOM)
	input.Description = strings.TrimSpace(input.Description)
	if input.Code == "" || len(input.Code) > 60 {
		fields["code"] = "Kode wajib 1-60 karakter."
	}
	if input.Name == "" || len(input.Name) > 200 {
		fields["name"] = "Nama wajib 1-200 karakter."
	}
	if len(input.Category) > 100 {
		fields["category"] = "Kategori maksimal 100 karakter."
	}
	if input.UOM == "" {
		input.UOM = "pcs"
	}
	if len(input.UOM) > 30 {
		fields["uom"] = "UOM maksimal 30 karakter."
	}
	if len(input.Description) > 2000 {
		fields["description"] = "Deskripsi maksimal 2000 karakter."
	}
	if len(fields) > 0 {
		return Input{}, &ValidationError{Fields: fields}
	}
	return input, nil
}

func normalizeNumber(input NumberInput) (NumberInput, error) {
	fields := map[string]string{}
	input.PartNumber = strings.TrimSpace(input.PartNumber)
	input.Brand = strings.TrimSpace(input.Brand)
	input.NumberType = strings.TrimSpace(input.NumberType)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.NumberType == "" {
		input.NumberType = "oem"
	}
	if input.PartNumber == "" || len(input.PartNumber) > 120 {
		fields["part_number"] = "Nomor part wajib 1-120 karakter."
	}
	if len(input.Brand) > 100 {
		fields["brand"] = "Brand maksimal 100 karakter."
	}
	if !validNumberType(input.NumberType) {
		fields["number_type"] = "Tipe nomor part tidak valid."
	}
	if len(input.Notes) > 500 {
		fields["notes"] = "Catatan maksimal 500 karakter."
	}
	if len(fields) > 0 {
		return NumberInput{}, &ValidationError{Fields: fields}
	}
	return input, nil
}

func scanSparepart(row interface{ Scan(...any) error }) (Sparepart, error) {
	var item Sparepart
	if err := row.Scan(&item.ID, &item.Code, &item.Name, &item.Category, &item.UOM,
		&item.Description, &item.IsActive, &item.PrimaryPartNumber,
		&item.CompatibilityCount, &item.ApprovedCompatibility, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Sparepart{}, err
	}
	item.Numbers = make([]Number, 0)
	item.Alternatives = make([]Alternative, 0)
	item.Compatibility = make([]Compatibility, 0)
	return item, nil
}

func scanNumber(row interface{ Scan(...any) error }) (Number, error) {
	var item Number
	if err := row.Scan(&item.ID, &item.SparepartID, &item.PartNumber, &item.Brand,
		&item.NumberType, &item.IsPrimary, &item.Notes, &item.CreatedAt); err != nil {
		return Number{}, err
	}
	return item, nil
}

func scanAlternative(row interface{ Scan(...any) error }) (Alternative, error) {
	var item Alternative
	if err := row.Scan(&item.ID, &item.SparepartID, &item.SparepartCode, &item.SparepartName,
		&item.AlternativeSparepartID, &item.AlternativeSparepartCode, &item.AlternativeSparepartName,
		&item.RelationType, &item.VerificationStatus, &item.Notes,
		&item.VerifiedBy, &item.VerifiedByName, &item.VerifiedAt,
		&item.CreatedAt, &item.UpdatedAt); err != nil {
		return Alternative{}, err
	}
	return item, nil
}

func scanCompatibility(row interface{ Scan(...any) error }) (Compatibility, error) {
	var item Compatibility
	if err := row.Scan(&item.ID, &item.SparepartID, &item.UnitModelID, &item.UnitModelName,
		&item.EngineModelID, &item.EngineModelName, &item.Configuration, &item.Status,
		&item.ConditionNote, &item.Source, &item.Notes, &item.VerifiedBy, &item.VerifiedByName,
		&item.VerifiedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Compatibility{}, err
	}
	return item, nil
}

func sparepartSortColumn(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "name":
		return "s.name"
	case "category":
		return "COALESCE(s.category, '')"
	case "uom":
		return "s.uom"
	case "is_active":
		return "s.is_active"
	case "updated_at":
		return "s.updated_at"
	default:
		return "s.code"
	}
}

func canManage(role string) bool {
	return role == "admin" || role == "planner"
}

func validNumberType(value string) bool {
	return value == "oem" || value == "aftermarket" || value == "superseded"
}

func validRelationType(value string) bool {
	return value == "equivalent" || value == "superseded" || value == "substitute"
}

func validVerificationStatus(value string) bool {
	return value == "pending_verification" || value == "approved_compatible" ||
		value == "conditional_compatible" || value == "not_compatible"
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
