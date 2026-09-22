package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"learnos/internal/ai"
	"learnos/internal/model"
	"learnos/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	courses        *service.CourseService
	graph          *service.KnowledgeGraphService
	cognitive      *service.CognitiveStateService
	challenges     *service.ChallengeService
	misconceptions *service.MisconceptionService
	exploration    *service.ExplorationService
	curriculum     *service.CurriculumService
	grounding      *service.GroundingService
	backups        *service.BackupService
	databaseHealth *service.DatabaseHealthService
	export         *service.ExportService
	consistency    *service.ConsistencyService
	domainInit     *service.DomainInitializationService
	aiConfig       *service.AIConfigurationService
	nextLesson     *service.NextLessonService
}

func NewHandler(courses *service.CourseService, graph *service.KnowledgeGraphService, optional ...interface{}) *Handler {
	var cognitiveService *service.CognitiveStateService
	var challengeService *service.ChallengeService
	var misconceptionService *service.MisconceptionService
	var explorationService *service.ExplorationService
	var curriculumService *service.CurriculumService
	var groundingService *service.GroundingService
	var backupService *service.BackupService
	var databaseHealthService *service.DatabaseHealthService
	var exportService *service.ExportService
	var consistencyService *service.ConsistencyService
	var domainInitService *service.DomainInitializationService
	var aiConfigurationService *service.AIConfigurationService
	var nextLessonService *service.NextLessonService
	for _, item := range optional {
		switch value := item.(type) {
		case *service.CognitiveStateService:
			cognitiveService = value
		case *service.ChallengeService:
			challengeService = value
		case *service.MisconceptionService:
			misconceptionService = value
		case *service.ExplorationService:
			explorationService = value
		case *service.CurriculumService:
			curriculumService = value
		case *service.GroundingService:
			groundingService = value
		case *service.BackupService:
			backupService = value
		case *service.DatabaseHealthService:
			databaseHealthService = value
		case *service.ExportService:
			exportService = value
		case *service.ConsistencyService:
			consistencyService = value
		case *service.DomainInitializationService:
			domainInitService = value
		case *service.AIConfigurationService:
			aiConfigurationService = value
		case *service.NextLessonService:
			nextLessonService = value
		}
	}
	return &Handler{courses: courses, graph: graph, cognitive: cognitiveService, challenges: challengeService, misconceptions: misconceptionService, exploration: explorationService, curriculum: curriculumService, grounding: groundingService, backups: backupService, databaseHealth: databaseHealthService, export: exportService, consistency: consistencyService, domainInit: domainInitService, aiConfig: aiConfigurationService, nextLesson: nextLessonService}
}

func (h *Handler) GetAIConfiguration(c *gin.Context) {
	configuration, err := h.aiConfig.Get(c.Request.Context())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configuration})
}

func (h *Handler) UpdateAIConfiguration(c *gin.Context) {
	var request service.UpdateAIConfigurationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid AI configuration request"})
		return
	}
	configuration, err := h.aiConfig.Update(c.Request.Context(), request)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configuration})
}

func (h *Handler) TestAIConnection(c *gin.Context) {
	result, err := h.aiConfig.TestConnection(c.Request.Context())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) CreateDomainDraft(c *gin.Context) {
	var request service.CreateDomainRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "DOMAIN_INPUT_INVALID", "message": "请检查领域名称、学习原因和期望深度。", "retryable": false}})
		return
	}
	view, err := h.domainInit.CreateSkeleton(c.Request.Context(), request)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": view})
}

func (h *Handler) GetDomainDraft(c *gin.Context) {
	id, ok := parseID(c.Param("draftId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain draft id"})
		return
	}
	view, err := h.domainInit.Get(c.Request.Context(), id)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *Handler) RegenerateDomainSkeleton(c *gin.Context) {
	h.runDomainStage(c, "regenerate", func(ctx context.Context, id uint, _ []string) (*service.DomainInitializationView, error) {
		return h.domainInit.RegenerateSkeleton(ctx, id)
	})
}
func (h *Handler) ExpandDomainStarter(c *gin.Context) {
	var request struct {
		UnitKeys []string `json:"unit_keys"`
	}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid starter unit request"})
			return
		}
	}
	h.runDomainStage(c, "starter", func(ctx context.Context, id uint, _ []string) (*service.DomainInitializationView, error) {
		return h.domainInit.ExpandStarter(ctx, id, request.UnitKeys)
	})
}
func (h *Handler) GenerateDomainInitialWorld(c *gin.Context) {
	h.runDomainStage(c, "initial_world", func(ctx context.Context, id uint, _ []string) (*service.DomainInitializationView, error) {
		return h.domainInit.GenerateInitialWorld(ctx, id)
	})
}
func (h *Handler) ApplyDomainDraft(c *gin.Context) {
	h.runDomainStage(c, "apply", func(ctx context.Context, id uint, _ []string) (*service.DomainInitializationView, error) {
		return h.domainInit.Apply(ctx, id)
	})
}
func (h *Handler) runDomainStage(c *gin.Context, _ string, fn func(context.Context, uint, []string) (*service.DomainInitializationView, error)) {
	id, ok := parseID(c.Param("draftId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain draft id"})
		return
	}
	view, err := fn(c.Request.Context(), id, nil)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *Handler) Health(c *gin.Context) {
	if h.databaseHealth != nil {
		health := h.databaseHealth.Ping(c.Request.Context())
		if health.Status != "ok" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": health.Status, "detail": health.Detail})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (h *Handler) CreateBackup(c *gin.Context) {
	backup, err := h.backups.Create(c.Request.Context())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": backup})
}

func (h *Handler) ListBackups(c *gin.Context) {
	backups, err := h.backups.List(c.Request.Context())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": backups})
}

func (h *Handler) RequestBackupRestore(c *gin.Context) {
	var request struct {
		BackupName   string `json:"backup_name"`
		Confirmation string `json:"confirmation"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid restore request"})
		return
	}
	view, err := h.backups.RequestRestore(c.Request.Context(), request.BackupName, request.Confirmation)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": view})
	h.backups.TriggerRestoreRestart()
}

func (h *Handler) ExportData(c *gin.Context) {
	format := c.Query("format")
	if format == "" {
		format = "json"
	}
	var data []byte
	var err error
	contentType, filename := "application/json; charset=utf-8", "learnos-export.json"
	if format == "markdown" {
		data, err = h.export.Markdown(c.Request.Context())
		contentType, filename = "text/markdown; charset=utf-8", "learnos-export.md"
	} else if format == "json" {
		data, err = h.export.JSON(c.Request.Context())
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format must be json or markdown"})
		return
	}
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, contentType, data)
}

func (h *Handler) GetDiagnostics(c *gin.Context) {
	diagnostics, err := h.databaseHealth.Diagnostics(c.Request.Context())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": diagnostics})
}

func (h *Handler) GetConsistency(c *gin.Context) {
	report, err := h.consistency.Check(c.Request.Context())
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": report})
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

func (h *Handler) DeleteCourse(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	if err := h.courses.Delete(c.Request.Context(), courseID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": true}})
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
	ID           uint              `json:"id"`
	Title        string            `json:"title"`
	CoreQuestion string            `json:"core_question"`
	Status       string            `json:"status"`
	ContentRole  model.ContentRole `json:"content_role"`
	DepthLevel   int               `json:"depth_level"`
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

	h.writeCurrentLesson(c, current)
}

type setCurrentLessonRequest struct {
	LessonID uint `json:"lesson_id"`
}

func (h *Handler) SetCurrentLesson(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	var request setCurrentLessonRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.LessonID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid current lesson request"})
		return
	}
	current, err := h.courses.SetCurrentLesson(c.Request.Context(), courseID, request.LessonID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.writeCurrentLesson(c, current)
}

func (h *Handler) GetNextLesson(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	if h.nextLesson == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "next lesson service unavailable"})
		return
	}
	view, err := h.nextLesson.Recommend(c.Request.Context(), courseID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *Handler) GetLessonForLearning(c *gin.Context) {
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
	current, err := h.courses.GetLessonForLearning(c.Request.Context(), courseID, lessonID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.writeCurrentLesson(c, current)
}

func (h *Handler) writeCurrentLesson(c *gin.Context, current *service.CurrentLesson) {
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
			ContentRole:  current.Lesson.ContentRole,
			DepthLevel:   current.Lesson.DepthLevel,
		},
	}})
}

type submitAnswerRequest struct {
	LessonID       uint   `json:"lesson_id"`
	Answer         string `json:"answer"`
	IdempotencyKey string `json:"idempotency_key"`
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
	result, err := h.courses.SubmitAnswer(c.Request.Context(), courseID, request.LessonID, request.Answer, request.IdempotencyKey)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) SubmitLessonAnswer(c *gin.Context) {
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
	var request struct {
		Answer         string `json:"answer"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer request"})
		return
	}
	result, err := h.courses.SubmitLessonAnswer(c.Request.Context(), courseID, lessonID, request.Answer, request.IdempotencyKey)
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
	IdempotencyKey  string `json:"idempotency_key"`
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
	challenge, err := h.challenges.Generate(c.Request.Context(), courseID, lessonID, request.ChallengeType, request.MisconceptionID, request.IdempotencyKey)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": challenge})
}

type answerChallengeRequest struct {
	Answer         string `json:"answer"`
	IdempotencyKey string `json:"idempotency_key"`
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
	result, err := h.challenges.Answer(c.Request.Context(), courseID, challengeID, request.Answer, request.IdempotencyKey)
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

type reviewMisconceptionRequest struct {
	Action string `json:"action"`
	Note   string `json:"note"`
}

func (h *Handler) ReviewMisconception(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	misconceptionID, ok := parseID(c.Param("misconceptionId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid misconception id"})
		return
	}
	var request reviewMisconceptionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid misconception review"})
		return
	}
	item, err := h.misconceptions.Review(c.Request.Context(), courseID, misconceptionID, request.Action, request.Note)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *Handler) GetExplorationRadar(c *gin.Context) {
	courseID, ok := parseID(c.Query("course_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	lessonID := uint(0)
	if raw := c.Query("lesson_id"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson id"})
			return
		}
		lessonID = uint(parsed)
	}
	limit := 6
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 20"})
			return
		}
		limit = parsed
	}
	radars, err := h.exploration.GetRadar(c.Request.Context(), courseID, lessonID, limit)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": radars})
}

type explorationCourseRequest struct {
	CourseID uint `json:"course_id"`
}

func (h *Handler) FindUnfamiliarKnowledge(c *gin.Context) {
	courseID, ok := explorationCourseID(c)
	if !ok {
		var request explorationCourseRequest
		if err := c.ShouldBindJSON(&request); err != nil || request.CourseID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
			return
		}
		courseID = request.CourseID
	}
	direction, err := h.exploration.FindUnfamiliar(c.Request.Context(), courseID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": direction})
}

func (h *Handler) SaveExplorationDirection(c *gin.Context) {
	h.setExplorationDirectionStatus(c, model.ExplorationDirectionSaved)
}

func (h *Handler) DismissExplorationDirection(c *gin.Context) {
	h.setExplorationDirectionStatus(c, model.ExplorationDirectionDismissed)
}

func (h *Handler) OpenExplorationDirection(c *gin.Context) {
	h.setExplorationDirectionStatus(c, model.ExplorationDirectionOpened)
}

func (h *Handler) setExplorationDirectionStatus(c *gin.Context, status string) {
	courseID, ok := parseID(c.Query("course_id"))
	if !ok {
		var request explorationCourseRequest
		if err := c.ShouldBindJSON(&request); err != nil || request.CourseID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
			return
		}
		courseID = request.CourseID
	}
	directionID, ok := parseID(c.Param("directionId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid direction id"})
		return
	}
	direction, err := h.exploration.SetDirectionStatus(c.Request.Context(), courseID, directionID, status)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": direction})
}

func (h *Handler) AddExplorationQuestion(c *gin.Context) {
	courseID, ok := parseID(c.Query("course_id"))
	if !ok {
		var request explorationCourseRequest
		if err := c.ShouldBindJSON(&request); err != nil || request.CourseID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
			return
		}
		courseID = request.CourseID
	}
	directionID, ok := parseID(c.Param("directionId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid direction id"})
		return
	}
	question, err := h.exploration.AddQuestion(c.Request.Context(), courseID, directionID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": question})
}

func (h *Handler) ListExplorationQuestions(c *gin.Context) {
	courseID, ok := parseID(c.Query("course_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	limit := 50
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 100"})
			return
		}
		limit = parsed
	}
	targetCourseID := uint(0)
	if raw := c.Query("target_course_id"); raw != "" {
		parsed, valid := parseID(raw)
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target course id"})
			return
		}
		targetCourseID = parsed
	}
	questions, err := h.exploration.ListQuestions(c.Request.Context(), courseID, c.Query("status"), limit, service.ExplorationQuestionFilter{
		TargetCourseID: targetCourseID,
		Priority:       c.Query("priority"),
	})
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": questions})
}

func (h *Handler) StartExplorationQuestion(c *gin.Context) {
	h.setExplorationQuestionStatus(c, model.ExplorationQuestionExploring)
}

func (h *Handler) LaterExplorationQuestion(c *gin.Context) {
	h.setExplorationQuestionStatus(c, model.ExplorationQuestionLater)
}

func (h *Handler) ResolveExplorationQuestion(c *gin.Context) {
	h.setExplorationQuestionStatus(c, model.ExplorationQuestionResolved)
}

func (h *Handler) ReopenExplorationQuestion(c *gin.Context) {
	h.setExplorationQuestionStatus(c, model.ExplorationQuestionOpen)
}

func (h *Handler) ArchiveExplorationQuestion(c *gin.Context) {
	h.setExplorationQuestionStatus(c, model.ExplorationQuestionArchived)
}

func (h *Handler) UndoExplorationQuestion(c *gin.Context) {
	var request explorationCourseRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.CourseID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	directionID, ok := parseID(c.Param("directionId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid direction id"})
		return
	}
	direction, err := h.exploration.UndoQuestion(c.Request.Context(), request.CourseID, directionID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": direction})
}

type explorationQuestionPriorityRequest struct {
	CourseID uint   `json:"course_id"`
	Priority string `json:"priority"`
}

func (h *Handler) SetExplorationQuestionPriority(c *gin.Context) {
	var request explorationQuestionPriorityRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.CourseID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question priority request"})
		return
	}
	questionID, ok := parseID(c.Param("questionId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}
	question, err := h.exploration.SetQuestionPriority(c.Request.Context(), request.CourseID, questionID, request.Priority)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": question})
}

func (h *Handler) setExplorationQuestionStatus(c *gin.Context, status string) {
	courseID, ok := parseID(c.Query("course_id"))
	if !ok {
		var request explorationCourseRequest
		if err := c.ShouldBindJSON(&request); err != nil || request.CourseID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
			return
		}
		courseID = request.CourseID
	}
	questionID, ok := parseID(c.Param("questionId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}
	question, err := h.exploration.SetQuestionStatus(c.Request.Context(), courseID, questionID, status)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": question})
}

func (h *Handler) GetExplorationHistory(c *gin.Context) {
	courseID, ok := parseID(c.Query("course_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	directions, err := h.exploration.ListHistory(c.Request.Context(), courseID, 50)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": directions})
}

func (h *Handler) GetCurriculum(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	view, err := h.curriculum.GetCurriculum(c.Request.Context(), courseID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *Handler) GetCurriculumCoverage(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	view, err := h.curriculum.GetCoverage(c.Request.Context(), courseID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

type curriculumDraftRequest struct {
	Scope           string `json:"scope"`
	Limit           int    `json:"limit"`
	BlueprintUnitID uint   `json:"blueprint_unit_id"`
}

func (h *Handler) CreateCurriculumDraft(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	var request curriculumDraftRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid curriculum draft request"})
			return
		}
	}
	var view *service.CurriculumDraftView
	var err error
	if request.BlueprintUnitID > 0 {
		view, err = h.curriculum.GenerateDraftForUnit(c.Request.Context(), courseID, request.BlueprintUnitID, request.Scope, request.Limit)
	} else {
		view, err = h.curriculum.GenerateDraft(c.Request.Context(), courseID, request.Scope, request.Limit)
	}
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": view})
}

func (h *Handler) ExpandBlueprintUnit(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	unitID, ok := parseID(c.Param("blueprintUnitId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid blueprint unit id"})
		return
	}
	view, err := h.curriculum.ExpandBlueprintUnit(c.Request.Context(), courseID, unitID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *Handler) ListCurriculumDrafts(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	drafts, err := h.curriculum.ListDrafts(c.Request.Context(), courseID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": drafts})
}

func (h *Handler) GetCurriculumDraft(c *gin.Context) {
	courseID, draftID, ok := parseCurriculumIDs(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid curriculum draft id"})
		return
	}
	draft, err := h.curriculum.GetDraft(c.Request.Context(), courseID, draftID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": draft})
}

func (h *Handler) ApplyCurriculumDraft(c *gin.Context) {
	courseID, draftID, ok := parseCurriculumIDs(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid curriculum draft id"})
		return
	}
	draft, err := h.curriculum.ApplyDraft(c.Request.Context(), courseID, draftID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": draft})
}

func (h *Handler) RejectCurriculumDraft(c *gin.Context) {
	courseID, draftID, ok := parseCurriculumIDs(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid curriculum draft id"})
		return
	}
	draft, err := h.curriculum.RejectDraft(c.Request.Context(), courseID, draftID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": draft})
}

func parseCurriculumIDs(c *gin.Context) (uint, uint, bool) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		return 0, 0, false
	}
	draftID, ok := parseID(c.Param("draftId"))
	return courseID, draftID, ok
}

func (h *Handler) ListSources(c *gin.Context) {
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	sources, err := h.grounding.ListSources(c.Request.Context(), c.Query("q"), c.Query("source_type"), limit)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sources})
}

func (h *Handler) CreateSource(c *gin.Context) {
	var input service.SourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source request"})
		return
	}
	source, err := h.grounding.CreateSource(c.Request.Context(), input)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": source})
}

func (h *Handler) GetSource(c *gin.Context) {
	sourceID, ok := parseID(c.Param("sourceId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	view, err := h.grounding.GetSource(c.Request.Context(), sourceID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *Handler) UpdateSource(c *gin.Context) {
	sourceID, ok := parseID(c.Param("sourceId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	var patch service.SourcePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source update"})
		return
	}
	view, err := h.grounding.UpdateSource(c.Request.Context(), sourceID, patch)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *Handler) CreateSourceEvidence(c *gin.Context) {
	sourceID, ok := parseID(c.Param("sourceId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	var input service.EvidenceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid evidence request"})
		return
	}
	evidence, err := h.grounding.AddEvidence(c.Request.Context(), sourceID, input)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": evidence})
}

func (h *Handler) ListSourceEvidence(c *gin.Context) {
	sourceID, ok := parseID(c.Param("sourceId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	evidence, err := h.grounding.ListEvidence(c.Request.Context(), sourceID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": evidence})
}

func (h *Handler) ListSourceGroundingLinks(c *gin.Context) {
	sourceID, ok := parseID(c.Param("sourceId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	links, err := h.grounding.ListSourceLinks(c.Request.Context(), sourceID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": links})
}

func (h *Handler) UpdateSourceEvidence(c *gin.Context) {
	evidenceID, ok := parseID(c.Param("evidenceId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid evidence id"})
		return
	}
	var patch service.EvidencePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid evidence update"})
		return
	}
	evidence, err := h.grounding.UpdateEvidence(c.Request.Context(), evidenceID, patch)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": evidence})
}

func (h *Handler) CreateGroundingLink(c *gin.Context) {
	var input service.GroundingLinkInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid grounding link request"})
		return
	}
	link, err := h.grounding.CreateLink(c.Request.Context(), input)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": link})
}

func (h *Handler) ListGroundingLinks(c *gin.Context) {
	targetID, ok := parseID(c.Query("target_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target id"})
		return
	}
	links, err := h.grounding.ListLinks(c.Request.Context(), c.Query("target_type"), targetID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": links})
}

func (h *Handler) ReviewGroundingLink(c *gin.Context) { h.reviewGroundingLink(c, true) }
func (h *Handler) RejectGroundingLink(c *gin.Context) { h.reviewGroundingLink(c, false) }
func (h *Handler) reviewGroundingLink(c *gin.Context, review bool) {
	linkID, ok := parseID(c.Param("linkId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid grounding link id"})
		return
	}
	var target *service.GroundingTargetView
	var err error
	if review {
		target, err = h.grounding.ReviewLink(c.Request.Context(), linkID)
	} else {
		target, err = h.grounding.RejectLink(c.Request.Context(), linkID)
	}
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": target})
}

func (h *Handler) ListSourceCredibility(c *gin.Context) {
	sourceID, ok := parseID(c.Param("sourceId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	items, err := h.grounding.ListCredibility(c.Request.Context(), sourceID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) CreateSourceCredibility(c *gin.Context) {
	sourceID, ok := parseID(c.Param("sourceId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	var input service.CredibilityInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credibility request"})
		return
	}
	item, err := h.grounding.CreateCredibility(c.Request.Context(), sourceID, input)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *Handler) ReviewSourceCredibility(c *gin.Context) {
	assessmentID, ok := parseID(c.Param("assessmentId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credibility assessment id"})
		return
	}
	item, err := h.grounding.ReviewCredibility(c.Request.Context(), assessmentID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *Handler) GetGroundingTarget(c *gin.Context) {
	targetID, ok := parseID(c.Param("targetId"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target id"})
		return
	}
	view, err := h.grounding.GetTarget(c.Request.Context(), c.Param("targetType"), targetID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *Handler) GetGroundingCoverage(c *gin.Context) {
	courseID, ok := parseID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course id"})
		return
	}
	view, err := h.grounding.GetCoverage(c.Request.Context(), courseID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func explorationCourseID(c *gin.Context) (uint, bool) {
	return parseID(c.Query("course_id"))
}

func parseID(raw string) (uint, bool) {
	id, err := strconv.ParseUint(raw, 10, 64)
	return uint(id), err == nil && id > 0
}

func (h *Handler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAIConfigurationInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "AI_CONFIGURATION_INVALID", "message": "AI 配置无效，请检查 Provider 和 API Key。", "retryable": false}})
	case errors.Is(err, service.ErrAIConnectionFailed):
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "AI_CONNECTION_FAILED", "message": "无法连接 AI 服务，请检查 Provider、模型、地址和 API Key。", "retryable": true}})
	case errors.Is(err, service.ErrBackupNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "BACKUP_NOT_FOUND", "message": "找不到指定备份文件。", "retryable": false}})
	case errors.Is(err, service.ErrRestoreConfirmationFailed):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "RESTORE_CONFIRMATION_REQUIRED", "message": "恢复确认文字不匹配。", "retryable": false}})
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
	case errors.Is(err, service.ErrExplorationDirectionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "exploration direction not found"})
	case errors.Is(err, service.ErrExplorationQuestionNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "exploration question not found"})
	case errors.Is(err, service.ErrExplorationNoUnknown):
		c.JSON(http.StatusNotFound, gin.H{"error": "no unfamiliar knowledge available"})
	case errors.Is(err, service.ErrExplorationInvalidStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exploration status"})
	case errors.Is(err, service.ErrCurriculumBlueprintNotFound), errors.Is(err, service.ErrCurriculumDraftNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "curriculum resource not found"})
	case errors.Is(err, service.ErrCurriculumUnitNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "curriculum blueprint unit not found"})
	case errors.Is(err, service.ErrCurriculumUnitExpansionInProgress):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "UNIT_EXPANSION_IN_PROGRESS", "message": "知识区域正在生成，请稍候。", "retryable": true}})
	case errors.Is(err, service.ErrCurriculumUnitAlreadyExpanded):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "UNIT_ALREADY_EXPANDED", "message": "这个知识区域已经展开。", "retryable": false}})
	case errors.Is(err, service.ErrCurriculumInvalidScope), errors.Is(err, service.ErrCurriculumDraftInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid curriculum draft request"})
	case errors.Is(err, service.ErrCurriculumDraftGenerationInProgress):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CURRICULUM_DRAFT_GENERATION_IN_PROGRESS", "message": "课程草案正在生成，请稍候。", "retryable": true}})
	case errors.Is(err, service.ErrCurriculumDraftAlreadyPending):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CURRICULUM_DRAFT_ALREADY_PENDING", "message": "你已经有一份待审核的课程草案。", "retryable": false}})
	case errors.Is(err, service.ErrCurriculumDraftApplied), errors.Is(err, service.ErrCurriculumKeyConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "curriculum draft cannot be applied"})
	case errors.Is(err, service.ErrKnowledgeSourceNotFound), errors.Is(err, service.ErrSourceEvidenceNotFound), errors.Is(err, service.ErrGroundingLinkNotFound), errors.Is(err, service.ErrCredibilityNotFound), errors.Is(err, service.ErrGroundingTargetNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "grounding resource not found"})
	case errors.Is(err, service.ErrSourceDuplicate), errors.Is(err, service.ErrGroundingLinkConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "grounding resource already exists"})
	case errors.Is(err, service.ErrSourceInvalid), errors.Is(err, service.ErrEvidenceInvalid), errors.Is(err, service.ErrGroundingLinkInvalid), errors.Is(err, service.ErrCredibilityInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid grounding request"})
	case errors.Is(err, service.ErrDomainDraftNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "domain initialization draft not found"})
	case errors.Is(err, service.ErrDomainAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "DOMAIN_ALREADY_EXISTS", "message": "这个学习领域已经存在。", "retryable": false}})
	case errors.Is(err, service.ErrDomainInputInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "DOMAIN_INPUT_INVALID", "message": "请检查领域名称、学习原因和期望深度。", "retryable": false}})
	case errors.Is(err, service.ErrDomainDraftInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain initialization request"})
	case errors.Is(err, service.ErrDomainDraftState):
		c.JSON(http.StatusConflict, gin.H{"error": "domain initialization stage is not available"})
	case errors.Is(err, service.ErrDomainDraftApplied):
		c.JSON(http.StatusConflict, gin.H{"error": "domain initialization draft already applied"})
	case errors.Is(err, service.ErrInvalidCognitiveState):
		c.JSON(http.StatusBadGateway, gin.H{"error": "AI_INVALID_COGNITIVE_RESPONSE"})
	case errors.Is(err, service.ErrChallengeNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "challenge not found"})
	case errors.Is(err, service.ErrChallengeAlreadyAnswered):
		c.JSON(http.StatusConflict, gin.H{"error": "challenge already answered"})
	case errors.Is(err, service.ErrIdempotencyConflict):
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "IDEMPOTENCY_CONFLICT", "message": "这次提交与已有请求不一致，请刷新页面后重试。", "retryable": false}})
	case errors.Is(err, service.ErrChallengeInvalid), errors.Is(err, service.ErrChallengeNotEligible):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid challenge request"})
	case errors.Is(err, service.ErrChallengeAIUnavailable), errors.Is(err, ai.ErrNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "AI_NOT_CONFIGURED", "message": "AI 尚未配置，本次数据未保存。", "retryable": false}})
	case errors.Is(err, service.ErrInvalidRelation), errors.Is(err, service.ErrPrerequisiteCycle):
		log.Printf("knowledge graph validation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge graph is invalid"})
	case errors.Is(err, ai.ErrNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "AI_NOT_CONFIGURED", "message": "AI 尚未配置，本次数据未保存。", "retryable": false}})
	case errors.Is(err, ai.ErrTimeout):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": gin.H{"code": "AI_TIMEOUT", "message": "AI 响应超时，本次数据未保存。", "retryable": true}})
	case errors.Is(err, ai.ErrRateLimited):
		c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"code": "AI_RATE_LIMITED", "message": "AI 请求过于频繁，请稍后重试。本次数据未保存。", "retryable": true}})
	case errors.Is(err, ai.ErrNetworkError):
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "AI_NETWORK_ERROR", "message": "暂时无法连接 AI 服务，本次数据未保存。", "retryable": true}})
	case errors.Is(err, ai.ErrEmptyContent):
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "AI_EMPTY_RESPONSE", "message": "AI 返回了空响应，本次数据未保存。", "retryable": true}})
	case errors.Is(err, ai.ErrInvalidResponse):
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "AI_INVALID_RESPONSE", "message": "AI 返回内容无法验证，本次数据未保存。", "retryable": true}})
	case errors.Is(err, service.ErrCurriculumUnitExpansion):
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "AI_INVALID_RESPONSE", "message": "AI 返回的知识区域无法验证，本次数据未保存。", "retryable": true}})
	case errors.Is(err, ai.ErrProvider):
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "AI_PROVIDER_ERROR", "message": "AI 服务暂时不可用，本次数据未保存。", "retryable": true}})
	default:
		log.Printf("learning request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "learning request failed"})
	}
}
