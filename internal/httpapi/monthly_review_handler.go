package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"learnos/internal/repository"
	"learnos/internal/service"
	"net/http"
)

func reviewData(c *gin.Context, data interface{}, err error) {
	if err == nil {
		c.JSON(200, gin.H{"data": data})
		return
	}
	status, code, msg := 500, "REVIEW_FAILED", "月度回顾暂时无法读取或保存，请重试。"
	switch {
	case errors.Is(err, service.ErrReviewInput):
		status, code, msg = 400, "REVIEW_INVALID", "请选择有效月份与事实预览，文本最多 10000 字，链接需为有效外部资料链接。"
	case errors.Is(err, repository.ErrReviewConflict):
		status, code, msg = 409, "REVIEW_CHANGED", "回顾已更新或预览已过期，请重新读取；当前输入仍可保留。"
	case errors.Is(err, gorm.ErrRecordNotFound):
		status, code, msg = 404, "REVIEW_NOT_FOUND", "回顾或事实版本不存在，或不属于当前用户。"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": msg, "retryable": status >= 500}})
}
func (h *Handler) MonthlyReviews(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	rows, err := h.reviews.List(c.Request.Context(), user)
	reviewData(c, rows, err)
}
func (h *Handler) MonthlyReview(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	doc, err := h.reviews.Get(c.Request.Context(), user, c.Param("month"))
	reviewData(c, doc, err)
}
func (h *Handler) PreviewMonthlyReview(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	doc, err := h.reviews.Generate(c.Request.Context(), user, c.Param("month"))
	reviewData(c, doc, err)
}
func (h *Handler) MonthlyReviewHistory(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	id, ok := projectPathID(c, "snapshot")
	if !ok {
		return
	}
	doc, err := h.reviews.HistorySnapshot(c.Request.Context(), user, c.Param("month"), id)
	reviewData(c, doc, err)
}
func (h *Handler) SaveMonthlyReview(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 150000)
	var in service.ReviewSaveInput
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil || d.Decode(new(interface{})) != io.EOF {
		reviewData(c, nil, service.ErrReviewInput)
		return
	}
	doc, err := h.reviews.Save(c.Request.Context(), user, c.Param("month"), in)
	reviewData(c, doc, err)
}
