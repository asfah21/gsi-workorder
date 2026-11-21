package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/notifications"
)

type NotificationService interface {
	List(ctx context.Context, userID int64, page, limit int, unreadOnly bool) ([]notifications.Notification, int, int, error)
	MarkRead(ctx context.Context, userID int64, ids []int64, all bool) (int64, error)
}

type NotificationHandler struct {
	service NotificationService
	logger  *slog.Logger
}

func NewNotificationHandler(service NotificationService, logger *slog.Logger) *NotificationHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &NotificationHandler{service: service, logger: logger}
}

type markReadRequest struct {
	IDs []int64 `json:"ids"`
	All bool    `json:"all"`
}

func (h *NotificationHandler) List(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	page, limit := parsePagination(c)
	if limit > 50 {
		limit = 50
	}
	items, total, unread, err := h.service.List(c.Request.Context(), user.ID, page, limit, parseBool(c.Query("unread")))
	if err != nil {
		h.writeError(c, err, "list notifications")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{
			"page":   page,
			"limit":  limit,
			"total":  total,
			"unread": unread,
		},
	})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request markReadRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body notifikasi tidak valid", nil)
		return
	}
	updated, err := h.service.MarkRead(c.Request.Context(), user.ID, request.IDs, request.All)
	if err != nil {
		h.writeError(c, err, "mark notifications read")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"updated": updated}})
}

func (h *NotificationHandler) writeError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, notifications.ErrValidation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data notifikasi tidak valid", nil)
	default:
		h.logger.Error("notification operation failed", "operation", operation, "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
	}
}
