package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/settings"
)

type SettingsService interface {
	Get(ctx context.Context) (settings.Snapshot, error)
	Update(ctx context.Context, actorID int64, patch settings.Patch) (settings.Snapshot, error)
}

type SettingsHandler struct {
	service SettingsService
	logger  *slog.Logger
}

func NewSettingsHandler(service SettingsService, logger *slog.Logger) *SettingsHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &SettingsHandler{service: service, logger: logger}
}

type settingsPatchRequest struct {
	P2HAutoWO             *bool `json:"p2h_auto_wo"`
	CandidateSLAHours     *int  `json:"candidate_sla_hours"`
	CandidateBackfillDays *int  `json:"candidate_backfill_days"`
	PMAutoWO              *bool `json:"pm_auto_wo"`
	PMUpcomingDays        *int  `json:"pm_upcoming_days"`
	PMDueSoonDays         *int  `json:"pm_due_soon_days"`
}

func (h *SettingsHandler) Get(c *gin.Context) {
	snapshot, err := h.service.Get(c.Request.Context())
	if err != nil {
		h.writeError(c, err, "get settings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": snapshot})
}

func (h *SettingsHandler) Update(c *gin.Context) {
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	var request settingsPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body setting tidak valid", nil)
		return
	}
	snapshot, err := h.service.Update(c.Request.Context(), actor.ID, settings.Patch{
		P2HAutoWO:             request.P2HAutoWO,
		CandidateSLAHours:     request.CandidateSLAHours,
		CandidateBackfillDays: request.CandidateBackfillDays,
		PMAutoWO:              request.PMAutoWO,
		PMUpcomingDays:        request.PMUpcomingDays,
		PMDueSoonDays:         request.PMDueSoonDays,
	})
	if err != nil {
		h.writeError(c, err, "update settings")
		return
	}
	h.logger.Info("settings updated", "actor_id", actor.ID)
	c.JSON(http.StatusOK, gin.H{"data": snapshot})
}

func (h *SettingsHandler) writeError(c *gin.Context, err error, operation string) {
	var validation *settings.ValidationError
	switch {
	case errors.As(err, &validation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Setting tidak valid", validation.Fields)
	case errors.Is(err, settings.ErrValidation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Setting tidak valid", nil)
	case errors.Is(err, settings.ErrNotFound):
		writeUnitError(c, http.StatusNotFound, "setting_not_found", "Setting tidak ditemukan", nil)
	default:
		h.logger.Error("settings operation failed", "operation", operation, "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
	}
}
