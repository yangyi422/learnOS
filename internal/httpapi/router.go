package httpapi

import (
	"io/fs"
	"net/http"
	"strings"

	"learnos/internal/config"
	"learnos/internal/middleware"
	"learnos/internal/service"

	"github.com/gin-gonic/gin"
)

func NewRouter(cfg config.Config, handler *Handler, webFS fs.FS, userServices ...*service.UserService) *gin.Engine {
	if cfg.Production() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Keep health checks public so Docker and reverse proxies can probe the app.
	router.GET("/healthz", handler.Health)
	router.GET("/health", handler.Health)
	router.POST("/api/v1/auth/login", handler.Login)

	var userService *service.UserService
	if len(userServices) > 0 {
		userService = userServices[0]
	}

	api := router.Group("/api/v1")
	if !cfg.Development() {
		if userService != nil {
			api.Use(middleware.SessionAuth(userService.AuthenticateSession))
		} else {
			api.Use(middleware.BasicAuth(cfg.Username, cfg.PasswordHash))
		}
	}
	{
		api.POST("/auth/logout", handler.Logout)
		api.GET("/auth/me", handler.CurrentUser)
		api.GET("/users", middleware.RequireAdmin(), handler.ListUsers)
		api.POST("/users", middleware.RequireAdmin(), handler.CreateUser)
		api.PATCH("/users/:id/status", middleware.RequireAdmin(), handler.SetUserStatus)
		api.GET("/users/:id/courses", middleware.RequireAdmin(), handler.ListUserCourses)
		api.GET("/courses", handler.ListCourses)
		api.DELETE("/courses/:id", handler.DeleteCourse)
		api.POST("/domains/drafts", handler.CreateDomainDraft)
		api.GET("/domains/drafts/:draftId", handler.GetDomainDraft)
		api.POST("/domains/drafts/:draftId/regenerate-skeleton", handler.RegenerateDomainSkeleton)
		api.POST("/domains/drafts/:draftId/expand-starter", handler.ExpandDomainStarter)
		api.POST("/domains/drafts/:draftId/generate-initial-world", handler.GenerateDomainInitialWorld)
		api.POST("/domains/drafts/:draftId/apply", handler.ApplyDomainDraft)
		api.GET("/courses/:id/current-lesson", handler.GetCurrentLesson)
		api.POST("/courses/:id/current-lesson", handler.SetCurrentLesson)
		api.GET("/courses/:id/next-lesson", handler.GetNextLesson)
		api.GET("/courses/:id/lessons/:lessonId/learning", handler.GetLessonForLearning)
		api.POST("/courses/:id/answers", handler.SubmitAnswer)
		api.POST("/courses/:id/lessons/:lessonId/answers", handler.SubmitLessonAnswer)
		api.GET("/courses/:id/learning-turns", handler.ListLearningTurns)
		api.GET("/courses/:id/knowledge-graph", handler.GetKnowledgeGraph)
		api.GET("/courses/:id/lessons/:lessonId/relations", handler.GetLessonRelations)
		api.GET("/courses/:id/cognitive-states", handler.GetCourseCognitiveStates)
		api.GET("/courses/:id/lessons/:lessonId/cognitive-state", handler.GetLessonCognitiveState)
		api.POST("/courses/:id/lessons/:lessonId/challenges", handler.CreateChallenge)
		api.POST("/courses/:id/challenges/:challengeId/answers", handler.AnswerChallenge)
		api.GET("/courses/:id/misconception-network", handler.GetMisconceptionNetwork)
		api.GET("/courses/:id/lessons/:lessonId/misconceptions", handler.ListLessonMisconceptions)
		api.POST("/courses/:id/misconceptions/:misconceptionId/review", handler.ReviewMisconception)
		api.GET("/exploration/radar", handler.GetExplorationRadar)
		api.POST("/exploration/unfamiliar", handler.FindUnfamiliarKnowledge)
		api.POST("/exploration/directions/:directionId/save", handler.SaveExplorationDirection)
		api.POST("/exploration/directions/:directionId/dismiss", handler.DismissExplorationDirection)
		api.POST("/exploration/directions/:directionId/open", handler.OpenExplorationDirection)
		api.POST("/exploration/directions/:directionId/questions", handler.AddExplorationQuestion)
		api.POST("/exploration/directions/:directionId/questions/undo", handler.UndoExplorationQuestion)
		api.GET("/exploration/questions", handler.ListExplorationQuestions)
		api.POST("/exploration/questions/:questionId/start", handler.StartExplorationQuestion)
		api.POST("/exploration/questions/:questionId/later", handler.LaterExplorationQuestion)
		api.POST("/exploration/questions/:questionId/resolve", handler.ResolveExplorationQuestion)
		api.POST("/exploration/questions/:questionId/reopen", handler.ReopenExplorationQuestion)
		api.POST("/exploration/questions/:questionId/priority", handler.SetExplorationQuestionPriority)
		api.POST("/exploration/questions/:questionId/archive", handler.ArchiveExplorationQuestion)
		api.GET("/exploration/history", handler.GetExplorationHistory)
		api.GET("/courses/:id/curriculum", handler.GetCurriculum)
		api.GET("/courses/:id/curriculum/coverage", handler.GetCurriculumCoverage)
		api.POST("/courses/:id/curriculum/units/:blueprintUnitId/expand", handler.ExpandBlueprintUnit)
		api.POST("/courses/:id/curriculum/drafts", handler.CreateCurriculumDraft)
		api.GET("/courses/:id/curriculum/drafts", handler.ListCurriculumDrafts)
		api.GET("/courses/:id/curriculum/drafts/:draftId", handler.GetCurriculumDraft)
		api.POST("/courses/:id/curriculum/drafts/:draftId/apply", handler.ApplyCurriculumDraft)
		api.POST("/courses/:id/curriculum/drafts/:draftId/reject", handler.RejectCurriculumDraft)
		api.GET("/sources", handler.ListSources)
		api.POST("/sources", handler.CreateSource)
		api.GET("/sources/:sourceId", handler.GetSource)
		api.PATCH("/sources/:sourceId", handler.UpdateSource)
		api.POST("/sources/:sourceId/evidence", handler.CreateSourceEvidence)
		api.GET("/sources/:sourceId/evidence", handler.ListSourceEvidence)
		api.GET("/sources/:sourceId/grounding-links", handler.ListSourceGroundingLinks)
		api.PATCH("/evidence/:evidenceId", handler.UpdateSourceEvidence)
		api.POST("/grounding/links", handler.CreateGroundingLink)
		api.GET("/grounding/links", handler.ListGroundingLinks)
		api.POST("/grounding/links/:linkId/review", handler.ReviewGroundingLink)
		api.POST("/grounding/links/:linkId/reject", handler.RejectGroundingLink)
		api.GET("/sources/:sourceId/credibility", handler.ListSourceCredibility)
		api.POST("/sources/:sourceId/credibility", handler.CreateSourceCredibility)
		api.POST("/sources/:sourceId/credibility/:assessmentId/review", handler.ReviewSourceCredibility)
		api.GET("/grounding/targets/:targetType/:targetId", handler.GetGroundingTarget)
		api.GET("/courses/:id/grounding/coverage", handler.GetGroundingCoverage)
		system := api.Group("/system", middleware.RequireAdmin())
		system.POST("/backups", handler.CreateBackup)
		system.GET("/backups", handler.ListBackups)
		system.POST("/backups/restore", handler.RequestBackupRestore)
		system.POST("/export", handler.ExportData)
		system.GET("/diagnostics", handler.GetDiagnostics)
		system.GET("/consistency", handler.GetConsistency)
		system.GET("/ai-config", handler.GetAIConfiguration)
		system.PATCH("/ai-config", handler.UpdateAIConfiguration)
		system.POST("/ai-config/test", handler.TestAIConnection)
	}

	fileServer := http.FileServer(http.FS(webFS))
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(webFS, path); err != nil {
			// Serve the SPA entry through the directory root. Passing
			// /index.html to http.FileServer makes it redirect to ./; on a
			// history-mode route that relative redirect points back to the
			// deep link and loops forever on browser refresh.
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return router
}
