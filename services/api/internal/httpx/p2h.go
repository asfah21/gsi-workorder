package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/p2h"
)

type P2HService interface {
	List(ctx context.Context, filter p2h.ReportFilter) ([]p2h.Report, int, error)
	Get(ctx context.Context, id int64) (p2h.Report, error)
}

type P2HHandler struct {
	service P2HService
	logger  *slog.Logger
}

func NewP2HHandler(service P2HService, logger *slog.Logger) *P2HHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &P2HHandler{service: service, logger: logger}
}

func (h *P2HHandler) ListReports(c *gin.Context) {
	page, limit := parsePagination(c)
	var unitID *int64
	if raw := strings.TrimSpace(c.Query("unit_id")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_id tidak valid", nil)
			return
		}
		unitID = &value
	}
	items, total, err := h.service.List(c.Request.Context(), p2h.ReportFilter{
		Query:  strings.TrimSpace(c.Query("q")),
		State:  strings.TrimSpace(c.Query("state")),
		UnitID: unitID,
		Page:   page,
		Limit:  limit,
		Sort:   c.Query("sort"),
		Order:  c.Query("order"),
	})
	if err != nil {
		h.writeError(c, err, "list P2H reports")
		return
	}
	responses := make([]reportResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, newReportResponse(item))
	}
	writeList(c, responses, page, limit, total)
}

func (h *P2HHandler) GetReport(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	report, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "get P2H report")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": newReportResponse(report)})
}

func (h *P2HHandler) writeError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, p2h.ErrReportNotFound):
		writeUnitError(c, http.StatusNotFound, "report_not_found", "Laporan P2H tidak ditemukan", nil)
	default:
		h.logger.Error("P2H operation failed", "operation", operation, "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
	}
}
