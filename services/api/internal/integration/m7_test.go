package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/units"
)

func TestM7UnitHistoryReporting(t *testing.T) {
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

	base := time.Date(2026, 11, 10, 7, 0, 0, 0, time.UTC)
	var adminID, mechanicID, supervisorID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m7-admin', 'Admin M7', 'admin', true) RETURNING id
	`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m7-mechanic', 'Mechanic M7', 'mechanic', true) RETURNING id
	`).Scan(&mechanicID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m7-supervisor', 'Supervisor M7', 'planner', true) RETURNING id
	`).Scan(&supervisorID); err != nil {
		t.Fatal(err)
	}

	var unitID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_type, merk, site_location, current_hm, current_km, is_active)
		VALUES ('LV-M7', 'Dump Truck', 'Test', 'Site M7', 120, 1000, true)
		RETURNING id
	`).Scan(&unitID); err != nil {
		t.Fatal(err)
	}
	var componentID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO components (code, name, category)
		VALUES ('C-M7', 'Hydraulic Pump', 'Hydraulic') RETURNING id
	`).Scan(&componentID); err != nil {
		t.Fatal(err)
	}
	var zeroComponentID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO components (code, name, category)
		VALUES ('C-M7-Z', 'Returned Component', 'Test') RETURNING id
	`).Scan(&zeroComponentID); err != nil {
		t.Fatal(err)
	}
	var sparepartID, fallbackPartID, fullyReturnedPartID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO spareparts (code, name, uom)
		VALUES ('SP-M7', 'Pump Seal Kit', 'pcs') RETURNING id
	`).Scan(&sparepartID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO spareparts (code, name, uom)
		VALUES ('SP-M7-FB', 'Filter', 'pcs') RETURNING id
	`).Scan(&fallbackPartID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO spareparts (code, name, uom)
		VALUES ('SP-M7-Z', 'Returned Part', 'pcs') RETURNING id
	`).Scan(&fullyReturnedPartID); err != nil {
		t.Fatal(err)
	}
	var warehouseID, locationID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO warehouses (code, name, site_location)
		VALUES ('WH-M7', 'Warehouse M7', 'Site M7') RETURNING id
	`).Scan(&warehouseID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO warehouse_locations (warehouse_id, code, name)
		VALUES ($1, 'A1', 'Rack A1') RETURNING id
	`, warehouseID).Scan(&locationID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO unit_meter_readings (unit_id, reading_type, value, recorded_at, source_type)
		VALUES ($1, 'hm', 90, $2, 'manual'), ($1, 'km', 1000, $2, 'manual')
	`, unitID, base.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	woA := insertM7WorkOrder(t, ctx, pool, unitID, "WO-M7-A", "Hydraulic breakdown A", "breakdown", "done", base, base.Add(time.Hour), base.Add(5*time.Hour), adminID, mechanicID, supervisorID)
	woB := insertM7WorkOrder(t, ctx, pool, unitID, "WO-M7-B", "Hydraulic breakdown B", "breakdown", "done", base.Add(150*time.Minute), base.Add(3*time.Hour), base.Add(7*time.Hour), adminID, mechanicID, supervisorID)
	woC := insertM7WorkOrder(t, ctx, pool, unitID, "WO-M7-C", "Preventive service", "preventive", "done", base.Add(450*time.Minute), base.Add(8*time.Hour), base.Add(10*time.Hour), adminID, mechanicID, supervisorID)
	_ = insertM7WorkOrder(t, ctx, pool, unitID, "WO-M7-D", "Inspection", "inspection", "done", base.Add(630*time.Minute), base.Add(11*time.Hour), base.Add(12*time.Hour), adminID, mechanicID, supervisorID)
	_ = insertM7WorkOrder(t, ctx, pool, unitID, "WO-M7-E", "Cancelled breakdown", "breakdown", "cancelled", base.Add(720*time.Minute), base.Add(12*time.Hour), base.Add(13*time.Hour), adminID, mechanicID, supervisorID)
	_ = insertM7WorkOrder(t, ctx, pool, unitID, "WO-M7-F", "Open corrective", "corrective", "in_progress", base.Add(780*time.Minute), base.Add(13*time.Hour), time.Time{}, adminID, mechanicID, supervisorID)

	if _, err := pool.Exec(ctx, `
		INSERT INTO wo_updates (wo_id, user_id, note, status_from, status_to, created_at)
		VALUES ($1, $2, 'waiting part', 'in_progress', 'waiting_part', $3),
		       ($1, $2, 'part arrived', 'waiting_part', 'in_progress', $4)
	`, woA, mechanicID, base.Add(90*time.Minute), base.Add(150*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO work_order_checklist_items (wo_id, sequence, component_id, task, result)
		VALUES ($1, 1, $2, 'Check seal', 'ok'), ($1, 2, $2, 'Check pressure', 'not_ok')
	`, woA, componentID); err != nil {
		t.Fatal(err)
	}

	issueA, _ := insertM7Usage(t, ctx, pool, woA, sparepartID, componentID, locationID, 2, floatPtr(100), floatPtr(1000), "issue", 0, mechanicID, base.Add(2*time.Hour))
	_, _ = insertM7Usage(t, ctx, pool, woA, sparepartID, componentID, locationID, 0.5, nil, nil, "return", issueA, mechanicID, base.Add(4*time.Hour))
	_, _ = insertM7Usage(t, ctx, pool, woB, sparepartID, componentID, locationID, 1, floatPtr(120), floatPtr(1030), "issue", 0, mechanicID, base.Add(270*time.Minute))
	_, _ = insertM7Usage(t, ctx, pool, woB, fallbackPartID, 0, locationID, 1, nil, nil, "issue", 0, mechanicID, base.Add(360*time.Minute))
	fullyReturnedIssue, _ := insertM7Usage(t, ctx, pool, woC, fullyReturnedPartID, zeroComponentID, locationID, 1, nil, nil, "issue", 0, mechanicID, base.Add(9*time.Hour))
	_, _ = insertM7Usage(t, ctx, pool, woC, fullyReturnedPartID, zeroComponentID, locationID, 1, nil, nil, "return", fullyReturnedIssue, mechanicID, base.Add(570*time.Minute))

	repo := units.NewPostgresRepository(pool)
	service := units.NewService(repo, nil)

	history, total, err := service.ListMaintenanceHistory(ctx, unitID, units.HistoryFilter{Page: 1, Limit: 25})
	if err != nil {
		t.Fatal(err)
	}
	if total != 6 || len(history) != 6 {
		t.Fatalf("history total = %d/%d, want 6", total, len(history))
	}
	historyA := findHistory(t, history, "WO-M7-A")
	if historyA.ElapsedMinutes == nil || *historyA.ElapsedMinutes != 240 {
		t.Fatalf("elapsed minutes = %v, want 240", historyA.ElapsedMinutes)
	}
	if historyA.UsageCount != 1 || historyA.UsageQty != 1.5 {
		t.Fatalf("usage = %d/%.1f, want 1/1.5", historyA.UsageCount, historyA.UsageQty)
	}
	if historyA.ChecklistTotal != 2 || historyA.ChecklistDone != 2 || historyA.ChecklistNotOK != 1 {
		t.Fatalf("checklist = %d/%d/%d", historyA.ChecklistTotal, historyA.ChecklistDone, historyA.ChecklistNotOK)
	}
	filtered, filteredTotal, err := service.ListMaintenanceHistory(ctx, unitID, units.HistoryFilter{
		Page: 1, Limit: 25, Status: "done", MaintenanceType: "breakdown",
	})
	if err != nil {
		t.Fatal(err)
	}
	if filteredTotal != 2 || len(filtered) != 2 {
		t.Fatalf("filtered history = %d/%d, want 2", filteredTotal, len(filtered))
	}

	componentHistory, componentTotal, err := service.ListComponentHistory(ctx, unitID, units.ComponentHistoryFilter{Page: 1, Limit: 25})
	if err != nil {
		t.Fatal(err)
	}
	if componentTotal != 3 || len(componentHistory) != 3 {
		t.Fatalf("component history = %d/%d, want 3", componentTotal, len(componentHistory))
	}
	eventA := findComponentEvent(t, componentHistory, "WO-M7-A")
	if eventA.Qty != 1.5 || eventA.ReplacementNo != 1 {
		t.Fatalf("component event A = %#v", eventA)
	}
	eventB := findComponentEvent(t, componentHistory, "WO-M7-B")
	if eventB.ComponentID == nil || *eventB.ComponentID != componentID {
		t.Fatalf("component event B component = %v", eventB.ComponentID)
	}
	if eventB.DaysSincePrevious == nil || *eventB.DaysSincePrevious != 0.1 {
		t.Fatalf("days since previous = %v, want 0.1", eventB.DaysSincePrevious)
	}
	if eventB.HMSincePrevious == nil || *eventB.HMSincePrevious != 20 {
		t.Fatalf("hm since previous = %v, want 20", eventB.HMSincePrevious)
	}
	fallbackEvent := findComponentEventByFallback(t, componentHistory, fallbackPartID)
	if !fallbackEvent.IsFallback || fallbackEvent.HM == nil || *fallbackEvent.HM != 90 {
		t.Fatalf("fallback event = %#v", fallbackEvent)
	}
	for _, row := range componentHistory {
		if row.ComponentID != nil && *row.ComponentID == zeroComponentID {
			t.Fatalf("fully returned component should be excluded: %#v", row)
		}
	}

	intervals, err := service.ListComponentIntervals(ctx, unitID)
	if err != nil {
		t.Fatal(err)
	}
	if len(intervals) != 2 {
		t.Fatalf("intervals = %d, want 2", len(intervals))
	}
	componentInterval := findInterval(t, intervals, componentID)
	if componentInterval.ReplacementCount != 2 || componentInterval.TotalQty != 2.5 {
		t.Fatalf("component interval = %#v", componentInterval)
	}
	if componentInterval.AvgIntervalDays == nil || *componentInterval.AvgIntervalDays != 0.1 {
		t.Fatalf("avg interval = %v, want 0.1", componentInterval.AvgIntervalDays)
	}
	fallbackInterval := findFallbackInterval(t, intervals, fallbackPartID)
	if fallbackInterval.ReplacementCount != 1 || fallbackInterval.TotalQty != 1 {
		t.Fatalf("fallback interval = %#v", fallbackInterval)
	}

	downtime, err := service.UnitDowntime(ctx, unitID, units.DowntimeFilter{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(downtime.Events) != 3 {
		t.Fatalf("downtime events = %d, want 3", len(downtime.Events))
	}
	if downtime.Summary.Events != 3 || downtime.Summary.PlannedEvents != 1 || downtime.Summary.UnplannedEvents != 2 {
		t.Fatalf("downtime counts = %#v", downtime.Summary)
	}
	if downtime.Summary.UnplannedMinutes != 360 {
		t.Fatalf("unplanned minutes = %.1f, want 360", downtime.Summary.UnplannedMinutes)
	}
	if downtime.Summary.PlannedMinutes != 120 || downtime.Summary.TotalMinutes != 480 {
		t.Fatalf("planned/total minutes = %.1f/%.1f, want 120/480", downtime.Summary.PlannedMinutes, downtime.Summary.TotalMinutes)
	}
	if downtime.Summary.WaitingMinutes != 60 {
		t.Fatalf("waiting minutes = %.1f, want 60", downtime.Summary.WaitingMinutes)
	}
	if downtime.Summary.MTTRMinutes != 240 {
		t.Fatalf("mttr = %.1f, want 240", downtime.Summary.MTTRMinutes)
	}
	from := base.Add(3 * time.Hour)
	to := base.Add(5 * time.Hour)
	window, err := service.UnitDowntime(ctx, unitID, units.DowntimeFilter{From: &from, To: &to, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if window.Summary.UnplannedMinutes != 120 || window.Summary.Events != 2 {
		t.Fatalf("window downtime = %#v", window.Summary)
	}

	options, err := service.ListComponents(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(options) != 2 {
		t.Fatalf("components = %#v", options)
	}
	foundComponent := false
	for _, option := range options {
		if option.ID == componentID {
			foundComponent = true
		}
	}
	if !foundComponent {
		t.Fatalf("component %d missing from options", componentID)
	}
}

func insertM7WorkOrder(t *testing.T, ctx context.Context, pool *pgxpool.Pool, unitID int64, code, title, maintenanceType, status string, created, started, closed time.Time, adminID, mechanicID, supervisorID int64) int64 {
	t.Helper()
	var closedValue any
	if !closed.IsZero() {
		closedValue = closed
	}
	var id int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO work_orders (
			code, unit_id, source, title, priority, status, maintenance_type,
			assignee_id, supervisor_id, created_by, started_at, closed_at, created_at
		)
		VALUES ($1, $2, 'manual', $3, 'high', $4, $5, $6, $7, $8,
		        CASE WHEN $9::timestamptz IS NULL THEN NULL ELSE $9 END, $10, $11)
		RETURNING id
	`, code, unitID, title, status, maintenanceType, mechanicID, supervisorID, adminID, started, closedValue, created).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func insertM7Usage(t *testing.T, ctx context.Context, pool *pgxpool.Pool, woID, sparepartID, componentID, locationID int64, qty float64, hm, km *float64, movementType string, returnOf int64, userID int64, usedAt time.Time) (int64, int64) {
	t.Helper()
	var transactionID, itemID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO inventory_transactions (code, type, status, posted_by, posted_at)
		VALUES ($1, $2, 'posted', $3, $4) RETURNING id
	`, "M7-"+movementType+"-"+usedAt.Format("150405.000"), movementType, userID, usedAt).Scan(&transactionID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO inventory_transaction_items (transaction_id, sparepart_id, from_location_id, qty)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, transactionID, sparepartID, locationID, qty).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	var componentValue any
	if componentID > 0 {
		componentValue = componentID
	}
	var returnValue any
	if returnOf > 0 {
		returnValue = returnOf
	}
	var usageID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO work_order_sparepart_usage (
			wo_id, sparepart_id, component_id, warehouse_location_id, qty, hm, km,
			movement_type, inventory_transaction_item_id, return_of_usage_id, used_by, used_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, woID, sparepartID, componentValue, locationID, qty, hm, km, movementType, itemID, returnValue, userID, usedAt).Scan(&usageID); err != nil {
		t.Fatal(err)
	}
	return usageID, itemID
}

func findHistory(t *testing.T, rows []units.MaintenanceHistory, code string) units.MaintenanceHistory {
	t.Helper()
	for _, row := range rows {
		if row.WOCode == code {
			return row
		}
	}
	t.Fatalf("history %s not found", code)
	return units.MaintenanceHistory{}
}

func findComponentEvent(t *testing.T, rows []units.ComponentHistory, code string) units.ComponentHistory {
	t.Helper()
	for _, row := range rows {
		if row.WOCode == code && !row.IsFallback {
			return row
		}
	}
	t.Fatalf("component event %s not found", code)
	return units.ComponentHistory{}
}

func findComponentEventByFallback(t *testing.T, rows []units.ComponentHistory, sparepartID int64) units.ComponentHistory {
	t.Helper()
	for _, row := range rows {
		if row.IsFallback && row.FallbackSparepartID != nil && *row.FallbackSparepartID == sparepartID {
			return row
		}
	}
	t.Fatalf("fallback component event %d not found", sparepartID)
	return units.ComponentHistory{}
}

func findInterval(t *testing.T, rows []units.ComponentInterval, componentID int64) units.ComponentInterval {
	t.Helper()
	for _, row := range rows {
		if row.ComponentID != nil && *row.ComponentID == componentID {
			return row
		}
	}
	t.Fatalf("component interval %d not found", componentID)
	return units.ComponentInterval{}
}

func findFallbackInterval(t *testing.T, rows []units.ComponentInterval, sparepartID int64) units.ComponentInterval {
	t.Helper()
	for _, row := range rows {
		if row.IsFallback && row.FallbackSparepartID != nil && *row.FallbackSparepartID == sparepartID {
			return row
		}
	}
	t.Fatalf("fallback interval %d not found", sparepartID)
	return units.ComponentInterval{}
}

func floatPtr(value float64) *float64 {
	return &value
}
