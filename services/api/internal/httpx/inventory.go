package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/inventory"
	"gsi-workorder/internal/material"
)

type InventoryHandler struct {
	service   inventory.Service
	materials material.Service
	logger    *slog.Logger
}

func NewInventoryHandler(service inventory.Service, materials material.Service, logger *slog.Logger) *InventoryHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &InventoryHandler{service: service, materials: materials, logger: logger}
}

func (h *InventoryHandler) recomputeMaterialReadiness(ctx context.Context, sparepartIDs []int64) {
	if h.materials == nil || len(sparepartIDs) == 0 {
		return
	}
	unique := make(map[int64]struct{}, len(sparepartIDs))
	ids := make([]int64, 0, len(sparepartIDs))
	for _, id := range sparepartIDs {
		if id <= 0 {
			continue
		}
		if _, exists := unique[id]; exists {
			continue
		}
		unique[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return
	}
	if _, err := h.materials.RecomputeForSpareparts(ctx, ids); err != nil {
		h.logger.Warn("material readiness recompute failed", "error", err)
	}
}

type warehouseRequest struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	SiteLocation string `json:"site_location"`
	IsActive     *bool  `json:"is_active"`
}

type warehousePatchRequest struct {
	Code         *string `json:"code"`
	Name         *string `json:"name"`
	SiteLocation *string `json:"site_location"`
	IsActive     *bool   `json:"is_active"`
}

type locationRequest struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	LocationType string `json:"location_type"`
	IsActive     *bool  `json:"is_active"`
}

type locationPatchRequest struct {
	Code         *string `json:"code"`
	Name         *string `json:"name"`
	LocationType *string `json:"location_type"`
	IsActive     *bool   `json:"is_active"`
}

type stockPolicyRequest struct {
	SparepartID int64    `json:"sparepart_id"`
	LocationID  int64    `json:"warehouse_location_id"`
	MinQty      float64  `json:"min_qty"`
	CriticalQty float64  `json:"critical_qty"`
	ReorderQty  *float64 `json:"reorder_qty"`
	IsActive    *bool    `json:"is_active"`
}

type stockPolicyPatchRequest struct {
	MinQty       *float64 `json:"min_qty"`
	CriticalQty  *float64 `json:"critical_qty"`
	ReorderQty   *float64 `json:"reorder_qty"`
	ClearReorder bool     `json:"clear_reorder_qty"`
	IsActive     *bool    `json:"is_active"`
}

type inventoryItemRequest struct {
	SparepartID    int64   `json:"sparepart_id"`
	FromLocationID *int64  `json:"from_location_id"`
	ToLocationID   *int64  `json:"to_location_id"`
	Qty            float64 `json:"qty"`
	Notes          string  `json:"notes"`
}

type inventoryPostRequest struct {
	ReferenceType  string                 `json:"reference_type"`
	ReferenceID    *int64                 `json:"reference_id"`
	ReferenceNo    string                 `json:"reference_no"`
	Notes          string                 `json:"notes"`
	IdempotencyKey string                 `json:"idempotency_key"`
	Items          []inventoryItemRequest `json:"items"`
}

type reverseRequest struct {
	Reason string `json:"reason"`
}

type opnameCreateRequest struct {
	WarehouseID int64  `json:"warehouse_id"`
	Notes       string `json:"notes"`
}

type opnameCountItemRequest struct {
	ItemID     int64   `json:"item_id"`
	CountedQty float64 `json:"counted_qty"`
	Notes      string  `json:"notes"`
}

type opnameCountRequest struct {
	Items []opnameCountItemRequest `json:"items"`
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

func (h *InventoryHandler) ListWarehouses(c *gin.Context) {
	page, limit := parsePagination(c)
	items, total, err := h.service.ListWarehouses(c.Request.Context(), c.Query("q"),
		parseBool(c.Query("active_only")), page, limit)
	if err != nil {
		h.writeError(c, err, "list warehouses")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *InventoryHandler) CreateWarehouse(c *gin.Context) {
	var request warehouseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body gudang tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.CreateWarehouse(c.Request.Context(),
		inventory.Actor{ID: actor.ID, Role: actor.Role}, inventory.WarehouseInput{
			Code: request.Code, Name: request.Name, SiteLocation: request.SiteLocation,
			IsActive: request.IsActive,
		})
	if err != nil {
		h.writeError(c, err, "create warehouse")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *InventoryHandler) UpdateWarehouse(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request warehousePatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body gudang tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.UpdateWarehouse(c.Request.Context(), id,
		inventory.Actor{ID: actor.ID, Role: actor.Role}, inventory.WarehousePatch{
			Code: request.Code, Name: request.Name, SiteLocation: request.SiteLocation,
			IsActive: request.IsActive,
		})
	if err != nil {
		h.writeError(c, err, "update warehouse")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *InventoryHandler) ListWarehouseLocations(c *gin.Context) {
	warehouseID, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.service.ListLocations(c.Request.Context(), &warehouseID,
		c.Query("q"), parseBool(c.Query("active_only")))
	if err != nil {
		h.writeError(c, err, "list warehouse locations")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *InventoryHandler) ListAllLocations(c *gin.Context) {
	activeOnly := true
	if raw := strings.TrimSpace(c.Query("active_only")); raw != "" {
		activeOnly = parseBool(raw)
	}
	items, err := h.service.ListLocations(c.Request.Context(), nil, c.Query("q"), activeOnly)
	if err != nil {
		h.writeError(c, err, "list locations")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *InventoryHandler) CreateLocation(c *gin.Context) {
	warehouseID, ok := parseID(c)
	if !ok {
		return
	}
	var request locationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body lokasi tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.CreateLocation(c.Request.Context(),
		inventory.Actor{ID: actor.ID, Role: actor.Role}, inventory.LocationInput{
			WarehouseID: warehouseID, Code: request.Code, Name: request.Name,
			LocationType: request.LocationType, IsActive: request.IsActive,
		})
	if err != nil {
		h.writeError(c, err, "create warehouse location")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *InventoryHandler) UpdateLocation(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request locationPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body lokasi tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.UpdateLocation(c.Request.Context(), id,
		inventory.Actor{ID: actor.ID, Role: actor.Role}, inventory.LocationPatch{
			Code: request.Code, Name: request.Name, LocationType: request.LocationType,
			IsActive: request.IsActive,
		})
	if err != nil {
		h.writeError(c, err, "update warehouse location")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *InventoryHandler) ListPolicies(c *gin.Context) {
	page, limit := parsePagination(c)
	sparepartID, err := parseOptionalQueryID(c.Query("sparepart_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter sparepart_id tidak valid", nil)
		return
	}
	locationID, err := parseOptionalQueryID(c.Query("warehouse_location_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter warehouse_location_id tidak valid", nil)
		return
	}
	items, total, err := h.service.ListPolicies(c.Request.Context(), inventory.PolicyFilter{
		Query: c.Query("q"), SparepartID: sparepartID, LocationID: locationID,
		ActiveOnly: parseBool(c.Query("active_only")), Page: page, Limit: limit,
	})
	if err != nil {
		h.writeError(c, err, "list stock policies")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *InventoryHandler) UpsertPolicy(c *gin.Context) {
	var request stockPolicyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body stock policy tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.UpsertPolicy(c.Request.Context(),
		inventory.Actor{ID: actor.ID, Role: actor.Role}, inventory.PolicyInput{
			SparepartID: request.SparepartID, LocationID: request.LocationID,
			MinQty: request.MinQty, CriticalQty: request.CriticalQty,
			ReorderQty: request.ReorderQty, IsActive: request.IsActive,
		})
	if err != nil {
		h.writeError(c, err, "upsert stock policy")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *InventoryHandler) UpdatePolicy(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request stockPolicyPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body stock policy tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.UpdatePolicy(c.Request.Context(), id,
		inventory.Actor{ID: actor.ID, Role: actor.Role}, inventory.PolicyPatch{
			MinQty: request.MinQty, CriticalQty: request.CriticalQty,
			ReorderQty: request.ReorderQty, ClearReorder: request.ClearReorder,
			IsActive: request.IsActive,
		})
	if err != nil {
		h.writeError(c, err, "update stock policy")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *InventoryHandler) ListBalances(c *gin.Context) {
	page, limit := parsePagination(c)
	warehouseID, err := parseOptionalQueryID(c.Query("warehouse_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter warehouse_id tidak valid", nil)
		return
	}
	locationID, err := parseOptionalQueryID(c.Query("warehouse_location_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter warehouse_location_id tidak valid", nil)
		return
	}
	items, total, err := h.service.ListBalances(c.Request.Context(), inventory.BalanceFilter{
		Query: c.Query("q"), WarehouseID: warehouseID, LocationID: locationID,
		Status: strings.TrimSpace(c.Query("status")), Page: page, Limit: limit,
		Sort: c.Query("sort"), Order: c.Query("order"),
	})
	if err != nil {
		h.writeError(c, err, "list stock balances")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *InventoryHandler) ListReceipts(c *gin.Context)    { h.listTransactions(c, "receipt") }
func (h *InventoryHandler) ListIssues(c *gin.Context)      { h.listTransactions(c, "issue") }
func (h *InventoryHandler) ListReturns(c *gin.Context)     { h.listTransactions(c, "return") }
func (h *InventoryHandler) ListTransfers(c *gin.Context)   { h.listTransactions(c, "transfer") }
func (h *InventoryHandler) ListAdjustments(c *gin.Context) { h.listTransactions(c, "adjustment") }

func (h *InventoryHandler) listTransactions(c *gin.Context, transactionType string) {
	page, limit := parsePagination(c)
	from, err := parseOptionalDate(c.Query("from"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter tanggal awal tidak valid", nil)
		return
	}
	to, err := parseOptionalDate(c.Query("to"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter tanggal akhir tidak valid", nil)
		return
	}
	items, total, err := h.service.ListTransactions(c.Request.Context(), inventory.TransactionFilter{
		Type: transactionType, Query: c.Query("q"), From: from, To: to,
		Page: page, Limit: limit,
	})
	if err != nil {
		h.writeError(c, err, "list inventory transactions")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *InventoryHandler) GetTransaction(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.GetTransaction(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "get inventory transaction")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *InventoryHandler) PostReceipt(c *gin.Context)    { h.postTransaction(c, "receipt") }
func (h *InventoryHandler) PostIssue(c *gin.Context)      { h.postTransaction(c, "issue") }
func (h *InventoryHandler) PostReturn(c *gin.Context)     { h.postTransaction(c, "return") }
func (h *InventoryHandler) PostTransfer(c *gin.Context)   { h.postTransaction(c, "transfer") }
func (h *InventoryHandler) PostAdjustment(c *gin.Context) { h.postTransaction(c, "adjustment") }

func (h *InventoryHandler) postTransaction(c *gin.Context, transactionType string) {
	var request inventoryPostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body transaksi inventory tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	items := make([]inventory.PostItem, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, inventory.PostItem{
			SparepartID: item.SparepartID, FromLocationID: item.FromLocationID,
			ToLocationID: item.ToLocationID, Qty: item.Qty, Notes: item.Notes,
		})
	}
	result, err := h.service.Post(c.Request.Context(),
		inventory.Actor{ID: actor.ID, Role: actor.Role}, inventory.PostInput{
			Type: transactionType, ReferenceType: request.ReferenceType,
			ReferenceID: request.ReferenceID, ReferenceNo: request.ReferenceNo,
			Notes: request.Notes, IdempotencyKey: request.IdempotencyKey, Items: items,
		})
	if err != nil {
		h.writeError(c, err, "post inventory transaction")
		return
	}
	sparepartIDs := make([]int64, 0, len(items))
	for _, item := range items {
		sparepartIDs = append(sparepartIDs, item.SparepartID)
	}
	h.recomputeMaterialReadiness(c.Request.Context(), sparepartIDs)
	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *InventoryHandler) Reverse(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request reverseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body reversal tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.Reverse(c.Request.Context(), id,
		inventory.Actor{ID: actor.ID, Role: actor.Role}, request.Reason)
	if err != nil {
		h.writeError(c, err, "reverse inventory transaction")
		return
	}
	sparepartIDs := make([]int64, 0, len(item.Items))
	for _, line := range item.Items {
		sparepartIDs = append(sparepartIDs, line.SparepartID)
	}
	h.recomputeMaterialReadiness(c.Request.Context(), sparepartIDs)
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *InventoryHandler) ListMovements(c *gin.Context) {
	page, limit := parsePagination(c)
	sparepartID, err := parseOptionalQueryID(c.Query("sparepart_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter sparepart_id tidak valid", nil)
		return
	}
	locationID, err := parseOptionalQueryID(c.Query("warehouse_location_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter warehouse_location_id tidak valid", nil)
		return
	}
	from, err := parseOptionalDate(c.Query("from"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter tanggal awal tidak valid", nil)
		return
	}
	to, err := parseOptionalDate(c.Query("to"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter tanggal akhir tidak valid", nil)
		return
	}
	items, total, err := h.service.ListMovements(c.Request.Context(), inventory.MovementFilter{
		Query: c.Query("q"), Type: strings.TrimSpace(c.Query("type")),
		SparepartID: sparepartID, LocationID: locationID, From: from, To: to,
		Page: page, Limit: limit,
	})
	if err != nil {
		h.writeError(c, err, "list inventory movements")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *InventoryHandler) Reconcile(c *gin.Context) {
	item, err := h.service.Reconcile(c.Request.Context())
	if err != nil {
		h.writeError(c, err, "reconcile inventory")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *InventoryHandler) ListOpnameSessions(c *gin.Context) {
	page, limit := parsePagination(c)
	warehouseID, err := parseOptionalQueryID(c.Query("warehouse_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter warehouse_id tidak valid", nil)
		return
	}
	items, total, err := h.service.ListOpnameSessions(c.Request.Context(), inventory.OpnameFilter{
		Status: strings.TrimSpace(c.Query("status")), WarehouseID: warehouseID,
		Page: page, Limit: limit,
	})
	if err != nil {
		h.writeError(c, err, "list stock opname sessions")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *InventoryHandler) CreateOpnameSession(c *gin.Context) {
	var request opnameCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body stock opname tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.CreateOpname(c.Request.Context(),
		inventory.Actor{ID: actor.ID, Role: actor.Role}, inventory.OpnameCreateInput{
			WarehouseID: request.WarehouseID, Notes: request.Notes,
		})
	if err != nil {
		h.writeError(c, err, "create stock opname session")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *InventoryHandler) GetOpnameSession(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.service.GetOpnameSession(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err, "get stock opname session")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *InventoryHandler) CountOpnameSession(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request opnameCountRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body hitung opname tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	items := make([]inventory.OpnameCountItem, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, inventory.OpnameCountItem{
			ItemID: item.ItemID, CountedQty: item.CountedQty, Notes: item.Notes,
		})
	}
	result, err := h.service.CountOpname(c.Request.Context(), id,
		inventory.Actor{ID: actor.ID, Role: actor.Role}, inventory.OpnameCountInput{Items: items})
	if err != nil {
		h.writeError(c, err, "count stock opname session")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *InventoryHandler) PostOpnameSession(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.PostOpname(c.Request.Context(), id,
		inventory.Actor{ID: actor.ID, Role: actor.Role})
	if err != nil {
		h.writeError(c, err, "post stock opname session")
		return
	}
	sparepartIDs := make([]int64, 0, len(item.Items))
	for _, line := range item.Items {
		sparepartIDs = append(sparepartIDs, line.SparepartID)
	}
	h.recomputeMaterialReadiness(c.Request.Context(), sparepartIDs)
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *InventoryHandler) CancelOpnameSession(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request cancelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body pembatalan tidak valid", nil)
		return
	}
	actor, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return
	}
	item, err := h.service.CancelOpname(c.Request.Context(), id,
		inventory.Actor{ID: actor.ID, Role: actor.Role}, request.Reason)
	if err != nil {
		h.writeError(c, err, "cancel stock opname session")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *InventoryHandler) writeError(c *gin.Context, err error, operation string) {
	var validation *inventory.ValidationError
	switch {
	case errors.As(err, &validation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data tidak valid", validation.Fields)
	case errors.Is(err, inventory.ErrValidation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data tidak valid", nil)
	case errors.Is(err, inventory.ErrInsufficientStock):
		writeUnitError(c, http.StatusConflict, "insufficient_stock", "Stok tersedia tidak mencukupi.", nil)
	case errors.Is(err, inventory.ErrInvalidState):
		writeUnitError(c, http.StatusConflict, "invalid_state", "Status transaksi tidak memungkinkan aksi ini.", nil)
	case errors.Is(err, inventory.ErrNotFound):
		writeUnitError(c, http.StatusNotFound, "not_found", "Data tidak ditemukan", nil)
	case errors.Is(err, inventory.ErrConflict):
		writeUnitError(c, http.StatusConflict, "conflict", "Data sudah ada atau melanggar constraint", nil)
	default:
		h.logger.Error("inventory operation failed", "operation", operation, "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
	}
}
