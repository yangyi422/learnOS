package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetConversation(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	lessonID, valid := parseID(c.Param("lessonId"))
	if !ok || !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer"})
		return
	}
	view, err := h.courses.GetConversation(c.Request.Context(), courseID, lessonID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}
func (h *Handler) Converse(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	lessonID, valid := parseID(c.Param("lessonId"))
	var request struct {
		Message string `json:"message"`
		Key     string `json:"idempotency_key"`
	}
	if !ok || !valid || c.ShouldBindJSON(&request) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer"})
		return
	}
	view, err := h.courses.Converse(c.Request.Context(), courseID, lessonID, request.Message, request.Key)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}
func (h *Handler) AdvanceConversation(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	lessonID, valid := parseID(c.Param("lessonId"))
	var request struct {
		Action string `json:"action"`
	}
	if !ok || !valid || c.ShouldBindJSON(&request) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer"})
		return
	}
	current, err := h.courses.AdvanceConversation(c.Request.Context(), courseID, lessonID, request.Action)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.writeCurrentLesson(c, current)
}

func (h *Handler) RetryConversation(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	lessonID, valid := parseID(c.Param("lessonId"))
	turnID, turnValid := parseID(c.Param("turn_id"))
	if !ok || !valid || !turnValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer"})
		return
	}
	view, err := h.courses.RetryConversation(c.Request.Context(), courseID, lessonID, turnID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}
