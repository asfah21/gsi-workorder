// Package workorders contains the small M2 work-order creation primitive used
// by P2H candidate approval and P2H auto-WO. Lifecycle/transition rules remain
// M3 scope.
package workorders

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// Querier is implemented by pgx.Tx and pgxpool.Pool.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type CreateInput struct {
	Source          string
	UnitID          *int64
	P2HReportID     *int64
	CandidateID     *int64
	PMScheduleID    *int64
	Title           string
	Description     string
	Priority        string
	Status          string
	MaintenanceType string
	AssigneeID      *int64
	SupervisorID    *int64
	DueDate         *time.Time
	CreatedBy       *int64
	Now             time.Time
	Location        *time.Location
}

type WorkOrder struct {
	ID              int64      `json:"id"`
	Code            string     `json:"code"`
	UnitID          *int64     `json:"unit_id,omitempty"`
	UnitCode        string     `json:"unit_code"`
	Source          string     `json:"source"`
	P2HReportID     *int64     `json:"p2h_report_id,omitempty"`
	CandidateID     *int64     `json:"candidate_id,omitempty"`
	PMScheduleID    *int64     `json:"pm_schedule_id,omitempty"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status"`
	MaintenanceType string     `json:"maintenance_type"`
	AssigneeID      *int64     `json:"assignee_id,omitempty"`
	AssigneeName    string     `json:"assignee_name"`
	SupervisorID    *int64     `json:"supervisor_id,omitempty"`
	SupervisorName  string     `json:"supervisor_name"`
	CreatedBy       *int64     `json:"created_by,omitempty"`
	CreatedByName   string     `json:"created_by_name"`
	DueDate         *time.Time `json:"due_date,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Create inserts a work order inside the caller's transaction. The monthly
// counter is incremented in the same transaction, so two concurrent approvals
// cannot receive the same code.
func Create(ctx context.Context, q Querier, input CreateInput) (WorkOrder, error) {
	if input.Source == "" {
		input.Source = "p2h"
	}
	if input.Status == "" {
		input.Status = "open"
	}
	if input.Priority == "" {
		input.Priority = "medium"
	}
	if input.MaintenanceType == "" {
		input.MaintenanceType = "corrective"
	}
	if input.Now.IsZero() {
		input.Now = time.Now()
	}
	if input.Location == nil {
		return WorkOrder{}, fmt.Errorf("workorders: location is required")
	}

	period := input.Now.In(input.Location).Format("200601")
	var number int
	if err := q.QueryRow(ctx, `
		INSERT INTO work_order_code_counters (period, last_number)
		VALUES ($1, 1)
		ON CONFLICT (period) DO UPDATE
		SET last_number = work_order_code_counters.last_number + 1,
		    updated_at = now()
		RETURNING last_number
	`, period).Scan(&number); err != nil {
		return WorkOrder{}, fmt.Errorf("next work-order number: %w", err)
	}

	code := fmt.Sprintf("WO-%s-%04d", period, number)
	order := WorkOrder{
		Code:            code,
		UnitID:          input.UnitID,
		Source:          input.Source,
		P2HReportID:     input.P2HReportID,
		CandidateID:     input.CandidateID,
		PMScheduleID:    input.PMScheduleID,
		Title:           input.Title,
		Description:     input.Description,
		Priority:        input.Priority,
		Status:          input.Status,
		MaintenanceType: input.MaintenanceType,
		AssigneeID:      input.AssigneeID,
		SupervisorID:    input.SupervisorID,
		DueDate:         input.DueDate,
		CreatedBy:       input.CreatedBy,
	}
	if err := q.QueryRow(ctx, `
		INSERT INTO work_orders (
			code, unit_id, source, p2h_report_id, candidate_id, pm_schedule_id,
			title, description, priority, status, maintenance_type, assignee_id,
			supervisor_id, due_date, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`, code, input.UnitID, input.Source, input.P2HReportID, input.CandidateID,
		input.PMScheduleID, input.Title, input.Description, input.Priority,
		input.Status, input.MaintenanceType, input.AssigneeID, input.SupervisorID,
		nullDate(input.DueDate), input.CreatedBy).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt); err != nil {
		return WorkOrder{}, fmt.Errorf("insert work order: %w", err)
	}
	return order, nil
}

func nullDate(value *time.Time) any {
	if value == nil {
		return nil
	}
	return pgtype.Date{Time: *value, Valid: true}
}
