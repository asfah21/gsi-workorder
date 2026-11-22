package reporting

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository reads reporting data exclusively through views and aggregate
// queries. It never writes and never calls an external service.
type Repository struct {
	pool *pgxpool.Pool
	loc  *time.Location
}

func NewRepository(pool *pgxpool.Pool, loc *time.Location) *Repository {
	if loc == nil {
		loc = time.UTC
	}
	return &Repository{pool: pool, loc: loc}
}

type queryArgs struct {
	args  []any
	where []string
}

func newQueryArgs(loc *time.Location) *queryArgs {
	return &queryArgs{args: []any{loc.String()}}
}

// addFilter appends period, unit, model, and status predicates. Column names
// are internal constants chosen by the caller, never user input.
func (q *queryArgs) addFilter(filter Filter, timeColumn, unitColumn, modelColumn, statusColumn string) {
	if filter.From != nil && timeColumn != "" {
		q.args = append(q.args, *filter.From)
		q.where = append(q.where, fmt.Sprintf("%s >= $%d", timeColumn, len(q.args)))
	}
	if filter.To != nil && timeColumn != "" {
		q.args = append(q.args, *filter.To)
		q.where = append(q.where, fmt.Sprintf("%s < $%d", timeColumn, len(q.args)))
	}
	if filter.UnitID != nil && unitColumn != "" {
		q.args = append(q.args, *filter.UnitID)
		q.where = append(q.where, fmt.Sprintf("%s = $%d", unitColumn, len(q.args)))
	}
	if filter.UnitModelID != nil && modelColumn != "" {
		q.args = append(q.args, *filter.UnitModelID)
		q.where = append(q.where, fmt.Sprintf("%s = $%d", modelColumn, len(q.args)))
	}
	if filter.Status != "" && statusColumn != "" {
		q.args = append(q.args, filter.Status)
		q.where = append(q.where, fmt.Sprintf("%s = $%d", statusColumn, len(q.args)))
	}
}

func (q *queryArgs) addLocation(filter Filter, columns ...string) {
	if filter.LocationID == nil || len(columns) == 0 {
		return
	}
	q.args = append(q.args, *filter.LocationID)
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, fmt.Sprintf("%s = $%d", column, len(q.args)))
	}
	q.where = append(q.where, "("+joinOr(parts)+")")
}

func (q *queryArgs) whereSQL() string {
	if len(q.where) == 0 {
		return ""
	}
	return " WHERE " + joinAnd(q.where)
}

func (q *queryArgs) paginate(filter Filter) (string, []any) {
	args := append([]any{}, q.args...)
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	clause := fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	return clause, args
}

func joinOr(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " OR "
		}
		result += part
	}
	return result
}

func joinAnd(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " AND "
		}
		result += part
	}
	return result
}

var sqlParamPattern = regexp.MustCompile(`\$(\d+)`)

// compactParams renumbers bound parameters so every statement only declares
// placeholders it actually uses. PostgreSQL rejects an untyped parameter that
// is bound but absent from the SQL text, which happens when a count query
// omits the timezone argument used by a bucket expression.
func compactParams(sql string, args []any) (string, []any) {
	used := map[int]bool{}
	for _, match := range sqlParamPattern.FindAllStringSubmatch(sql, -1) {
		value, err := strconv.Atoi(match[1])
		if err == nil {
			used[value] = true
		}
	}
	indices := make([]int, 0, len(used))
	for index := range used {
		indices = append(indices, index)
	}
	sort.Ints(indices)

	remap := make(map[int]int, len(indices))
	compact := make([]any, 0, len(indices))
	for newIndex, oldIndex := range indices {
		remap[oldIndex] = newIndex + 1
		if oldIndex <= len(args) {
			compact = append(compact, args[oldIndex-1])
		}
	}
	rewritten := sqlParamPattern.ReplaceAllStringFunc(sql, func(match string) string {
		oldIndex, err := strconv.Atoi(match[1:])
		if err != nil {
			return match
		}
		return "$" + strconv.Itoa(remap[oldIndex])
	})
	return rewritten, compact
}

func (r *Repository) query(ctx context.Context, sql string, args ...any) ([]map[string]any, error) {
	sql, args = compactParams(sql, args)
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	result := make([]map[string]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make(map[string]any, len(fields))
		for i, field := range fields {
			row[string(field.Name)] = values[i]
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *Repository) summaryRow(ctx context.Context, sql string, args ...any) (map[string]any, error) {
	rows, err := r.query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return map[string]any{}, nil
	}
	return rows[0], nil
}

func (r *Repository) tableWithCount(ctx context.Context, columns []Column, filter Filter, selectSQL, countSQL, orderSQL string, args []any) (Table, error) {
	countSQL, countArgs := compactParams(countSQL, args)
	var total int
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return Table{}, err
	}
	limitClause, pageArgs := paginateArgs(args, filter)
	rows, err := r.query(ctx, selectSQL+orderSQL+limitClause, pageArgs...)
	if err != nil {
		return Table{}, err
	}
	return Table{
		Columns: columns,
		Rows:    rows,
		Meta:    Pagination{Page: filter.Page, Limit: filter.Limit, Total: total},
	}, nil
}

func paginateArgs(args []any, filter Filter) (string, []any) {
	result := append([]any{}, args...)
	result = append(result, filter.Limit, (filter.Page-1)*filter.Limit)
	return fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(result)-1, len(result)), result
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	default:
		return 0
	}
}

// Summary extends the M1 dashboard with live PM, material, stock, and downtime
// aggregates. Every value is computed from transaction tables or read-only views.
func (r *Repository) Summary(ctx context.Context) (map[string]any, error) {
	sql := `
		SELECT
			(SELECT count(*) FROM units)::int AS units_total,
			(SELECT count(*) FROM units WHERE is_active)::int AS units_active,
			(SELECT count(*) FROM units WHERE is_active AND is_breakdown)::int AS units_breakdown,
			(SELECT count(*) FROM unit_models WHERE is_active)::int AS unit_models,
			(SELECT count(*) FROM engine_models WHERE is_active)::int AS engine_models,
			(SELECT count(*) FROM unit_meter_readings WHERE recorded_at >= now() - interval '30 days')::int AS readings_last_30_days,
			(SELECT count(*) FROM v_pm_due WHERE status = 'overdue')::int AS pm_overdue,
			(SELECT count(*) FROM v_pm_due WHERE status = 'due')::int AS pm_due,
			(SELECT count(*) FROM v_pm_due d, app_settings s
				WHERE s.key = 'pm_due_soon_days'
				  AND d.due_date IS NOT NULL
				  AND d.due_date > (now() AT TIME ZONE $1::text)::date
				  AND d.due_date <= (now() AT TIME ZONE $1::text)::date + s.value::int
				  AND d.status NOT IN ('completed', 'cancelled', 'skipped'))::int AS pm_due_soon,
			(SELECT count(*) FROM material_requirements WHERE status = 'shortage')::int AS material_shortage,
			(SELECT count(*) FROM v_low_stock WHERE stock_status = 'low')::int AS stock_low,
			(SELECT count(*) FROM v_low_stock WHERE stock_status = 'critical')::int AS stock_critical,
			(SELECT count(*) FROM work_orders WHERE status = 'waiting_part')::int AS waiting_part_units,
			(SELECT count(*) FROM work_orders WHERE status = 'open')::int AS wo_open,
			(SELECT count(*) FROM work_orders WHERE status = 'in_progress')::int AS wo_in_progress,
			(SELECT count(*) FROM work_orders WHERE maintenance_type = 'breakdown' AND status NOT IN ('done', 'cancelled'))::int AS breakdown_open,
			(SELECT COALESCE(sum(downtime_minutes), 0)::float8 FROM v_unit_downtime
				WHERE downtime_end >= now() - interval '30 days') AS downtime_minutes_30d,
			(SELECT COALESCE(avg(downtime_minutes), 0)::float8 FROM v_unit_downtime
				WHERE downtime_end >= now() - interval '30 days' AND downtime_class = 'unplanned') AS mttr_minutes_30d
		`
	return r.summaryRow(ctx, sql, r.loc.String())
}

var pmColumns = []Column{
	{Key: "schedule_id", Label: "ID jadwal"},
	{Key: "plan_name", Label: "Rencana PM"},
	{Key: "unit_code", Label: "Unit"},
	{Key: "unit_model_name", Label: "Model"},
	{Key: "cycle_no", Label: "Siklus"},
	{Key: "status", Label: "Status"},
	{Key: "bucket", Label: "Bucket"},
	{Key: "due_date", Label: "Jatuh tempo"},
	{Key: "days_to_due", Label: "Hari menuju due"},
	{Key: "work_order_code", Label: "Work order"},
}

const pmBucketExpr = `
	CASE
		WHEN d.status IN ('completed', 'cancelled', 'skipped') THEN d.status
		WHEN d.status = 'overdue' THEN 'overdue'
		WHEN d.status = 'wo_created' THEN 'wo_created'
		WHEN d.due_date IS NULL THEN 'scheduled'
		WHEN d.due_date <= (now() AT TIME ZONE $1::text)::date THEN 'due'
		WHEN d.due_date <= (now() AT TIME ZONE $1::text)::date + COALESCE(soon.value::int, 7) THEN 'due soon'
		WHEN d.due_date <= (now() AT TIME ZONE $1::text)::date + COALESCE(upcoming.value::int, 30) THEN 'upcoming'
		ELSE 'scheduled'
	END
`

func (r *Repository) PM(ctx context.Context, filter Filter) (DashboardPM, error) {
	q := newQueryArgs(r.loc)
	q.addFilter(filter, "d.created_at", "d.unit_id", "d.unit_model_id", "")
	base := `
		FROM v_pm_due d
		LEFT JOIN app_settings soon ON soon.key = 'pm_due_soon_days'
		LEFT JOIN app_settings upcoming ON upcoming.key = 'pm_upcoming_days'
	` + q.whereSQL()
	if filter.Status != "" {
		args := append([]any{}, q.args...)
		base = `
		FROM v_pm_due d
		LEFT JOIN app_settings soon ON soon.key = 'pm_due_soon_days'
		LEFT JOIN app_settings upcoming ON upcoming.key = 'pm_upcoming_days'
		` + q.whereSQL() + ` AND ` + pmBucketExpr + ` = $` + fmt.Sprint(len(args)+1)
		q.args = append(q.args, filter.Status)
	}

	summarySQL := `SELECT ` + pmBucketExpr + ` AS bucket, count(*)::int AS count ` + base + ` GROUP BY 1`
	summaryRows, err := r.query(ctx, summarySQL, q.args...)
	if err != nil {
		return DashboardPM{}, fmt.Errorf("pm dashboard summary: %w", err)
	}
	summary := map[string]any{
		"total": 0, "upcoming": 0, "due_soon": 0, "due": 0,
		"overdue": 0, "wo_created": 0, "scheduled": 0,
	}
	for _, row := range summaryRows {
		bucket, _ := row["bucket"].(string)
		count := asInt(row["count"])
		summary[bucket] = count
		summary["total"] = asInt(summary["total"]) + count
	}

	selectSQL := `
		SELECT d.schedule_id, d.plan_name, d.unit_code,
		       COALESCE(d.unit_model_name, '') AS unit_model_name,
		       d.cycle_no, d.status, ` + pmBucketExpr + ` AS bucket,
		       d.due_date, d.due_hm::float8 AS due_hm, d.due_km::float8 AS due_km,
		       (d.due_date - (now() AT TIME ZONE $1::text)::date) AS days_to_due,
		       d.work_order_code
	` + base
	orderSQL := ` ORDER BY d.due_date NULLS LAST, d.schedule_id DESC`
	countSQL := `SELECT count(*) ` + base
	table, err := r.tableWithCount(ctx, pmColumns, filter, selectSQL, countSQL, orderSQL, q.args)
	if err != nil {
		return DashboardPM{}, fmt.Errorf("pm dashboard due: %w", err)
	}
	return DashboardPM{Summary: summary, Due: table}, nil
}

var materialColumns = []Column{
	{Key: "requirement_code", Label: "Kode material"},
	{Key: "wo_code", Label: "Work order"},
	{Key: "unit_code", Label: "Unit"},
	{Key: "derived_status", Label: "Status"},
	{Key: "shortage_count", Label: "Item shortage"},
	{Key: "shortage_qty", Label: "Qty shortage"},
	{Key: "waiting_minutes", Label: "Menunggu (menit)"},
}

const materialBase = `
	FROM v_material_readiness r
	JOIN work_orders wo ON wo.id = r.wo_id
	LEFT JOIN units u ON u.id = wo.unit_id
`

func (r *Repository) Material(ctx context.Context, filter Filter) (DashboardMaterial, error) {
	q := newQueryArgs(r.loc)
	q.addFilter(filter, "wo.created_at", "wo.unit_id", "u.unit_model_id", "r.derived_status")
	where := q.whereSQL()
	base := materialBase + where

	summarySQL := `
		SELECT
			count(DISTINCT r.material_requirement_id)::int AS total,
			count(DISTINCT r.material_requirement_id) FILTER (WHERE r.derived_status = 'ready')::int AS ready,
			count(DISTINCT r.material_requirement_id) FILTER (WHERE r.derived_status = 'partially_ready')::int AS partially_ready,
			count(DISTINCT r.material_requirement_id) FILTER (WHERE r.derived_status = 'shortage')::int AS shortage,
			count(DISTINCT r.material_requirement_id) FILTER (WHERE r.derived_status = 'reserved')::int AS reserved,
			count(DISTINCT r.material_requirement_id) FILTER (WHERE r.derived_status = 'issued')::int AS issued,
			count(DISTINCT r.material_requirement_id) FILTER (WHERE r.derived_status = 'cancelled')::int AS cancelled,
			COALESCE(sum(r.shortage) FILTER (WHERE r.derived_status = 'shortage'), 0)::float8 AS shortage_qty
	` + base
	summary, err := r.summaryRow(ctx, summarySQL, q.args...)
	if err != nil {
		return DashboardMaterial{}, fmt.Errorf("material dashboard summary: %w", err)
	}
	var waitingPart int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM work_orders WHERE status = 'waiting_part'`).Scan(&waitingPart); err != nil {
		return DashboardMaterial{}, fmt.Errorf("material dashboard waiting part: %w", err)
	}
	summary["waiting_part"] = waitingPart

	selectSQL := `
		SELECT r.material_requirement_id, r.requirement_code, wo.code AS wo_code,
		       COALESCE(u.code, '') AS unit_code, r.derived_status,
		       count(*) FILTER (WHERE r.shortage > 0 AND NOT r.is_optional)::int AS shortage_count,
		       COALESCE(sum(r.shortage) FILTER (WHERE r.shortage > 0 AND NOT r.is_optional), 0)::float8 AS shortage_qty,
		       0::float8 AS waiting_minutes
	` + base + ` GROUP BY r.material_requirement_id, r.requirement_code, wo.code, u.code, r.derived_status`
	countSQL := `SELECT count(DISTINCT r.material_requirement_id) ` + base
	requirements, err := r.tableWithCount(ctx, materialColumns, filter, selectSQL, countSQL,
		` ORDER BY r.material_requirement_id DESC`, q.args)
	if err != nil {
		return DashboardMaterial{}, fmt.Errorf("material dashboard requirements: %w", err)
	}

	agingColumns := []Column{
		{Key: "age_bucket", Label: "Umur"},
		{Key: "items", Label: "Item shortage"},
	}
	agingRows, err := r.query(ctx, `
		SELECT age_bucket, count(*)::int AS items
		FROM v_shortage_aging
		GROUP BY age_bucket
		ORDER BY CASE age_bucket WHEN '0-3' THEN 1 WHEN '4-7' THEN 2 WHEN '8-14' THEN 3 ELSE 4 END
	`)
	if err != nil {
		return DashboardMaterial{}, fmt.Errorf("material dashboard aging: %w", err)
	}
	aging := Table{
		Columns: agingColumns,
		Rows:    agingRows,
		Meta:    Pagination{Page: 1, Limit: len(agingRows), Total: len(agingRows)},
	}
	return DashboardMaterial{Summary: summary, Requirements: requirements, Aging: aging}, nil
}

var lowStockColumns = []Column{
	{Key: "sparepart_code", Label: "Kode sparepart"},
	{Key: "sparepart_name", Label: "Nama sparepart"},
	{Key: "warehouse_code", Label: "Gudang"},
	{Key: "location_code", Label: "Lokasi"},
	{Key: "on_hand", Label: "On hand"},
	{Key: "reserved", Label: "Reserved"},
	{Key: "available", Label: "Available"},
	{Key: "min_qty", Label: "Min"},
	{Key: "critical_qty", Label: "Critical"},
	{Key: "stock_status", Label: "Status"},
}

var movementColumns = []Column{
	{Key: "transaction_code", Label: "Kode transaksi"},
	{Key: "type", Label: "Tipe"},
	{Key: "status", Label: "Status"},
	{Key: "sparepart_code", Label: "Kode sparepart"},
	{Key: "sparepart_name", Label: "Nama sparepart"},
	{Key: "location", Label: "Lokasi"},
	{Key: "qty", Label: "Qty"},
	{Key: "posted_at", Label: "Waktu posting"},
	{Key: "bucket", Label: "Bucket"},
	{Key: "reference_no", Label: "Referensi"},
}

func (r *Repository) Inventory(ctx context.Context, filter Filter) (DashboardInventory, error) {
	lowQ := newQueryArgs(r.loc)
	lowQ.addLocation(filter, "warehouse_location_id")
	lowWhere := lowQ.whereSQL()
	lowBase := ` FROM v_low_stock` + lowWhere
	lowSummarySQL := `
		SELECT count(*)::int AS total,
		       count(*) FILTER (WHERE stock_status = 'low')::int AS low,
		       count(*) FILTER (WHERE stock_status = 'critical')::int AS critical
		` + lowBase
	summary, err := r.summaryRow(ctx, lowSummarySQL, lowQ.args...)
	if err != nil {
		return DashboardInventory{}, fmt.Errorf("inventory dashboard summary: %w", err)
	}
	movementSummary, err := r.summaryRow(ctx, `
		SELECT
			count(*)::int AS movements,
			COALESCE(sum(qty) FILTER (WHERE type = 'receipt' AND status = 'posted'), 0)::float8 AS receipt_qty,
			COALESCE(sum(qty) FILTER (WHERE type = 'issue' AND status = 'posted'), 0)::float8 AS issue_qty
		FROM v_inventory_movement
		WHERE ($1::timestamptz IS NULL OR posted_at >= $1)
		  AND ($2::timestamptz IS NULL OR posted_at < $2)
	`, nullableTime(filter.From), nullableTime(filter.To))
	if err != nil {
		return DashboardInventory{}, fmt.Errorf("inventory movement summary: %w", err)
	}
	for key, value := range movementSummary {
		summary[key] = value
	}

	lowSelect := `
		SELECT sparepart_code, sparepart_name, warehouse_code, location_code,
		       on_hand::float8, reserved::float8, available::float8,
		       min_qty::float8, critical_qty::float8, stock_status
	` + lowBase
	lowTable, err := r.tableWithCount(ctx, lowStockColumns, filter, lowSelect,
		`SELECT count(*) `+lowBase, ` ORDER BY stock_status, sparepart_code`, lowQ.args)
	if err != nil {
		return DashboardInventory{}, fmt.Errorf("inventory dashboard low stock: %w", err)
	}

	bucketExpr, err := bucketSQL(filter.Bucket, "posted_at", 1)
	if err != nil {
		return DashboardInventory{}, err
	}
	moveQ := newQueryArgs(r.loc)
	moveQ.addFilter(filter, "posted_at", "", "", "status")
	moveQ.addLocation(filter, "from_location_id", "to_location_id")
	moveWhere := moveQ.whereSQL()
	moveSelect := `
		SELECT transaction_code, type, status, sparepart_code, sparepart_name,
		       COALESCE(NULLIF(from_location_code, ''), '') AS from_location,
		       COALESCE(NULLIF(to_location_code, ''), from_location_code, '') AS to_location,
		       '' AS location,
		       qty::float8, posted_at, ` + bucketExpr + ` AS bucket, COALESCE(reference_no, '') AS reference_no
		FROM v_inventory_movement` + moveWhere
	movementColumnsLocal := append([]Column{}, movementColumns...)
	// "location" is a computed display column; keep the schema stable.
	moveTable, err := r.tableWithCount(ctx, movementColumnsLocal, filter, moveSelect,
		`SELECT count(*) FROM v_inventory_movement`+moveWhere, ` ORDER BY posted_at DESC, transaction_id DESC`, moveQ.args)
	if err != nil {
		return DashboardInventory{}, fmt.Errorf("inventory dashboard movements: %w", err)
	}
	return DashboardInventory{Summary: summary, LowStock: lowTable, Movements: moveTable}, nil
}

var activityColumns = []Column{
	{Key: "bucket", Label: "Bucket"},
	{Key: "status", Label: "Status"},
	{Key: "maintenance_type", Label: "Tipe"},
	{Key: "wo_count", Label: "Jumlah WO"},
	{Key: "completed", Label: "Selesai"},
	{Key: "elapsed_minutes", Label: "Durasi (menit)"},
}

var downtimeColumns = []Column{
	{Key: "bucket", Label: "Bucket"},
	{Key: "downtime_class", Label: "Kelas"},
	{Key: "events", Label: "Event"},
	{Key: "downtime_minutes", Label: "Downtime (menit)"},
	{Key: "waiting_minutes", Label: "Waiting part (menit)"},
	{Key: "mttr_minutes", Label: "MTTR (menit)"},
}

var waitingColumns = []Column{
	{Key: "wo_code", Label: "Work order"},
	{Key: "unit_code", Label: "Unit"},
	{Key: "title", Label: "Judul"},
	{Key: "material_status", Label: "Status material"},
	{Key: "waiting_since", Label: "Sejak"},
	{Key: "waiting_minutes", Label: "Menunggu (menit)"},
}

func (r *Repository) Maintenance(ctx context.Context, filter Filter) (DashboardMaintenance, error) {
	summary, err := r.summaryRow(ctx, `
		SELECT
			count(*)::int AS total,
			count(*) FILTER (WHERE status = 'open')::int AS open,
			count(*) FILTER (WHERE status = 'in_progress')::int AS in_progress,
			count(*) FILTER (WHERE status = 'waiting_part')::int AS waiting_part,
			count(*) FILTER (WHERE status = 'done')::int AS done,
			count(*) FILTER (WHERE status = 'cancelled')::int AS cancelled,
			count(*) FILTER (WHERE status NOT IN ('done', 'cancelled'))::int AS backlog,
			count(*) FILTER (WHERE maintenance_type = 'breakdown')::int AS breakdown,
			count(*) FILTER (WHERE maintenance_type = 'preventive')::int AS preventive,
			count(*) FILTER (WHERE maintenance_type = 'inspection')::int AS inspection,
			count(*) FILTER (WHERE maintenance_type = 'corrective')::int AS corrective
		FROM work_orders
		WHERE ($1::timestamptz IS NULL OR created_at >= $1)
		  AND ($2::timestamptz IS NULL OR created_at < $2)
		  AND ($3::bigint IS NULL OR unit_id = $3)
	`, nullableTime(filter.From), nullableTime(filter.To), nullableID(filter.UnitID))
	if err != nil {
		return DashboardMaintenance{}, fmt.Errorf("maintenance dashboard summary: %w", err)
	}

	activityBucket, err := bucketSQL(filter.Bucket, "created_at", 1)
	if err != nil {
		return DashboardMaintenance{}, err
	}
	actQ := newQueryArgs(r.loc)
	actQ.addFilter(filter, "created_at", "unit_id", "", "status")
	actWhere := actQ.whereSQL()
	actSelect := `
		SELECT ` + activityBucket + ` AS bucket, status, maintenance_type,
		       count(*)::int AS wo_count,
		       count(*) FILTER (WHERE closed_at IS NOT NULL)::int AS completed,
		       COALESCE(sum(GREATEST(0, EXTRACT(EPOCH FROM
		           (closed_at - COALESCE(started_at, created_at))) / 60)), 0)::float8 AS elapsed_minutes
		FROM v_unit_maintenance_history` + actWhere
	activity, err := r.tableWithCount(ctx, activityColumns, filter, actSelect,
		`SELECT count(*) FROM v_unit_maintenance_history`+actWhere,
		` GROUP BY 1, 2, 3 ORDER BY 1 DESC, 2, 3`, actQ.args)
	if err != nil {
		return DashboardMaintenance{}, fmt.Errorf("maintenance dashboard activity: %w", err)
	}

	downBucket, err := bucketSQL(filter.Bucket, "downtime_end", 1)
	if err != nil {
		return DashboardMaintenance{}, err
	}
	downQ := newQueryArgs(r.loc)
	downQ.addFilter(filter, "downtime_end", "d.unit_id", "u.unit_model_id", "d.downtime_class")
	downFrom := downQ.whereSQL()
	downBase := ` FROM v_unit_downtime d LEFT JOIN units u ON u.id = d.unit_id` + downFrom
	downSummarySQL := `
		SELECT COALESCE(sum(downtime_minutes), 0)::float8 AS total_minutes,
		       COALESCE(sum(downtime_minutes) FILTER (WHERE downtime_class = 'planned'), 0)::float8 AS planned_minutes,
		       COALESCE(sum(downtime_minutes) FILTER (WHERE downtime_class = 'unplanned'), 0)::float8 AS unplanned_minutes,
		       COALESCE(sum(waiting_minutes), 0)::float8 AS waiting_minutes,
		       COALESCE(avg(downtime_minutes) FILTER (WHERE downtime_class = 'unplanned'), 0)::float8 AS mttr_minutes,
		       count(*)::int AS events
	` + downBase
	downSummary, err := r.summaryRow(ctx, downSummarySQL, downQ.args...)
	if err != nil {
		return DashboardMaintenance{}, fmt.Errorf("maintenance downtime summary: %w", err)
	}
	for key, value := range downSummary {
		summary[key] = value
	}
	downSelect := `
		SELECT ` + downBucket + ` AS bucket, d.downtime_class, count(*)::int AS events,
		       COALESCE(sum(downtime_minutes), 0)::float8 AS downtime_minutes,
		       COALESCE(sum(waiting_minutes), 0)::float8 AS waiting_minutes,
		       COALESCE(avg(downtime_minutes), 0)::float8 AS mttr_minutes
	` + downBase
	downtime, err := r.tableWithCount(ctx, downtimeColumns, filter, downSelect,
		`SELECT count(*)`+downBase, ` GROUP BY 1, 2 ORDER BY 1 DESC, 2`, downQ.args)
	if err != nil {
		return DashboardMaintenance{}, fmt.Errorf("maintenance dashboard downtime: %w", err)
	}

	waitQ := newQueryArgs(r.loc)
	waitQ.addFilter(filter, "", "unit_id", "", "material_status")
	waitWhere := waitQ.whereSQL()
	waitBase := ` FROM v_waiting_part_units` + waitWhere
	waitSummary, err := r.summaryRow(ctx, `SELECT count(*)::int AS units_waiting, COALESCE(sum(waiting_minutes), 0)::float8 AS waiting_minutes_total `+waitBase, waitQ.args...)
	if err != nil {
		return DashboardMaintenance{}, fmt.Errorf("maintenance waiting summary: %w", err)
	}
	for key, value := range waitSummary {
		summary[key] = value
	}
	waitSelect := `
		SELECT wo_code, unit_code, title, material_status, waiting_since, waiting_minutes::float8
	` + waitBase
	waiting, err := r.tableWithCount(ctx, waitingColumns, filter, waitSelect,
		`SELECT count(*)`+waitBase, ` ORDER BY waiting_minutes DESC, wo_id DESC`, waitQ.args)
	if err != nil {
		return DashboardMaintenance{}, fmt.Errorf("maintenance dashboard waiting: %w", err)
	}
	return DashboardMaintenance{Summary: summary, Activity: activity, Downtime: downtime, Waiting: waiting}, nil
}

// Report dispatches to one of the five read-only report queries. The name is
// checked against an allowlist maintained by the HTTP layer.
func (r *Repository) Report(ctx context.Context, name string, filter Filter) (Table, error) {
	switch name {
	case "pm-compliance":
		return r.pmCompliance(ctx, filter)
	case "shortage-aging":
		return r.shortageAging(ctx, filter)
	case "sparepart-consumption":
		return r.sparepartConsumption(ctx, filter)
	case "inventory-movement":
		return r.inventoryMovement(ctx, filter)
	case "component-replacement":
		return r.componentReplacement(ctx, filter)
	default:
		return Table{}, ErrInvalidFilter
	}
}

func groupColumn(groupBy string) string {
	switch groupBy {
	case "unit":
		return "unit_code"
	case "model":
		return "unit_model_name"
	default:
		return "plan_name"
	}
}

func (r *Repository) pmCompliance(ctx context.Context, filter Filter) (Table, error) {
	group := groupColumn(filter.GroupBy)
	q := newQueryArgs(r.loc)
	q.addFilter(filter, "created_at", "unit_id", "unit_model_id", "")
	bucketExpr, err := bucketSQL(filter.Bucket, "created_at", 1)
	if err != nil {
		return Table{}, err
	}
	where := q.whereSQL()
	selectSQL := `
		SELECT ` + group + ` AS target, ` + bucketExpr + ` AS bucket,
		       count(*)::int AS total,
		       count(*) FILTER (WHERE is_completed)::int AS completed,
		       count(*) FILTER (WHERE is_on_time)::int AS on_time,
		       count(*) FILTER (WHERE is_overdue)::int AS overdue,
		       count(*) FILTER (WHERE is_missed)::int AS missed,
		       round(100.0 * count(*) FILTER (WHERE is_completed) / NULLIF(count(*), 0), 1)::float8 AS completion_rate,
		       round(100.0 * count(*) FILTER (WHERE is_on_time) / NULLIF(count(*) FILTER (WHERE is_completed), 0), 1)::float8 AS on_time_rate
		FROM v_pm_compliance` + where + `
		GROUP BY 1, 2`
	columns := []Column{
		{Key: "bucket", Label: "Bucket"},
		{Key: "target", Label: "Target"},
		{Key: "total", Label: "Total"},
		{Key: "completed", Label: "Selesai"},
		{Key: "on_time", Label: "Tepat waktu"},
		{Key: "overdue", Label: "Terlambat"},
		{Key: "missed", Label: "Dilewati"},
		{Key: "completion_rate", Label: "% selesai"},
		{Key: "on_time_rate", Label: "% tepat waktu"},
	}
	return r.tableWithCount(ctx, columns, filter, selectSQL,
		`SELECT count(*) FROM (SELECT 1 FROM v_pm_compliance`+where+` GROUP BY `+group+`, `+bucketExpr+`) x`,
		` ORDER BY 1 DESC, 2`, q.args)
}

func (r *Repository) shortageAging(ctx context.Context, filter Filter) (Table, error) {
	q := newQueryArgs(r.loc)
	q.addFilter(filter, "shortage_since", "unit_id", "", "age_bucket")
	where := q.whereSQL()
	selectSQL := `
		SELECT requirement_code, wo_code, unit_code, sparepart_code, sparepart_name,
		       qty_required::float8, shortage::float8, age_days::int, age_bucket,
		       shortage_since
		FROM v_shortage_aging` + where
	columns := []Column{
		{Key: "requirement_code", Label: "Kode material"},
		{Key: "wo_code", Label: "Work order"},
		{Key: "unit_code", Label: "Unit"},
		{Key: "sparepart_code", Label: "Kode sparepart"},
		{Key: "sparepart_name", Label: "Nama sparepart"},
		{Key: "qty_required", Label: "Qty diminta"},
		{Key: "shortage", Label: "Kurang"},
		{Key: "age_days", Label: "Umur (hari)"},
		{Key: "age_bucket", Label: "Bucket umur"},
		{Key: "shortage_since", Label: "Sejak"},
	}
	return r.tableWithCount(ctx, columns, filter, selectSQL,
		`SELECT count(*) FROM v_shortage_aging`+where,
		` ORDER BY age_days DESC, requirement_code`, q.args)
}

func (r *Repository) sparepartConsumption(ctx context.Context, filter Filter) (Table, error) {
	q := newQueryArgs(r.loc)
	q.addFilter(filter, "used_at", "unit_id", "unit_model_id", "movement_type")
	q.addLocation(filter, "warehouse_location_id")
	where := q.whereSQL()
	bucketExpr, err := bucketSQL(filter.Bucket, "used_at", 1)
	if err != nil {
		return Table{}, err
	}
	selectSQL := `
		SELECT sparepart_code, sparepart_name, COALESCE(unit_code, '') AS unit_code,
		       COALESCE(component_code, '') AS component_code,
		       COALESCE(component_name, '') AS component_name,
		       sum(net_qty)::float8 AS net_qty, count(*)::int AS usage_count,
		       min(used_at) AS first_used_at, max(used_at) AS last_used_at,
		       ` + bucketExpr + ` AS bucket
		FROM v_sparepart_consumption` + where + `
		GROUP BY 1, 2, 3, 4, 5, 10
		HAVING sum(net_qty) > 0`
	columns := []Column{
		{Key: "bucket", Label: "Bucket"},
		{Key: "sparepart_code", Label: "Kode sparepart"},
		{Key: "sparepart_name", Label: "Nama sparepart"},
		{Key: "unit_code", Label: "Unit"},
		{Key: "component_name", Label: "Komponen"},
		{Key: "net_qty", Label: "Qty bersih"},
		{Key: "usage_count", Label: "Jumlah pemakaian"},
		{Key: "last_used_at", Label: "Terakhir dipakai"},
	}
	return r.tableWithCount(ctx, columns, filter, selectSQL,
		`SELECT count(*) FROM (SELECT 1 FROM v_sparepart_consumption`+where+` GROUP BY sparepart_code, sparepart_name, unit_code, component_code, component_name, `+bucketExpr+`
		 HAVING sum(net_qty) > 0) x`,
		` ORDER BY 1 DESC, 2`, q.args)
}

func (r *Repository) inventoryMovement(ctx context.Context, filter Filter) (Table, error) {
	q := newQueryArgs(r.loc)
	q.addFilter(filter, "posted_at", "", "", "status")
	q.addLocation(filter, "from_location_id", "to_location_id")
	where := q.whereSQL()
	bucketExpr, err := bucketSQL(filter.Bucket, "posted_at", 1)
	if err != nil {
		return Table{}, err
	}
	selectSQL := `
		SELECT transaction_code, type, status, sparepart_code, sparepart_name,
		       COALESCE(NULLIF(from_location_code, ''), '') AS from_location,
		       COALESCE(NULLIF(to_location_code, ''), '') AS to_location,
		       qty::float8, posted_at, ` + bucketExpr + ` AS bucket,
		       COALESCE(reference_no, '') AS reference_no,
		       COALESCE(posted_by_name, '') AS posted_by_name
		FROM v_inventory_movement` + where
	columns := []Column{
		{Key: "bucket", Label: "Bucket"},
		{Key: "transaction_code", Label: "Kode transaksi"},
		{Key: "type", Label: "Tipe"},
		{Key: "status", Label: "Status"},
		{Key: "sparepart_code", Label: "Kode sparepart"},
		{Key: "sparepart_name", Label: "Nama sparepart"},
		{Key: "from_location", Label: "Dari lokasi"},
		{Key: "to_location", Label: "Ke lokasi"},
		{Key: "qty", Label: "Qty"},
		{Key: "posted_at", Label: "Waktu posting"},
		{Key: "reference_no", Label: "Referensi"},
		{Key: "posted_by_name", Label: "Dicatat oleh"},
	}
	return r.tableWithCount(ctx, columns, filter, selectSQL,
		`SELECT count(*) FROM v_inventory_movement`+where,
		` ORDER BY posted_at DESC, transaction_id DESC`, q.args)
}

func (r *Repository) componentReplacement(ctx context.Context, filter Filter) (Table, error) {
	q := newQueryArgs(r.loc)
	q.addFilter(filter, "last_replaced_at", "unit_id", "", "")
	where := q.whereSQL()
	bucketExpr, err := bucketSQL(filter.Bucket, "last_replaced_at", 1)
	if err != nil {
		return Table{}, err
	}
	selectSQL := `
		SELECT unit_code, component_code,
		       COALESCE(component_name, '') AS component_name,
		       replacement_count::int, total_qty::float8,
		       avg_interval_days::float8, avg_interval_hm::float8, avg_interval_km::float8,
		       first_replaced_at, last_replaced_at,
		       ` + bucketExpr + ` AS bucket
		FROM v_component_replacement_frequency` + where
	columns := []Column{
		{Key: "bucket", Label: "Bucket"},
		{Key: "unit_code", Label: "Unit"},
		{Key: "component_code", Label: "Kode komponen"},
		{Key: "component_name", Label: "Komponen"},
		{Key: "replacement_count", Label: "Jumlah penggantian"},
		{Key: "total_qty", Label: "Total qty"},
		{Key: "avg_interval_days", Label: "Rata-rata hari"},
		{Key: "avg_interval_hm", Label: "Rata-rata HM"},
		{Key: "avg_interval_km", Label: "Rata-rata KM"},
		{Key: "last_replaced_at", Label: "Terakhir diganti"},
	}
	return r.tableWithCount(ctx, columns, filter, selectSQL,
		`SELECT count(*) FROM v_component_replacement_frequency`+where,
		` ORDER BY 1 DESC, 2, 3`, q.args)
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableID(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}
