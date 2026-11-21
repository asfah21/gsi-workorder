package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/pm"
	"gsi-workorder/internal/workorders"
)

func TestM4PMFlow(t *testing.T) {
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
	truncateM4(t, ctx, pool)

	loc, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		t.Fatal(err)
	}
	var adminID, plannerID, mechanicID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m4-admin', 'Admin M4', 'admin', true) RETURNING id
	`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m4-planner', 'Planner M4', 'planner', true) RETURNING id
	`).Scan(&plannerID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m4-mechanic', 'Mechanic M4', 'mechanic', true) RETURNING id
	`).Scan(&mechanicID); err != nil {
		t.Fatal(err)
	}
	var modelID, unitID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO unit_models (manufacturer, model, is_active)
		VALUES ('Test', 'M4', true) RETURNING id
	`).Scan(&modelID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_model_id, current_hm, current_km, is_active)
		VALUES ('M4-UNIT', $1, 0, 0, true) RETURNING id
	`, modelID).Scan(&unitID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO app_settings (key, value, updated_at) VALUES
		('pm_auto_wo', 'false', now()),
		('pm_upcoming_days', '30', now()),
		('pm_due_soon_days', '7', now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`); err != nil {
		t.Fatal(err)
	}

	repo := pm.NewRepository(pool, loc)
	woRepo := workorders.NewRepository(pool, loc)
	today := time.Now().In(loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	intervalHM := 100.0
	intervalKM := 1000.0
	intervalDays := 30
	plan, err := repo.CreatePlan(ctx, pm.Actor{ID: plannerID, Role: "planner"}, pm.PlanInput{
		Name:                "PM Test Unit",
		UnitID:              &unitID,
		Status:              "active",
		EffectiveFrom:       today,
		IntervalHM:          &intervalHM,
		IntervalKM:          &intervalKM,
		IntervalDays:        &intervalDays,
		IntervalRule:        "first_reached",
		DefaultPriority:     "high",
		DefaultAssigneeID:   &mechanicID,
		DefaultSupervisorID: &plannerID,
		Checklist: []pm.ChecklistInput{
			{Sequence: 1, Task: "Cek oli", Standard: "Level normal"},
			{Sequence: 2, Task: "Cek filter", Standard: "Bersih"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Checklist) != 2 {
		t.Fatalf("checklist count = %d, want 2", len(plan.Checklist))
	}
	updatedHM := 120.0
	plan, err = repo.UpdatePlan(ctx, plan.ID, pm.Actor{ID: plannerID, Role: "planner"}, pm.PlanPatch{IntervalHM: &updatedHM})
	if err != nil {
		t.Fatal(err)
	}
	if plan.IntervalHM == nil || *plan.IntervalHM != 120 {
		t.Fatalf("updated interval HM = %v, want 120", plan.IntervalHM)
	}
	schedules, total, err := repo.ListSchedules(ctx, pm.ScheduleFilter{
		PlanID:     &plan.ID,
		ActiveOnly: true,
		Page:       1,
		Limit:      10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(schedules) != 1 {
		t.Fatalf("schedules = %d/%d, want 1", total, len(schedules))
	}
	if schedules[0].DueHM == nil || *schedules[0].DueHM != 120 {
		t.Fatalf("due HM = %v, want 120", schedules[0].DueHM)
	}
	if schedules[0].DueKM == nil || *schedules[0].DueKM != 1000 {
		t.Fatalf("due KM = %v, want 1000", schedules[0].DueKM)
	}

	generated, err := repo.GenerateWO(ctx, schedules[0].ID, pm.Actor{ID: plannerID, Role: "planner"})
	if err != nil {
		t.Fatal(err)
	}
	if generated.WorkOrder.Source != "pm" || generated.WorkOrder.MaintenanceType != "preventive" {
		t.Fatalf("unexpected generated WO: %+v", generated.WorkOrder)
	}
	detail, err := woRepo.Get(ctx, generated.WorkOrder.ID, workorders.Actor{ID: plannerID, Role: "planner"})
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Checklist) != 2 {
		t.Fatalf("WO checklist count = %d, want 2", len(detail.Checklist))
	}

	if _, err := woRepo.Update(ctx, detail.ID, workorders.Actor{ID: adminID, Role: "admin"}, workorders.PatchInput{
		Status:     "in_progress",
		AssigneeID: &mechanicID,
	}); err != nil {
		t.Fatal(err)
	}
	for _, item := range detail.Checklist {
		if _, err := woRepo.UpdateChecklistItem(ctx, detail.ID, item.ID, workorders.Actor{ID: mechanicID, Role: "mechanic"}, "ok"); err != nil {
			t.Fatal(err)
		}
	}
	hm := 110.0
	km := 1100.0
	if _, err := woRepo.Update(ctx, detail.ID, workorders.Actor{ID: mechanicID, Role: "mechanic"}, workorders.PatchInput{
		Status:    "done",
		Note:      "PM selesai tanpa reading lengkap",
		HMReading: &hm,
	}); err == nil {
		t.Fatal("PM completion without required KM reading must be rejected")
	}
	completed, err := woRepo.Update(ctx, detail.ID, workorders.Actor{ID: mechanicID, Role: "mechanic"}, workorders.PatchInput{
		Status:    "done",
		Note:      "PM selesai dengan reading",
		HMReading: &hm,
		KMReading: &km,
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "done" {
		t.Fatalf("status = %q, want done", completed.Status)
	}
	var currentHM, currentKM float64
	if err := pool.QueryRow(ctx, `SELECT current_hm, current_km FROM units WHERE id = $1`, unitID).Scan(&currentHM, &currentKM); err != nil {
		t.Fatal(err)
	}
	if currentHM != 110 || currentKM != 1100 {
		t.Fatalf("unit readings = %v/%v, want 110/1100", currentHM, currentKM)
	}
	var nextDueHM, nextDueKM float64
	if err := pool.QueryRow(ctx, `
		SELECT due_hm, due_km FROM pm_schedules
		WHERE pm_plan_id = $1 AND unit_id = $2 AND cycle_no = 2
	`, plan.ID, unitID).Scan(&nextDueHM, &nextDueKM); err != nil {
		t.Fatal(err)
	}
	if nextDueHM != 230 || nextDueKM != 2100 {
		t.Fatalf("next due = %v/%v, want 230/2100", nextDueHM, nextDueKM)
	}
	compliance, err := repo.Compliance(ctx, pm.ComplianceFilter{PlanID: &plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if compliance.Completed != 1 || compliance.Total != 2 {
		t.Fatalf("compliance = %+v, want completed 1 of 2", compliance)
	}

	var unit3ID, unit4ID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_model_id, current_hm, current_km, is_active)
		VALUES ('M4-OVERRIDE', $1, 0, 0, true) RETURNING id
	`, modelID).Scan(&unit3ID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_model_id, current_hm, current_km, is_active)
		VALUES ('M4-CANCEL', $1, 0, 0, true) RETURNING id
	`, modelID).Scan(&unit4ID); err != nil {
		t.Fatal(err)
	}
	fiveDays := 5
	plan3, err := repo.CreatePlan(ctx, pm.Actor{ID: plannerID, Role: "planner"}, pm.PlanInput{
		Name:          "PM Override Test",
		UnitID:        &unit3ID,
		Status:        "active",
		EffectiveFrom: today,
		IntervalDays:  &fiveDays,
		IntervalRule:  "first_reached",
	})
	if err != nil {
		t.Fatal(err)
	}
	plan3Schedules, _, err := repo.ListSchedules(ctx, pm.ScheduleFilter{PlanID: &plan3.ID, ActiveOnly: true, Page: 1, Limit: 10})
	if err != nil || len(plan3Schedules) != 1 {
		t.Fatalf("override schedule: %v/%d", err, len(plan3Schedules))
	}
	overrideDate := today.AddDate(0, 0, 10)
	overridden, err := repo.OverrideDue(ctx, plan3Schedules[0].ID, pm.Actor{ID: plannerID, Role: "planner"}, pm.OverrideInput{
		DueDate: &overrideDate,
		Reason:  "koreksi jadwal operasional",
	})
	if err != nil {
		t.Fatal(err)
	}
	if overridden.OverrideReason == "" || overridden.DueDate == nil || !overridden.DueDate.Equal(overrideDate) {
		t.Fatalf("override not persisted: %+v", overridden)
	}
	skipped, err := repo.Skip(ctx, plan3Schedules[0].ID, pm.Actor{ID: plannerID, Role: "planner"}, "unit belum siap")
	if err != nil {
		t.Fatal(err)
	}
	if skipped.Status != "skipped" || skipped.SkipReason == "" {
		t.Fatalf("skip not persisted: %+v", skipped)
	}

	plan4, err := repo.CreatePlan(ctx, pm.Actor{ID: plannerID, Role: "planner"}, pm.PlanInput{
		Name:          "PM Cancel Test",
		UnitID:        &unit4ID,
		Status:        "active",
		EffectiveFrom: today,
		IntervalDays:  &fiveDays,
		IntervalRule:  "first_reached",
	})
	if err != nil {
		t.Fatal(err)
	}
	plan4Schedules, _, err := repo.ListSchedules(ctx, pm.ScheduleFilter{PlanID: &plan4.ID, ActiveOnly: true, Page: 1, Limit: 10})
	if err != nil || len(plan4Schedules) != 1 {
		t.Fatalf("cancel schedule: %v/%d", err, len(plan4Schedules))
	}
	cancelled, err := repo.Cancel(ctx, plan4Schedules[0].ID, pm.Actor{ID: plannerID, Role: "planner"}, "dibatalkan planner")
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != "cancelled" || cancelled.CancelReason == "" {
		t.Fatalf("cancel not persisted: %+v", cancelled)
	}

	modelPlan, err := repo.CreatePlan(ctx, pm.Actor{ID: plannerID, Role: "planner"}, pm.PlanInput{
		Name:          "PM Model Test",
		UnitModelID:   &modelID,
		Status:        "active",
		EffectiveFrom: today,
		IntervalDays:  &fiveDays,
		IntervalRule:  "first_reached",
	})
	if err != nil {
		t.Fatal(err)
	}
	modelSchedules, modelTotal, err := repo.ListSchedules(ctx, pm.ScheduleFilter{
		PlanID: &modelPlan.ID, ActiveOnly: true, Page: 1, Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if modelTotal != 3 || len(modelSchedules) != 3 {
		t.Fatalf("model plan schedules = %d/%d, want 3", modelTotal, len(modelSchedules))
	}

	// Auto-WO must stay off while the global flag is false, then create exactly
	// one WO when both controls are enabled.
	var unit2ID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_model_id, current_hm, current_km, is_active)
		VALUES ('M4-AUTO', $1, 0, 0, true) RETURNING id
	`, modelID).Scan(&unit2ID); err != nil {
		t.Fatal(err)
	}
	past := today.AddDate(0, 0, -10)
	oneDay := 1
	autoPlan, err := repo.CreatePlan(ctx, pm.Actor{ID: plannerID, Role: "planner"}, pm.PlanInput{
		Name:            "PM Auto Test",
		UnitID:          &unit2ID,
		Status:          "active",
		EffectiveFrom:   past,
		IntervalDays:    &oneDay,
		IntervalRule:    "first_reached",
		DefaultPriority: "medium",
		AutoGenerateWO:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	evaluation, err := repo.EvaluateDue(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.AutoGenerated != 0 {
		t.Fatalf("auto-generated = %d, want 0 while global flag is false", evaluation.AutoGenerated)
	}
	if _, err := pool.Exec(ctx, `UPDATE app_settings SET value = 'true' WHERE key = 'pm_auto_wo'`); err != nil {
		t.Fatal(err)
	}
	evaluation, err = repo.EvaluateDue(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.AutoGenerated != 1 {
		t.Fatalf("auto-generated = %d, want 1 with both controls on", evaluation.AutoGenerated)
	}
	var autoWO int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM work_orders w
		JOIN pm_schedules s ON s.id = w.pm_schedule_id
		WHERE s.pm_plan_id = $1 AND w.source = 'pm'
	`, autoPlan.ID).Scan(&autoWO); err != nil {
		t.Fatal(err)
	}
	if autoWO != 1 {
		t.Fatalf("auto PM WOs = %d, want 1", autoWO)
	}
}

func truncateM4(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			work_order_checklist_items,
			wo_updates, wo_attachments, wo_findings, wo_candidates,
			notifications, work_orders, pm_schedules, pm_plan_checklist_items,
			pm_plans, unit_meter_readings, units, unit_models, engine_models,
			p2h_findings, p2h_reports, work_order_code_counters,
			app_settings_history, app_settings, users
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatal(err)
	}
}
