package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"learnos/internal/model"
	"learnos/internal/service"
)

func (h *Handler) SetMedia(media *service.MediaService) { h.media = media }

func (h *Handler) mediaUpsert(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	var input []struct {
		Source, SourceID, ExternalID, Type, Title, Description string
		Authors, Genres, Tags                                  []string
		CoverID                                                string `json:"cover_id"`
		Year                                                   int
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
	if c.ShouldBindJSON(&input) != nil {
		mediaError(c, 400, "媒体数据无效")
		return
	}
	items := make([]model.MediaItem, len(input))
	for i, v := range input {
		if v.CoverID != "" {
			if h.images == nil {
				mediaError(c, 400, "封面不可用")
				return
			}
			if _, err := h.images.Metadata(c.Request.Context(), user, v.CoverID); err != nil {
				mediaError(c, 400, "封面不可用")
				return
			}
		}
		items[i] = model.MediaItem{CoverID: v.CoverID, Source: v.Source, SourceID: v.SourceID, ExternalID: v.ExternalID, Type: v.Type, Title: v.Title, Description: v.Description, Authors: v.Authors, Genres: v.Genres, Tags: v.Tags, Year: v.Year}
	}
	err := h.media.Upsert(c.Request.Context(), user, items)
	if errors.Is(err, service.ErrMediaInput) {
		mediaError(c, 400, "媒体数据无效")
		return
	}
	if err != nil {
		mediaError(c, 500, "媒体保存失败")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"accepted": len(items)}})
}

func (h *Handler) mediaList(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	search := strings.TrimSpace(c.Query("q"))
	if err != nil || page < 1 || page > 100000 || len(search) > 256 {
		mediaError(c, 400, "搜索参数无效")
		return
	}
	items, total, err := h.media.List(c.Request.Context(), user, c.Query("type"), c.Query("source"), search, page)
	if err != nil {
		mediaError(c, 500, "媒体读取失败")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"items": items, "total": total}})
}

func mediaError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": "MEDIA_ERROR", "message": message, "retryable": status >= 500}})
}
