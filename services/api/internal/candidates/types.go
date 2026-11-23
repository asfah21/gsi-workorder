package candidates

import (
	"errors"
	"time"

	"gsi-workorder/internal/p2h"
	"gsi-workorder/internal/workorders"
)

const (
	MaxBulk          = 50
	DefaultPageSize  = 25
	MaxPageSize      = 100
	minSkipReasonLen = 5
	maxTextLen       = 1000

	StatusPending   = "pending"
	StatusValidated = "validated"
	StatusConverted = "converted"
	StatusSkipped   = "skipped"
)

var (
	ErrNotFound   = errors.New("candidates: not found")
	ErrConflict   = errors.New("candidates: conflict")
	ErrValidation = errors.New("candidates: validation failed")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "candidates: validation failed" }
func (e *ValidationError) Unwrap() error { return ErrValidation }

type Candidate struct {
	ID                int64      `json:"id"`
	P2HReportID       *int64     `json:"p2h_report_id,omitempty"`
	Source            string     `json:"source"`
	UnitID            *int64     `json:"unit_id,omitempty"`
	UnitCode          string     `json:"unit_code"`
	UnitType          string     `json:"unit_type"`
	ReportName        string     `json:"report_name"`
	ReportDate        *time.Time `json:"report_date,omitempty"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	WorkCategory      string     `json:"work_category"`
	IsBreakdown       bool       `json:"is_breakdown"`
	SuggestedPriority string     `json:"suggested_priority"`
	Status            string     `json:"status"`
	FindingsCount     int        `json:"findings_count"`
	RusakCount        int        `json:"rusak_count"`
	DecidedBy         *int64     `json:"decided_by,omitempty"`
	DecidedByName     string     `json:"decided_by_name"`
	DecidedAt         *time.Time `json:"decided_at,omitempty"`
	SkipReason        string     `json:"skip_reason"`
	DecisionNote      string     `json:"decision_note"`
	CreatedBy         *int64     `json:"created_by,omitempty"`
	CreatedByName     string     `json:"created_by_name"`
	ValidatedBy       *int64     `json:"validated_by,omitempty"`
	ValidatedByName   string     `json:"validated_by_name"`
	ValidatedAt       *time.Time `json:"validated_at,omitempty"`
	ConvertedBy       *int64     `json:"converted_by,omitempty"`
	ConvertedByName   string     `json:"converted_by_name"`
	ConvertedAt       *time.Time `json:"converted_at,omitempty"`
	WorkOrderID       *int64     `json:"wo_id,omitempty"`
	WorkOrderCode     string     `json:"wo_code"`
	NotifiedAt        *time.Time `json:"notified_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type WorkOrderSummary struct {
	ID           int64     `json:"id"`
	Code         string    `json:"code"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Priority     string    `json:"priority"`
	Status       string    `json:"status"`
	AssigneeID   *int64    `json:"assignee_id,omitempty"`
	AssigneeName string    `json:"assignee_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type Detail struct {
	Candidate Candidate
	Report    *p2h.Report
	WorkOrder *WorkOrderSummary
}

type Filter struct {
	Status   string
	UnitType string
	Query    string
	Backlog  bool
	Page     int
	Limit    int
	Sort     string
	Order    string
}

type CreateInput struct {
	UnitID       int64
	Title        string
	Description  string
	WorkCategory string
	IsBreakdown  bool
	Priority     string
	ActorID      int64
}

type ValidateInput struct {
	Note string
}

type ApproveInput struct {
	Priority   string
	AssigneeID *int64
	Note       string
}

type SkipInput struct {
	Reason string
}

type ApproveResult struct {
	Candidate Candidate
	WorkOrder workorders.WorkOrder
}

type BulkInput struct {
	IDs        []int64
	Action     string
	Reason     string
	Priority   string
	AssigneeID *int64
	Note       string
}

type BulkResult struct {
	Approved  []ApproveResult
	Validated []Candidate
	Converted []ApproveResult
	Skipped   []Candidate
}

func ValidPriority(priority string) bool {
	switch priority {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func ValidWorkCategory(category string) bool {
	switch category {
	case "Service", "Scheduled", "Unscheduled", "Incident", "UBOP":
		return true
	default:
		return false
	}
}
