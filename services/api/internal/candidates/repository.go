package candidates

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/notifications"
	"gsi-workorder/internal/p2h"
	"gsi-workorder/internal/workorders"
)

type Repository struct {
	pool    *pgxpool.Pool
	loc     *time.Location
	reports *p2h.Repository
}

func NewRepository(pool *pgxpool.Pool, loc *time.Location) *Repository {
	return &Repository{
		pool:    pool,
		loc:     loc,
		reports: p2h.NewRepository(pool, loc),
	}
}

func (r *Repository) List(ctx context.Context, filter Filter) ([]Candidate, int, error) {
	args := make([]any, 0, 3)
	where := []string{"1 = 1"}
	if filter.Status != "" && filter.Status != "all" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("c.status = $%d", len(args)))
	}
	if filter.UnitType != "" {
		args = append(args, filter.UnitType)
		where = append(where, fmt.Sprintf("r.unit_type = $%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		position := len(args)
		where = append(where, fmt.Sprintf(`(COALESCE(r.name, '') ILIKE $%d
			OR COALESCE(u.code, '') ILIKE $%d
			OR COALESCE(r.foreman_name, '') ILIKE $%d
			OR COALESCE(r.employee_name, '') ILIKE $%d
			OR COALESCE(r.location, '') ILIKE $%d)`,
			position, position, position, position, position))
	}

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM wo_candidates c
		JOIN p2h_reports r ON r.id = c.p2h_report_id
		LEFT JOIN units u ON u.id = c.unit_id
		WHERE `+strings.Join(where, " AND "), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count candidates: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	orderColumn := candidateSortColumn(filter.Sort)
	orderDirection := "DESC"
	if filter.Order == "asc" {
		orderDirection = "ASC"
	}
	rows, err := r.pool.Query(ctx, candidateSelect+`
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY `+orderColumn+` `+orderDirection+`, c.id DESC
		LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list candidates: %w", err)
	}
	defer rows.Close()

	items := make([]Candidate, 0)
	for rows.Next() {
		item, err := scanCandidate(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate candidates: %w", err)
	}
	return items, total, nil
}

func (r *Repository) Get(ctx context.Context, id int64) (Detail, error) {
	item, err := r.getCandidate(ctx, r.pool, id, false)
	if err != nil {
		return Detail{}, err
	}
	report, err := r.reports.GetReport(ctx, item.P2HReportID)
	if err != nil {
		return Detail{}, err
	}
	order, err := r.workOrder(ctx, item.WorkOrderID)
	if err != nil {
		return Detail{}, err
	}
	return Detail{Candidate: item, Report: report, WorkOrder: order}, nil
}

func (r *Repository) Approve(ctx context.Context, id, actorID int64, input ApproveInput) (ApproveResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ApproveResult{}, fmt.Errorf("begin candidate approval: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := r.approveTx(ctx, tx, id, actorID, input)
	if err != nil {
		return ApproveResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ApproveResult{}, fmt.Errorf("commit candidate approval: %w", err)
	}
	return result, nil
}

func (r *Repository) Skip(ctx context.Context, id, actorID int64, input SkipInput) (Candidate, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Candidate{}, fmt.Errorf("begin candidate skip: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	item, err := r.skipTx(ctx, tx, id, actorID, input)
	if err != nil {
		return Candidate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Candidate{}, fmt.Errorf("commit candidate skip: %w", err)
	}
	return item, nil
}

func (r *Repository) Bulk(ctx context.Context, actorID int64, input BulkInput) (BulkResult, error) {
	if len(input.IDs) < 1 || len(input.IDs) > MaxBulk {
		return BulkResult{}, &ValidationError{Fields: map[string]string{"ids": "Pilih 1 sampai 50 usulan."}}
	}
	seen := make(map[int64]struct{}, len(input.IDs))
	for _, id := range input.IDs {
		if id <= 0 {
			return BulkResult{}, &ValidationError{Fields: map[string]string{"ids": "ID usulan tidak valid."}}
		}
		if _, exists := seen[id]; exists {
			return BulkResult{}, &ValidationError{Fields: map[string]string{"ids": "ID usulan duplikat."}}
		}
		seen[id] = struct{}{}
	}
	switch input.Action {
	case "approve", "skip":
	default:
		return BulkResult{}, &ValidationError{Fields: map[string]string{"action": "Aksi harus approve atau skip."}}
	}
	if input.Action == "skip" {
		if err := validateReason(input.Reason); err != nil {
			return BulkResult{}, err
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return BulkResult{}, fmt.Errorf("begin candidate bulk: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result := BulkResult{}
	for _, id := range input.IDs {
		if input.Action == "approve" {
			approved, err := r.approveTx(ctx, tx, id, actorID, ApproveInput{
				Priority:   input.Priority,
				AssigneeID: input.AssigneeID,
				Note:       input.Note,
			})
			if err != nil {
				return BulkResult{}, err
			}
			result.Approved = append(result.Approved, approved)
		} else {
			skipped, err := r.skipTx(ctx, tx, id, actorID, SkipInput{Reason: input.Reason})
			if err != nil {
				return BulkResult{}, err
			}
			result.Skipped = append(result.Skipped, skipped)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return BulkResult{}, fmt.Errorf("commit candidate bulk: %w", err)
	}
	return result, nil
}

func (r *Repository) approveTx(ctx context.Context, tx pgx.Tx, id, actorID int64, input ApproveInput) (ApproveResult, error) {
	item, err := r.getCandidate(ctx, tx, id, true)
	if err != nil {
		return ApproveResult{}, err
	}
	if item.Status != "pending" {
		return ApproveResult{}, ErrConflict
	}
	priority := strings.TrimSpace(input.Priority)
	if priority == "" {
		priority = item.SuggestedPriority
	}
	if !ValidPriority(priority) {
		return ApproveResult{}, &ValidationError{Fields: map[string]string{"priority": "Prioritas tidak valid."}}
	}
	if len(strings.TrimSpace(input.Note)) > maxTextLen {
		return ApproveResult{}, &ValidationError{Fields: map[string]string{"note": "Catatan maksimal 1000 karakter."}}
	}
	if input.AssigneeID != nil {
		var exists int64
		if err := tx.QueryRow(ctx, `
			SELECT id FROM users WHERE id = $1 AND is_active = true FOR SHARE
		`, *input.AssigneeID).Scan(&exists); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ApproveResult{}, &ValidationError{Fields: map[string]string{"assignee_id": "Assignee tidak ditemukan atau tidak aktif."}}
			}
			return ApproveResult{}, fmt.Errorf("validate assignee: %w", err)
		}
	}

	findings, err := r.findingsTx(ctx, tx, item.P2HReportID)
	if err != nil {
		return ApproveResult{}, err
	}
	now := time.Now()
	title := "Tindak lanjut P2H " + candidateReportName(item)
	description := buildDescription(item, findings, priority, input.Note)
	order, err := workorders.Create(ctx, tx, workorders.CreateInput{
		Source:      "p2h",
		UnitID:      item.UnitID,
		P2HReportID: &item.P2HReportID,
		CandidateID: &item.ID,
		Title:       title,
		Description: description,
		Priority:    priority,
		Status:      "open",
		AssigneeID:  input.AssigneeID,
		CreatedBy:   &actorID,
		Now:         now,
		Location:    r.loc,
	})
	if err != nil {
		return ApproveResult{}, mapWriteError(err)
	}
	for _, finding := range findings {
		if _, err := tx.Exec(ctx, `
			INSERT INTO wo_findings (wo_id, finding_id)
			VALUES ($1, $2)
			ON CONFLICT (wo_id, finding_id) DO NOTHING
		`, order.ID, finding.ID); err != nil {
			return ApproveResult{}, fmt.Errorf("link finding %d to WO: %w", finding.ID, err)
		}
	}

	var note any
	if strings.TrimSpace(input.Note) != "" {
		note = strings.TrimSpace(input.Note)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE wo_candidates
		SET status = 'approved',
		    decided_by = $2,
		    decided_at = $3,
		    decision_note = $4,
		    skip_reason = NULL,
		    wo_id = $5
		WHERE id = $1
	`, item.ID, actorID, now, note, order.ID); err != nil {
		return ApproveResult{}, fmt.Errorf("approve candidate %d: %w", item.ID, err)
	}
	item.Status = "approved"
	item.DecidedBy = &actorID
	item.DecidedAt = &now
	item.DecisionNote = strings.TrimSpace(input.Note)
	item.WorkOrderID = &order.ID
	item.WorkOrderCode = order.Code

	if _, err := notifications.CreateForRoles(ctx, tx, []string{"admin", "planner"}, notifications.CreateInput{
		Type:    "p2h_candidate_approved",
		Title:   "Usulan WO disetujui",
		Body:    fmt.Sprintf("%s · %s", candidateReportName(item), order.Code),
		RefType: "work_order",
		RefID:   &order.ID,
	}, nil); err != nil {
		return ApproveResult{}, err
	}
	if input.AssigneeID != nil {
		if _, err := notifications.CreateForUsers(ctx, tx, []int64{*input.AssigneeID}, notifications.CreateInput{
			Type:    "p2h_work_order_assigned",
			Title:   "WO baru ditugaskan",
			Body:    fmt.Sprintf("%s · %s", order.Code, candidateReportName(item)),
			RefType: "work_order",
			RefID:   &order.ID,
		}); err != nil {
			return ApproveResult{}, err
		}
	}
	return ApproveResult{Candidate: item, WorkOrder: order}, nil
}

func (r *Repository) skipTx(ctx context.Context, tx pgx.Tx, id, actorID int64, input SkipInput) (Candidate, error) {
	item, err := r.getCandidate(ctx, tx, id, true)
	if err != nil {
		return Candidate{}, err
	}
	if item.Status != "pending" {
		return Candidate{}, ErrConflict
	}
	reason, err := validateReasonValue(input.Reason)
	if err != nil {
		return Candidate{}, err
	}
	now := time.Now()
	if _, err := tx.Exec(ctx, `
		UPDATE wo_candidates
		SET status = 'skipped',
		    decided_by = $2,
		    decided_at = $3,
		    skip_reason = $4,
		    decision_note = NULL
		WHERE id = $1
	`, item.ID, actorID, now, reason); err != nil {
		return Candidate{}, fmt.Errorf("skip candidate %d: %w", item.ID, err)
	}
	item.Status = "skipped"
	item.DecidedBy = &actorID
	item.DecidedAt = &now
	item.SkipReason = reason

	if _, err := notifications.CreateForRoles(ctx, tx, []string{"admin", "planner"}, notifications.CreateInput{
		Type:    "p2h_candidate_skipped",
		Title:   "Usulan WO ditolak",
		Body:    fmt.Sprintf("%s · %s", candidateReportName(item), reason),
		RefType: "wo_candidate",
		RefID:   &item.ID,
	}, nil); err != nil {
		return Candidate{}, err
	}
	return item, nil
}

func (r *Repository) getCandidate(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, id int64, lock bool) (Candidate, error) {
	query := candidateSelect + ` WHERE c.id = $1`
	if lock {
		query += ` FOR UPDATE OF c`
	}
	row := q.QueryRow(ctx, query, id)
	item, err := scanCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Candidate{}, ErrNotFound
	}
	if err != nil {
		return Candidate{}, fmt.Errorf("get candidate: %w", err)
	}
	return item, nil
}

func (r *Repository) findingsTx(ctx context.Context, tx pgx.Tx, reportID int64) ([]p2h.Finding, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, odoo_line_id, report_id, question_id, COALESCE(component, ''),
		       COALESCE(category, ''), COALESCE(category_norm, ''), condition,
		       COALESCE(remark, '')
		FROM p2h_findings
		WHERE report_id = $1
		ORDER BY category_norm ASC, component ASC, id ASC
	`, reportID)
	if err != nil {
		return nil, fmt.Errorf("list candidate findings: %w", err)
	}
	defer rows.Close()

	items := make([]p2h.Finding, 0)
	for rows.Next() {
		var item p2h.Finding
		if err := rows.Scan(&item.ID, &item.OdooLineID, &item.ReportID,
			&item.QuestionID, &item.Component, &item.Category,
			&item.CategoryNorm, &item.Condition, &item.Remark); err != nil {
			return nil, fmt.Errorf("scan candidate finding: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate candidate findings: %w", err)
	}
	return items, nil
}

func (r *Repository) workOrder(ctx context.Context, id *int64) (*WorkOrderSummary, error) {
	if id == nil {
		return nil, nil
	}
	var item WorkOrderSummary
	err := r.pool.QueryRow(ctx, `
		SELECT w.id, w.code, COALESCE(w.title, ''), COALESCE(w.description, ''),
		       COALESCE(w.priority, 'medium'), w.status, w.assignee_id,
		       COALESCE(a.name, ''), w.created_at
		FROM work_orders w
		LEFT JOIN users a ON a.id = w.assignee_id
		WHERE w.id = $1
	`, *id).Scan(&item.ID, &item.Code, &item.Title, &item.Description,
		&item.Priority, &item.Status, &item.AssigneeID, &item.AssigneeName, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get candidate work order: %w", err)
	}
	return &item, nil
}

const candidateSelect = `
	SELECT c.id, c.p2h_report_id, c.unit_id, COALESCE(u.code, ''),
	       COALESCE(r.unit_type, ''), COALESCE(r.name, ''), r.report_date,
	       c.suggested_priority, c.status, COALESCE(c.findings_count, 0),
	       COALESCE(c.rusak_count, 0), c.decided_by, COALESCE(du.name, ''),
	       c.decided_at, COALESCE(c.skip_reason, ''), COALESCE(c.decision_note, ''),
	       c.wo_id, COALESCE(w.code, ''), c.notified_at, c.created_at
	FROM wo_candidates c
	JOIN p2h_reports r ON r.id = c.p2h_report_id
	LEFT JOIN units u ON u.id = c.unit_id
	LEFT JOIN users du ON du.id = c.decided_by
	LEFT JOIN work_orders w ON w.id = c.wo_id
`

type scanner interface {
	Scan(dest ...any) error
}

func scanCandidate(row scanner) (Candidate, error) {
	var item Candidate
	err := row.Scan(
		&item.ID, &item.P2HReportID, &item.UnitID, &item.UnitCode,
		&item.UnitType, &item.ReportName, &item.ReportDate,
		&item.SuggestedPriority, &item.Status, &item.FindingsCount,
		&item.RusakCount, &item.DecidedBy, &item.DecidedByName,
		&item.DecidedAt, &item.SkipReason, &item.DecisionNote,
		&item.WorkOrderID, &item.WorkOrderCode, &item.NotifiedAt, &item.CreatedAt,
	)
	if err != nil {
		return Candidate{}, err
	}
	return item, nil
}

func candidateSortColumn(raw string) string {
	switch raw {
	case "report_date":
		return "r.report_date"
	case "unit_code":
		return "u.code"
	case "findings_count":
		return "c.findings_count"
	case "suggested_priority":
		return "c.suggested_priority"
	case "status":
		return "c.status"
	default:
		return "c.created_at"
	}
}

func validateReason(raw string) error {
	_, err := validateReasonValue(raw)
	return err
}

func validateReasonValue(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	if len(reason) < minSkipReasonLen {
		return "", &ValidationError{Fields: map[string]string{"reason": "Alasan minimal 5 karakter."}}
	}
	if len(reason) > maxTextLen {
		return "", &ValidationError{Fields: map[string]string{"reason": "Alasan maksimal 1000 karakter."}}
	}
	return reason, nil
}

func candidateReportName(item Candidate) string {
	if strings.TrimSpace(item.ReportName) != "" {
		return item.ReportName
	}
	if strings.TrimSpace(item.UnitCode) != "" {
		return item.UnitCode
	}
	return fmt.Sprintf("P2H-%d", item.P2HReportID)
}

func buildDescription(item Candidate, findings []p2h.Finding, priority, note string) string {
	var builder strings.Builder
	builder.WriteString("Tindak lanjut temuan P2H ")
	builder.WriteString(candidateReportName(item))
	if item.UnitCode != "" {
		builder.WriteString(" · Unit ")
		builder.WriteString(item.UnitCode)
	}
	if item.ReportDate != nil {
		builder.WriteString(" · ")
		builder.WriteString(item.ReportDate.Format("2006-01-02"))
	}
	builder.WriteString("\nPrioritas: ")
	builder.WriteString(priority)
	for index, finding := range findings {
		if index >= 10 {
			builder.WriteString(fmt.Sprintf("\n- ... dan %d temuan lain", len(findings)-10))
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
	if strings.TrimSpace(note) != "" {
		builder.WriteString("\nCatatan approval: ")
		builder.WriteString(strings.TrimSpace(note))
	}
	return builder.String()
}

func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}
