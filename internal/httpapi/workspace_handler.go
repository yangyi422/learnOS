package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"learnos/internal/repository"
	"learnos/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) GetToday(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	projectID, ok := optionalProjectID(c)
	if !ok {
		return
	}
	date := c.Query("date")
	if _, err := time.Parse("2006-01-02", date); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "TODAY_DATE_INVALID", "message": "日期格式无效。", "retryable": false}})
		return
	}
	today, err := h.projects.Today(c.Request.Context(), userID, date, projectID)
	if err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": today})
}

func optionalProjectID(c *gin.Context) (*uint, bool) {
	raw := c.Query("project_id")
	if raw == "" || raw == "all" {
		return nil, true
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return nil, false
	}
	id := uint(value)
	return &id, true
}

func (h *Handler) GetWorkspaceHome(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	view, err := h.workspace.Home(c.Request.Context(), userID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "WORKSPACE_UNAVAILABLE", "message": "工作区暂时无法加载。", "retryable": true}})
		return
	}
	if len(view.Errors) == 4 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"data": view, "error": gin.H{"code": "WORKSPACE_SECTIONS_UNAVAILABLE", "message": "工作区内容暂时无法加载，请重试。", "retryable": true}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *Handler) ListInbox(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	view, err := h.inbox.List(c.Request.Context(), userID, c.Query("status"))
	if err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *Handler) CreateInboxItem(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	var input struct {
		Content    string `json:"content"`
		CaptureKey string `json:"capture_key"`
	}
	if c.ShouldBindJSON(&input) != nil {
		inboxError(c, service.ErrInvalidInboxInput)
		return
	}
	item, err := h.inbox.CreateWithKey(c.Request.Context(), userID, input.Content, input.CaptureKey)
	if err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *Handler) UpdateInboxItem(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	itemID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	var input struct {
		Content string `json:"content"`
	}
	if c.ShouldBindJSON(&input) != nil {
		inboxError(c, service.ErrInvalidInboxInput)
		return
	}
	item, err := h.inbox.Update(c.Request.Context(), userID, itemID, input.Content)
	if err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *Handler) ConvertInboxItemToTask(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	itemID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	var input service.InboxConvertInput
	if c.ShouldBindJSON(&input) != nil {
		inboxError(c, service.ErrInvalidInboxInput)
		return
	}
	item, task, err := h.inbox.ConvertToTask(c.Request.Context(), userID, itemID, input)
	if err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"item": item, "task": task}})
}

func (h *Handler) ArchiveInboxItem(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	itemID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	if err := h.inbox.Archive(c.Request.Context(), userID, itemID); err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"archived": true}})
}

func (h *Handler) DeleteInboxItem(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	itemID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	if err := h.inbox.Delete(c.Request.Context(), userID, itemID); err != nil {
		inboxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": true}})
}

func inboxError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidRecordInput), errors.Is(err, service.ErrInvalidInboxInput), errors.Is(err, service.ErrInvalidProjectInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INBOX_INPUT_INVALID", "message": "请检查内容、项目或外部链接格式。", "retryable": false}})
	case errors.Is(err, repository.ErrInboxItemNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "INBOX_ITEM_NOT_FOUND", "message": "内容、记录或所属项目不存在。", "retryable": false}})
	case errors.Is(err, repository.ErrInboxCaptureConflict):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "INBOX_CAPTURE_CONFLICT", "message": "该请求已经保存过其他内容，请刷新后编辑原记录。", "retryable": false}})
	case errors.Is(err, repository.ErrInboxAlreadyProcessed):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "INBOX_ITEM_ALREADY_PROCESSED", "message": "这条内容已经整理过，请刷新查看。", "retryable": false}})
	case errors.Is(err, repository.ErrInboxAlreadyArchived):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "INBOX_ITEM_ARCHIVED", "message": "已归档的内容不能执行此操作。", "retryable": false}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INBOX_OPERATION_FAILED", "message": "操作失败，请稍后重试。", "retryable": true}})
	}
}
