package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/units"
)

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// UnitService is the application surface used by unit-master HTTP handlers.
type UnitService interface {
	ListUnitModels(ctx context.Context, filter units.ModelFilter) ([]units.UnitModel, int, error)
	CreateUnitModel(ctx context.Context, input units.UnitModelInput) (units.UnitModel, error)
	UpdateUnitModel(ctx context.Context, id int64, patch units.UnitModelPatch) (units.UnitModel, error)

	ListEngineModels(ctx context.Context, filter units.ModelFilter) ([]units.EngineModel, int, error)
	CreateEngineModel(ctx context.Context, input units.EngineModelInput) (units.EngineModel, error)
	UpdateEngineModel(ctx context.Context, id int64, patch units.EngineModelPatch) (units.EngineModel, error)

	ListUnits(ctx context.Context, filter units.UnitFilter) ([]units.Unit, int, error)
	GetUnit(ctx context.Context, id int64) (units.Unit, error)
	CreateUnit(ctx context.Context, input units.UnitInput) (units.Unit, error)
	ImportUnits(ctx context.Context, file []byte) (units.ImportResult, error)
	UpdateUnit(ctx context.Context, id int64, patch units.UnitPatch) (units.Unit, error)

	ListMeterReadings(ctx context.Context, unitID int64, limit int) ([]units.MeterReading, error)
	AddManualReading(ctx context.Context, unitID int64, input units.MeterReadingInput, actorRole string) (units.MeterReading, error)
	DashboardSummary(ctx context.Context) (units.DashboardSummary, error)

	ListMaintenanceHistory(ctx context.Context, unitID int64, filter units.HistoryFilter) ([]units.MaintenanceHistory, int, error)
	ListComponentHistory(ctx context.Context, unitID int64, filter units.ComponentHistoryFilter) ([]units.ComponentHistory, int, error)
	ListComponentIntervals(ctx context.Context, unitID int64) ([]units.ComponentInterval, error)
	UnitDowntime(ctx context.Context, unitID int64, filter units.DowntimeFilter) (units.UnitDowntime, error)
	ListComponents(ctx context.Context, activeOnly bool) ([]units.Component, error)
}

type UnitHandler struct {
	service UnitService
	logger  *slog.Logger
	loc     *time.Location
}

func NewUnitHandler(service UnitService, logger *slog.Logger, loc ...*time.Location) *UnitHandler {
	if logger == nil {
		logger = slog.Default()
	}
	location := appLocation(loc)
	return &UnitHandler{service: service, logger: logger, loc: location}
}

// appLocation resolves the configured IANA location. A missing location is a
// configuration error: there is deliberately no fixed-offset fallback.
func appLocation(locations []*time.Location) *time.Location {
	if len(locations) > 0 && locations[0] != nil {
		return locations[0]
	}
	location, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		panic("invalid application timezone Asia/Makassar: " + err.Error())
	}
	return location
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
	page, limit := parsePagination(c)
	items, total, err := h.service.ListUnitModels(c.Request.Context(), units.ModelFilter{
		Query:      c.Query("q"),
		ActiveOnly: parseBool(c.Query("active_only")),
		Page:       page,
		Limit:      limit,
		Sort:       c.Query("sort"),
		Order:      c.Query("order"),
	})
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
	page, limit := parsePagination(c)
	items, total, err := h.service.ListEngineModels(c.Request.Context(), units.ModelFilter{
		Query:      c.Query("q"),
		ActiveOnly: parseBool(c.Query("active_only")),
		Page:       page,
		Limit:      limit,
		Sort:       c.Query("sort"),
		Order:      c.Query("order"),
	})
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
	page, limit := parsePagination(c)
	filter, ok := h.parseUnitFilter(c)
	if !ok {
		return
	}
	filter.Page = page
	filter.Limit = limit
	items, total, err := h.service.ListUnits(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err, "list units")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *UnitHandler) parseUnitFilter(c *gin.Context) (units.UnitFilter, bool) {
	active, err := parseOptionalBool(c.Query("is_active"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter is_active tidak valid", nil)
		return units.UnitFilter{}, false
	}
	var unitModelID *int64
	if raw := strings.TrimSpace(c.Query("unit_model_id")); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || value <= 0 {
			writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_model_id tidak valid", nil)
			return units.UnitFilter{}, false
		}
		unitModelID = &value
	}
	return units.UnitFilter{
		Query:        c.Query("q"),
		IsActive:     active,
		UnitModelID:  unitModelID,
		SiteLocation: c.Query("site_location"),
		Sort:         c.Query("sort"),
		Order:        c.Query("order"),
	}, true
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

func (h *UnitHandler) ImportUnits(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, units.UnitImportMaxFileBytes+(1<<20))
	header, err := c.FormFile("file")
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "File XLSX wajib diunggah", nil)
		return
	}
	if header.Size > units.UnitImportMaxFileBytes {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Ukuran file maksimal 5 MB", nil)
		return
	}
	uploaded, err := header.Open()
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "File tidak dapat dibaca", nil)
		return
	}
	defer uploaded.Close()

	data, err := io.ReadAll(io.LimitReader(uploaded, units.UnitImportMaxFileBytes+1))
	if err != nil || len(data) > units.UnitImportMaxFileBytes {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Ukuran file maksimal 5 MB", nil)
		return
	}
	result, err := h.service.ImportUnits(c.Request.Context(), data)
	if err != nil {
		h.writeError(c, err, "import units")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *UnitHandler) ExportUnits(c *gin.Context) {
	filter, ok := h.parseUnitFilter(c)
	if !ok {
		return
	}
	const pageSize = units.MaxPageLimit
	const maxRows = 10000
	filter.Page = 1
	filter.Limit = pageSize

	all := make([]units.Unit, 0)
	for {
		batch, total, err := h.service.ListUnits(c.Request.Context(), filter)
		if err != nil {
			h.writeError(c, err, "export units")
			return
		}
		all = append(all, batch...)
		if len(batch) == 0 || len(all) >= total || len(all) >= maxRows {
			break
		}
		filter.Page++
	}
	if len(all) > maxRows {
		h.logger.Warn("unit export truncated", "max_rows", maxRows)
		all = all[:maxRows]
	}

	unitModels, engineModels := h.referenceModels(c)
	var buffer bytes.Buffer
	if err := units.WriteUnitsXLSX(&buffer, all, unitModels, engineModels); err != nil {
		h.logger.Error("render unit xlsx failed", "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "XLSX tidak dapat dibuat", nil)
		return
	}
	h.writeXLSX(c, buffer.Bytes(), fmt.Sprintf("unit-%s.xlsx", time.Now().In(h.loc).Format("2006-01-02")))
}

func (h *UnitHandler) UnitImportTemplate(c *gin.Context) {
	unitModels, engineModels := h.referenceModels(c)
	var buffer bytes.Buffer
	if err := units.WriteUnitImportTemplate(&buffer, unitModels, engineModels); err != nil {
		h.logger.Error("render unit import template failed", "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Template tidak dapat dibuat", nil)
		return
	}
	h.writeXLSX(c, buffer.Bytes(), "template-import-unit.xlsx")
}

func (h *UnitHandler) referenceModels(c *gin.Context) ([]units.UnitModel, []units.EngineModel) {
	unitModels, _, err := h.service.ListUnitModels(c.Request.Context(), units.ModelFilter{
		Page: 1, Limit: units.MaxPageLimit, Sort: "manufacturer", Order: "asc",
	})
	if err != nil {
		h.logger.Warn("load unit models for xlsx failed", "error", err)
		unitModels = nil
	}
	engineModels, _, err := h.service.ListEngineModels(c.Request.Context(), units.ModelFilter{
		Page: 1, Limit: units.MaxPageLimit, Sort: "manufacturer", Order: "asc",
	})
	if err != nil {
		h.logger.Warn("load engine models for xlsx failed", "error", err)
		engineModels = nil
	}
	return unitModels, engineModels
}

func (h *UnitHandler) writeXLSX(c *gin.Context, data []byte, filename string) {
	c.Header("Content-Type", xlsxContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, xlsxContentType, data)
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

func (h *UnitHandler) ListMaintenanceHistory(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	page, limit := parsePagination(c)
	from, to, err := h.parseUnitPeriod(c)
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter periode tidak valid", nil)
		return
	}
	items, total, err := h.service.ListMaintenanceHistory(c.Request.Context(), id, units.HistoryFilter{
		Page:            page,
		Limit:           limit,
		Status:          c.Query("status"),
		MaintenanceType: c.Query("maintenance_type"),
		From:            from,
		To:              to,
	})
	if err != nil {
		h.writeError(c, err, "list maintenance history")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *UnitHandler) ListUnitComponents(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	page, limit := parsePagination(c)
	from, to, err := h.parseUnitPeriod(c)
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter periode tidak valid", nil)
		return
	}
	items, total, err := h.service.ListComponentHistory(c.Request.Context(), id, units.ComponentHistoryFilter{
		Page:  page,
		Limit: limit,
		From:  from,
		To:    to,
	})
	if err != nil {
		h.writeError(c, err, "list component history")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *UnitHandler) ListComponentIntervals(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ListComponentIntervals(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "list component intervals")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *UnitHandler) UnitDowntime(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	from, to, err := h.parseUnitPeriod(c)
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter periode tidak valid", nil)
		return
	}
	result, err := h.service.UnitDowntime(c.Request.Context(), id, units.DowntimeFilter{
		From:  from,
		To:    to,
		Limit: parsePositiveInt(c.Query("limit"), 50),
	})
	if err != nil {
		h.writeError(c, err, "unit downtime")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *UnitHandler) ListComponents(c *gin.Context) {
	items, err := h.service.ListComponents(c.Request.Context(), parseBool(c.Query("active_only")))
	if err != nil {
		h.writeError(c, err, "list components")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
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

func parsePagination(c *gin.Context) (int, int) {
	page := parsePositiveInt(c.Query("page"), 1)
	limit := parsePositiveInt(c.Query("limit"), units.DefaultLimit)
	if limit > units.MaxPageLimit {
		limit = units.MaxPageLimit
	}
	return page, limit
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 {
		return fallback
	}
	return value
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

// parseUnitPeriod treats from/to as inclusive WITA calendar dates. The upper
// bound is converted to the next day's WITA midnight and remains exclusive.
func (h *UnitHandler) parseUnitPeriod(c *gin.Context) (*time.Time, *time.Time, error) {
	from, err := parseUnitDate(c.Query("from"), h.loc)
	if err != nil {
		return nil, nil, err
	}
	to, err := parseUnitDate(c.Query("to"), h.loc)
	if err != nil {
		return nil, nil, err
	}
	if to != nil {
		next := to.AddDate(0, 0, 1)
		to = &next
	}
	return from, to, nil
}

func parseUnitDate(raw string, loc *time.Location) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := time.ParseInLocation("2006-01-02", raw, loc)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
