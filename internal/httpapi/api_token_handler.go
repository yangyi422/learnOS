package httpapi

import (
	"errors"
	"net/http"

	"learnos/internal/repository"
	"learnos/internal/service"

	"github.com/gin-gonic/gin"
)

// ListAPITokens returns the signed-in user's capture credentials (never secrets).
func (h *Handler) ListAPITokens(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	if h.apiTokens == nil {
		apiTokenError(c, errors.New("api token service unavailable"))
		return
	}
	tokens, err := h.apiTokens.List(c.Request.Context(), userID)
	if err != nil {
		apiTokenError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tokens})
}

// CreateAPIToken mints a capture-only credential and returns the secret once.
func (h *Handler) CreateAPIToken(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	if h.apiTokens == nil {
		apiTokenError(c, errors.New("api token service unavailable"))
		return
	}
	var input struct {
		Name          string `json:"name"`
		ExpiresInDays int    `json:"expires_in_days"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "TOKEN_INPUT_INVALID", "message": "请检查凭证名称。", "retryable": false}})
		return
	}
	created, err := h.apiTokens.Create(c.Request.Context(), userID, input.Name, input.ExpiresInDays)
	if err != nil {
		apiTokenError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *Handler) RevokeAPIToken(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	if h.apiTokens == nil {
		apiTokenError(c, errors.New("api token service unavailable"))
		return
	}
	tokenID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	if err := h.apiTokens.Revoke(c.Request.Context(), userID, tokenID); err != nil {
		apiTokenError(c, err)
		return
	}
	// Same shape as the other DELETE endpoints, so the shared fetch client can
	// parse the response.
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"revoked": true}})
}

// CaptureInboxItem is the narrow write path for API tokens: it only ever creates
// an inbox item for the credential's own user.
func (h *Handler) CaptureInboxItem(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	if h.inbox == nil {
		inboxError(c, errors.New("inbox service unavailable"))
		return
	}
	var input struct {
		Content    string `json:"content"`
		CaptureKey string `json:"capture_key"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "CAPTURE_INPUT_INVALID", "message": "请提交文本内容。", "retryable": false}})
		return
	}
	item, err := h.inbox.CreateWithKey(c.Request.Context(), userID, input.Content, input.CaptureKey)
	if err != nil {
		captureError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"id":          item.ID,
		"content":     item.Content,
		"status":      item.Status,
		"source_type": item.SourceType,
		"source_url":  item.SourceURL,
		"created_at":  item.CreatedAt,
	}})
}

func apiTokenError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidTokenInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "TOKEN_INPUT_INVALID", "message": "凭证名称或有效期不合法。", "retryable": false}})
	case errors.Is(err, repository.ErrAPITokenNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "TOKEN_NOT_FOUND", "message": "凭证不存在或已撤销。", "retryable": false}})
	case errors.Is(err, service.ErrInvalidAPIToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "认证失败，请检查凭证是否有效。", "retryable": false}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "TOKEN_OPERATION_FAILED", "message": "操作失败，请稍后重试。", "retryable": true}})
	}
}

func captureError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInboxInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "CAPTURE_INPUT_INVALID", "message": "内容为空或超出长度限制。", "retryable": false}})
	case errors.Is(err, repository.ErrInboxCaptureConflict):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CAPTURE_DUPLICATE", "message": "该请求已经保存过其他内容，请重新打开插件。", "retryable": false}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "CAPTURE_FAILED", "message": "保存失败，请稍后重试。", "retryable": true}})
	}
}
