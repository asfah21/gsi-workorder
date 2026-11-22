package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/inventory"
	"gsi-workorder/internal/material"
	"gsi-workorder/internal/pm"
	"gsi-workorder/internal/spareparts"
	"gsi-workorder/internal/workorders"
)

func TestM6MaterialFlow(t *testing.T) {
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
	var adminID, plannerID, storekeeperID, mechanicID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m6-admin', 'Admin M6', 'admin', true) RETURNING id
	`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m6-planner', 'Planner M6', 'planner', true) RETURNING id
	`).Scan(&plannerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m6-store', 'Storekeeper M6', 'storekeeper', true) RETURNING id
	`).Scan(&storekeeperID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m6-mechanic', 'Mechanic M6', 'mechanic', true) RETURNING id
	`).Scan(&mechanicID); err != nil {
		t.Fatal(err)
	}
	var unitModelID, unitID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO unit_models (manufacturer, model, is_active)
		VALUES ('M6 Unit', 'FMX', true) RETURNING id
	`).Scan(&unitModelID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_model_id, current_hm, current_km, is_active)
		VALUES ('M6-UNIT', $1, 100, 1000, true) RETURNING id
	`, unitModelID).Scan(&unitID); err != nil {
		t.Fatal(err)
	}

	sparepartRepo := spareparts.NewRepository(pool)
	admin := spareparts.Actor{ID: adminID, Role: "admin"}
	createSparepart := func(code, name string) spareparts.Sparepart {
		t.Helper()
		item, err := sparepartRepo.Create(ctx, admin, spareparts.Input{
			Code: code, Name: name, Category: "M6", UOM: "pcs",
		})
		if err != nil {
			t.Fatal(err)
		}
		return item
	}
	partA := createSparepart("M6-A", "Part A")
	partB := createSparepart("M6-B", "Part B")
	partC := createSparepart("M6-C", "Part C")
	partD := createSparepart("M6-D", "Part D")
	partE := createSparepart("M6-E", "Part E")
	partF := createSparepart("M6-F", "Part F")
	partPM := createSparepart("M6-PM", "Part PM")

	invRepo := inventory.NewRepository(pool, loc)
	warehouse, err := invRepo.CreateWarehouse(ctx, inventory.Actor{ID: plannerID, Role: "planner"}, inventory.WarehouseInput{
		Code: "M6-WH", Name: "Gudang M6",
	})
	if err != nil {
		t.Fatal(err)
	}
	location, err := invRepo.CreateLocation(ctx, inventory.Actor{ID: plannerID, Role: "planner"}, inventory.LocationInput{
		WarehouseID: warehouse.ID, Code: "A1", Name: "Rak A1", LocationType: "rack",
	})
	if err != nil {
		t.Fatal(err)
	}
	storeActor := inventory.Actor{ID: storekeeperID, Role: "storekeeper"}
	woID := insertWorkOrder(t, ctx, pool, "WO-M6-0001", unitID, mechanicID, plannerID, "in_progress")
	matRepo := material.NewRepository(pool, invRepo, loc)
	matPlanner := material.Actor{ID: plannerID, Role: "planner"}
	matStore := material.Actor{ID: storekeeperID, Role: "storekeeper"}

	req, err := matRepo.Create(ctx, matPlanner, material.CreateInput{
		WOID: woID, Source: "manual", Notes: "Material uji M6",
		Items: []material.ItemInput{
			{SparepartID: partA.ID, QtyRequired: 10},
			{SparepartID: partB.ID, QtyRequired: 5},
			{SparepartID: partC.ID, QtyRequired: 3, IsOptional: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.Status != "draft" || req.ItemCount != 3 || req.RequiredCount != 2 {
		t.Fatalf("created requirement = %+v", req)
	}
	planned := "planned"
	req, err = matRepo.Update(ctx, req.ID, matPlanner, material.PatchInput{Status: &planned})
	if err != nil {
		t.Fatal(err)
	}
	if req.Status != "planned" {
		t.Fatalf("planned status = %q", req.Status)
	}
	req, err = matRepo.CheckStock(ctx, req.ID, matPlanner)
	if err != nil {
		t.Fatal(err)
	}
	if req.Status != "shortage" || req.ShortageSince == nil || req.ShortageCount != 2 {
		t.Fatalf("shortage requirement = %+v", req)
	}
	var agingCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM v_shortage_aging WHERE material_requirement_id = $1`, req.ID).Scan(&agingCount); err != nil {
		t.Fatal(err)
	}
	if agingCount != 2 {
		t.Fatalf("shortage aging rows = %d, want 2", agingCount)
	}

	// Stock arrival clears total shortage but leaves B partial.
	if _, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "receipt", ReferenceNo: "M6-RECEIPT-1",
		Items: []inventory.PostItem{
			{SparepartID: partA.ID, ToLocationID: &location.ID, Qty: 12},
			{SparepartID: partB.ID, ToLocationID: &location.ID, Qty: 2},
			{SparepartID: partC.ID, ToLocationID: &location.ID, Qty: 5},
			{SparepartID: partD.ID, ToLocationID: &location.ID, Qty: 5},
			{SparepartID: partE.ID, ToLocationID: &location.ID, Qty: 2},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := matRepo.RecomputeForSpareparts(ctx, []int64{partA.ID, partB.ID, partC.ID, partD.ID, partE.ID}); err != nil {
		t.Fatal(err)
	}
	req, err = matRepo.Get(ctx, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if req.Status != "partially_ready" || req.ShortageSince != nil {
		t.Fatalf("partially ready requirement = %+v", req)
	}

	// Reserve and replay the same request idempotently.
	req, err = matRepo.Reserve(ctx, req.ID, matStore, material.ReserveInput{IdempotencyKey: "m6-reserve-1", Notes: "Reserve uji"})
	if err != nil {
		t.Fatal(err)
	}
	if req.DerivedStatus != "partially_ready" {
		t.Fatalf("reserve derived status = %q, want partially_ready", req.DerivedStatus)
	}
	mustBalanceReserved(t, ctx, pool, partA.ID, location.ID, 10)
	replayed, err := matRepo.Reserve(ctx, req.ID, matStore, material.ReserveInput{IdempotencyKey: "m6-reserve-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(replayed.Items) != len(req.Items) {
		t.Fatalf("idempotent reserve items = %d", len(replayed.Items))
	}
	var reservationCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inventory_reservations WHERE status = 'active'`).Scan(&reservationCount); err != nil {
		t.Fatal(err)
	}
	if reservationCount != 3 {
		t.Fatalf("active reservations = %d, want 3", reservationCount)
	}

	// Bringing B to full coverage changes reservation status to reserved.
	if _, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "receipt", ReferenceNo: "M6-EXTRA-B2",
		Items: []inventory.PostItem{{SparepartID: partB.ID, ToLocationID: &location.ID, Qty: 3}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := matRepo.RecomputeForSpareparts(ctx, []int64{partB.ID}); err != nil {
		t.Fatal(err)
	}
	req, err = matRepo.Get(ctx, req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if req.DerivedStatus != "reserved" {
		t.Fatalf("reserved derived status = %q, want reserved", req.DerivedStatus)
	}

	recon, err := invRepo.Reconcile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if recon.Mismatch != 0 {
		t.Fatalf("reconciliation with reservations = %d, rows %+v", recon.Mismatch, recon.Rows)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE inventory_balances SET reserved = reserved + 1
		WHERE sparepart_id = $1 AND warehouse_location_id = $2
	`, partA.ID, location.ID); err != nil {
		t.Fatal(err)
	}
	recon, err = invRepo.Reconcile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if recon.Mismatch != 1 || recon.Rows[0].ReservedDifference != 1 {
		t.Fatalf("reserved reconciliation = %+v", recon)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE inventory_balances SET reserved = reserved - 1
		WHERE sparepart_id = $1 AND warehouse_location_id = $2
	`, partA.ID, location.ID); err != nil {
		t.Fatal(err)
	}

	// Issue consumes reservation FIFO and records usage.
	var partAItemID int64
	for _, item := range req.Items {
		if item.SparepartID == partA.ID {
			partAItemID = item.ID
		}
	}
	req, err = matRepo.Issue(ctx, req.ID, matStore, material.IssueInput{
		IdempotencyKey: "m6-issue-1", Notes: "Issue uji",
		Items: []material.IssueItemInput{{MaterialItemID: partAItemID, Qty: 4}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.Status != "issued" || len(req.Usages) != 1 || req.Usages[0].MovementType != "issue" {
		t.Fatalf("issued requirement = %+v", req)
	}
	mustBalanceOnHand(t, ctx, pool, partA.ID, location.ID, 8)
	mustBalanceReserved(t, ctx, pool, partA.ID, location.ID, 6)
	issueUsageID := req.Usages[0].ID

	req, err = matRepo.Return(ctx, req.ID, matStore, material.ReturnInput{
		IdempotencyKey: "m6-return-1",
		Items:          []material.ReturnItemInput{{UsageID: &issueUsageID, Qty: 1, Notes: "Sisa"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Usages) != 2 {
		t.Fatalf("usage after return = %+v", req.Usages)
	}
	mustBalanceOnHand(t, ctx, pool, partA.ID, location.ID, 9)
	if req.Usages[0].MovementType != "return" || req.Usages[0].ReturnOfUsageID == nil {
		t.Fatalf("return usage = %+v", req.Usages[0])
	}
	if _, err := matRepo.Return(ctx, req.ID, matStore, material.ReturnInput{
		Items: []material.ReturnItemInput{{UsageID: &issueUsageID, Qty: 4}},
	}); !errors.Is(err, material.ErrValidation) {
		t.Fatalf("return after full return error = %v, want validation", err)
	}

	// Release one optional reservation explicitly.
	reservations, total, err := matRepo.ListReservations(ctx, material.ReservationFilter{
		Status: "active", RequirementID: &req.ID, Page: 1, Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(reservations) != 3 {
		t.Fatalf("reservations = %d/%d", total, len(reservations))
	}
	var optionalReservation material.Reservation
	for _, reservation := range reservations {
		if reservation.SparepartID == partC.ID {
			optionalReservation = reservation
		}
	}
	if optionalReservation.ID == 0 {
		t.Fatal("optional reservation not found")
	}
	released, err := matRepo.ReleaseReservation(ctx, optionalReservation.ID, matStore, "Tidak jadi dipakai")
	if err != nil {
		t.Fatal(err)
	}
	if released.Status != "released" || released.Qty != 0 {
		t.Fatalf("released reservation = %+v", released)
	}
	mustBalanceReserved(t, ctx, pool, partC.ID, location.ID, 0)

	// Work-order waiting_part guard and completion use the material state.
	woRepo := workorders.NewRepository(pool, loc)
	woRepo.SetMaterialRepository(matRepo)
	wo2ID := insertWorkOrder(t, ctx, pool, "WO-M6-0002", unitID, mechanicID, plannerID, "in_progress")
	req2, err := matRepo.Create(ctx, matPlanner, material.CreateInput{
		WOID: wo2ID, Source: "manual",
		Items: []material.ItemInput{{SparepartID: partF.ID, QtyRequired: 3}},
	})
	if err != nil {
		t.Fatal(err)
	}
	req2, err = matRepo.CheckStock(ctx, req2.ID, matPlanner)
	if err != nil {
		t.Fatal(err)
	}
	if req2.Status != "shortage" {
		t.Fatalf("WO2 material status = %q, want shortage", req2.Status)
	}
	if _, err := woRepo.Update(ctx, wo2ID, workorders.Actor{ID: plannerID, Role: "planner"}, workorders.PatchInput{
		Status: "waiting_part", Note: "Menunggu part D",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := woRepo.Update(ctx, wo2ID, workorders.Actor{ID: plannerID, Role: "planner"}, workorders.PatchInput{
		Status: "in_progress", Note: "Coba kembali",
	}); err == nil {
		t.Fatal("waiting_part to in_progress while shortage should fail")
	}
	if _, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "receipt", ReferenceNo: "M6-F",
		Items: []inventory.PostItem{{SparepartID: partF.ID, ToLocationID: &location.ID, Qty: 3}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := matRepo.RecomputeForSpareparts(ctx, []int64{partF.ID}); err != nil {
		t.Fatal(err)
	}
	req2, err = matRepo.Reserve(ctx, req2.ID, matStore, material.ReserveInput{})
	if err != nil {
		t.Fatal(err)
	}
	if req2.DerivedStatus != "reserved" {
		t.Fatalf("WO2 reserved status = %q, want reserved", req2.DerivedStatus)
	}
	if _, err := woRepo.Update(ctx, wo2ID, workorders.Actor{ID: mechanicID, Role: "mechanic"}, workorders.PatchInput{
		Status: "in_progress", Note: "Material siap",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := woRepo.Update(ctx, wo2ID, workorders.Actor{ID: mechanicID, Role: "mechanic"}, workorders.PatchInput{
		Status: "done", Note: "Pekerjaan selesai",
	}); err != nil {
		t.Fatal(err)
	}
	req2, err = matRepo.Get(ctx, req2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if req2.Status != "completed" {
		t.Fatalf("WO2 material status = %q, want completed", req2.Status)
	}
	mustBalanceReserved(t, ctx, pool, partF.ID, location.ID, 0)

	// WO cancel releases active reservations and cancels material.
	wo3ID := insertWorkOrder(t, ctx, pool, "WO-M6-0003", unitID, mechanicID, plannerID, "in_progress")
	req3, err := matRepo.Create(ctx, matPlanner, material.CreateInput{
		WOID: wo3ID, Source: "manual",
		Items: []material.ItemInput{{SparepartID: partE.ID, QtyRequired: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	req3, err = matRepo.CheckStock(ctx, req3.ID, matPlanner)
	if err != nil {
		t.Fatal(err)
	}
	req3, err = matRepo.Reserve(ctx, req3.ID, matStore, material.ReserveInput{})
	if err != nil {
		t.Fatal(err)
	}
	mustBalanceReserved(t, ctx, pool, partE.ID, location.ID, 2)
	if _, err := woRepo.Update(ctx, wo3ID, workorders.Actor{ID: plannerID, Role: "planner"}, workorders.PatchInput{
		Status: "cancelled", Reason: "Dibatalkan uji M6",
	}); err != nil {
		t.Fatal(err)
	}
	req3, err = matRepo.Get(ctx, req3.ID)
	if err != nil {
		t.Fatal(err)
	}
	if req3.Status != "cancelled" {
		t.Fatalf("WO3 material status = %q, want cancelled", req3.Status)
	}
	mustBalanceReserved(t, ctx, pool, partE.ID, location.ID, 0)

	// PM plan materials are snapshotted when the PM WO is generated.
	pmRepo := pm.NewRepository(pool, loc)
	pmPlanner := pm.Actor{ID: plannerID, Role: "planner"}
	today := time.Now().In(loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	intervalDays := 30
	plan, err := pmRepo.CreatePlan(ctx, pmPlanner, pm.PlanInput{
		Name: "PM M6 Material", UnitID: &unitID, Status: "active",
		EffectiveFrom: today, IntervalDays: &intervalDays,
		IntervalRule: "first_reached", DefaultPriority: "medium",
		Checklist: []pm.ChecklistInput{{Sequence: 1, Task: "Cek part", Standard: "Sesuai"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pmRepo.AddPlanMaterial(ctx, plan.ID, pmPlanner, pm.PlanMaterialInput{
		SparepartID: partPM.ID, Qty: 2, IsOptional: true, Notes: "PM part",
	}); err != nil {
		t.Fatal(err)
	}
	schedules, _, err := pmRepo.ListSchedules(ctx, pm.ScheduleFilter{PlanID: &plan.ID, ActiveOnly: true, Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(schedules) == 0 {
		t.Fatal("PM schedule not found")
	}
	generated, err := pmRepo.GenerateWO(ctx, schedules[0].ID, pmPlanner)
	if err != nil {
		t.Fatal(err)
	}
	pmRequirements, pmTotal, err := matRepo.List(ctx, material.Filter{WOID: &generated.WorkOrder.ID, Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if pmTotal != 1 || len(pmRequirements) != 1 {
		t.Fatalf("PM material requirements = %d/%d", pmTotal, len(pmRequirements))
	}
	pmReq := pmRequirements[0]
	if pmReq.Source != "pm" || pmReq.Status != "draft" || len(pmReq.Items) != 1 || !pmReq.Items[0].IsOptional {
		t.Fatalf("PM material snapshot = %+v", pmReq)
	}

}

func insertWorkOrder(t *testing.T, ctx context.Context, pool *pgxpool.Pool, code string, unitID, assigneeID, createdBy int64, status string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO work_orders (
			code, unit_id, source, title, priority, status, maintenance_type,
			assignee_id, created_by, started_at
		)
		VALUES ($1, $2, 'manual', 'Material M6 test', 'medium', $3, 'corrective',
		        $4, $5, CASE WHEN $3 = 'in_progress' THEN now() ELSE NULL END)
		RETURNING id
	`, code, unitID, status, assigneeID, createdBy).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func truncateM6(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			work_order_sparepart_usage,
			inventory_reservations,
			material_requirement_updates,
			material_requirement_items,
			material_requirements,
			stock_opname_items, stock_opname_sessions,
			inventory_transaction_items, inventory_transactions,
			inventory_balances, inventory_stock_policies,
			warehouse_locations, warehouses,
			pm_plan_materials,
			sparepart_compatibility, sparepart_alternatives,
			sparepart_numbers, spareparts,
			components,
			notifications, wo_updates, wo_attachments, wo_findings,
			wo_candidates, work_orders, pm_schedules, pm_plan_checklist_items,
			pm_plans, unit_meter_readings, units, unit_models, engine_models,
			p2h_findings, p2h_reports, work_order_code_counters,
			app_settings_history, app_settings, users
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		ALTER SEQUENCE inventory_transaction_code_seq RESTART WITH 1;
		ALTER SEQUENCE stock_opname_code_seq RESTART WITH 1;
		ALTER SEQUENCE material_requirement_code_seq RESTART WITH 1;
	`); err != nil {
		t.Fatal(err)
	}
}

func mustBalanceOnHand(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sparepartID, locationID int64, want float64) {
	t.Helper()
	var got float64
	if err := pool.QueryRow(ctx, `
		SELECT on_hand::float8 FROM inventory_balances
		WHERE sparepart_id = $1 AND warehouse_location_id = $2
	`, sparepartID, locationID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("on_hand = %v, want %v", got, want)
	}
}

func mustBalanceReserved(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sparepartID, locationID int64, want float64) {
	t.Helper()
	var got float64
	if err := pool.QueryRow(ctx, `
		SELECT reserved::float8 FROM inventory_balances
		WHERE sparepart_id = $1 AND warehouse_location_id = $2
	`, sparepartID, locationID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("reserved = %v, want %v", got, want)
	}
}
