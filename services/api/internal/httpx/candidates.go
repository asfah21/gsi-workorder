package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/candidates"
)

type CandidateService interface {
	List(ctx context.Context, filter candidates.Filter) ([]candidates.Candidate, int, error)
	Get(ctx context.Context, id int64) (candidates.Detail, error)
	Create(ctx context.Context, input candidates.CreateInput) (candidates.Candidate, error)
	Validate(ctx context.Context, id, actorID int64, input candidates.ValidateInput) (candidates.Candidate, error)
	Approve(ctx context.Context, id, actorID int64, input candidates.ApproveInput) (candidates.ApproveResult, error)
	Convert(ctx context.Context, id, actorID int64, input candidates.ApproveInput) (candidates.ApproveResult, error)
	Skip(ctx context.Context, id, actorID int64, input candidates.SkipInput) (candidates.Candidate, error)
	Bulk(ctx context.Context, actorID int64, input candidates.BulkInput) (candidates.BulkResult, error)
}

type CandidateHandler struct {
	service CandidateService
	logger  *slog.Logger
}

func NewCandidateHandler(service CandidateService, logger *slog.Logger) *CandidateHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &CandidateHandler{service: service, logger: logger}
}

type createRequest struct {
	UnitID       int64  `json:"unit_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	WorkCategory string `json:"work_category"`
	IsBreakdown  bool   `json:"is_breakdown"`
	Priority     string `json:"suggested_priority"`
}

type validateRequest struct {
	Note string `json:"note"`
}

type approveRequest struct {
	Priority   string `json:"priority"`
	AssigneeID *int64 `json:"assignee_id"`
	Note       string `json:"note"`
}

type skipRequest struct {
	Reason string `json:"reason"`
}

type bulkRequest struct {
	IDs        []int64 `json:"ids"`
	Action     string  `json:"action"`
	Reason     string  `json:"reason"`
	Priority   string  `json:"priority"`
	AssigneeID *int64  `json:"assignee_id"`
	Note       string  `json:"note"`
}

func (h *CandidateHandler) List(c *gin.Context) {
	page, limit := parsePagination(c)
	status := strings.TrimSpace(c.Query("status"))
	backlog := status == "backlog" || parseBool(c.Query("backlog"))
	if backlog {
		status = "all"
	} else if status == "" {
		status = "pending"
	}
	unitType := strings.TrimSpace(c.Query("unit_type"))
	items, total, err := h.service.List(c.Request.Context(), candidates.Filter{
		Status:   status,
		UnitType: unitType,
		Query:    strings.TrimSpace(c.Query("q")),
		Backlog:  backlog,
		Page:     page,
		Limit:    limit,
		Sort:     c.Query("sort"),
		Order:    c.Query("order"),
	})
	if err != nil {
		h.writeError(c, err, "list candidates")
		return
	}
	writeList(c, newCandidateResponses(items), page, limit, total)
}

func (h *CandidateHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	detail, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "get candidate")
		return
	}
	data := gin.H{
		"candidate":  newCandidateResponse(detail.Candidate),
		"work_order": detail.WorkOrder,
	}
	if detail.Report != nil {
		data["report"] = newReportResponse(*detail.Report)
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *CandidateHandler) Create(c *gin.Context) {
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request createRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body Work Request tidak valid", nil)
		return
	}
	item, err := h.service.Create(c.Request.Context(), candidates.CreateInput{
		UnitID:       request.UnitID,
		Title:        request.Title,
		Description:  request.Description,
		WorkCategory: request.WorkCategory,
		IsBreakdown:  request.IsBreakdown,
		Priority:     request.Priority,
		ActorID:      actor.ID,
	})
	if err != nil {
		h.writeError(c, err, "create work request")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": newCandidateResponse(item)})
}

func (h *CandidateHandler) Validate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request validateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body validasi tidak valid", nil)
		return
	}
	item, err := h.service.Validate(c.Request.Context(), id, actor.ID, candidates.ValidateInput{Note: request.Note})
	if err != nil {
		h.writeError(c, err, "validate work request")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": newCandidateResponse(item)})
}

func (h *CandidateHandler) Convert(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request approveRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body konversi tidak valid", nil)
		return
	}
	result, err := h.service.Convert(c.Request.Context(), id, actor.ID, candidates.ApproveInput{
		Priority:   request.Priority,
		AssigneeID: request.AssigneeID,
		Note:       request.Note,
	})
	if err != nil {
		h.writeError(c, err, "convert work request")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": approveResponse{
		Candidate: newCandidateResponse(result.Candidate),
		WorkOrder: newWorkOrderResponse(result.WorkOrder),
	}})
}

func (h *CandidateHandler) Approve(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request approveRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body approval tidak valid", nil)
		return
	}
	result, err := h.service.Approve(c.Request.Context(), id, actor.ID, candidates.ApproveInput{
		Priority:   request.Priority,
		AssigneeID: request.AssigneeID,
		Note:       request.Note,
	})
	if err != nil {
		h.writeError(c, err, "approve candidate")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": approveResponse{
		Candidate: newCandidateResponse(result.Candidate),
		WorkOrder: newWorkOrderResponse(result.WorkOrder),
	}})
}

func (h *CandidateHandler) Skip(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request skipRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body penolakan tidak valid", nil)
		return
	}
	item, err := h.service.Skip(c.Request.Context(), id, actor.ID, candidates.SkipInput{Reason: request.Reason})
	if err != nil {
		// A missing or too-short skip reason is a request validation error
		// (400), not an unprocessable entity. Keep other candidate endpoints
		// on the shared 422 validation mapping.
		h.writeErrorWithValidationStatus(c, err, "skip candidate", http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": newCandidateResponse(item)})
}

func (h *CandidateHandler) Bulk(c *gin.Context) {
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request bulkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body bulk tidak valid", nil)
		return
	}
	result, err := h.service.Bulk(c.Request.Context(), actor.ID, candidates.BulkInput{
		IDs:        request.IDs,
		Action:     request.Action,
		Reason:     request.Reason,
		Priority:   request.Priority,
		AssigneeID: request.AssigneeID,
		Note:       request.Note,
	})
	if err != nil {
		h.writeError(c, err, "bulk candidates")
		return
	}
	response := bulkResponse{
		Approved:  make([]approveResponse, 0, len(result.Approved)),
		Validated: newCandidateResponses(result.Validated),
		Converted: make([]approveResponse, 0, len(result.Converted)),
		Skipped:   newCandidateResponses(result.Skipped),
	}
	for _, item := range result.Approved {
		response.Approved = append(response.Approved, approveResponse{
			Candidate: newCandidateResponse(item.Candidate),
			WorkOrder: newWorkOrderResponse(item.WorkOrder),
		})
	}
	for _, item := range result.Converted {
		response.Converted = append(response.Converted, approveResponse{
			Candidate: newCandidateResponse(item.Candidate),
			WorkOrder: newWorkOrderResponse(item.WorkOrder),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": response})
}

func (h *CandidateHandler) writeError(c *gin.Context, err error, operation string) {
	h.writeErrorWithValidationStatus(c, err, operation, http.StatusUnprocessableEntity)
}

func (h *CandidateHandler) writeErrorWithValidationStatus(c *gin.Context, err error, operation string, validationStatus int) {
	var validation *candidates.ValidationError
	switch {
	case errors.As(err, &validation):
		writeUnitError(c, validationStatus, "validation_error", "Data usulan tidak valid", validation.Fields)
	case errors.Is(err, candidates.ErrNotFound):
		writeUnitError(c, http.StatusNotFound, "candidate_not_found", "Usulan tidak ditemukan", nil)
	case errors.Is(err, candidates.ErrConflict):
		writeUnitError(c, http.StatusConflict, "candidate_conflict", "Usulan sudah diputuskan atau tidak lagi pending", nil)
	case errors.Is(err, candidates.ErrValidation):
		writeUnitError(c, validationStatus, "validation_error", "Data usulan tidak valid", nil)
	default:
		h.logger.Error("candidate operation failed", "operation", operation, "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
	}
}
