package httpx

import (
	"time"

	"gsi-workorder/internal/candidates"
	"gsi-workorder/internal/p2h"
	"gsi-workorder/internal/workorders"
)

type reportResponse struct {
	ID              int64             `json:"id"`
	OdooReportID    int64             `json:"odoo_report_id"`
	Name            string            `json:"name"`
	ReportDate      string            `json:"report_date"`
	UnitID          *int64            `json:"unit_id,omitempty"`
	UnitCode        string            `json:"unit_code"`
	UnitType        string            `json:"unit_type"`
	Shift           string            `json:"shift"`
	ForemanName     string            `json:"foreman_name"`
	EmployeeName    string            `json:"employee_name"`
	Location        string            `json:"location"`
	State           string            `json:"state"`
	ApprovalStatus  string            `json:"approval_status"`
	KMAkhir         *float64          `json:"km_akhir,omitempty"`
	HMAwal          *float64          `json:"hm_awal,omitempty"`
	Notes           string            `json:"notes"`
	CheckedItems    int               `json:"checked_items"`
	FindingsCount   int               `json:"findings_count"`
	RusakCount      int               `json:"rusak_count"`
	DetailSyncedAt  *time.Time        `json:"detail_synced_at,omitempty"`
	SyncedAt        *time.Time        `json:"synced_at,omitempty"`
	CandidateID     *int64            `json:"candidate_id,omitempty"`
	CandidateStatus string            `json:"candidate_status"`
	Findings        []findingResponse `json:"findings"`
}

type findingResponse struct {
	ID         int64  `json:"id"`
	OdooLineID int64  `json:"odoo_line_id"`
	QuestionID *int64 `json:"question_id,omitempty"`
	Component  string `json:"component"`
	Category   string `json:"category"`
	Condition  string `json:"condition"`
	Remark     string `json:"remark"`
}

type candidateResponse struct {
	ID                int64      `json:"id"`
	P2HReportID       int64      `json:"p2h_report_id"`
	UnitID            *int64     `json:"unit_id,omitempty"`
	UnitCode          string     `json:"unit_code"`
	UnitType          string     `json:"unit_type"`
	ReportName        string     `json:"report_name"`
	ReportDate        string     `json:"report_date"`
	SuggestedPriority string     `json:"suggested_priority"`
	Status            string     `json:"status"`
	FindingsCount     int        `json:"findings_count"`
	RusakCount        int        `json:"rusak_count"`
	DecidedBy         *int64     `json:"decided_by,omitempty"`
	DecidedByName     string     `json:"decided_by_name"`
	DecidedAt         *time.Time `json:"decided_at,omitempty"`
	SkipReason        string     `json:"skip_reason"`
	DecisionNote      string     `json:"decision_note"`
	WorkOrderID       *int64     `json:"wo_id,omitempty"`
	WorkOrderCode     string     `json:"wo_code"`
	NotifiedAt        *time.Time `json:"notified_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type workOrderResponse struct {
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

type approveResponse struct {
	Candidate candidateResponse `json:"candidate"`
	WorkOrder workOrderResponse `json:"work_order"`
}

type bulkResponse struct {
	Approved []approveResponse   `json:"approved"`
	Skipped  []candidateResponse `json:"skipped"`
}

func newReportResponse(report p2h.Report) reportResponse {
	response := reportResponse{
		ID:              report.ID,
		OdooReportID:    report.OdooReportID,
		Name:            report.Name,
		ReportDate:      dateString(report.ReportDate),
		UnitID:          report.UnitID,
		UnitCode:        report.UnitCode,
		UnitType:        report.UnitType,
		Shift:           report.Shift,
		ForemanName:     report.ForemanName,
		EmployeeName:    report.EmployeeName,
		Location:        report.Location,
		State:           report.State,
		ApprovalStatus:  report.ApprovalStatus,
		KMAkhir:         report.KMAkhir,
		HMAwal:          report.HMAwal,
		Notes:           report.Notes,
		CheckedItems:    report.CheckedItems,
		FindingsCount:   report.FindingsCount,
		RusakCount:      report.RusakCount,
		DetailSyncedAt:  report.DetailSyncedAt,
		SyncedAt:        report.SyncedAt,
		CandidateID:     report.CandidateID,
		CandidateStatus: report.CandidateStatus,
		Findings:        newFindingResponses(report.Findings),
	}
	return response
}

func newFindingResponses(items []p2h.Finding) []findingResponse {
	responses := make([]findingResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, findingResponse{
			ID:         item.ID,
			OdooLineID: item.OdooLineID,
			QuestionID: item.QuestionID,
			Component:  item.Component,
			Category:   item.Category,
			Condition:  item.Condition,
			Remark:     item.Remark,
		})
	}
	return responses
}

func newCandidateResponse(item candidates.Candidate) candidateResponse {
	return candidateResponse{
		ID:                item.ID,
		P2HReportID:       item.P2HReportID,
		UnitID:            item.UnitID,
		UnitCode:          item.UnitCode,
		UnitType:          item.UnitType,
		ReportName:        item.ReportName,
		ReportDate:        dateString(item.ReportDate),
		SuggestedPriority: item.SuggestedPriority,
		Status:            item.Status,
		FindingsCount:     item.FindingsCount,
		RusakCount:        item.RusakCount,
		DecidedBy:         item.DecidedBy,
		DecidedByName:     item.DecidedByName,
		DecidedAt:         item.DecidedAt,
		SkipReason:        item.SkipReason,
		DecisionNote:      item.DecisionNote,
		WorkOrderID:       item.WorkOrderID,
		WorkOrderCode:     item.WorkOrderCode,
		NotifiedAt:        item.NotifiedAt,
		CreatedAt:         item.CreatedAt,
	}
}

func newCandidateResponses(items []candidates.Candidate) []candidateResponse {
	responses := make([]candidateResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, newCandidateResponse(item))
	}
	return responses
}

func newWorkOrderResponse(order workorders.WorkOrder) workOrderResponse {
	return workOrderResponse{
		ID:          order.ID,
		Code:        order.Code,
		UnitID:      order.UnitID,
		Source:      order.Source,
		P2HReportID: order.P2HReportID,
		CandidateID: order.CandidateID,
		Title:       order.Title,
		Description: order.Description,
		Priority:    order.Priority,
		Status:      order.Status,
		AssigneeID:  order.AssigneeID,
		CreatedBy:   order.CreatedBy,
		CreatedAt:   order.CreatedAt,
		UpdatedAt:   order.UpdatedAt,
	}
}

func dateString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}
