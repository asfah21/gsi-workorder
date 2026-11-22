package units

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const maintenanceHistorySelect = `
	SELECT wo_id, unit_id, unit_code, wo_code, source, maintenance_type, status,
	       priority, title, description, p2h_report_id, candidate_id, pm_schedule_id,
	       assignee_id, assignee_name, supervisor_id, supervisor_name,
	       created_by, created_by_name, due_date, started_at, closed_at, created_at,
	       elapsed_minutes, checklist_total, checklist_done, checklist_not_ok,
	       findings_count, usage_count, usage_qty, material_status
	FROM v_unit_maintenance_history
`

const componentHistorySelect = `
	SELECT unit_id, unit_code, wo_id, wo_code, wo_title, maintenance_type, wo_status,
	       closed_at, component_id, component_code, component_name, is_fallback,
	       fallback_sparepart_id, sparepart_codes, sparepart_names, sparepart_count,
	       qty, used_at, hm, km, replacement_no, previous_used_at,
	       days_since_previous, hm_since_previous, km_since_previous
	FROM v_unit_component_history
`

func (r *PostgresRepository) ListMaintenanceHistory(ctx context.Context, unitID int64, filter HistoryFilter) ([]MaintenanceHistory, int, error) {
	where, args := historyWhere(unitID, filter.Status, filter.MaintenanceType, filter.From, filter.To)
	var total int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM v_unit_maintenance_history WHERE "+where,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count unit maintenance history: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	query := maintenanceHistorySelect + " WHERE " + where + `
		ORDER BY created_at DESC, wo_id DESC
		LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list unit maintenance history: %w", err)
	}
	defer rows.Close()

	items := make([]MaintenanceHistory, 0)
	for rows.Next() {
		var item MaintenanceHistory
		if err := rows.Scan(
			&item.WOID, &item.UnitID, &item.UnitCode, &item.WOCode, &item.Source,
			&item.MaintenanceType, &item.Status, &item.Priority, &item.Title, &item.Description,
			&item.P2HReportID, &item.CandidateID, &item.PMScheduleID,
			&item.AssigneeID, &item.AssigneeName, &item.SupervisorID, &item.SupervisorName,
			&item.CreatedBy, &item.CreatedByName, &item.DueDate, &item.StartedAt,
			&item.ClosedAt, &item.CreatedAt, &item.ElapsedMinutes,
			&item.ChecklistTotal, &item.ChecklistDone, &item.ChecklistNotOK,
			&item.FindingsCount, &item.UsageCount, &item.UsageQty, &item.MaterialStatus,
		); err != nil {
			return nil, 0, fmt.Errorf("scan unit maintenance history: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate unit maintenance history: %w", err)
	}
	return items, total, nil
}

func (r *PostgresRepository) ListComponentHistory(ctx context.Context, unitID int64, filter ComponentHistoryFilter) ([]ComponentHistory, int, error) {
	where, args := componentHistoryWhere(unitID, filter.From, filter.To)
	var total int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM v_unit_component_history WHERE "+where,
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count unit component history: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	query := componentHistorySelect + " WHERE " + where + `
		ORDER BY used_at DESC, wo_id DESC
		LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list unit component history: %w", err)
	}
	defer rows.Close()

	items := make([]ComponentHistory, 0)
	for rows.Next() {
		var item ComponentHistory
		if err := rows.Scan(
			&item.UnitID, &item.UnitCode, &item.WOID, &item.WOCode, &item.WOTitle,
			&item.MaintenanceType, &item.WOStatus, &item.ClosedAt,
			&item.ComponentID, &item.ComponentCode, &item.ComponentName, &item.IsFallback,
			&item.FallbackSparepartID, &item.SparepartCodes, &item.SparepartNames,
			&item.SparepartCount, &item.Qty, &item.UsedAt, &item.HM, &item.KM,
			&item.ReplacementNo, &item.PreviousUsedAt, &item.DaysSincePrevious,
			&item.HMSincePrevious, &item.KMSincePrevious,
		); err != nil {
			return nil, 0, fmt.Errorf("scan unit component history: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate unit component history: %w", err)
	}
	return items, total, nil
}

func (r *PostgresRepository) ListComponentIntervals(ctx context.Context, unitID int64) ([]ComponentInterval, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT unit_id, unit_code, component_id, fallback_sparepart_id,
		       component_code, component_name, is_fallback, replacement_count,
		       first_replaced_at, last_replaced_at, avg_interval_days,
		       avg_interval_hm, avg_interval_km, total_qty, last_hm, last_km
		FROM v_component_replacement_frequency
		WHERE unit_id = $1
		ORDER BY last_replaced_at DESC, component_name ASC
		LIMIT 500
	`, unitID)
	if err != nil {
		return nil, fmt.Errorf("list component intervals: %w", err)
	}
	defer rows.Close()

	items := make([]ComponentInterval, 0)
	for rows.Next() {
		var item ComponentInterval
		if err := rows.Scan(
			&item.UnitID, &item.UnitCode, &item.ComponentID, &item.FallbackSparepartID,
			&item.ComponentCode, &item.ComponentName, &item.IsFallback,
			&item.ReplacementCount, &item.FirstReplacedAt, &item.LastReplacedAt,
			&item.AvgIntervalDays, &item.AvgIntervalHM, &item.AvgIntervalKM,
			&item.TotalQty, &item.LastHM, &item.LastKM,
		); err != nil {
			return nil, fmt.Errorf("scan component interval: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate component intervals: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) ListUnitDowntime(ctx context.Context, unitID int64, filter DowntimeFilter) ([]DowntimeEvent, error) {
	where := []string{"unit_id = $1"}
	args := []any{unitID}
	if filter.From != nil {
		args = append(args, *filter.From)
		where = append(where, fmt.Sprintf("downtime_end > $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		where = append(where, fmt.Sprintf("(downtime_start IS NULL OR downtime_start < $%d)", len(args)))
	}
	rows, err := r.pool.Query(ctx, `
		SELECT wo_id, unit_id, unit_code, wo_code, title, maintenance_type,
		       COALESCE(priority, ''), assignee_id, assignee_name, supervisor_id,
		       supervisor_name, started_at, downtime_start, downtime_end,
		       downtime_class, downtime_minutes, waiting_minutes
		FROM v_unit_downtime
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY downtime_end DESC, wo_id DESC
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list unit downtime: %w", err)
	}
	defer rows.Close()

	events := make([]DowntimeEvent, 0)
	for rows.Next() {
		var event DowntimeEvent
		if err := rows.Scan(
			&event.WOID, &event.UnitID, &event.UnitCode, &event.WOCode, &event.Title,
			&event.MaintenanceType, &event.Priority, &event.AssigneeID, &event.AssigneeName,
			&event.SupervisorID, &event.SupervisorName, &event.StartedAt,
			&event.DowntimeStart, &event.DowntimeEnd, &event.DowntimeClass,
			&event.DowntimeMinutes, &event.WaitingMinutes,
		); err != nil {
			return nil, fmt.Errorf("scan unit downtime: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unit downtime: %w", err)
	}
	return events, nil
}

func (r *PostgresRepository) ListComponents(ctx context.Context, activeOnly bool) ([]Component, error) {
	where := ""
	if activeOnly {
		where = " WHERE is_active = true"
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, COALESCE(category, ''), is_active
		FROM components`+where+`
		ORDER BY code ASC
		LIMIT 500
	`)
	if err != nil {
		return nil, fmt.Errorf("list components: %w", err)
	}
	defer rows.Close()

	items := make([]Component, 0)
	for rows.Next() {
		var item Component
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Category, &item.IsActive); err != nil {
			return nil, fmt.Errorf("scan component: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate components: %w", err)
	}
	return items, nil
}

func historyWhere(unitID int64, status, maintenanceType string, from, to *time.Time) (string, []any) {
	where := []string{"unit_id = $1"}
	args := []any{unitID}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if maintenanceType != "" {
		args = append(args, maintenanceType)
		where = append(where, fmt.Sprintf("maintenance_type = $%d", len(args)))
	}
	if from != nil {
		args = append(args, *from)
		where = append(where, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if to != nil {
		args = append(args, *to)
		where = append(where, fmt.Sprintf("created_at < $%d", len(args)))
	}
	return strings.Join(where, " AND "), args
}

func componentHistoryWhere(unitID int64, from, to *time.Time) (string, []any) {
	where := []string{"unit_id = $1"}
	args := []any{unitID}
	if from != nil {
		args = append(args, *from)
		where = append(where, fmt.Sprintf("used_at >= $%d", len(args)))
	}
	if to != nil {
		args = append(args, *to)
		where = append(where, fmt.Sprintf("used_at < $%d", len(args)))
	}
	return strings.Join(where, " AND "), args
}
