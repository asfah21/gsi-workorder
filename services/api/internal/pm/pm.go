package pm

import (
	"context"
	"errors"
	"time"

	"gsi-workorder/internal/workorders"
)

var (
	ErrNotFound   = errors.New("pm: not found")
	ErrConflict   = errors.New("pm: conflict")
	ErrValidation = errors.New("pm: validation")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "pm: validation failed" }
func (e *ValidationError) Unwrap() error { return ErrValidation }

type Actor struct {
	ID   int64
	Role string
}

type ChecklistItem struct {
	ID          int64     `json:"id"`
	PlanID      int64     `json:"pm_plan_id"`
	Sequence    int       `json:"sequence"`
	ComponentID *int64    `json:"component_id,omitempty"`
	Component   string    `json:"component"`
	Task        string    `json:"task"`
	Standard    string    `json:"standard"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ChecklistInput struct {
	Sequence    int
	ComponentID *int64
	Task        string
	Standard    string
}

type Plan struct {
	ID                    int64           `json:"id"`
	Name                  string          `json:"name"`
	Description           string          `json:"description"`
	UnitID                *int64          `json:"unit_id,omitempty"`
	UnitCode              string          `json:"unit_code"`
	UnitModelID           *int64          `json:"unit_model_id,omitempty"`
	UnitModelName         string          `json:"unit_model_name"`
	Status                string          `json:"status"`
	EffectiveFrom         time.Time       `json:"effective_from"`
	EffectiveTo           *time.Time      `json:"effective_to,omitempty"`
	IntervalHM            *float64        `json:"interval_hm,omitempty"`
	IntervalKM            *float64        `json:"interval_km,omitempty"`
	IntervalDays          *int            `json:"interval_days,omitempty"`
	IntervalRule          string          `json:"interval_rule"`
	UpcomingDays          *int            `json:"upcoming_days,omitempty"`
	DueSoonDays           *int            `json:"due_soon_days,omitempty"`
	UpcomingHM            *float64        `json:"upcoming_hm,omitempty"`
	UpcomingKM            *float64        `json:"upcoming_km,omitempty"`
	DueSoonHM             *float64        `json:"due_soon_hm,omitempty"`
	DueSoonKM             *float64        `json:"due_soon_km,omitempty"`
	AutoGenerateWO        bool            `json:"auto_generate_wo"`
	DefaultPriority       string          `json:"default_priority"`
	DefaultAssigneeID     *int64          `json:"default_assignee_id,omitempty"`
	DefaultAssigneeName   string          `json:"default_assignee_name"`
	DefaultSupervisorID   *int64          `json:"default_supervisor_id,omitempty"`
	DefaultSupervisorName string          `json:"default_supervisor_name"`
	CreatedBy             *int64          `json:"created_by,omitempty"`
	CreatedByName         string          `json:"created_by_name"`
	Checklist             []ChecklistItem `json:"checklist"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

type PlanInput struct {
	Name                string
	Description         string
	UnitID              *int64
	UnitModelID         *int64
	Status              string
	EffectiveFrom       time.Time
	EffectiveTo         *time.Time
	IntervalHM          *float64
	IntervalKM          *float64
	IntervalDays        *int
	IntervalRule        string
	UpcomingDays        *int
	DueSoonDays         *int
	UpcomingHM          *float64
	UpcomingKM          *float64
	DueSoonHM           *float64
	DueSoonKM           *float64
	AutoGenerateWO      bool
	DefaultPriority     string
	DefaultAssigneeID   *int64
	DefaultSupervisorID *int64
	Checklist           []ChecklistInput
}

type PlanPatch struct {
	Name                   *string
	Description            *string
	Status                 *string
	EffectiveFrom          *time.Time
	EffectiveTo            *time.Time
	ClearEffectiveTo       bool
	IntervalHM             *float64
	ClearIntervalHM        bool
	IntervalKM             *float64
	ClearIntervalKM        bool
	IntervalDays           *int
	ClearIntervalDays      bool
	IntervalRule           *string
	UpcomingDays           *int
	ClearUpcomingDays      bool
	DueSoonDays            *int
	ClearDueSoonDays       bool
	UpcomingHM             *float64
	ClearUpcomingHM        bool
	UpcomingKM             *float64
	ClearUpcomingKM        bool
	DueSoonHM              *float64
	ClearDueSoonHM         bool
	DueSoonKM              *float64
	ClearDueSoonKM         bool
	AutoGenerateWO         *bool
	DefaultPriority        *string
	DefaultAssigneeID      *int64
	ClearDefaultAssignee   bool
	DefaultSupervisorID    *int64
	ClearDefaultSupervisor bool
	Checklist              *[]ChecklistInput
}

type PlanFilter struct {
	Status      string
	Query       string
	UnitID      *int64
	UnitModelID *int64
	Page        int
	Limit       int
	Sort        string
	Order       string
}

type Schedule struct {
	ID              int64      `json:"id"`
	PlanID          int64      `json:"pm_plan_id"`
	PlanName        string     `json:"plan_name"`
	UnitID          int64      `json:"unit_id"`
	UnitCode        string     `json:"unit_code"`
	UnitModelName   string     `json:"unit_model_name"`
	CycleNo         int        `json:"cycle_no"`
	Status          string     `json:"status"`
	Bucket          string     `json:"bucket"`
	DaysToDue       *int       `json:"days_to_due,omitempty"`
	DueHM           *float64   `json:"due_hm,omitempty"`
	DueKM           *float64   `json:"due_km,omitempty"`
	DueDate         *time.Time `json:"due_date,omitempty"`
	LastCompletedHM *float64   `json:"last_completed_hm,omitempty"`
	LastCompletedKM *float64   `json:"last_completed_km,omitempty"`
	LastCompletedAt *time.Time `json:"last_completed_at,omitempty"`
	OverrideReason  string     `json:"override_reason"`
	OverriddenAt    *time.Time `json:"overridden_at,omitempty"`
	SkipReason      string     `json:"skip_reason"`
	SkippedAt       *time.Time `json:"skipped_at,omitempty"`
	CancelReason    string     `json:"cancel_reason"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	WorkOrderID     *int64     `json:"work_order_id,omitempty"`
	WorkOrderCode   string     `json:"work_order_code"`
	Plan            *Plan      `json:"plan,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type ScheduleFilter struct {
	Status     string
	Bucket     string
	Query      string
	UnitID     *int64
	PlanID     *int64
	ActiveOnly bool
	Page       int
	Limit      int
}

type GenerateResult struct {
	Schedule  Schedule             `json:"schedule"`
	WorkOrder workorders.WorkOrder `json:"work_order"`
}

type ComplianceRow struct {
	PlanID         int64   `json:"pm_plan_id"`
	PlanName       string  `json:"plan_name"`
	Target         string  `json:"target"`
	Total          int     `json:"total"`
	Completed      int     `json:"completed"`
	OnTime         int     `json:"on_time"`
	Overdue        int     `json:"overdue"`
	Missed         int     `json:"missed"`
	CompletionRate float64 `json:"completion_rate"`
	OnTimeRate     float64 `json:"on_time_rate"`
}

type Compliance struct {
	Rows           []ComplianceRow `json:"rows"`
	Total          int             `json:"total"`
	Completed      int             `json:"completed"`
	OnTime         int             `json:"on_time"`
	Overdue        int             `json:"overdue"`
	Missed         int             `json:"missed"`
	CompletionRate float64         `json:"completion_rate"`
	OnTimeRate     float64         `json:"on_time_rate"`
}

type ServiceRepository interface {
	ListPlans(ctx context.Context, filter PlanFilter) ([]Plan, int, error)
	GetPlan(ctx context.Context, id int64) (Plan, error)
	CreatePlan(ctx context.Context, actor Actor, input PlanInput) (Plan, error)
	UpdatePlan(ctx context.Context, id int64, actor Actor, patch PlanPatch) (Plan, error)
	SetPlanStatus(ctx context.Context, id int64, actor Actor, status string) (Plan, error)
	ListSchedules(ctx context.Context, filter ScheduleFilter) ([]Schedule, int, error)
	GetSchedule(ctx context.Context, id int64) (Schedule, error)
	GenerateWO(ctx context.Context, id int64, actor Actor) (GenerateResult, error)
	OverrideDue(ctx context.Context, id int64, actor Actor, input OverrideInput) (Schedule, error)
	Skip(ctx context.Context, id int64, actor Actor, reason string) (Schedule, error)
	Cancel(ctx context.Context, id int64, actor Actor, reason string) (Schedule, error)
	EvaluateDue(ctx context.Context, now time.Time) (EvaluateResult, error)
	Compliance(ctx context.Context, filter ComplianceFilter) (Compliance, error)
}

type OverrideInput struct {
	DueHM   *float64
	DueKM   *float64
	DueDate *time.Time
	Reason  string
}

type ComplianceFilter struct {
	PlanID *int64
	UnitID *int64
	From   *time.Time
	To     *time.Time
}

type EvaluateResult struct {
	Updated       int `json:"updated"`
	Notified      int `json:"notified"`
	AutoGenerated int `json:"auto_generated"`
}
