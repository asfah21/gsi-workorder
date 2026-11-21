package workorders

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("workorders: not found")
	ErrForbidden  = errors.New("workorders: forbidden")
	ErrConflict   = errors.New("workorders: conflict")
	ErrValidation = errors.New("workorders: validation")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "workorders: validation failed"
}

type Actor struct {
	ID   int64
	Role string
}

type Filter struct {
	Status     string
	Source     string
	Query      string
	UnitID     *int64
	AssigneeID *int64
	Page       int
	Limit      int
	Sort       string
	Order      string
	Actor      Actor
}

type CreateManualInput struct {
	UnitID          int64
	Title           string
	Description     string
	Priority        string
	MaintenanceType string
	AssigneeID      *int64
	SupervisorID    *int64
	DueDate         *time.Time
}

type PatchInput struct {
	Status          string
	Note            string
	Reason          string
	Priority        *string
	MaintenanceType *string
	AssigneeID      *int64
	SupervisorID    *int64
	ClearAssignee   bool
	ClearSupervisor bool
	Title           *string
	Description     *string
}

type UpdateRow struct {
	ID         int64     `json:"id"`
	WOID       int64     `json:"wo_id"`
	UserID     *int64    `json:"user_id,omitempty"`
	UserName   string    `json:"user_name"`
	Note       string    `json:"note"`
	StatusFrom string    `json:"status_from"`
	StatusTo   string    `json:"status_to"`
	CreatedAt  time.Time `json:"created_at"`
}

type Attachment struct {
	ID         int64     `json:"id"`
	WOID       int64     `json:"wo_id"`
	FilePath   string    `json:"file_path"`
	Mime       string    `json:"mime"`
	SizeBytes  int       `json:"size_bytes"`
	UploadedBy *int64    `json:"uploaded_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Finding struct {
	ID         int64  `json:"id"`
	OdooLineID int64  `json:"odoo_line_id"`
	QuestionID *int64 `json:"question_id,omitempty"`
	Component  string `json:"component"`
	Category   string `json:"category"`
	Condition  string `json:"condition"`
	Remark     string `json:"remark"`
}

type Detail struct {
	WorkOrder
	Findings    []Finding    `json:"findings"`
	Updates     []UpdateRow  `json:"updates"`
	Attachments []Attachment `json:"attachments"`
}

type Repository struct {
	pool *pgxpool.Pool
	loc  *time.Location
}

func NewRepository(pool *pgxpool.Pool, loc *time.Location) *Repository {
	if loc == nil {
		loc = time.UTC
	}
	return &Repository{pool: pool, loc: loc}
}

const workOrderSelect = `
	SELECT w.id, w.code, w.unit_id, COALESCE(u.code, ''), w.source,
	       w.p2h_report_id, w.candidate_id, w.title, COALESCE(w.description, ''),
	       COALESCE(w.priority, 'medium'), w.status,
	       COALESCE(w.maintenance_type, 'corrective'),
	       w.assignee_id, COALESCE(a.name, ''),
	       w.supervisor_id, COALESCE(s.name, ''),
	       w.created_by, COALESCE(cb.name, ''),
	       w.due_date, w.started_at, w.closed_at, w.created_at, w.updated_at
	FROM work_orders w
	LEFT JOIN units u ON u.id = w.unit_id
	LEFT JOIN users a ON a.id = w.assignee_id
	LEFT JOIN users s ON s.id = w.supervisor_id
	LEFT JOIN users cb ON cb.id = w.created_by
`

func (r *Repository) List(ctx context.Context, filter Filter) ([]WorkOrder, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	args := make([]any, 0, 8)
	where := []string{"1 = 1"}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("w.status = $%d", len(args)))
	}
	if filter.Source != "" {
		args = append(args, filter.Source)
		where = append(where, fmt.Sprintf("w.source = $%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+filter.Query+"%")
		where = append(where, fmt.Sprintf("(w.code ILIKE $%d OR w.title ILIKE $%d OR COALESCE(u.code, '') ILIKE $%d)", len(args), len(args), len(args)))
	}
	if filter.UnitID != nil {
		args = append(args, *filter.UnitID)
		where = append(where, fmt.Sprintf("w.unit_id = $%d", len(args)))
	}
	if filter.AssigneeID != nil {
		args = append(args, *filter.AssigneeID)
		where = append(where, fmt.Sprintf("w.assignee_id = $%d", len(args)))
	}
	if filter.Actor.Role == "mechanic" {
		args = append(args, filter.Actor.ID)
		where = append(where, fmt.Sprintf("(w.assignee_id = $%d OR w.assignee_id IS NULL)", len(args)))
	}

	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM work_orders w
		LEFT JOIN units u ON u.id = w.unit_id
		WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count work orders: %w", err)
	}

	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	orderColumn := workOrderSortColumn(filter.Sort)
	orderDirection := "ASC"
	if strings.EqualFold(filter.Order, "desc") {
		orderDirection = "DESC"
	}
	rows, err := r.pool.Query(ctx, workOrderSelect+`
		WHERE `+whereSQL+`
		ORDER BY `+orderColumn+` `+orderDirection+`, w.id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list work orders: %w", err)
	}
	defer rows.Close()

	items := make([]WorkOrder, 0)
	for rows.Next() {
		item, err := scanWorkOrder(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan work order: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate work orders: %w", err)
	}
	return items, total, nil
}

func (r *Repository) Get(ctx context.Context, id int64, actor Actor) (Detail, error) {
	item, err := r.getWorkOrder(ctx, r.pool, id)
	if err != nil {
		return Detail{}, err
	}
	if !canView(actor, item) {
		return Detail{}, ErrForbidden
	}

	findings, err := r.findings(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	updates, err := r.updates(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	attachments, err := r.attachments(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	return Detail{WorkOrder: item, Findings: findings, Updates: updates, Attachments: attachments}, nil
}

func (r *Repository) CreateManual(ctx context.Context, actor Actor, input CreateManualInput) (WorkOrder, error) {
	if actor.Role != "admin" && actor.Role != "planner" {
		return WorkOrder{}, ErrForbidden
	}
	title := strings.TrimSpace(input.Title)
	if len(title) < 3 || len(title) > 200 {
		return WorkOrder{}, &ValidationError{Fields: map[string]string{"title": "Judul wajib 3-200 karakter."}}
	}
	if len(strings.TrimSpace(input.Description)) > 2000 {
		return WorkOrder{}, &ValidationError{Fields: map[string]string{"description": "Deskripsi maksimal 2000 karakter."}}
	}
	priority := strings.TrimSpace(input.Priority)
	if priority == "" {
		priority = "medium"
	}
	if !validPriority(priority) {
		return WorkOrder{}, &ValidationError{Fields: map[string]string{"priority": "Prioritas tidak valid."}}
	}
	maintenanceType := strings.TrimSpace(input.MaintenanceType)
	if maintenanceType == "" {
		maintenanceType = "corrective"
	}
	if !validMaintenanceType(maintenanceType) {
		return WorkOrder{}, &ValidationError{Fields: map[string]string{"maintenance_type": "Tipe maintenance tidak valid."}}
	}
	if input.UnitID <= 0 {
		return WorkOrder{}, &ValidationError{Fields: map[string]string{"unit_id": "Unit wajib dipilih."}}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return WorkOrder{}, fmt.Errorf("begin manual work order: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureUnit(ctx, tx, input.UnitID); err != nil {
		return WorkOrder{}, err
	}
	if err := ensureUser(ctx, tx, input.AssigneeID, "assignee"); err != nil {
		return WorkOrder{}, err
	}
	if err := ensureUser(ctx, tx, input.SupervisorID, "supervisor"); err != nil {
		return WorkOrder{}, err
	}

	order, err := Create(ctx, tx, CreateInput{
		Source:          "manual",
		UnitID:          &input.UnitID,
		Title:           title,
		Description:     strings.TrimSpace(input.Description),
		Priority:        priority,
		Status:          "open",
		MaintenanceType: maintenanceType,
		AssigneeID:      input.AssigneeID,
		SupervisorID:    input.SupervisorID,
		DueDate:         input.DueDate,
		CreatedBy:       &actor.ID,
		Now:             time.Now(),
		Location:        r.loc,
	})
	if err != nil {
		return WorkOrder{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkOrder{}, fmt.Errorf("commit manual work order: %w", err)
	}

	detail, err := r.Get(ctx, order.ID, actor)
	if err != nil {
		return WorkOrder{}, err
	}
	return detail.WorkOrder, nil
}

func (r *Repository) Update(ctx context.Context, id int64, actor Actor, input PatchInput) (WorkOrder, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return WorkOrder{}, fmt.Errorf("begin work-order update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	item, err := r.getWorkOrder(ctx, tx, id, true)
	if err != nil {
		return WorkOrder{}, err
	}
	if !canModify(actor, item) {
		return WorkOrder{}, ErrForbidden
	}
	original := item

	now := time.Now()
	statusFrom := item.Status
	statusTo := item.Status
	note := strings.TrimSpace(input.Note)
	reason := strings.TrimSpace(input.Reason)
	if note == "" {
		note = reason
	}

	if input.Status != "" && input.Status != item.Status {
		statusTo = input.Status
		if err := validateTransition(item, actor, input, note); err != nil {
			return WorkOrder{}, err
		}
		switch {
		case statusTo == "in_progress":
			if item.UnitID != nil {
				if err := ensureUnit(ctx, tx, *item.UnitID); err != nil {
					return WorkOrder{}, err
				}
			}
			if item.AssigneeID == nil {
				if input.AssigneeID != nil {
					item.AssigneeID = input.AssigneeID
				} else if actor.Role == "mechanic" {
					item.AssigneeID = &actor.ID
				}
			}
			if item.AssigneeID == nil {
				return WorkOrder{}, &ValidationError{Fields: map[string]string{"assignee_id": "Assignee wajib diisi sebelum mulai."}}
			}
			if item.StartedAt == nil {
				item.StartedAt = &now
			}
		case statusTo == "done":
			item.ClosedAt = &now
		case statusTo == "cancelled":
			item.ClosedAt = &now
		}
	}

	if input.Title != nil {
		if actor.Role == "mechanic" {
			return WorkOrder{}, ErrForbidden
		}
		title := strings.TrimSpace(*input.Title)
		if len(title) < 3 || len(title) > 200 {
			return WorkOrder{}, &ValidationError{Fields: map[string]string{"title": "Judul wajib 3-200 karakter."}}
		}
		item.Title = title
	}
	if input.Description != nil {
		if actor.Role == "mechanic" {
			return WorkOrder{}, ErrForbidden
		}
		description := strings.TrimSpace(*input.Description)
		if len(description) > 2000 {
			return WorkOrder{}, &ValidationError{Fields: map[string]string{"description": "Deskripsi maksimal 2000 karakter."}}
		}
		item.Description = description
	}
	if input.Priority != nil {
		priority := strings.TrimSpace(*input.Priority)
		if !validPriority(priority) {
			return WorkOrder{}, &ValidationError{Fields: map[string]string{"priority": "Prioritas tidak valid."}}
		}
		item.Priority = priority
	}
	if input.MaintenanceType != nil {
		if actor.Role == "mechanic" {
			return WorkOrder{}, ErrForbidden
		}
		maintenanceType := strings.TrimSpace(*input.MaintenanceType)
		if !validMaintenanceType(maintenanceType) {
			return WorkOrder{}, &ValidationError{Fields: map[string]string{"maintenance_type": "Tipe maintenance tidak valid."}}
		}
		item.MaintenanceType = maintenanceType
	}
	if input.ClearAssignee {
		if actor.Role == "mechanic" {
			return WorkOrder{}, ErrForbidden
		}
		item.AssigneeID = nil
	} else if input.AssigneeID != nil {
		if actor.Role == "mechanic" && *input.AssigneeID != actor.ID {
			return WorkOrder{}, ErrForbidden
		}
		if err := ensureUser(ctx, tx, input.AssigneeID, "assignee"); err != nil {
			return WorkOrder{}, err
		}
		item.AssigneeID = input.AssigneeID
	}
	if input.ClearSupervisor {
		if actor.Role == "mechanic" {
			return WorkOrder{}, ErrForbidden
		}
		item.SupervisorID = nil
	} else if input.SupervisorID != nil {
		if actor.Role == "mechanic" {
			return WorkOrder{}, ErrForbidden
		}
		if err := ensureUser(ctx, tx, input.SupervisorID, "supervisor"); err != nil {
			return WorkOrder{}, err
		}
		item.SupervisorID = input.SupervisorID
	}
	if (statusTo == "in_progress" || statusTo == "waiting_part") && item.AssigneeID == nil {
		return WorkOrder{}, &ValidationError{Fields: map[string]string{"assignee_id": "Assignee wajib diisi untuk status ini."}}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE work_orders
		SET title = $2,
		    description = NULLIF($3, ''),
		    priority = $4,
		    status = $5,
		    maintenance_type = $6,
		    assignee_id = $7,
		    supervisor_id = $8,
		    started_at = $9,
		    closed_at = $10,
		    updated_at = now()
		WHERE id = $1
	`, id, item.Title, item.Description, item.Priority, statusTo, item.MaintenanceType,
		item.AssigneeID, item.SupervisorID, nullTimestamp(item.StartedAt), nullTimestamp(item.ClosedAt)); err != nil {
		return WorkOrder{}, fmt.Errorf("update work order %d: %w", id, err)
	}

	if note == "" {
		note = changeSummary(original, item)
	}
	if statusFrom != statusTo || note != "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO wo_updates (wo_id, user_id, note, status_from, status_to)
			VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''))
		`, id, actor.ID, note, statusFrom, statusTo); err != nil {
			return WorkOrder{}, fmt.Errorf("append work-order update: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return WorkOrder{}, fmt.Errorf("commit work-order update: %w", err)
	}
	detail, err := r.Get(ctx, id, actor)
	if err != nil {
		return WorkOrder{}, err
	}
	return detail.WorkOrder, nil
}

func (r *Repository) AddUpdate(ctx context.Context, id int64, actor Actor, note string) (UpdateRow, error) {
	item, err := r.getWorkOrder(ctx, r.pool, id)
	if err != nil {
		return UpdateRow{}, err
	}
	if !canModify(actor, item) {
		return UpdateRow{}, ErrForbidden
	}
	note = strings.TrimSpace(note)
	if note == "" || len(note) > 1000 {
		return UpdateRow{}, &ValidationError{Fields: map[string]string{"note": "Catatan wajib 1-1000 karakter."}}
	}

	var update UpdateRow
	err = r.pool.QueryRow(ctx, `
		INSERT INTO wo_updates (wo_id, user_id, note)
		VALUES ($1, $2, $3)
		RETURNING id, wo_id, user_id, note, COALESCE(status_from, ''), COALESCE(status_to, ''), created_at
	`, id, actor.ID, note).Scan(&update.ID, &update.WOID, &update.UserID, &update.Note,
		&update.StatusFrom, &update.StatusTo, &update.CreatedAt)
	if err != nil {
		return UpdateRow{}, fmt.Errorf("add work-order note: %w", err)
	}
	return update, nil
}

func (r *Repository) AddAttachment(ctx context.Context, id int64, actor Actor, filePath, mime string, size int) (Attachment, error) {
	item, err := r.getWorkOrder(ctx, r.pool, id)
	if err != nil {
		return Attachment{}, err
	}
	if !canModify(actor, item) {
		return Attachment{}, ErrForbidden
	}
	if strings.TrimSpace(filePath) == "" || strings.TrimSpace(mime) == "" || size <= 0 {
		return Attachment{}, &ValidationError{Fields: map[string]string{"file": "Lampiran tidak valid."}}
	}
	var attachment Attachment
	err = r.pool.QueryRow(ctx, `
		INSERT INTO wo_attachments (wo_id, file_path, mime, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, wo_id, file_path, mime, size_bytes, uploaded_by, created_at
	`, id, filePath, mime, size, actor.ID).Scan(&attachment.ID, &attachment.WOID,
		&attachment.FilePath, &attachment.Mime, &attachment.SizeBytes,
		&attachment.UploadedBy, &attachment.CreatedAt)
	if err != nil {
		return Attachment{}, fmt.Errorf("add work-order attachment: %w", err)
	}
	return attachment, nil
}

func (r *Repository) GetAttachment(ctx context.Context, id, attachmentID int64, actor Actor) (Attachment, error) {
	item, err := r.getWorkOrder(ctx, r.pool, id)
	if err != nil {
		return Attachment{}, err
	}
	if !canView(actor, item) {
		return Attachment{}, ErrForbidden
	}
	var attachment Attachment
	err = r.pool.QueryRow(ctx, `
		SELECT id, wo_id, file_path, COALESCE(mime, ''), COALESCE(size_bytes, 0),
		       uploaded_by, created_at
		FROM wo_attachments
		WHERE id = $1 AND wo_id = $2
	`, attachmentID, id).Scan(&attachment.ID, &attachment.WOID, &attachment.FilePath,
		&attachment.Mime, &attachment.SizeBytes, &attachment.UploadedBy, &attachment.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	if err != nil {
		return Attachment{}, fmt.Errorf("get work-order attachment: %w", err)
	}
	return attachment, nil
}

func (r *Repository) getWorkOrder(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, id int64, forUpdate ...bool) (WorkOrder, error) {
	query := workOrderSelect + ` WHERE w.id = $1`
	if len(forUpdate) > 0 && forUpdate[0] {
		query += ` FOR UPDATE OF w`
	}
	item, err := scanWorkOrder(q.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkOrder{}, ErrNotFound
	}
	if err != nil {
		return WorkOrder{}, fmt.Errorf("get work order %d: %w", id, err)
	}
	return item, nil
}

func (r *Repository) findings(ctx context.Context, woID int64) ([]Finding, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT f.id, f.odoo_line_id, f.question_id, COALESCE(f.component, ''),
		       COALESCE(f.category, ''), f.condition, COALESCE(f.remark, '')
		FROM wo_findings wf
		JOIN p2h_findings f ON f.id = wf.finding_id
		WHERE wf.wo_id = $1
		ORDER BY f.id
	`, woID)
	if err != nil {
		return nil, fmt.Errorf("list work-order findings: %w", err)
	}
	defer rows.Close()
	items := make([]Finding, 0)
	for rows.Next() {
		var item Finding
		if err := rows.Scan(&item.ID, &item.OdooLineID, &item.QuestionID, &item.Component,
			&item.Category, &item.Condition, &item.Remark); err != nil {
			return nil, fmt.Errorf("scan work-order finding: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) updates(ctx context.Context, woID int64) ([]UpdateRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.wo_id, u.user_id, COALESCE(usr.name, ''),
		       COALESCE(u.note, ''), COALESCE(u.status_from, ''),
		       COALESCE(u.status_to, ''), u.created_at
		FROM wo_updates u
		LEFT JOIN users usr ON usr.id = u.user_id
		WHERE u.wo_id = $1
		ORDER BY u.created_at ASC, u.id ASC
	`, woID)
	if err != nil {
		return nil, fmt.Errorf("list work-order updates: %w", err)
	}
	defer rows.Close()
	items := make([]UpdateRow, 0)
	for rows.Next() {
		var item UpdateRow
		if err := rows.Scan(&item.ID, &item.WOID, &item.UserID, &item.UserName,
			&item.Note, &item.StatusFrom, &item.StatusTo, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan work-order update: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) attachments(ctx context.Context, woID int64) ([]Attachment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, wo_id, file_path, COALESCE(mime, ''), COALESCE(size_bytes, 0),
		       uploaded_by, created_at
		FROM wo_attachments
		WHERE wo_id = $1
		ORDER BY created_at ASC, id ASC
	`, woID)
	if err != nil {
		return nil, fmt.Errorf("list work-order attachments: %w", err)
	}
	defer rows.Close()
	items := make([]Attachment, 0)
	for rows.Next() {
		var item Attachment
		if err := rows.Scan(&item.ID, &item.WOID, &item.FilePath, &item.Mime,
			&item.SizeBytes, &item.UploadedBy, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan work-order attachment: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanWorkOrder(row interface{ Scan(...any) error }) (WorkOrder, error) {
	var item WorkOrder
	var due pgtype.Date
	var started, closed pgtype.Timestamptz
	err := row.Scan(
		&item.ID, &item.Code, &item.UnitID, &item.UnitCode, &item.Source,
		&item.P2HReportID, &item.CandidateID, &item.Title, &item.Description,
		&item.Priority, &item.Status, &item.MaintenanceType,
		&item.AssigneeID, &item.AssigneeName, &item.SupervisorID, &item.SupervisorName,
		&item.CreatedBy, &item.CreatedByName, &due, &started,
		&closed, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return WorkOrder{}, err
	}
	if due.Valid {
		value := due.Time
		item.DueDate = &value
	}
	if started.Valid {
		value := started.Time
		item.StartedAt = &value
	}
	if closed.Valid {
		value := closed.Time
		item.ClosedAt = &value
	}
	return item, nil
}

func nullTimestamp(value *time.Time) any {
	if value == nil {
		return nil
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func workOrderSortColumn(raw string) string {
	switch raw {
	case "code":
		return "w.code"
	case "title":
		return "w.title"
	case "priority":
		return "w.priority"
	case "status":
		return "w.status"
	case "updated_at":
		return "w.updated_at"
	default:
		return "w.created_at"
	}
}

func changeSummary(before, after WorkOrder) string {
	changes := make([]string, 0)
	if before.Title != after.Title {
		changes = append(changes, "judul")
	}
	if before.Description != after.Description {
		changes = append(changes, "deskripsi")
	}
	if before.Priority != after.Priority {
		changes = append(changes, "prioritas")
	}
	if before.MaintenanceType != after.MaintenanceType {
		changes = append(changes, "tipe maintenance")
	}
	if !sameID(before.AssigneeID, after.AssigneeID) {
		changes = append(changes, "assignee")
	}
	if !sameID(before.SupervisorID, after.SupervisorID) {
		changes = append(changes, "supervisor")
	}
	if len(changes) == 0 {
		return ""
	}
	return "Perubahan: " + strings.Join(changes, ", ")
}

func sameID(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func validPriority(value string) bool {
	switch value {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func validMaintenanceType(value string) bool {
	switch value {
	case "preventive", "corrective", "breakdown", "inspection":
		return true
	default:
		return false
	}
}

func canView(actor Actor, item WorkOrder) bool {
	switch actor.Role {
	case "admin", "planner", "storekeeper", "viewer":
		return true
	case "mechanic":
		return item.AssigneeID == nil || *item.AssigneeID == actor.ID
	default:
		return false
	}
}

func canModify(actor Actor, item WorkOrder) bool {
	switch actor.Role {
	case "admin", "planner":
		return true
	case "mechanic":
		return item.AssigneeID == nil || *item.AssigneeID == actor.ID
	default:
		return false
	}
}

func validateTransition(item WorkOrder, actor Actor, input PatchInput, note string) error {
	from := item.Status
	to := input.Status
	if from == "done" || from == "cancelled" {
		return ErrConflict
	}
	switch {
	case from == "open" && to == "in_progress":
	case from == "in_progress" && to == "waiting_part":
		if len(note) < 5 {
			return &ValidationError{Fields: map[string]string{"note": "Alasan waiting part wajib minimal 5 karakter."}}
		}
	case from == "waiting_part" && to == "in_progress":
	case from == "in_progress" && to == "done":
		if len(note) < 5 {
			return &ValidationError{Fields: map[string]string{"note": "Alasan penutupan wajib minimal 5 karakter."}}
		}
	case to == "cancelled" && (from == "open" || from == "in_progress" || from == "waiting_part"):
		if actor.Role != "admin" && actor.Role != "planner" {
			return ErrForbidden
		}
		if len(note) < 5 {
			return &ValidationError{Fields: map[string]string{"reason": "Alasan pembatalan wajib minimal 5 karakter."}}
		}
	default:
		return ErrConflict
	}
	return nil
}

func ensureUnit(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, unitID int64) error {
	var exists int
	if err := q.QueryRow(ctx, `SELECT 1 FROM units WHERE id = $1 AND is_active = true`, unitID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &ValidationError{Fields: map[string]string{"unit_id": "Unit tidak ditemukan atau tidak aktif."}}
		}
		return fmt.Errorf("validate unit: %w", err)
	}
	return nil
}

func ensureUser(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, userID *int64, role string) error {
	if userID == nil {
		return nil
	}
	var exists int
	query := `SELECT 1 FROM users WHERE id = $1 AND is_active = true`
	switch role {
	case "supervisor":
		query += ` AND role IN ('admin', 'planner')`
	case "assignee":
		query += ` AND role IN ('admin', 'planner', 'mechanic')`
	}
	if err := q.QueryRow(ctx, query, *userID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			field := "assignee_id"
			if role == "supervisor" {
				field = "supervisor_id"
			}
			return &ValidationError{Fields: map[string]string{field: "User tidak ditemukan atau tidak aktif."}}
		}
		return fmt.Errorf("validate user: %w", err)
	}
	return nil
}
