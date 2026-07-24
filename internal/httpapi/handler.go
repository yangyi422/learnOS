package httpapi

import (
	"net/http"

	"learnos/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	courses *service.CourseService
}

func NewHandler(courses *service.CourseService) *Handler {
	return &Handler{courses: courses}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (h *Handler) ListCourses(c *gin.Context) {
	courses, err := h.courses.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list courses",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": courses,
	})
}
