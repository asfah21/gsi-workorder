package units

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
)

// HistoryFilter is shared by the unit timeline and component history queries.
type HistoryFilter struct {
	Page            int
	Limit           int
	Status          string
	MaintenanceType string
	From            *time.Time
	To              *time.Time
}

type MaintenanceHistory struct {
	WOID            int64      `json:"wo_id"`
	UnitID          int64      `json:"unit_id"`
	UnitCode        string     `json:"unit_code"`
	WOCode          string     `json:"wo_code"`
	Source          string     `json:"source"`
	MaintenanceType string     `json:"maintenance_type"`
	Status          string     `json:"status"`
	Priority        string     `json:"priority"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	P2HReportID     *int64     `json:"p2h_report_id,omitempty"`
	CandidateID     *int64     `json:"candidate_id,omitempty"`
	PMScheduleID    *int64     `json:"pm_schedule_id,omitempty"`
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
	ElapsedMinutes  *float64   `json:"elapsed_minutes,omitempty"`
	ChecklistTotal  int        `json:"checklist_total"`
	ChecklistDone   int        `json:"checklist_done"`
	ChecklistNotOK  int        `json:"checklist_not_ok"`
	FindingsCount   int        `json:"findings_count"`
	UsageCount      int        `json:"usage_count"`
	UsageQty        float64    `json:"usage_qty"`
	MaterialStatus  string     `json:"material_status"`
}

type ComponentHistoryFilter struct {
	Page  int
	Limit int
	From  *time.Time
	To    *time.Time
}

type ComponentHistory struct {
	UnitID              int64      `json:"unit_id"`
	UnitCode            string     `json:"unit_code"`
	WOID                int64      `json:"wo_id"`
	WOCode              string     `json:"wo_code"`
	WOTitle             string     `json:"wo_title"`
	MaintenanceType     string     `json:"maintenance_type"`
	WOStatus            string     `json:"wo_status"`
	ClosedAt            *time.Time `json:"closed_at,omitempty"`
	ComponentID         *int64     `json:"component_id,omitempty"`
	ComponentCode       string     `json:"component_code"`
	ComponentName       string     `json:"component_name"`
	IsFallback          bool       `json:"is_fallback"`
	FallbackSparepartID *int64     `json:"fallback_sparepart_id,omitempty"`
	SparepartCodes      string     `json:"sparepart_codes"`
	SparepartNames      string     `json:"sparepart_names"`
	SparepartCount      int        `json:"sparepart_count"`
	Qty                 float64    `json:"qty"`
	UsedAt              time.Time  `json:"used_at"`
	HM                  *float64   `json:"hm,omitempty"`
	KM                  *float64   `json:"km,omitempty"`
	ReplacementNo       int64      `json:"replacement_no"`
	PreviousUsedAt      *time.Time `json:"previous_used_at,omitempty"`
	DaysSincePrevious   *float64   `json:"days_since_previous,omitempty"`
	HMSincePrevious     *float64   `json:"hm_since_previous,omitempty"`
	KMSincePrevious     *float64   `json:"km_since_previous,omitempty"`
}

type ComponentInterval struct {
	UnitID              int64     `json:"unit_id"`
	UnitCode            string    `json:"unit_code"`
	ComponentID         *int64    `json:"component_id,omitempty"`
	FallbackSparepartID *int64    `json:"fallback_sparepart_id,omitempty"`
	ComponentCode       string    `json:"component_code"`
	ComponentName       string    `json:"component_name"`
	IsFallback          bool      `json:"is_fallback"`
	ReplacementCount    int       `json:"replacement_count"`
	FirstReplacedAt     time.Time `json:"first_replaced_at"`
	LastReplacedAt      time.Time `json:"last_replaced_at"`
	AvgIntervalDays     *float64  `json:"avg_interval_days,omitempty"`
	AvgIntervalHM       *float64  `json:"avg_interval_hm,omitempty"`
	AvgIntervalKM       *float64  `json:"avg_interval_km,omitempty"`
	TotalQty            float64   `json:"total_qty"`
	LastHM              *float64  `json:"last_hm,omitempty"`
	LastKM              *float64  `json:"last_km,omitempty"`
}

type DowntimeFilter struct {
	From  *time.Time
	To    *time.Time
	Limit int
}

type DowntimeEvent struct {
	WOID            int64      `json:"wo_id"`
	UnitID          int64      `json:"unit_id"`
	UnitCode        string     `json:"unit_code"`
	WOCode          string     `json:"wo_code"`
	Title           string     `json:"title"`
	MaintenanceType string     `json:"maintenance_type"`
	Priority        string     `json:"priority"`
	AssigneeID      *int64     `json:"assignee_id,omitempty"`
	AssigneeName    string     `json:"assignee_name"`
	SupervisorID    *int64     `json:"supervisor_id,omitempty"`
	SupervisorName  string     `json:"supervisor_name"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	DowntimeStart   *time.Time `json:"downtime_start,omitempty"`
	DowntimeEnd     time.Time  `json:"downtime_end"`
	DowntimeClass   string     `json:"downtime_class"`
	DowntimeMinutes *float64   `json:"downtime_minutes,omitempty"`
	WaitingMinutes  float64    `json:"waiting_minutes"`
}

type DowntimeSummary struct {
	TotalMinutes     float64 `json:"total_minutes"`
	PlannedMinutes   float64 `json:"planned_minutes"`
	UnplannedMinutes float64 `json:"unplanned_minutes"`
	WaitingMinutes   float64 `json:"waiting_minutes"`
	MTTRMinutes      float64 `json:"mttr_minutes"`
	Events           int     `json:"events"`
	PlannedEvents    int     `json:"planned_events"`
	UnplannedEvents  int     `json:"unplanned_events"`
}

type UnitDowntime struct {
	Summary DowntimeSummary `json:"summary"`
	Events  []DowntimeEvent `json:"events"`
}

type Component struct {
	ID       int64  `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Category string `json:"category"`
	IsActive bool   `json:"is_active"`
}

func (s *Service) ListMaintenanceHistory(ctx context.Context, unitID int64, filter HistoryFilter) ([]MaintenanceHistory, int, error) {
	if err := s.validateHistoryFilter(unitID, &filter); err != nil {
		return nil, 0, err
	}
	if err := s.ensureUnit(ctx, unitID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListMaintenanceHistory(ctx, unitID, filter)
}

func (s *Service) ListComponentHistory(ctx context.Context, unitID int64, filter ComponentHistoryFilter) ([]ComponentHistory, int, error) {
	if err := validateUnitID(unitID); err != nil {
		return nil, 0, err
	}
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	if err := validatePeriod(filter.From, filter.To); err != nil {
		return nil, 0, err
	}
	if err := s.ensureUnit(ctx, unitID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListComponentHistory(ctx, unitID, filter)
}

func (s *Service) ListComponentIntervals(ctx context.Context, unitID int64) ([]ComponentInterval, error) {
	if err := validateUnitID(unitID); err != nil {
		return nil, err
	}
	if err := s.ensureUnit(ctx, unitID); err != nil {
		return nil, err
	}
	return s.repo.ListComponentIntervals(ctx, unitID)
}

func (s *Service) UnitDowntime(ctx context.Context, unitID int64, filter DowntimeFilter) (UnitDowntime, error) {
	if err := validateUnitID(unitID); err != nil {
		return UnitDowntime{}, err
	}
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > MaxPageLimit {
		filter.Limit = MaxPageLimit
	}
	if err := validatePeriod(filter.From, filter.To); err != nil {
		return UnitDowntime{}, err
	}
	if err := s.ensureUnit(ctx, unitID); err != nil {
		return UnitDowntime{}, err
	}
	events, err := s.repo.ListUnitDowntime(ctx, unitID, filter)
	if err != nil {
		return UnitDowntime{}, err
	}
	result := buildDowntimeResult(events, filter)
	if len(events) > filter.Limit {
		result.Events = events[:filter.Limit]
	}
	return result, nil
}

func (s *Service) ListComponents(ctx context.Context, activeOnly bool) ([]Component, error) {
	return s.repo.ListComponents(ctx, activeOnly)
}

func (s *Service) validateHistoryFilter(unitID int64, filter *HistoryFilter) error {
	if err := validateUnitID(unitID); err != nil {
		return err
	}
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.MaintenanceType = strings.ToLower(strings.TrimSpace(filter.MaintenanceType))
	if filter.Status != "" && !allowedWorkOrderStatus(filter.Status) {
		return &ValidationError{Fields: map[string]string{"status": "Status work order tidak valid"}}
	}
	if filter.MaintenanceType != "" && !allowedMaintenanceType(filter.MaintenanceType) {
		return &ValidationError{Fields: map[string]string{"maintenance_type": "Tipe maintenance tidak valid"}}
	}
	return validatePeriod(filter.From, filter.To)
}

func (s *Service) ensureUnit(ctx context.Context, unitID int64) error {
	_, err := s.repo.GetUnit(ctx, unitID)
	return err
}

func validateUnitID(unitID int64) error {
	if unitID <= 0 {
		return &ValidationError{Fields: map[string]string{"unit_id": "ID unit tidak valid"}}
	}
	return nil
}

func validatePeriod(from, to *time.Time) error {
	if from != nil && to != nil && !from.Before(*to) {
		return &ValidationError{Fields: map[string]string{"period": "Periode tidak valid"}}
	}
	return nil
}

func allowedWorkOrderStatus(value string) bool {
	switch value {
	case "open", "in_progress", "waiting_part", "done", "cancelled":
		return true
	default:
		return false
	}
}

func allowedMaintenanceType(value string) bool {
	switch value {
	case "preventive", "corrective", "breakdown", "inspection":
		return true
	default:
		return false
	}
}

type downtimeSpan struct {
	start time.Time
	end   time.Time
}

// buildDowntimeResult derives union-based downtime totals. Overlapping work
// orders for the same unit are merged so the same clock time is never counted
// twice.
func buildDowntimeResult(events []DowntimeEvent, filter DowntimeFilter) UnitDowntime {
	summary := DowntimeSummary{Events: len(events)}
	allSpans := make([]downtimeSpan, 0, len(events))
	plannedSpans := make([]downtimeSpan, 0, len(events))
	unplannedSpans := make([]downtimeSpan, 0, len(events))
	var unplannedDuration float64
	for _, event := range events {
		summary.WaitingMinutes += event.WaitingMinutes
		if event.DowntimeClass == "planned" {
			summary.PlannedEvents++
		} else {
			summary.UnplannedEvents++
		}
		span, ok := clippedDowntimeSpan(event, filter.From, filter.To)
		if !ok {
			continue
		}
		allSpans = append(allSpans, span)
		minutes := span.end.Sub(span.start).Minutes()
		if event.DowntimeClass == "planned" {
			plannedSpans = append(plannedSpans, span)
		} else {
			unplannedSpans = append(unplannedSpans, span)
			unplannedDuration += minutes
		}
	}
	summary.TotalMinutes = roundMinutes(unionMinutes(allSpans))
	summary.PlannedMinutes = roundMinutes(unionMinutes(plannedSpans))
	summary.UnplannedMinutes = roundMinutes(unionMinutes(unplannedSpans))
	summary.WaitingMinutes = roundMinutes(summary.WaitingMinutes)
	if summary.UnplannedEvents > 0 {
		summary.MTTRMinutes = roundMinutes(unplannedDuration / float64(summary.UnplannedEvents))
	}
	if events == nil {
		events = []DowntimeEvent{}
	}
	return UnitDowntime{Summary: summary, Events: events}
}

func clippedDowntimeSpan(event DowntimeEvent, from, to *time.Time) (downtimeSpan, bool) {
	if event.DowntimeStart == nil {
		return downtimeSpan{}, false
	}
	start := *event.DowntimeStart
	end := event.DowntimeEnd
	if from != nil && start.Before(*from) {
		start = *from
	}
	if to != nil && !end.Before(*to) {
		end = *to
	}
	if !start.Before(end) {
		return downtimeSpan{}, false
	}
	return downtimeSpan{start: start, end: end}, true
}

func unionMinutes(spans []downtimeSpan) float64 {
	if len(spans) == 0 {
		return 0
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start.Equal(spans[j].start) {
			return spans[i].end.Before(spans[j].end)
		}
		return spans[i].start.Before(spans[j].start)
	})
	var total float64
	current := spans[0]
	for _, span := range spans[1:] {
		if span.start.After(current.end) {
			total += current.end.Sub(current.start).Minutes()
			current = span
			continue
		}
		if span.end.After(current.end) {
			current.end = span.end
		}
	}
	total += current.end.Sub(current.start).Minutes()
	return total
}

func roundMinutes(value float64) float64 {
	return math.Round(value*10) / 10
}
