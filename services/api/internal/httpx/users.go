package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/users"
)

type UserDirectoryService interface {
	ListActive(ctx context.Context, roles []string, query string, limit int) ([]users.User, int, error)
}

type UserDirectoryHandler struct {
	service UserDirectoryService
	logger  *slog.Logger
}

func NewUserDirectoryHandler(service UserDirectoryService, logger *slog.Logger) *UserDirectoryHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &UserDirectoryHandler{service: service, logger: logger}
}

func (h *UserDirectoryHandler) List(c *gin.Context) {
	roles := make([]string, 0)
	for _, raw := range strings.Split(c.Query("roles"), ",") {
		role := strings.TrimSpace(raw)
		if role != "" {
			roles = append(roles, role)
		}
	}
	limit := parsePositiveInt(c.Query("limit"), 100)
	if limit > 2000 {
		limit = 2000
	}
	items, total, err := h.service.ListActive(
		c.Request.Context(),
		roles,
		strings.TrimSpace(c.Query("q")),
		limit,
	)
	if err != nil {
		h.logger.Error("list user directory failed", "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
		return
	}
	page := 1
	writeList(c, items, page, limit, total)
}
