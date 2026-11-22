package pm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PlanMaterial struct {
	ID            int64   `json:"id"`
	PlanID        int64   `json:"pm_plan_id"`
	SparepartID   int64   `json:"sparepart_id"`
	SparepartCode string  `json:"sparepart_code"`
	SparepartName string  `json:"sparepart_name"`
	UOM           string  `json:"uom"`
	Qty           float64 `json:"qty"`
	IsOptional    bool    `json:"is_optional"`
	Notes         string  `json:"notes"`
}

type PlanMaterialInput struct {
	SparepartID int64
	Qty         float64
	IsOptional  bool
	Notes       string
}

type PlanMaterialPatch struct {
	Qty        *float64
	IsOptional *bool
	Notes      *string
}

func (r *Repository) ListPlanMaterials(ctx context.Context, planID int64) ([]PlanMaterial, error) {
	if err := r.requirePlan(ctx, planID); err != nil {
		return nil, err
	}
	return queryPlanMaterials(ctx, r.pool, planID)
}

func (r *Repository) AddPlanMaterial(ctx context.Context, planID int64, actor Actor, input PlanMaterialInput) (PlanMaterial, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return PlanMaterial{}, ErrValidation
	}
	if err := r.requirePlan(ctx, planID); err != nil {
		return PlanMaterial{}, err
	}
	normalized, err := normalizePlanMaterial(input)
	if err != nil {
		return PlanMaterial{}, err
	}
	var id int64
	err = r.pool.QueryRow(ctx, `
		INSERT INTO pm_plan_materials (pm_plan_id, sparepart_id, qty, is_optional, notes)
		SELECT $1, s.id, $3, $4, NULLIF($5, '')
		FROM spareparts s
		WHERE s.id = $2 AND s.is_active = true
		RETURNING id
	`, planID, normalized.SparepartID, normalized.Qty, normalized.IsOptional,
		normalized.Notes).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return PlanMaterial{}, &ValidationError{Fields: map[string]string{"sparepart_id": "Sparepart tidak ditemukan atau tidak aktif."}}
	}
	if isUniqueViolation(err) {
		return PlanMaterial{}, ErrConflict
	}
	if err != nil {
		return PlanMaterial{}, fmt.Errorf("add PM plan material: %w", err)
	}
	return r.getPlanMaterial(ctx, planID, id)
}

func (r *Repository) UpdatePlanMaterial(ctx context.Context, planID, materialID int64, actor Actor, patch PlanMaterialPatch) (PlanMaterial, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return PlanMaterial{}, ErrValidation
	}
	current, err := r.getPlanMaterial(ctx, planID, materialID)
	if err != nil {
		return PlanMaterial{}, err
	}
	if patch.Qty != nil {
		current.Qty = *patch.Qty
	}
	if patch.IsOptional != nil {
		current.IsOptional = *patch.IsOptional
	}
	if patch.Notes != nil {
		current.Notes = strings.TrimSpace(*patch.Notes)
	}
	normalized, err := normalizePlanMaterial(PlanMaterialInput{
		SparepartID: current.SparepartID,
		Qty:         current.Qty,
		IsOptional:  current.IsOptional,
		Notes:       current.Notes,
	})
	if err != nil {
		return PlanMaterial{}, err
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE pm_plan_materials
		SET qty = $3, is_optional = $4, notes = NULLIF($5, '')
		WHERE id = $1 AND pm_plan_id = $2
	`, materialID, planID, normalized.Qty, normalized.IsOptional, normalized.Notes)
	if err != nil {
		return PlanMaterial{}, fmt.Errorf("update PM plan material: %w", err)
	}
	return r.getPlanMaterial(ctx, planID, materialID)
}

func (r *Repository) DeletePlanMaterial(ctx context.Context, planID, materialID int64, actor Actor) error {
	if actor.Role != "admin" && actor.Role != "planner" {
		return ErrValidation
	}
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM pm_plan_materials WHERE id = $1 AND pm_plan_id = $2
	`, materialID, planID)
	if err != nil {
		return fmt.Errorf("delete PM plan material: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) getPlanMaterial(ctx context.Context, planID, materialID int64) (PlanMaterial, error) {
	items, err := queryPlanMaterials(ctx, r.pool, planID)
	if err != nil {
		return PlanMaterial{}, err
	}
	for _, item := range items {
		if item.ID == materialID {
			return item, nil
		}
	}
	return PlanMaterial{}, ErrNotFound
}

type planMaterialQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func queryPlanMaterials(ctx context.Context, q planMaterialQuerier, planID int64) ([]PlanMaterial, error) {
	rows, err := q.Query(ctx, `
		SELECT m.id, m.pm_plan_id, m.sparepart_id, s.code, s.name, s.uom,
		       m.qty::float8, m.is_optional, COALESCE(m.notes, '')
		FROM pm_plan_materials m
		JOIN spareparts s ON s.id = m.sparepart_id
		WHERE m.pm_plan_id = $1
		ORDER BY s.code ASC
	`, planID)
	if err != nil {
		return nil, fmt.Errorf("list PM plan materials: %w", err)
	}
	defer rows.Close()
	items := make([]PlanMaterial, 0)
	for rows.Next() {
		var item PlanMaterial
		if err := rows.Scan(&item.ID, &item.PlanID, &item.SparepartID, &item.SparepartCode,
			&item.SparepartName, &item.UOM, &item.Qty, &item.IsOptional, &item.Notes); err != nil {
			return nil, fmt.Errorf("scan PM plan material: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM plan materials: %w", err)
	}
	return items, nil
}

func (r *Repository) requirePlan(ctx context.Context, planID int64) error {
	var exists int64
	if err := r.pool.QueryRow(ctx, `SELECT id FROM pm_plans WHERE id = $1`, planID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("check PM plan: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func normalizePlanMaterial(input PlanMaterialInput) (PlanMaterialInput, error) {
	fields := map[string]string{}
	input.Notes = strings.TrimSpace(input.Notes)
	if input.SparepartID <= 0 {
		fields["sparepart_id"] = "Sparepart wajib dipilih."
	}
	if input.Qty <= 0 {
		fields["qty"] = "Qty material harus lebih dari 0."
	}
	if len(input.Notes) > 500 {
		fields["notes"] = "Catatan maksimal 500 karakter."
	}
	if len(fields) > 0 {
		return PlanMaterialInput{}, &ValidationError{Fields: fields}
	}
	return input, nil
}
