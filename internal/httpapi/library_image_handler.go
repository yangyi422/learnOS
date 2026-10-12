package httpapi

import (
	"github.com/gin-gonic/gin"
	"io"
	"learnos/internal/librarymedia"
	"learnos/internal/service"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Multipart uploads cannot rely on JSON's cross-origin preflight protection.
// Reject browser cross-origin forms; same-site cookies alone do not cover subdomains.
func imageSameOrigin(c *gin.Context) bool {
	if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
		c.AbortWithStatusJSON(403, gin.H{"error": "cross-origin upload rejected"})
		return false
	}
	if origin := c.GetHeader("Origin"); origin != "" {
		u, e := url.Parse(origin)
		if e != nil || (u.Scheme != "http" && u.Scheme != "https") || !strings.EqualFold(u.Host, c.Request.Host) {
			c.AbortWithStatusJSON(403, gin.H{"error": "cross-origin upload rejected"})
			return false
		}
	}
	return true
}
func (h *Handler) UploadLibraryImage(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok || !imageSameOrigin(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, librarymedia.MaxBytes+(64<<10))
	// Stream multipart rather than leaving oversized temporary multipart files.
	reader, err := c.Request.MultipartReader()
	if err != nil {
		libraryData(c, nil, service.ErrLibraryInput)
		return
	}
	var fileName string
	var bytesReader io.Reader
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "image" || part.FileName() == "" {
		libraryData(c, nil, service.ErrLibraryInput)
		return
	}
	defer part.Close()
	fileName = part.FileName()
	bytesReader = part
	row, err := h.images.Upload(c.Request.Context(), user, fileName, bytesReader)
	if err != nil {
		libraryData(c, nil, err)
		return
	}
	// Upload request deliberately supports exactly one image. The bounded first
	// part is complete; no client filename or remaining parts are persisted.
	libraryData(c, row, nil)
}
func (h *Handler) ReadLibraryImage(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok {
		return
	}
	variant := c.Query("variant")
	if variant != "" && variant != service.LibraryCardVariant {
		libraryData(c, nil, service.ErrLibraryInput)
		return
	}
	row, err := h.images.Metadata(c.Request.Context(), user, c.Param("image"))
	if err != nil {
		libraryData(c, nil, err)
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	// Private storage is permitted, but every reuse must authenticate/revalidate.
	c.Header("Cache-Control", "private, no-cache")
	c.Header("Vary", "Cookie, Authorization")
	etag := `"` + row.SHA256 + `-` + variant + `"`
	c.Header("ETag", etag)
	c.Header("Cross-Origin-Resource-Policy", "same-origin")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Header("Content-Disposition", "inline")
	for _, candidate := range strings.Split(c.GetHeader("If-None-Match"), ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == etag {
			c.Status(http.StatusNotModified)
			return
		}
	}
	data, err := h.images.ResponseBytes(c.Request.Context(), row, variant)
	if err != nil {
		c.Header("Cache-Control", "private, no-store")
		c.Header("ETag", "")
		libraryData(c, nil, err)
		return
	}
	mime := row.MIME
	if variant != "" {
		mime = "image/jpeg"
	}
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(200, mime, data)
}
func (h *Handler) CleanupLibraryImages(c *gin.Context) {
	user, ok := projectUserID(c)
	if !ok || !imageSameOrigin(c) {
		return
	}
	var in struct {
		Confirm bool `json:"confirm"`
	}
	if !libraryJSON(c, &in) {
		return
	}
	if !in.Confirm {
		libraryData(c, nil, service.ErrLibraryInput)
		return
	}
	count, err := h.images.Cleanup(c.Request.Context(), user)
	libraryData(c, gin.H{"removed": count}, err)
}

func (h *Handler) DownloadBackup(c *gin.Context) {
	rows, err := h.backups.List(c.Request.Context())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	for _, row := range rows {
		if row.Name == c.Param("name") {
			mime := "application/octet-stream"
			if strings.HasSuffix(row.Name, ".zip") {
				mime = "application/zip"
			}
			c.Header("Content-Type", mime)
			c.Header("Cache-Control", "private, no-store")
			c.Header("X-Content-Type-Options", "nosniff")
			c.FileAttachment(row.Path, row.Name)
			return
		}
	}
	c.JSON(404, gin.H{"error": "backup not found"})
}
