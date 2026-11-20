package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/units"
)

// UnitService is the application surface used by unit-master HTTP handlers.
type UnitService interface {
	ListUnitModels(ctx context.Context, q string, activeOnly bool, page, limit int) ([]units.UnitModel, int, error)
	CreateUnitModel(ctx context.Context, input units.UnitModelInput) (units.UnitModel, error)
	UpdateUnitModel(ctx context.Context, id int64, patch units.UnitModelPatch) (units.UnitModel, error)

	ListEngineModels(ctx context.Context, q string, activeOnly bool, page, limit int) ([]units.EngineModel, int, error)
	CreateEngineModel(ctx context.Context, input units.EngineModelInput) (units.EngineModel, error)
	UpdateEngineModel(ctx context.Context, id int64, patch units.EngineModelPatch) (units.EngineModel, error)

	ListUnits(ctx context.Context, filter units.UnitFilter) ([]units.Unit, int, error)
	GetUnit(ctx context.Context, id int64) (units.Unit, error)
	CreateUnit(ctx context.Context, input units.UnitInput) (units.Unit, error)
	UpdateUnit(ctx context.Context, id int64, patch units.UnitPatch) (units.Unit, error)

	ListMeterReadings(ctx context.Context, unitID int64, limit int) ([]units.MeterReading, error)
	AddManualReading(ctx context.Context, unitID int64, input units.MeterReadingInput, actorRole string) (units.MeterReading, error)
	DashboardSummary(ctx context.Context) (units.DashboardSummary, error)
}

type UnitHandler struct {
	service UnitService
	logger  *slog.Logger
}

func NewUnitHandler(service UnitService, logger *slog.Logger) *UnitHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &UnitHandler{service: service, logger: logger}
}

type unitModelRequest struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Category     string `json:"category"`
	Notes        string `json:"notes"`
	IsActive     *bool  `json:"is_active"`
}

type unitModelPatchRequest struct {
	Manufacturer *string `json:"manufacturer"`
	Model        *string `json:"model"`
	Category     *string `json:"category"`
	Notes        *string `json:"notes"`
	IsActive     *bool   `json:"is_active"`
}

type engineModelRequest struct {
	Manufacturer   string `json:"manufacturer"`
	Model          string `json:"model"`
	FuelType       string `json:"fuel_type"`
	DisplacementCC *int   `json:"displacement_cc"`
	Notes          string `json:"notes"`
	IsActive       *bool  `json:"is_active"`
}

type engineModelPatchRequest struct {
	Manufacturer   *string `json:"manufacturer"`
	Model          *string `json:"model"`
	FuelType       *string `json:"fuel_type"`
	DisplacementCC *int    `json:"displacement_cc"`
	Notes          *string `json:"notes"`
	IsActive       *bool   `json:"is_active"`
}

type unitRequest struct {
	Code               string `json:"code"`
	UnitType           string `json:"unit_type"`
	Merk               string `json:"merk"`
	ClassUnit          string `json:"class_unit"`
	LicensePlate       string `json:"license_plate"`
	SiteLocation       string `json:"site_location"`
	UnitModelID        *int64 `json:"unit_model_id"`
	EngineModelID      *int64 `json:"engine_model_id"`
	SerialNumber       string `json:"serial_number"`
	EngineSerialNumber string `json:"engine_serial_number"`
	IsBreakdown        *bool  `json:"is_breakdown"`
	IsActive           *bool  `json:"is_active"`
}

type unitPatchRequest struct {
	Code               *string `json:"code"`
	UnitType           *string `json:"unit_type"`
	Merk               *string `json:"merk"`
	ClassUnit          *string `json:"class_unit"`
	LicensePlate       *string `json:"license_plate"`
	SiteLocation       *string `json:"site_location"`
	UnitModelID        *int64  `json:"unit_model_id"`
	EngineModelID      *int64  `json:"engine_model_id"`
	SerialNumber       *string `json:"serial_number"`
	EngineSerialNumber *string `json:"engine_serial_number"`
	IsBreakdown        *bool   `json:"is_breakdown"`
	IsActive           *bool   `json:"is_active"`
}

type meterReadingRequest struct {
	ReadingType string    `json:"reading_type"`
	Value       float64   `json:"value"`
	RecordedAt  time.Time `json:"recorded_at"`
	Notes       string    `json:"notes"`
}

func (h *UnitHandler) ListUnitModels(c *gin.Context) {
	page, limit, err := parsePagination(c)
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_pagination", err.Error(), nil)
		return
	}
	items, total, err := h.service.ListUnitModels(c.Request.Context(), c.Query("q"), parseBool(c.Query("active_only")), page, limit)
	if err != nil {
		h.writeError(c, err, "list unit models")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *UnitHandler) CreateUnitModel(c *gin.Context) {
	var request unitModelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body unit model tidak valid", nil)
		return
	}
	item, err := h.service.CreateUnitModel(c.Request.Context(), units.UnitModelInput{
		Manufacturer: request.Manufacturer,
		Model:        request.Model,
		Category:     request.Category,
		Notes:        request.Notes,
		IsActive:     request.IsActive,
	})
	if err != nil {
		h.writeError(c, err, "create unit model")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *UnitHandler) UpdateUnitModel(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request unitModelPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body unit model tidak valid", nil)
		return
	}
	item, err := h.service.UpdateUnitModel(c.Request.Context(), id, units.UnitModelPatch{
		Manufacturer: request.Manufacturer,
		Model:        request.Model,
		Category:     request.Category,
		Notes:        request.Notes,
		IsActive:     request.IsActive,
	})
	if err != nil {
		h.writeError(c, err, "update unit model")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *UnitHandler) ListEngineModels(c *gin.Context) {
	page, limit, err := parsePagination(c)
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_pagination", err.Error(), nil)
		return
	}
	items, total, err := h.service.ListEngineModels(c.Request.Context(), c.Query("q"), parseBool(c.Query("active_only")), page, limit)
	if err != nil {
		h.writeError(c, err, "list engine models")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *UnitHandler) CreateEngineModel(c *gin.Context) {
	var request engineModelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body engine model tidak valid", nil)
		return
	}
	item, err := h.service.CreateEngineModel(c.Request.Context(), units.EngineModelInput{
		Manufacturer:   request.Manufacturer,
		Model:          request.Model,
		FuelType:       request.FuelType,
		DisplacementCC: request.DisplacementCC,
		Notes:          request.Notes,
		IsActive:       request.IsActive,
	})
	if err != nil {
		h.writeError(c, err, "create engine model")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *UnitHandler) UpdateEngineModel(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request engineModelPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body engine model tidak valid", nil)
		return
	}
	item, err := h.service.UpdateEngineModel(c.Request.Context(), id, units.EngineModelPatch{
		Manufacturer:   request.Manufacturer,
		Model:          request.Model,
		FuelType:       request.FuelType,
		DisplacementCC: request.DisplacementCC,
		Notes:          request.Notes,
		IsActive:       request.IsActive,
	})
	if err != nil {
		h.writeError(c, err, "update engine model")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *UnitHandler) ListUnits(c *gin.Context) {
	page, limit, err := parsePagination(c)
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_pagination", err.Error(), nil)
		return
	}
	active, err := parseOptionalBool(c.Query("is_active"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter is_active tidak valid", nil)
		return
	}
	var unitModelID *int64
	if raw := strings.TrimSpace(c.Query("unit_model_id")); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || value <= 0 {
			writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_model_id tidak valid", nil)
			return
		}
		unitModelID = &value
	}
	items, total, err := h.service.ListUnits(c.Request.Context(), units.UnitFilter{
		Query:        c.Query("q"),
		IsActive:     active,
		UnitModelID:  unitModelID,
		SiteLocation: c.Query("site_location"),
		Page:         page,
		Limit:        limit,
		Sort:         c.Query("sort"),
		Order:        c.Query("order"),
	})
	if err != nil {
		h.writeError(c, err, "list units")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *UnitHandler) GetUnit(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.GetUnit(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "get unit")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *UnitHandler) CreateUnit(c *gin.Context) {
	var request unitRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body unit tidak valid", nil)
		return
	}
	item, err := h.service.CreateUnit(c.Request.Context(), units.UnitInput{
		Code:               request.Code,
		UnitType:           request.UnitType,
		Merk:               request.Merk,
		ClassUnit:          request.ClassUnit,
		LicensePlate:       request.LicensePlate,
		SiteLocation:       request.SiteLocation,
		UnitModelID:        request.UnitModelID,
		EngineModelID:      request.EngineModelID,
		SerialNumber:       request.SerialNumber,
		EngineSerialNumber: request.EngineSerialNumber,
		IsBreakdown:        request.IsBreakdown,
		IsActive:           request.IsActive,
	})
	if err != nil {
		h.writeError(c, err, "create unit")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *UnitHandler) UpdateUnit(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request unitPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body unit tidak valid", nil)
		return
	}
	item, err := h.service.UpdateUnit(c.Request.Context(), id, units.UnitPatch{
		Code:               request.Code,
		UnitType:           request.UnitType,
		Merk:               request.Merk,
		ClassUnit:          request.ClassUnit,
		LicensePlate:       request.LicensePlate,
		SiteLocation:       request.SiteLocation,
		UnitModelID:        request.UnitModelID,
		EngineModelID:      request.EngineModelID,
		SerialNumber:       request.SerialNumber,
		EngineSerialNumber: request.EngineSerialNumber,
		IsBreakdown:        request.IsBreakdown,
		IsActive:           request.IsActive,
	})
	if err != nil {
		h.writeError(c, err, "update unit")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *UnitHandler) ListMeterReadings(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.service.ListMeterReadings(c.Request.Context(), id, limit)
	if err != nil {
		h.writeError(c, err, "list meter readings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *UnitHandler) AddMeterReading(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request meterReadingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body reading tidak valid", nil)
		return
	}
	user, _ := currentUser(c)
	item, err := h.service.AddManualReading(c.Request.Context(), id, units.MeterReadingInput{
		ReadingType: request.ReadingType,
		Value:       request.Value,
		RecordedAt:  request.RecordedAt,
		Notes:       request.Notes,
	}, user.Role)
	if err != nil {
		h.writeError(c, err, "add meter reading")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *UnitHandler) DashboardSummary(c *gin.Context) {
	item, err := h.service.DashboardSummary(c.Request.Context())
	if err != nil {
		h.writeError(c, err, "dashboard summary")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *UnitHandler) writeError(c *gin.Context, err error, operation string) {
	var validation *units.ValidationError
	switch {
	case errors.As(err, &validation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data tidak valid", validation.Fields)
	case errors.Is(err, units.ErrValidation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data tidak valid", nil)
	case errors.Is(err, units.ErrNotFound):
		writeUnitError(c, http.StatusNotFound, "not_found", "Data tidak ditemukan", nil)
	case errors.Is(err, units.ErrConflict):
		writeUnitError(c, http.StatusConflict, "conflict", "Data sudah ada atau melanggar constraint", nil)
	case errors.Is(err, units.ErrInactive):
		writeUnitError(c, http.StatusConflict, "unit_inactive", "Unit tidak aktif", nil)
	case errors.Is(err, units.ErrReadingRegression):
		writeUnitError(c, http.StatusConflict, "reading_below_current", "Reading lebih rendah dari nilai saat ini; hanya admin yang dapat melakukan koreksi dengan alasan", nil)
	default:
		h.logger.Error("unit operation failed", "operation", operation, "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
	}
}

func writeList(c *gin.Context, data any, page, limit, total int) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": gin.H{"page": page, "limit": limit, "total": total},
	})
}

func writeUnitError(c *gin.Context, status int, code, message string, fields map[string]string) {
	errorBody := gin.H{"code": code, "message": message}
	if len(fields) > 0 {
		errorBody["fields"] = fields
	}
	c.JSON(status, gin.H{"error": errorBody})
}

func parsePagination(c *gin.Context) (int, int, error) {
	page := 1
	limit := units.DefaultLimit
	if raw := c.Query("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return 0, 0, errors.New("page harus bilangan positif")
		}
		page = value
	}
	if raw := c.Query("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > units.MaxPageLimit {
			return 0, 0, errors.New("limit harus antara 1 dan 100")
		}
		limit = value
	}
	return page, limit, nil
}

func parseOptionalBool(raw string) (*bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func parseBool(raw string) bool {
	value, _ := strconv.ParseBool(raw)
	return value
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		writeUnitError(c, http.StatusBadRequest, "invalid_id", "ID tidak valid", nil)
		return 0, false
	}
	return id, true
}
