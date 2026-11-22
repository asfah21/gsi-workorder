package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/pm"
)

type pmPlanMaterialRequest struct {
	SparepartID int64   `json:"sparepart_id"`
	Qty         float64 `json:"qty"`
	IsOptional  bool    `json:"is_optional"`
	Notes       string  `json:"notes"`
}

type pmPlanMaterialPatchRequest struct {
	Qty        *float64 `json:"qty"`
	IsOptional *bool    `json:"is_optional"`
	Notes      *string  `json:"notes"`
}

func (h *PMHandler) ListPlanMaterials(c *gin.Context) {
	planID, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ListPlanMaterials(c.Request.Context(), planID)
	if err != nil {
		h.writeError(c, err, "list PM plan materials")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *PMHandler) AddPlanMaterial(c *gin.Context) {
	planID, ok := parseID(c)
	if !ok {
		return
	}
	var request pmPlanMaterialRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body material plan tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.AddPlanMaterial(c.Request.Context(), planID,
		pm.Actor{ID: actor.ID, Role: actor.Role}, pm.PlanMaterialInput{
			SparepartID: request.SparepartID,
			Qty:         request.Qty,
			IsOptional:  request.IsOptional,
			Notes:       request.Notes,
		})
	if err != nil {
		h.writeError(c, err, "add PM plan material")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *PMHandler) UpdatePlanMaterial(c *gin.Context) {
	planID, ok := parseID(c)
	if !ok {
		return
	}
	materialID, ok := parseParamID(c, "material_id")
	if !ok {
		return
	}
	var request pmPlanMaterialPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body material plan tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.UpdatePlanMaterial(c.Request.Context(), planID, materialID,
		pm.Actor{ID: actor.ID, Role: actor.Role}, pm.PlanMaterialPatch{
			Qty: request.Qty, IsOptional: request.IsOptional, Notes: request.Notes,
		})
	if err != nil {
		h.writeError(c, err, "update PM plan material")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *PMHandler) DeletePlanMaterial(c *gin.Context) {
	planID, ok := parseID(c)
	if !ok {
		return
	}
	materialID, ok := parseParamID(c, "material_id")
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	if err := h.service.DeletePlanMaterial(c.Request.Context(), planID, materialID,
		pm.Actor{ID: actor.ID, Role: actor.Role}); err != nil {
		h.writeError(c, err, "delete PM plan material")
		return
	}
	c.Status(http.StatusNoContent)
}
