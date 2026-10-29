package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"learnos/internal/repository"
	"learnos/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetToolState returns the signed-in user's saved document for a tool.
func (h *Handler) GetToolState(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	if h.toolStates == nil {
		toolStateError(c, errors.New("tool state service unavailable"))
		return
	}
	view, err := h.toolStates.Get(c.Request.Context(), userID, c.Param("key"))
	if err != nil {
		toolStateError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

// SaveToolState writes the document with an optimistic revision check.
func (h *Handler) SaveToolState(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	if h.toolStates == nil {
		toolStateError(c, errors.New("tool state service unavailable"))
		return
	}
	var input struct {
		State    json.RawMessage `json:"state"`
		Revision int64           `json:"revision"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "TOOL_STATE_INVALID", "message": "规划数据格式不正确。", "retryable": false}})
		return
	}
	view, err := h.toolStates.Save(c.Request.Context(), userID, c.Param("key"), input.State, input.Revision)
	if err != nil {
		toolStateError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func toolStateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUnknownTool):
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "TOOL_NOT_FOUND", "message": "没有这个工具。", "retryable": false}})
	case errors.Is(err, service.ErrInvalidToolState):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "TOOL_STATE_INVALID", "message": "规划数据格式不正确或超出大小限制。", "retryable": false}})
	case errors.Is(err, repository.ErrToolStateNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "TOOL_STATE_NOT_FOUND", "message": "账号里还没有保存过这个工具的规划。", "retryable": false}})
	case errors.Is(err, repository.ErrToolStateConflict):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "TOOL_STATE_CONFLICT", "message": "这份规划已在别处更新，请重新载入后再修改。", "retryable": false}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "TOOL_STATE_FAILED", "message": "保存失败，请稍后重试。", "retryable": true}})
	}
}
