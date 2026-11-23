package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/p2h"
)

func TestM8CP2HReadingsIngestionAndBackfill(t *testing.T) {
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

	var adminID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m8c-admin', 'Admin M8C', 'admin', true) RETURNING id
	`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}

	loc, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		t.Fatal(err)
	}
	repo := p2h.NewRepository(pool, loc)

	reportDate := time.Date(2026, 11, 20, 0, 0, 0, 0, loc)
	hm := 120.5
	km := 5000.0
	report := p2h.Report{
		OdooReportID: 9001,
		Name:         "P2H-M8C-1",
		ReportDate:   &reportDate,
		UnitCode:     "UNIT-M8C",
		UnitType:     "dump_truck",
		HMAwal:       &hm,
		KMAkhir:      &km,
	}
	if _, err := repo.UpsertMetadata(ctx, report); err != nil {
		t.Fatalf("UpsertMetadata() error = %v", err)
	}

	var unitID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM units WHERE code = 'UNIT-M8C'`).Scan(&unitID); err != nil {
		t.Fatal(err)
	}
	assertUnitReading(t, ctx, pool, unitID, 120.5, 5000)
	assertReadingCount(t, ctx, pool, unitID, 2)

	// Re-syncing the same report must not duplicate readings.
	if _, err := repo.UpsertMetadata(ctx, report); err != nil {
		t.Fatalf("UpsertMetadata() retry error = %v", err)
	}
	assertReadingCount(t, ctx, pool, unitID, 2)

	// A corrected P2H value updates the reading and the unit cache.
	hm = 130
	km = 5100
	report.HMAwal = &hm
	report.KMAkhir = &km
	if _, err := repo.UpsertMetadata(ctx, report); err != nil {
		t.Fatalf("UpsertMetadata() correction error = %v", err)
	}
	assertUnitReading(t, ctx, pool, unitID, 130, 5100)
	assertReadingCount(t, ctx, pool, unitID, 2)

	// Backfill an older report. It must create readings but must not overwrite
	// the newer current reading.
	oldDate := time.Date(2026, 11, 19, 0, 0, 0, 0, loc)
	if _, err := pool.Exec(ctx, `
		INSERT INTO p2h_reports (odoo_report_id, name, report_date, unit_id, unit_type, hm_awal, km_akhir)
		VALUES (9000, 'P2H-M8C-OLD', $1, $2, 'dump_truck', 100, 4000)
	`, oldDate, unitID); err != nil {
		t.Fatal(err)
	}
	result, err := repo.BackfillReadings(ctx)
	if err != nil {
		t.Fatalf("BackfillReadings() error = %v", err)
	}
	if result.Reports != 2 {
		t.Fatalf("backfill reports = %d, want 2", result.Reports)
	}
	assertUnitReading(t, ctx, pool, unitID, 130, 5100)
	assertReadingCount(t, ctx, pool, unitID, 4)

	// Backfill a newer report. It becomes the current reading.
	newDate := time.Date(2026, 11, 21, 0, 0, 0, 0, loc)
	if _, err := pool.Exec(ctx, `
		INSERT INTO p2h_reports (odoo_report_id, name, report_date, unit_id, unit_type, hm_awal, km_akhir)
		VALUES (9002, 'P2H-M8C-NEW', $1, $2, 'dump_truck', 150, 5300)
	`, newDate, unitID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.BackfillReadings(ctx); err != nil {
		t.Fatalf("BackfillReadings() newer error = %v", err)
	}
	assertUnitReading(t, ctx, pool, unitID, 150, 5300)
	assertReadingCount(t, ctx, pool, unitID, 6)

	// Running backfill again is idempotent and writes no changed readings.
	result, err = repo.BackfillReadings(ctx)
	if err != nil {
		t.Fatalf("BackfillReadings() idempotent error = %v", err)
	}
	if result.Readings != 0 {
		t.Fatalf("idempotent backfill readings written = %d, want 0", result.Readings)
	}
	assertReadingCount(t, ctx, pool, unitID, 6)

	// A lower P2H reading is stored as an anomaly, does not change current
	// HM/KM, and produces an admin summary notification.
	anomalyDate := time.Date(2026, 11, 22, 0, 0, 0, 0, loc)
	if _, err := pool.Exec(ctx, `
		INSERT INTO p2h_reports (odoo_report_id, name, report_date, unit_id, unit_type, hm_awal, km_akhir)
		VALUES (9004, 'P2H-M8C-ANOMALY', $1, $2, 'dump_truck', 140, 5200)
	`, anomalyDate, unitID); err != nil {
		t.Fatal(err)
	}
	result, err = repo.BackfillReadings(ctx)
	if err != nil {
		t.Fatalf("BackfillReadings() anomaly error = %v", err)
	}
	if result.Anomalies != 2 {
		t.Fatalf("backfill anomalies = %d, want 2", result.Anomalies)
	}
	assertUnitReading(t, ctx, pool, unitID, 150, 5300)
	assertReadingCount(t, ctx, pool, unitID, 8)
	assertAnomalyCount(t, ctx, pool, 9004, 2)
	var summaryCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM notifications
		WHERE user_id = $1 AND type = 'p2h_reading_anomaly_summary'
	`, adminID).Scan(&summaryCount); err != nil {
		t.Fatal(err)
	}
	if summaryCount != 1 {
		t.Fatalf("anomaly summary notifications = %d, want 1", summaryCount)
	}

	// A lower reading arriving through normal P2H sync follows the same rule
	// and notifies admins per reading type.
	syncAnomalyDate := time.Date(2026, 11, 23, 0, 0, 0, 0, loc)
	syncHM := 140.0
	syncKM := 5200.0
	if _, err := repo.UpsertMetadata(ctx, p2h.Report{
		OdooReportID: 9005,
		Name:         "P2H-M8C-SYNC-ANOMALY",
		ReportDate:   &syncAnomalyDate,
		UnitCode:     "UNIT-M8C",
		UnitType:     "dump_truck",
		HMAwal:       &syncHM,
		KMAkhir:      &syncKM,
	}); err != nil {
		t.Fatalf("UpsertMetadata(sync anomaly) error = %v", err)
	}
	assertUnitReading(t, ctx, pool, unitID, 150, 5300)
	assertAnomalyCount(t, ctx, pool, 9005, 2)
	var perReportCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM notifications
		WHERE user_id = $1 AND type = 'p2h_reading_anomaly'
	`, adminID).Scan(&perReportCount); err != nil {
		t.Fatal(err)
	}
	if perReportCount != 2 {
		t.Fatalf("per-report anomaly notifications = %d, want 2", perReportCount)
	}

	// Clearing P2H values removes the readings and resets the cache when no
	// other reading exists for that unit.
	clearDate := time.Date(2026, 11, 22, 0, 0, 0, 0, loc)
	clearHM := 50.0
	clearKM := 500.0
	clearReport := p2h.Report{
		OdooReportID: 9003,
		Name:         "P2H-M8C-CLEAR",
		ReportDate:   &clearDate,
		UnitCode:     "UNIT-M8C-CLEAR",
		UnitType:     "dump_truck",
		HMAwal:       &clearHM,
		KMAkhir:      &clearKM,
	}
	if _, err := repo.UpsertMetadata(ctx, clearReport); err != nil {
		t.Fatalf("UpsertMetadata(clear) error = %v", err)
	}
	var clearUnitID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM units WHERE code = 'UNIT-M8C-CLEAR'`).Scan(&clearUnitID); err != nil {
		t.Fatal(err)
	}
	assertUnitReading(t, ctx, pool, clearUnitID, 50, 500)
	assertReadingCount(t, ctx, pool, clearUnitID, 2)

	clearReport.HMAwal = nil
	clearReport.KMAkhir = nil
	if _, err := repo.UpsertMetadata(ctx, clearReport); err != nil {
		t.Fatalf("UpsertMetadata(clear nil) error = %v", err)
	}
	assertUnitReading(t, ctx, pool, clearUnitID, 0, 0)
	assertReadingCount(t, ctx, pool, clearUnitID, 0)
}

func assertUnitReading(t *testing.T, ctx context.Context, pool *pgxpool.Pool, unitID int64, wantHM, wantKM float64) {
	t.Helper()
	var hm, km float64
	if err := pool.QueryRow(ctx, `
		SELECT current_hm, current_km FROM units WHERE id = $1
	`, unitID).Scan(&hm, &km); err != nil {
		t.Fatal(err)
	}
	if hm != wantHM || km != wantKM {
		t.Fatalf("unit reading = %v/%v, want %v/%v", hm, km, wantHM, wantKM)
	}
}

func assertReadingCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, unitID int64, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM unit_meter_readings
		WHERE unit_id = $1 AND source_type = 'p2h'
	`, unitID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("P2H reading count = %d, want %d", count, want)
	}
}

func assertAnomalyCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, odooReportID int64, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM unit_meter_readings r
		JOIN p2h_reports p ON p.id = r.source_id
		WHERE r.source_type = 'p2h'
		  AND p.odoo_report_id = $1
		  AND r.is_anomaly = true
	`, odooReportID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("P2H anomaly count for report %d = %d, want %d", odooReportID, count, want)
	}
}
