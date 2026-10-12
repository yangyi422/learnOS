package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"learnos/internal/librarymedia"
	"learnos/internal/repository"
	"learnos/internal/service"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func libraryData(c *gin.Context, data interface{}, err error) {
	if err == nil {
		c.JSON(200, gin.H{"data": data})
		return
	}
	var maxBytes *http.MaxBytesError
	status, code, message := 500, "LIBRARY_FAILED", "资料暂时无法读取或保存，请重试。"
	switch {
	case errors.Is(err, service.ErrImageMissing):
		status, code, message = 409, "LIBRARY_IMAGE_MISSING", "本地图片不存在或不可读取，请检查图片引用；本次操作未完成。"
	case errors.Is(err, service.ErrImageSize) || errors.As(err, &maxBytes):
		status, code, message = 413, "IMAGE_TOO_LARGE", "单张图片不能超过 10MB。"
	case errors.Is(err, service.ErrImageQuota):
		status, code, message = 409, "IMAGE_QUOTA", "图片存储已达到上限，请备份并清理未使用图片，或调整服务器配额。"
	case errors.Is(err, librarymedia.ErrInvalid):
		status, code, message = 400, "IMAGE_INVALID", "仅支持完整的 PNG、JPEG、WebP 图片；最长边 12000px，总像素不超过 2400 万。"
	case errors.Is(err, service.ErrLibraryInput):
		status, code, message = 400, "LIBRARY_INVALID", "请检查名称、正文长度、分组及链接格式。"
	case errors.Is(err, repository.ErrLibraryConflict):
		status, code, message = 409, "LIBRARY_CHANGED", "资料已被更新，请重新读取后再保存；当前输入可以保留。"
	case errors.Is(err, repository.ErrLibrarySectionNotEmpty):
		status, code, message = 409, "SECTION_NOT_EMPTY", "分组仍有页面，请先移动页面，或明确选择移至专题根目录。"
	case errors.Is(err, gorm.ErrRecordNotFound):
		status, code, message = 404, "LIBRARY_NOT_FOUND", "资料已删除或不可访问。"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "retryable": status >= 500}})
}
func libraryJSON(c *gin.Context, in interface{}) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1100000)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if d.Decode(in) != nil || d.Decode(new(interface{})) != io.EOF {
		libraryData(c, nil, service.ErrLibraryInput)
		return false
	}
	return true
}
func libraryIDs(c *gin.Context, sub string) (uint, uint, uint, bool) {
	user, ok := projectUserID(c)
	if !ok {
		return 0, 0, 0, false
	}
	col, ok := projectPathID(c, "collection")
	if !ok {
		return 0, 0, 0, false
	}
	var id uint
	if sub != "" {
		id, ok = projectPathID(c, sub)
	}
	return user, col, id, ok
}
func (h *Handler) LibraryCollections(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	rows, err := h.library.List(c.Request.Context(), user)
	libraryData(c, rows, err)
}
func (h *Handler) LibraryCollection(c *gin.Context) {
	user, col, _, ok := libraryIDs(c, "")
	if !ok {
		return
	}
	row, err := h.library.Detail(c.Request.Context(), user, col)
	libraryData(c, row, err)
}
func (h *Handler) SaveLibraryCollection(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	var id uint
	if c.Param("collection") != "" {
		id, ok = projectPathID(c, "collection")
		if !ok {
			return
		}
	}
	var in service.LibraryCollectionInput
	if !libraryJSON(c, &in) {
		return
	}
	row, err := h.library.SaveCollection(c.Request.Context(), user, id, in)
	libraryData(c, row, err)
}
func (h *Handler) DeleteLibraryCollection(c *gin.Context) {
	user, col, _, ok := libraryIDs(c, "")
	if !ok {
		return
	}
	var in struct {
		Confirm bool `json:"confirm"`
		Version uint `json:"version"`
	}
	if !libraryJSON(c, &in) {
		return
	}
	if !in.Confirm {
		libraryData(c, nil, service.ErrLibraryInput)
		return
	}
	libraryData(c, gin.H{"deleted": true}, h.library.DeleteCollection(c.Request.Context(), user, col, in.Version))
}
func (h *Handler) SaveLibrarySection(c *gin.Context) {
	user, col, _, ok := libraryIDs(c, "")
	if !ok {
		return
	}
	var id uint
	if c.Param("section") != "" {
		id, ok = projectPathID(c, "section")
		if !ok {
			return
		}
	}
	var in struct {
		Name string `json:"name"`
	}
	if !libraryJSON(c, &in) {
		return
	}
	row, err := h.library.SaveSection(c.Request.Context(), user, col, id, in.Name)
	libraryData(c, row, err)
}
func (h *Handler) DeleteLibrarySection(c *gin.Context) {
	user, col, id, ok := libraryIDs(c, "section")
	if !ok {
		return
	}
	var in struct {
		Confirm    bool `json:"confirm"`
		MoveToRoot bool `json:"move_to_root"`
	}
	if !libraryJSON(c, &in) {
		return
	}
	if !in.Confirm {
		libraryData(c, nil, service.ErrLibraryInput)
		return
	}
	libraryData(c, gin.H{"deleted": true}, h.library.DeleteSection(c.Request.Context(), user, col, id, in.MoveToRoot))
}
func (h *Handler) LibraryPages(c *gin.Context) {
	user, col, _, ok := libraryIDs(c, "")
	if !ok {
		return
	}
	page := 1
	var err error
	if c.Query("page") != "" {
		page, err = strconv.Atoi(c.Query("page"))
		if err != nil {
			libraryData(c, nil, service.ErrLibraryInput)
			return
		}
	}
	var section *uint
	root := c.Query("section") == "root"
	if v := c.Query("section"); v != "" && !root {
		id, e := strconv.ParseUint(v, 10, 32)
		if e != nil || id == 0 {
			libraryData(c, nil, service.ErrLibraryInput)
			return
		}
		x := uint(id)
		section = &x
	}
	rows, err := h.library.Pages(c.Request.Context(), user, col, c.Query("q"), section, root, page)
	libraryData(c, rows, err)
}
func (h *Handler) LibraryPage(c *gin.Context) {
	user, col, id, ok := libraryIDs(c, "page")
	if !ok {
		return
	}
	row, err := h.library.Page(c.Request.Context(), user, col, id)
	libraryData(c, row, err)
}
func (h *Handler) SaveLibraryPage(c *gin.Context) {
	user, col, _, ok := libraryIDs(c, "")
	if !ok {
		return
	}
	var id uint
	if c.Param("page") != "" {
		id, ok = projectPathID(c, "page")
		if !ok {
			return
		}
	}
	var in service.LibraryPageInput
	if !libraryJSON(c, &in) {
		return
	}
	row, err := h.library.SavePage(c.Request.Context(), user, col, id, in)
	libraryData(c, row, err)
}
func (h *Handler) DeleteLibraryPage(c *gin.Context) {
	user, col, id, ok := libraryIDs(c, "page")
	if !ok {
		return
	}
	var in struct {
		Confirm bool `json:"confirm"`
		Version uint `json:"version"`
	}
	if !libraryJSON(c, &in) {
		return
	}
	if !in.Confirm {
		libraryData(c, nil, service.ErrLibraryInput)
		return
	}
	libraryData(c, gin.H{"deleted": true}, h.library.DeletePage(c.Request.Context(), user, col, id, in.Version))
}
func (h *Handler) ExportLibrary(c *gin.Context) {
	user, col, _, ok := libraryIDs(c, "")
	if !ok {
		return
	}
	f, err := os.CreateTemp("", "lifeos-library-export-*.zip")
	if err != nil {
		libraryData(c, nil, err)
		return
	}
	defer os.Remove(f.Name())
	err = h.library.ExportTo(c.Request.Context(), user, col, f)
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		libraryData(c, nil, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="lifeos-library-`+strconv.FormatUint(uint64(col), 10)+`.zip"`)
	c.Header("Content-Type", "application/zip")
	c.File(f.Name())
}

func (h *Handler) LibraryLinks(c *gin.Context) {
	user, col, _, ok := libraryIDs(c, "")
	if !ok {
		return
	}
	parts := strings.Split(c.Query("ids"), ",")
	if len(parts) > 100 {
		libraryData(c, nil, service.ErrLibraryInput)
		return
	}
	ids := []uint{}
	for _, part := range parts {
		id, err := strconv.ParseUint(part, 10, 32)
		if err != nil || id == 0 {
			libraryData(c, nil, service.ErrLibraryInput)
			return
		}
		ids = append(ids, uint(id))
	}
	out, err := h.library.LinkStatus(c.Request.Context(), user, col, ids)
	libraryData(c, out, err)
}
