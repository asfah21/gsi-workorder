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
)

// Querier is implemented by pgx.Tx and pgxpool.Pool.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type CreateInput struct {
	Source      string
	UnitID      *int64
	P2HReportID *int64
	CandidateID *int64
	Title       string
	Description string
	Priority    string
	Status      string
	AssigneeID  *int64
	CreatedBy   *int64
	Now         time.Time
	Location    *time.Location
}

type WorkOrder struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	UnitID      *int64    `json:"unit_id,omitempty"`
	Source      string    `json:"source"`
	P2HReportID *int64    `json:"p2h_report_id,omitempty"`
	CandidateID *int64    `json:"candidate_id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	AssigneeID  *int64    `json:"assignee_id,omitempty"`
	CreatedBy   *int64    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
		Code:        code,
		UnitID:      input.UnitID,
		Source:      input.Source,
		P2HReportID: input.P2HReportID,
		CandidateID: input.CandidateID,
		Title:       input.Title,
		Description: input.Description,
		Priority:    input.Priority,
		Status:      input.Status,
		AssigneeID:  input.AssigneeID,
		CreatedBy:   input.CreatedBy,
	}
	if err := q.QueryRow(ctx, `
		INSERT INTO work_orders (
			code, unit_id, source, p2h_report_id, candidate_id, title, description,
			priority, status, assignee_id, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`, code, input.UnitID, input.Source, input.P2HReportID, input.CandidateID,
		input.Title, input.Description, input.Priority, input.Status,
		input.AssigneeID, input.CreatedBy).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt); err != nil {
		return WorkOrder{}, fmt.Errorf("insert work order: %w", err)
	}
	return order, nil
}
