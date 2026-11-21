package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/pm"
)

type PMService interface {
	ListPlans(ctx context.Context, filter pm.PlanFilter) ([]pm.Plan, int, error)
	GetPlan(ctx context.Context, id int64) (pm.Plan, error)
	CreatePlan(ctx context.Context, actor pm.Actor, input pm.PlanInput) (pm.Plan, error)
	UpdatePlan(ctx context.Context, id int64, actor pm.Actor, patch pm.PlanPatch) (pm.Plan, error)
	SetPlanStatus(ctx context.Context, id int64, actor pm.Actor, status string) (pm.Plan, error)
	GetPlanChecklist(ctx context.Context, planID int64) ([]pm.ChecklistItem, error)
	AddChecklistItem(ctx context.Context, planID int64, actor pm.Actor, input pm.ChecklistInput) (pm.ChecklistItem, error)
	ListSchedules(ctx context.Context, filter pm.ScheduleFilter) ([]pm.Schedule, int, error)
	GetSchedule(ctx context.Context, id int64) (pm.Schedule, error)
	GenerateWO(ctx context.Context, id int64, actor pm.Actor) (pm.GenerateResult, error)
	OverrideDue(ctx context.Context, id int64, actor pm.Actor, input pm.OverrideInput) (pm.Schedule, error)
	Skip(ctx context.Context, id int64, actor pm.Actor, reason string) (pm.Schedule, error)
	Cancel(ctx context.Context, id int64, actor pm.Actor, reason string) (pm.Schedule, error)
	Compliance(ctx context.Context, filter pm.ComplianceFilter) (pm.Compliance, error)
}

type PMHandler struct {
	service PMService
	logger  *slog.Logger
}

func NewPMHandler(service PMService, logger *slog.Logger) *PMHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &PMHandler{service: service, logger: logger}
}

type pmChecklistRequest struct {
	Sequence    int    `json:"sequence"`
	ComponentID *int64 `json:"component_id"`
	Task        string `json:"task"`
	Standard    string `json:"standard"`
}

type pmPlanRequest struct {
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	UnitID              *int64               `json:"unit_id"`
	UnitModelID         *int64               `json:"unit_model_id"`
	Status              string               `json:"status"`
	EffectiveFrom       string               `json:"effective_from"`
	EffectiveTo         string               `json:"effective_to"`
	IntervalHM          *float64             `json:"interval_hm"`
	IntervalKM          *float64             `json:"interval_km"`
	IntervalDays        *int                 `json:"interval_days"`
	IntervalRule        string               `json:"interval_rule"`
	UpcomingDays        *int                 `json:"upcoming_days"`
	DueSoonDays         *int                 `json:"due_soon_days"`
	UpcomingHM          *float64             `json:"upcoming_hm"`
	UpcomingKM          *float64             `json:"upcoming_km"`
	DueSoonHM           *float64             `json:"due_soon_hm"`
	DueSoonKM           *float64             `json:"due_soon_km"`
	AutoGenerateWO      bool                 `json:"auto_generate_wo"`
	DefaultPriority     string               `json:"default_priority"`
	DefaultAssigneeID   *int64               `json:"default_assignee_id"`
	DefaultSupervisorID *int64               `json:"default_supervisor_id"`
	Checklist           []pmChecklistRequest `json:"checklist"`
}

type pmPlanPatchRequest struct {
	Name                   *string               `json:"name"`
	Description            *string               `json:"description"`
	Status                 *string               `json:"status"`
	EffectiveFrom          *string               `json:"effective_from"`
	EffectiveTo            *string               `json:"effective_to"`
	ClearEffectiveTo       bool                  `json:"clear_effective_to"`
	IntervalHM             *float64              `json:"interval_hm"`
	ClearIntervalHM        bool                  `json:"clear_interval_hm"`
	IntervalKM             *float64              `json:"interval_km"`
	ClearIntervalKM        bool                  `json:"clear_interval_km"`
	IntervalDays           *int                  `json:"interval_days"`
	ClearIntervalDays      bool                  `json:"clear_interval_days"`
	IntervalRule           *string               `json:"interval_rule"`
	UpcomingDays           *int                  `json:"upcoming_days"`
	ClearUpcomingDays      bool                  `json:"clear_upcoming_days"`
	DueSoonDays            *int                  `json:"due_soon_days"`
	ClearDueSoonDays       bool                  `json:"clear_due_soon_days"`
	UpcomingHM             *float64              `json:"upcoming_hm"`
	ClearUpcomingHM        bool                  `json:"clear_upcoming_hm"`
	UpcomingKM             *float64              `json:"upcoming_km"`
	ClearUpcomingKM        bool                  `json:"clear_upcoming_km"`
	DueSoonHM              *float64              `json:"due_soon_hm"`
	ClearDueSoonHM         bool                  `json:"clear_due_soon_hm"`
	DueSoonKM              *float64              `json:"due_soon_km"`
	ClearDueSoonKM         bool                  `json:"clear_due_soon_km"`
	AutoGenerateWO         *bool                 `json:"auto_generate_wo"`
	DefaultPriority        *string               `json:"default_priority"`
	DefaultAssigneeID      *int64                `json:"default_assignee_id"`
	ClearDefaultAssignee   bool                  `json:"clear_default_assignee"`
	DefaultSupervisorID    *int64                `json:"default_supervisor_id"`
	ClearDefaultSupervisor bool                  `json:"clear_default_supervisor"`
	Checklist              *[]pmChecklistRequest `json:"checklist"`
}

type pmStatusRequest struct {
	Status string `json:"status"`
}

type pmReasonRequest struct {
	Reason string `json:"reason"`
}

type pmOverrideRequest struct {
	DueHM   *float64 `json:"due_hm"`
	DueKM   *float64 `json:"due_km"`
	DueDate string   `json:"due_date"`
	Reason  string   `json:"reason"`
}

func (h *PMHandler) ListPlans(c *gin.Context) {
	page, limit := parsePagination(c)
	unitID, err := parseOptionalQueryID(c.Query("unit_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_id tidak valid", nil)
		return
	}
	modelID, err := parseOptionalQueryID(c.Query("unit_model_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_model_id tidak valid", nil)
		return
	}
	items, total, err := h.service.ListPlans(c.Request.Context(), pm.PlanFilter{
		Status:      strings.TrimSpace(c.Query("status")),
		Query:       strings.TrimSpace(c.Query("q")),
		UnitID:      unitID,
		UnitModelID: modelID,
		Page:        page,
		Limit:       limit,
		Sort:        c.Query("sort"),
		Order:       c.Query("order"),
	})
	if err != nil {
		h.writeError(c, err, "list PM plans")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *PMHandler) GetPlan(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.GetPlan(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "get PM plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *PMHandler) CreatePlan(c *gin.Context) {
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request pmPlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body PM plan tidak valid", nil)
		return
	}
	effectiveFrom, err := parseDate(request.EffectiveFrom)
	if err != nil {
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data PM plan tidak valid", map[string]string{"effective_from": "Format tanggal harus YYYY-MM-DD."})
		return
	}
	effectiveTo, err := parseOptionalDate(request.EffectiveTo)
	if err != nil {
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data PM plan tidak valid", map[string]string{"effective_to": "Format tanggal harus YYYY-MM-DD."})
		return
	}
	item, err := h.service.CreatePlan(c.Request.Context(), pm.Actor{ID: actor.ID, Role: actor.Role}, pm.PlanInput{
		Name:                request.Name,
		Description:         request.Description,
		UnitID:              request.UnitID,
		UnitModelID:         request.UnitModelID,
		Status:              request.Status,
		EffectiveFrom:       effectiveFrom,
		EffectiveTo:         effectiveTo,
		IntervalHM:          request.IntervalHM,
		IntervalKM:          request.IntervalKM,
		IntervalDays:        request.IntervalDays,
		IntervalRule:        request.IntervalRule,
		UpcomingDays:        request.UpcomingDays,
		DueSoonDays:         request.DueSoonDays,
		UpcomingHM:          request.UpcomingHM,
		UpcomingKM:          request.UpcomingKM,
		DueSoonHM:           request.DueSoonHM,
		DueSoonKM:           request.DueSoonKM,
		AutoGenerateWO:      request.AutoGenerateWO,
		DefaultPriority:     request.DefaultPriority,
		DefaultAssigneeID:   request.DefaultAssigneeID,
		DefaultSupervisorID: request.DefaultSupervisorID,
		Checklist:           convertChecklist(request.Checklist),
	})
	if err != nil {
		h.writeError(c, err, "create PM plan")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *PMHandler) UpdatePlan(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request pmPlanPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body PM plan tidak valid", nil)
		return
	}
	patch := pm.PlanPatch{
		Name:                   request.Name,
		Description:            request.Description,
		Status:                 request.Status,
		ClearEffectiveTo:       request.ClearEffectiveTo,
		IntervalHM:             request.IntervalHM,
		ClearIntervalHM:        request.ClearIntervalHM,
		IntervalKM:             request.IntervalKM,
		ClearIntervalKM:        request.ClearIntervalKM,
		IntervalDays:           request.IntervalDays,
		ClearIntervalDays:      request.ClearIntervalDays,
		IntervalRule:           request.IntervalRule,
		UpcomingDays:           request.UpcomingDays,
		ClearUpcomingDays:      request.ClearUpcomingDays,
		DueSoonDays:            request.DueSoonDays,
		ClearDueSoonDays:       request.ClearDueSoonDays,
		UpcomingHM:             request.UpcomingHM,
		ClearUpcomingHM:        request.ClearUpcomingHM,
		UpcomingKM:             request.UpcomingKM,
		ClearUpcomingKM:        request.ClearUpcomingKM,
		DueSoonHM:              request.DueSoonHM,
		ClearDueSoonHM:         request.ClearDueSoonHM,
		DueSoonKM:              request.DueSoonKM,
		ClearDueSoonKM:         request.ClearDueSoonKM,
		AutoGenerateWO:         request.AutoGenerateWO,
		DefaultPriority:        request.DefaultPriority,
		DefaultAssigneeID:      request.DefaultAssigneeID,
		ClearDefaultAssignee:   request.ClearDefaultAssignee,
		DefaultSupervisorID:    request.DefaultSupervisorID,
		ClearDefaultSupervisor: request.ClearDefaultSupervisor,
	}
	if request.EffectiveFrom != nil {
		value, err := parseDate(*request.EffectiveFrom)
		if err != nil {
			writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data PM plan tidak valid", map[string]string{"effective_from": "Format tanggal harus YYYY-MM-DD."})
			return
		}
		patch.EffectiveFrom = &value
	}
	if request.EffectiveTo != nil {
		value, err := parseOptionalDate(*request.EffectiveTo)
		if err != nil {
			writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data PM plan tidak valid", map[string]string{"effective_to": "Format tanggal harus YYYY-MM-DD."})
			return
		}
		patch.EffectiveTo = value
	}
	if request.Checklist != nil {
		items := convertChecklist(*request.Checklist)
		patch.Checklist = &items
	}
	item, err := h.service.UpdatePlan(c.Request.Context(), id, pm.Actor{ID: actor.ID, Role: actor.Role}, patch)
	if err != nil {
		h.writeError(c, err, "update PM plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *PMHandler) SetPlanStatus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request pmStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body status plan tidak valid", nil)
		return
	}
	item, err := h.service.SetPlanStatus(c.Request.Context(), id, pm.Actor{ID: actor.ID, Role: actor.Role}, request.Status)
	if err != nil {
		h.writeError(c, err, "set PM plan status")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *PMHandler) ListChecklistItems(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.GetPlanChecklist(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "list PM checklist")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *PMHandler) CreateChecklistItem(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request pmChecklistRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body checklist tidak valid", nil)
		return
	}
	item, err := h.service.AddChecklistItem(c.Request.Context(), id, pm.Actor{ID: actor.ID, Role: actor.Role}, pm.ChecklistInput{
		Sequence:    request.Sequence,
		ComponentID: request.ComponentID,
		Task:        request.Task,
		Standard:    request.Standard,
	})
	if err != nil {
		h.writeError(c, err, "create PM checklist")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *PMHandler) ListSchedules(c *gin.Context) {
	page, limit := parsePagination(c)
	unitID, err := parseOptionalQueryID(c.Query("unit_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_id tidak valid", nil)
		return
	}
	planID, err := parseOptionalQueryID(c.Query("pm_plan_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter pm_plan_id tidak valid", nil)
		return
	}
	items, total, err := h.service.ListSchedules(c.Request.Context(), pm.ScheduleFilter{
		Status:     strings.TrimSpace(c.Query("status")),
		Bucket:     strings.TrimSpace(c.Query("bucket")),
		Query:      strings.TrimSpace(c.Query("q")),
		UnitID:     unitID,
		PlanID:     planID,
		ActiveOnly: parseBool(c.Query("active_only")),
		Page:       page,
		Limit:      limit,
	})
	if err != nil {
		h.writeError(c, err, "list PM schedules")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *PMHandler) GetSchedule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.GetSchedule(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "get PM schedule")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *PMHandler) GenerateWO(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	result, err := h.service.GenerateWO(c.Request.Context(), id, pm.Actor{ID: actor.ID, Role: actor.Role})
	if err != nil {
		h.writeError(c, err, "generate PM work order")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *PMHandler) OverrideDue(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request pmOverrideRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body override tidak valid", nil)
		return
	}
	dueDate, err := parseOptionalDate(request.DueDate)
	if err != nil {
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data override tidak valid", map[string]string{"due_date": "Format tanggal harus YYYY-MM-DD."})
		return
	}
	item, err := h.service.OverrideDue(c.Request.Context(), id, pm.Actor{ID: actor.ID, Role: actor.Role}, pm.OverrideInput{
		DueHM:   request.DueHM,
		DueKM:   request.DueKM,
		DueDate: dueDate,
		Reason:  request.Reason,
	})
	if err != nil {
		h.writeError(c, err, "override PM due")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *PMHandler) Skip(c *gin.Context) {
	h.finish(c, "skip")
}

func (h *PMHandler) Cancel(c *gin.Context) {
	h.finish(c, "cancel")
}

func (h *PMHandler) finish(c *gin.Context, action string) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request pmReasonRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body alasan tidak valid", nil)
		return
	}
	var item pm.Schedule
	var err error
	if action == "skip" {
		item, err = h.service.Skip(c.Request.Context(), id, pm.Actor{ID: actor.ID, Role: actor.Role}, request.Reason)
	} else {
		item, err = h.service.Cancel(c.Request.Context(), id, pm.Actor{ID: actor.ID, Role: actor.Role}, request.Reason)
	}
	if err != nil {
		h.writeError(c, err, action+" PM schedule")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *PMHandler) Due(c *gin.Context) {
	page, limit := parsePagination(c)
	unitID, err := parseOptionalQueryID(c.Query("unit_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_id tidak valid", nil)
		return
	}
	planID, err := parseOptionalQueryID(c.Query("pm_plan_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter pm_plan_id tidak valid", nil)
		return
	}
	items, total, err := h.service.ListSchedules(c.Request.Context(), pm.ScheduleFilter{
		Bucket:     strings.TrimSpace(c.Query("bucket")),
		Query:      strings.TrimSpace(c.Query("q")),
		UnitID:     unitID,
		PlanID:     planID,
		ActiveOnly: parseBool(c.Query("active_only")),
		Page:       page,
		Limit:      limit,
	})
	if err != nil {
		h.writeError(c, err, "list PM due")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *PMHandler) Compliance(c *gin.Context) {
	planID, err := parseOptionalQueryID(c.Query("pm_plan_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter pm_plan_id tidak valid", nil)
		return
	}
	unitID, err := parseOptionalQueryID(c.Query("unit_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_id tidak valid", nil)
		return
	}
	from, err := parseOptionalDate(c.Query("from"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter from tidak valid", nil)
		return
	}
	to, err := parseOptionalDate(c.Query("to"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter to tidak valid", nil)
		return
	}
	result, err := h.service.Compliance(c.Request.Context(), pm.ComplianceFilter{
		PlanID: planID,
		UnitID: unitID,
		From:   from,
		To:     to,
	})
	if err != nil {
		h.writeError(c, err, "PM compliance")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *PMHandler) writeError(c *gin.Context, err error, operation string) {
	var validation *pm.ValidationError
	switch {
	case errors.As(err, &validation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data PM tidak valid", validation.Fields)
	case errors.Is(err, pm.ErrNotFound):
		writeUnitError(c, http.StatusNotFound, "pm_not_found", "Data PM tidak ditemukan", nil)
	case errors.Is(err, pm.ErrConflict):
		writeUnitError(c, http.StatusConflict, "pm_conflict", "Data PM tidak dapat diproses pada status ini", nil)
	case errors.Is(err, pm.ErrValidation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data PM tidak valid", nil)
	default:
		h.logger.Error("PM operation failed", "operation", operation, "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
	}
}

func convertChecklist(items []pmChecklistRequest) []pm.ChecklistInput {
	result := make([]pm.ChecklistInput, 0, len(items))
	for _, item := range items {
		result = append(result, pm.ChecklistInput{
			Sequence:    item.Sequence,
			ComponentID: item.ComponentID,
			Task:        item.Task,
			Standard:    item.Standard,
		})
	}
	return result
}

func parseDate(raw string) (time.Time, error) {
	value, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(raw), time.UTC)
	if err != nil {
		return time.Time{}, err
	}
	return value, nil
}

func parseOptionalDate(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := parseDate(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
