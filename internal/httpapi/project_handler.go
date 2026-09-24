package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"learnos/internal/auth"
	"learnos/internal/repository"
	"learnos/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func projectUserID(c *gin.Context) (uint, bool) {
	principal, ok := auth.PrincipalFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return 0, false
	}
	return principal.UserID, true
}

func projectPathID(c *gin.Context, key string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(key), 10, 32)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(value), true
}

func projectError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidProjectInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project input"})
	case errors.Is(err, repository.ErrTaskOrder):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task order"})
	case errors.Is(err, repository.ErrTaskConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "task changed; reload and try again"})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "project or task not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "project operation failed"})
	}
}

func (h *Handler) ListProjects(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	projects, err := h.projects.List(c.Request.Context(), userID)
	if err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": projects})
}

func (h *Handler) CreateProject(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	var input service.ProjectPatch
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project input"})
		return
	}
	project, err := h.projects.CreateProject(c.Request.Context(), userID, input)
	if err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": project})
}

func (h *Handler) UpdateProject(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	projectID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	var patch service.ProjectPatch
	if c.ShouldBindJSON(&patch) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project input"})
		return
	}
	if err := h.projects.Update(c.Request.Context(), userID, projectID, patch); err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"updated": true}})
}

func (h *Handler) CreateProjectTask(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	projectID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	var input struct {
		Title string `json:"title"`
	}
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task input"})
		return
	}
	task, err := h.projects.CreateTask(c.Request.Context(), userID, projectID, input.Title)
	if err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": task})
}

func (h *Handler) UpdateProjectTask(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	projectID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	taskID, ok := projectPathID(c, "taskId")
	if !ok {
		return
	}
	var patch service.ProjectTaskPatch
	if c.ShouldBindJSON(&patch) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task input"})
		return
	}
	if err := h.projects.UpdateTask(c.Request.Context(), userID, projectID, taskID, patch); err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"updated": true}})
}

func (h *Handler) DeleteProjectTask(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	projectID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	taskID, ok := projectPathID(c, "taskId")
	if !ok {
		return
	}
	if err := h.projects.DeleteTask(c.Request.Context(), userID, projectID, taskID); err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": true}})
}

func (h *Handler) ListTasks(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	var projectID *uint
	if raw := c.Query("project_id"); raw != "" && raw != "all" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || parsed == 0 {
			projectError(c, service.ErrInvalidProjectInput)
			return
		}
		id := uint(parsed)
		projectID = &id
	}
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			projectError(c, service.ErrInvalidProjectInput)
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := c.Query("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			projectError(c, service.ErrInvalidProjectInput)
			return
		}
		offset = parsed
	}
	tasks, err := h.projects.ListTasks(c.Request.Context(), userID, projectID, c.Query("status"), c.Query("due"), limit, offset)
	if err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks})
}

func (h *Handler) CreateTask(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	var input service.ProjectTaskInput
	if c.ShouldBindJSON(&input) != nil {
		projectError(c, service.ErrInvalidProjectInput)
		return
	}
	task, err := h.projects.CreateTaskWithInput(c.Request.Context(), userID, input)
	if err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": task})
}

func (h *Handler) UpdateTask(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	taskID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	var patch service.ProjectTaskPatch
	if c.ShouldBindJSON(&patch) != nil {
		projectError(c, service.ErrInvalidProjectInput)
		return
	}
	task, err := h.projects.EditTask(c.Request.Context(), userID, taskID, patch)
	if err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": task})
}

func (h *Handler) MoveTask(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	taskID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	var input struct {
		Status            string `json:"status"`
		BeforeID          uint   `json:"before_id"`
		AfterID           uint   `json:"after_id"`
		ExpectedUpdatedAt string `json:"expected_updated_at"`
	}
	if c.ShouldBindJSON(&input) != nil {
		projectError(c, service.ErrInvalidProjectInput)
		return
	}
	var expected *time.Time
	if input.ExpectedUpdatedAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, input.ExpectedUpdatedAt)
		if err != nil {
			projectError(c, service.ErrInvalidProjectInput)
			return
		}
		expected = &parsed
	}
	task, err := h.projects.MoveTask(c.Request.Context(), userID, taskID, input.Status, input.BeforeID, input.AfterID, expected)
	if err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": task})
}

func (h *Handler) DeleteTask(c *gin.Context) {
	userID, ok := projectUserID(c)
	if !ok {
		return
	}
	taskID, ok := projectPathID(c, "id")
	if !ok {
		return
	}
	if err := h.projects.DeleteTaskByID(c.Request.Context(), userID, taskID); err != nil {
		projectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": true}})
}
