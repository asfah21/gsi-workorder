package p2h

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/notifications"
	"gsi-workorder/internal/workorders"
)

type Repository struct {
	pool *pgxpool.Pool
	loc  *time.Location
}

func NewRepository(pool *pgxpool.Pool, loc *time.Location) *Repository {
	return &Repository{pool: pool, loc: loc}
}

type metadataResult struct {
	ReportID    int64
	UnitID      *int64
	NeedsDetail bool
}

type SaveResult struct {
	Findings  int
	Candidate *CandidateCreated
}

func (r *Repository) UpsertMetadata(ctx context.Context, report Report) (metadataResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return metadataResult{}, fmt.Errorf("begin report metadata: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	unitID, err := upsertUnit(ctx, tx, report)
	if err != nil {
		return metadataResult{}, err
	}

	var result metadataResult
	result.UnitID = unitID
	if err := tx.QueryRow(ctx, `
		INSERT INTO p2h_reports (
			odoo_report_id, name, report_date, unit_id, unit_type, shift,
			foreman_name, employee_name, location, state, approval_status,
			km_akhir, hm_awal, notes, synced_at
		)
		VALUES ($1, NULLIF($2, ''), $3, $4, NULLIF($5, ''), NULLIF($6, ''),
		        NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''),
		        NULLIF($11, ''), $12, $13, NULLIF($14, ''), now())
		ON CONFLICT (odoo_report_id) DO UPDATE
		SET name = EXCLUDED.name,
		    report_date = EXCLUDED.report_date,
		    unit_id = EXCLUDED.unit_id,
		    unit_type = EXCLUDED.unit_type,
		    shift = EXCLUDED.shift,
		    foreman_name = EXCLUDED.foreman_name,
		    employee_name = EXCLUDED.employee_name,
		    location = EXCLUDED.location,
		    state = EXCLUDED.state,
		    approval_status = EXCLUDED.approval_status,
		    km_akhir = EXCLUDED.km_akhir,
		    hm_awal = EXCLUDED.hm_awal,
		    notes = EXCLUDED.notes,
		    synced_at = now(),
		    detail_synced_at = CASE
		        WHEN p2h_reports.state IS DISTINCT FROM EXCLUDED.state
		          OR p2h_reports.approval_status IS DISTINCT FROM EXCLUDED.approval_status
		        THEN NULL
		        ELSE p2h_reports.detail_synced_at
		    END
		RETURNING id, unit_id, (detail_synced_at IS NULL)
	`, report.OdooReportID, report.Name, report.ReportDate, unitID, report.UnitType,
		report.Shift, report.ForemanName, report.EmployeeName, report.Location,
		report.State, report.ApprovalStatus, report.KMAkhir, report.HMAwal,
		report.Notes).Scan(&result.ReportID, &result.UnitID, &result.NeedsDetail); err != nil {
		return metadataResult{}, fmt.Errorf("upsert report %d: %w", report.OdooReportID, err)
	}

	if err := r.ingestReportReadings(ctx, tx, result.ReportID, result.UnitID, report.ReportDate, report.HMAwal, report.KMAkhir); err != nil {
		return metadataResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return metadataResult{}, fmt.Errorf("commit report metadata: %w", err)
	}
	return result, nil
}

func upsertUnit(ctx context.Context, tx pgx.Tx, report Report) (*int64, error) {
	code := strings.TrimSpace(report.UnitCode)
	if code == "" && (report.OdooUnitID == nil || *report.OdooUnitID <= 0) {
		return nil, nil
	}
	if code == "" {
		code = fmt.Sprintf("ODOO-%d", *report.OdooUnitID)
	}

	var id int64
	var err error
	if report.OdooUnitID != nil && *report.OdooUnitID > 0 {
		err = tx.QueryRow(ctx, `
			SELECT id
			FROM units
			WHERE odoo_unit_id = $1
			FOR UPDATE
		`, *report.OdooUnitID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `
				SELECT id
				FROM units
				WHERE lower(btrim(code)) = lower(btrim($1))
				LIMIT 1
				FOR UPDATE
			`, code).Scan(&id)
			if errors.Is(err, pgx.ErrNoRows) {
				if err := tx.QueryRow(ctx, `
					INSERT INTO units (odoo_unit_id, code, unit_type, synced_at)
					VALUES ($1, $2, NULLIF($3, ''), now())
					RETURNING id
				`, *report.OdooUnitID, code, report.UnitType).Scan(&id); err != nil {
					return nil, fmt.Errorf("insert P2H unit %d: %w", *report.OdooUnitID, err)
				}
			} else if err != nil {
				return nil, fmt.Errorf("find P2H unit by code: %w", err)
			} else if _, err := tx.Exec(ctx, `
				UPDATE units
				SET odoo_unit_id = $2,
				    unit_type = COALESCE(NULLIF($3, ''), unit_type),
				    synced_at = now()
				WHERE id = $1
			`, id, *report.OdooUnitID, report.UnitType); err != nil {
				return nil, fmt.Errorf("attach P2H unit %d: %w", *report.OdooUnitID, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("find P2H unit by Odoo id: %w", err)
		} else if _, err := tx.Exec(ctx, `
			UPDATE units
			SET unit_type = COALESCE(NULLIF($2, ''), unit_type),
			    synced_at = now()
			WHERE id = $1
		`, id, report.UnitType); err != nil {
			return nil, fmt.Errorf("refresh P2H unit %d: %w", id, err)
		}
	} else {
		err = tx.QueryRow(ctx, `
			SELECT id
			FROM units
			WHERE lower(btrim(code)) = lower(btrim($1))
			LIMIT 1
			FOR UPDATE
		`, code).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			if err := tx.QueryRow(ctx, `
				INSERT INTO units (code, unit_type, synced_at)
				VALUES ($1, NULLIF($2, ''), now())
				RETURNING id
			`, code, report.UnitType).Scan(&id); err != nil {
				return nil, fmt.Errorf("insert P2H unit by code: %w", err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("find P2H unit by code: %w", err)
		}
	}
	return &id, nil
}

func (r *Repository) SaveDetail(ctx context.Context, report Report, settings SettingsValues, now time.Time) (SaveResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SaveResult{}, fmt.Errorf("begin report detail: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var reportID int64
	var unitID *int64
	if err := tx.QueryRow(ctx, `
		SELECT id, unit_id
		FROM p2h_reports
		WHERE odoo_report_id = $1
		FOR UPDATE
	`, report.OdooReportID).Scan(&reportID, &unitID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SaveResult{}, fmt.Errorf("report %d metadata is missing", report.OdooReportID)
		}
		return SaveResult{}, fmt.Errorf("lock report %d: %w", report.OdooReportID, err)
	}

	findingIDs := make([]int64, 0, len(report.Findings))
	rusakCount := 0
	for _, finding := range report.Findings {
		var findingID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO p2h_findings (
				odoo_line_id, report_id, question_id, component, category,
				category_norm, condition, remark
			)
			VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
			        $7, NULLIF($8, ''))
			ON CONFLICT (odoo_line_id) DO UPDATE
			SET report_id = EXCLUDED.report_id,
			    question_id = EXCLUDED.question_id,
			    component = EXCLUDED.component,
			    category = EXCLUDED.category,
			    category_norm = EXCLUDED.category_norm,
			    condition = EXCLUDED.condition,
			    remark = EXCLUDED.remark
			RETURNING id
		`, finding.OdooLineID, reportID, finding.QuestionID, finding.Component,
			finding.Category, finding.CategoryNorm, finding.Condition,
			finding.Remark).Scan(&findingID); err != nil {
			return SaveResult{}, fmt.Errorf("upsert finding %d: %w", finding.OdooLineID, err)
		}
		findingIDs = append(findingIDs, findingID)
		if finding.Condition == "rusak" {
			rusakCount++
		}
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM p2h_findings f
		WHERE f.report_id = $1
		  AND NOT (f.id = ANY($2))
		  AND NOT EXISTS (
		      SELECT 1 FROM wo_findings wf WHERE wf.finding_id = f.id
		  )
	`, reportID, findingIDs); err != nil {
		return SaveResult{}, fmt.Errorf("remove stale findings for report %d: %w", reportID, err)
	}

	raw := strings.TrimSpace(string(report.Raw))
	if raw == "" {
		raw = "{}"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE p2h_reports
		SET checked_items = $2,
		    findings_count = $3,
		    detail_synced_at = now(),
		    synced_at = now(),
		    raw = $4::jsonb
		WHERE id = $1
	`, reportID, maxInt(report.CheckedItems, len(report.Findings)), len(report.Findings), raw); err != nil {
		return SaveResult{}, fmt.Errorf("update report detail %d: %w", reportID, err)
	}

	result := SaveResult{Findings: len(report.Findings)}
	if len(report.Findings) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return SaveResult{}, fmt.Errorf("commit empty report detail: %w", err)
		}
		return result, nil
	}

	var candidateID int64
	var candidateStatus string
	err = tx.QueryRow(ctx, `
		SELECT id, status
		FROM wo_candidates
		WHERE p2h_report_id = $1
		FOR UPDATE
	`, reportID).Scan(&candidateID, &candidateStatus)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return SaveResult{}, fmt.Errorf("commit report detail: %w", err)
		}
		return result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return SaveResult{}, fmt.Errorf("read candidate for report %d: %w", reportID, err)
	}

	archived := isArchived(report.ReportDate, now, r.loc, settings.BackfillDays)
	priority := "medium"
	if rusakCount > 0 {
		priority = "high"
	}
	status := "pending"
	if archived {
		status = "archived"
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO wo_candidates (
			p2h_report_id, unit_id, suggested_priority, findings_count,
			rusak_count, status
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, reportID, unitID, priority, len(report.Findings), rusakCount, status).Scan(&candidateID); err != nil {
		return SaveResult{}, fmt.Errorf("insert candidate for report %d: %w", reportID, err)
	}
	result.Candidate = &CandidateCreated{
		ID:       candidateID,
		Archived: archived,
	}

	if archived {
		if err := tx.Commit(ctx); err != nil {
			return SaveResult{}, fmt.Errorf("commit archived candidate: %w", err)
		}
		return result, nil
	}

	if settings.AutoWO {
		title := "Tindak lanjut P2H " + reportDisplayName(report)
		description := candidateDescription(report, priority)
		order, err := workorders.Create(ctx, tx, workorders.CreateInput{
			Source:      "p2h",
			UnitID:      unitID,
			P2HReportID: &reportID,
			CandidateID: &candidateID,
			Title:       title,
			Description: description,
			Priority:    priority,
			Status:      "open",
			Now:         now,
			Location:    r.loc,
		})
		if err != nil {
			return SaveResult{}, err
		}
		if err := linkFindings(ctx, tx, order.ID, findingIDs); err != nil {
			return SaveResult{}, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE wo_candidates
			SET status = 'auto_created', wo_id = $2
			WHERE id = $1
		`, candidateID, order.ID); err != nil {
			return SaveResult{}, fmt.Errorf("mark candidate auto-created: %w", err)
		}
		result.Candidate.AutoWO = true
		result.Candidate.WorkOrderID = &order.ID
		if _, err := notifications.CreateForRoles(ctx, tx, []string{"admin", "planner"}, notifications.CreateInput{
			Type:    "p2h_auto_wo",
			Title:   "WO otomatis dibuat dari P2H",
			Body:    fmt.Sprintf("%s · %s", reportDisplayName(report), order.Code),
			RefType: "work_order",
			RefID:   &order.ID,
		}, nil); err != nil {
			return SaveResult{}, err
		}
	} else {
		if _, err := notifications.CreateForRoles(ctx, tx, []string{"admin", "planner"}, notifications.CreateInput{
			Type:    "p2h_candidate_new",
			Title:   "Usulan WO baru",
			Body:    fmt.Sprintf("%s · %d temuan", reportDisplayName(report), len(report.Findings)),
			RefType: "wo_candidate",
			RefID:   &candidateID,
		}, nil); err != nil {
			return SaveResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SaveResult{}, fmt.Errorf("commit candidate for report %d: %w", reportID, err)
	}
	return result, nil
}

func (r *Repository) LoadSettings(ctx context.Context) (SettingsValues, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT key, COALESCE(value, '')
		FROM app_settings
		WHERE key IN ('p2h_auto_wo', 'candidate_backfill_days', 'candidate_sla_hours')
	`)
	if err != nil {
		return SettingsValues{}, fmt.Errorf("read P2H settings: %w", err)
	}
	defer rows.Close()

	values := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return SettingsValues{}, fmt.Errorf("scan P2H setting: %w", err)
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return SettingsValues{}, fmt.Errorf("iterate P2H settings: %w", err)
	}

	settings := SettingsValues{BackfillDays: 30, SLAHours: 24}
	if parsed, err := strconv.ParseBool(values["p2h_auto_wo"]); err == nil {
		settings.AutoWO = parsed
	}
	if parsed, err := strconv.Atoi(values["candidate_backfill_days"]); err == nil && parsed >= 0 {
		settings.BackfillDays = parsed
	}
	if parsed, err := strconv.Atoi(values["candidate_sla_hours"]); err == nil && parsed > 0 {
		settings.SLAHours = parsed
	}
	return settings, nil
}

func (r *Repository) StartSyncRun(ctx context.Context, kind string) (int64, error) {
	var id int64
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO sync_runs (kind, status, started_at)
		VALUES ($1, 'running', now())
		RETURNING id
	`, kind).Scan(&id); err != nil {
		return 0, fmt.Errorf("start P2H sync run: %w", err)
	}
	return id, nil
}

func (r *Repository) FinishSyncRun(ctx context.Context, id int64, status, errorMessage string, counters SyncCounters) error {
	var errorValue any
	if strings.TrimSpace(errorMessage) != "" {
		errorValue = errorMessage
	}
	if _, err := r.pool.Exec(ctx, `
		UPDATE sync_runs
		SET status = $2,
		    finished_at = now(),
		    reports = $3,
		    details = $4,
		    findings = $5,
		    candidates = $6,
		    work_orders = $7,
		    error = $8
		WHERE id = $1
	`, id, status, counters.Reports, counters.Details, counters.Findings,
		counters.Candidates, counters.WorkOrders, errorValue); err != nil {
		return fmt.Errorf("finish P2H sync run: %w", err)
	}
	return nil
}

const syncLockKey int64 = 20261118

// TrySyncLock takes a session-level PostgreSQL advisory lock on a dedicated
// connection. The returned release function must be called exactly once.
func (r *Repository) TrySyncLock(ctx context.Context) (func(), bool, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("acquire P2H sync lock connection: %w", err)
	}
	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, syncLockKey).Scan(&locked); err != nil {
		conn.Release()
		return nil, false, fmt.Errorf("try P2H sync lock: %w", err)
	}
	if !locked {
		conn.Release()
		return nil, false, nil
	}
	release := func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, `SELECT pg_advisory_unlock($1)`, syncLockKey)
		conn.Release()
	}
	return release, true, nil
}

func (r *Repository) SendCandidateReminders(ctx context.Context, now time.Time, slaHours int) (int, int, error) {
	if slaHours < 1 {
		slaHours = 24
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin P2H reminder transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT c.id, COALESCE(r.name, ''), COALESCE(u.code, ''),
		       COALESCE(c.findings_count, 0)
		FROM wo_candidates c
		JOIN p2h_reports r ON r.id = c.p2h_report_id
		LEFT JOIN units u ON u.id = c.unit_id
		WHERE c.status = 'pending'
		  AND c.notified_at IS NULL
		  AND c.created_at <= $1::timestamptz - make_interval(hours => $2)
		ORDER BY c.created_at ASC
		FOR UPDATE OF c SKIP LOCKED
		LIMIT 500
	`, now, slaHours)
	if err != nil {
		return 0, 0, fmt.Errorf("list P2H reminders: %w", err)
	}
	type reminder struct {
		id         int64
		reportName string
		unitCode   string
		findings   int
	}
	items := make([]reminder, 0)
	for rows.Next() {
		var item reminder
		if err := rows.Scan(&item.id, &item.reportName, &item.unitCode, &item.findings); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("scan P2H reminder: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, 0, fmt.Errorf("iterate P2H reminders: %w", err)
	}
	rows.Close()

	candidates := 0
	created := 0
	for _, item := range items {
		body := item.reportName
		if item.unitCode != "" {
			body += " · " + item.unitCode
		}
		body += fmt.Sprintf(" · %d temuan", item.findings)
		count, err := notifications.CreateForRoles(ctx, tx, []string{"admin", "planner"}, notifications.CreateInput{
			Type:    "p2h_candidate_reminder",
			Title:   "Pengingat usulan WO P2H",
			Body:    body,
			RefType: "wo_candidate",
			RefID:   &item.id,
		}, nil)
		if err != nil {
			return 0, 0, err
		}
		if count == 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE wo_candidates
			SET notified_at = now()
			WHERE id = $1 AND notified_at IS NULL
		`, item.id); err != nil {
			return 0, 0, fmt.Errorf("mark candidate %d notified: %w", item.id, err)
		}
		candidates++
		created += count
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit P2H reminders: %w", err)
	}
	return candidates, created, nil
}

func (r *Repository) SetSyncState(ctx context.Context, key, value string) error {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO sync_state (key, value, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = now()
	`, key, value); err != nil {
		return fmt.Errorf("set P2H sync state %q: %w", key, err)
	}
	return nil
}

func (r *Repository) ListReports(ctx context.Context, filter ReportFilter) ([]Report, int, error) {
	args := make([]any, 0, 6)
	where := []string{"1 = 1"}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		where = append(where, fmt.Sprintf(`(COALESCE(r.name, '') ILIKE $%d
			OR COALESCE(u.code, '') ILIKE $%d
			OR COALESCE(r.foreman_name, '') ILIKE $%d
			OR COALESCE(r.employee_name, '') ILIKE $%d
			OR COALESCE(r.location, '') ILIKE $%d)`, len(args), len(args), len(args), len(args), len(args)))
	}
	if filter.State != "" {
		args = append(args, filter.State)
		where = append(where, fmt.Sprintf("r.state = $%d", len(args)))
	}
	if filter.UnitID != nil {
		args = append(args, *filter.UnitID)
		where = append(where, fmt.Sprintf("r.unit_id = $%d", len(args)))
	}

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM p2h_reports r
		LEFT JOIN units u ON u.id = r.unit_id
		WHERE `+strings.Join(where, " AND "), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count P2H reports: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	orderColumn := reportSortColumn(filter.Sort)
	orderDirection := "DESC"
	if filter.Order == "asc" {
		orderDirection = "ASC"
	}

	rows, err := r.pool.Query(ctx, reportSelect+`
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY `+orderColumn+` `+orderDirection+`, r.id DESC
		LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list P2H reports: %w", err)
	}
	defer rows.Close()

	reports := make([]Report, 0)
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate P2H reports: %w", err)
	}
	return reports, total, nil
}

func (r *Repository) GetReport(ctx context.Context, id int64) (Report, error) {
	row := r.pool.QueryRow(ctx, reportSelect+` WHERE r.id = $1`, id)
	report, err := scanReport(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, ErrReportNotFound
	}
	if err != nil {
		return Report{}, err
	}
	report.Findings, err = r.ListFindings(ctx, id)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func (r *Repository) ListFindings(ctx context.Context, reportID int64) ([]Finding, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, odoo_line_id, report_id, question_id, COALESCE(component, ''),
		       COALESCE(category, ''), COALESCE(category_norm, ''), condition,
		       COALESCE(remark, '')
		FROM p2h_findings
		WHERE report_id = $1
		ORDER BY category_norm ASC, component ASC, id ASC
	`, reportID)
	if err != nil {
		return nil, fmt.Errorf("list report findings: %w", err)
	}
	defer rows.Close()

	items := make([]Finding, 0)
	for rows.Next() {
		var item Finding
		if err := rows.Scan(&item.ID, &item.OdooLineID, &item.ReportID,
			&item.QuestionID, &item.Component, &item.Category,
			&item.CategoryNorm, &item.Condition, &item.Remark); err != nil {
			return nil, fmt.Errorf("scan report finding: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report findings: %w", err)
	}
	return items, nil
}

const reportSelect = `
	SELECT r.id, r.odoo_report_id, COALESCE(r.name, ''), r.report_date,
	       r.unit_id, COALESCE(u.code, ''), COALESCE(r.unit_type, ''),
	       COALESCE(r.shift, ''), COALESCE(r.foreman_name, ''),
	       COALESCE(r.employee_name, ''), COALESCE(r.location, ''),
	       COALESCE(r.state, ''), COALESCE(r.approval_status, ''),
	       r.km_akhir, r.hm_awal, COALESCE(r.notes, ''),
	       COALESCE(r.checked_items, 0), COALESCE(r.findings_count, 0),
	       COALESCE((SELECT count(*) FROM p2h_findings f
	                 WHERE f.report_id = r.id AND f.condition = 'rusak'), 0),
	       r.detail_synced_at, r.synced_at, c.id, COALESCE(c.status, '')
	FROM p2h_reports r
	LEFT JOIN units u ON u.id = r.unit_id
	LEFT JOIN wo_candidates c ON c.p2h_report_id = r.id
`

type scanner interface {
	Scan(dest ...any) error
}

func scanReport(row scanner) (Report, error) {
	var report Report
	err := row.Scan(
		&report.ID, &report.OdooReportID, &report.Name, &report.ReportDate,
		&report.UnitID, &report.UnitCode, &report.UnitType, &report.Shift,
		&report.ForemanName, &report.EmployeeName, &report.Location,
		&report.State, &report.ApprovalStatus, &report.KMAkhir, &report.HMAwal,
		&report.Notes, &report.CheckedItems, &report.FindingsCount,
		&report.RusakCount, &report.DetailSyncedAt, &report.SyncedAt,
		&report.CandidateID, &report.CandidateStatus,
	)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func reportSortColumn(raw string) string {
	switch raw {
	case "name":
		return "r.name"
	case "unit_code":
		return "u.code"
	case "state":
		return "r.state"
	case "findings_count":
		return "r.findings_count"
	case "checked_items":
		return "r.checked_items"
	default:
		return "r.report_date"
	}
}

func linkFindings(ctx context.Context, tx pgx.Tx, woID int64, findingIDs []int64) error {
	for _, findingID := range findingIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO wo_findings (wo_id, finding_id)
			VALUES ($1, $2)
			ON CONFLICT (wo_id, finding_id) DO NOTHING
		`, woID, findingID); err != nil {
			return fmt.Errorf("link finding %d to WO %d: %w", findingID, woID, err)
		}
	}
	return nil
}

func reportDisplayName(report Report) string {
	if strings.TrimSpace(report.Name) != "" {
		return report.Name
	}
	if strings.TrimSpace(report.UnitCode) != "" {
		return report.UnitCode
	}
	return fmt.Sprintf("P2H-%d", report.OdooReportID)
}

func candidateDescription(report Report, priority string) string {
	var builder strings.Builder
	name := reportDisplayName(report)
	builder.WriteString("Tindak lanjut temuan P2H ")
	builder.WriteString(name)
	if report.UnitCode != "" {
		builder.WriteString(" · Unit ")
		builder.WriteString(report.UnitCode)
	}
	if report.ReportDate != nil {
		builder.WriteString(" · ")
		builder.WriteString(report.ReportDate.Format("2006-01-02"))
	}
	builder.WriteString("\nPrioritas usulan: ")
	builder.WriteString(priority)
	for index, finding := range report.Findings {
		if index >= 10 {
			builder.WriteString(fmt.Sprintf("\n- ... dan %d temuan lain", len(report.Findings)-10))
			break
		}
		builder.WriteString("\n- [")
		builder.WriteString(finding.Condition)
		builder.WriteString("] ")
		builder.WriteString(finding.Component)
		if finding.Category != "" {
			builder.WriteString(" (")
			builder.WriteString(finding.Category)
			builder.WriteString(")")
		}
		if finding.Remark != "" {
			builder.WriteString(": ")
			builder.WriteString(finding.Remark)
		}
	}
	return builder.String()
}

func isArchived(reportDate *time.Time, now time.Time, loc *time.Location, days int) bool {
	if reportDate == nil {
		return false
	}
	if days < 0 {
		days = 0
	}
	year, month, day := now.In(loc).Date()
	cutoff := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -days)
	return reportDate.Before(cutoff)
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
