package p2h

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"gsi-workorder/internal/notifications"
)

const p2hReadingMax = 99999999999.9

// ReadingBackfillResult summarizes the DB-only backfill command.
type ReadingBackfillResult struct {
	Reports   int64
	Readings  int64
	Units     int64
	Anomalies int64
}

// ingestReportReadings upserts the HM/KM readings carried by one P2H report.
// It runs inside the same transaction as UpsertMetadata so a report and its
// readings cannot diverge. The unique index on
// (unit_id, reading_type, source_type, source_id) makes retries idempotent.
func (r *Repository) ingestReportReadings(ctx context.Context, tx pgx.Tx, reportID int64, unitID *int64, reportDate *time.Time, hm, km *float64) error {
	if unitID == nil {
		return r.deleteP2HReadingsForSource(ctx, tx, reportID)
	}
	if reportDate == nil {
		return nil
	}

	recordedAt := r.reportReadingTime(*reportDate)
	oldUnits, err := r.deleteStaleP2HReadings(ctx, tx, reportID, *unitID)
	if err != nil {
		return err
	}

	if validP2HReading(hm) {
		if err := upsertP2HReading(ctx, tx, *unitID, "hm", roundReading(*hm), recordedAt, reportID, "P2H HM awal"); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `
		DELETE FROM unit_meter_readings
		WHERE source_type = 'p2h' AND source_id = $1 AND reading_type = 'hm'
	`, reportID); err != nil {
		return fmt.Errorf("remove stale P2H HM reading %d: %w", reportID, err)
	}

	if validP2HReading(km) {
		if err := upsertP2HReading(ctx, tx, *unitID, "km", roundReading(*km), recordedAt, reportID, "P2H KM akhir"); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `
		DELETE FROM unit_meter_readings
		WHERE source_type = 'p2h' AND source_id = $1 AND reading_type = 'km'
	`, reportID); err != nil {
		return fmt.Errorf("remove stale P2H KM reading %d: %w", reportID, err)
	}

	for _, readingType := range []string{"hm", "km"} {
		if err := recomputeP2HAnomalies(ctx, tx, *unitID, readingType); err != nil {
			return err
		}
		if err := notifyP2HAnomaly(ctx, tx, reportID, *unitID, readingType); err != nil {
			return err
		}
	}
	if err := refreshUnitMeterCache(ctx, tx, *unitID); err != nil {
		return err
	}
	for _, oldUnitID := range uniqueUnitIDs(oldUnits) {
		if err := recomputeAndRefreshUnit(ctx, tx, oldUnitID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) deleteP2HReadingsForSource(ctx context.Context, tx pgx.Tx, reportID int64) error {
	rows, err := tx.Query(ctx, `
		DELETE FROM unit_meter_readings
		WHERE source_type = 'p2h' AND source_id = $1
		RETURNING unit_id
	`, reportID)
	if err != nil {
		return fmt.Errorf("delete P2H readings for report %d: %w", reportID, err)
	}
	defer rows.Close()

	unitIDs := make([]int64, 0)
	for rows.Next() {
		var unitID int64
		if err := rows.Scan(&unitID); err != nil {
			return fmt.Errorf("scan deleted P2H reading unit: %w", err)
		}
		unitIDs = append(unitIDs, unitID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate deleted P2H reading units: %w", err)
	}
	rows.Close()
	for _, unitID := range uniqueUnitIDs(unitIDs) {
		if err := recomputeAndRefreshUnit(ctx, tx, unitID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) deleteStaleP2HReadings(ctx context.Context, tx pgx.Tx, reportID, unitID int64) ([]int64, error) {
	rows, err := tx.Query(ctx, `
		DELETE FROM unit_meter_readings
		WHERE source_type = 'p2h' AND source_id = $1 AND unit_id <> $2
		RETURNING unit_id
	`, reportID, unitID)
	if err != nil {
		return nil, fmt.Errorf("delete stale P2H readings for report %d: %w", reportID, err)
	}
	defer rows.Close()

	unitIDs := make([]int64, 0)
	for rows.Next() {
		var oldUnitID int64
		if err := rows.Scan(&oldUnitID); err != nil {
			return nil, fmt.Errorf("scan stale P2H reading unit: %w", err)
		}
		unitIDs = append(unitIDs, oldUnitID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stale P2H reading units: %w", err)
	}
	return unitIDs, nil
}

func upsertP2HReading(ctx context.Context, tx pgx.Tx, unitID int64, readingType string, value float64, recordedAt time.Time, reportID int64, notes string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO unit_meter_readings (
			unit_id, reading_type, value, recorded_at, source_type, source_id, notes
		)
		VALUES ($1, $2, $3, $4, 'p2h', $5, $6)
		ON CONFLICT (unit_id, reading_type, source_type, source_id)
			WHERE source_id IS NOT NULL
		DO UPDATE
		SET value = EXCLUDED.value,
		    recorded_at = EXCLUDED.recorded_at,
		    notes = EXCLUDED.notes
		WHERE unit_meter_readings.value IS DISTINCT FROM EXCLUDED.value
		   OR unit_meter_readings.recorded_at IS DISTINCT FROM EXCLUDED.recorded_at
		   OR unit_meter_readings.notes IS DISTINCT FROM EXCLUDED.notes
	`, unitID, readingType, value, recordedAt, reportID, notes); err != nil {
		return fmt.Errorf("upsert P2H %s reading %d: %w", readingType, reportID, err)
	}
	return nil
}

// recomputeP2HAnomalies marks a P2H reading as anomaly when its value is lower
// than the immediately preceding reading for the same unit and meter type.
// Manual/WO readings remain valid; only P2H rows are marked. Recomputing the
// whole unit/type series makes corrections deterministic when an earlier P2H
// value changes.
func recomputeP2HAnomalies(ctx context.Context, tx pgx.Tx, unitID int64, readingType string) error {
	if _, err := tx.Exec(ctx, `
		WITH RECURSIVE ordered AS (
			SELECT id, source_type, value,
			       row_number() OVER (ORDER BY recorded_at, id) AS seq
			FROM unit_meter_readings
			WHERE unit_id = $1 AND reading_type = $2
		),
		walk AS (
			SELECT id, source_type, value, seq,
			       NULL::numeric AS previous_valid,
			       false AS anomaly
			FROM ordered
			WHERE seq = 1
			UNION ALL
			SELECT o.id, o.source_type, o.value, o.seq,
			       CASE WHEN w.anomaly THEN w.previous_valid ELSE w.value END,
			       (
			           o.source_type = 'p2h'
			           AND (CASE WHEN w.anomaly THEN w.previous_valid ELSE w.value END) IS NOT NULL
			           AND o.value < (CASE WHEN w.anomaly THEN w.previous_valid ELSE w.value END)
			       )
			FROM ordered o
			JOIN walk w ON o.seq = w.seq + 1
		)
		UPDATE unit_meter_readings r
		SET is_anomaly = x.anomaly,
		    anomaly_reason = CASE
		        WHEN x.anomaly THEN 'Nilai P2H lebih rendah dari reading valid sebelumnya'
		        ELSE NULL
		    END
		FROM walk x
		WHERE r.id = x.id
	`, unitID, readingType); err != nil {
		return fmt.Errorf("recompute P2H %s anomalies for unit %d: %w", readingType, unitID, err)
	}
	return nil
}

func notifyP2HAnomaly(ctx context.Context, tx pgx.Tx, reportID, unitID int64, readingType string) error {
	var isAnomaly bool
	err := tx.QueryRow(ctx, `
		SELECT is_anomaly
		FROM unit_meter_readings
		WHERE source_type = 'p2h' AND source_id = $1 AND reading_type = $2
	`, reportID, readingType).Scan(&isAnomaly)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read P2H %s anomaly status %d: %w", readingType, reportID, err)
	}
	if !isAnomaly {
		return nil
	}
	label := strings.ToUpper(readingType)
	if _, err := notifications.CreateForRoles(ctx, tx, []string{"admin"}, notifications.CreateInput{
		Type:      "p2h_reading_anomaly",
		Title:     "Anomali reading P2H",
		Body:      fmt.Sprintf("Unit #%d: nilai %s pada laporan P2H #%d lebih rendah dari reading sebelumnya dan tidak dipakai untuk current unit.", unitID, label, reportID),
		RefType:   "p2h_report",
		RefID:     &reportID,
		DedupeKey: fmt.Sprintf("p2h-reading-anomaly:%d:%s", reportID, readingType),
	}, nil); err != nil {
		return fmt.Errorf("notify P2H %s anomaly %d: %w", readingType, reportID, err)
	}
	return nil
}

func recomputeAndRefreshUnit(ctx context.Context, tx pgx.Tx, unitID int64) error {
	for _, readingType := range []string{"hm", "km"} {
		if err := recomputeP2HAnomalies(ctx, tx, unitID, readingType); err != nil {
			return err
		}
	}
	return refreshUnitMeterCache(ctx, tx, unitID)
}

func uniqueUnitIDs(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// refreshUnitMeterCache makes units.current_* equal to the latest reading by
// recorded_at. It intentionally does not use "latest value wins" without a
// timestamp, so backfilling an old report cannot overwrite a newer reading.
func refreshUnitMeterCache(ctx context.Context, tx pgx.Tx, unitID int64) error {
	if _, err := tx.Exec(ctx, `
		UPDATE units u
		SET current_hm = COALESCE((
		        SELECT r.value
		        FROM unit_meter_readings r
		        WHERE r.unit_id = u.id AND r.reading_type = 'hm' AND r.is_anomaly = false
		        ORDER BY r.recorded_at DESC, r.id DESC
		        LIMIT 1
		    ), 0),
		    hm_updated_at = (
		        SELECT r.recorded_at
		        FROM unit_meter_readings r
		        WHERE r.unit_id = u.id AND r.reading_type = 'hm' AND r.is_anomaly = false
		        ORDER BY r.recorded_at DESC, r.id DESC
		        LIMIT 1
		    ),
		    current_km = COALESCE((
		        SELECT r.value
		        FROM unit_meter_readings r
		        WHERE r.unit_id = u.id AND r.reading_type = 'km' AND r.is_anomaly = false
		        ORDER BY r.recorded_at DESC, r.id DESC
		        LIMIT 1
		    ), 0),
		    km_updated_at = (
		        SELECT r.recorded_at
		        FROM unit_meter_readings r
		        WHERE r.unit_id = u.id AND r.reading_type = 'km' AND r.is_anomaly = false
		        ORDER BY r.recorded_at DESC, r.id DESC
		        LIMIT 1
		    )
		WHERE u.id = $1
	`, unitID); err != nil {
		return fmt.Errorf("refresh unit %d meter cache: %w", unitID, err)
	}
	return nil
}

func recomputeAllP2HAnomalies(ctx context.Context, tx pgx.Tx) error {
	type meterKey struct {
		unitID      int64
		readingType string
	}
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT unit_id, reading_type
		FROM unit_meter_readings
		WHERE source_type = 'p2h'
		ORDER BY unit_id, reading_type
	`)
	if err != nil {
		return fmt.Errorf("list P2H meter series for anomaly recompute: %w", err)
	}
	keys := make([]meterKey, 0)
	for rows.Next() {
		var key meterKey
		if err := rows.Scan(&key.unitID, &key.readingType); err != nil {
			rows.Close()
			return fmt.Errorf("scan P2H meter series: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate P2H meter series: %w", err)
	}
	rows.Close()

	for _, key := range keys {
		if err := recomputeP2HAnomalies(ctx, tx, key.unitID, key.readingType); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) reportReadingTime(reportDate time.Time) time.Time {
	loc := r.loc
	if loc == nil {
		loc = time.UTC
	}
	year, month, day := reportDate.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

// BackfillReadings creates missing P2H readings from existing p2h_reports rows
// and refreshes units.current_hm/current_km. It is safe to run repeatedly.
func (r *Repository) BackfillReadings(ctx context.Context) (ReadingBackfillResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReadingBackfillResult{}, fmt.Errorf("begin P2H reading backfill: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	loc := "UTC"
	if r.loc != nil {
		loc = r.loc.String()
	}

	var result ReadingBackfillResult
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM p2h_reports
		WHERE unit_id IS NOT NULL
		  AND report_date IS NOT NULL
		  AND (hm_awal IS NOT NULL OR km_akhir IS NOT NULL)
	`).Scan(&result.Reports); err != nil {
		return ReadingBackfillResult{}, fmt.Errorf("count P2H reports for reading backfill: %w", err)
	}

	hmTag, err := tx.Exec(ctx, `
		INSERT INTO unit_meter_readings (
			unit_id, reading_type, value, recorded_at, source_type, source_id, notes
		)
		SELECT unit_id, 'hm', hm_awal, report_date::timestamp AT TIME ZONE $1,
		       'p2h', id, 'P2H HM awal'
		FROM p2h_reports
		WHERE unit_id IS NOT NULL
		  AND report_date IS NOT NULL
		  AND hm_awal IS NOT NULL
		  AND hm_awal >= 0
		  AND hm_awal <= $2
		ORDER BY id
		ON CONFLICT (unit_id, reading_type, source_type, source_id)
			WHERE source_id IS NOT NULL
		DO UPDATE
		SET value = EXCLUDED.value,
		    recorded_at = EXCLUDED.recorded_at,
		    notes = EXCLUDED.notes
		WHERE unit_meter_readings.value IS DISTINCT FROM EXCLUDED.value
		   OR unit_meter_readings.recorded_at IS DISTINCT FROM EXCLUDED.recorded_at
		   OR unit_meter_readings.notes IS DISTINCT FROM EXCLUDED.notes
	`, loc, p2hReadingMax)
	if err != nil {
		return ReadingBackfillResult{}, fmt.Errorf("backfill P2H HM readings: %w", err)
	}

	kmTag, err := tx.Exec(ctx, `
		INSERT INTO unit_meter_readings (
			unit_id, reading_type, value, recorded_at, source_type, source_id, notes
		)
		SELECT unit_id, 'km', km_akhir, report_date::timestamp AT TIME ZONE $1,
		       'p2h', id, 'P2H KM akhir'
		FROM p2h_reports
		WHERE unit_id IS NOT NULL
		  AND report_date IS NOT NULL
		  AND km_akhir IS NOT NULL
		  AND km_akhir >= 0
		  AND km_akhir <= $2
		ORDER BY id
		ON CONFLICT (unit_id, reading_type, source_type, source_id)
			WHERE source_id IS NOT NULL
		DO UPDATE
		SET value = EXCLUDED.value,
		    recorded_at = EXCLUDED.recorded_at,
		    notes = EXCLUDED.notes
		WHERE unit_meter_readings.value IS DISTINCT FROM EXCLUDED.value
		   OR unit_meter_readings.recorded_at IS DISTINCT FROM EXCLUDED.recorded_at
		   OR unit_meter_readings.notes IS DISTINCT FROM EXCLUDED.notes
	`, loc, p2hReadingMax)
	if err != nil {
		return ReadingBackfillResult{}, fmt.Errorf("backfill P2H KM readings: %w", err)
	}
	result.Readings = hmTag.RowsAffected() + kmTag.RowsAffected()

	deletedRows, err := tx.Query(ctx, `
		DELETE FROM unit_meter_readings r
		USING p2h_reports p
		WHERE r.source_type = 'p2h'
		  AND r.source_id = p.id
		  AND (p.unit_id IS NULL OR r.unit_id <> p.unit_id)
		RETURNING r.unit_id
	`)
	if err != nil {
		return ReadingBackfillResult{}, fmt.Errorf("remove stale P2H readings after backfill: %w", err)
	}
	deletedUnitIDs := make([]int64, 0)
	for deletedRows.Next() {
		var deletedUnitID int64
		if err := deletedRows.Scan(&deletedUnitID); err != nil {
			deletedRows.Close()
			return ReadingBackfillResult{}, fmt.Errorf("scan stale P2H reading unit after backfill: %w", err)
		}
		deletedUnitIDs = append(deletedUnitIDs, deletedUnitID)
	}
	if err := deletedRows.Err(); err != nil {
		deletedRows.Close()
		return ReadingBackfillResult{}, fmt.Errorf("iterate stale P2H reading units after backfill: %w", err)
	}
	deletedRows.Close()

	if err := recomputeAllP2HAnomalies(ctx, tx); err != nil {
		return ReadingBackfillResult{}, err
	}
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM unit_meter_readings WHERE is_anomaly = true
	`).Scan(&result.Anomalies); err != nil {
		return ReadingBackfillResult{}, fmt.Errorf("count P2H reading anomalies: %w", err)
	}
	if result.Anomalies > 0 {
		if _, err := notifications.CreateForRoles(ctx, tx, []string{"admin"}, notifications.CreateInput{
			Type:      "p2h_reading_anomaly_summary",
			Title:     "Anomali reading P2H",
			Body:      fmt.Sprintf("Backfill menemukan %d reading P2H yang lebih rendah dari reading sebelumnya.", result.Anomalies),
			RefType:   "p2h_backfill",
			DedupeKey: "p2h-reading-anomaly-backfill:" + time.Now().UTC().Format("2006-01-02"),
		}, nil); err != nil {
			return ReadingBackfillResult{}, fmt.Errorf("notify P2H anomaly summary: %w", err)
		}
	}

	unitTag, err := tx.Exec(ctx, `
		UPDATE units u
		SET current_hm = COALESCE((
		        SELECT r.value FROM unit_meter_readings r
		        WHERE r.unit_id = u.id AND r.reading_type = 'hm' AND r.is_anomaly = false
		        ORDER BY r.recorded_at DESC, r.id DESC LIMIT 1
		    ), 0),
		    hm_updated_at = (
		        SELECT r.recorded_at FROM unit_meter_readings r
		        WHERE r.unit_id = u.id AND r.reading_type = 'hm' AND r.is_anomaly = false
		        ORDER BY r.recorded_at DESC, r.id DESC LIMIT 1
		    ),
		    current_km = COALESCE((
		        SELECT r.value FROM unit_meter_readings r
		        WHERE r.unit_id = u.id AND r.reading_type = 'km' AND r.is_anomaly = false
		        ORDER BY r.recorded_at DESC, r.id DESC LIMIT 1
		    ), 0),
		    km_updated_at = (
		        SELECT r.recorded_at FROM unit_meter_readings r
		        WHERE r.unit_id = u.id AND r.reading_type = 'km' AND r.is_anomaly = false
		        ORDER BY r.recorded_at DESC, r.id DESC LIMIT 1
		    )
		WHERE EXISTS (
		    SELECT 1 FROM unit_meter_readings r WHERE r.unit_id = u.id
		)
	`)
	if err != nil {
		return ReadingBackfillResult{}, fmt.Errorf("refresh unit meter cache after backfill: %w", err)
	}
	result.Units = unitTag.RowsAffected()
	for _, unitID := range uniqueUnitIDs(deletedUnitIDs) {
		if err := recomputeAndRefreshUnit(ctx, tx, unitID); err != nil {
			return ReadingBackfillResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ReadingBackfillResult{}, fmt.Errorf("commit P2H reading backfill: %w", err)
	}
	return result, nil
}

func validP2HReading(value *float64) bool {
	if value == nil {
		return false
	}
	return !math.IsNaN(*value) && !math.IsInf(*value, 0) && *value >= 0 && *value <= p2hReadingMax
}

func roundReading(value float64) float64 {
	return math.Round(value*10) / 10
}
