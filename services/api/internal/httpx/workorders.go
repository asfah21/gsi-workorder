package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/workorders"
)

const maxAttachmentBytes = 5 << 20

type WorkOrderService interface {
	List(ctx context.Context, filter workorders.Filter) ([]workorders.WorkOrder, int, error)
	Get(ctx context.Context, id int64, actor workorders.Actor) (workorders.Detail, error)
	CreateManual(ctx context.Context, actor workorders.Actor, input workorders.CreateManualInput) (workorders.WorkOrder, error)
	Update(ctx context.Context, id int64, actor workorders.Actor, input workorders.PatchInput) (workorders.WorkOrder, error)
	AddUpdate(ctx context.Context, id int64, actor workorders.Actor, note string) (workorders.UpdateRow, error)
	AddAttachment(ctx context.Context, id int64, actor workorders.Actor, filePath, mime string, size int) (workorders.Attachment, error)
	GetAttachment(ctx context.Context, id, attachmentID int64, actor workorders.Actor) (workorders.Attachment, error)
}

type WorkOrderHandler struct {
	service   WorkOrderService
	logger    *slog.Logger
	uploadDir string
}

func NewWorkOrderHandler(service WorkOrderService, logger *slog.Logger, uploadDir string) *WorkOrderHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WorkOrderHandler{service: service, logger: logger, uploadDir: uploadDir}
}

type createWorkOrderRequest struct {
	UnitID          int64  `json:"unit_id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	Priority        string `json:"priority"`
	MaintenanceType string `json:"maintenance_type"`
	AssigneeID      *int64 `json:"assignee_id"`
	SupervisorID    *int64 `json:"supervisor_id"`
	DueDate         string `json:"due_date"`
}

type updateWorkOrderRequest struct {
	Status          string  `json:"status"`
	Note            string  `json:"note"`
	Reason          string  `json:"reason"`
	Priority        *string `json:"priority"`
	MaintenanceType *string `json:"maintenance_type"`
	AssigneeID      *int64  `json:"assignee_id"`
	SupervisorID    *int64  `json:"supervisor_id"`
	Title           *string `json:"title"`
	Description     *string `json:"description"`
}

type workOrderNoteRequest struct {
	Note string `json:"note"`
}

func (h *WorkOrderHandler) List(c *gin.Context) {
	actor, ok := workOrderActor(c)
	if !ok {
		return
	}
	page, limit := parsePagination(c)
	unitID, err := parseOptionalQueryID(c.Query("unit_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter unit_id tidak valid", nil)
		return
	}
	assigneeID, err := parseOptionalQueryID(c.Query("assignee_id"))
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_filter", "Filter assignee_id tidak valid", nil)
		return
	}
	items, total, err := h.service.List(c.Request.Context(), workorders.Filter{
		Status:     strings.TrimSpace(c.Query("status")),
		Source:     strings.TrimSpace(c.Query("source")),
		Query:      strings.TrimSpace(c.Query("q")),
		UnitID:     unitID,
		AssigneeID: assigneeID,
		Page:       page,
		Limit:      limit,
		Sort:       c.Query("sort"),
		Order:      c.Query("order"),
		Actor:      actor,
	})
	if err != nil {
		h.writeError(c, err, "list work orders")
		return
	}
	writeList(c, items, page, limit, total)
}

func (h *WorkOrderHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := workOrderActor(c)
	if !ok {
		return
	}
	detail, err := h.service.Get(c.Request.Context(), id, actor)
	if err != nil {
		h.writeError(c, err, "get work order")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": detail})
}

func (h *WorkOrderHandler) Create(c *gin.Context) {
	actor, ok := workOrderActor(c)
	if !ok {
		return
	}
	var request createWorkOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body work order tidak valid", nil)
		return
	}
	dueDate, err := parseDueDate(request.DueDate)
	if err != nil {
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data work order tidak valid", map[string]string{"due_date": "Format tanggal harus YYYY-MM-DD."})
		return
	}
	item, err := h.service.CreateManual(c.Request.Context(), actor, workorders.CreateManualInput{
		UnitID:          request.UnitID,
		Title:           request.Title,
		Description:     request.Description,
		Priority:        request.Priority,
		MaintenanceType: request.MaintenanceType,
		AssigneeID:      request.AssigneeID,
		SupervisorID:    request.SupervisorID,
		DueDate:         dueDate,
	})
	if err != nil {
		h.writeError(c, err, "create work order")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *WorkOrderHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := workOrderActor(c)
	if !ok {
		return
	}
	var request updateWorkOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body work order tidak valid", nil)
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, actor, workorders.PatchInput{
		Status:          strings.TrimSpace(request.Status),
		Note:            request.Note,
		Reason:          request.Reason,
		Priority:        request.Priority,
		MaintenanceType: request.MaintenanceType,
		AssigneeID:      request.AssigneeID,
		SupervisorID:    request.SupervisorID,
		Title:           request.Title,
		Description:     request.Description,
	})
	if err != nil {
		h.writeError(c, err, "update work order")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *WorkOrderHandler) AddUpdate(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := workOrderActor(c)
	if !ok {
		return
	}
	var request workOrderNoteRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Body catatan tidak valid", nil)
		return
	}
	item, err := h.service.AddUpdate(c.Request.Context(), id, actor, request.Note)
	if err != nil {
		h.writeError(c, err, "add work-order note")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *WorkOrderHandler) UploadAttachment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := workOrderActor(c)
	if !ok {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAttachmentBytes+(1<<20))
	header, err := c.FormFile("file")
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "File lampiran wajib diisi", nil)
		return
	}
	if header.Size <= 0 || header.Size > maxAttachmentBytes {
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Ukuran lampiran maksimal 5 MB", map[string]string{"file": "Maksimal 5 MB."})
		return
	}

	source, err := header.Open()
	if err != nil {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Lampiran tidak dapat dibaca", nil)
		return
	}
	defer source.Close()

	sniff := make([]byte, 512)
	n, err := io.ReadFull(source, sniff)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		writeUnitError(c, http.StatusBadRequest, "invalid_request", "Lampiran tidak dapat dibaca", nil)
		return
	}
	mime := http.DetectContentType(sniff[:n])
	extension := ""
	switch mime {
	case "image/jpeg":
		extension = ".jpg"
	case "image/png":
		extension = ".png"
	default:
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Lampiran harus JPG atau PNG", map[string]string{"file": "Hanya JPG/PNG."})
		return
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Lampiran tidak dapat diproses", nil)
		return
	}

	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Lampiran tidak dapat diproses", nil)
		return
	}
	relativePath := filepath.Join("workorders", strconv.FormatInt(id, 10), hex.EncodeToString(random)+extension)
	absolutePath := filepath.Join(h.uploadDir, relativePath)
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
		h.logger.Error("create attachment directory failed", "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Lampiran tidak dapat disimpan", nil)
		return
	}

	target, err := os.OpenFile(absolutePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		h.logger.Error("create attachment file failed", "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Lampiran tidak dapat disimpan", nil)
		return
	}
	written, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil || closeErr != nil || written != header.Size {
		_ = os.Remove(absolutePath)
		h.logger.Error("write attachment file failed", "copy_error", copyErr, "close_error", closeErr, "written", written)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Lampiran tidak dapat disimpan", nil)
		return
	}

	item, err := h.service.AddAttachment(c.Request.Context(), id, actor, relativePath, mime, int(written))
	if err != nil {
		_ = os.Remove(absolutePath)
		h.writeError(c, err, "upload work-order attachment")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *WorkOrderHandler) DownloadAttachment(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	attachmentID, err := strconv.ParseInt(c.Param("attachment_id"), 10, 64)
	if err != nil || attachmentID <= 0 {
		writeUnitError(c, http.StatusBadRequest, "invalid_id", "ID lampiran tidak valid", nil)
		return
	}
	actor, ok := workOrderActor(c)
	if !ok {
		return
	}
	attachment, err := h.service.GetAttachment(c.Request.Context(), id, attachmentID, actor)
	if err != nil {
		h.writeError(c, err, "download work-order attachment")
		return
	}
	path := filepath.Join(h.uploadDir, attachment.FilePath)
	cleanBase := filepath.Clean(h.uploadDir) + string(os.PathSeparator)
	if !strings.HasPrefix(filepath.Clean(path), cleanBase) {
		h.writeError(c, workorders.ErrForbidden, "download work-order attachment")
		return
	}
	if attachment.Mime != "" {
		c.Header("Content-Type", attachment.Mime)
	}
	c.File(path)
}

func (h *WorkOrderHandler) writeError(c *gin.Context, err error, operation string) {
	var validation *workorders.ValidationError
	switch {
	case errors.As(err, &validation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data work order tidak valid", validation.Fields)
	case errors.Is(err, workorders.ErrNotFound):
		writeUnitError(c, http.StatusNotFound, "work_order_not_found", "Work order tidak ditemukan", nil)
	case errors.Is(err, workorders.ErrForbidden):
		writeUnitError(c, http.StatusForbidden, "forbidden", "Tidak berwenang mengubah work order ini", nil)
	case errors.Is(err, workorders.ErrConflict):
		writeUnitError(c, http.StatusConflict, "invalid_transition", "Transisi status work order tidak valid", nil)
	case errors.Is(err, workorders.ErrValidation):
		writeUnitError(c, http.StatusUnprocessableEntity, "validation_error", "Data work order tidak valid", nil)
	default:
		h.logger.Error("work-order operation failed", "operation", operation, "error", err)
		writeUnitError(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan internal", nil)
	}
}

func workOrderActor(c *gin.Context) (workorders.Actor, bool) {
	user, ok := currentUser(c)
	if !ok {
		writeUnitError(c, http.StatusUnauthorized, "unauthorized", "Sesi tidak valid", nil)
		return workorders.Actor{}, false
	}
	return workorders.Actor{ID: user.ID, Role: user.Role}, true
}

func parseOptionalQueryID(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, fmt.Errorf("invalid id")
	}
	return &value, nil
}

func parseDueDate(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
