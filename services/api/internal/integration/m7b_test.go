package integration_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/reporting"
)

func TestM7BDashboardAndReports(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	truncateM6(t, ctx, pool)

	loc, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)

	var adminID, mechanicID int64
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active) VALUES ('m7b-admin', 'Admin M7B', 'admin', true) RETURNING id
	`).Scan(&adminID))
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active) VALUES ('m7b-mechanic', 'Mekanik M7B', 'mechanic', true) RETURNING id
	`).Scan(&mechanicID))

	var modelID, unitA, unitB int64
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO unit_models (manufacturer, model, category, is_active)
		VALUES ('Caterpillar', '320', 'excavator', true) RETURNING id
	`).Scan(&modelID))
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_type, unit_model_id, is_active)
		VALUES ('UNIT-M7B-A', 'excavator', $1, true) RETURNING id
	`, modelID).Scan(&unitA))
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_type, is_active) VALUES ('UNIT-M7B-B', 'excavator', true) RETURNING id
	`).Scan(&unitB))

	var sparepartA, sparepartB, componentID, warehouseID, locationID int64
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO spareparts (code, name, uom) VALUES ('BRG-M7B', 'Bearing M7B', 'pcs') RETURNING id
	`).Scan(&sparepartA))
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO spareparts (code, name, uom) VALUES ('FLT-M7B', 'Filter M7B', 'pcs') RETURNING id
	`).Scan(&sparepartB))
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO components (code, name, is_active) VALUES ('CMP-M7B', 'Komponen M7B', true) RETURNING id
	`).Scan(&componentID))
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO warehouses (code, name) VALUES ('WH-M7B', 'Gudang M7B') RETURNING id
	`).Scan(&warehouseID))
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO warehouse_locations (warehouse_id, code, name)
		VALUES ($1, 'LOC-M7B', 'Rak M7B') RETURNING id
	`, warehouseID).Scan(&locationID))
	execOK(t, ctx, pool, `
		INSERT INTO inventory_stock_policies (sparepart_id, warehouse_location_id, min_qty, critical_qty, is_active)
		VALUES ($1, $2, 5, 2, true), ($3, $2, 5, 2, false)
	`, sparepartA, locationID, sparepartB)
	execOK(t, ctx, pool, `
		INSERT INTO inventory_balances (sparepart_id, warehouse_location_id, on_hand)
		VALUES ($1, $2, 1), ($3, $2, 0)
	`, sparepartA, locationID, sparepartB)

	var planA, planB int64
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO pm_plans (name, unit_id, status, effective_from, interval_days, created_by)
		VALUES ('PM M7B A', $1, 'active', DATE '2026-01-01', 30, $2) RETURNING id
	`, unitA, adminID).Scan(&planA))
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO pm_plans (name, unit_model_id, status, effective_from, interval_days, created_by)
		VALUES ('PM M7B B', $1, 'active', DATE '2026-01-01', 30, $2) RETURNING id
	`, modelID, adminID).Scan(&planB))
	execOK(t, ctx, pool, `
		INSERT INTO pm_schedules (pm_plan_id, unit_id, cycle_no, status, due_date, created_at)
		VALUES ($1, $2, 1, 'overdue', DATE '2026-11-01', $4),
		       ($3, $2, 1, 'scheduled', DATE '2026-11-20', $4)
	`, planA, unitA, planB, base)

	woBreakdown := insertM7WorkOrder(t, ctx, pool, unitA, "M7B-WO-BREAKDOWN", "Breakdown M7B", "breakdown", "done",
		base, base.Add(time.Hour), base.Add(5*time.Hour), adminID, mechanicID, adminID)
	_ = insertM7WorkOrder(t, ctx, pool, unitA, "M7B-WO-PM", "PM M7B", "preventive", "done",
		base, base.Add(6*time.Hour), base.Add(7*time.Hour), adminID, mechanicID, adminID)
	woWaiting := insertM7WorkOrder(t, ctx, pool, unitB, "M7B-WO-WAITING", "Waiting M7B", "breakdown", "waiting_part",
		base.Add(8*time.Hour), time.Time{}, time.Time{}, adminID, mechanicID, adminID)
	_ = insertM7WorkOrder(t, ctx, pool, unitB, "M7B-WO-CANCELLED", "Cancelled M7B", "breakdown", "cancelled",
		base, base, base.Add(time.Hour), adminID, mechanicID, adminID)
	_ = insertM7WorkOrder(t, ctx, pool, unitB, "M7B-WO-INSPECTION", "Inspection M7B", "inspection", "done",
		base, base, base.Add(2*time.Hour), adminID, mechanicID, adminID)
	execOK(t, ctx, pool, `
		INSERT INTO wo_updates (wo_id, user_id, note, status_from, status_to, created_at)
		VALUES ($1, $2, 'waiting sparepart', 'in_progress', 'waiting_part', $3)
	`, woWaiting, mechanicID, base.Add(9*time.Hour))

	var requirementID int64
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO material_requirements (code, wo_id, status, shortage_since, requested_by, requested_at, created_at)
		VALUES ('MR-M7B-001', $1, 'shortage', $2, $3, $2, $2) RETURNING id
	`, woWaiting, base.Add(10*time.Hour), mechanicID).Scan(&requirementID))
	execOK(t, ctx, pool, `
		INSERT INTO material_requirement_items (material_requirement_id, sparepart_id, qty_required)
		VALUES ($1, $2, 3)
	`, requirementID, sparepartB)

	_ = insertM7Transaction(t, ctx, pool, "M7B-RCP", "receipt", adminID, base.Add(2*time.Hour), "", nil, nil, &locationID, sparepartA, 10)
	issueID := insertM7Transaction(t, ctx, pool, "M7B-ISS", "issue", mechanicID, base.Add(3*time.Hour), "M7B-WO-BREAKDOWN", nil, &locationID, nil, sparepartA, 2)
	issueItemID := m7bItemID(t, ctx, pool, issueID)
	returnID := insertM7Transaction(t, ctx, pool, "M7B-RET", "return", mechanicID, base.Add(4*time.Hour), "M7B-WO-BREAKDOWN", &issueID, &locationID, nil, sparepartA, 0.5)
	returnItemID := m7bItemID(t, ctx, pool, returnID)
	execOK(t, ctx, pool, `
		INSERT INTO work_order_sparepart_usage (
			wo_id, sparepart_id, component_id, warehouse_location_id, qty,
			movement_type, inventory_transaction_item_id, used_by, used_at
		) VALUES
			($1, $2, $3, $4, 2, 'issue', $5, $6, $7),
			($1, $2, $3, $4, 0.5, 'return', $8, $6, $9)
	`, woBreakdown, sparepartA, componentID, locationID, issueItemID, mechanicID, base.Add(250*time.Minute), returnItemID, base.Add(350*time.Minute))

	repo := reporting.NewRepository(pool, loc)
	service := reporting.NewService(repo, loc)
	filter := reporting.Filter{
		Bucket: reporting.BucketDay, GroupBy: "plan", Format: reporting.FormatJSON,
		Page: 1, Limit: 25,
		From: ptrTime(time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)),
		To:   ptrTime(time.Date(2026, 11, 30, 16, 0, 0, 0, time.UTC)),
	}

	summary, err := repo.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assertInt(t, summary, "units_total", 2)
	assertInt(t, summary, "pm_overdue", 1)
	assertInt(t, summary, "material_shortage", 1)
	assertInt(t, summary, "stock_critical", 1)
	assertInt(t, summary, "waiting_part_units", 1)

	pm, err := repo.PM(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	assertInt(t, pm.Summary, "total", 2)
	assertInt(t, pm.Summary, "overdue", 1)
	if len(pm.Due.Rows) != 2 {
		t.Fatalf("pm due rows = %d, want 2: %#v", len(pm.Due.Rows), pm.Due.Rows)
	}

	material, err := repo.Material(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	assertInt(t, material.Summary, "shortage", 1)
	if material.Requirements.Meta.Total != 1 {
		t.Fatalf("material requirements = %d, want 1", material.Requirements.Meta.Total)
	}

	inventory, err := repo.Inventory(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	assertInt(t, inventory.Summary, "critical", 1)
	if len(inventory.LowStock.Rows) != 1 {
		t.Fatalf("low stock rows = %#v", inventory.LowStock.Rows)
	}
	if inventory.Movements.Meta.Total != 3 {
		t.Fatalf("movement rows = %d, want 3", inventory.Movements.Meta.Total)
	}
	for _, row := range inventory.Movements.Rows {
		if row["bucket"] == nil || !strings.HasPrefix(row["bucket"].(string), "2026-11-") {
			t.Fatalf("movement bucket = %v", row["bucket"])
		}
	}

	maintenance, err := repo.Maintenance(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	assertInt(t, maintenance.Summary, "total", 5)
	assertInt(t, maintenance.Summary, "done", 3)
	assertFloat(t, maintenance.Summary, "total_minutes", 300)
	assertFloat(t, maintenance.Summary, "unplanned_minutes", 240)
	assertFloat(t, maintenance.Summary, "planned_minutes", 60)
	assertInt(t, maintenance.Summary, "units_waiting", 1)
	if len(maintenance.Waiting.Rows) != 1 {
		t.Fatalf("waiting rows = %#v", maintenance.Waiting.Rows)
	}

	consumption, _, err := service.Report(ctx, "sparepart-consumption", filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(consumption.Rows) != 1 || asFloat(t, consumption.Rows[0]["net_qty"]) != 1.5 {
		t.Fatalf("consumption rows = %#v", consumption.Rows)
	}
	compliance, _, err := service.Report(ctx, "pm-compliance", filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(compliance.Rows) == 0 {
		t.Fatal("pm compliance rows empty")
	}

	var buffer bytes.Buffer
	if err := service.RenderCSV(&buffer, consumption); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buffer.String(), "Qty bersih") || !strings.Contains(buffer.String(), "1.5") {
		t.Fatalf("csv output = %q", buffer.String())
	}
}

func insertM7Transaction(t *testing.T, ctx context.Context, pool *pgxpool.Pool, code, kind string, userID int64, postedAt time.Time, referenceNo string, reversalOf *int64, fromLocation, toLocation *int64, sparepartID int64, qty float64) int64 {
	t.Helper()
	var reversalValue, referenceValue any
	if reversalOf != nil {
		reversalValue = *reversalOf
	}
	if referenceNo != "" {
		referenceValue = referenceNo
	}
	var fromValue, toValue any
	if fromLocation != nil {
		fromValue = *fromLocation
	}
	if toLocation != nil {
		toValue = *toLocation
	}
	var id int64
	mustScan(t, pool.QueryRow(ctx, `
		INSERT INTO inventory_transactions (code, type, status, reference_no, reversal_of_id, posted_by, posted_at)
		VALUES ($1, $2, 'posted', $3, $4, $5, $6) RETURNING id
	`, code, kind, referenceValue, reversalValue, userID, postedAt).Scan(&id))
	execOK(t, ctx, pool, `
		INSERT INTO inventory_transaction_items (transaction_id, sparepart_id, from_location_id, to_location_id, qty)
		VALUES ($1, $2, $3, $4, $5)
	`, id, sparepartID, fromValue, toValue, qty)
	return id
}

func m7bItemID(t *testing.T, ctx context.Context, pool *pgxpool.Pool, transactionID int64) int64 {
	t.Helper()
	var id int64
	mustScan(t, pool.QueryRow(ctx, `
		SELECT id FROM inventory_transaction_items WHERE transaction_id = $1 ORDER BY id DESC LIMIT 1
	`, transactionID).Scan(&id))
	return id
}

func mustScan(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func execOK(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		t.Fatal(err)
	}
}

func ptrTime(value time.Time) *time.Time { return &value }

func asFloat(t *testing.T, value any) float64 {
	t.Helper()
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		t.Fatalf("value %v is not numeric", value)
		return 0
	}
}

func assertInt(t *testing.T, values map[string]any, key string, want int) {
	t.Helper()
	got := 0
	switch typed := values[key].(type) {
	case int:
		got = typed
	case int32:
		got = int(typed)
	case int64:
		got = int(typed)
	default:
		got = int(asFloat(t, values[key]))
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", key, got, want)
	}
}

func assertFloat(t *testing.T, values map[string]any, key string, want float64) {
	t.Helper()
	got := asFloat(t, values[key])
	if got != want {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}
