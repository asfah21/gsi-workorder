package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/inventory"
	"gsi-workorder/internal/pm"
	"gsi-workorder/internal/spareparts"
)

func TestM5SparepartInventoryFlow(t *testing.T) {
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
	truncateM5(t, ctx, pool)

	loc, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		t.Fatal(err)
	}
	var adminID, plannerID, storekeeperID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m5-admin', 'Admin M5', 'admin', true) RETURNING id
	`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m5-planner', 'Planner M5', 'planner', true) RETURNING id
	`).Scan(&plannerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m5-store', 'Storekeeper M5', 'storekeeper', true) RETURNING id
	`).Scan(&storekeeperID); err != nil {
		t.Fatal(err)
	}
	var unitModelID, engineModelID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO unit_models (manufacturer, model, is_active)
		VALUES ('M5 Unit', 'FMX', true) RETURNING id
	`).Scan(&unitModelID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO engine_models (manufacturer, model, is_active)
		VALUES ('M5 Engine', 'D13', true) RETURNING id
	`).Scan(&engineModelID); err != nil {
		t.Fatal(err)
	}

	sparepartRepo := spareparts.NewRepository(pool)
	admin := spareparts.Actor{ID: adminID, Role: "admin"}
	planner := spareparts.Actor{ID: plannerID, Role: "planner"}

	oil, err := sparepartRepo.Create(ctx, admin, spareparts.Input{
		Code: "M5-OIL", Name: "Filter Oli M5", Category: "Filter", UOM: "pcs",
		Description: "Filter oli untuk unit uji M5",
	})
	if err != nil {
		t.Fatal(err)
	}
	alt, err := sparepartRepo.Create(ctx, admin, spareparts.Input{
		Code: "M5-FLT", Name: "Filter Alternatif M5", Category: "Filter", UOM: "pcs",
	})
	if err != nil {
		t.Fatal(err)
	}
	if oil.Code != "M5-OIL" || !oil.IsActive {
		t.Fatalf("created sparepart = %+v", oil)
	}

	oeNumber, err := sparepartRepo.CreateNumber(ctx, oil.ID, admin, spareparts.NumberInput{
		PartNumber: "OEM-123", Brand: "Yanmar", NumberType: "oem", IsPrimary: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sparepartRepo.CreateNumber(ctx, oil.ID, admin, spareparts.NumberInput{
		PartNumber: "AF-123", Brand: "Aftermarket Co", NumberType: "aftermarket",
	}); err != nil {
		t.Fatal(err)
	}
	numbers, err := sparepartRepo.ListNumbers(ctx, oil.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(numbers) != 2 || !numbers[0].IsPrimary || numbers[0].ID != oeNumber.ID {
		t.Fatalf("numbers = %+v, want OEM primary first", numbers)
	}
	oil, err = sparepartRepo.Get(ctx, oil.ID)
	if err != nil {
		t.Fatal(err)
	}
	if oil.PrimaryPartNumber != "OEM-123" {
		t.Fatalf("primary part number = %q, want OEM-123", oil.PrimaryPartNumber)
	}

	alternative, err := sparepartRepo.CreateAlternative(ctx, oil.ID, admin, spareparts.AlternativeInput{
		AlternativeSparepartID: alt.ID,
		RelationType:           "equivalent",
		Notes:                  "Cross reference uji",
	})
	if err != nil {
		t.Fatal(err)
	}
	if alternative.VerificationStatus != "pending_verification" {
		t.Fatalf("alternative status = %q, want pending", alternative.VerificationStatus)
	}
	if _, err := sparepartRepo.VerifyAlternative(ctx, oil.ID, alternative.ID, planner, spareparts.VerifyInput{
		Status: "conditional_compatible",
	}); !errors.Is(err, spareparts.ErrValidation) {
		t.Fatalf("conditional alternative without notes error = %v, want validation", err)
	}
	alternative, err = sparepartRepo.VerifyAlternative(ctx, oil.ID, alternative.ID, planner, spareparts.VerifyInput{
		Status: "approved_compatible",
		Notes:  "Terverifikasi uji",
	})
	if err != nil {
		t.Fatal(err)
	}
	if alternative.VerificationStatus != "approved_compatible" || alternative.VerifiedBy == nil {
		t.Fatalf("verified alternative = %+v", alternative)
	}

	compatibility, err := sparepartRepo.CreateCompatibility(ctx, oil.ID, admin, spareparts.CompatibilityInput{
		UnitModelID:   &unitModelID,
		EngineModelID: &engineModelID,
		Configuration: "4x4 heavy duty",
		Source:        "Katalog uji",
		Notes:         "Catatan compatibility",
	})
	if err != nil {
		t.Fatal(err)
	}
	if compatibility.Status != "pending_verification" {
		t.Fatalf("compatibility status = %q, want pending", compatibility.Status)
	}
	if _, err := sparepartRepo.VerifyCompatibility(ctx, oil.ID, compatibility.ID, planner, spareparts.VerifyInput{
		Status: "conditional_compatible",
	}); !errors.Is(err, spareparts.ErrValidation) {
		t.Fatalf("conditional compatibility without note error = %v, want validation", err)
	}
	compatibility, err = sparepartRepo.VerifyCompatibility(ctx, oil.ID, compatibility.ID, planner, spareparts.VerifyInput{
		Status:        "conditional_compatible",
		ConditionNote: "Gunakan hanya untuk unit dengan pendingin tambahan",
	})
	if err != nil {
		t.Fatal(err)
	}
	if compatibility.Status != "conditional_compatible" || compatibility.ConditionNote == "" || compatibility.VerifiedBy == nil {
		t.Fatalf("verified compatibility = %+v", compatibility)
	}

	searchResults, total, err := sparepartRepo.List(ctx, spareparts.Filter{Query: "Yanmar", Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(searchResults) != 1 || searchResults[0].ID != oil.ID {
		t.Fatalf("search by brand = total %d, items %+v", total, searchResults)
	}
	searchResults, total, err = sparepartRepo.List(ctx, spareparts.Filter{
		UnitModelID:         &unitModelID,
		EngineModelID:       &engineModelID,
		CompatibilityStatus: "conditional_compatible",
		Page:                1,
		Limit:               10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(searchResults) != 1 || searchResults[0].ID != oil.ID {
		t.Fatalf("search by compatibility = total %d, items %+v", total, searchResults)
	}
	searchResults, total, err = sparepartRepo.List(ctx, spareparts.Filter{Query: "4x4", Page: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || searchResults[0].ID != oil.ID {
		t.Fatalf("search by configuration = total %d, items %+v", total, searchResults)
	}

	invRepo := inventory.NewRepository(pool, loc)
	warehouse, err := invRepo.CreateWarehouse(ctx, inventory.Actor{ID: plannerID, Role: "planner"}, inventory.WarehouseInput{
		Code: "M5-WH1", Name: "Gudang M5", SiteLocation: "Site M5",
	})
	if err != nil {
		t.Fatal(err)
	}
	location1, err := invRepo.CreateLocation(ctx, inventory.Actor{ID: plannerID, Role: "planner"}, inventory.LocationInput{
		WarehouseID: warehouse.ID, Code: "A1", Name: "Rack A1", LocationType: "rack",
	})
	if err != nil {
		t.Fatal(err)
	}
	warehouse2, err := invRepo.CreateWarehouse(ctx, inventory.Actor{ID: plannerID, Role: "planner"}, inventory.WarehouseInput{
		Code: "M5-WH2", Name: "Gudang M5 Kedua",
	})
	if err != nil {
		t.Fatal(err)
	}
	location2, err := invRepo.CreateLocation(ctx, inventory.Actor{ID: plannerID, Role: "planner"}, inventory.LocationInput{
		WarehouseID: warehouse2.ID, Code: "B1", Name: "Bin B1", LocationType: "bin",
	})
	if err != nil {
		t.Fatal(err)
	}

	policy, err := invRepo.UpsertPolicy(ctx, inventory.Actor{ID: plannerID, Role: "planner"}, inventory.PolicyInput{
		SparepartID: oil.ID, LocationID: location1.ID, MinQty: 10, CriticalQty: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if policy.MinQty != 10 || policy.CriticalQty != 5 {
		t.Fatalf("policy = %+v", policy)
	}

	storeActor := inventory.Actor{ID: storekeeperID, Role: "storekeeper"}
	plannerActor := inventory.Actor{ID: plannerID, Role: "planner"}

	receipt, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "receipt", ReferenceType: "manual", ReferenceNo: "DO-M5-1",
		Notes: "Penerimaan awal", IdempotencyKey: "m5-receipt-1",
		Items: []inventory.PostItem{{SparepartID: oil.ID, ToLocationID: &location1.ID, Qty: 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Type != "receipt" || receipt.Status != "posted" || receipt.ItemCount != 1 {
		t.Fatalf("receipt = %+v", receipt)
	}
	replayed, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "receipt", ReferenceType: "manual", ReferenceNo: "DO-M5-1",
		Notes: "Replay", IdempotencyKey: "m5-receipt-1",
		Items: []inventory.PostItem{{SparepartID: oil.ID, ToLocationID: &location1.ID, Qty: 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != receipt.ID {
		t.Fatalf("idempotent replay id = %d, want %d", replayed.ID, receipt.ID)
	}
	var idempotentCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM inventory_transactions WHERE idempotency_key = 'm5-receipt-1'
	`).Scan(&idempotentCount); err != nil {
		t.Fatal(err)
	}
	if idempotentCount != 1 {
		t.Fatalf("idempotent transaction rows = %d, want 1", idempotentCount)
	}
	mustBalance(t, ctx, pool, oil.ID, location1.ID, 20)

	if _, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "issue", ReferenceNo: "ISS-M5-1", Notes: "Pemakaian unit",
		Items: []inventory.PostItem{{SparepartID: oil.ID, FromLocationID: &location1.ID, Qty: 6}},
	}); err != nil {
		t.Fatal(err)
	}
	mustBalance(t, ctx, pool, oil.ID, location1.ID, 14)
	if _, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "issue", ReferenceNo: "ISS-M5-FAIL", Notes: "Uji stok negatif",
		Items: []inventory.PostItem{{SparepartID: oil.ID, FromLocationID: &location1.ID, Qty: 100}},
	}); !errors.Is(err, inventory.ErrInsufficientStock) {
		t.Fatalf("negative stock error = %v, want insufficient stock", err)
	}
	mustBalance(t, ctx, pool, oil.ID, location1.ID, 14)

	if _, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "return", ReferenceNo: "RET-M5-1",
		Items: []inventory.PostItem{{SparepartID: oil.ID, ToLocationID: &location1.ID, Qty: 2}},
	}); err != nil {
		t.Fatal(err)
	}
	mustBalance(t, ctx, pool, oil.ID, location1.ID, 16)
	if _, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "transfer", ReferenceNo: "TRF-M5-1",
		Items: []inventory.PostItem{{
			SparepartID: oil.ID, FromLocationID: &location1.ID,
			ToLocationID: &location2.ID, Qty: 4,
		}},
	}); err != nil {
		t.Fatal(err)
	}
	mustBalance(t, ctx, pool, oil.ID, location1.ID, 12)
	mustBalance(t, ctx, pool, oil.ID, location2.ID, 4)

	adjustment, err := invRepo.Post(ctx, plannerActor, inventory.PostInput{
		Type: "adjustment", ReferenceNo: "ADJ-M5-1", Notes: "Koreksi hasil hitung",
		Items: []inventory.PostItem{{SparepartID: oil.ID, ToLocationID: &location1.ID, Qty: 3}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mustBalance(t, ctx, pool, oil.ID, location1.ID, 15)
	if _, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "adjustment", ReferenceNo: "ADJ-M5-DENY", Notes: "Uji role",
		Items: []inventory.PostItem{{SparepartID: oil.ID, ToLocationID: &location1.ID, Qty: 1}},
	}); !errors.Is(err, inventory.ErrValidation) {
		t.Fatalf("storekeeper adjustment error = %v, want validation", err)
	}
	if _, err := invRepo.Reverse(ctx, adjustment.ID, storeActor, "Uji role reversal"); !errors.Is(err, inventory.ErrValidation) {
		t.Fatalf("storekeeper reversal error = %v, want validation", err)
	}
	reversal, err := invRepo.Reverse(ctx, adjustment.ID, plannerActor, "Salah input adjustment")
	if err != nil {
		t.Fatal(err)
	}
	if reversal.ReversalOfID == nil || *reversal.ReversalOfID != adjustment.ID {
		t.Fatalf("reversal = %+v", reversal)
	}
	mustBalance(t, ctx, pool, oil.ID, location1.ID, 12)
	var originalStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM inventory_transactions WHERE id = $1`, adjustment.ID).Scan(&originalStatus); err != nil {
		t.Fatal(err)
	}
	if originalStatus != "reversed" {
		t.Fatalf("original adjustment status = %q, want reversed", originalStatus)
	}

	reconciliation, err := invRepo.Reconcile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if reconciliation.Mismatch != 0 {
		t.Fatalf("reconciliation mismatch after reversal = %d, rows %+v", reconciliation.Mismatch, reconciliation.Rows)
	}
	movements, movementTotal, err := invRepo.ListMovements(ctx, inventory.MovementFilter{Page: 1, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if movementTotal != 6 || len(movements) != 6 {
		t.Fatalf("movements = %d/%d, want 6", movementTotal, len(movements))
	}
	balances, balanceTotal, err := invRepo.ListBalances(ctx, inventory.BalanceFilter{Query: "M5-OIL", Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if balanceTotal != 2 || len(balances) != 2 {
		t.Fatalf("balances = %d/%d, want 2", balanceTotal, len(balances))
	}

	if _, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "issue", ReferenceNo: "ISS-M5-LOW", Notes: "Turunkan ke low",
		Items: []inventory.PostItem{{SparepartID: oil.ID, FromLocationID: &location1.ID, Qty: 4}},
	}); err != nil {
		t.Fatal(err)
	}
	mustBalance(t, ctx, pool, oil.ID, location1.ID, 8)
	alertTime := time.Now().In(loc)
	alertResult, err := invRepo.SendLowStockAlerts(ctx, alertTime)
	if err != nil {
		t.Fatal(err)
	}
	if alertResult.Low != 1 || alertResult.Critical != 0 || alertResult.Notified == 0 {
		t.Fatalf("first low alert = %+v, want low 1 notified > 0", alertResult)
	}
	alertResult, err = invRepo.SendLowStockAlerts(ctx, alertTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if alertResult.Notified != 0 {
		t.Fatalf("duplicate low alert notified = %d, want 0", alertResult.Notified)
	}

	if _, err := invRepo.Post(ctx, storeActor, inventory.PostInput{
		Type: "issue", ReferenceNo: "ISS-M5-CRIT", Notes: "Turunkan ke critical",
		Items: []inventory.PostItem{{SparepartID: oil.ID, FromLocationID: &location1.ID, Qty: 4}},
	}); err != nil {
		t.Fatal(err)
	}
	mustBalance(t, ctx, pool, oil.ID, location1.ID, 4)
	alertResult, err = invRepo.SendLowStockAlerts(ctx, alertTime)
	if err != nil {
		t.Fatal(err)
	}
	if alertResult.Critical != 1 || alertResult.Low != 0 || alertResult.Notified == 0 {
		t.Fatalf("critical alert = %+v, want critical 1 notified > 0", alertResult)
	}

	opname, err := invRepo.CreateOpname(ctx, storeActor, inventory.OpnameCreateInput{
		WarehouseID: warehouse.ID, Notes: "Opname uji M5",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opname.Status != "counting" || len(opname.Items) != 1 || opname.Items[0].SystemQty != 4 {
		t.Fatalf("opname session = %+v", opname)
	}
	opname, err = invRepo.CountOpname(ctx, opname.ID, storeActor, inventory.OpnameCountInput{
		Items: []inventory.OpnameCountItem{{ItemID: opname.Items[0].ID, CountedQty: 6, Notes: "Hitung fisik"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if opname.Status != "review" || opname.CountedCount != 1 {
		t.Fatalf("counted opname = %+v", opname)
	}
	opname, err = invRepo.PostOpname(ctx, opname.ID, plannerActor)
	if err != nil {
		t.Fatal(err)
	}
	if opname.Status != "posted" {
		t.Fatalf("posted opname status = %q", opname.Status)
	}
	mustBalance(t, ctx, pool, oil.ID, location1.ID, 6)
	reconciliation, err = invRepo.Reconcile(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if reconciliation.Mismatch != 0 {
		t.Fatalf("reconciliation mismatch after opname = %d", reconciliation.Mismatch)
	}

	cancelled, err := invRepo.CreateOpname(ctx, storeActor, inventory.OpnameCreateInput{WarehouseID: warehouse2.ID})
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err = invRepo.CancelOpname(ctx, cancelled.ID, plannerActor, "Dibatalkan untuk uji")
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != "cancelled" || cancelled.CancelReason == "" {
		t.Fatalf("cancelled opname = %+v", cancelled)
	}

	var unitID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_model_id, current_hm, current_km, is_active)
		VALUES ('M5-UNIT', $1, 0, 0, true) RETURNING id
	`, unitModelID).Scan(&unitID); err != nil {
		t.Fatal(err)
	}
	var planID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO pm_plans (name, unit_id, status, effective_from, interval_days, created_by)
		VALUES ('PM M5 Material', $1, 'active', CURRENT_DATE, 30, $2)
		RETURNING id
	`, unitID, plannerID).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	pmRepo := pm.NewRepository(pool, loc)
	pmActor := pm.Actor{ID: plannerID, Role: "planner"}
	material, err := pmRepo.AddPlanMaterial(ctx, planID, pmActor, pm.PlanMaterialInput{
		SparepartID: oil.ID, Qty: 2, Notes: "Ganti filter",
	})
	if err != nil {
		t.Fatal(err)
	}
	secondMaterial, err := pmRepo.AddPlanMaterial(ctx, planID, pmActor, pm.PlanMaterialInput{
		SparepartID: alt.ID, Qty: 1, IsOptional: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	updatedQty := 3.0
	material, err = pmRepo.UpdatePlanMaterial(ctx, planID, material.ID, pmActor, pm.PlanMaterialPatch{Qty: &updatedQty})
	if err != nil {
		t.Fatal(err)
	}
	if material.Qty != 3 {
		t.Fatalf("updated material qty = %v, want 3", material.Qty)
	}
	if err := pmRepo.DeletePlanMaterial(ctx, planID, secondMaterial.ID, pmActor); err != nil {
		t.Fatal(err)
	}
	materials, err := pmRepo.ListPlanMaterials(ctx, planID)
	if err != nil {
		t.Fatal(err)
	}
	if len(materials) != 1 || materials[0].ID != material.ID {
		t.Fatalf("PM materials = %+v", materials)
	}

	active := false
	if _, err := sparepartRepo.Update(ctx, alt.ID, admin, spareparts.Patch{IsActive: &active}); err != nil {
		t.Fatal(err)
	}
	activeOnly := true
	searchResults, total, err = sparepartRepo.List(ctx, spareparts.Filter{
		Query: "M5-FLT", ActiveOnly: &activeOnly, Page: 1, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(searchResults) != 0 {
		t.Fatalf("inactive sparepart search = %d/%d, want 0", total, len(searchResults))
	}
}

func truncateM5(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			stock_opname_items, stock_opname_sessions,
			inventory_transaction_items, inventory_transactions,
			inventory_balances, inventory_stock_policies,
			warehouse_locations, warehouses,
			pm_plan_materials,
			sparepart_compatibility, sparepart_alternatives,
			sparepart_numbers, spareparts,
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
	`); err != nil {
		t.Fatal(err)
	}
}

func mustBalance(t *testing.T, ctx context.Context, pool *pgxpool.Pool, sparepartID, locationID int64, want float64) {
	t.Helper()
	var got float64
	if err := pool.QueryRow(ctx, `
		SELECT on_hand::float8
		FROM inventory_balances
		WHERE sparepart_id = $1 AND warehouse_location_id = $2
	`, sparepartID, locationID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("balance sparepart %d location %d = %v, want %v", sparepartID, locationID, got, want)
	}
}
