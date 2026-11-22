package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/spareparts"
)

type SparepartHandler struct {
	service spareparts.Service
	logger  *slog.Logger
}

func NewSparepartHandler(service spareparts.Service, logger *slog.Logger) *SparepartHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &SparepartHandler{service: service, logger: logger}
}

type sparepartRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	UOM         string `json:"uom"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type sparepartPatchRequest struct {
	Code        *string `json:"code"`
	Name        *string `json:"name"`
	Category    *string `json:"category"`
	UOM         *string `json:"uom"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type sparepartNumberRequest struct {
	PartNumber string `json:"part_number"`
	Brand      string `json:"brand"`
	NumberType string `json:"number_type"`
	IsPrimary  bool   `json:"is_primary"`
	Notes      string `json:"notes"`
}

type sparepartNumberPatchRequest struct {
	PartNumber *string `json:"part_number"`
	Brand      *string `json:"brand"`
	NumberType *string `json:"number_type"`
	IsPrimary  *bool   `json:"is_primary"`
	Notes      *string `json:"notes"`
}

type sparepartAlternativeRequest struct {
	AlternativeSparepartID int64  `json:"alternative_sparepart_id"`
	RelationType           string `json:"relation_type"`
	Notes                  string `json:"notes"`
}

type sparepartCompatibilityRequest struct {
	UnitModelID   *int64 `json:"unit_model_id"`
	EngineModelID *int64 `json:"engine_model_id"`
	Configuration string `json:"configuration"`
	Source        string `json:"source"`
	Notes         string `json:"notes"`
}

type sparepartVerifyRequest struct {
	Status        string `json:"status"`
	ConditionNote string `json:"condition_note"`
	Notes         string `json:"notes"`
	Source        string `json:"source"`
}

func (h *SparepartHandler) List(c *gin.Context) {
	page, limit := parsePagination(c)
	activeOnly, err := parseOptionalBool(c.Query("active_only"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter active_only tidak valid", nil)
		return
	}
	unitModelID, err := parseOptionalQueryID(c.Query("unit_model_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_model_id tidak valid", nil)
		return
	}
	engineModelID, err := parseOptionalQueryID(c.Query("engine_model_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter engine_model_id tidak valid", nil)
		return
	}
	items, total, err := h.service.List(c.Request.Context(), spareparts.Filter{
		Query:               strings.TrimSpace(c.Query("q")),
		Category:            strings.TrimSpace(c.Query("category")),
		ActiveOnly:          activeOnly,
		UnitModelID:         unitModelID,
		EngineModelID:       engineModelID,
		CompatibilityStatus: strings.TrimSpace(c.Query("compatibility_status")),
		Page:                page,
		Limit:               limit,
		Sort:                c.Query("sort"),
		Order:               c.Query("order"),
	})
	if err != nil {
		h.writeError(c, err, "list spareparts")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *SparepartHandler) Search(c *gin.Context) {
	page, limit := parsePagination(c)
	activeOnly := true
	unitModelID, err := parseOptionalQueryID(c.Query("unit_model_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_model_id tidak valid", nil)
		return
	}
	engineModelID, err := parseOptionalQueryID(c.Query("engine_model_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter engine_model_id tidak valid", nil)
		return
	}
	items, total, err := h.service.List(c.Request.Context(), spareparts.Filter{
		Query:               strings.TrimSpace(c.Query("q")),
		Category:            strings.TrimSpace(c.Query("category")),
		ActiveOnly:          &activeOnly,
		UnitModelID:         unitModelID,
		EngineModelID:       engineModelID,
		CompatibilityStatus: strings.TrimSpace(c.Query("compatibility_status")),
		Page:                page,
		Limit:               limit,
		Sort:                "code",
	})
	if err != nil {
		h.writeError(c, err, "search spareparts")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *SparepartHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "get sparepart")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *SparepartHandler) Create(c *gin.Context) {
	var request sparepartRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body sparepart tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.Create(c.Request.Context(), spareparts.Actor{ID: actor.ID, Role: actor.Role},
		spareparts.Input{
			Code:        request.Code,
			Name:        request.Name,
			Category:    request.Category,
			UOM:         request.UOM,
			Description: request.Description,
			IsActive:    request.IsActive,
		})
	if err != nil {
		h.writeError(c, err, "create sparepart")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *SparepartHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request sparepartPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body sparepart tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.Update(c.Request.Context(), id,
		spareparts.Actor{ID: actor.ID, Role: actor.Role}, spareparts.Patch{
			Code:        request.Code,
			Name:        request.Name,
			Category:    request.Category,
			UOM:         request.UOM,
			Description: request.Description,
			IsActive:    request.IsActive,
		})
	if err != nil {
		h.writeError(c, err, "update sparepart")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *SparepartHandler) ListNumbers(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ListNumbers(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "list sparepart numbers")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *SparepartHandler) CreateNumber(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request sparepartNumberRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body nomor part tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.CreateNumber(c.Request.Context(), id,
		spareparts.Actor{ID: actor.ID, Role: actor.Role}, spareparts.NumberInput{
			PartNumber: request.PartNumber,
			Brand:      request.Brand,
			NumberType: request.NumberType,
			IsPrimary:  request.IsPrimary,
			Notes:      request.Notes,
		})
	if err != nil {
		h.writeError(c, err, "create sparepart number")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *SparepartHandler) UpdateNumber(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	numberID, ok := parseParamID(c, "number_id")
	if !ok {
		return
	}
	var request sparepartNumberPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body nomor part tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.UpdateNumber(c.Request.Context(), id, numberID,
		spareparts.Actor{ID: actor.ID, Role: actor.Role}, spareparts.NumberPatch{
			PartNumber: request.PartNumber,
			Brand:      request.Brand,
			NumberType: request.NumberType,
			IsPrimary:  request.IsPrimary,
			Notes:      request.Notes,
		})
	if err != nil {
		h.writeError(c, err, "update sparepart number")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *SparepartHandler) ListAlternatives(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ListAlternatives(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "list sparepart alternatives")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *SparepartHandler) CreateAlternative(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request sparepartAlternativeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body alternatif tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.CreateAlternative(c.Request.Context(), id,
		spareparts.Actor{ID: actor.ID, Role: actor.Role}, spareparts.AlternativeInput{
			AlternativeSparepartID: request.AlternativeSparepartID,
			RelationType:           request.RelationType,
			Notes:                  request.Notes,
		})
	if err != nil {
		h.writeError(c, err, "create sparepart alternative")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *SparepartHandler) VerifyAlternative(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	alternativeID, ok := parseParamID(c, "alternative_id")
	if !ok {
		return
	}
	var request sparepartVerifyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body verifikasi tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.VerifyAlternative(c.Request.Context(), id, alternativeID,
		spareparts.Actor{ID: actor.ID, Role: actor.Role}, spareparts.VerifyInput{
			Status: request.Status, ConditionNote: request.ConditionNote,
			Notes: request.Notes, Source: request.Source,
		})
	if err != nil {
		h.writeError(c, err, "verify sparepart alternative")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *SparepartHandler) ListCompatibility(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ListCompatibility(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "list sparepart compatibility")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *SparepartHandler) CreateCompatibility(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request sparepartCompatibilityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body compatibility tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.CreateCompatibility(c.Request.Context(), id,
		spareparts.Actor{ID: actor.ID, Role: actor.Role}, spareparts.CompatibilityInput{
			UnitModelID:   request.UnitModelID,
			EngineModelID: request.EngineModelID,
			Configuration: request.Configuration,
			Source:        request.Source,
			Notes:         request.Notes,
		})
	if err != nil {
		h.writeError(c, err, "create sparepart compatibility")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *SparepartHandler) VerifyCompatibility(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	compatibilityID, ok := parseParamID(c, "compat_id")
	if !ok {
		return
	}
	var request sparepartVerifyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body verifikasi tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.VerifyCompatibility(c.Request.Context(), id, compatibilityID,
		spareparts.Actor{ID: actor.ID, Role: actor.Role}, spareparts.VerifyInput{
			Status: request.Status, ConditionNote: request.ConditionNote,
			Notes: request.Notes, Source: request.Source,
		})
	if err != nil {
		h.writeError(c, err, "verify sparepart compatibility")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *SparepartHandler) writeError(c *gin.Context, err error, operation string) {
	var validation *spareparts.ValidationError
	switch {
	case errors.As(err, &validation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data tidak valid", validation.Fields)
	case errors.Is(err, spareparts.ErrValidation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data tidak valid", nil)
	case errors.Is(err, spareparts.ErrNotFound):
		writeUnitError(c, http.StatusNotFound, "not_found", "Data tidak ditemukan", nil)
	case errors.Is(err, spareparts.ErrConflict):
		writeUnitError(c, http.StatusConflict, "conflict", "Data sudah ada atau melanggar constraint", nil)
	default:
		h.logger.Error("sparepart operation failed", "operation", operation, "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
	}
}

func parseParamID(c *gin.Context, name string) (int64, bool) {
	value, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || value <= 0 {
		writeUnitError(c, http.StatusBadRequest, "invalid_id", "ID tidak valid", nil)
		return 0, false
	}
	return value, true
}
