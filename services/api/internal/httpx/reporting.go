package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/reporting"
)

// ReportingService is the application surface used by dashboard and report
// handlers. All methods are read-only.
type ReportingService interface {
	Summary(ctx context.Context) (map[string]any, error)
	PM(ctx context.Context, filter reporting.Filter) (reporting.DashboardPM, error)
	Material(ctx context.Context, filter reporting.Filter) (reporting.DashboardMaterial, error)
	Inventory(ctx context.Context, filter reporting.Filter) (reporting.DashboardInventory, error)
	Maintenance(ctx context.Context, filter reporting.Filter) (reporting.DashboardMaintenance, error)
	Report(ctx context.Context, name string, filter reporting.Filter) (reporting.Table, string, error)
	RenderCSV(w io.Writer, table reporting.Table) error
}

type ReportingHandler struct {
	service ReportingService
	loc     *time.Location
	logger  *slog.Logger
}

func NewReportingHandler(service ReportingService, loc *time.Location, logger *slog.Logger) *ReportingHandler {
	if loc == nil {
		loc = appLocation(nil)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ReportingHandler{service: service, loc: loc, logger: logger}
}

func (h *ReportingHandler) Summary(c *gin.Context) {
	summary, err := h.service.Summary(c.Request.Context())
	if err != nil {
		h.writeError(c, err, "dashboard summary")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": summary})
}

func (h *ReportingHandler) PM(c *gin.Context) {
	filter, ok := h.parseFilter(c)
	if !ok {
		return
	}
	result, err := h.service.PM(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err, "dashboard pm")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *ReportingHandler) Material(c *gin.Context) {
	filter, ok := h.parseFilter(c)
	if !ok {
		return
	}
	result, err := h.service.Material(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err, "dashboard material")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *ReportingHandler) Inventory(c *gin.Context) {
	filter, ok := h.parseFilter(c)
	if !ok {
		return
	}
	result, err := h.service.Inventory(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err, "dashboard inventory")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *ReportingHandler) Maintenance(c *gin.Context) {
	filter, ok := h.parseFilter(c)
	if !ok {
		return
	}
	result, err := h.service.Maintenance(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err, "dashboard maintenance")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *ReportingHandler) Report(c *gin.Context) {
	name := c.Param("name")
	allowed := false
	for _, candidate := range reporting.ReportNames() {
		if candidate == name {
			allowed = true
			break
		}
	}
	if !allowed {
		writeUnitError(c, http.StatusNotFound, "not_found", "Laporan tidak ditemukan", nil)
		return
	}
	filter, ok := h.parseFilter(c)
	if !ok {
		return
	}
	table, filename, err := h.service.Report(c.Request.Context(), name, filter)
	if err != nil {
		h.writeError(c, err, "report "+name)
		return
	}
	if filter.Format == reporting.FormatCSV {
		var buffer bytes.Buffer
		if err := h.service.RenderCSV(&buffer, table); err != nil {
			h.logger.Error("render csv failed", "report", name, "error", err)
			writeUnitError(c, http.StatusInternalServerError, "internal_error", "Gagal membuat CSV", nil)
			return
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": table})
}

func (h *ReportingHandler) parseFilter(c *gin.Context) (reporting.Filter, bool) {
	filter, err := reporting.ParseFilter(c.Request.URL.Query(), h.loc)
	if err != nil {
		var validation *reporting.ValidationError
		if errors.As(err, &validation) {
			writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter tidak valid", validation.Fields)
		} else {
			writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter tidak valid", nil)
		}
		return reporting.Filter{}, false
	}
	return filter, true
}

func (h *ReportingHandler) writeError(c *gin.Context, err error, operation string) {
	if errors.Is(err, reporting.ErrInvalidFilter) {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter tidak valid", nil)
		return
	}
	h.logger.Error("reporting operation failed", "operation", operation, "error", err)
	writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
}
