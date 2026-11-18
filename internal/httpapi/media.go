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
		Library                                                string `json:"library"`
		Source, SourceID, ExternalID, Type, Title, Description string
		Authors, Genres, Tags                                  []string
		CoverID                                                string `json:"cover_id"`
		Year                                                   int
		Rating                                                 float64
		// Episode fields, ignored for movies, series and books.
		SeriesExternalID, SeasonExternalID, PremiereDate string
		SeasonNumber, EpisodeNumber, RuntimeSeconds      int
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
		items[i] = model.MediaItem{Library: v.Library, CoverID: v.CoverID, Source: v.Source, SourceID: v.SourceID, ExternalID: v.ExternalID, Type: v.Type, Title: v.Title, Description: v.Description, Authors: v.Authors, Genres: v.Genres, Tags: v.Tags, Year: v.Year, Rating: v.Rating,
			SeriesExternalID: v.SeriesExternalID, SeasonExternalID: v.SeasonExternalID, SeasonNumber: v.SeasonNumber, EpisodeNumber: v.EpisodeNumber, PremiereDate: v.PremiereDate, RuntimeSeconds: v.RuntimeSeconds}
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
	items, total, err := h.media.List(c.Request.Context(), user, c.Query("type"), c.Query("source"), search, page, c.Query("show_r18") == "true")
	if err != nil {
		mediaError(c, 500, "媒体读取失败")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"items": items, "total": total}})
}

func mediaError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": "MEDIA_ERROR", "message": message, "retryable": status >= 500}})
}

func (h *Handler) mediaBatchDelete(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	var input struct {
		IDs []uint `json:"ids"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<10)
	if c.ShouldBindJSON(&input) != nil {
		mediaError(c, http.StatusBadRequest, "请选择有效的电影记录（每次 1–100 项）")
		return
	}
	err := h.media.DeleteMovies(c.Request.Context(), user, input.IDs)
	switch {
	case errors.Is(err, service.ErrMediaInput):
		mediaError(c, http.StatusBadRequest, "请选择有效且不重复的电影记录（每次 1–100 项）")
	case errors.Is(err, service.ErrMediaNotFound):
		mediaError(c, http.StatusNotFound, "部分电影记录已不存在或不可删除，请刷新后重选。")
	case err != nil:
		mediaError(c, http.StatusInternalServerError, "媒体删除失败，请重试。")
	default:
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": len(input.IDs)}})
	}
}

// mediaSeries serves the TV detail page: the series row plus its season summary.
func (h *Handler) mediaSeries(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	externalID := strings.TrimSpace(c.Param("externalID"))
	if !validMediaExternalID(externalID) {
		mediaError(c, 400, "媒体标识无效")
		return
	}
	detail, err := h.media.Series(c.Request.Context(), user, externalID, c.Query("show_r18") == "true")
	if errors.Is(err, service.ErrMediaNotFound) {
		mediaError(c, 404, "剧集不存在")
		return
	}
	if err != nil {
		mediaError(c, 500, "媒体读取失败")
		return
	}
	c.JSON(200, gin.H{"data": detail})
}

// mediaEpisodes lists episodes of one series; ?season=N narrows it to one season.
func (h *Handler) mediaEpisodes(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	externalID := strings.TrimSpace(c.Param("externalID"))
	if !validMediaExternalID(externalID) {
		mediaError(c, 400, "媒体标识无效")
		return
	}
	season := -1
	if raw := strings.TrimSpace(c.Query("season")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > 999 {
			mediaError(c, 400, "季参数无效")
			return
		}
		season = parsed
	}
	items, err := h.media.Episodes(c.Request.Context(), user, externalID, season, c.Query("show_r18") == "true")
	if errors.Is(err, service.ErrMediaNotFound) {
		mediaError(c, 404, "剧集不存在")
		return
	}
	if err != nil {
		mediaError(c, 500, "媒体读取失败")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"items": items, "total": len(items)}})
}

// validMediaExternalID accepts Jellyfin ids only: 32 hex characters, optionally dashed.
func validMediaExternalID(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F' || r == '-') {
			return false
		}
	}
	return true
}
