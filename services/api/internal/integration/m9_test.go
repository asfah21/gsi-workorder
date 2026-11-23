package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/candidates"
)

func TestM9WorkRequestLifecycleAndBacklog(t *testing.T) {
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

	var mechanicID, dispatchID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m9-mechanic', 'Mechanic M9', 'mechanic', true) RETURNING id
	`).Scan(&mechanicID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m9-dispatch', 'Dispatch M9', 'dispatch', true) RETURNING id
	`).Scan(&dispatchID); err != nil {
		t.Fatal(err)
	}

	var unitID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_type, is_active)
		VALUES ('UNIT-M9', 'dump_truck', true) RETURNING id
	`).Scan(&unitID); err != nil {
		t.Fatal(err)
	}

	service := candidates.NewService(candidates.NewRepository(pool, loc))

	created, err := service.Create(ctx, candidates.CreateInput{
		UnitID:       unitID,
		Title:        "REQ-1 Hydraulic leak",
		Description:  "Hose bocor di boom",
		WorkCategory: "Service",
		IsBreakdown:  false,
		Priority:     "high",
		ActorID:      mechanicID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != candidates.StatusPending || created.Source != "manual" || created.P2HReportID != nil {
		t.Fatalf("created candidate = %#v", created)
	}
	if created.CreatedBy == nil || *created.CreatedBy != mechanicID {
		t.Fatalf("created_by = %v, want %d", created.CreatedBy, mechanicID)
	}

	validated, err := service.Validate(ctx, created.ID, dispatchID, candidates.ValidateInput{Note: "Disetujui dispatch"})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if validated.Status != candidates.StatusValidated || validated.ValidatedBy == nil || *validated.ValidatedBy != dispatchID {
		t.Fatalf("validated candidate = %#v", validated)
	}

	converted, err := service.Convert(ctx, created.ID, dispatchID, candidates.ApproveInput{
		Priority: "high",
		Note:     "Kerjakan shift 1",
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if converted.Candidate.Status != candidates.StatusConverted {
		t.Fatalf("converted status = %q", converted.Candidate.Status)
	}
	if converted.WorkOrder.Source != "manual" || converted.WorkOrder.MaintenanceType != "corrective" {
		t.Fatalf("work order = %#v", converted.WorkOrder)
	}
	if converted.WorkOrder.P2HReportID != nil {
		t.Fatalf("manual work order has p2h_report_id = %v", converted.WorkOrder.P2HReportID)
	}

	detail, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if detail.Report != nil {
		t.Fatalf("manual detail report = %#v, want nil", detail.Report)
	}

	backlogCandidate, err := service.Create(ctx, candidates.CreateInput{
		UnitID:       unitID,
		Title:        "REQ-2 Backlog service",
		WorkCategory: "Service",
		Priority:     "medium",
		ActorID:      mechanicID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, candidates.CreateInput{
		UnitID:       unitID,
		Title:        "REQ-3 Breakdown incident",
		WorkCategory: "Incident",
		IsBreakdown:  true,
		Priority:     "critical",
		ActorID:      mechanicID,
	}); err != nil {
		t.Fatal(err)
	}
	skippedCandidate, err := service.Create(ctx, candidates.CreateInput{
		UnitID:       unitID,
		Title:        "REQ-4 Scheduled service",
		WorkCategory: "Scheduled",
		Priority:     "low",
		ActorID:      mechanicID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Validate(ctx, skippedCandidate.ID, dispatchID, candidates.ValidateInput{}); err != nil {
		t.Fatal(err)
	}
	skipped, err := service.Skip(ctx, skippedCandidate.ID, dispatchID, candidates.SkipInput{Reason: "Digabung ke WO lain"})
	if err != nil {
		t.Fatalf("Skip() error = %v", err)
	}
	if skipped.Status != candidates.StatusSkipped {
		t.Fatalf("skipped status = %q", skipped.Status)
	}

	backlog, total, err := service.List(ctx, candidates.Filter{
		Backlog: true,
		Page:    1,
		Limit:   20,
	})
	if err != nil {
		t.Fatalf("List(backlog) error = %v", err)
	}
	if total != 1 || len(backlog) != 1 || backlog[0].ID != backlogCandidate.ID {
		t.Fatalf("backlog total/items = %d/%#v, want only %d", total, backlog, backlogCandidate.ID)
	}
}
