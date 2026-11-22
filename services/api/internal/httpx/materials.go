package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/inventory"
	"gsi-workorder/internal/material"
)

type MaterialHandler struct {
	service material.Service
	logger  *slog.Logger
}

func NewMaterialHandler(service material.Service, logger *slog.Logger) *MaterialHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &MaterialHandler{service: service, logger: logger}
}

type materialItemRequest struct {
	SparepartID          int64   `json:"sparepart_id"`
	QtyRequired          float64 `json:"qty_required"`
	IsOptional           bool    `json:"is_optional"`
	PreferredWarehouseID *int64  `json:"preferred_warehouse_id"`
	Notes                string  `json:"notes"`
}

type materialCreateRequest struct {
	WOID   int64                 `json:"wo_id"`
	Source string                `json:"source"`
	Notes  string                `json:"notes"`
	Items  []materialItemRequest `json:"items"`
}

type materialPatchRequest struct {
	Status *string                `json:"status"`
	Notes  *string                `json:"notes"`
	Items  *[]materialItemRequest `json:"items"`
}

type materialReserveRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Notes          string `json:"notes"`
}

type materialIssueItemRequest struct {
	MaterialItemID int64   `json:"material_requirement_item_id"`
	Qty            float64 `json:"qty"`
	ComponentID    *int64  `json:"component_id"`
	Notes          string  `json:"notes"`
}

type materialIssueRequest struct {
	IdempotencyKey string                     `json:"idempotency_key"`
	Notes          string                     `json:"notes"`
	Items          []materialIssueItemRequest `json:"items"`
}

type materialReturnItemRequest struct {
	UsageID        *int64  `json:"usage_id"`
	MaterialItemID *int64  `json:"material_requirement_item_id"`
	LocationID     *int64  `json:"warehouse_location_id"`
	Qty            float64 `json:"qty"`
	Notes          string  `json:"notes"`
}

type materialReturnRequest struct {
	IdempotencyKey string                      `json:"idempotency_key"`
	Notes          string                      `json:"notes"`
	Items          []materialReturnItemRequest `json:"items"`
}

type materialCancelRequest struct {
	Reason string `json:"reason"`
}

type materialReservationCreateRequest struct {
	MaterialItemID int64   `json:"material_requirement_item_id"`
	LocationID     int64   `json:"warehouse_location_id"`
	Qty            float64 `json:"qty"`
	Notes          string  `json:"notes"`
}

type materialReservationReleaseRequest struct {
	Reason string `json:"reason"`
}

func (h *MaterialHandler) ListRequirements(c *gin.Context) {
	page, limit := parsePagination(c)
	woID, err := parseOptionalQueryID(c.Query("wo_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter wo_id tidak valid", nil)
		return
	}
	items, total, err := h.service.List(c.Request.Context(), material.Filter{
		WOID: woID, Status: strings.TrimSpace(c.Query("status")),
		Query: c.Query("q"), Page: page, Limit: limit,
	})
	if err != nil {
		h.writeError(c, err, "list material requirements")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *MaterialHandler) GetRequirement(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "get material requirement")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *MaterialHandler) CreateRequirement(c *gin.Context) {
	var request materialCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body material requirement tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.Create(c.Request.Context(), material.Actor{ID: actor.ID, Role: actor.Role},
		material.CreateInput{
			WOID: request.WOID, Source: request.Source, Notes: request.Notes,
			Items: materialItems(request.Items),
		})
	if err != nil {
		h.writeError(c, err, "create material requirement")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *MaterialHandler) UpdateRequirement(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request materialPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body material requirement tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var items *[]material.ItemInput
	if request.Items != nil {
		converted := materialItems(*request.Items)
		items = &converted
	}
	item, err := h.service.Update(c.Request.Context(), id,
		material.Actor{ID: actor.ID, Role: actor.Role}, material.PatchInput{
			Status: request.Status, Notes: request.Notes, Items: items,
		})
	if err != nil {
		h.writeError(c, err, "update material requirement")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *MaterialHandler) ListUpdates(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ListUpdates(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "list material updates")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *MaterialHandler) ListUsages(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ListUsages(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "list material usages")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *MaterialHandler) CheckStock(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.CheckStock(c.Request.Context(), id, material.Actor{ID: actor.ID, Role: actor.Role})
	if err != nil {
		h.writeError(c, err, "check material stock")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *MaterialHandler) Reserve(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request materialReserveRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body reserve tidak valid", nil)
			return
		}
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.Reserve(c.Request.Context(), id,
		material.Actor{ID: actor.ID, Role: actor.Role}, material.ReserveInput{
			IdempotencyKey: request.IdempotencyKey, Notes: request.Notes,
		})
	if err != nil {
		h.writeError(c, err, "reserve material")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *MaterialHandler) Issue(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request materialIssueRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body issue material tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	items := make([]material.IssueItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, material.IssueItemInput{
			MaterialItemID: item.MaterialItemID, Qty: item.Qty,
			ComponentID: item.ComponentID, Notes: item.Notes,
		})
	}
	result, err := h.service.Issue(c.Request.Context(), id,
		material.Actor{ID: actor.ID, Role: actor.Role}, material.IssueInput{
			IdempotencyKey: request.IdempotencyKey, Notes: request.Notes, Items: items,
		})
	if err != nil {
		h.writeError(c, err, "issue material")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *MaterialHandler) Return(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request materialReturnRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body return material tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	items := make([]material.ReturnItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, material.ReturnItemInput{
			UsageID: item.UsageID, MaterialItemID: item.MaterialItemID,
			LocationID: item.LocationID, Qty: item.Qty, Notes: item.Notes,
		})
	}
	result, err := h.service.Return(c.Request.Context(), id,
		material.Actor{ID: actor.ID, Role: actor.Role}, material.ReturnInput{
			IdempotencyKey: request.IdempotencyKey, Notes: request.Notes, Items: items,
		})
	if err != nil {
		h.writeError(c, err, "return material")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *MaterialHandler) Cancel(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request materialCancelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body pembatalan material tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.Cancel(c.Request.Context(), id,
		material.Actor{ID: actor.ID, Role: actor.Role}, request.Reason)
	if err != nil {
		h.writeError(c, err, "cancel material requirement")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *MaterialHandler) ListReservations(c *gin.Context) {
	page, limit := parsePagination(c)
	sparepartID, err := parseOptionalQueryID(c.Query("sparepart_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter sparepart_id tidak valid", nil)
		return
	}
	requirementID, err := parseOptionalQueryID(c.Query("material_requirement_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter material_requirement_id tidak valid", nil)
		return
	}
	woID, err := parseOptionalQueryID(c.Query("wo_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter wo_id tidak valid", nil)
		return
	}
	items, total, err := h.service.ListReservations(c.Request.Context(), material.ReservationFilter{
		Status: strings.TrimSpace(c.Query("status")), SparepartID: sparepartID,
		RequirementID: requirementID, WOID: woID, Page: page, Limit: limit,
	})
	if err != nil {
		h.writeError(c, err, "list inventory reservations")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *MaterialHandler) CreateReservation(c *gin.Context) {
	var request materialReservationCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body reservasi tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.CreateReservation(c.Request.Context(),
		material.Actor{ID: actor.ID, Role: actor.Role}, material.CreateReservationInput{
			MaterialItemID: request.MaterialItemID, LocationID: request.LocationID,
			Qty: request.Qty, Notes: request.Notes,
		})
	if err != nil {
		h.writeError(c, err, "create inventory reservation")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *MaterialHandler) ReleaseReservation(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request materialReservationReleaseRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body release reservasi tidak valid", nil)
			return
		}
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.ReleaseReservation(c.Request.Context(), id,
		material.Actor{ID: actor.ID, Role: actor.Role}, request.Reason)
	if err != nil {
		h.writeError(c, err, "release inventory reservation")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func materialItems(request []materialItemRequest) []material.ItemInput {
	items := make([]material.ItemInput, 0, len(request))
	for _, item := range request {
		items = append(items, material.ItemInput{
			SparepartID: item.SparepartID, QtyRequired: item.QtyRequired,
			IsOptional: item.IsOptional, PreferredWarehouseID: item.PreferredWarehouseID,
			Notes: item.Notes,
		})
	}
	return items
}

func (h *MaterialHandler) writeError(c *gin.Context, err error, operation string) {
	var materialValidation *material.ValidationError
	var inventoryValidation *inventory.ValidationError
	switch {
	case errors.As(err, &materialValidation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data material tidak valid", materialValidation.Fields)
	case errors.As(err, &inventoryValidation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data material tidak valid", inventoryValidation.Fields)
	case errors.Is(err, material.ErrValidation), errors.Is(err, inventory.ErrValidation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data material tidak valid", nil)
	case errors.Is(err, material.ErrInvalidState), errors.Is(err, inventory.ErrInvalidState):
		writeUnitError(c, http.StatusConflict, "invalid_state", "Status material tidak memungkinkan aksi ini.", nil)
	case errors.Is(err, inventory.ErrInsufficientStock):
		writeUnitError(c, http.StatusConflict, "insufficient_stock", "Stok tersedia tidak mencukupi.", nil)
	case errors.Is(err, material.ErrNotFound), errors.Is(err, inventory.ErrNotFound):
		writeUnitError(c, http.StatusNotFound, "not_found", "Data tidak ditemukan", nil)
	case errors.Is(err, material.ErrConflict), errors.Is(err, inventory.ErrConflict):
		writeUnitError(c, http.StatusConflict, "conflict", "Data sudah ada atau melanggar constraint", nil)
	default:
		h.logger.Error("material operation failed", "operation", operation, "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
	}
}
