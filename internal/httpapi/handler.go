package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"learnos/internal/ai"
	"learnos/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	courses        *service.CourseService
	graph          *service.KnowledgeGraphService
	cognitive      *service.CognitiveStateService
	challenges     *service.ChallengeService
	misconceptions *service.MisconceptionService
}

func NewHandler(courses *service.CourseService, graph *service.KnowledgeGraphService, optional ...interface{}) *Handler {
	var cognitiveService *service.CognitiveStateService
	var challengeService *service.ChallengeService
	var misconceptionService *service.MisconceptionService
	for _, item := range optional {
		switch value := item.(type) {
		case *service.CognitiveStateService:
			cognitiveService = value
		case *service.ChallengeService:
			challengeService = value
		case *service.MisconceptionService:
			misconceptionService = value
		}
	}
	return &Handler{courses: courses, graph: graph, cognitive: cognitiveService, challenges: challengeService, misconceptions: misconceptionService}
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

type currentCourseResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type currentUnitResponse struct {
	ID        uint   `json:"id"`
	Title     string `json:"title"`
	Objective string `json:"objective"`
}

type currentLessonResponse struct {
	ID           uint   `json:"id"`
	Title        string `json:"title"`
	CoreQuestion string `json:"core_question"`
	Status       string `json:"status"`
}

func (h *Handler) GetCurrentLesson(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}

	current, err := h.courses.GetCurrentLesson(c.Request.Context(), courseID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"course": currentCourseResponse{ID: current.Course.ID, Name: current.Course.Name},
		"unit": currentUnitResponse{
			ID:        current.Unit.ID,
			Title:     current.Unit.Title,
			Objective: current.Unit.Objective,
		},
		"lesson": currentLessonResponse{
			ID:           current.Lesson.ID,
			Title:        current.Lesson.Title,
			CoreQuestion: current.Lesson.CoreQuestion,
			Status:       string(current.Lesson.Status),
		},
	}})
}

type submitAnswerRequest struct {
	LessonID uint   `json:"lesson_id"`
	Answer   string `json:"answer"`
}

func (h *Handler) SubmitAnswer(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}

	var request submitAnswerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer request"})
		return
	}
	result, err := h.courses.SubmitAnswer(c.Request.Context(), courseID, request.LessonID, request.Answer)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) ListLearningTurns(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}

	limit := 10
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 1 || parsedLimit > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 50"})
			return
		}
		limit = parsedLimit
	}

	turns, err := h.courses.ListLearningTurns(c.Request.Context(), courseID, uint(limit))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": turns})
}

func (h *Handler) GetKnowledgeGraph(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	graph, err := h.graph.GetCourseKnowledgeGraph(c.Request.Context(), courseID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": graph})
}

func (h *Handler) GetLessonRelations(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	lessonID, ok := parseID(c.Param("lessonId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson id"})
		return
	}
	relations, err := h.graph.GetLessonRelations(c.Request.Context(), courseID, lessonID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": relations})
}

func (h *Handler) GetCourseCognitiveStates(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	states, err := h.cognitive.GetCourseCognitiveStates(c.Request.Context(), courseID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": states})
}

func (h *Handler) GetLessonCognitiveState(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	lessonID, ok := parseID(c.Param("lessonId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson id"})
		return
	}
	detail, err := h.cognitive.GetLessonCognitiveState(c.Request.Context(), courseID, lessonID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": detail})
}

type createChallengeRequest struct {
	ChallengeType   string `json:"challenge_type"`
	MisconceptionID *uint  `json:"misconception_id"`
}

func (h *Handler) CreateChallenge(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	lessonID, ok := parseID(c.Param("lessonId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson id"})
		return
	}
	var request createChallengeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid challenge request"})
		return
	}
	challenge, err := h.challenges.Generate(c.Request.Context(), courseID, lessonID, request.ChallengeType, request.MisconceptionID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": challenge})
}

type answerChallengeRequest struct {
	Answer string `json:"answer"`
}

func (h *Handler) AnswerChallenge(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	challengeID, ok := parseID(c.Param("challengeId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid challenge id"})
		return
	}
	var request answerChallengeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid challenge answer request"})
		return
	}
	result, err := h.challenges.Answer(c.Request.Context(), courseID, challengeID, request.Answer)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetMisconceptionNetwork(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	network, err := h.misconceptions.GetNetwork(c.Request.Context(), courseID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": network})
}

func (h *Handler) ListLessonMisconceptions(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	lessonID, ok := parseID(c.Param("lessonId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson id"})
		return
	}
	items, err := h.misconceptions.ListLesson(c.Request.Context(), courseID, lessonID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func parseID(raw string) (uint, bool) {
	id, err := strconv.ParseUint(raw, 10, 64)
	return uint(id), err == nil && id > 0
}

func (h *Handler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCourseNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
	case errors.Is(err, service.ErrCurrentLessonNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "current lesson not found"})
	case errors.Is(err, service.ErrInvalidAnswer):
		c.JSON(http.StatusBadRequest, gin.H{"error": "answer is required and must be at most 5000 characters"})
	case errors.Is(err, service.ErrLessonNotCurrent):
		c.JSON(http.StatusBadRequest, gin.H{"error": "lesson is not the current lesson"})
	case errors.Is(err, service.ErrInvalidCourseID):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
	case errors.Is(err, service.ErrLessonNotInCourse):
		c.JSON(http.StatusNotFound, gin.H{"error": "lesson not found in course"})
	case errors.Is(err, service.ErrInvalidCognitiveState):
		c.JSON(http.StatusBadGateway, gin.H{"error": "AI_INVALID_COGNITIVE_RESPONSE"})
	case errors.Is(err, service.ErrChallengeNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "challenge not found"})
	case errors.Is(err, service.ErrChallengeAlreadyAnswered):
		c.JSON(http.StatusConflict, gin.H{"error": "challenge already answered"})
	case errors.Is(err, service.ErrChallengeInvalid), errors.Is(err, service.ErrChallengeNotEligible):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid challenge request"})
	case errors.Is(err, service.ErrChallengeAIUnavailable), errors.Is(err, ai.ErrNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI_NOT_CONFIGURED"})
	case errors.Is(err, service.ErrInvalidRelation), errors.Is(err, service.ErrPrerequisiteCycle):
		log.Printf("knowledge graph validation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge graph is invalid"})
	case errors.Is(err, ai.ErrNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI_NOT_CONFIGURED"})
	case errors.Is(err, ai.ErrTimeout):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "AI_TIMEOUT"})
	case errors.Is(err, ai.ErrInvalidResponse):
		c.JSON(http.StatusBadGateway, gin.H{"error": "AI_INVALID_RESPONSE"})
	case errors.Is(err, ai.ErrProvider):
		c.JSON(http.StatusBadGateway, gin.H{"error": "AI_PROVIDER_ERROR"})
	default:
		log.Printf("learning request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "learning request failed"})
	}
}
