package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/candidates"
	"gsi-workorder/internal/notifications"
	"gsi-workorder/internal/p2h"
)

func TestM2P2HFlow(t *testing.T) {
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
	truncateM2(t, ctx, pool)

	loc, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		t.Fatal(err)
	}
	var adminID, plannerID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active) VALUES ('admin', 'Admin', 'admin', true)
		RETURNING id
	`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active) VALUES ('planner', 'Planner', 'planner', true)
		RETURNING id
	`).Scan(&plannerID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO app_settings (key, value, updated_at) VALUES
		('p2h_auto_wo', 'false', now()),
		('candidate_sla_hours', '24', now()),
		('candidate_backfill_days', '30', now())
	`); err != nil {
		t.Fatal(err)
	}

	repo := p2h.NewRepository(pool, loc)
	candidateRepo := candidates.NewRepository(pool, loc)
	notificationRepo := notifications.NewRepository(pool)
	today := time.Now().In(loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	report := p2h.Report{
		OdooReportID:   1001,
		Name:           "P2H-1001",
		ReportDate:     &today,
		OdooUnitID:     int64Ptr(1438),
		UnitCode:       "LV.02.011",
		UnitType:       "light_vehicle",
		State:          "submitted",
		ApprovalStatus: "pending",
		CheckedItems:   3,
		Raw:            json.RawMessage(`{"id":1001}`),
		Findings: []p2h.Finding{
			{OdooLineID: 9001, Component: "Lampu", Category: "KELILING", CategoryNorm: "keliling", Condition: "rusak"},
			{OdooLineID: 9002, Component: "Toolkit", Category: "PERLENGKAPAN", CategoryNorm: "perlengkapan", Condition: "tidak_ada"},
		},
	}
	metadata, err := repo.UpsertMetadata(ctx, report)
	if err != nil {
		t.Fatal(err)
	}
	if !metadata.NeedsDetail {
		t.Fatal("new report should need detail")
	}
	settings, err := repo.LoadSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := repo.SaveDetail(ctx, report, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if saved.Candidate == nil || saved.Candidate.ID == 0 {
		t.Fatal("candidate was not created")
	}
	assertCount(t, ctx, pool, "work_orders", 0)
	assertCount(t, ctx, pool, "wo_candidates", 1)
	assertCandidateStatus(t, ctx, pool, saved.Candidate.ID, "pending")

	// Syncing the same report again must not duplicate candidate or findings.
	if _, err := repo.UpsertMetadata(ctx, report); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SaveDetail(ctx, report, settings, time.Now()); err != nil {
		t.Fatal(err)
	}
	assertCount(t, ctx, pool, "wo_candidates", 1)
	assertCount(t, ctx, pool, "p2h_findings", 2)

	approved, err := candidateRepo.Approve(ctx, saved.Candidate.ID, plannerID, candidates.ApproveInput{})
	if err != nil {
		t.Fatal(err)
	}
	if approved.WorkOrder.Code == "" {
		t.Fatal("approved work order has no code")
	}
	assertCount(t, ctx, pool, "work_orders", 1)
	assertCount(t, ctx, pool, "wo_findings", 2)
	assertCandidateStatus(t, ctx, pool, saved.Candidate.ID, "approved")
	if _, err := candidateRepo.Approve(ctx, saved.Candidate.ID, plannerID, candidates.ApproveInput{}); !errors.Is(err, candidates.ErrConflict) {
		t.Fatalf("duplicate approve error = %v, want conflict", err)
	}

	second := makeReport(1002, 9003, "Ban", "rusak", &today, "TRK.01")
	secondMeta, err := repo.UpsertMetadata(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	if !secondMeta.NeedsDetail {
		t.Fatal("second report should need detail")
	}
	secondSaved, err := repo.SaveDetail(ctx, second, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := candidateRepo.Skip(ctx, secondSaved.Candidate.ID, plannerID, candidates.SkipInput{Reason: "no"}); !errors.Is(err, candidates.ErrValidation) {
		t.Fatalf("short skip reason error = %v, want validation", err)
	}
	if _, err := candidateRepo.Skip(ctx, secondSaved.Candidate.ID, plannerID, candidates.SkipInput{Reason: "tidak relevan"}); err != nil {
		t.Fatal(err)
	}
	assertCandidateStatus(t, ctx, pool, secondSaved.Candidate.ID, "skipped")
	assertCount(t, ctx, pool, "work_orders", 1)

	// Auto-WO is opt-in and must only affect new candidates.
	if _, err := pool.Exec(ctx, `UPDATE app_settings SET value = 'true' WHERE key = 'p2h_auto_wo'`); err != nil {
		t.Fatal(err)
	}
	settings, err = repo.LoadSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	auto := makeReport(1003, 9004, "Rem", "tidak_ada", &today, "TRK.02")
	if _, err := repo.UpsertMetadata(ctx, auto); err != nil {
		t.Fatal(err)
	}
	autoSaved, err := repo.SaveDetail(ctx, auto, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if autoSaved.Candidate == nil || !autoSaved.Candidate.AutoWO {
		t.Fatal("auto-WO candidate was not created")
	}
	assertCandidateStatus(t, ctx, pool, autoSaved.Candidate.ID, "auto_created")
	assertCount(t, ctx, pool, "work_orders", 2)
	if _, err := pool.Exec(ctx, `UPDATE app_settings SET value = 'false' WHERE key = 'p2h_auto_wo'`); err != nil {
		t.Fatal(err)
	}

	// Old reports are archived without notifications.
	settings, err = repo.LoadSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	oldDate := today.AddDate(0, 0, -31)
	oldReport := makeReport(1004, 9005, "Kaca", "rusak", &oldDate, "TRK.03")
	if _, err := repo.UpsertMetadata(ctx, oldReport); err != nil {
		t.Fatal(err)
	}
	oldSaved, err := repo.SaveDetail(ctx, oldReport, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if oldSaved.Candidate == nil || !oldSaved.Candidate.Archived {
		t.Fatal("old report should produce an archived candidate")
	}
	assertCandidateStatus(t, ctx, pool, oldSaved.Candidate.ID, "archived")

	// Reminder is sent once after SLA.
	third := makeReport(1005, 9006, "Filter", "rusak", &today, "TRK.04")
	if _, err := repo.UpsertMetadata(ctx, third); err != nil {
		t.Fatal(err)
	}
	thirdSaved, err := repo.SaveDetail(ctx, third, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE wo_candidates SET created_at = now() - interval '2 days' WHERE id = $1`, thirdSaved.Candidate.ID); err != nil {
		t.Fatal(err)
	}
	candidatesNotified, notificationCount, err := repo.SendCandidateReminders(ctx, time.Now(), 24)
	if err != nil {
		t.Fatal(err)
	}
	if candidatesNotified != 1 || notificationCount != 2 {
		t.Fatalf("reminder counts = %d/%d, want 1/2", candidatesNotified, notificationCount)
	}
	candidatesNotified, _, err = repo.SendCandidateReminders(ctx, time.Now(), 24)
	if err != nil {
		t.Fatal(err)
	}
	if candidatesNotified != 0 {
		t.Fatalf("second reminder count = %d, want 0", candidatesNotified)
	}

	// Bulk validation and atomic approval.
	fourth := makeReport(1006, 9007, "Aki", "rusak", &today, "TRK.05")
	if _, err := repo.UpsertMetadata(ctx, fourth); err != nil {
		t.Fatal(err)
	}
	fourthSaved, err := repo.SaveDetail(ctx, fourth, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	tooMany := make([]int64, 51)
	for i := range tooMany {
		tooMany[i] = int64(i + 1)
	}
	if _, err := candidateRepo.Bulk(ctx, plannerID, candidates.BulkInput{IDs: tooMany, Action: "approve"}); !errors.Is(err, candidates.ErrValidation) {
		t.Fatalf("bulk 51 error = %v, want validation", err)
	}
	bulk, err := candidateRepo.Bulk(ctx, plannerID, candidates.BulkInput{
		IDs:    []int64{thirdSaved.Candidate.ID, fourthSaved.Candidate.ID},
		Action: "approve",
		Note:   "bulk QA",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bulk.Approved) != 2 {
		t.Fatalf("bulk approved = %d, want 2", len(bulk.Approved))
	}
	assertCount(t, ctx, pool, "work_orders", 4)
	if _, err := candidateRepo.Bulk(ctx, plannerID, candidates.BulkInput{
		IDs:    []int64{thirdSaved.Candidate.ID, fourthSaved.Candidate.ID},
		Action: "approve",
	}); !errors.Is(err, candidates.ErrConflict) {
		t.Fatalf("repeat bulk error = %v, want conflict", err)
	}

	items, total, unread, err := notificationRepo.List(ctx, adminID, 1, 20, false)
	if err != nil {
		t.Fatal(err)
	}
	if total == 0 || unread == 0 || len(items) == 0 {
		t.Fatalf("notification list total/unread/items = %d/%d/%d", total, unread, len(items))
	}
	if _, err := notificationRepo.MarkRead(ctx, adminID, nil, true); err != nil {
		t.Fatal(err)
	}
	_, _, unread, err = notificationRepo.List(ctx, adminID, 1, 20, false)
	if err != nil {
		t.Fatal(err)
	}
	if unread != 0 {
		t.Fatalf("unread after mark all = %d, want 0", unread)
	}

	// A report with no findings must not create a candidate and must still
	// record the number of checked checklist items.
	clean := p2h.Report{
		OdooReportID:   1008,
		Name:           "P2H-CLEAN",
		ReportDate:     &today,
		OdooUnitID:     int64Ptr(1008),
		UnitCode:       "TRK.06",
		UnitType:       "heavy_vehicle",
		State:          "submitted",
		ApprovalStatus: "pending",
		CheckedItems:   5,
		Raw:            json.RawMessage(`{"id":1008}`),
	}
	if _, err := repo.UpsertMetadata(ctx, clean); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SaveDetail(ctx, clean, settings, time.Now()); err != nil {
		t.Fatal(err)
	}
	assertCount(t, ctx, pool, "wo_candidates", 6)
	var checked, findings int
	if err := pool.QueryRow(ctx, `
		SELECT checked_items, findings_count FROM p2h_reports WHERE odoo_report_id = 1008
	`).Scan(&checked, &findings); err != nil {
		t.Fatal(err)
	}
	if checked != 5 || findings != 0 {
		t.Fatalf("clean report counts = %d/%d, want 5/0", checked, findings)
	}
}

type fakeP2HClient struct {
	reports []p2h.Report
	details map[int64]p2h.Report
}

func (f *fakeP2HClient) Configured() bool { return true }

func (f *fakeP2HClient) List(_ context.Context, limit, offset int) (p2h.ListPage, error) {
	if offset >= len(f.reports) {
		return p2h.ListPage{Reports: []p2h.Report{}, Total: len(f.reports)}, nil
	}
	end := offset + limit
	if end > len(f.reports) {
		end = len(f.reports)
	}
	return p2h.ListPage{Reports: f.reports[offset:end], Total: len(f.reports)}, nil
}

func (f *fakeP2HClient) Get(_ context.Context, id int64) (p2h.Report, error) {
	report, ok := f.details[id]
	if !ok {
		return p2h.Report{}, fmt.Errorf("missing report %d", id)
	}
	return report, nil
}

func TestM2SyncServiceIdempotent(t *testing.T) {
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
	truncateM2(t, ctx, pool)

	loc, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"admin", "planner"} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO users (nik, name, role, is_active) VALUES ($1, $1, $1, true)
		`, role); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO app_settings (key, value, updated_at) VALUES
		('p2h_auto_wo', 'false', now()),
		('candidate_sla_hours', '24', now()),
		('candidate_backfill_days', '30', now())
	`); err != nil {
		t.Fatal(err)
	}

	today := time.Now().In(loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	reports := []p2h.Report{
		makeReport(7001, 91001, "Lampu", "rusak", &today, "SYNC.01"),
		makeReport(7002, 91002, "Ban", "tidak_ada", &today, "SYNC.02"),
	}
	details := map[int64]p2h.Report{}
	for _, report := range reports {
		detail := report
		detail.CheckedItems = 4
		details[report.OdooReportID] = detail
	}
	client := &fakeP2HClient{reports: reports, details: details}
	repo := p2h.NewRepository(pool, loc)
	service := p2h.NewSyncService(client, repo, 2, loc, nil)

	if err := service.Sync(ctx, false); err != nil {
		t.Fatal(err)
	}
	assertCount(t, ctx, pool, "p2h_reports", 2)
	assertCount(t, ctx, pool, "p2h_findings", 2)
	assertCount(t, ctx, pool, "wo_candidates", 2)
	assertCount(t, ctx, pool, "work_orders", 0)
	var successRuns int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sync_runs WHERE status = 'success'`).Scan(&successRuns); err != nil {
		t.Fatal(err)
	}
	if successRuns != 1 {
		t.Fatalf("success sync runs = %d, want 1", successRuns)
	}

	if err := service.Sync(ctx, false); err != nil {
		t.Fatal(err)
	}
	assertCount(t, ctx, pool, "p2h_reports", 2)
	assertCount(t, ctx, pool, "p2h_findings", 2)
	assertCount(t, ctx, pool, "wo_candidates", 2)
	assertCount(t, ctx, pool, "work_orders", 0)

	if err := service.Sync(ctx, true); err != nil {
		t.Fatal(err)
	}
	assertCount(t, ctx, pool, "wo_candidates", 2)
	assertCount(t, ctx, pool, "work_orders", 0)
}

func makeReport(id, lineID int64, component, condition string, date *time.Time, unitCode string) p2h.Report {
	return p2h.Report{
		OdooReportID:   id,
		Name:           "P2H-" + time.Unix(id, 0).Format("150405"),
		ReportDate:     date,
		OdooUnitID:     int64Ptr(id),
		UnitCode:       unitCode,
		UnitType:       "heavy_vehicle",
		State:          "submitted",
		ApprovalStatus: "pending",
		CheckedItems:   1,
		Raw:            json.RawMessage(fmt.Sprintf(`{"id":%d}`, id)),
		Findings: []p2h.Finding{
			{OdooLineID: lineID, Component: component, Category: "TEST", CategoryNorm: "test", Condition: condition},
		},
	}
}

func truncateM2(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			notifications, wo_findings, work_orders, wo_candidates,
			p2h_findings, p2h_reports, unit_meter_readings, units,
			app_settings_history, app_settings, sync_runs, sync_state,
			work_order_code_counters, users
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func assertCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}

func assertCandidateStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id int64, want string) {
	t.Helper()
	var got string
	if err := pool.QueryRow(ctx, `SELECT status FROM wo_candidates WHERE id = $1`, id).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("candidate %d status = %q, want %q", id, got, want)
	}
}

func int64Ptr(value int64) *int64 { return &value }
